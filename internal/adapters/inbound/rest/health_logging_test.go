package rest_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/morphy76/vuhive-cloud/internal/adapters/inbound/rest"
)

type logEntry struct {
	Level      string  `json:"level"`
	Message    string  `json:"message"`
	Path       string  `json:"path"`
	Status     int     `json:"status"`
	DurationMs float64 `json:"duration_ms"`
}

func parseLogs(buf *bytes.Buffer) []logEntry {
	var entries []logEntry
	decoder := json.NewDecoder(bytes.NewReader(buf.Bytes()))
	for decoder.More() {
		var entry logEntry
		if err := decoder.Decode(&entry); err == nil {
			entries = append(entries, entry)
		}
	}
	return entries
}

func TestHealthProbeTracker_Transitions(t *testing.T) {
	tracker := rest.NewHealthProbeTracker()

	// Initial check: 200 OK -> changed = true, isHealthy = true
	changed, isHealthy := tracker.RecordAndCheckChange("/healthz", http.StatusOK)
	assert.True(t, changed, "first check should report state change")
	assert.True(t, isHealthy, "status 200 should be healthy")

	// Consecutive check: 200 OK -> changed = false, isHealthy = true
	changed, isHealthy = tracker.RecordAndCheckChange("/healthz", http.StatusOK)
	assert.False(t, changed, "consecutive same-status check should not report state change")
	assert.True(t, isHealthy)

	// Another consecutive check: 204 OK -> changed = false, isHealthy = true
	changed, isHealthy = tracker.RecordAndCheckChange("/healthz", http.StatusNoContent)
	assert.False(t, changed, "consecutive 2xx status should remain healthy without change")
	assert.True(t, isHealthy)

	// Transition to unhealthy: 503 -> changed = true, isHealthy = false
	changed, isHealthy = tracker.RecordAndCheckChange("/healthz", http.StatusServiceUnavailable)
	assert.True(t, changed, "transition to unhealthy should report state change")
	assert.False(t, isHealthy, "status 503 should be unhealthy")

	// Consecutive unhealthy: 500 -> changed = false, isHealthy = false
	changed, isHealthy = tracker.RecordAndCheckChange("/healthz", http.StatusInternalServerError)
	assert.False(t, changed, "consecutive unhealthy check should not report state change")
	assert.False(t, isHealthy)

	// Recovery to healthy: 200 -> changed = true, isHealthy = true
	changed, isHealthy = tracker.RecordAndCheckChange("/healthz", http.StatusOK)
	assert.True(t, changed, "recovery to healthy should report state change")
	assert.True(t, isHealthy)

	// Independent endpoint tracking: /api/v1/health should start unknown
	changed, isHealthy = tracker.RecordAndCheckChange("/api/v1/health", http.StatusInternalServerError)
	assert.True(t, changed, "new endpoint should report initial state change")
	assert.False(t, isHealthy)
}

func TestHealthProbeTracker_Concurrent(t *testing.T) {
	tracker := rest.NewHealthProbeTracker()
	var wg sync.WaitGroup
	const goroutines = 50

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			status := http.StatusOK
			if idx%2 == 0 {
				status = http.StatusServiceUnavailable
			}
			tracker.RecordAndCheckChange("/healthz", status)
		}(i)
	}

	wg.Wait()
}

func TestLoggingMiddleware_HealthProbeStatefulLogging(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Capture zerolog output to buffer
	var logBuf bytes.Buffer
	origLogger := log.Logger
	log.Logger = zerolog.New(&logBuf)
	defer func() {
		log.Logger = origLogger
	}()

	tracker := rest.NewHealthProbeTracker()
	r := gin.New()
	r.Use(rest.LoggingMiddlewareWithTracker(tracker))

	var currentHealthStatus = http.StatusOK
	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(currentHealthStatus, gin.H{"status": "ok"})
	})
	r.GET("/version", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"version": "1.0.0"})
	})

	t.Run("first health probe logs Info", func(t *testing.T) {
		logBuf.Reset()
		req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		entries := parseLogs(&logBuf)
		require.Len(t, entries, 1, "first probe must emit exactly one log entry")
		assert.Equal(t, "info", entries[0].Level, "good final status must log at info level")
		assert.Equal(t, "/healthz", entries[0].Path)
		assert.Equal(t, http.StatusOK, entries[0].Status)
	})

	t.Run("consecutive healthy probes produce zero logs", func(t *testing.T) {
		for i := 0; i < 5; i++ {
			logBuf.Reset()
			req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
			entries := parseLogs(&logBuf)
			assert.Empty(t, entries, "consecutive healthy probe must produce zero log output")
		}
	})

	t.Run("transition to unhealthy logs Warn", func(t *testing.T) {
		currentHealthStatus = http.StatusServiceUnavailable
		logBuf.Reset()

		req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusServiceUnavailable, w.Code)
		entries := parseLogs(&logBuf)
		require.Len(t, entries, 1, "transition to unhealthy must emit exactly one log entry")
		assert.Equal(t, "warn", entries[0].Level, "bad final status must log at warn level")
		assert.Equal(t, "/healthz", entries[0].Path)
		assert.Equal(t, http.StatusServiceUnavailable, entries[0].Status)
	})

	t.Run("consecutive unhealthy probes produce zero logs", func(t *testing.T) {
		for i := 0; i < 5; i++ {
			logBuf.Reset()
			req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, http.StatusServiceUnavailable, w.Code)
			entries := parseLogs(&logBuf)
			assert.Empty(t, entries, "consecutive unhealthy probe must produce zero log output")
		}
	})

	t.Run("recovery back to healthy logs Info", func(t *testing.T) {
		currentHealthStatus = http.StatusOK
		logBuf.Reset()

		req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		entries := parseLogs(&logBuf)
		require.Len(t, entries, 1, "recovery to healthy must emit exactly one log entry")
		assert.Equal(t, "info", entries[0].Level, "recovered good final status must log at info level")
		assert.Equal(t, "/healthz", entries[0].Path)
		assert.Equal(t, http.StatusOK, entries[0].Status)
	})

	t.Run("non-probe route logs normally on every request", func(t *testing.T) {
		logBuf.Reset()
		req := httptest.NewRequest(http.MethodGet, "/version", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		entries := parseLogs(&logBuf)
		// Non-probe route emits debug on enter and info on complete
		assert.NotEmpty(t, entries, "non-probe endpoint must still emit logs")
	})
}
