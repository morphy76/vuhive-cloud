package circuitbreaker

import (
	"errors"
	"fmt"
	"time"

	"github.com/morphy76/vuhive-cloud/internal/bff/domain/model"
	"github.com/sony/gobreaker/v2"
)

// State represents the operational state of the circuit breaker.
type State = gobreaker.State

const (
	StateClosed   = gobreaker.StateClosed
	StateHalfOpen = gobreaker.StateHalfOpen
	StateOpen     = gobreaker.StateOpen
)

// Counts holds numbers of requests and outcomes.
type Counts = gobreaker.Counts

// Config configures the circuit breaker behavior.
type Config struct {
	Name          string
	MaxRequests   uint32
	Interval      time.Duration
	Timeout       time.Duration
	FailureRatio  float64
	MinRequests   uint32
	OnStateChange func(name string, from, to State)
}

// CircuitBreaker encapsulates a two-step circuit breaker state machine.
type CircuitBreaker struct {
	cb *gobreaker.TwoStepCircuitBreaker[any]
}

// New constructs an initialized CircuitBreaker.
func New(cfg Config) *CircuitBreaker {
	name := cfg.Name
	if name == "" {
		name = "circuit-breaker"
	}

	maxRequests := cfg.MaxRequests
	if maxRequests == 0 {
		maxRequests = 3
	}

	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 10 * time.Second
	}

	minReqs := cfg.MinRequests
	if minReqs == 0 {
		minReqs = 3
	}

	ratio := cfg.FailureRatio
	if ratio <= 0 {
		ratio = 0.5
	}

	settings := gobreaker.Settings{
		Name:        name,
		MaxRequests: maxRequests,
		Interval:    cfg.Interval,
		Timeout:     timeout,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			if counts.Requests < minReqs {
				return false
			}
			failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
			return failureRatio >= ratio
		},
		OnStateChange: func(name string, from, to gobreaker.State) {
			if cfg.OnStateChange != nil {
				cfg.OnStateChange(name, from, to)
			}
		},
	}

	return &CircuitBreaker{
		cb: gobreaker.NewTwoStepCircuitBreaker[any](settings),
	}
}

var errRecordedFailure = errors.New("circuit breaker recorded failure")

// Allow checks if a request can proceed through the circuit breaker.
// If allowed, it returns a done function to record success or failure.
// If the circuit is open, it returns model.ErrCircuitOpen.
func (c *CircuitBreaker) Allow() (done func(success bool), err error) {
	d, err := c.cb.Allow()
	if err != nil {
		if errors.Is(err, gobreaker.ErrOpenState) || errors.Is(err, gobreaker.ErrTooManyRequests) {
			return nil, fmt.Errorf("%w: %v", model.ErrCircuitOpen, err)
		}
		return nil, err
	}

	done = func(success bool) {
		if success {
			d(nil)
		} else {
			d(errRecordedFailure)
		}
	}

	return done, nil
}

// Execute runs a function within the protection of the circuit breaker.
func (c *CircuitBreaker) Execute(fn func() error) error {
	done, err := c.Allow()
	if err != nil {
		return err
	}

	err = fn()
	done(err == nil)
	return err
}

// State returns the current operational state of the circuit breaker.
func (c *CircuitBreaker) State() State {
	return c.cb.State()
}

// IsOpen returns true if the circuit breaker is in the Open state.
func (c *CircuitBreaker) IsOpen() bool {
	return c.cb.State() == StateOpen
}

// Counts returns internal request counts.
func (c *CircuitBreaker) Counts() Counts {
	return c.cb.Counts()
}

// Name returns the configured name of the circuit breaker.
func (c *CircuitBreaker) Name() string {
	return c.cb.Name()
}
