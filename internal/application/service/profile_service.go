package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/rs/zerolog"

	"github.com/morphy76/vuhive-cloud/internal/application/ports/inbound"
	"github.com/morphy76/vuhive-cloud/internal/application/ports/outbound"
	"github.com/morphy76/vuhive-cloud/internal/domain/model"
)

const (
	defaultCPURequest    = "1000m"
	defaultCPULimit      = "2000m"
	defaultMemoryRequest = "1Gi"
	defaultMemoryLimit   = "2Gi"
	defaultRunnerImage   = "alpine:3.20"
)

// ProfileService implements inbound.ProfilesUseCase to orchestrate runner profile lifecycle.
type ProfileService struct {
	profileRepo outbound.RunnerProfileRepository
}

// NewProfileService constructs a new ProfileService.
func NewProfileService(profileRepo outbound.RunnerProfileRepository) *ProfileService {
	return &ProfileService{
		profileRepo: profileRepo,
	}
}

// CreateProfile validates input parameters, applies defaults, creates a new RunnerProfile, and persists it.
func (s *ProfileService) CreateProfile(ctx context.Context, cmd inbound.CreateProfileCommand) (*model.RunnerProfile, error) {
	start := time.Now()
	name := strings.TrimSpace(cmd.Name)
	if name == "" {
		return nil, model.ErrEmptyName
	}

	log := zerolog.Ctx(ctx).With().
		Str("op", "ProfileService.CreateProfile").
		Str("profile_name", name).
		Logger()
	log.Debug().Msg("creating runner profile")

	cpuReq := strings.TrimSpace(cmd.CPURequest)
	if cpuReq == "" {
		cpuReq = defaultCPURequest
	}
	cpuLim := strings.TrimSpace(cmd.CPULimit)
	if cpuLim == "" {
		cpuLim = defaultCPULimit
	}
	memReq := strings.TrimSpace(cmd.MemoryRequest)
	if memReq == "" {
		memReq = defaultMemoryRequest
	}
	memLim := strings.TrimSpace(cmd.MemoryLimit)
	if memLim == "" {
		memLim = defaultMemoryLimit
	}

	resources, err := model.NewResourceRequirements(cpuReq, cpuLim, memReq, memLim)
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("invalid resource requirements")
		return nil, err
	}

	runnerImg := strings.TrimSpace(cmd.RunnerImage)
	if runnerImg == "" {
		runnerImg = defaultRunnerImage
	}

	profile, err := model.NewRunnerProfile(
		name,
		strings.TrimSpace(cmd.Description),
		runnerImg,
		resources,
		cmd.NodeSelector,
		cmd.Affinity,
		cmd.Tolerations,
	)
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed creating runner profile entity")
		return nil, err
	}

	if err := s.profileRepo.Save(ctx, profile); err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed saving runner profile to repository")
		return nil, err
	}

	log.Info().
		Str("profile_id", profile.ID()).
		Dur("duration_ms", time.Since(start)).
		Msg("successfully created runner profile")
	return profile, nil
}

// GetProfile retrieves a single RunnerProfile by ID.
func (s *ProfileService) GetProfile(ctx context.Context, id string) (*model.RunnerProfile, error) {
	start := time.Now()
	trimmedID := strings.TrimSpace(id)
	if trimmedID == "" {
		return nil, fmt.Errorf("%w: profile ID cannot be empty", model.ErrValidation)
	}

	log := zerolog.Ctx(ctx).With().
		Str("op", "ProfileService.GetProfile").
		Str("profile_id", trimmedID).
		Logger()
	log.Debug().Msg("retrieving runner profile")

	profile, err := s.profileRepo.FindByID(ctx, trimmedID)
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed retrieving runner profile")
		return nil, err
	}

	log.Info().Dur("duration_ms", time.Since(start)).Msg("successfully retrieved runner profile")
	return profile, nil
}

