package model

import (
	"fmt"
	"strings"
	"time"
)

// BarrierStatus represents the state of a distributed barrier rendezvous session.
type BarrierStatus string

const (
	BarrierStatusPending  BarrierStatus = "PENDING"
	BarrierStatusReady    BarrierStatus = "READY"
	BarrierStatusReleased BarrierStatus = "RELEASED"
	BarrierStatusAborted  BarrierStatus = "ABORTED"
	BarrierStatusTimedOut BarrierStatus = "TIMED_OUT"
)

// WorkerStatus represents the rendezvous status of an individual worker participant.
type WorkerStatus string

const (
	WorkerStatusWaiting WorkerStatus = "WAITING"
	WorkerStatusReady   WorkerStatus = "READY"
	WorkerStatusAborted WorkerStatus = "ABORTED"
)

const (
	DefaultBarrierTimeout = 60 * time.Second
	DefaultReleaseDelay   = 300 * time.Millisecond
	MaxBarrierWorkers     = 10000
)

// WorkerParticipant represents an active worker registered in the barrier.
type WorkerParticipant struct {
	WorkerID    string
	Status      WorkerStatus
	JoinedAt    time.Time
	ReadyAt     *time.Time
	ErrorReason string
}

// BarrierSession aggregates state and participant tracking for distributed pod startup coordination.
type BarrierSession struct {
	runID           string
	totalWorkers    int
	status          BarrierStatus
	timeout         time.Duration
	releaseDelay    time.Duration
	targetStartTime *time.Time
	abortReason     string
	createdAt       time.Time
	participants    map[string]*WorkerParticipant
}

// NewBarrierSession creates a new BarrierSession in PENDING state.
func NewBarrierSession(
	runID string,
	totalWorkers int,
	timeout time.Duration,
	releaseDelay time.Duration,
) (*BarrierSession, error) {
	trimmedRunID := strings.TrimSpace(runID)
	if trimmedRunID == "" {
		return nil, fmt.Errorf("%w: run ID cannot be empty", ErrValidation)
	}
	if totalWorkers < 1 || totalWorkers > MaxBarrierWorkers {
		return nil, ErrInvalidWorkerCount
	}

	if timeout <= 0 {
		timeout = DefaultBarrierTimeout
	}
	if releaseDelay <= 0 {
		releaseDelay = DefaultReleaseDelay
	}

	return &BarrierSession{
		runID:        trimmedRunID,
		totalWorkers: totalWorkers,
		status:       BarrierStatusPending,
		timeout:      timeout,
		releaseDelay: releaseDelay,
		createdAt:    time.Now().UTC(),
		participants: make(map[string]*WorkerParticipant, totalWorkers),
	}, nil
}

// RunID returns the associated test run ID.
func (s *BarrierSession) RunID() string {
	return s.runID
}

// TotalWorkers returns the expected total count of participating workers.
func (s *BarrierSession) TotalWorkers() int {
	return s.totalWorkers
}

// Status returns the current barrier status.
func (s *BarrierSession) Status() BarrierStatus {
	return s.status
}

// Timeout returns the maximum wait duration before timing out.
func (s *BarrierSession) Timeout() time.Duration {
	return s.timeout
}

// ReleaseDelay returns the synchronization delay buffer after all workers arrive.
func (s *BarrierSession) ReleaseDelay() time.Duration {
	return s.releaseDelay
}

// TargetStartTime returns the synchronized wall-clock start timestamp if released.
func (s *BarrierSession) TargetStartTime() *time.Time {
	return s.targetStartTime
}

// AbortReason returns the reason if aborted.
func (s *BarrierSession) AbortReason() string {
	return s.abortReason
}

// CreatedAt returns when the barrier session was created.
func (s *BarrierSession) CreatedAt() time.Time {
	return s.createdAt
}

// IsTerminal returns true if the barrier is in a final, immutable state.
func (s *BarrierSession) IsTerminal() bool {
	return s.status == BarrierStatusReleased || s.status == BarrierStatusAborted || s.status == BarrierStatusTimedOut
}

// IsReleased returns true if the barrier has released all workers.
func (s *BarrierSession) IsReleased() bool {
	return s.status == BarrierStatusReleased
}

