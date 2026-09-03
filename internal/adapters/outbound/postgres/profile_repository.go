package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/morphy76/vuhive-cloud/internal/application/ports/outbound"
	"github.com/morphy76/vuhive-cloud/internal/domain/model"
	"github.com/rs/zerolog"
)

// RunnerProfileRepository implements outbound.RunnerProfileRepository using pgxpool.
type RunnerProfileRepository struct {
	pool *pgxpool.Pool
}

// NewRunnerProfileRepository constructs a new RunnerProfileRepository.
func NewRunnerProfileRepository(pool *pgxpool.Pool) *RunnerProfileRepository {
	return &RunnerProfileRepository{pool: pool}
}

// Save inserts or updates a RunnerProfile entity.
func (r *RunnerProfileRepository) Save(ctx context.Context, profile *model.RunnerProfile) error {
	start := time.Now()
	log := zerolog.Ctx(ctx).With().Str("op", "RunnerProfileRepository.Save").Str("profile_id", profile.ID()).Logger()
	log.Debug().Msg("saving runner profile")

	nodeSelectorJSON, err := json.Marshal(profile.NodeSelector())
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed to marshal nodeSelector")
		return fmt.Errorf("failed to marshal nodeSelector: %w", err)
	}

	affinityJSON, err := json.Marshal(profile.Affinity())
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed to marshal affinity")
		return fmt.Errorf("failed to marshal affinity: %w", err)
	}

	tolerationsJSON, err := json.Marshal(profile.Tolerations())
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed to marshal tolerations")
		return fmt.Errorf("failed to marshal tolerations: %w", err)
	}

	query := `
		INSERT INTO runner_profiles (
			id, name, description, runner_image,
			cpu_request, cpu_limit, memory_request, memory_limit,
			node_selector, affinity, tolerations,
			created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		ON CONFLICT (id) DO UPDATE SET
			name = EXCLUDED.name,
			description = EXCLUDED.description,
			runner_image = EXCLUDED.runner_image,
			cpu_request = EXCLUDED.cpu_request,
			cpu_limit = EXCLUDED.cpu_limit,
			memory_request = EXCLUDED.memory_request,
			memory_limit = EXCLUDED.memory_limit,
			node_selector = EXCLUDED.node_selector,
			affinity = EXCLUDED.affinity,
			tolerations = EXCLUDED.tolerations,
			updated_at = EXCLUDED.updated_at
	`
	_, err = r.pool.Exec(ctx, query,
		profile.ID(),
		profile.Name(),
		profile.Description(),
		profile.RunnerImage(),
		profile.Resources().CPURequest(),
		profile.Resources().CPULimit(),
		profile.Resources().MemoryRequest(),
		profile.Resources().MemoryLimit(),
		nodeSelectorJSON,
		affinityJSON,
		tolerationsJSON,
		profile.CreatedAt(),
		profile.UpdatedAt(),
	)
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed to save runner profile")
		return MapError(err)
	}

	log.Info().Dur("duration_ms", time.Since(start)).Msg("successfully saved runner profile")
	return nil
}

// FindByID retrieves a RunnerProfile entity by ID.
func (r *RunnerProfileRepository) FindByID(ctx context.Context, id string) (*model.RunnerProfile, error) {
	start := time.Now()
	log := zerolog.Ctx(ctx).With().Str("op", "RunnerProfileRepository.FindByID").Str("profile_id", id).Logger()
	log.Debug().Msg("finding runner profile by id")

	query := `
		SELECT id, name, description, runner_image,
		       cpu_request, cpu_limit, memory_request, memory_limit,
		       node_selector, affinity, tolerations,
		       created_at, updated_at
		FROM runner_profiles
		WHERE id = $1
	`
	var (
		profileID        string
		name             string
		description      string
		runnerImage      string
		cpuRequest       string
		cpuLimit         string
		memoryRequest    string
		memoryLimit      string
		nodeSelectorJSON []byte
		affinityJSON     []byte
		tolerationsJSON  []byte
		createdAt        time.Time
		updatedAt        time.Time
	)
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&profileID, &name, &description, &runnerImage,
		&cpuRequest, &cpuLimit, &memoryRequest, &memoryLimit,
		&nodeSelectorJSON, &affinityJSON, &tolerationsJSON,
		&createdAt, &updatedAt,
	)
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed to find runner profile by id")
		return nil, MapError(err)
	}

	profile, err := unmarshalProfile(
		profileID, name, description, runnerImage,
		cpuRequest, cpuLimit, memoryRequest, memoryLimit,
		nodeSelectorJSON, affinityJSON, tolerationsJSON,
		createdAt, updatedAt,
	)
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed to reconstitute runner profile")
		return nil, err
	}

	log.Info().Dur("duration_ms", time.Since(start)).Msg("successfully found runner profile by id")
	return profile, nil
}

