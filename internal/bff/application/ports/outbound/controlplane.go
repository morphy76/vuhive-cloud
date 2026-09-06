package outbound

import (
	"context"
	"time"
)

// ControlPlaneHealth models health information reported by upstream cmd/server.
type ControlPlaneHealth struct {
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
}

// ControlPlaneVersion models version details returned by upstream cmd/server.
type ControlPlaneVersion struct {
	Version   string `json:"version"`
	Commit    string `json:"commit"`
	BuildTime string `json:"build_time"`
}

// ControlPlaneClient defines the driven outbound port for communicating with the control plane server.
type ControlPlaneClient interface {
	CheckHealth(ctx context.Context) (*ControlPlaneHealth, error)
	GetVersion(ctx context.Context) (*ControlPlaneVersion, error)
	GetActiveRunsCount(ctx context.Context) (int64, error)
	ListRecentSuites(ctx context.Context, limit int) ([]SuiteSummary, error)
	ListProfiles(ctx context.Context) ([]ProfileSummary, error)
	GetRun(ctx context.Context, id string) (*RunDetail, error)
	GetRunReportURL(ctx context.Context, id string) (string, error)
	GetRunLogsURL(ctx context.Context, id string) (string, error)
}

// RunMetrics represents indexed performance KPIs returned by the control plane.
type RunMetrics struct {
	TotalIterations int64   `json:"total_iterations"`
	TotalRequests   int64   `json:"total_requests"`
	AvgTPS          float64 `json:"avg_tps"`
	P50DurationMs   float64 `json:"p50_duration_ms"`
	P90DurationMs   float64 `json:"p90_duration_ms"`
	P95DurationMs   float64 `json:"p95_duration_ms"`
	P99DurationMs   float64 `json:"p99_duration_ms"`
	ErrorRatePct    float64 `json:"error_rate_pct"`
}

// RunDetail models test run execution metadata and metrics.
type RunDetail struct {
	ID              string     `json:"id"`
	SuiteID         string     `json:"suite_id"`
	ArtifactID      string     `json:"artifact_id"`
	ConfigurationID *string    `json:"configuration_id,omitempty"`
	RunnerProfileID string     `json:"runner_profile_id"`
	ScheduleID      *string    `json:"schedule_id,omitempty"`
	Status          string     `json:"status"`
	K8sJobName      string     `json:"k8s_job_name,omitempty"`
	K8sNamespace    string     `json:"k8s_namespace,omitempty"`
	StartedAt       *string    `json:"started_at,omitempty"`
	FinishedAt      *string    `json:"finished_at,omitempty"`
	DurationMs      *int64     `json:"duration_ms,omitempty"`
	ExitCode        *int       `json:"exit_code,omitempty"`
	SLAPassed       *bool      `json:"sla_passed,omitempty"`
	Metrics         RunMetrics `json:"metrics"`
	S3ReportKey     string     `json:"s3_report_key,omitempty"`
	S3LogsKey       string     `json:"s3_logs_key,omitempty"`
	AbortReason     string     `json:"abort_reason,omitempty"`
	CreatedAt       string     `json:"created_at"`
}

// SuiteSummary models a test suite entry.
type SuiteSummary struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	State       string `json:"state"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

// ProfileSummary models a runner profile entry.
type ProfileSummary struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	RunnerImage string `json:"runner_image"`
	CPURequest  string `json:"cpu_request"`
	CPULimit    string `json:"cpu_limit"`
	MemoryLimit string `json:"memory_limit"`
	CreatedAt   string `json:"created_at"`
}