// RegisteredCount returns the number of registered workers.
func (s *BarrierSession) RegisteredCount() int {
	return len(s.participants)
}

// ReadyCount returns the number of workers that have signaled readiness.
func (s *BarrierSession) ReadyCount() int {
	count := 0
	for _, p := range s.participants {
		if p.Status == WorkerStatusReady {
			count++
		}
	}
	return count
}

// Participants returns a copy of participant states.
func (s *BarrierSession) Participants() map[string]WorkerParticipant {
	result := make(map[string]WorkerParticipant, len(s.participants))
	for k, v := range s.participants {
		result[k] = *v
	}
	return result
}

// RegisterWorker adds a worker to the session in WAITING status.
func (s *BarrierSession) RegisterWorker(workerID string) error {
	trimmedID := strings.TrimSpace(workerID)
	if trimmedID == "" {
		return fmt.Errorf("%w: worker ID cannot be empty", ErrValidation)
	}

	if s.status == BarrierStatusAborted {
		return ErrBarrierAborted
	}
	if s.status == BarrierStatusTimedOut {
		return ErrBarrierTimeout
	}
	if s.status == BarrierStatusReleased {
		return ErrBarrierReleased
	}

	if _, exists := s.participants[trimmedID]; exists {
		return ErrWorkerAlreadyRegistered
	}

	s.participants[trimmedID] = &WorkerParticipant{
		WorkerID: trimmedID,
		Status:   WorkerStatusWaiting,
		JoinedAt: time.Now().UTC(),
	}

	return nil
}

// MarkWorkerReady updates a worker's state to READY and checks if all workers are ready.
func (s *BarrierSession) MarkWorkerReady(workerID string) (bool, error) {
	trimmedID := strings.TrimSpace(workerID)
	if s.status == BarrierStatusAborted {
		return false, ErrBarrierAborted
	}
	if s.status == BarrierStatusTimedOut {
		return false, ErrBarrierTimeout
	}
	if s.status == BarrierStatusReleased {
		return false, ErrBarrierReleased
	}

	p, exists := s.participants[trimmedID]
	if !exists {
		return false, ErrNotFound
	}

	now := time.Now().UTC()
	p.Status = WorkerStatusReady
	p.ReadyAt = &now

	allReady := s.ReadyCount() >= s.totalWorkers
	if allReady {
		s.status = BarrierStatusReady
	}

	return allReady, nil
}

// MarkWorkerAborted marks a worker as aborted and transitions the whole session to ABORTED.
func (s *BarrierSession) MarkWorkerAborted(workerID, reason string) error {
	trimmedID := strings.TrimSpace(workerID)
	trimmedReason := strings.TrimSpace(reason)
	if trimmedReason == "" {
		trimmedReason = "worker aborted rendezvous"
	}

	if p, exists := s.participants[trimmedID]; exists {
		p.Status = WorkerStatusAborted
		p.ErrorReason = trimmedReason
	}

	s.status = BarrierStatusAborted
	s.abortReason = trimmedReason
	return nil
}

// Release transitions the session to RELEASED and sets the synchronized target start timestamp.
func (s *BarrierSession) Release() (*time.Time, error) {
	if s.status == BarrierStatusReleased {
		return nil, ErrBarrierReleased
	}
	if s.status == BarrierStatusAborted {
		return nil, ErrBarrierAborted
	}
	if s.status == BarrierStatusTimedOut {
		return nil, ErrBarrierTimeout
	}

	target := time.Now().UTC().Add(s.releaseDelay)
	s.targetStartTime = &target
	s.status = BarrierStatusReleased
	return &target, nil
}

// HasTimedOut returns true if the given reference time exceeds creation time + timeout.
func (s *BarrierSession) HasTimedOut(now time.Time) bool {
	return now.After(s.createdAt.Add(s.timeout))
}

// TimeoutSession transitions the session to TIMED_OUT.
func (s *BarrierSession) TimeoutSession() error {
	if s.IsTerminal() {
		return ErrTerminalState
	}
	s.status = BarrierStatusTimedOut
	return nil
}
