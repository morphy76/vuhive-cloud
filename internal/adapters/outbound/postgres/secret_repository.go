package postgres

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/morphy76/vuhive-cloud/internal/application/ports/outbound"
	"github.com/morphy76/vuhive-cloud/internal/domain/model"
	"github.com/rs/zerolog"
)

// SecretRepository implements outbound.SecretRepository using pgxpool.
type SecretRepository struct {
	pool *pgxpool.Pool
}

// NewSecretRepository constructs a new SecretRepository.
func NewSecretRepository(pool *pgxpool.Pool) *SecretRepository {
	return &SecretRepository{pool: pool}
}

// Save inserts or updates a Secret entity.
func (r *SecretRepository) Save(ctx context.Context, secret *model.Secret) error {
	start := time.Now()
	log := zerolog.Ctx(ctx).With().Str("op", "SecretRepository.Save").Str("secret_id", secret.ID()).Logger()
	log.Debug().Msg("saving secret")

	query := `
		INSERT INTO secrets (id, suite_id, key, encrypted_value, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (id) DO UPDATE SET
			encrypted_value = EXCLUDED.encrypted_value,
			updated_at = EXCLUDED.updated_at
	`
	_, err := r.pool.Exec(ctx, query,
		secret.ID(),
		secret.SuiteID(),
		secret.Key(),
		secret.EncryptedValue(),
		secret.CreatedAt(),
		secret.UpdatedAt(),
	)
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed to save secret")
		return MapError(err)
	}

	log.Info().Dur("duration_ms", time.Since(start)).Msg("successfully saved secret")
	return nil
}

// FindByID retrieves a Secret entity by ID.
func (r *SecretRepository) FindByID(ctx context.Context, id string) (*model.Secret, error) {
	start := time.Now()
	log := zerolog.Ctx(ctx).With().Str("op", "SecretRepository.FindByID").Str("secret_id", id).Logger()
	log.Debug().Msg("finding secret by id")

	query := `
		SELECT id, suite_id, key, encrypted_value, created_at, updated_at
		FROM secrets
		WHERE id = $1
	`
	var (
		secretID       string
		suiteID        string
		key            string
		encryptedValue []byte
		createdAt      time.Time
		updatedAt      time.Time
	)
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&secretID, &suiteID, &key, &encryptedValue, &createdAt, &updatedAt,
	)
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed to find secret by id")
		return nil, MapError(err)
	}

	secret, err := model.NewSecretWithID(secretID, suiteID, key, encryptedValue, createdAt, updatedAt)
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed to reconstitute secret")
		return nil, err
	}

	log.Info().Dur("duration_ms", time.Since(start)).Msg("successfully found secret by id")
	return secret, nil
}

// FindBySuiteIDAndKey retrieves a Secret entity by suite ID and key name.
func (r *SecretRepository) FindBySuiteIDAndKey(ctx context.Context, suiteID, key string) (*model.Secret, error) {
	start := time.Now()
	log := zerolog.Ctx(ctx).With().Str("op", "SecretRepository.FindBySuiteIDAndKey").Str("suite_id", suiteID).Str("key", key).Logger()
	log.Debug().Msg("finding secret by suite id and key")

	query := `
		SELECT id, suite_id, key, encrypted_value, created_at, updated_at
		FROM secrets
		WHERE suite_id = $1 AND key = $2
	`
	var (
		secretID       string
		sID            string
		k              string
		encryptedValue []byte
		createdAt      time.Time
		updatedAt      time.Time
	)
	err := r.pool.QueryRow(ctx, query, suiteID, key).Scan(
		&secretID, &sID, &k, &encryptedValue, &createdAt, &updatedAt,
	)
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed to find secret by suite id and key")
		return nil, MapError(err)
	}

	secret, err := model.NewSecretWithID(secretID, sID, k, encryptedValue, createdAt, updatedAt)
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed to reconstitute secret")
		return nil, err
	}

	log.Info().Dur("duration_ms", time.Since(start)).Msg("successfully found secret by suite id and key")
	return secret, nil
}

// ListBySuiteID returns all Secret entities for a given test suite.
func (r *SecretRepository) ListBySuiteID(ctx context.Context, suiteID string) ([]*model.Secret, error) {
	start := time.Now()
	log := zerolog.Ctx(ctx).With().Str("op", "SecretRepository.ListBySuiteID").Str("suite_id", suiteID).Logger()
	log.Debug().Msg("listing secrets by suite id")

	query := `
		SELECT id, suite_id, key, encrypted_value, created_at, updated_at
		FROM secrets
		WHERE suite_id = $1
		ORDER BY key ASC
	`
	rows, err := r.pool.Query(ctx, query, suiteID)
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed to query secrets by suite id")
		return nil, MapError(err)
	}
	defer rows.Close()

	var secrets []*model.Secret
	for rows.Next() {
		var (
			secretID       string
			sID            string
			key            string
			encryptedValue []byte
			createdAt      time.Time
			updatedAt      time.Time
		)
		if err := rows.Scan(
			&secretID, &sID, &key, &encryptedValue, &createdAt, &updatedAt,
		); err != nil {
			log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed to scan secret row")
			return nil, MapError(err)
		}
		secret, err := model.NewSecretWithID(secretID, sID, key, encryptedValue, createdAt, updatedAt)
		if err != nil {
			log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed to reconstitute secret from row")
			return nil, err
		}
		secrets = append(secrets, secret)
	}

	if err := rows.Err(); err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("error iterating secret rows")
		return nil, MapError(err)
	}

	log.Info().Int("count", len(secrets)).Dur("duration_ms", time.Since(start)).Msg("successfully listed secrets by suite id")
	return secrets, nil
}

// Delete removes a Secret by ID.
func (r *SecretRepository) Delete(ctx context.Context, id string) error {
	start := time.Now()
	log := zerolog.Ctx(ctx).With().Str("op", "SecretRepository.Delete").Str("secret_id", id).Logger()
	log.Debug().Msg("deleting secret")

	cmdTag, err := r.pool.Exec(ctx, `DELETE FROM secrets WHERE id = $1`, id)
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed to delete secret")
		return MapError(err)
	}

	if cmdTag.RowsAffected() == 0 {
		log.Error().Dur("duration_ms", time.Since(start)).Msg("secret not found for deletion")
		return model.ErrNotFound
	}

	log.Info().Dur("duration_ms", time.Since(start)).Msg("successfully deleted secret")
	return nil
}

// Static compile-time interface assertion
var _ outbound.SecretRepository = (*SecretRepository)(nil)
