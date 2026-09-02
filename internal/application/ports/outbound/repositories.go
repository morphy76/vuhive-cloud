package outbound

import (
	"context"

	"github.com/morphy76/vuhive-cloud/internal/domain/model"
)

// TestSuiteRepository defines the driven persistence port for TestSuite aggregates.
type TestSuiteRepository interface {
	Save(ctx context.Context, suite *model.TestSuite) error
	FindByID(ctx context.Context, id string) (*model.TestSuite, error)
	FindByName(ctx context.Context, name string) (*model.TestSuite, error)
	List(ctx context.Context) ([]*model.TestSuite, error)
	Delete(ctx context.Context, id string) error
}

// ArtifactRepository defines the driven persistence port for Artifact entities.
type ArtifactRepository interface {
	Save(ctx context.Context, artifact *model.Artifact) error
	FindByID(ctx context.Context, id string) (*model.Artifact, error)
	ListBySuiteID(ctx context.Context, suiteID string) ([]*model.Artifact, error)
	Delete(ctx context.Context, id string) error
}

// ConfigurationRepository defines the driven persistence port for Configuration entities.
type ConfigurationRepository interface {
	Save(ctx context.Context, config *model.Configuration) error
	FindByID(ctx context.Context, id string) (*model.Configuration, error)
	ListBySuiteID(ctx context.Context, suiteID string) ([]*model.Configuration, error)
	Delete(ctx context.Context, id string) error
}

// RunnerProfileRepository defines the driven persistence port for RunnerProfile entities.
type RunnerProfileRepository interface {
	Save(ctx context.Context, profile *model.RunnerProfile) error
	FindByID(ctx context.Context, id string) (*model.RunnerProfile, error)
	FindByName(ctx context.Context, name string) (*model.RunnerProfile, error)
	List(ctx context.Context) ([]*model.RunnerProfile, error)
	Delete(ctx context.Context, id string) error
}

// TestRunRepository defines the driven persistence port for TestRun aggregates.
type TestRunRepository interface {
	Save(ctx context.Context, run *model.TestRun) error
	FindByID(ctx context.Context, id string) (*model.TestRun, error)
	List(ctx context.Context, suiteID string, status model.RunStatus) ([]*model.TestRun, error)
	Delete(ctx context.Context, id string) error
}

// ScheduleRepository defines the driven persistence port for Schedule aggregates.
type ScheduleRepository interface {
	Save(ctx context.Context, schedule *model.Schedule) error
	FindByID(ctx context.Context, id string) (*model.Schedule, error)
	ListBySuiteID(ctx context.Context, suiteID string) ([]*model.Schedule, error)
	ListActive(ctx context.Context) ([]*model.Schedule, error)
	Delete(ctx context.Context, id string) error
}
