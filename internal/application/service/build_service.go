package service

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"golang.org/x/sync/errgroup"

	"github.com/morphy76/vuhive-cloud/internal/application/ports/inbound"
	"github.com/morphy76/vuhive-cloud/internal/application/ports/outbound"
	"github.com/morphy76/vuhive-cloud/internal/domain/model"
)

const (
	defaultPresignLifetime = 1 * time.Hour
)

// BuildService implements the inbound.BuildsUseCase port to orchestrate multi-arch binary compilation.
type BuildService struct {
	artifactRepo outbound.ArtifactRepository
	storage      outbound.StoragePort
	orchestrator outbound.BuildOrchestratorPort
}

// NewBuildService creates a new BuildService with the supplied outbound ports.
func NewBuildService(
	artifactRepo outbound.ArtifactRepository,
	storage outbound.StoragePort,
	orchestrator outbound.BuildOrchestratorPort,
) *BuildService {
	return &BuildService{
		artifactRepo: artifactRepo,
		storage:      storage,
		orchestrator: orchestrator,
	}
}

// BuildArtifact triggers compilation for a specific artifact, streams logs, and updates the artifact status.
func (s *BuildService) BuildArtifact(ctx context.Context, suiteID, artifactID string) (*model.Artifact, error) {
	start := time.Now()
	trimmedSuiteID := strings.TrimSpace(suiteID)
	trimmedArtifactID := strings.TrimSpace(artifactID)

	if trimmedSuiteID == "" || trimmedArtifactID == "" {
		return nil, fmt.Errorf("%w: suiteID and artifactID must not be empty", model.ErrValidation)
	}

	log := zerolog.Ctx(ctx).With().
		Str("op", "BuildService.BuildArtifact").
		Str("suite_id", trimmedSuiteID).
		Str("artifact_id", trimmedArtifactID).
		Logger()
	log.Debug().Msg("starting artifact build")

	artifact, err := s.artifactRepo.FindByID(ctx, trimmedArtifactID)
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed fetching artifact from repository")
		return nil, err
	}

	if artifact.SuiteID() != trimmedSuiteID {
		err := fmt.Errorf("%w: artifact does not belong to suite %s", model.ErrValidation, trimmedSuiteID)
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("suite ID mismatch")
		return nil, err
	}

	sourceKey := formatSourceKey(trimmedSuiteID)
	exists, err := s.storage.Exists(ctx, sourceKey)
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed checking source tarball existence")
		return nil, err
	}
	if !exists {
		err := fmt.Errorf("%w: source tarball not found at %s", model.ErrNotFound, sourceKey)
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("source tarball missing")
		return nil, err
	}

	if err := artifact.MarkBuilding(); err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed marking artifact as building")
		return nil, err
	}
	if err := s.artifactRepo.Save(ctx, artifact); err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed saving building state")
		return nil, err
	}

	sourceURL, err := s.storage.PresignDownload(ctx, sourceKey, defaultPresignLifetime)
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed generating presigned source download url")
		return nil, err
	}

	binaryKey, err := formatBinaryKey(trimmedSuiteID, trimmedArtifactID, artifact.Platform())
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("invalid platform for binary key")
		return nil, err
	}

	binaryUploadURL, err := s.storage.PresignUpload(ctx, binaryKey, defaultPresignLifetime)
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed generating presigned binary upload url")
		return nil, err
	}

	buildOpts := outbound.BuildJobOptions{
		SuiteID:         trimmedSuiteID,
		ArtifactID:      trimmedArtifactID,
		Platform:        artifact.Platform(),
		SourceURL:       sourceURL,
		BinaryUploadURL: binaryUploadURL,
	}

	jobName, err := s.orchestrator.DispatchBuildJob(ctx, buildOpts)
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed dispatching build job")
		_ = artifact.MarkFailed(err.Error(), "")
		_ = s.artifactRepo.Save(ctx, artifact)
		return nil, err
	}

	execResult, waitErr := s.orchestrator.WaitForJob(ctx, jobName)

	logsKey := formatLogsKey(trimmedSuiteID, trimmedArtifactID)
	if execResult != nil && execResult.Logs != nil {
		defer execResult.Logs.Close()
		logBuf := new(bytes.Buffer)
		_, _ = io.Copy(logBuf, execResult.Logs)
		_ = s.storage.Upload(ctx, logsKey, bytes.NewReader(logBuf.Bytes()), int64(logBuf.Len()), "text/plain")
	}

	if waitErr != nil {
		log.Error().Err(waitErr).Dur("duration_ms", time.Since(start)).Msg("build job execution failed")
		if markErr := artifact.MarkFailed(waitErr.Error(), logsKey); markErr == nil {
			_ = s.artifactRepo.Save(ctx, artifact)
		}
		return artifact, waitErr
	}

	checksum := execResult.SHA256Checksum
	if checksum == "" {
		computed, err := s.computeSHA256FromStorage(ctx, binaryKey)
		if err == nil {
			checksum = computed
		}
	}

	if err := artifact.MarkReady(binaryKey, checksum); err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed marking artifact as ready")
		return nil, err
	}

	if err := s.artifactRepo.Save(ctx, artifact); err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed saving ready artifact to repository")
		return nil, err
	}

	log.Info().
		Str("status", string(artifact.Status())).
		Str("binary_key", binaryKey).
		Str("checksum", checksum).
		Dur("duration_ms", time.Since(start)).
		Msg("completed artifact build")

	return artifact, nil
}

