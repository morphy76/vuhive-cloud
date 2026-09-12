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
	domainservice "github.com/morphy76/vuhive-cloud/internal/domain/service"
)

const (
	defaultPresignLifetime = 1 * time.Hour
)

// BuildService implements the inbound.BuildsUseCase port to orchestrate multi-arch binary compilation.
type BuildService struct {
	suiteRepo      outbound.TestSuiteRepository
	artifactRepo   outbound.ArtifactRepository
	storage        outbound.StoragePort
	orchestrator   outbound.BuildOrchestratorPort
	staticAnalyzer *domainservice.StaticAnalyzer
}

var _ inbound.BuildsUseCase = (*BuildService)(nil)

// NewBuildService creates a new BuildService with the supplied outbound ports and optional static analyzer.
func NewBuildService(
	suiteRepo outbound.TestSuiteRepository,
	artifactRepo outbound.ArtifactRepository,
	storage outbound.StoragePort,
	orchestrator outbound.BuildOrchestratorPort,
	analyzer ...*domainservice.StaticAnalyzer,
) *BuildService {
	var sa *domainservice.StaticAnalyzer
	if len(analyzer) > 0 {
		sa = analyzer[0]
	}
	return &BuildService{
		suiteRepo:      suiteRepo,
		artifactRepo:   artifactRepo,
		storage:        storage,
		orchestrator:   orchestrator,
		staticAnalyzer: sa,
	}
}

// TriggerBuild stages a source tarball in S3, creates artifact records, and initiates compilation asynchronously.
func (s *BuildService) TriggerBuild(
	ctx context.Context,
	suiteID string,
	platform *model.Platform,
	source io.Reader,
	size int64,
) ([]*model.Artifact, error) {
	return s.TriggerBuildWithOptions(ctx, suiteID, platform, source, size, inbound.BuildOptions{})
}

