package model

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

const (
	// EventRunStatusChanged represents state changes for an active or completed test run.
	EventRunStatusChanged = "run_status_changed"

	// EventBuildStatusChanged represents state changes for an artifact compilation job.
	EventBuildStatusChanged = "build_status_changed"

	// EventSystemHeartbeat represents regular keep-alive heartbeats carrying system telemetry.
	EventSystemHeartbeat = "system_heartbeat"
)

// ServerSentEvent models an individual SSE frame pushed to connected clients.
type ServerSentEvent struct {
	ID        string          `json:"id"`
	Event     string          `json:"event"`
	Data      json.RawMessage `json:"data"`
	Timestamp time.Time       `json:"timestamp"`
}

// RunMetrics represents indexed performance KPIs within domain event payloads.
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

// RunStatusChangedPayload encapsulates data for EventRunStatusChanged.
type RunStatusChangedPayload struct {
	RunID          string      `json:"run_id"`
	SuiteID        string      `json:"suite_id"`
	Status         string      `json:"status"`
	PreviousStatus string      `json:"previous_status,omitempty"`
	K8sJobName     string      `json:"k8s_job_name,omitempty"`
	StartedAt      *string     `json:"started_at,omitempty"`
	FinishedAt     *string     `json:"finished_at,omitempty"`
	DurationMs     *int64      `json:"duration_ms,omitempty"`
	ExitCode       *int        `json:"exit_code,omitempty"`
	SLAPassed      *bool       `json:"sla_passed,omitempty"`
	Metrics        *RunMetrics `json:"metrics,omitempty"`
	Timestamp      time.Time   `json:"timestamp"`
}

// BuildStatusChangedPayload encapsulates data for EventBuildStatusChanged.
type BuildStatusChangedPayload struct {
	ArtifactID     string    `json:"artifact_id"`
	SuiteID        string    `json:"suite_id"`
	Platform       string    `json:"platform"`
	Status         string    `json:"status"`
	PreviousStatus string    `json:"previous_status,omitempty"`
	S3BinaryKey    string    `json:"s3_binary_key,omitempty"`
	SHA256Checksum string    `json:"sha256_checksum,omitempty"`
	ErrorMessage   string    `json:"error_message,omitempty"`
	Timestamp      time.Time `json:"timestamp"`
}

// SystemHeartbeatPayload encapsulates data for EventSystemHeartbeat.
type SystemHeartbeatPayload struct {
	Status        string    `json:"status"`
	ActiveClients int       `json:"active_clients"`
	ActiveRuns    int64     `json:"active_runs"`
	Timestamp     time.Time `json:"timestamp"`
}

// NewServerSentEvent constructs and validates a generic ServerSentEvent.
func NewServerSentEvent(id, event string, data []byte, timestamp time.Time) (ServerSentEvent, error) {
	if strings.TrimSpace(id) == "" {
		return ServerSentEvent{}, fmt.Errorf("%w: event id cannot be empty", ErrInvalidParameter)
	}
	if strings.TrimSpace(event) == "" {
		return ServerSentEvent{}, fmt.Errorf("%w: event type cannot be empty", ErrInvalidParameter)
	}
	if len(data) == 0 {
		return ServerSentEvent{}, fmt.Errorf("%w: event data cannot be empty", ErrInvalidParameter)
	}
	if timestamp.IsZero() {
		timestamp = time.Now().UTC()
	}

	return ServerSentEvent{
		ID:        strings.TrimSpace(id),
		Event:     strings.TrimSpace(event),
		Data:      json.RawMessage(data),
		Timestamp: timestamp,
	}, nil
}

// NewRunStatusChangedEvent constructs an EventRunStatusChanged ServerSentEvent.
func NewRunStatusChangedEvent(id string, payload RunStatusChangedPayload) (ServerSentEvent, error) {
	if strings.TrimSpace(payload.RunID) == "" {
		return ServerSentEvent{}, fmt.Errorf("%w: run id cannot be empty", ErrInvalidParameter)
	}
	if strings.TrimSpace(payload.Status) == "" {
		return ServerSentEvent{}, fmt.Errorf("%w: run status cannot be empty", ErrInvalidParameter)
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return ServerSentEvent{}, fmt.Errorf("%w: failed marshaling run status payload: %v", ErrInternal, err)
	}

	return NewServerSentEvent(id, EventRunStatusChanged, data, payload.Timestamp)
}

// NewBuildStatusChangedEvent constructs an EventBuildStatusChanged ServerSentEvent.
func NewBuildStatusChangedEvent(id string, payload BuildStatusChangedPayload) (ServerSentEvent, error) {
	if strings.TrimSpace(payload.ArtifactID) == "" {
		return ServerSentEvent{}, fmt.Errorf("%w: artifact id cannot be empty", ErrInvalidParameter)
	}
	if strings.TrimSpace(payload.Status) == "" {
		return ServerSentEvent{}, fmt.Errorf("%w: build status cannot be empty", ErrInvalidParameter)
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return ServerSentEvent{}, fmt.Errorf("%w: failed marshaling build status payload: %v", ErrInternal, err)
	}

	return NewServerSentEvent(id, EventBuildStatusChanged, data, payload.Timestamp)
}

// NewSystemHeartbeatEvent constructs an EventSystemHeartbeat ServerSentEvent.
func NewSystemHeartbeatEvent(id string, payload SystemHeartbeatPayload) (ServerSentEvent, error) {
	if strings.TrimSpace(payload.Status) == "" {
		payload.Status = "UP"
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return ServerSentEvent{}, fmt.Errorf("%w: failed marshaling heartbeat payload: %v", ErrInternal, err)
	}

	return NewServerSentEvent(id, EventSystemHeartbeat, data, payload.Timestamp)
}
