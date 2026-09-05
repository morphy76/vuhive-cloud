package rest

import (
	"encoding/json"
	"time"

	"github.com/morphy76/vuhive-cloud/internal/domain/model"
)

// ArtifactResponse represents the JSON payload for a compiled binary artifact.
type ArtifactResponse struct {
	ID             string `json:"id"`
	SuiteID        string `json:"suite_id"`
	Platform       string `json:"platform"`
	S3BinaryKey    string `json:"s3_binary_key,omitempty"`
	SHA256Checksum string `json:"sha256_checksum,omitempty"`
	BuildLogsS3Key string `json:"build_logs_s3_key,omitempty"`
	Status         string `json:"status"`
	ErrorMessage   string `json:"error_message,omitempty"`
	CreatedAt      string `json:"created_at"`
}

// ArtifactListResponse represents the response containing a list of artifacts.
type ArtifactListResponse struct {
	Artifacts []ArtifactResponse `json:"artifacts"`
	Count     int                `json:"count"`
}

// BuildTriggerResponse represents the response returned when an asynchronous build is initiated.
type BuildTriggerResponse struct {
	Message   string             `json:"message"`
	Artifacts []ArtifactResponse `json:"artifacts"`
}

// ErrorResponse represents a standardized API error response.
type ErrorResponse struct {
	Error string `json:"error"`
}

// ToArtifactResponse converts a domain model.Artifact into an ArtifactResponse DTO.
func ToArtifactResponse(a *model.Artifact) ArtifactResponse {
	return ArtifactResponse{
		ID:             a.ID(),
		SuiteID:        a.SuiteID(),
		Platform:       string(a.Platform()),
		S3BinaryKey:    a.S3BinaryKey(),
		SHA256Checksum: a.SHA256Checksum(),
		BuildLogsS3Key: a.BuildLogsS3Key(),
		Status:         string(a.Status()),
		ErrorMessage:   a.ErrorMessage(),
		CreatedAt:      a.CreatedAt().Format(time.RFC3339),
	}
}

// ToArtifactListResponse converts a slice of domain model.Artifact into an ArtifactListResponse DTO.
func ToArtifactListResponse(artifacts []*model.Artifact) ArtifactListResponse {
	items := make([]ArtifactResponse, 0, len(artifacts))
	for _, a := range artifacts {
		items = append(items, ToArtifactResponse(a))
	}
	return ArtifactListResponse{
		Artifacts: items,
		Count:     len(items),
	}
}

// TolerationDTO represents a Kubernetes toleration in HTTP requests and responses.
type TolerationDTO struct {
	Key               string `json:"key,omitempty"`
	Operator          string `json:"operator,omitempty"`
	Value             string `json:"value,omitempty"`
	Effect            string `json:"effect,omitempty"`
	TolerationSeconds *int64 `json:"toleration_seconds,omitempty"`
}

// NodeAffinityTermDTO represents a Kubernetes node affinity term in HTTP requests and responses.
type NodeAffinityTermDTO struct {
	Key      string   `json:"key"`
	Operator string   `json:"operator"`
	Values   []string `json:"values,omitempty"`
}

// AffinityDTO represents Kubernetes node affinity in HTTP requests and responses.
type AffinityDTO struct {
	NodeSelectorTerms []NodeAffinityTermDTO `json:"node_selector_terms,omitempty"`
}

// CreateProfileRequest represents the JSON request payload to create a new RunnerProfile.
type CreateProfileRequest struct {
	Name          string            `json:"name" binding:"required"`
	Description   string            `json:"description,omitempty"`
	RunnerImage   string            `json:"runner_image,omitempty"`
	CPURequest    string            `json:"cpu_request,omitempty"`
	CPULimit      string            `json:"cpu_limit,omitempty"`
	MemoryRequest string            `json:"memory_request,omitempty"`
	MemoryLimit   string            `json:"memory_limit,omitempty"`
	NodeSelector  map[string]string `json:"node_selector,omitempty"`
	Affinity      *AffinityDTO      `json:"affinity,omitempty"`
	Tolerations   []TolerationDTO   `json:"tolerations,omitempty"`
}

