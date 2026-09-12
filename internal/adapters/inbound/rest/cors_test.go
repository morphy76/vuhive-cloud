package rest_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/morphy76/vuhive-cloud/internal/adapters/inbound/rest"
	"github.com/stretchr/testify/assert"
)

func TestCORSMiddleware_DefaultWildcard(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := rest.SetupRouter(nil, nil, nil, nil)

	t.Run("OPTIONS /api/openapi.json returns 204 with CORS headers", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodOptions, "/api/openapi.json", nil)
		req.Header.Set("Origin", "http://localhost:8081")
		req.Header.Set("Access-Control-Request-Method", "GET")
		req.Header.Set("Access-Control-Request-Headers", "Content-Type, X-Request-ID")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)
		assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
		assert.Contains(t, w.Header().Get("Access-Control-Allow-Methods"), "GET")
		assert.Contains(t, w.Header().Get("Access-Control-Allow-Methods"), "OPTIONS")
		assert.Contains(t, w.Header().Get("Access-Control-Allow-Headers"), "Origin")
		assert.Contains(t, w.Header().Get("Access-Control-Allow-Headers"), "X-Request-ID")
		assert.Contains(t, w.Header().Get("Access-Control-Expose-Headers"), "X-Request-ID")
		assert.Equal(t, "86400", w.Header().Get("Access-Control-Max-Age"))
	})

	t.Run("OPTIONS /api/openapi.yaml returns 204 with CORS headers", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodOptions, "/api/openapi.yaml", nil)
		req.Header.Set("Origin", "http://localhost:8081")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)
		assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
	})

	t.Run("OPTIONS /api/v1/health returns 204 with CORS headers", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodOptions, "/api/v1/health", nil)
		req.Header.Set("Origin", "http://localhost:3000")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)
		assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
	})

	t.Run("OPTIONS on unmapped path returns 204 with CORS headers", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodOptions, "/api/v1/nonexistent", nil)
		req.Header.Set("Origin", "http://localhost:8081")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)
		assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
	})

	t.Run("GET /api/openapi.json with Origin returns 200 with CORS headers", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/openapi.json", nil)
		req.Header.Set("Origin", "http://localhost:8081")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
		assert.Contains(t, w.Header().Get("Access-Control-Expose-Headers"), "X-Request-ID")
	})

	t.Run("GET /healthz without Origin still succeeds", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestCORSMiddleware_ConfiguredOrigins(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("Allowed origin matches request origin", func(t *testing.T) {
		r := gin.New()
		r.Use(rest.CORSMiddleware("http://trusted-app.local", "https://dashboard.example.com"))
		r.GET("/test", func(c *gin.Context) {
			c.Status(http.StatusOK)
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Origin", "http://trusted-app.local")
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "http://trusted-app.local", w.Header().Get("Access-Control-Allow-Origin"))
		assert.Equal(t, "Origin", w.Header().Get("Vary"))
	})

	t.Run("Disallowed origin does not get Access-Control-Allow-Origin", func(t *testing.T) {
		r := gin.New()
		r.Use(rest.CORSMiddleware("http://trusted-app.local"))
		r.GET("/test", func(c *gin.Context) {
			c.Status(http.StatusOK)
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Origin", "http://malicious-site.com")
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Empty(t, w.Header().Get("Access-Control-Allow-Origin"))
	})

	t.Run("Environment variable CORS_ALLOWED_ORIGINS is parsed", func(t *testing.T) {
		t.Setenv("CORS_ALLOWED_ORIGINS", "http://env-origin-1.local, http://env-origin-2.local")

		r := gin.New()
		r.Use(rest.CORSMiddleware())
		r.OPTIONS("/test", func(c *gin.Context) {
			c.Status(http.StatusOK)
		})

		req := httptest.NewRequest(http.MethodOptions, "/test", nil)
		req.Header.Set("Origin", "http://env-origin-2.local")
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)
		assert.Equal(t, "http://env-origin-2.local", w.Header().Get("Access-Control-Allow-Origin"))
		assert.Equal(t, "Origin", w.Header().Get("Vary"))
	})
}
