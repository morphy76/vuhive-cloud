package event

import (
	"time"

	"github.com/morphy76/vuhive-cloud/internal/domain/model"
)

// DomainEvent represents an immutable event that occurred in the domain.
type DomainEvent interface {
	EventName() string
	OccurredAt() time.Time
}

// RunStarted indicates that a TestRun has been dispatched and started on Kubernetes.
type RunStarted struct {
	RunID      string
	SuiteID    string
	K8sJobName string
	occurredAt time.Time
}

func NewRunStarted(runID, suiteID, k8sJobName string, occurredAt time.Time) *RunStarted {
	return &RunStarted{
		RunID:      runID,
		SuiteID:    suiteID,
		K8sJobName: k8sJobName,
		occurredAt: occurredAt,
	}
}

func (e *RunStarted) EventName() string {
	return "RunStarted"
}

func (e *RunStarted) OccurredAt() time.Time {
	return e.occurredAt
}

// RunCompleted indicates that a TestRun finished successfully and its report was ingested.
type RunCompleted struct {
	RunID      string
	SuiteID    string
	SLAPassed  bool
	Metrics    model.RunMetrics
	occurredAt time.Time
}

func NewRunCompleted(runID, suiteID string, slaPassed bool, metrics model.RunMetrics, occurredAt time.Time) *RunCompleted {
	return &RunCompleted{
		RunID:      runID,
		SuiteID:    suiteID,
		SLAPassed:  slaPassed,
		Metrics:    metrics,
		occurredAt: occurredAt,
	}
}

func (e *RunCompleted) EventName() string {
	return "RunCompleted"
}

func (e *RunCompleted) OccurredAt() time.Time {
	return e.occurredAt
}

// RunFailed indicates that a TestRun failed due to non-zero exit code or execution errors.
type RunFailed struct {
	RunID      string
	SuiteID    string
	ExitCode   int
	occurredAt time.Time
}

func NewRunFailed(runID, suiteID string, exitCode int, occurredAt time.Time) *RunFailed {
	return &RunFailed{
		RunID:      runID,
		SuiteID:    suiteID,
		ExitCode:   exitCode,
		occurredAt: occurredAt,
	}
}

func (e *RunFailed) EventName() string {
	return "RunFailed"
}

func (e *RunFailed) OccurredAt() time.Time {
	return e.occurredAt
}

// RunAborted indicates that a TestRun was cancelled/aborted prior to completion.
type RunAborted struct {
	RunID      string
	SuiteID    string
	Reason     string
	occurredAt time.Time
}

func NewRunAborted(runID, suiteID, reason string, occurredAt time.Time) *RunAborted {
	return &RunAborted{
		RunID:      runID,
		SuiteID:    suiteID,
		Reason:     reason,
		occurredAt: occurredAt,
	}
}

func (e *RunAborted) EventName() string {
	return "RunAborted"
}

func (e *RunAborted) OccurredAt() time.Time {
	return e.occurredAt
}

// ArtifactReady indicates that an artifact build was completed and stored in S3.
type ArtifactReady struct {
	ArtifactID  string
	SuiteID     string
	Platform    model.Platform
	S3BinaryKey string
	Checksum    string
	occurredAt  time.Time
}

func NewArtifactReady(artifactID, suiteID string, platform model.Platform, s3Key, checksum string, occurredAt time.Time) *ArtifactReady {
	return &ArtifactReady{
		ArtifactID:  artifactID,
		SuiteID:     suiteID,
		Platform:    platform,
		S3BinaryKey: s3Key,
		Checksum:    checksum,
		occurredAt:  occurredAt,
	}
}

func (e *ArtifactReady) EventName() string {
	return "ArtifactReady"
}

func (e *ArtifactReady) OccurredAt() time.Time {
	return e.occurredAt
}

// BuildFailed indicates that an artifact build job or AST validation failed.
type BuildFailed struct {
	ArtifactID   string
	SuiteID      string
	ErrorMessage string
	occurredAt   time.Time
}

func NewBuildFailed(artifactID, suiteID, errorMessage string, occurredAt time.Time) *BuildFailed {
	return &BuildFailed{
		ArtifactID:   artifactID,
		SuiteID:      suiteID,
		ErrorMessage: errorMessage,
		occurredAt:   occurredAt,
	}
}

func (e *BuildFailed) EventName() string {
	return "BuildFailed"
}

func (e *BuildFailed) OccurredAt() time.Time {
	return e.occurredAt
}

// Compile-time interface assertions
var (
	_ DomainEvent = (*RunStarted)(nil)
	_ DomainEvent = (*RunCompleted)(nil)
	_ DomainEvent = (*RunFailed)(nil)
	_ DomainEvent = (*RunAborted)(nil)
	_ DomainEvent = (*ArtifactReady)(nil)
	_ DomainEvent = (*BuildFailed)(nil)
)
