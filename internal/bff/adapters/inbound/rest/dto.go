package rest

import (
	"time"

	"github.com/morphy76/vuhive-cloud/internal/bff/application/ports/inbound"
	"github.com/morphy76/vuhive-cloud/internal/bff/domain/model"
)

// HealthResponse represents the response payload for /healthz.
type HealthResponse struct {
	Status string `json:"status"`
}

// VersionResponse represents the response payload for /version.
type VersionResponse struct {
	Version   string `json:"version"`
	Commit    string `json:"commit,omitempty"`
	BuildTime string `json:"build_time,omitempty"`
}

// StatusResponse represents the aggregated BFF and control plane status payload.
type StatusResponse struct {
	BFFStatus           string                 `json:"bff_status"`
	BFFVersion          string                 `json:"bff_version"`
	ControlPlaneStatus  string                 `json:"control_plane_status"`
	ControlPlaneVersion string                 `json:"control_plane_version,omitempty"`
	Timestamp           time.Time              `json:"timestamp"`
	Metadata            map[string]interface{} `json:"metadata,omitempty"`
}

// CreateSessionRequest encapsulates the request body for creating a BFF session.
type CreateSessionRequest struct {
	SessionID  string            `json:"session_id" binding:"required"`
	UserID     string            `json:"user_id" binding:"required"`
	TTLSeconds int64             `json:"ttl_seconds"`
	Metadata   map[string]string `json:"metadata,omitempty"`
}

// SessionResponse represents the serialized client session payload.
type SessionResponse struct {
	ID        string            `json:"id"`
	UserID    string            `json:"user_id"`
	CreatedAt time.Time         `json:"created_at"`
	ExpiresAt time.Time         `json:"expires_at"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

// ErrorResponse represents a standardized JSON error message.
type ErrorResponse struct {
	Error string `json:"error"`
}

// ToStatusResponse maps application SystemStatus to inbound REST DTO.
func ToStatusResponse(s *inbound.SystemStatus) StatusResponse {
	return StatusResponse{
		BFFStatus:           s.BFFStatus,
		BFFVersion:          s.BFFVersion,
		ControlPlaneStatus:  s.ControlPlaneStatus,
		ControlPlaneVersion: s.ControlPlaneVersion,
		Timestamp:           s.Timestamp,
		Metadata:            s.Metadata,
	}
}

// ToSessionResponse maps domain ClientSession to inbound REST DTO.
func ToSessionResponse(s *model.ClientSession) SessionResponse {
	return SessionResponse{
		ID:        string(s.ID),
		UserID:    s.UserID,
		CreatedAt: s.CreatedAt,
		ExpiresAt: s.ExpiresAt,
		Metadata:  s.Metadata,
	}
}

// SuiteSummaryDTO models test suite overview in REST responses.
type SuiteSummaryDTO struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	State       string `json:"state"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

// ProfileSummaryDTO models runner profile overview in REST responses.
type ProfileSummaryDTO struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	RunnerImage string `json:"runner_image"`
	CPURequest  string `json:"cpu_request"`
	CPULimit    string `json:"cpu_limit"`
	MemoryLimit string `json:"memory_limit"`
	CreatedAt   string `json:"created_at"`
}

// DashboardResponse models the composite dashboard overview payload.
type DashboardResponse struct {
	BFFStatus           string              `json:"bff_status"`
	BFFVersion          string              `json:"bff_version"`
	ControlPlaneStatus  string              `json:"control_plane_status"`
	ControlPlaneVersion string              `json:"control_plane_version,omitempty"`
	ActiveRunsCount     int64               `json:"active_runs_count"`
	RecentSuites        []SuiteSummaryDTO   `json:"recent_suites"`
	ProfilesCount       int                 `json:"profiles_count"`
	ProfilesSummary     []ProfileSummaryDTO `json:"profiles_summary"`
	Timestamp           time.Time           `json:"timestamp"`
}

// ArtifactLinksDTO models direct download links to test run artifacts.
type ArtifactLinksDTO struct {
	ReportURL string `json:"report_url,omitempty"`
	LogsURL   string `json:"logs_url,omitempty"`
}

// RunMetricsDTO models performance metrics returned in unified run details.
type RunMetricsDTO struct {
	TotalIterations int64   `json:"total_iterations"`
	TotalRequests   int64   `json:"total_requests"`
	AvgTPS          float64 `json:"avg_tps"`
	P50DurationMs   float64 `json:"p50_duration_ms"`
	P90DurationMs   float64 `json:"p90_duration_ms"`
	P95DurationMs   float64 `json:"p95_duration_ms"`
	P99DurationMs   float64 `json:"p99_duration_ms"`
	ErrorRatePct    float64 `json:"error_rate_pct"`
}

