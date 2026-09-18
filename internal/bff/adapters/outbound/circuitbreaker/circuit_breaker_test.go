package circuitbreaker_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/morphy76/vuhive-cloud/internal/bff/adapters/outbound/circuitbreaker"
	"github.com/morphy76/vuhive-cloud/internal/bff/domain/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type roundTripperFunc func(req *http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestCircuitBreaker_StateTransitions(t *testing.T) {
	var stateTransitions []string
	var mu sync.Mutex

	cb := circuitbreaker.New(circuitbreaker.Config{
		Name:         "test-cb",
		MaxRequests:  2,
		Timeout:      100 * time.Millisecond,
		FailureRatio: 0.5,
		MinRequests:  3,
		OnStateChange: func(name string, from, to circuitbreaker.State) {
			mu.Lock()
			defer mu.Unlock()
			stateTransitions = append(stateTransitions, from.String()+"->"+to.String())
		},
	})

	assert.Equal(t, circuitbreaker.StateClosed, cb.State())
	assert.False(t, cb.IsOpen())

	// 3 consecutive failures to trip the breaker
	for i := 0; i < 3; i++ {
		done, err := cb.Allow()
		require.NoError(t, err)
		done(false) // record failure
	}

	// Breaker should now be Open
	assert.Equal(t, circuitbreaker.StateOpen, cb.State())
	assert.True(t, cb.IsOpen())

	// Subsequent calls should be rejected immediately with ErrCircuitOpen
	_, err := cb.Allow()
	require.Error(t, err)
	assert.True(t, errors.Is(err, model.ErrCircuitOpen))

	// Wait for open timeout to elapse
	time.Sleep(120 * time.Millisecond)

	// In Half-Open state, MaxRequests (2) consecutive successes will transition back to Closed
	done1, err := cb.Allow()
	require.NoError(t, err)
	assert.Equal(t, circuitbreaker.StateHalfOpen, cb.State())
	done1(true)

	done2, err := cb.Allow()
	require.NoError(t, err)
	done2(true)

	// Breaker should now be back to Closed
	assert.Equal(t, circuitbreaker.StateClosed, cb.State())
	assert.False(t, cb.IsOpen())

	mu.Lock()
	defer mu.Unlock()
	assert.Contains(t, stateTransitions, "closed->open")
	assert.Contains(t, stateTransitions, "open->half-open")
	assert.Contains(t, stateTransitions, "half-open->closed")
}

func TestResilientTransport(t *testing.T) {
	t.Run("successful request under 200 returns response", func(t *testing.T) {
		cb := circuitbreaker.New(circuitbreaker.Config{
			Name:         "transport-success",
			MaxRequests:  1,
			Timeout:      100 * time.Millisecond,
			FailureRatio: 0.5,
		})
		base := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBufferString(`ok`)),
				Header:     make(http.Header),
			}, nil
		})

		transport := circuitbreaker.NewResilientTransport(base, cb, 2*time.Second)
		req := httptest.NewRequest(http.MethodGet, "http://controlplane/api/v1/test", nil)
		resp, err := transport.RoundTrip(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("4xx does not trip circuit breaker", func(t *testing.T) {
		cb := circuitbreaker.New(circuitbreaker.Config{
			Name:         "transport-4xx",
			MaxRequests:  1,
			Timeout:      100 * time.Millisecond,
			FailureRatio: 0.5,
			MinRequests:  3,
		})
		base := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusNotFound,
				Body:       io.NopCloser(bytes.NewBufferString(`not found`)),
				Header:     make(http.Header),
			}, nil
		})

		transport := circuitbreaker.NewResilientTransport(base, cb, 2*time.Second)
		req := httptest.NewRequest(http.MethodGet, "http://controlplane/api/v1/missing", nil)
		for i := 0; i < 5; i++ {
			resp, err := transport.RoundTrip(req)
			require.NoError(t, err)
			assert.Equal(t, http.StatusNotFound, resp.StatusCode)
		}
		assert.Equal(t, circuitbreaker.StateClosed, cb.State())
	})

	t.Run("5xx trips circuit breaker open and fast fails", func(t *testing.T) {
		cb := circuitbreaker.New(circuitbreaker.Config{
			Name:         "transport-5xx",
			MaxRequests:  1,
			Timeout:      100 * time.Millisecond,
			FailureRatio: 0.5,
			MinRequests:  3,
		})
		base := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusInternalServerError,
				Body:       io.NopCloser(bytes.NewBufferString(`server error`)),
				Header:     make(http.Header),
			}, nil
		})

		transport := circuitbreaker.NewResilientTransport(base, cb, 2*time.Second)
		req := httptest.NewRequest(http.MethodGet, "http://controlplane/api/v1/error", nil)

		// Cause repeated 500s to trip the breaker
		for i := 0; i < 3; i++ {
			resp, err := transport.RoundTrip(req)
			require.NoError(t, err)
			assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
		}

		assert.True(t, cb.IsOpen())

		// Subsequent request fast fails immediately
		_, err := transport.RoundTrip(req)
		require.Error(t, err)
		assert.True(t, errors.Is(err, model.ErrCircuitOpen))
	})

	t.Run("timeout cancels hanging requests", func(t *testing.T) {
		hangingBase := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			<-req.Context().Done()
			return nil, req.Context().Err()
		})

		isolatedCB := circuitbreaker.New(circuitbreaker.Config{
			Name:    "timeout-cb",
			Timeout: 1 * time.Second,
		})

		transport := circuitbreaker.NewResilientTransport(hangingBase, isolatedCB, 50*time.Millisecond)
		req := httptest.NewRequest(http.MethodGet, "http://controlplane/api/v1/hang", nil)

		start := time.Now()
		_, err := transport.RoundTrip(req)
		duration := time.Since(start)

		require.Error(t, err)
		assert.True(t, errors.Is(err, context.DeadlineExceeded))
		assert.Less(t, duration, 500*time.Millisecond)
	})
}
