package rest

import (
	"net/http"
	"os"
	"strings"
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

// CORSMiddleware provides Cross-Origin Resource Sharing (CORS) headers and handles
// preflight HTTP OPTIONS requests for OpenAPI and REST endpoints.
// Allowed origins can be passed directly as arguments, or retrieved from the
// CORS_ALLOWED_ORIGINS environment variable (comma-separated).
// If no origins are configured, it defaults to allowing all origins ("*").
func CORSMiddleware(origins ...string) gin.HandlerFunc {
	var allowedOrigins []string
	if len(origins) > 0 {
		for _, o := range origins {
			for _, part := range strings.Split(o, ",") {
				trimmed := strings.TrimSpace(part)
				if trimmed != "" {
					allowedOrigins = append(allowedOrigins, trimmed)
				}
			}
		}
	} else if envOrigins := os.Getenv("CORS_ALLOWED_ORIGINS"); envOrigins != "" {
		for _, part := range strings.Split(envOrigins, ",") {
			trimmed := strings.TrimSpace(part)
			if trimmed != "" {
				allowedOrigins = append(allowedOrigins, trimmed)
			}
		}
	}

	allowAll := len(allowedOrigins) == 0
	for _, o := range allowedOrigins {
		if o == "*" {
			allowAll = true
			break
		}
	}

	allowedOriginsMap := make(map[string]struct{}, len(allowedOrigins))
	for _, o := range allowedOrigins {
		allowedOriginsMap[o] = struct{}{}
	}

	return func(c *gin.Context) {
		reqOrigin := c.GetHeader("Origin")

		if allowAll {
			c.Header("Access-Control-Allow-Origin", "*")
		} else if reqOrigin != "" {
			if _, ok := allowedOriginsMap[reqOrigin]; ok {
				c.Header("Access-Control-Allow-Origin", reqOrigin)
				c.Header("Vary", "Origin")
			}
		}

		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, HEAD")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, X-Request-ID")
		c.Header("Access-Control-Expose-Headers", "Content-Length, X-Request-ID")
		c.Header("Access-Control-Max-Age", "86400")

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
