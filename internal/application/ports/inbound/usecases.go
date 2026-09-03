package inbound

import (
	"context"

	"github.com/morphy76/vuhive-cloud/internal/domain/model"
)

// SuitesUseCase defines driving use cases for managing TestSuite aggregates.
type SuitesUseCase interface {
	CreateSuite(ctx context.Context, name, description string) (*model.TestSuite, error)
	GetSuite(ctx context.Context, id string) (*model.TestSuite, error)
	ListSuites(ctx context.Context) ([]*model.TestSuite, error)
	ArchiveSuite(ctx context.Context, id string) error
}

// RunsUseCase defines driving use cases for triggering, tracking, and aborting TestRun aggregates.
type RunsUseCase interface {
	TriggerRun(ctx context.Context, suiteID, artifactID string, configID *string, runnerProfileID string) (*model.TestRun, error)
	GetRun(ctx context.Context, id string) (*model.TestRun, error)
	ListRuns(ctx context.Context, suiteID string, status model.RunStatus) ([]*model.TestRun, error)
	AbortRun(ctx context.Context, id string, reason string) error
}

// SchedulesUseCase defines driving use cases for managing recurring TestSchedule aggregates.
type SchedulesUseCase interface {
	CreateSchedule(ctx context.Context, suiteID, artifactID string, configID *string, runnerProfileID, name, cronExpr string) (*model.Schedule, error)
	GetSchedule(ctx context.Context, id string) (*model.Schedule, error)
	ListSchedules(ctx context.Context) ([]*model.Schedule, error)
	DeleteSchedule(ctx context.Context, id string) error
}

// BuildsUseCase defines driving use cases for compiling test suite sources into binary artifacts.
type BuildsUseCase interface {
	BuildArtifact(ctx context.Context, suiteID, artifactID string) (*model.Artifact, error)
	BuildSuite(ctx context.Context, suiteID string) ([]*model.Artifact, error)
	GetArtifact(ctx context.Context, id string) (*model.Artifact, error)
	ListArtifacts(ctx context.Context, suiteID string) ([]*model.Artifact, error)
}

