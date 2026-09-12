package rest

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

// DefaultHealthProbePaths identifies endpoints treated as health probes in BFF for stateful logging.
var DefaultHealthProbePaths = map[string]bool{
	"/healthz": true,
}

var defaultHealthProbeTracker = NewHealthProbeTracker()

// LoggingMiddleware logs incoming HTTP requests using zerolog.
// For health probe endpoints, logging is stateful and emits only upon status change (Info on good, Warn on bad).
func LoggingMiddleware() gin.HandlerFunc {
	return LoggingMiddlewareWithTracker(defaultHealthProbeTracker)
}

// LoggingMiddlewareWithTracker creates a LoggingMiddleware with an explicit HealthProbeTracker.
func LoggingMiddlewareWithTracker(tracker *HealthProbeTracker) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery
		isProbe := DefaultHealthProbePaths[path]

		reqID := c.GetHeader("X-Request-ID")
		if reqID == "" {
			reqID = uuid.New().String()
		}
		c.Header("X-Request-ID", reqID)

		reqLog := log.With().
			Str("request_id", reqID).
			Str("method", c.Request.Method).
			Str("path", path).
			Logger()

		ctx := reqLog.WithContext(c.Request.Context())
		c.Request = c.Request.WithContext(ctx)

		if isProbe {
			c.Next()

			statusCode := c.Writer.Status()
			latency := time.Since(start)

			changed, isHealthy := tracker.RecordAndCheckChange(path, statusCode)
			if !changed {
				return
			}

			if isHealthy {
				reqLog.Info().
					Int("status", statusCode).
					Dur("latency_ms", latency).
					Str("client_ip", c.ClientIP()).
					Msg("health probe status changed to healthy")
			} else {
				reqLog.Warn().
					Int("status", statusCode).
					Dur("latency_ms", latency).
					Str("client_ip", c.ClientIP()).
					Msg("health probe status changed to unhealthy")
			}
			return
		}

		c.Next()

		latency := time.Since(start)
		statusCode := c.Writer.Status()

		evt := reqLog.Info()
		if statusCode >= 500 {
			evt = reqLog.Error()
		} else if statusCode >= 400 {
			evt = reqLog.Warn()
		}

		if raw != "" {
			evt = evt.Str("query", raw)
		}

		evt.
			Int("status", statusCode).
			Dur("latency_ms", latency).
			Str("client_ip", c.ClientIP()).
			Msg("handled http request")
	}
}

// RecoveryMiddleware handles panics cleanly and returns HTTP 500.
func RecoveryMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				log.Error().Interface("panic", err).Msg("recovered from panic in http handler")
				c.AbortWithStatusJSON(http.StatusInternalServerError, ErrorResponse{
					Error: "internal server error",
				})
			}
		}()
		c.Next()
	}
}