// TriggerBuildWithOptions stages a source tarball in S3 with optional static analysis flags, creates artifact records, and initiates compilation asynchronously.
func (s *BuildService) TriggerBuildWithOptions(
	ctx context.Context,
	suiteID string,
	platform *model.Platform,
	source io.Reader,
	size int64,
	opts inbound.BuildOptions,
) ([]*model.Artifact, error) {
	start := time.Now()
	trimmedSuiteID := strings.TrimSpace(suiteID)
	if trimmedSuiteID == "" {
		return nil, fmt.Errorf("%w: suiteID cannot be empty", model.ErrValidation)
	}
	if source == nil {
		return nil, fmt.Errorf("%w: source archive cannot be nil", model.ErrValidation)
	}

	var targetPlatforms []model.Platform
	if platform != nil {
		if !platform.IsValid() {
			return nil, model.ErrInvalidPlatform
		}
		targetPlatforms = []model.Platform{*platform}
	} else {
		targetPlatforms = []model.Platform{model.PlatformLinuxAmd64, model.PlatformLinuxArm64}
	}

	log := zerolog.Ctx(ctx).With().
		Str("op", "BuildService.TriggerBuild").
		Str("suite_id", trimmedSuiteID).
		Logger()
	log.Debug().Msg("starting source upload and async build trigger")

	if s.suiteRepo != nil {
		if _, err := s.suiteRepo.FindByID(ctx, trimmedSuiteID); err != nil {
			log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed verifying test suite")
			return nil, err
		}
	}

	uploadReader := source
	uploadSize := size

	if s.staticAnalyzer != nil {
		preparedBytes, _, err := s.staticAnalyzer.PrepareSourceArchive(source, domainservice.StaticAnalysisOptions{
			AllowInsecureImports: opts.AllowInsecureImports,
		})
		if err != nil {
			log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("pre-build static analysis failed")
			return nil, err
		}
		uploadReader = bytes.NewReader(preparedBytes)
		uploadSize = int64(len(preparedBytes))
	}

	sourceKey := formatSourceKey(trimmedSuiteID)
	if err := s.storage.Upload(ctx, sourceKey, uploadReader, uploadSize, "application/gzip"); err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed uploading source archive to s3")
		return nil, fmt.Errorf("failed to upload source archive: %w", err)
	}

	existingArtifacts, err := s.artifactRepo.ListBySuiteID(ctx, trimmedSuiteID)
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed listing existing artifacts")
		return nil, err
	}

	artifacts := make([]*model.Artifact, 0, len(targetPlatforms))
	for _, p := range targetPlatforms {
		var found *model.Artifact
		for _, a := range existingArtifacts {
			if a.Platform() == p && a.Status() != model.ArtifactStatusReady {
				found = a
				break
			}
		}

		if found == nil {
			newArt, err := model.NewArtifact(trimmedSuiteID, p)
			if err != nil {
				log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed creating artifact model")
				return nil, err
			}
			if err := s.artifactRepo.Save(ctx, newArt); err != nil {
				log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed saving initial artifact")
				return nil, err
			}
			artifacts = append(artifacts, newArt)
		} else {
			// Reset a previously FAILED artifact so it can be re-compiled.
			if found.Status() == model.ArtifactStatusFailed {
				if err := found.RetryBuild(); err != nil {
					log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed resetting failed artifact for retry")
					return nil, err
				}
				if err := s.artifactRepo.Save(ctx, found); err != nil {
					log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed persisting artifact retry reset")
					return nil, err
				}
			}
			artifacts = append(artifacts, found)
		}
	}

	// Trigger build jobs asynchronously
	for _, art := range artifacts {
		artToBuild := art
		go func() {
			bgCtx := context.Background()
			bgLog := zerolog.Nop().With().
				Str("op", "BuildService.AsyncBuild").
				Str("suite_id", trimmedSuiteID).
				Str("artifact_id", artToBuild.ID()).
				Str("platform", string(artToBuild.Platform())).
				Logger()
			bgCtx = bgLog.WithContext(bgCtx)
			if _, err := s.BuildArtifact(bgCtx, trimmedSuiteID, artToBuild.ID()); err != nil {
				bgLog.Error().Err(err).Msg("asynchronous artifact build failed")
			} else {
				bgLog.Info().Msg("asynchronous artifact build completed successfully")
			}
		}()
	}

	log.Info().
		Int("artifacts_count", len(artifacts)).
		Dur("duration_ms", time.Since(start)).
		Msg("successfully triggered asynchronous build")

	return artifacts, nil
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
		defer func() {
			_ = execResult.Logs.Close()
		}()
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
			// Reset a previously FAILED artifact so it can be re-compiled.
			if found.Status() == model.ArtifactStatusFailed {
				if err := found.RetryBuild(); err != nil {
					log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed resetting failed artifact for retry")
					return nil, err
				}
				if err := s.artifactRepo.Save(ctx, found); err != nil {
					log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed persisting artifact retry reset")
					return nil, err
				}
			}
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

	if s.suiteRepo != nil {
		if _, err := s.suiteRepo.FindByID(ctx, trimmedSuiteID); err != nil {
			log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed finding suite for listing artifacts")
			return nil, err
		}
	}

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

