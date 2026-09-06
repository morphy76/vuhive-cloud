package rest

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

// NewControlPlaneProxy creates a reverse proxy Gin handler that transparently forwards requests
// from /api/bff/v1/<path> to the upstream control plane server under /api/v1/<path>.
func NewControlPlaneProxy(targetBaseURL string, transport http.RoundTripper) gin.HandlerFunc {
	parsedURL, err := url.Parse(strings.TrimRight(targetBaseURL, "/"))
	if err != nil {
		panic("invalid targetBaseURL for control plane proxy: " + err.Error())
	}

	proxy := &httputil.ReverseProxy{
		Transport: transport,
		Director: func(req *http.Request) {
			req.URL.Scheme = parsedURL.Scheme
			req.URL.Host = parsedURL.Host
			req.Host = parsedURL.Host

			// Rewrite /api/bff/v1/ prefix to /api/v1/
			path := req.URL.Path
			if strings.HasPrefix(path, "/api/bff/v1/") {
				req.URL.Path = "/api/v1/" + strings.TrimPrefix(path, "/api/bff/v1/")
			} else if path == "/api/bff/v1" {
				req.URL.Path = "/api/v1"
			}
		},
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			log := zerolog.Ctx(r.Context()).With().
				Str("op", "ControlPlaneProxy").
				Str("target", targetBaseURL).
				Logger()
			log.Error().Err(err).Msg("control plane reverse proxy request failed")
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadGateway)
			_, _ = w.Write([]byte(`{"error":"control plane gateway error"}`))
		},
	}

	return func(c *gin.Context) {
		start := time.Now()
		log := zerolog.Ctx(c.Request.Context()).With().
			Str("op", "ControlPlaneProxy.Serve").
			Str("method", c.Request.Method).
			Str("path", c.Request.URL.Path).
			Logger()
		log.Debug().Msg("proxying request to control plane")

		proxy.ServeHTTP(proxyResponseWriter{ResponseWriter: c.Writer}, c.Request)

		log.Info().
			Int("status", c.Writer.Status()).
			Dur("duration_ms", time.Since(start)).
			Msg("completed proxied request to control plane")
	}
}