// BuildSuite triggers multi-arch compilation for all target platforms (linux/amd64 and linux/arm64) of a test suite.
func (s *BuildService) BuildSuite(ctx context.Context, suiteID string) ([]*model.Artifact, error) {
	start := time.Now()
	trimmedSuiteID := strings.TrimSpace(suiteID)
	if trimmedSuiteID == "" {
		return nil, fmt.Errorf("%w: suiteID cannot be empty", model.ErrValidation)
	}

	log := zerolog.Ctx(ctx).With().
		Str("op", "BuildService.BuildSuite").
		Str("suite_id", trimmedSuiteID).
		Logger()
	log.Debug().Msg("starting multi-arch suite build")

	sourceKey := formatSourceKey(trimmedSuiteID)
	exists, err := s.storage.Exists(ctx, sourceKey)
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed checking source tarball existence")
		return nil, err
	}
	if !exists {
		err := fmt.Errorf("%w: source tarball not found at %s", model.ErrNotFound, sourceKey)
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("source tarball missing")
		return nil, err
	}

	existingArtifacts, err := s.artifactRepo.ListBySuiteID(ctx, trimmedSuiteID)
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed listing suite artifacts")
		return nil, err
	}

	targetPlatforms := []model.Platform{model.PlatformLinuxAmd64, model.PlatformLinuxArm64}
	artifactsToBuild := make([]*model.Artifact, 0, len(targetPlatforms))

	for _, platform := range targetPlatforms {
		var found *model.Artifact
		for _, a := range existingArtifacts {
			if a.Platform() == platform && a.Status() != model.ArtifactStatusReady {
				found = a
				break
			}
		}

		if found == nil {
			newArt, err := model.NewArtifact(trimmedSuiteID, platform)
			if err != nil {
				return nil, err
			}
			if err := s.artifactRepo.Save(ctx, newArt); err != nil {
				return nil, err
			}
			artifactsToBuild = append(artifactsToBuild, newArt)
		} else {
			artifactsToBuild = append(artifactsToBuild, found)
		}
	}

	var mu sync.Mutex
	results := make([]*model.Artifact, 0, len(artifactsToBuild))

	g, groupCtx := errgroup.WithContext(ctx)
	for _, art := range artifactsToBuild {
		artToBuild := art
		g.Go(func() error {
			built, err := s.BuildArtifact(groupCtx, trimmedSuiteID, artToBuild.ID())
			if err != nil {
				return err
			}
			mu.Lock()
			results = append(results, built)
			mu.Unlock()
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("multi-arch build failed")
		return nil, err
	}

	log.Info().
		Int("compiled_count", len(results)).
		Dur("duration_ms", time.Since(start)).
		Msg("completed multi-arch suite build")

	return results, nil
}

// GetArtifact returns the artifact with the given identifier.
func (s *BuildService) GetArtifact(ctx context.Context, id string) (*model.Artifact, error) {
	start := time.Now()
	trimmedID := strings.TrimSpace(id)
	if trimmedID == "" {
		return nil, fmt.Errorf("%w: artifact id cannot be empty", model.ErrValidation)
	}

	log := zerolog.Ctx(ctx).With().
		Str("op", "BuildService.GetArtifact").
		Str("artifact_id", trimmedID).
		Logger()
	log.Debug().Msg("fetching artifact")

	artifact, err := s.artifactRepo.FindByID(ctx, trimmedID)
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed fetching artifact")
		return nil, err
	}

	log.Info().Dur("duration_ms", time.Since(start)).Msg("completed artifact retrieval")
	return artifact, nil
}

// ListArtifacts returns all binary artifacts associated with a test suite.
func (s *BuildService) ListArtifacts(ctx context.Context, suiteID string) ([]*model.Artifact, error) {
	start := time.Now()
	trimmedSuiteID := strings.TrimSpace(suiteID)
	if trimmedSuiteID == "" {
		return nil, fmt.Errorf("%w: suiteID cannot be empty", model.ErrValidation)
	}

	log := zerolog.Ctx(ctx).With().
		Str("op", "BuildService.ListArtifacts").
		Str("suite_id", trimmedSuiteID).
		Logger()
	log.Debug().Msg("listing artifacts for suite")

	artifacts, err := s.artifactRepo.ListBySuiteID(ctx, trimmedSuiteID)
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed listing artifacts")
		return nil, err
	}

	log.Info().
		Int("count", len(artifacts)).
		Dur("duration_ms", time.Since(start)).
		Msg("completed listing artifacts")

	return artifacts, nil
}

func (s *BuildService) computeSHA256FromStorage(ctx context.Context, binaryKey string) (string, error) {
	reader, err := s.storage.Download(ctx, binaryKey)
	if err != nil {
		return "", err
	}
	defer reader.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, reader); err != nil {
		return "", err
	}
	return hex.EncodeToString(hasher.Sum(nil)), nil
}

func formatSourceKey(suiteID string) string {
	return fmt.Sprintf("suites/%s/sources/source.tar.gz", suiteID)
}

func formatBinaryKey(suiteID, artifactID string, platform model.Platform) (string, error) {
	switch platform {
	case model.PlatformLinuxAmd64:
		return fmt.Sprintf("suites/%s/artifacts/%s/linux-amd64/runner", suiteID, artifactID), nil
	case model.PlatformLinuxArm64:
		return fmt.Sprintf("suites/%s/artifacts/%s/linux-arm64/runner", suiteID, artifactID), nil
	default:
		return "", model.ErrInvalidPlatform
	}
}

func formatLogsKey(suiteID, artifactID string) string {
	return fmt.Sprintf("suites/%s/artifacts/%s/build.log", suiteID, artifactID)
}

// Compile-time interface assertion
var _ inbound.BuildsUseCase = (*BuildService)(nil)
