package circuitbreaker

import (
	"context"
	"net/http"
	"time"
)

// CircuitBreakerChecker defines the interface for checking if a circuit breaker is open.
type CircuitBreakerChecker interface {
	IsOpen() bool
}

var (
	_ http.RoundTripper     = (*ResilientTransport)(nil)
	_ CircuitBreakerChecker = (*ResilientTransport)(nil)
)

// ResilientTransport decorates an underlying http.RoundTripper with circuit breaker
// protection and request timeout governance.
type ResilientTransport struct {
	transport http.RoundTripper
	cb        *CircuitBreaker
	timeout   time.Duration
}

// NewResilientTransport creates a new ResilientTransport instance.
func NewResilientTransport(base http.RoundTripper, cb *CircuitBreaker, timeout time.Duration) *ResilientTransport {
	if base == nil {
		base = http.DefaultTransport
	}
	return &ResilientTransport{
		transport: base,
		cb:        cb,
		timeout:   timeout,
	}
}

// RoundTrip executes an HTTP request governed by the circuit breaker and timeout.
func (t *ResilientTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Apply timeout context if timeout is configured and request has no deadline
	var cancel context.CancelFunc
	if t.timeout > 0 {
		if _, hasDeadline := req.Context().Deadline(); !hasDeadline {
			var ctx context.Context
			ctx, cancel = context.WithTimeout(req.Context(), t.timeout)
			req = req.WithContext(ctx)
		}
	}
	if cancel != nil {
		defer cancel()
	}

	// Pre-flight check through circuit breaker
	if t.cb != nil {
		done, err := t.cb.Allow()
		if err != nil {
			return nil, err
		}

		resp, err := t.transport.RoundTrip(req)
		if err != nil {
			done(false)
			return nil, err
		}

		// 5xx status codes indicate upstream server degradation
		if resp.StatusCode >= 500 {
			done(false)
		} else {
			done(true)
		}

		return resp, nil
	}

	return t.transport.RoundTrip(req)
}

// IsOpen returns whether the underlying circuit breaker is currently open.
func (t *ResilientTransport) IsOpen() bool {
	if t.cb != nil {
		return t.cb.IsOpen()
	}
	return false
}

// CircuitBreaker returns the underlying *CircuitBreaker.
func (t *ResilientTransport) CircuitBreaker() *CircuitBreaker {
	return t.cb
}