// RunDetailResponse models the unified composite run detail response.
type RunDetailResponse struct {
	ID              string           `json:"id"`
	SuiteID         string           `json:"suite_id"`
	ArtifactID      string           `json:"artifact_id"`
	ConfigurationID *string          `json:"configuration_id,omitempty"`
	RunnerProfileID string           `json:"runner_profile_id"`
	ScheduleID      *string          `json:"schedule_id,omitempty"`
	Status          string           `json:"status"`
	K8sJobName      string           `json:"k8s_job_name,omitempty"`
	K8sNamespace    string           `json:"k8s_namespace,omitempty"`
	StartedAt       *string          `json:"started_at,omitempty"`
	FinishedAt      *string          `json:"finished_at,omitempty"`
	DurationMs      *int64           `json:"duration_ms,omitempty"`
	ExitCode        *int             `json:"exit_code,omitempty"`
	SLAPassed       *bool            `json:"sla_passed,omitempty"`
	Metrics         RunMetricsDTO    `json:"metrics"`
	S3ReportKey     string           `json:"s3_report_key,omitempty"`
	S3LogsKey       string           `json:"s3_logs_key,omitempty"`
	AbortReason     string           `json:"abort_reason,omitempty"`
	CreatedAt       string           `json:"created_at"`
	ArtifactLinks   ArtifactLinksDTO `json:"artifact_links"`
}

// ToDashboardResponse maps inbound.DashboardOverview to DashboardResponse DTO.
func ToDashboardResponse(d *inbound.DashboardOverview) DashboardResponse {
	suites := make([]SuiteSummaryDTO, 0, len(d.RecentSuites))
	for _, s := range d.RecentSuites {
		suites = append(suites, SuiteSummaryDTO{
			ID:          s.ID,
			Name:        s.Name,
			Description: s.Description,
			State:       s.State,
			CreatedAt:   s.CreatedAt,
			UpdatedAt:   s.UpdatedAt,
		})
	}

	profiles := make([]ProfileSummaryDTO, 0, len(d.ProfilesSummary))
	for _, p := range d.ProfilesSummary {
		profiles = append(profiles, ProfileSummaryDTO{
			ID:          p.ID,
			Name:        p.Name,
			Description: p.Description,
			RunnerImage: p.RunnerImage,
			CPURequest:  p.CPURequest,
			CPULimit:    p.CPULimit,
			MemoryLimit: p.MemoryLimit,
			CreatedAt:   p.CreatedAt,
		})
	}

	return DashboardResponse{
		BFFStatus:           d.BFFStatus,
		BFFVersion:          d.BFFVersion,
		ControlPlaneStatus:  d.ControlPlaneStatus,
		ControlPlaneVersion: d.ControlPlaneVersion,
		ActiveRunsCount:     d.ActiveRunsCount,
		RecentSuites:        suites,
		ProfilesCount:       d.ProfilesCount,
		ProfilesSummary:     profiles,
		Timestamp:           d.Timestamp,
	}
}

// ToRunDetailResponse maps inbound.RunDetailComposite to RunDetailResponse DTO.
func ToRunDetailResponse(r *inbound.RunDetailComposite) RunDetailResponse {
	return RunDetailResponse{
		ID:              r.ID,
		SuiteID:         r.SuiteID,
		ArtifactID:      r.ArtifactID,
		ConfigurationID: r.ConfigurationID,
		RunnerProfileID: r.RunnerProfileID,
		ScheduleID:      r.ScheduleID,
		Status:          r.Status,
		K8sJobName:      r.K8sJobName,
		K8sNamespace:    r.K8sNamespace,
		StartedAt:       r.StartedAt,
		FinishedAt:      r.FinishedAt,
		DurationMs:      r.DurationMs,
		ExitCode:        r.ExitCode,
		SLAPassed:       r.SLAPassed,
		Metrics: RunMetricsDTO{
			TotalIterations: r.Metrics.TotalIterations,
			TotalRequests:   r.Metrics.TotalRequests,
			AvgTPS:          r.Metrics.AvgTPS,
			P50DurationMs:   r.Metrics.P50DurationMs,
			P90DurationMs:   r.Metrics.P90DurationMs,
			P95DurationMs:   r.Metrics.P95DurationMs,
			P99DurationMs:   r.Metrics.P99DurationMs,
			ErrorRatePct:    r.Metrics.ErrorRatePct,
		},
		S3ReportKey: r.S3ReportKey,
		S3LogsKey:   r.S3LogsKey,
		AbortReason: r.AbortReason,
		CreatedAt:   r.CreatedAt,
		ArtifactLinks: ArtifactLinksDTO{
			ReportURL: r.ArtifactLinks.ReportURL,
			LogsURL:   r.ArtifactLinks.LogsURL,
		},
	}
}