// CancelBuild cancels an active or pending compilation build, terminates its Kubernetes Job, and transitions the artifact to CANCELLED.
func (s *BuildService) CancelBuild(ctx context.Context, suiteID, artifactID, reason string) (*model.Artifact, error) {
	start := time.Now()
	trimmedSuiteID := strings.TrimSpace(suiteID)
	trimmedArtifactID := strings.TrimSpace(artifactID)
	if trimmedSuiteID == "" || trimmedArtifactID == "" {
		return nil, fmt.Errorf("%w: suiteID and artifactID cannot be empty", model.ErrValidation)
	}

	trimmedReason := strings.TrimSpace(reason)
	if trimmedReason == "" {
		trimmedReason = "manual cancellation"
	}

	log := zerolog.Ctx(ctx).With().
		Str("op", "BuildService.CancelBuild").
		Str("suite_id", trimmedSuiteID).
		Str("artifact_id", trimmedArtifactID).
		Str("reason", trimmedReason).
		Logger()
	log.Debug().Msg("starting build cancellation")

	artifact, err := s.artifactRepo.FindByID(ctx, trimmedArtifactID)
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed fetching artifact to cancel")
		return nil, err
	}

	if artifact.SuiteID() != trimmedSuiteID {
		err := fmt.Errorf("%w: artifact %s does not belong to suite %s", model.ErrValidation, trimmedArtifactID, trimmedSuiteID)
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("suite ID mismatch")
		return nil, err
	}

	if artifact.Status() == model.ArtifactStatusReady || artifact.Status() == model.ArtifactStatusFailed || artifact.Status() == model.ArtifactStatusCancelled {
		err := fmt.Errorf("%w: artifact %s is already in terminal status %s", model.ErrTerminalState, trimmedArtifactID, artifact.Status())
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("cannot cancel artifact in terminal state")
		return nil, err
	}

	if s.orchestrator != nil {
		jobName := formatBuildJobName(artifact.ID())
		if err := s.orchestrator.DeleteJob(ctx, jobName); err != nil {
			log.Warn().Err(err).Str("job_name", jobName).Msg("deleting build job in kubernetes reported warning; continuing")
		}
	}

	if err := artifact.Cancel(trimmedReason); err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed transitioning artifact state to CANCELLED")
		return nil, err
	}

	if err := s.artifactRepo.Save(ctx, artifact); err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed persisting cancelled artifact")
		return nil, err
	}

	log.Info().
		Str("status", string(artifact.Status())).
		Dur("duration_ms", time.Since(start)).
		Msg("completed artifact build cancellation")

	return artifact, nil
}

// RetryBuild resets a FAILED or CANCELLED artifact to PENDING and triggers asynchronous re-compilation.
func (s *BuildService) RetryBuild(ctx context.Context, suiteID, artifactID string) (*model.Artifact, error) {
	start := time.Now()
	trimmedSuiteID := strings.TrimSpace(suiteID)
	trimmedArtifactID := strings.TrimSpace(artifactID)
	if trimmedSuiteID == "" || trimmedArtifactID == "" {
		return nil, fmt.Errorf("%w: suiteID and artifactID cannot be empty", model.ErrValidation)
	}

	log := zerolog.Ctx(ctx).With().
		Str("op", "BuildService.RetryBuild").
		Str("suite_id", trimmedSuiteID).
		Str("artifact_id", trimmedArtifactID).
		Logger()
	log.Debug().Msg("starting artifact build retry")

	if s.suiteRepo != nil {
		if _, err := s.suiteRepo.FindByID(ctx, trimmedSuiteID); err != nil {
			log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed verifying test suite for retry")
			return nil, err
		}
	}

	artifact, err := s.artifactRepo.FindByID(ctx, trimmedArtifactID)
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed fetching artifact to retry")
		return nil, err
	}

	if artifact.SuiteID() != trimmedSuiteID {
		err := fmt.Errorf("%w: artifact %s does not belong to suite %s", model.ErrValidation, trimmedArtifactID, trimmedSuiteID)
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("suite ID mismatch")
		return nil, err
	}

	if artifact.Status() != model.ArtifactStatusFailed && artifact.Status() != model.ArtifactStatusCancelled {
		err := fmt.Errorf("%w: only FAILED or CANCELLED artifacts can be retried (current status: %s)", model.ErrInvalidStateTransition, artifact.Status())
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("invalid artifact status for retry")
		return nil, err
	}

	sourceKey := formatSourceKey(trimmedSuiteID)
	exists, err := s.storage.Exists(ctx, sourceKey)
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed checking source archive existence")
		return nil, err
	}
	if !exists {
		err := fmt.Errorf("%w: source archive not found at %s", model.ErrNotFound, sourceKey)
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("source archive missing for retry")
		return nil, err
	}

	if err := artifact.RetryBuild(); err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed resetting artifact state for retry")
		return nil, err
	}

	if err := s.artifactRepo.Save(ctx, artifact); err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed persisting reset artifact")
		return nil, err
	}

	currentStatus := string(artifact.Status())
	platformStr := string(artifact.Platform())

	// Trigger asynchronous compilation
	go func() {
		bgCtx := context.Background()
		bgLog := zerolog.Nop().With().
			Str("op", "BuildService.AsyncRetryBuild").
			Str("suite_id", trimmedSuiteID).
			Str("artifact_id", trimmedArtifactID).
			Str("platform", platformStr).
			Logger()
		bgCtx = bgLog.WithContext(bgCtx)
		if _, err := s.BuildArtifact(bgCtx, trimmedSuiteID, trimmedArtifactID); err != nil {
			bgLog.Error().Err(err).Msg("asynchronous artifact retry build failed")
		} else {
			bgLog.Info().Msg("asynchronous artifact retry build completed successfully")
		}
	}()

	log.Info().
		Str("status", currentStatus).
		Dur("duration_ms", time.Since(start)).
		Msg("successfully initiated artifact build retry")

	return artifact, nil
}