// ListProfiles retrieves all available RunnerProfiles.
func (s *ProfileService) ListProfiles(ctx context.Context) ([]*model.RunnerProfile, error) {
	start := time.Now()
	log := zerolog.Ctx(ctx).With().
		Str("op", "ProfileService.ListProfiles").
		Logger()
	log.Debug().Msg("listing runner profiles")

	profiles, err := s.profileRepo.List(ctx)
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed listing runner profiles")
		return nil, err
	}

	log.Info().
		Int("count", len(profiles)).
		Dur("duration_ms", time.Since(start)).
		Msg("successfully listed runner profiles")
	return profiles, nil
}

// UpdateProfile updates the attributes and Kubernetes specifications of an existing RunnerProfile.
func (s *ProfileService) UpdateProfile(ctx context.Context, id string, cmd inbound.UpdateProfileCommand) (*model.RunnerProfile, error) {
	start := time.Now()
	trimmedID := strings.TrimSpace(id)
	if trimmedID == "" {
		return nil, fmt.Errorf("%w: profile ID cannot be empty", model.ErrValidation)
	}

	log := zerolog.Ctx(ctx).With().
		Str("op", "ProfileService.UpdateProfile").
		Str("profile_id", trimmedID).
		Logger()
	log.Debug().Msg("updating runner profile")

	existing, err := s.profileRepo.FindByID(ctx, trimmedID)
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed finding runner profile for update")
		return nil, err
	}

	name := strings.TrimSpace(cmd.Name)
	if name == "" {
		return nil, model.ErrEmptyName
	}

	cpuReq := strings.TrimSpace(cmd.CPURequest)
	if cpuReq == "" {
		cpuReq = existing.Resources().CPURequest()
	}
	cpuLim := strings.TrimSpace(cmd.CPULimit)
	if cpuLim == "" {
		cpuLim = existing.Resources().CPULimit()
	}
	memReq := strings.TrimSpace(cmd.MemoryRequest)
	if memReq == "" {
		memReq = existing.Resources().MemoryRequest()
	}
	memLim := strings.TrimSpace(cmd.MemoryLimit)
	if memLim == "" {
		memLim = existing.Resources().MemoryLimit()
	}

	resources, err := model.NewResourceRequirements(cpuReq, cpuLim, memReq, memLim)
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("invalid resource requirements during update")
		return nil, err
	}

	runnerImg := strings.TrimSpace(cmd.RunnerImage)
	if runnerImg == "" {
		runnerImg = existing.RunnerImage()
	}

	if err := existing.UpdateDetails(
		name,
		strings.TrimSpace(cmd.Description),
		runnerImg,
		resources,
		cmd.NodeSelector,
		cmd.Affinity,
		cmd.Tolerations,
	); err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed updating runner profile details")
		return nil, err
	}

	if err := s.profileRepo.Save(ctx, existing); err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed saving updated runner profile")
		return nil, err
	}

	log.Info().Dur("duration_ms", time.Since(start)).Msg("successfully updated runner profile")
	return existing, nil
}

// DeleteProfile deletes a RunnerProfile by ID.
func (s *ProfileService) DeleteProfile(ctx context.Context, id string) error {
	start := time.Now()
	trimmedID := strings.TrimSpace(id)
	if trimmedID == "" {
		return fmt.Errorf("%w: profile ID cannot be empty", model.ErrValidation)
	}

	log := zerolog.Ctx(ctx).With().
		Str("op", "ProfileService.DeleteProfile").
		Str("profile_id", trimmedID).
		Logger()
	log.Debug().Msg("deleting runner profile")

	if err := s.profileRepo.Delete(ctx, trimmedID); err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed deleting runner profile")
		return err
	}

	log.Info().Dur("duration_ms", time.Since(start)).Msg("successfully deleted runner profile")
	return nil
}

// Compile-time interface assertion
var _ inbound.ProfilesUseCase = (*ProfileService)(nil)
