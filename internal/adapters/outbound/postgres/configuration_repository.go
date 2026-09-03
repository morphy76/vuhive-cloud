package postgres

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/morphy76/vuhive-cloud/internal/application/ports/outbound"
	"github.com/morphy76/vuhive-cloud/internal/domain/model"
	"github.com/rs/zerolog"
)

// ConfigurationRepository implements outbound.ConfigurationRepository using pgxpool.
type ConfigurationRepository struct {
	pool *pgxpool.Pool
}

// NewConfigurationRepository constructs a new ConfigurationRepository.
func NewConfigurationRepository(pool *pgxpool.Pool) *ConfigurationRepository {
	return &ConfigurationRepository{pool: pool}
}

// Save inserts or updates a Configuration entity.
func (r *ConfigurationRepository) Save(ctx context.Context, config *model.Configuration) error {
	start := time.Now()
	log := zerolog.Ctx(ctx).With().Str("op", "ConfigurationRepository.Save").Str("config_id", config.ID()).Logger()
	log.Debug().Msg("saving configuration")

	query := `
		INSERT INTO configurations (id, suite_id, name, content_yaml, s3_config_key, is_default, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (id) DO UPDATE SET
			suite_id = EXCLUDED.suite_id,
			name = EXCLUDED.name,
			content_yaml = EXCLUDED.content_yaml,
			s3_config_key = EXCLUDED.s3_config_key,
			is_default = EXCLUDED.is_default
	`
	_, err := r.pool.Exec(ctx, query,
		config.ID(),
		config.SuiteID(),
		config.Name(),
		config.ContentYAML(),
		config.S3ConfigKey(),
		config.IsDefault(),
		config.CreatedAt(),
	)
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed to save configuration")
		return MapError(err)
	}

	log.Info().Dur("duration_ms", time.Since(start)).Msg("successfully saved configuration")
	return nil
}

// FindByID retrieves a Configuration entity by ID.
func (r *ConfigurationRepository) FindByID(ctx context.Context, id string) (*model.Configuration, error) {
	start := time.Now()
	log := zerolog.Ctx(ctx).With().Str("op", "ConfigurationRepository.FindByID").Str("config_id", id).Logger()
	log.Debug().Msg("finding configuration by id")

	query := `
		SELECT id, suite_id, name, content_yaml, s3_config_key, is_default, created_at
		FROM configurations
		WHERE id = $1
	`
	var (
		configID    string
		suiteID     string
		name        string
		contentYAML string
		s3ConfigKey string
		isDefault   bool
		createdAt   time.Time
	)
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&configID, &suiteID, &name, &contentYAML, &s3ConfigKey, &isDefault, &createdAt,
	)
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed to find configuration by id")
		return nil, MapError(err)
	}

	config, err := model.NewConfigurationWithID(
		configID, suiteID, name, contentYAML, s3ConfigKey, isDefault, createdAt,
	)
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed to reconstitute configuration")
		return nil, err
	}

	log.Info().Dur("duration_ms", time.Since(start)).Msg("successfully found configuration by id")
	return config, nil
}

// ListBySuiteID returns all Configuration entities for a given test suite.
func (r *ConfigurationRepository) ListBySuiteID(ctx context.Context, suiteID string) ([]*model.Configuration, error) {
	start := time.Now()
	log := zerolog.Ctx(ctx).With().Str("op", "ConfigurationRepository.ListBySuiteID").Str("suite_id", suiteID).Logger()
	log.Debug().Msg("listing configurations by suite id")

	query := `
		SELECT id, suite_id, name, content_yaml, s3_config_key, is_default, created_at
		FROM configurations
		WHERE suite_id = $1
		ORDER BY created_at ASC
	`
	rows, err := r.pool.Query(ctx, query, suiteID)
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed to query configurations by suite id")
		return nil, MapError(err)
	}
	defer rows.Close()

	var configs []*model.Configuration
	for rows.Next() {
		var (
			configID    string
			sID         string
			name        string
			contentYAML string
			s3ConfigKey string
			isDefault   bool
			createdAt   time.Time
		)
		if err := rows.Scan(
			&configID, &sID, &name, &contentYAML, &s3ConfigKey, &isDefault, &createdAt,
		); err != nil {
			log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed to scan configuration row")
			return nil, MapError(err)
		}
		config, err := model.NewConfigurationWithID(
			configID, sID, name, contentYAML, s3ConfigKey, isDefault, createdAt,
		)
		if err != nil {
			log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed to reconstitute configuration from row")
			return nil, err
		}
		configs = append(configs, config)
	}

	if err := rows.Err(); err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("error iterating configuration rows")
		return nil, MapError(err)
	}

	log.Info().Int("count", len(configs)).Dur("duration_ms", time.Since(start)).Msg("successfully listed configurations by suite id")
	return configs, nil
}

// Delete removes a Configuration by ID.
func (r *ConfigurationRepository) Delete(ctx context.Context, id string) error {
	start := time.Now()
	log := zerolog.Ctx(ctx).With().Str("op", "ConfigurationRepository.Delete").Str("config_id", id).Logger()
	log.Debug().Msg("deleting configuration")

	cmdTag, err := r.pool.Exec(ctx, `DELETE FROM configurations WHERE id = $1`, id)
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed to delete configuration")
		return MapError(err)
	}

	if cmdTag.RowsAffected() == 0 {
		log.Error().Dur("duration_ms", time.Since(start)).Msg("configuration not found for deletion")
		return model.ErrNotFound
	}

	log.Info().Dur("duration_ms", time.Since(start)).Msg("successfully deleted configuration")
	return nil
}

// Static compile-time interface assertion
var _ outbound.ConfigurationRepository = (*ConfigurationRepository)(nil)
