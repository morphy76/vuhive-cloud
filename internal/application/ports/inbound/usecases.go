package inbound

import (
	"context"
	"io"
	"time"

	"github.com/morphy76/vuhive-cloud/internal/domain/model"
)

// SuitesUseCase defines driving use cases for managing TestSuite aggregates.
type SuitesUseCase interface {
	CreateSuite(ctx context.Context, name, description string) (*model.TestSuite, error)
	GetSuite(ctx context.Context, id string) (*model.TestSuite, error)
	ListSuites(ctx context.Context) ([]*model.TestSuite, error)
	ArchiveSuite(ctx context.Context, id string) error
}

// TriggerRunCommand encapsulates input parameters for triggering a new test run.
type TriggerRunCommand struct {
	SuiteID         string
	ArtifactID      string
	ConfigurationID *string
	RunnerProfileID string
}

// CompleteRunCommand encapsulates input parameters for finalizing a completed test run.
type CompleteRunCommand struct {
	RunID       string
	ExitCode    *int
	ReportKey   string
	LogsKey     string
	FinishedAt  *time.Time
	SummaryJSON []byte
}

// RunsUseCase defines driving use cases for triggering, tracking, and aborting TestRun aggregates.
type RunsUseCase interface {
	TriggerRun(ctx context.Context, cmd TriggerRunCommand) (*model.TestRun, error)
	GetRun(ctx context.Context, id string) (*model.TestRun, error)
	ListRuns(ctx context.Context, suiteID string, status model.RunStatus) ([]*model.TestRun, error)
	AbortRun(ctx context.Context, id string, reason string) (*model.TestRun, error)
	CompleteRun(ctx context.Context, cmd CompleteRunCommand) (*model.TestRun, error)
}

// SchedulesUseCase defines driving use cases for managing recurring TestSchedule aggregates.
type SchedulesUseCase interface {
	CreateSchedule(ctx context.Context, suiteID, artifactID string, configID *string, runnerProfileID, name, cronExpr string) (*model.Schedule, error)
	GetSchedule(ctx context.Context, id string) (*model.Schedule, error)
	ListSchedules(ctx context.Context) ([]*model.Schedule, error)
	UpdateSchedule(ctx context.Context, id string, cronExpr string) (*model.Schedule, error)
	DeleteSchedule(ctx context.Context, id string) error
}

// BuildsUseCase defines driving use cases for compiling test suite sources into binary artifacts.
type BuildsUseCase interface {
	TriggerBuild(ctx context.Context, suiteID string, platform *model.Platform, source io.Reader, size int64) ([]*model.Artifact, error)
	BuildArtifact(ctx context.Context, suiteID, artifactID string) (*model.Artifact, error)
	BuildSuite(ctx context.Context, suiteID string) ([]*model.Artifact, error)
	GetArtifact(ctx context.Context, id string) (*model.Artifact, error)
	ListArtifacts(ctx context.Context, suiteID string) ([]*model.Artifact, error)
}

// CreateProfileCommand encapsulates input parameters for creating a new runner profile.
type CreateProfileCommand struct {
	Name          string
	Description   string
	RunnerImage   string
	CPURequest    string
	CPULimit      string
	MemoryRequest string
	MemoryLimit   string
	NodeSelector  map[string]string
	Affinity      model.Affinity
	Tolerations   []model.Toleration
}

// UpdateProfileCommand encapsulates input parameters for updating a runner profile.
type UpdateProfileCommand struct {
	Name          string
	Description   string
	RunnerImage   string
	CPURequest    string
	CPULimit      string
	MemoryRequest string
	MemoryLimit   string
	NodeSelector  map[string]string
	Affinity      model.Affinity
	Tolerations   []model.Toleration
}

// ProfilesUseCase defines driving use cases for managing reusable RunnerProfile entities.
type ProfilesUseCase interface {
	CreateProfile(ctx context.Context, cmd CreateProfileCommand) (*model.RunnerProfile, error)
	GetProfile(ctx context.Context, id string) (*model.RunnerProfile, error)
	ListProfiles(ctx context.Context) ([]*model.RunnerProfile, error)
	UpdateProfile(ctx context.Context, id string, cmd UpdateProfileCommand) (*model.RunnerProfile, error)
	DeleteProfile(ctx context.Context, id string) error
}
