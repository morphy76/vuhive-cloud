package rest

import (
	"io"
	"io/fs"
	"mime"
	"net/http"
	"net/http/httputil"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

// Static compile-time interface assertion
var _ http.Handler = (*SPAHandler)(nil)

// SPAConfig holds the configuration for serving the single-page application.
type SPAConfig struct {
	// FileSystem is the filesystem containing the static production bundle (e.g. web.DistFS).
	FileSystem fs.FS
	// DevProxyURL is an optional upstream URL (e.g. Vite dev server) to proxy requests to during local development.
	DevProxyURL *url.URL
	// Transport is an optional http.RoundTripper for the reverse proxy.
	Transport http.RoundTripper
}

// SPAHandler serves SPA static assets, handles fallback routing to index.html,
// applies optimal HTTP caching headers, and optionally reverse-proxies to a dev server.
type SPAHandler struct {
	fileSystem fs.FS
	devProxy   *httputil.ReverseProxy
}

// NewSPAHandler creates a new SPAHandler instance.
func NewSPAHandler(cfg SPAConfig) *SPAHandler {
	h := &SPAHandler{
		fileSystem: cfg.FileSystem,
	}
	if cfg.DevProxyURL != nil {
		h.devProxy = httputil.NewSingleHostReverseProxy(cfg.DevProxyURL)
		if cfg.Transport != nil {
			h.devProxy.Transport = cfg.Transport
		}
	}
	return h
}

// proxyResponseWriter wraps http.ResponseWriter without forwarding http.CloseNotifier,
// preventing runtime type assertion panics when running against gin.ResponseWriter or httptest.ResponseRecorder.
type proxyResponseWriter struct {
	http.ResponseWriter
}

// Serve handles a Gin context by invoking ServeHTTP.
func (h *SPAHandler) Serve(c *gin.Context) {
	h.ServeHTTP(c.Writer, c.Request)
}

// ServeHTTP satisfies the standard http.Handler interface.
func (h *SPAHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	ctx := r.Context()
	log := zerolog.Ctx(ctx).With().Str("op", "SPAHandler.ServeHTTP").Str("path", r.URL.Path).Logger()
	log.Debug().Msg("serving SPA request")

	// 1. If dev reverse proxy is configured, delegate non-API requests to the dev server
	if h.devProxy != nil {
		h.devProxy.ServeHTTP(proxyResponseWriter{ResponseWriter: w}, r)
		log.Debug().Dur("duration_ms", time.Since(start)).Msg("proxied request to dev server")
		return
	}

	if h.fileSystem == nil {
		http.NotFound(w, r)
		return
	}

	cleanPath := strings.TrimPrefix(filepath.Clean(r.URL.Path), "/")

	// 2. Reject unmatched API routes from falling back to index.html
	if strings.HasPrefix(cleanPath, "api/") || cleanPath == "api" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":"endpoint not found"}`))
		return
	}

	// 3. Direct request to root index
	if cleanPath == "" || cleanPath == "." || cleanPath == "index.html" {
		h.serveIndex(w)
		log.Debug().Dur("duration_ms", time.Since(start)).Msg("served index.html")
		return
	}

	// 4. Try opening the requested file in the static filesystem
	f, err := h.fileSystem.Open(cleanPath)
	if err == nil {
		defer func() { _ = f.Close() }()
		stat, err := f.Stat()
		if err == nil && !stat.IsDir() {
			h.serveStaticFile(w, cleanPath, f)
			log.Debug().Dur("duration_ms", time.Since(start)).Msg("served static asset")
			return
		}
	}

	// 5. If requesting an asset under /assets/ that does not exist, return 404 (do NOT fallback to index.html)
	if strings.HasPrefix(cleanPath, "assets/") {
		http.NotFound(w, r)
		return
	}

	// 6. SPA fallback: Route all other non-API routes (e.g. /suites/*, /runs/*) to index.html
	h.serveIndex(w)
	log.Debug().Dur("duration_ms", time.Since(start)).Msg("served SPA fallback index.html")
}

func (h *SPAHandler) serveIndex(w http.ResponseWriter) {
	indexFile, err := h.fileSystem.Open("index.html")
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte("index.html not found"))
		return
	}
	defer func() { _ = indexFile.Close() }()

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.WriteHeader(http.StatusOK)
	_, _ = io.Copy(w, indexFile)
}

func (h *SPAHandler) serveStaticFile(w http.ResponseWriter, path string, f fs.File) {
	contentType := detectMIMEType(path)
	if contentType != "" {
		w.Header().Set("Content-Type", contentType)
	}

	// Apply optimal HTTP caching headers
	if strings.HasPrefix(path, "assets/") {
		// Hashed production assets: cache immutably for 1 year
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	} else if path == "sw.js" || strings.HasSuffix(path, "service-worker.js") {
		// Service worker must never be cached aggressively
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	} else if path == "manifest.webmanifest" || path == "manifest.json" {
		// Web app manifest
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	}

	w.WriteHeader(http.StatusOK)
	_, _ = io.Copy(w, f)
}

func detectMIMEType(path string) string {
	ext := filepath.Ext(path)
	switch ext {
	case ".js", ".mjs":
		return "application/javascript"
	case ".css":
		return "text/css"
	case ".html":
		return "text/html; charset=utf-8"
	case ".svg":
		return "image/svg+xml"
	case ".json":
		if strings.Contains(path, "manifest") {
			return "application/manifest+json"
		}
		return "application/json"
	case ".webmanifest":
		return "application/manifest+json"
	default:
		return mime.TypeByExtension(ext)
	}
}
