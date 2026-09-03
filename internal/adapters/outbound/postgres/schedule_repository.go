package postgres

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/morphy76/vuhive-cloud/internal/application/ports/outbound"
	"github.com/morphy76/vuhive-cloud/internal/domain/model"
	"github.com/rs/zerolog"
)

// ScheduleRepository implements outbound.ScheduleRepository using pgxpool.
type ScheduleRepository struct {
	pool *pgxpool.Pool
}

// NewScheduleRepository constructs a new ScheduleRepository.
func NewScheduleRepository(pool *pgxpool.Pool) *ScheduleRepository {
	return &ScheduleRepository{pool: pool}
}

// Save inserts or updates a Schedule aggregate.
func (r *ScheduleRepository) Save(ctx context.Context, schedule *model.Schedule) error {
	start := time.Now()
	log := zerolog.Ctx(ctx).With().Str("op", "ScheduleRepository.Save").Str("schedule_id", schedule.ID()).Logger()
	log.Debug().Msg("saving schedule")

	query := `
		INSERT INTO schedules (
			id, suite_id, artifact_id, configuration_id, runner_profile_id,
			name, cron_expression, k8s_cronjob_name, is_active,
			created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (id) DO UPDATE SET
			suite_id = EXCLUDED.suite_id,
			artifact_id = EXCLUDED.artifact_id,
			configuration_id = EXCLUDED.configuration_id,
			runner_profile_id = EXCLUDED.runner_profile_id,
			name = EXCLUDED.name,
			cron_expression = EXCLUDED.cron_expression,
			k8s_cronjob_name = EXCLUDED.k8s_cronjob_name,
			is_active = EXCLUDED.is_active,
			updated_at = EXCLUDED.updated_at
	`
	_, err := r.pool.Exec(ctx, query,
		schedule.ID(),
		schedule.SuiteID(),
		schedule.ArtifactID(),
		schedule.ConfigurationID(),
		schedule.RunnerProfileID(),
		schedule.Name(),
		schedule.CronExpression(),
		schedule.K8sCronJobName(),
		schedule.IsActive(),
		schedule.CreatedAt(),
		schedule.UpdatedAt(),
	)
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed to save schedule")
		return MapError(err)
	}

	log.Info().Dur("duration_ms", time.Since(start)).Msg("successfully saved schedule")
	return nil
}

// FindByID retrieves a Schedule aggregate by ID.
func (r *ScheduleRepository) FindByID(ctx context.Context, id string) (*model.Schedule, error) {
	start := time.Now()
	log := zerolog.Ctx(ctx).With().Str("op", "ScheduleRepository.FindByID").Str("schedule_id", id).Logger()
	log.Debug().Msg("finding schedule by id")

	query := `
		SELECT id, suite_id, artifact_id, configuration_id, runner_profile_id,
		       name, cron_expression, k8s_cronjob_name, is_active,
		       created_at, updated_at
		FROM schedules
		WHERE id = $1
	`
	var (
		scheduleID      string
		suiteID         string
		artifactID      string
		configurationID *string
		runnerProfileID string
		name            string
		cronExpression  string
		k8sCronJobName  string
		isActive        bool
		createdAt       time.Time
		updatedAt       time.Time
	)
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&scheduleID, &suiteID, &artifactID, &configurationID, &runnerProfileID,
		&name, &cronExpression, &k8sCronJobName, &isActive,
		&createdAt, &updatedAt,
	)
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed to find schedule by id")
		return nil, MapError(err)
	}

	schedule, err := model.NewScheduleWithID(
		scheduleID, suiteID, artifactID, configurationID, runnerProfileID,
		name, cronExpression, k8sCronJobName, isActive,
		createdAt, updatedAt,
	)
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed to reconstitute schedule")
		return nil, err
	}

	log.Info().Dur("duration_ms", time.Since(start)).Msg("successfully found schedule by id")
	return schedule, nil
}

