package model

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

// RunStatus represents the lifecycle state of a TestRun aggregate.
type RunStatus string

const (
	RunStatusQueued    RunStatus = "QUEUED"
	RunStatusRunning   RunStatus = "RUNNING"
	RunStatusCompleted RunStatus = "COMPLETED"
	RunStatusFailed    RunStatus = "FAILED"
	RunStatusAborted   RunStatus = "ABORTED"
)

// IsValid checks whether the RunStatus is a recognized status.
func (s RunStatus) IsValid() bool {
	switch s {
	case RunStatusQueued, RunStatusRunning, RunStatusCompleted, RunStatusFailed, RunStatusAborted:
		return true
	default:
		return false
	}
}

// IsTerminal returns true if the status represents a final, unalterable state.
func (s RunStatus) IsTerminal() bool {
	switch s {
	case RunStatusCompleted, RunStatusFailed, RunStatusAborted:
		return true
	default:
		return false
	}
}

// RunMetrics represents indexed key performance indicator (KPI) telemetry.
type RunMetrics struct {
	TotalIterations int64
	TotalRequests   int64
	AvgTPS          float64
	P50DurationMs   float64
	P90DurationMs   float64
	P95DurationMs   float64
	P99DurationMs   float64
	ErrorRatePct    float64
}

// TestRun represents an execution instance of a compiled test suite on Kubernetes.
type TestRun struct {
	id              string
	suiteID         string
	artifactID      string
	configurationID *string
	runnerProfileID string
	scheduleID      *string
	status          RunStatus
	k8sJobName      string
	k8sNamespace    string
	startedAt       *time.Time
	finishedAt      *time.Time
	exitCode        *int
	slaPassed       *bool
	metrics         RunMetrics
	s3ReportKey     string
	s3LogsKey       string
	summaryJSON     []byte
	abortReason     string
	createdAt       time.Time
}

const DefaultRunnerNamespace = "vuhive-runners"

// NewTestRun creates a new TestRun aggregate in QUEUED status.
func NewTestRun(
	suiteID, artifactID string,
	configurationID *string,
	runnerProfileID string,
	scheduleID *string,
) (*TestRun, error) {
	trimmedSuiteID := strings.TrimSpace(suiteID)
	trimmedArtifactID := strings.TrimSpace(artifactID)
	trimmedProfileID := strings.TrimSpace(runnerProfileID)

	if trimmedSuiteID == "" || trimmedArtifactID == "" || trimmedProfileID == "" {
		return nil, ErrValidation
	}

	var cfgID *string
	if configurationID != nil && strings.TrimSpace(*configurationID) != "" {
		c := strings.TrimSpace(*configurationID)
		cfgID = &c
	}

	var schedID *string
	if scheduleID != nil && strings.TrimSpace(*scheduleID) != "" {
		s := strings.TrimSpace(*scheduleID)
		schedID = &s
	}

	now := time.Now().UTC()
	return &TestRun{
		id:              uuid.NewString(),
		suiteID:         trimmedSuiteID,
		artifactID:      trimmedArtifactID,
		configurationID: cfgID,
		runnerProfileID: trimmedProfileID,
		scheduleID:      schedID,
		status:          RunStatusQueued,
		k8sNamespace:    DefaultRunnerNamespace,
		createdAt:       now,
	}, nil
}

// NewTestRunWithID reconstructs a TestRun aggregate from persistence.
func NewTestRunWithID(
	id, suiteID, artifactID string,
	configurationID *string,
	runnerProfileID string,
	scheduleID *string,
	status RunStatus,
	k8sJobName, k8sNamespace string,
	startedAt, finishedAt *time.Time,
	exitCode *int,
	slaPassed *bool,
	metrics RunMetrics,
	s3ReportKey, s3LogsKey string,
	summaryJSON []byte,
	abortReason string,
	createdAt time.Time,
) (*TestRun, error) {
	if !status.IsValid() {
		return nil, ErrInvalidStateTransition
	}

	ns := strings.TrimSpace(k8sNamespace)
	if ns == "" {
		ns = DefaultRunnerNamespace
	}

	return &TestRun{
		id:              id,
		suiteID:         suiteID,
		artifactID:      artifactID,
		configurationID: configurationID,
		runnerProfileID: runnerProfileID,
		scheduleID:      scheduleID,
		status:          status,
		k8sJobName:      k8sJobName,
		k8sNamespace:    ns,
		startedAt:       startedAt,
		finishedAt:      finishedAt,
		exitCode:        exitCode,
		slaPassed:       slaPassed,
		metrics:         metrics,
		s3ReportKey:     s3ReportKey,
		s3LogsKey:       s3LogsKey,
		summaryJSON:     summaryJSON,
		abortReason:     abortReason,
		createdAt:       createdAt,
	}, nil
}

// ID returns the unique identifier.
func (r *TestRun) ID() string {
	return r.id
}

// EntityID implements the Entity interface.
func (r *TestRun) EntityID() string {
	return r.id
}

