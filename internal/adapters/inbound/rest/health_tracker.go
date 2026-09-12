package rest

import (
	"sync"
)

// ProbeState represents the evaluated status state of a health probe.
type ProbeState int

const (
	// ProbeStateUnknown indicates a probe that has not yet been evaluated.
	ProbeStateUnknown ProbeState = iota
	// ProbeStateHealthy indicates a probe evaluation that succeeded (status code < 400).
	ProbeStateHealthy
	// ProbeStateUnhealthy indicates a probe evaluation that failed (status code >= 400).
	ProbeStateUnhealthy
)

// HealthProbeTracker maintains thread-safe health probe status transitions across endpoints.
type HealthProbeTracker struct {
	mu     sync.Mutex
	states map[string]ProbeState
}

// NewHealthProbeTracker constructs a thread-safe HealthProbeTracker instance.
func NewHealthProbeTracker() *HealthProbeTracker {
	return &HealthProbeTracker{
		states: make(map[string]ProbeState),
	}
}

// RecordAndCheckChange evaluates the HTTP status code for an endpoint and reports
// whether the health state changed compared to the previous evaluation.
// Returns:
// - changed: true if the state changed from unknown or flipped between healthy/unhealthy.
// - isHealthy: true if the new state is healthy (statusCode < 400).
func (t *HealthProbeTracker) RecordAndCheckChange(endpoint string, statusCode int) (changed bool, isHealthy bool) {
	t.mu.Lock()
	defer t.mu.Unlock()

	newState := ProbeStateHealthy
	if statusCode >= 400 {
		newState = ProbeStateUnhealthy
	}

	prevState, exists := t.states[endpoint]
	if !exists || prevState != newState {
		t.states[endpoint] = newState
		return true, newState == ProbeStateHealthy
	}

	return false, newState == ProbeStateHealthy
}

// Reset clears all recorded states.
func (t *HealthProbeTracker) Reset() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.states = make(map[string]ProbeState)
}