// UpdateProfileRequest represents the JSON request payload to update an existing RunnerProfile.
type UpdateProfileRequest struct {
	Name          string            `json:"name" binding:"required"`
	Description   string            `json:"description,omitempty"`
	RunnerImage   string            `json:"runner_image,omitempty"`
	CPURequest    string            `json:"cpu_request,omitempty"`
	CPULimit      string            `json:"cpu_limit,omitempty"`
	MemoryRequest string            `json:"memory_request,omitempty"`
	MemoryLimit   string            `json:"memory_limit,omitempty"`
	NodeSelector  map[string]string `json:"node_selector,omitempty"`
	Affinity      *AffinityDTO      `json:"affinity,omitempty"`
	Tolerations   []TolerationDTO   `json:"tolerations,omitempty"`
}

// ProfileResponse represents the JSON response payload for a RunnerProfile entity.
type ProfileResponse struct {
	ID            string            `json:"id"`
	Name          string            `json:"name"`
	Description   string            `json:"description"`
	RunnerImage   string            `json:"runner_image"`
	CPURequest    string            `json:"cpu_request"`
	CPULimit      string            `json:"cpu_limit"`
	MemoryRequest string            `json:"memory_request"`
	MemoryLimit   string            `json:"memory_limit"`
	NodeSelector  map[string]string `json:"node_selector"`
	Affinity      AffinityDTO       `json:"affinity"`
	Tolerations   []TolerationDTO   `json:"tolerations"`
	CreatedAt     string            `json:"created_at"`
	UpdatedAt     string            `json:"updated_at"`
}

// ProfileListResponse represents the JSON response payload containing a list of RunnerProfiles.
type ProfileListResponse struct {
	Profiles []ProfileResponse `json:"profiles"`
	Count    int               `json:"count"`
}

// FromAffinityDTO converts an AffinityDTO into a domain model.Affinity.
func FromAffinityDTO(dto *AffinityDTO) model.Affinity {
	if dto == nil {
		return model.Affinity{}
	}
	terms := make([]model.NodeAffinityTerm, 0, len(dto.NodeSelectorTerms))
	for _, t := range dto.NodeSelectorTerms {
		terms = append(terms, model.NodeAffinityTerm{
			Key:      t.Key,
			Operator: t.Operator,
			Values:   t.Values,
		})
	}
	return model.Affinity{NodeSelectorTerms: terms}
}

// ToAffinityDTO converts a domain model.Affinity into an AffinityDTO.
func ToAffinityDTO(affinity model.Affinity) AffinityDTO {
	terms := make([]NodeAffinityTermDTO, 0, len(affinity.NodeSelectorTerms))
	for _, t := range affinity.NodeSelectorTerms {
		terms = append(terms, NodeAffinityTermDTO{
			Key:      t.Key,
			Operator: t.Operator,
			Values:   t.Values,
		})
	}
	return AffinityDTO{NodeSelectorTerms: terms}
}

// FromTolerationsDTO converts a slice of TolerationDTO into domain model.Toleration slices.
func FromTolerationsDTO(dtos []TolerationDTO) []model.Toleration {
	if len(dtos) == 0 {
		return nil
	}
	tolerations := make([]model.Toleration, 0, len(dtos))
	for _, t := range dtos {
		tolerations = append(tolerations, model.Toleration{
			Key:               t.Key,
			Operator:          t.Operator,
			Value:             t.Value,
			Effect:            t.Effect,
			TolerationSeconds: t.TolerationSeconds,
		})
	}
	return tolerations
}