// AggregateType implements the AggregateRoot interface.
func (r *TestRun) AggregateType() string {
	return "TestRun"
}

// SuiteID returns the parent test suite ID.
func (r *TestRun) SuiteID() string {
	return r.suiteID
}

// ArtifactID returns the compiled artifact ID.
func (r *TestRun) ArtifactID() string {
	return r.artifactID
}

// ConfigurationID returns optional attached configuration ID.
func (r *TestRun) ConfigurationID() *string {
	return r.configurationID
}

// RunnerProfileID returns the runner profile ID.
func (r *TestRun) RunnerProfileID() string {
	return r.runnerProfileID
}

// ScheduleID returns optional schedule ID if triggered by a schedule.
func (r *TestRun) ScheduleID() *string {
	return r.scheduleID
}

// Status returns the current execution status.
func (r *TestRun) Status() RunStatus {
	return r.status
}

// K8sJobName returns the Kubernetes Job name.
func (r *TestRun) K8sJobName() string {
	return r.k8sJobName
}

// K8sNamespace returns the Kubernetes namespace.
func (r *TestRun) K8sNamespace() string {
	return r.k8sNamespace
}

// StartedAt returns the timestamp when the run began execution.
func (r *TestRun) StartedAt() *time.Time {
	return r.startedAt
}

// FinishedAt returns the timestamp when the run completed, failed, or aborted.
func (r *TestRun) FinishedAt() *time.Time {
	return r.finishedAt
}

// ExitCode returns the process exit code.
func (r *TestRun) ExitCode() *int {
	return r.exitCode
}

// SLAPassed returns whether the test run satisfied all configured SLAs.
func (r *TestRun) SLAPassed() *bool {
	return r.slaPassed
}

// Metrics returns the indexed summary metrics.
func (r *TestRun) Metrics() RunMetrics {
	return r.metrics
}

// S3ReportKey returns the storage key of summary.json.
func (r *TestRun) S3ReportKey() string {
	return r.s3ReportKey
}

// S3LogsKey returns the storage key of run.log.
func (r *TestRun) S3LogsKey() string {
	return r.s3LogsKey
}

// SummaryJSON returns raw report JSON bytes.
func (r *TestRun) SummaryJSON() []byte {
	return r.summaryJSON
}

// AbortReason returns reason why run was cancelled/aborted.
func (r *TestRun) AbortReason() string {
	return r.abortReason
}

// CreatedAt returns the timestamp when the run was requested.
func (r *TestRun) CreatedAt() time.Time {
	return r.createdAt
}

// SetK8sJobName sets the Kubernetes Job name.
func (r *TestRun) SetK8sJobName(jobName string) {
	r.k8sJobName = strings.TrimSpace(jobName)
}

// Start transitions the run from QUEUED to RUNNING.
func (r *TestRun) Start(jobName string, startTime time.Time) error {
	if r.status.IsTerminal() {
		return ErrTerminalState
	}
	if r.status != RunStatusQueued {
		return ErrInvalidStateTransition
	}

	trimmedName := strings.TrimSpace(jobName)
	if trimmedName != "" {
		r.k8sJobName = trimmedName
	}
	r.startedAt = &startTime
	r.status = RunStatusRunning
	return nil
}

// Complete transitions the run from RUNNING to COMPLETED.
func (r *TestRun) Complete(
	metrics RunMetrics,
	s3ReportKey, s3LogsKey string,
	summaryJSON []byte,
	slaPassed bool,
	finishTime time.Time,
) error {
	if r.status.IsTerminal() {
		return ErrTerminalState
	}
	if r.status != RunStatusRunning {
		return ErrInvalidStateTransition
	}

	code := 0
	r.exitCode = &code
	r.slaPassed = &slaPassed
	r.metrics = metrics
	r.s3ReportKey = strings.TrimSpace(s3ReportKey)
	r.s3LogsKey = strings.TrimSpace(s3LogsKey)
	r.summaryJSON = summaryJSON
	r.finishedAt = &finishTime
	r.status = RunStatusCompleted
	return nil
}

// Fail transitions the run to FAILED.
func (r *TestRun) Fail(exitCode int, s3LogsKey string, finishTime time.Time) error {
	if r.status.IsTerminal() {
		return ErrTerminalState
	}

	sla := false
	r.exitCode = &exitCode
	r.slaPassed = &sla
	r.s3LogsKey = strings.TrimSpace(s3LogsKey)
	r.finishedAt = &finishTime
	r.status = RunStatusFailed
	return nil
}

// Abort cancels the run transitioning it to ABORTED.
func (r *TestRun) Abort(reason string, abortTime time.Time) error {
	if r.status.IsTerminal() {
		return ErrTerminalState
	}

	r.abortReason = strings.TrimSpace(reason)
	r.finishedAt = &abortTime
	r.status = RunStatusAborted
	return nil
}

// Compile-time interface assertions
var (
	_ Entity        = (*TestRun)(nil)
	_ AggregateRoot = (*TestRun)(nil)
)