// ListBySuiteID returns all Schedule aggregates for a given test suite.
func (r *ScheduleRepository) ListBySuiteID(ctx context.Context, suiteID string) ([]*model.Schedule, error) {
	start := time.Now()
	log := zerolog.Ctx(ctx).With().Str("op", "ScheduleRepository.ListBySuiteID").Str("suite_id", suiteID).Logger()
	log.Debug().Msg("listing schedules by suite id")

	query := `
		SELECT id, suite_id, artifact_id, configuration_id, runner_profile_id,
		       name, cron_expression, k8s_cronjob_name, is_active,
		       created_at, updated_at
		FROM schedules
		WHERE suite_id = $1
		ORDER BY created_at ASC
	`
	rows, err := r.pool.Query(ctx, query, suiteID)
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed to query schedules by suite id")
		return nil, MapError(err)
	}
	defer rows.Close()

	var schedules []*model.Schedule
	for rows.Next() {
		var (
			scheduleID      string
			sID             string
			artifactID      string
			configurationID *string
			runnerProfileID string
			name            string
			cronExpression  string
			k8sCronJobName  string
			isActive        bool
			createdAt       time.Time
			updatedAt       time.Time
		)
		if err := rows.Scan(
			&scheduleID, &sID, &artifactID, &configurationID, &runnerProfileID,
			&name, &cronExpression, &k8sCronJobName, &isActive,
			&createdAt, &updatedAt,
		); err != nil {
			log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed to scan schedule row")
			return nil, MapError(err)
		}
		schedule, err := model.NewScheduleWithID(
			scheduleID, sID, artifactID, configurationID, runnerProfileID,
			name, cronExpression, k8sCronJobName, isActive,
			createdAt, updatedAt,
		)
		if err != nil {
			log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed to reconstitute schedule from row")
			return nil, err
		}
		schedules = append(schedules, schedule)
	}

	if err := rows.Err(); err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("error iterating schedule rows")
		return nil, MapError(err)
	}

	log.Info().Int("count", len(schedules)).Dur("duration_ms", time.Since(start)).Msg("successfully listed schedules by suite id")
	return schedules, nil
}

// ListActive returns all active Schedule aggregates across all suites.
func (r *ScheduleRepository) ListActive(ctx context.Context) ([]*model.Schedule, error) {
	start := time.Now()
	log := zerolog.Ctx(ctx).With().Str("op", "ScheduleRepository.ListActive").Logger()
	log.Debug().Msg("listing active schedules")

	query := `
		SELECT id, suite_id, artifact_id, configuration_id, runner_profile_id,
		       name, cron_expression, k8s_cronjob_name, is_active,
		       created_at, updated_at
		FROM schedules
		WHERE is_active = TRUE
		ORDER BY created_at ASC
	`
	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed to query active schedules")
		return nil, MapError(err)
	}
	defer rows.Close()

	var schedules []*model.Schedule
	for rows.Next() {
		var (
			scheduleID      string
			suiteID         string
			artifactID      string
			configurationID *string
			runnerProfileID string
			name            string
			cronExpression  string
			k8sCronJobName  string
			isActive        bool
			createdAt       time.Time
			updatedAt       time.Time
		)
		if err := rows.Scan(
			&scheduleID, &suiteID, &artifactID, &configurationID, &runnerProfileID,
			&name, &cronExpression, &k8sCronJobName, &isActive,
			&createdAt, &updatedAt,
		); err != nil {
			log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed to scan active schedule row")
			return nil, MapError(err)
		}
		schedule, err := model.NewScheduleWithID(
			scheduleID, suiteID, artifactID, configurationID, runnerProfileID,
			name, cronExpression, k8sCronJobName, isActive,
			createdAt, updatedAt,
		)
		if err != nil {
			log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed to reconstitute schedule from row")
			return nil, err
		}
		schedules = append(schedules, schedule)
	}

	if err := rows.Err(); err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("error iterating active schedule rows")
		return nil, MapError(err)
	}

	log.Info().Int("count", len(schedules)).Dur("duration_ms", time.Since(start)).Msg("successfully listed active schedules")
	return schedules, nil
}

// Delete removes a Schedule by ID.
func (r *ScheduleRepository) Delete(ctx context.Context, id string) error {
	start := time.Now()
	log := zerolog.Ctx(ctx).With().Str("op", "ScheduleRepository.Delete").Str("schedule_id", id).Logger()
	log.Debug().Msg("deleting schedule")

	cmdTag, err := r.pool.Exec(ctx, `DELETE FROM schedules WHERE id = $1`, id)
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed to delete schedule")
		return MapError(err)
	}

	if cmdTag.RowsAffected() == 0 {
		log.Error().Dur("duration_ms", time.Since(start)).Msg("schedule not found for deletion")
		return model.ErrNotFound
	}

	log.Info().Dur("duration_ms", time.Since(start)).Msg("successfully deleted schedule")
	return nil
}

// Static compile-time interface assertion
var _ outbound.ScheduleRepository = (*ScheduleRepository)(nil)