// ToTolerationsDTO converts a slice of domain model.Toleration into TolerationDTO slices.
func ToTolerationsDTO(tolerations []model.Toleration) []TolerationDTO {
	if len(tolerations) == 0 {
		return []TolerationDTO{}
	}
	dtos := make([]TolerationDTO, 0, len(tolerations))
	for _, t := range tolerations {
		dtos = append(dtos, TolerationDTO{
			Key:               t.Key,
			Operator:          t.Operator,
			Value:             t.Value,
			Effect:            t.Effect,
			TolerationSeconds: t.TolerationSeconds,
		})
	}
	return dtos
}

// ToProfileResponse converts a domain model.RunnerProfile into a ProfileResponse DTO.
func ToProfileResponse(p *model.RunnerProfile) ProfileResponse {
	nodeSelector := p.NodeSelector()
	if nodeSelector == nil {
		nodeSelector = make(map[string]string)
	}

	return ProfileResponse{
		ID:            p.ID(),
		Name:          p.Name(),
		Description:   p.Description(),
		RunnerImage:   p.RunnerImage(),
		CPURequest:    p.Resources().CPURequest(),
		CPULimit:      p.Resources().CPULimit(),
		MemoryRequest: p.Resources().MemoryRequest(),
		MemoryLimit:   p.Resources().MemoryLimit(),
		NodeSelector:  nodeSelector,
		Affinity:      ToAffinityDTO(p.Affinity()),
		Tolerations:   ToTolerationsDTO(p.Tolerations()),
		CreatedAt:     p.CreatedAt().Format(time.RFC3339),
		UpdatedAt:     p.UpdatedAt().Format(time.RFC3339),
	}
}

// ToProfileListResponse converts a slice of domain model.RunnerProfile into a ProfileListResponse DTO.
func ToProfileListResponse(profiles []*model.RunnerProfile) ProfileListResponse {
	items := make([]ProfileResponse, 0, len(profiles))
	for _, p := range profiles {
		items = append(items, ToProfileResponse(p))
	}
	return ProfileListResponse{
		Profiles: items,
		Count:    len(items),
	}
}

// CreateScheduleRequest defines the JSON payload for creating a recurring test schedule.
type CreateScheduleRequest struct {
	SuiteID         string  `json:"suite_id" binding:"required"`
	ArtifactID      string  `json:"artifact_id" binding:"required"`
	ConfigurationID *string `json:"configuration_id"`
	RunnerProfileID string  `json:"runner_profile_id" binding:"required"`
	Name            string  `json:"name" binding:"required"`
	CronExpression  string  `json:"cron_expression" binding:"required"`
}

// UpdateScheduleRequest defines the JSON payload for updating an existing schedule.
type UpdateScheduleRequest struct {
	CronExpression string `json:"cron_expression" binding:"required"`
}

// ScheduleResponse represents the JSON response for a Schedule aggregate.
type ScheduleResponse struct {
	ID              string  `json:"id"`
	SuiteID         string  `json:"suite_id"`
	ArtifactID      string  `json:"artifact_id"`
	ConfigurationID *string `json:"configuration_id,omitempty"`
	RunnerProfileID string  `json:"runner_profile_id"`
	Name            string  `json:"name"`
	CronExpression  string  `json:"cron_expression"`
	K8sCronJobName  string  `json:"k8s_cronjob_name"`
	IsActive        bool    `json:"is_active"`
	CreatedAt       string  `json:"created_at"`
	UpdatedAt       string  `json:"updated_at"`
}

// ScheduleListResponse represents the JSON response for a list of schedules.
type ScheduleListResponse struct {
	Schedules []ScheduleResponse `json:"schedules"`
	Count     int                `json:"count"`
}

// ToScheduleResponse maps a domain Schedule aggregate to a ScheduleResponse DTO.
func ToScheduleResponse(s *model.Schedule) ScheduleResponse {
	return ScheduleResponse{
		ID:              s.ID(),
		SuiteID:         s.SuiteID(),
		ArtifactID:      s.ArtifactID(),
		ConfigurationID: s.ConfigurationID(),
		RunnerProfileID: s.RunnerProfileID(),
		Name:            s.Name(),
		CronExpression:  s.CronExpression(),
		K8sCronJobName:  s.K8sCronJobName(),
		IsActive:        s.IsActive(),
		CreatedAt:       s.CreatedAt().Format(time.RFC3339),
		UpdatedAt:       s.UpdatedAt().Format(time.RFC3339),
	}
}