// DeleteArtifact cleans up binary and log storage assets and removes an artifact record.
func (s *BuildService) DeleteArtifact(ctx context.Context, suiteID, artifactID string) error {
	start := time.Now()
	trimmedSuiteID := strings.TrimSpace(suiteID)
	trimmedArtifactID := strings.TrimSpace(artifactID)
	if trimmedSuiteID == "" || trimmedArtifactID == "" {
		return fmt.Errorf("%w: suiteID and artifactID cannot be empty", model.ErrValidation)
	}

	log := zerolog.Ctx(ctx).With().
		Str("op", "BuildService.DeleteArtifact").
		Str("suite_id", trimmedSuiteID).
		Str("artifact_id", trimmedArtifactID).
		Logger()
	log.Debug().Msg("starting artifact deletion")

	artifact, err := s.artifactRepo.FindByID(ctx, trimmedArtifactID)
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed fetching artifact for deletion")
		return err
	}

	if artifact.SuiteID() != trimmedSuiteID {
		err := fmt.Errorf("%w: artifact %s does not belong to suite %s", model.ErrValidation, trimmedArtifactID, trimmedSuiteID)
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("suite ID mismatch")
		return err
	}

	if artifact.Status() == model.ArtifactStatusBuilding {
		err := fmt.Errorf("%w: cannot delete artifact while compilation is active; cancel build first", model.ErrConflict)
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("active build deletion rejected")
		return err
	}

	if s.storage != nil {
		if artifact.S3BinaryKey() != "" {
			if err := s.storage.Delete(ctx, artifact.S3BinaryKey()); err != nil {
				log.Warn().Err(err).Str("s3_key", artifact.S3BinaryKey()).Msg("failed deleting artifact binary from storage; continuing")
			}
		}
		if artifact.BuildLogsS3Key() != "" {
			if err := s.storage.Delete(ctx, artifact.BuildLogsS3Key()); err != nil {
				log.Warn().Err(err).Str("s3_key", artifact.BuildLogsS3Key()).Msg("failed deleting build logs from storage; continuing")
			}
		}
	}

	if err := s.artifactRepo.Delete(ctx, trimmedArtifactID); err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed deleting artifact from repository")
		return err
	}

	log.Info().Dur("duration_ms", time.Since(start)).Msg("completed artifact deletion")
	return nil
}

func formatBuildJobName(artifactID string) string {
	cleaned := strings.ToLower(artifactID)
	name := fmt.Sprintf("vuhive-build-%s", cleaned)
	if len(name) > 63 {
		name = name[:63]
	}
	return strings.TrimRight(name, "-")
}

func (s *BuildService) computeSHA256FromStorage(ctx context.Context, binaryKey string) (string, error) {
	reader, err := s.storage.Download(ctx, binaryKey)
	if err != nil {
		return "", err
	}
	defer func() {
		_ = reader.Close()
	}()

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
