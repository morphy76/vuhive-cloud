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

// ScheduleService implements inbound.SchedulesUseCase to orchestrate test schedule creation,
// native Kubernetes CronJob management, and recurring execution tracking.
type ScheduleService struct {
	suiteRepo    outbound.TestSuiteRepository
	artifactRepo outbound.ArtifactRepository
	configRepo   outbound.ConfigurationRepository
	profileRepo  outbound.RunnerProfileRepository
	scheduleRepo outbound.ScheduleRepository
	orchestrator outbound.ScheduleOrchestratorPort
}

// NewScheduleService constructs a new ScheduleService.
func NewScheduleService(
	suiteRepo outbound.TestSuiteRepository,
	artifactRepo outbound.ArtifactRepository,
	configRepo outbound.ConfigurationRepository,
	profileRepo outbound.RunnerProfileRepository,
	scheduleRepo outbound.ScheduleRepository,
	orchestrator outbound.ScheduleOrchestratorPort,
) *ScheduleService {
	return &ScheduleService{
		suiteRepo:    suiteRepo,
		artifactRepo: artifactRepo,
		configRepo:   configRepo,
		profileRepo:  profileRepo,
		scheduleRepo: scheduleRepo,
		orchestrator: orchestrator,
	}
}

// CreateSchedule validates dependencies, persists a new Schedule aggregate, and provisions a native K8s CronJob.
func (s *ScheduleService) CreateSchedule(
	ctx context.Context,
	suiteID, artifactID string,
	configID *string,
	runnerProfileID, name, cronExpr string,
) (*model.Schedule, error) {
	start := time.Now()
	trimmedSuiteID := strings.TrimSpace(suiteID)
	trimmedArtifactID := strings.TrimSpace(artifactID)
	trimmedProfileID := strings.TrimSpace(runnerProfileID)
	trimmedName := strings.TrimSpace(name)
	trimmedCronExpr := strings.TrimSpace(cronExpr)

	log := zerolog.Ctx(ctx).With().
		Str("op", "ScheduleService.CreateSchedule").
		Str("suite_id", trimmedSuiteID).
		Str("artifact_id", trimmedArtifactID).
		Str("profile_id", trimmedProfileID).
		Str("name", trimmedName).
		Str("cron_expression", trimmedCronExpr).
		Logger()
	log.Debug().Msg("starting schedule creation")

	if trimmedSuiteID == "" || trimmedArtifactID == "" || trimmedProfileID == "" {
		err := fmt.Errorf("%w: suite_id, artifact_id, and runner_profile_id are required", model.ErrValidation)
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("create schedule validation failed")
		return nil, err
	}
	if trimmedName == "" {
		err := model.ErrEmptyName
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("schedule name cannot be empty")
		return nil, err
	}
	if err := model.ValidateCronExpression(trimmedCronExpr); err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("invalid cron expression")
		return nil, err
	}

	// 1. Verify TestSuite exists and is ACTIVE
	suite, err := s.suiteRepo.FindByID(ctx, trimmedSuiteID)
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed fetching test suite")
		return nil, err
	}
	if suite.State() != model.TestSuiteStateActive {
		err := fmt.Errorf("%w: test suite %s is not active (status: %s)", model.ErrInvalidStateTransition, trimmedSuiteID, suite.State())
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("test suite is not in active state")
		return nil, err
	}

	// 2. Verify Artifact exists, belongs to suite, and is READY
	artifact, err := s.artifactRepo.FindByID(ctx, trimmedArtifactID)
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed fetching artifact")
		return nil, err
	}
	if artifact.SuiteID() != suite.ID() {
		err := fmt.Errorf("%w: artifact %s does not belong to suite %s", model.ErrValidation, trimmedArtifactID, trimmedSuiteID)
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("artifact does not belong to suite")
		return nil, err
	}
	if artifact.Status() != model.ArtifactStatusReady {
		err := fmt.Errorf("%w: artifact %s is not ready (status: %s)", model.ErrInvalidStateTransition, trimmedArtifactID, artifact.Status())
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("artifact is not ready for execution")
		return nil, err
	}

	// 3. Verify Configuration if provided
	var configKey string
	if configID != nil && strings.TrimSpace(*configID) != "" {
		cfgID := strings.TrimSpace(*configID)
		cfg, err := s.configRepo.FindByID(ctx, cfgID)
		if err != nil {
			log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed fetching configuration")
			return nil, err
		}
		if cfg.SuiteID() != suite.ID() {
			err := fmt.Errorf("%w: configuration %s does not belong to suite %s", model.ErrValidation, cfgID, trimmedSuiteID)
			log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("configuration does not belong to suite")
			return nil, err
		}
		configKey = cfg.S3ConfigKey()
	}

	// 4. Verify RunnerProfile exists
	profile, err := s.profileRepo.FindByID(ctx, trimmedProfileID)
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed fetching runner profile")
		return nil, err
	}

	// 5. Create Schedule aggregate
	schedule, err := model.NewSchedule(
		suite.ID(),
		artifact.ID(),
		configID,
		profile.ID(),
		trimmedName,
		trimmedCronExpr,
	)
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed creating schedule aggregate")
		return nil, err
	}

	// 6. Save Schedule in database
	if err := s.scheduleRepo.Save(ctx, schedule); err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed saving schedule to repository")
		return nil, err
	}

	// 7. Provision native Kubernetes CronJob
	jobOpts := outbound.RunnerJobOptions{
		S3BinaryKey: artifact.S3BinaryKey(),
		S3ConfigKey: configKey,
	}

	_, err = s.orchestrator.CreateCronJob(ctx, schedule, profile, jobOpts)
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed creating native cronjob in kubernetes; rolling back")
		_ = s.scheduleRepo.Delete(ctx, schedule.ID())
		return nil, err
	}

	log.Info().
		Str("schedule_id", schedule.ID()).
		Str("k8s_cronjob_name", schedule.K8sCronJobName()).
		Dur("duration_ms", time.Since(start)).
		Msg("completed schedule creation")

	return schedule, nil
}

