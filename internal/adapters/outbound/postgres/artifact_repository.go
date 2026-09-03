package postgres

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/morphy76/vuhive-cloud/internal/application/ports/outbound"
	"github.com/morphy76/vuhive-cloud/internal/domain/model"
	"github.com/rs/zerolog"
)

// ArtifactRepository implements outbound.ArtifactRepository using pgxpool.
type ArtifactRepository struct {
	pool *pgxpool.Pool
}

// NewArtifactRepository constructs a new ArtifactRepository.
func NewArtifactRepository(pool *pgxpool.Pool) *ArtifactRepository {
	return &ArtifactRepository{pool: pool}
}

// Save inserts or updates an Artifact entity.
func (r *ArtifactRepository) Save(ctx context.Context, artifact *model.Artifact) error {
	start := time.Now()
	log := zerolog.Ctx(ctx).With().Str("op", "ArtifactRepository.Save").Str("artifact_id", artifact.ID()).Logger()
	log.Debug().Msg("saving artifact")

	query := `
		INSERT INTO artifacts (id, suite_id, platform, s3_binary_key, sha256_checksum, build_logs_s3_key, status, error_message, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (id) DO UPDATE SET
			suite_id = EXCLUDED.suite_id,
			platform = EXCLUDED.platform,
			s3_binary_key = EXCLUDED.s3_binary_key,
			sha256_checksum = EXCLUDED.sha256_checksum,
			build_logs_s3_key = EXCLUDED.build_logs_s3_key,
			status = EXCLUDED.status,
			error_message = EXCLUDED.error_message
	`
	_, err := r.pool.Exec(ctx, query,
		artifact.ID(),
		artifact.SuiteID(),
		string(artifact.Platform()),
		artifact.S3BinaryKey(),
		artifact.SHA256Checksum(),
		artifact.BuildLogsS3Key(),
		string(artifact.Status()),
		artifact.ErrorMessage(),
		artifact.CreatedAt(),
	)
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed to save artifact")
		return MapError(err)
	}

	log.Info().Dur("duration_ms", time.Since(start)).Msg("successfully saved artifact")
	return nil
}

// FindByID retrieves an Artifact entity by ID.
func (r *ArtifactRepository) FindByID(ctx context.Context, id string) (*model.Artifact, error) {
	start := time.Now()
	log := zerolog.Ctx(ctx).With().Str("op", "ArtifactRepository.FindByID").Str("artifact_id", id).Logger()
	log.Debug().Msg("finding artifact by id")

	query := `
		SELECT id, suite_id, platform, s3_binary_key, sha256_checksum, build_logs_s3_key, status, error_message, created_at
		FROM artifacts
		WHERE id = $1
	`
	var (
		artifactID     string
		suiteID        string
		platform       string
		s3BinaryKey    string
		sha256Checksum string
		buildLogsS3Key string
		status         string
		errorMessage   string
		createdAt      time.Time
	)
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&artifactID, &suiteID, &platform, &s3BinaryKey, &sha256Checksum,
		&buildLogsS3Key, &status, &errorMessage, &createdAt,
	)
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed to find artifact by id")
		return nil, MapError(err)
	}

	artifact, err := model.NewArtifactWithID(
		artifactID, suiteID, model.Platform(platform),
		s3BinaryKey, sha256Checksum, buildLogsS3Key,
		model.ArtifactStatus(status), errorMessage, createdAt,
	)
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed to reconstitute artifact")
		return nil, err
	}

	log.Info().Dur("duration_ms", time.Since(start)).Msg("successfully found artifact by id")
	return artifact, nil
}

// ListBySuiteID returns all Artifact entities for a given test suite.
func (r *ArtifactRepository) ListBySuiteID(ctx context.Context, suiteID string) ([]*model.Artifact, error) {
	start := time.Now()
	log := zerolog.Ctx(ctx).With().Str("op", "ArtifactRepository.ListBySuiteID").Str("suite_id", suiteID).Logger()
	log.Debug().Msg("listing artifacts by suite id")

	query := `
		SELECT id, suite_id, platform, s3_binary_key, sha256_checksum, build_logs_s3_key, status, error_message, created_at
		FROM artifacts
		WHERE suite_id = $1
		ORDER BY created_at ASC
	`
	rows, err := r.pool.Query(ctx, query, suiteID)
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed to query artifacts by suite id")
		return nil, MapError(err)
	}
	defer rows.Close()

	var artifacts []*model.Artifact
	for rows.Next() {
		var (
			artifactID     string
			sID            string
			platform       string
			s3BinaryKey    string
			sha256Checksum string
			buildLogsS3Key string
			status         string
			errorMessage   string
			createdAt      time.Time
		)
		if err := rows.Scan(
			&artifactID, &sID, &platform, &s3BinaryKey, &sha256Checksum,
			&buildLogsS3Key, &status, &errorMessage, &createdAt,
		); err != nil {
			log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed to scan artifact row")
			return nil, MapError(err)
		}
		artifact, err := model.NewArtifactWithID(
			artifactID, sID, model.Platform(platform),
			s3BinaryKey, sha256Checksum, buildLogsS3Key,
			model.ArtifactStatus(status), errorMessage, createdAt,
		)
		if err != nil {
			log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed to reconstitute artifact from row")
			return nil, err
		}
		artifacts = append(artifacts, artifact)
	}

	if err := rows.Err(); err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("error iterating artifact rows")
		return nil, MapError(err)
	}

	log.Info().Int("count", len(artifacts)).Dur("duration_ms", time.Since(start)).Msg("successfully listed artifacts by suite id")
	return artifacts, nil
}

// Delete removes an Artifact by ID.
func (r *ArtifactRepository) Delete(ctx context.Context, id string) error {
	start := time.Now()
	log := zerolog.Ctx(ctx).With().Str("op", "ArtifactRepository.Delete").Str("artifact_id", id).Logger()
	log.Debug().Msg("deleting artifact")

	cmdTag, err := r.pool.Exec(ctx, `DELETE FROM artifacts WHERE id = $1`, id)
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed to delete artifact")
		return MapError(err)
	}

	if cmdTag.RowsAffected() == 0 {
		log.Error().Dur("duration_ms", time.Since(start)).Msg("artifact not found for deletion")
		return model.ErrNotFound
	}

	log.Info().Dur("duration_ms", time.Since(start)).Msg("successfully deleted artifact")
	return nil
}

// Static compile-time interface assertion
var _ outbound.ArtifactRepository = (*ArtifactRepository)(nil)
