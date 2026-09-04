package rest

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// LoggingMiddleware injects a request-scoped zerolog.Logger into context.Context and logs incoming HTTP requests.
func LoggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		reqID := c.GetHeader("X-Request-ID")
		if reqID == "" {
			reqID = uuid.NewString()
		}
		c.Header("X-Request-ID", reqID)

		reqLogger := log.Logger.With().
			Str("request_id", reqID).
			Str("method", c.Request.Method).
			Str("path", c.Request.URL.Path).
			Str("remote_ip", c.ClientIP()).
			Logger()

		ctx := reqLogger.WithContext(c.Request.Context())
		c.Request = c.Request.WithContext(ctx)

		reqLogger.Debug().Msg("incoming HTTP request")

		c.Next()

		status := c.Writer.Status()
		duration := time.Since(start)

		logEvent := reqLogger.Info()
		if status >= 500 {
			logEvent = reqLogger.Error()
		} else if status >= 400 {
			logEvent = reqLogger.Warn()
		}

		logEvent.
			Int("status", status).
			Dur("duration_ms", duration).
			Msg("completed HTTP request")
	}
}

// RecoveryMiddleware handles panics gracefully and emits structured error logs.
func RecoveryMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				reqLogger := zerolog.Ctx(c.Request.Context())
				reqLogger.Error().
					Interface("panic", r).
					Msg("panic recovered in HTTP handler")

				c.AbortWithStatusJSON(http.StatusInternalServerError, ErrorResponse{
					Error: "internal server error",
				})
			}
		}()
		c.Next()
	}
}