// GetSchedule retrieves a Schedule aggregate by its unique identifier.
func (s *ScheduleService) GetSchedule(ctx context.Context, id string) (*model.Schedule, error) {
	start := time.Now()
	trimmedID := strings.TrimSpace(id)
	if trimmedID == "" {
		return nil, fmt.Errorf("%w: schedule id cannot be empty", model.ErrValidation)
	}

	log := zerolog.Ctx(ctx).With().
		Str("op", "ScheduleService.GetSchedule").
		Str("schedule_id", trimmedID).
		Logger()
	log.Debug().Msg("fetching schedule")

	schedule, err := s.scheduleRepo.FindByID(ctx, trimmedID)
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed fetching schedule by id")
		return nil, err
	}

	log.Info().
		Str("name", schedule.Name()).
		Dur("duration_ms", time.Since(start)).
		Msg("completed schedule retrieval")

	return schedule, nil
}

// ListSchedules returns all active Schedule aggregates.
func (s *ScheduleService) ListSchedules(ctx context.Context) ([]*model.Schedule, error) {
	start := time.Now()
	log := zerolog.Ctx(ctx).With().
		Str("op", "ScheduleService.ListSchedules").
		Logger()
	log.Debug().Msg("listing active schedules")

	schedules, err := s.scheduleRepo.ListActive(ctx)
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed listing active schedules")
		return nil, err
	}

	log.Info().
		Int("count", len(schedules)).
		Dur("duration_ms", time.Since(start)).
		Msg("completed active schedules listing")

	return schedules, nil
}

// UpdateSchedule updates the cron expression and synchronizes with the native Kubernetes CronJob.
func (s *ScheduleService) UpdateSchedule(ctx context.Context, id string, cronExpr string) (*model.Schedule, error) {
	start := time.Now()
	trimmedID := strings.TrimSpace(id)
	trimmedCronExpr := strings.TrimSpace(cronExpr)

	log := zerolog.Ctx(ctx).With().
		Str("op", "ScheduleService.UpdateSchedule").
		Str("schedule_id", trimmedID).
		Str("cron_expression", trimmedCronExpr).
		Logger()
	log.Debug().Msg("starting schedule update")

	if trimmedID == "" {
		return nil, fmt.Errorf("%w: schedule id cannot be empty", model.ErrValidation)
	}

	schedule, err := s.scheduleRepo.FindByID(ctx, trimmedID)
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed finding schedule to update")
		return nil, err
	}

	if err := schedule.UpdateCronExpression(trimmedCronExpr); err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed updating schedule cron expression")
		return nil, err
	}

	if err := s.orchestrator.UpdateCronJob(ctx, schedule); err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed updating native cronjob in kubernetes")
		return nil, err
	}

	if err := s.scheduleRepo.Save(ctx, schedule); err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed persisting updated schedule")
		return nil, err
	}

	log.Info().
		Dur("duration_ms", time.Since(start)).
		Msg("completed schedule update")

	return schedule, nil
}

// DeleteSchedule deletes the native Kubernetes CronJob and removes the Schedule aggregate from repository.
func (s *ScheduleService) DeleteSchedule(ctx context.Context, id string) error {
	start := time.Now()
	trimmedID := strings.TrimSpace(id)
	if trimmedID == "" {
		return fmt.Errorf("%w: schedule id cannot be empty", model.ErrValidation)
	}

	log := zerolog.Ctx(ctx).With().
		Str("op", "ScheduleService.DeleteSchedule").
		Str("schedule_id", trimmedID).
		Logger()
	log.Debug().Msg("starting schedule deletion")

	schedule, err := s.scheduleRepo.FindByID(ctx, trimmedID)
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed finding schedule for deletion")
		return err
	}

	// Delete from Kubernetes
	if err := s.orchestrator.DeleteCronJob(ctx, schedule.K8sCronJobName(), ""); err != nil {
		log.Warn().Err(err).Msg("error while deleting cronjob from kubernetes; continuing with database deletion")
	}

	// Delete from Database
	if err := s.scheduleRepo.Delete(ctx, trimmedID); err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed deleting schedule from repository")
		return err
	}

	log.Info().
		Dur("duration_ms", time.Since(start)).
		Msg("completed schedule deletion")

	return nil
}

// Compile-time static interface verification
var _ inbound.SchedulesUseCase = (*ScheduleService)(nil)