// FindByName retrieves a RunnerProfile entity by its unique name.
func (r *RunnerProfileRepository) FindByName(ctx context.Context, name string) (*model.RunnerProfile, error) {
	start := time.Now()
	log := zerolog.Ctx(ctx).With().Str("op", "RunnerProfileRepository.FindByName").Str("profile_name", name).Logger()
	log.Debug().Msg("finding runner profile by name")

	query := `
		SELECT id, name, description, runner_image,
		       cpu_request, cpu_limit, memory_request, memory_limit,
		       node_selector, affinity, tolerations,
		       created_at, updated_at
		FROM runner_profiles
		WHERE name = $1
	`
	var (
		profileID        string
		profileName      string
		description      string
		runnerImage      string
		cpuRequest       string
		cpuLimit         string
		memoryRequest    string
		memoryLimit      string
		nodeSelectorJSON []byte
		affinityJSON     []byte
		tolerationsJSON  []byte
		createdAt        time.Time
		updatedAt        time.Time
	)
	err := r.pool.QueryRow(ctx, query, name).Scan(
		&profileID, &profileName, &description, &runnerImage,
		&cpuRequest, &cpuLimit, &memoryRequest, &memoryLimit,
		&nodeSelectorJSON, &affinityJSON, &tolerationsJSON,
		&createdAt, &updatedAt,
	)
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed to find runner profile by name")
		return nil, MapError(err)
	}

	profile, err := unmarshalProfile(
		profileID, profileName, description, runnerImage,
		cpuRequest, cpuLimit, memoryRequest, memoryLimit,
		nodeSelectorJSON, affinityJSON, tolerationsJSON,
		createdAt, updatedAt,
	)
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed to reconstitute runner profile")
		return nil, err
	}

	log.Info().Dur("duration_ms", time.Since(start)).Msg("successfully found runner profile by name")
	return profile, nil
}

// List returns all RunnerProfile entities ordered by created_at.
func (r *RunnerProfileRepository) List(ctx context.Context) ([]*model.RunnerProfile, error) {
	start := time.Now()
	log := zerolog.Ctx(ctx).With().Str("op", "RunnerProfileRepository.List").Logger()
	log.Debug().Msg("listing runner profiles")

	query := `
		SELECT id, name, description, runner_image,
		       cpu_request, cpu_limit, memory_request, memory_limit,
		       node_selector, affinity, tolerations,
		       created_at, updated_at
		FROM runner_profiles
		ORDER BY created_at ASC
	`
	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed to query runner profiles")
		return nil, MapError(err)
	}
	defer rows.Close()

	var profiles []*model.RunnerProfile
	for rows.Next() {
		var (
			profileID        string
			name             string
			description      string
			runnerImage      string
			cpuRequest       string
			cpuLimit         string
			memoryRequest    string
			memoryLimit      string
			nodeSelectorJSON []byte
			affinityJSON     []byte
			tolerationsJSON  []byte
			createdAt        time.Time
			updatedAt        time.Time
		)
		if err := rows.Scan(
			&profileID, &name, &description, &runnerImage,
			&cpuRequest, &cpuLimit, &memoryRequest, &memoryLimit,
			&nodeSelectorJSON, &affinityJSON, &tolerationsJSON,
			&createdAt, &updatedAt,
		); err != nil {
			log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed to scan runner profile row")
			return nil, MapError(err)
		}

		profile, err := unmarshalProfile(
			profileID, name, description, runnerImage,
			cpuRequest, cpuLimit, memoryRequest, memoryLimit,
			nodeSelectorJSON, affinityJSON, tolerationsJSON,
			createdAt, updatedAt,
		)
		if err != nil {
			log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed to reconstitute runner profile from row")
			return nil, err
		}
		profiles = append(profiles, profile)
	}

	if err := rows.Err(); err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("error iterating runner profile rows")
		return nil, MapError(err)
	}

	log.Info().Int("count", len(profiles)).Dur("duration_ms", time.Since(start)).Msg("successfully listed runner profiles")
	return profiles, nil
}

// Delete removes a RunnerProfile by ID.
func (r *RunnerProfileRepository) Delete(ctx context.Context, id string) error {
	start := time.Now()
	log := zerolog.Ctx(ctx).With().Str("op", "RunnerProfileRepository.Delete").Str("profile_id", id).Logger()
	log.Debug().Msg("deleting runner profile")

	cmdTag, err := r.pool.Exec(ctx, `DELETE FROM runner_profiles WHERE id = $1`, id)
	if err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed to delete runner profile")
		return MapError(err)
	}

	if cmdTag.RowsAffected() == 0 {
		log.Error().Dur("duration_ms", time.Since(start)).Msg("runner profile not found for deletion")
		return model.ErrNotFound
	}

	log.Info().Dur("duration_ms", time.Since(start)).Msg("successfully deleted runner profile")
	return nil
}

func unmarshalProfile(
	profileID, name, description, runnerImage string,
	cpuRequest, cpuLimit, memoryRequest, memoryLimit string,
	nodeSelectorJSON, affinityJSON, tolerationsJSON []byte,
	createdAt, updatedAt time.Time,
) (*model.RunnerProfile, error) {
	resources, err := model.NewResourceRequirements(cpuRequest, cpuLimit, memoryRequest, memoryLimit)
	if err != nil {
		return nil, err
	}

	var nodeSelector map[string]string
	if len(nodeSelectorJSON) > 0 {
		if err := json.Unmarshal(nodeSelectorJSON, &nodeSelector); err != nil {
			return nil, fmt.Errorf("failed to unmarshal nodeSelector: %w", err)
		}
	}

	var affinity model.Affinity
	if len(affinityJSON) > 0 {
		if err := json.Unmarshal(affinityJSON, &affinity); err != nil {
			return nil, fmt.Errorf("failed to unmarshal affinity: %w", err)
		}
	}

	var tolerations []model.Toleration
	if len(tolerationsJSON) > 0 {
		if err := json.Unmarshal(tolerationsJSON, &tolerations); err != nil {
			return nil, fmt.Errorf("failed to unmarshal tolerations: %w", err)
		}
	}

	return model.NewRunnerProfileWithID(
		profileID, name, description, runnerImage,
		resources, nodeSelector, affinity, tolerations,
		createdAt, updatedAt,
	)
}

// Static compile-time interface assertion
var _ outbound.RunnerProfileRepository = (*RunnerProfileRepository)(nil)
