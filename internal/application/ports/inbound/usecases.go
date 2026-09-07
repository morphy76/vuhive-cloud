package inbound

import (
	"context"
	"io"
	"time"

	"github.com/morphy76/vuhive-cloud/internal/domain/model"
)

// UpdateSuiteCommand encapsulates parameters for updating a TestSuite.
type UpdateSuiteCommand struct {
	Name        string
	Description string
	State       *model.TestSuiteState
}

// SuitesUseCase defines driving use cases for managing TestSuite aggregates.
type SuitesUseCase interface {
	CreateSuite(ctx context.Context, name, description string) (*model.TestSuite, error)
	GetSuite(ctx context.Context, id string) (*model.TestSuite, error)
	ListSuites(ctx context.Context) ([]*model.TestSuite, error)
	UpdateSuite(ctx context.Context, id string, cmd UpdateSuiteCommand) (*model.TestSuite, error)
	DeleteSuite(ctx context.Context, id string) error
	ArchiveSuite(ctx context.Context, id string) error
}

// CreateConfigCommand encapsulates input parameters for creating and uploading a test configuration.
type CreateConfigCommand struct {
	SuiteID     string
	Name        string
	ContentYAML string
	IsDefault   bool
}

// ConfigsUseCase defines driving use cases for managing test scenario configurations.
type ConfigsUseCase interface {
	CreateConfig(ctx context.Context, cmd CreateConfigCommand) (*model.Configuration, error)
	GetConfig(ctx context.Context, suiteID, configID string) (*model.Configuration, error)
	ListConfigs(ctx context.Context, suiteID string) ([]*model.Configuration, error)
	DeleteConfig(ctx context.Context, suiteID, configID string) error
}

// TriggerRunCommand encapsulates input parameters for triggering a new test run.
type TriggerRunCommand struct {
	SuiteID         string
	ArtifactID      string
	ConfigurationID *string
	RunnerProfileID string
	RunnerNamespace *string
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

// RunsUseCase defines driving use cases for triggering, tracking, querying, and aborting TestRun aggregates.
type RunsUseCase interface {
	TriggerRun(ctx context.Context, cmd TriggerRunCommand) (*model.TestRun, error)
	GetRun(ctx context.Context, id string) (*model.TestRun, error)
	ListRuns(ctx context.Context, filter model.RunFilter) ([]*model.TestRun, int64, error)
	AbortRun(ctx context.Context, id string, reason string) (*model.TestRun, error)
	CompleteRun(ctx context.Context, cmd CompleteRunCommand) (*model.TestRun, error)
	GetRunReport(ctx context.Context, id string) (io.ReadCloser, error)
	GetRunReportURL(ctx context.Context, id string, lifetime time.Duration) (string, error)
	GetRunLogs(ctx context.Context, id string) (io.ReadCloser, error)
	GetRunLogsURL(ctx context.Context, id string, lifetime time.Duration) (string, error)
}

// SchedulesUseCase defines driving use cases for managing recurring TestSchedule aggregates.
type SchedulesUseCase interface {
	CreateSchedule(ctx context.Context, suiteID, artifactID string, configID *string, runnerProfileID, name, cronExpr string) (*model.Schedule, error)
	GetSchedule(ctx context.Context, id string) (*model.Schedule, error)
	ListSchedules(ctx context.Context) ([]*model.Schedule, error)
	UpdateSchedule(ctx context.Context, id string, cronExpr string) (*model.Schedule, error)
	DeleteSchedule(ctx context.Context, id string) error
}

// BuildOptions provides optional configuration parameters for artifact compilation.
type BuildOptions struct {
	AllowInsecureImports bool
}

// BuildsUseCase defines driving use cases for compiling test suite sources into binary artifacts.
type BuildsUseCase interface {
	TriggerBuild(ctx context.Context, suiteID string, platform *model.Platform, source io.Reader, size int64) ([]*model.Artifact, error)
	TriggerBuildWithOptions(ctx context.Context, suiteID string, platform *model.Platform, source io.Reader, size int64, opts BuildOptions) ([]*model.Artifact, error)
	BuildArtifact(ctx context.Context, suiteID, artifactID string) (*model.Artifact, error)
	BuildSuite(ctx context.Context, suiteID string) ([]*model.Artifact, error)
	GetArtifact(ctx context.Context, id string) (*model.Artifact, error)
	ListArtifacts(ctx context.Context, suiteID string) ([]*model.Artifact, error)
}

// CreateProfileCommand encapsulates input parameters for creating a new runner profile.
type CreateProfileCommand struct {
	Name                  string
	Description           string
	RunnerImage           string
	CPURequest            string
	CPULimit              string
	MemoryRequest         string
	MemoryLimit           string
	NodeSelector          map[string]string
	Affinity              model.Affinity
	Tolerations           []model.Toleration
	ActiveDeadlineSeconds *int64
	RuntimeClassName      *string
}

// UpdateProfileCommand encapsulates input parameters for updating a runner profile.
type UpdateProfileCommand struct {
	Name                  string
	Description           string
	RunnerImage           string
	CPURequest            string
	CPULimit              string
	MemoryRequest         string
	MemoryLimit           string
	NodeSelector          map[string]string
	Affinity              model.Affinity
	Tolerations           []model.Toleration
	ActiveDeadlineSeconds *int64
	RuntimeClassName      *string
}

// ProfilesUseCase defines driving use cases for managing reusable RunnerProfile entities.
type ProfilesUseCase interface {
	CreateProfile(ctx context.Context, cmd CreateProfileCommand) (*model.RunnerProfile, error)
	GetProfile(ctx context.Context, id string) (*model.RunnerProfile, error)
	ListProfiles(ctx context.Context) ([]*model.RunnerProfile, error)
	UpdateProfile(ctx context.Context, id string, cmd UpdateProfileCommand) (*model.RunnerProfile, error)
	DeleteProfile(ctx context.Context, id string) error
}

// HousekeepingCommand encapsulates optional parameters for triggering a housekeeping run.
type HousekeepingCommand struct {
	SuiteID *string
	DryRun  bool
}

// HousekeepingResult reports the audit counts of entities processed during a housekeeping run.
type HousekeepingResult struct {
	PurgedLogsCount      int64    `json:"purged_logs_count"`
	PurgedReportsCount   int64    `json:"purged_reports_count"`
	PurgedSourcesCount   int64    `json:"purged_sources_count"`
	PurgedBinariesCount  int64    `json:"purged_binaries_count"`
	PrunedRunsCount      int64    `json:"pruned_runs_count"`
	ArchivedRunsCount    int64    `json:"archived_runs_count"`
	PurgedArtifactsCount int64    `json:"purged_artifacts_count"`
	DurationMs           int64    `json:"duration_ms"`
	Errors               []string `json:"errors,omitempty"`
}

// HousekeepingUseCase defines driving use cases for artifact retention, cleanup, and database pruning.
type HousekeepingUseCase interface {
	RunHousekeeping(ctx context.Context, cmd HousekeepingCommand) (*HousekeepingResult, error)
	GetGlobalPolicy(ctx context.Context) model.RetentionPolicy
	ConfigureBucketLifecycle(ctx context.Context) error
}
