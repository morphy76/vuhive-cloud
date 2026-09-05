package rest

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

// LoggingMiddleware logs incoming HTTP requests using zerolog.
func LoggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

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
			path = path + "?" + raw
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
