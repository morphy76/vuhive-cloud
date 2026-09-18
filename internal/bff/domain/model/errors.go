package model

import (
	"errors"
	"fmt"
)

var (
	// ErrSessionNotFound is returned when a requested BFF session does not exist.
	ErrSessionNotFound = errors.New("bff session not found")

	// ErrSessionExpired is returned when attempting an operation on an expired session.
	ErrSessionExpired = errors.New("bff session has expired")

	// ErrInvalidSession is returned when a session aggregate is in an invalid state.
	ErrInvalidSession = errors.New("invalid bff session")

	// ErrConcurrentSessionModification is returned when optimistic locking or concurrent write conflicts occur.
	ErrConcurrentSessionModification = errors.New("concurrent session modification")

	// ErrControlPlaneUnavailable is returned when upstream cmd/server cannot be reached.
	ErrControlPlaneUnavailable = errors.New("control plane service is unavailable")

	// ErrInvalidParameter is returned when an input argument fails validation.
	ErrInvalidParameter = errors.New("invalid parameter provided")

	// ErrUnauthorized is returned when caller authentication or credentials are missing or invalid.
	ErrUnauthorized = errors.New("unauthorized request")

	// ErrRunNotFound is returned when a requested test run cannot be found.
	ErrRunNotFound = errors.New("test run not found")

	// ErrSuiteNotFound is returned when a requested test suite cannot be found.
	ErrSuiteNotFound = errors.New("test suite not found")

	// ErrProfileNotFound is returned when a requested runner profile cannot be found.
	ErrProfileNotFound = errors.New("runner profile not found")

	// ErrInternal is returned when an unexpected internal error occurs.
	ErrInternal = errors.New("internal bff error")

	// ErrCircuitOpen is returned when an upstream service circuit breaker is open.
	ErrCircuitOpen = errors.New("circuit breaker is open")
)

// DomainError wraps a base domain sentinel error with contextual details.
type DomainError struct {
	Sentinel error
	Cause    error
}

// Error formats the wrapped domain error string.
func (e *DomainError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Sentinel.Error(), e.Cause)
	}
	return e.Sentinel.Error()
}

// Unwrap returns the underlying sentinel and cause errors for errors.Is and errors.As traversal.
func (e *DomainError) Unwrap() []error {
	if e.Cause != nil {
		return []error{e.Sentinel, e.Cause}
	}
	return []error{e.Sentinel}
}

// NewDomainError creates a new DomainError instance wrapping a sentinel with cause.
func NewDomainError(sentinel, cause error) error {
	return &DomainError{
		Sentinel: sentinel,
		Cause:    cause,
	}
}
