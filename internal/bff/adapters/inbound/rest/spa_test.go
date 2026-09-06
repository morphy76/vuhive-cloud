package rest_test

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/gin-gonic/gin"
	"github.com/morphy76/vuhive-cloud/internal/bff/adapters/inbound/rest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestFileSystem() fstest.MapFS {
	return fstest.MapFS{
		"index.html": &fstest.MapFile{
			Data: []byte("<!DOCTYPE html><html><head><title>App</title></head><body><div id=\"root\"></div></body></html>"),
		},
		"manifest.webmanifest": &fstest.MapFile{
			Data: []byte(`{"name":"vuhive-cloud","display":"standalone"}`),
		},
		"manifest.json": &fstest.MapFile{
			Data: []byte(`{"name":"vuhive-cloud","display":"standalone"}`),
		},
		"sw.js": &fstest.MapFile{
			Data: []byte(`self.addEventListener('fetch', () => {});`),
		},
		"favicon.svg": &fstest.MapFile{
			Data: []byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 10 10"></svg>`),
		},
		"assets/index-hash123.js": &fstest.MapFile{
			Data: []byte(`console.log("bundle");`),
		},
		"assets/style-hash123.css": &fstest.MapFile{
			Data: []byte(`body { margin: 0; }`),
		},
	}
}

func setupTestRouterWithSPA(fs fstest.MapFS, devProxyURL *url.URL) *gin.Engine {
	mockSvc := new(MockBFFService)
	spaConfig := rest.SPAConfig{
		FileSystem:  fs,
		DevProxyURL: devProxyURL,
	}
	return rest.SetupRouter(mockSvc, "0.1.0", spaConfig)
}

func TestSPAHandler_StaticAndSPARouting(t *testing.T) {
	fs := newTestFileSystem()
	router := setupTestRouterWithSPA(fs, nil)

	t.Run("GET / returns index.html with no-cache headers", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Header().Get("Content-Type"), "text/html")
		assert.Equal(t, "no-cache, no-store, must-revalidate", rec.Header().Get("Cache-Control"))
		assert.Contains(t, rec.Body.String(), "<div id=\"root\"></div>")
	})

	t.Run("GET /suites/123/runs returns index.html for deep client routes", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/suites/123/runs", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Header().Get("Content-Type"), "text/html")
		assert.Equal(t, "no-cache, no-store, must-revalidate", rec.Header().Get("Cache-Control"))
		assert.Contains(t, rec.Body.String(), "<div id=\"root\"></div>")
	})

	t.Run("GET /assets/index-hash123.js returns 200 with immutable cache", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/assets/index-hash123.js", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Header().Get("Content-Type"), "javascript")
		assert.Equal(t, "public, max-age=31536000, immutable", rec.Header().Get("Cache-Control"))
		assert.Equal(t, `console.log("bundle");`, rec.Body.String())
	})

	t.Run("GET /assets/style-hash123.css returns 200 with immutable cache", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/assets/style-hash123.css", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Header().Get("Content-Type"), "text/css")
		assert.Equal(t, "public, max-age=31536000, immutable", rec.Header().Get("Cache-Control"))
		assert.Equal(t, `body { margin: 0; }`, rec.Body.String())
	})

	t.Run("GET /assets/notfound.js returns 404 and does NOT fallback to index.html", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/assets/notfound.js", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusNotFound, rec.Code)
		assert.NotContains(t, rec.Body.String(), "<div id=\"root\"></div>")
	})

	t.Run("GET /manifest.webmanifest returns manifest MIME type and no-cache", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/manifest.webmanifest", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Header().Get("Content-Type"), "application/manifest+json")
		assert.Equal(t, "no-cache, no-store, must-revalidate", rec.Header().Get("Cache-Control"))
	})

	t.Run("GET /manifest.json returns manifest MIME type and no-cache", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/manifest.json", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Header().Get("Content-Type"), "application/manifest+json")
		assert.Equal(t, "no-cache, no-store, must-revalidate", rec.Header().Get("Cache-Control"))
	})

	t.Run("GET /sw.js returns javascript MIME type and no-cache", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/sw.js", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Header().Get("Content-Type"), "application/javascript")
		assert.Equal(t, "no-cache, no-store, must-revalidate", rec.Header().Get("Cache-Control"))
	})

	t.Run("GET /favicon.svg returns image/svg+xml MIME type", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/favicon.svg", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Header().Get("Content-Type"), "image/svg+xml")
	})

	t.Run("GET /api/v1/bff/unknown returns 404 JSON, NOT index.html", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/bff/unknown", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusNotFound, rec.Code)
		assert.Contains(t, rec.Header().Get("Content-Type"), "application/json")
		assert.NotContains(t, rec.Body.String(), "<div id=\"root\"></div>")
	})
}

type roundTripFunc func(req *http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestSPAHandler_DevProxy(t *testing.T) {
	proxyURL, err := url.Parse("http://localhost:5173")
	require.NoError(t, err)

	mockTransport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		body := fmt.Sprintf("Vite dev server response for %s", req.URL.Path)
		resp := &http.Response{
			StatusCode: http.StatusOK,
			Status:     "200 OK",
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(body)),
			Request:    req,
		}
		resp.Header.Set("X-Proxied-By", "Vite-Dev-Server")
		return resp, nil
	})

	mockSvc := new(MockBFFService)
	spaConfig := rest.SPAConfig{
		FileSystem:  newTestFileSystem(),
		DevProxyURL: proxyURL,
		Transport:   mockTransport,
	}
	router := rest.SetupRouter(mockSvc, "0.1.0", spaConfig)

	t.Run("proxies non-API route to dev server", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/suites/456", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "Vite-Dev-Server", rec.Header().Get("X-Proxied-By"))
		assert.Contains(t, rec.Body.String(), "Vite dev server response for /suites/456")
	})

	t.Run("does NOT proxy registered API route /healthz", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/healthz", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Empty(t, rec.Header().Get("X-Proxied-By"))
		assert.Contains(t, rec.Body.String(), `"status":"ok"`)
	})
}