// ToScheduleListResponse maps a slice of domain Schedule aggregates to a ScheduleListResponse DTO.
func ToScheduleListResponse(schedules []*model.Schedule) ScheduleListResponse {
	items := make([]ScheduleResponse, 0, len(schedules))
	for _, s := range schedules {
		items = append(items, ToScheduleResponse(s))
	}
	return ScheduleListResponse{
		Schedules: items,
		Count:     len(items),
	}
}

// CompleteRunRequest encapsulates the request body for POST /api/v1/runs/{id}/complete.
type CompleteRunRequest struct {
	RunID       string                 `json:"run_id,omitempty"`
	ExitCode    *int                   `json:"exit_code,omitempty"`
	ReportKey   string                 `json:"report_key,omitempty"`
	LogsKey     string                 `json:"logs_key,omitempty"`
	FinishedAt  *string                `json:"finished_at,omitempty"`
	Summary     map[string]interface{} `json:"summary,omitempty"`
	SummaryJSON json.RawMessage        `json:"summary_json,omitempty"`
}

// RunMetricsDTO represents indexed performance KPIs in REST responses.
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

// RunResponse represents the JSON response for a TestRun aggregate.
type RunResponse struct {
	ID              string        `json:"id"`
	SuiteID         string        `json:"suite_id"`
	ArtifactID      string        `json:"artifact_id"`
	ConfigurationID *string       `json:"configuration_id,omitempty"`
	RunnerProfileID string        `json:"runner_profile_id"`
	ScheduleID      *string       `json:"schedule_id,omitempty"`
	Status          string        `json:"status"`
	K8sJobName      string        `json:"k8s_job_name,omitempty"`
	K8sNamespace    string        `json:"k8s_namespace,omitempty"`
	StartedAt       *string       `json:"started_at,omitempty"`
	FinishedAt      *string       `json:"finished_at,omitempty"`
	ExitCode        *int          `json:"exit_code,omitempty"`
	SLAPassed       *bool         `json:"sla_passed,omitempty"`
	Metrics         RunMetricsDTO `json:"metrics"`
	S3ReportKey     string        `json:"s3_report_key,omitempty"`
	S3LogsKey       string        `json:"s3_logs_key,omitempty"`
	AbortReason     string        `json:"abort_reason,omitempty"`
	CreatedAt       string        `json:"created_at"`
}

// ToRunMetricsDTO maps domain model.RunMetrics to a RunMetricsDTO.
func ToRunMetricsDTO(m model.RunMetrics) RunMetricsDTO {
	return RunMetricsDTO{
		TotalIterations: m.TotalIterations,
		TotalRequests:   m.TotalRequests,
		AvgTPS:          m.AvgTPS,
		P50DurationMs:   m.P50DurationMs,
		P90DurationMs:   m.P90DurationMs,
		P95DurationMs:   m.P95DurationMs,
		P99DurationMs:   m.P99DurationMs,
		ErrorRatePct:    m.ErrorRatePct,
	}
}

