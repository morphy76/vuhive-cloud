package postgres

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/morphy76/vuhive-cloud/internal/application/ports/outbound"
	"github.com/morphy76/vuhive-cloud/internal/domain/model"
	"github.com/rs/zerolog"
)

// TestSuiteRepository implements outbound.TestSuiteRepository using pgxpool.
type TestSuiteRepository struct {
	pool *pgxpool.Pool
}

// NewTestSuiteRepository constructs a new TestSuiteRepository.
func NewTestSuiteRepository(pool *pgxpool.Pool) *TestSuiteRepository {
	return &TestSuiteRepository{pool: pool}
}

// Save inserts or updates a TestSuite aggregate.
func (r *TestSuiteRepository) Save(ctx context.Context, suite *model.TestSuite) error {
	start := time.Now()
	log := zerolog.Ctx(ctx).With().Str("op", "TestSuiteRepository.Save").Str("suite_id", suite.ID()).Logger()
	log.Debug().Msg("saving test suite")

	query := `
		INSERT INTO test_suites (id, name, description, state, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (id) DO UPDATE SET
			name = EXCLUDED.name,
			description = EXCLUDED.description,
			state = EXCLUDED.state,
			updated_at = EXCLUDED.updated_at
	`
	_, err := r.pool.Exec(ctx, query,
		suite.ID(),
		suite.Name(),
		suite.Description(),
		string(suite.State()),
		suite.CreatedAt(),
		suite.UpdatedAt(),
	)
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed to save test suite")
		return MapError(err)
	}

	log.Info().Dur("duration_ms", time.Since(start)).Msg("successfully saved test suite")
	return nil
}

// FindByID retrieves a TestSuite aggregate by ID.
func (r *TestSuiteRepository) FindByID(ctx context.Context, id string) (*model.TestSuite, error) {
	start := time.Now()
	log := zerolog.Ctx(ctx).With().Str("op", "TestSuiteRepository.FindByID").Str("suite_id", id).Logger()
	log.Debug().Msg("finding test suite by id")

	query := `
		SELECT id, name, description, state, created_at, updated_at
		FROM test_suites
		WHERE id = $1
	`
	var (
		suiteID     string
		name        string
		description string
		state       string
		createdAt   time.Time
		updatedAt   time.Time
	)
	err := r.pool.QueryRow(ctx, query, id).Scan(&suiteID, &name, &description, &state, &createdAt, &updatedAt)
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed to find test suite by id")
		return nil, MapError(err)
	}

	suite, err := model.NewTestSuiteWithID(suiteID, name, description, model.TestSuiteState(state), createdAt, updatedAt)
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed to reconstitute test suite")
		return nil, err
	}

	log.Info().Dur("duration_ms", time.Since(start)).Msg("successfully found test suite by id")
	return suite, nil
}

// FindByName retrieves a TestSuite aggregate by its unique name.
func (r *TestSuiteRepository) FindByName(ctx context.Context, name string) (*model.TestSuite, error) {
	start := time.Now()
	log := zerolog.Ctx(ctx).With().Str("op", "TestSuiteRepository.FindByName").Str("suite_name", name).Logger()
	log.Debug().Msg("finding test suite by name")

	query := `
		SELECT id, name, description, state, created_at, updated_at
		FROM test_suites
		WHERE name = $1
	`
	var (
		suiteID     string
		suiteName   string
		description string
		state       string
		createdAt   time.Time
		updatedAt   time.Time
	)
	err := r.pool.QueryRow(ctx, query, name).Scan(&suiteID, &suiteName, &description, &state, &createdAt, &updatedAt)
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed to find test suite by name")
		return nil, MapError(err)
	}

	suite, err := model.NewTestSuiteWithID(suiteID, suiteName, description, model.TestSuiteState(state), createdAt, updatedAt)
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed to reconstitute test suite")
		return nil, err
	}

	log.Info().Dur("duration_ms", time.Since(start)).Msg("successfully found test suite by name")
	return suite, nil
}

// List returns all TestSuite aggregates ordered by created_at.
func (r *TestSuiteRepository) List(ctx context.Context) ([]*model.TestSuite, error) {
	start := time.Now()
	log := zerolog.Ctx(ctx).With().Str("op", "TestSuiteRepository.List").Logger()
	log.Debug().Msg("listing all test suites")

	query := `
		SELECT id, name, description, state, created_at, updated_at
		FROM test_suites
		ORDER BY created_at ASC
	`
	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed to query test suites")
		return nil, MapError(err)
	}
	defer rows.Close()

	var suites []*model.TestSuite
	for rows.Next() {
		var (
			suiteID     string
			name        string
			description string
			state       string
			createdAt   time.Time
			updatedAt   time.Time
		)
		if err := rows.Scan(&suiteID, &name, &description, &state, &createdAt, &updatedAt); err != nil {
			log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed to scan test suite row")
			return nil, MapError(err)
		}
		suite, err := model.NewTestSuiteWithID(suiteID, name, description, model.TestSuiteState(state), createdAt, updatedAt)
		if err != nil {
			log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed to reconstitute test suite from row")
			return nil, err
		}
		suites = append(suites, suite)
	}

	if err := rows.Err(); err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("error iterating test suite rows")
		return nil, MapError(err)
	}

	log.Info().Int("count", len(suites)).Dur("duration_ms", time.Since(start)).Msg("successfully listed test suites")
	return suites, nil
}

// Delete removes a TestSuite by ID.
func (r *TestSuiteRepository) Delete(ctx context.Context, id string) error {
	start := time.Now()
	log := zerolog.Ctx(ctx).With().Str("op", "TestSuiteRepository.Delete").Str("suite_id", id).Logger()
	log.Debug().Msg("deleting test suite")

	cmdTag, err := r.pool.Exec(ctx, `DELETE FROM test_suites WHERE id = $1`, id)
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed to delete test suite")
		return MapError(err)
	}

	if cmdTag.RowsAffected() == 0 {
		log.Error().Dur("duration_ms", time.Since(start)).Msg("test suite not found for deletion")
		return model.ErrNotFound
	}

	log.Info().Dur("duration_ms", time.Since(start)).Msg("successfully deleted test suite")
	return nil
}

// Static compile-time interface assertion
var _ outbound.TestSuiteRepository = (*TestSuiteRepository)(nil)