// ToRunResponse maps a domain TestRun aggregate to a RunResponse DTO.
func ToRunResponse(r *model.TestRun) RunResponse {
	var startedAtStr *string
	if r.StartedAt() != nil {
		s := r.StartedAt().Format(time.RFC3339)
		startedAtStr = &s
	}

	var finishedAtStr *string
	if r.FinishedAt() != nil {
		s := r.FinishedAt().Format(time.RFC3339)
		finishedAtStr = &s
	}

	return RunResponse{
		ID:              r.ID(),
		SuiteID:         r.SuiteID(),
		ArtifactID:      r.ArtifactID(),
		ConfigurationID: r.ConfigurationID(),
		RunnerProfileID: r.RunnerProfileID(),
		ScheduleID:      r.ScheduleID(),
		Status:          string(r.Status()),
		K8sJobName:      r.K8sJobName(),
		K8sNamespace:    r.K8sNamespace(),
		StartedAt:       startedAtStr,
		FinishedAt:      finishedAtStr,
		ExitCode:        r.ExitCode(),
		SLAPassed:       r.SLAPassed(),
		Metrics:         ToRunMetricsDTO(r.Metrics()),
		S3ReportKey:     r.S3ReportKey(),
		S3LogsKey:       r.S3LogsKey(),
		AbortReason:     r.AbortReason(),
		CreatedAt:       r.CreatedAt().Format(time.RFC3339),
	}
}

// BarrierAwaitRequest defines the JSON payload sent by worker pods to wait at the start barrier.
type BarrierAwaitRequest struct {
	WorkerID       string `json:"worker_id" binding:"required"`
	TotalWorkers   int    `json:"total_workers" binding:"required,min=1"`
	TimeoutMs      *int   `json:"timeout_ms"`
	ReleaseDelayMs *int   `json:"release_delay_ms"`
}

// BarrierAbortRequest defines the JSON payload sent by a worker to abort the start barrier.
type BarrierAbortRequest struct {
	WorkerID string `json:"worker_id" binding:"required"`
	Reason   string `json:"reason" binding:"required"`
}

// BarrierParticipantResponse represents an individual worker participant in the barrier response.
type BarrierParticipantResponse struct {
	WorkerID    string  `json:"worker_id"`
	Status      string  `json:"status"`
	JoinedAt    string  `json:"joined_at"`
	ReadyAt     *string `json:"ready_at,omitempty"`
	ErrorReason string  `json:"error_reason,omitempty"`
}

// BarrierResponse represents the JSON response for barrier rendezvous endpoints.
type BarrierResponse struct {
	RunID           string                       `json:"run_id"`
	WorkerID        string                       `json:"worker_id,omitempty"`
	Status          string                       `json:"status"`
	TotalWorkers    int                          `json:"total_workers"`
	ReadyWorkers    int                          `json:"ready_workers"`
	TargetStartTime *string                      `json:"target_start_time,omitempty"`
	StartInMs       int64                        `json:"start_in_ms,omitempty"`
	AbortReason     string                       `json:"abort_reason,omitempty"`
	Participants    []BarrierParticipantResponse `json:"participants,omitempty"`
}

// ToBarrierResponse converts a domain model.BarrierSession into a BarrierResponse DTO.
func ToBarrierResponse(s *model.BarrierSession) BarrierResponse {
	var targetTimeStr *string
	var startInMs int64
	if s.TargetStartTime() != nil {
		tStr := s.TargetStartTime().Format(time.RFC3339Nano)
		targetTimeStr = &tStr
		d := time.Until(*s.TargetStartTime())
		if d > 0 {
			startInMs = d.Milliseconds()
		}
	}

	participants := make([]BarrierParticipantResponse, 0, len(s.Participants()))
	for _, p := range s.Participants() {
		var readyAtStr *string
		if p.ReadyAt != nil {
			r := p.ReadyAt.Format(time.RFC3339Nano)
			readyAtStr = &r
		}
		participants = append(participants, BarrierParticipantResponse{
			WorkerID:    p.WorkerID,
			Status:      string(p.Status),
			JoinedAt:    p.JoinedAt.Format(time.RFC3339Nano),
			ReadyAt:     readyAtStr,
			ErrorReason: p.ErrorReason,
		})
	}

	return BarrierResponse{
		RunID:           s.RunID(),
		Status:          string(s.Status()),
		TotalWorkers:    s.TotalWorkers(),
		ReadyWorkers:    s.ReadyCount(),
		TargetStartTime: targetTimeStr,
		StartInMs:       startInMs,
		AbortReason:     s.AbortReason(),
		Participants:    participants,
	}
}

