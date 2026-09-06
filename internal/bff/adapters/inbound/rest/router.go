package rest

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/morphy76/vuhive-cloud/internal/bff/application/ports/inbound"
	"github.com/morphy76/vuhive-cloud/web"
)

// SetupRouter configures and returns the Gin HTTP engine with BFF routes, middleware, and SPA file serving.
func SetupRouter(bffService inbound.BFFService, version string, spaCfg ...SPAConfig) *gin.Engine {
	return SetupRouterWithProxy(bffService, version, "http://localhost:8080", nil, spaCfg...)
}

// SetupRouterWithProxy configures the Gin HTTP engine with control plane proxying capabilities.
func SetupRouterWithProxy(bffService inbound.BFFService, version string, controlPlaneURL string, proxyTransport http.RoundTripper, spaCfg ...SPAConfig) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()

	router.Use(LoggingMiddleware())
	router.Use(RecoveryMiddleware())

	handler := NewHandler(bffService, version)

	// Liveness and health endpoints
	router.GET("/healthz", handler.Healthz)
	router.GET("/version", handler.Version)

	// Primary BFF API endpoints (/api/bff/v1) per Issue #59
	bffV1 := router.Group("/api/bff/v1")
	{
		bffV1.GET("/dashboard", handler.GetDashboard)
		bffV1.GET("/runs/:id", handler.GetRunDetail)
		bffV1.GET("/status", handler.GetStatus)
		bffV1.POST("/sessions", handler.CreateSession)
		bffV1.GET("/sessions/:id", handler.GetSession)

		// Transparent reverse proxy to control plane server for entity CRUD operations
		if controlPlaneURL != "" {
			proxyHandler := NewControlPlaneProxy(controlPlaneURL, proxyTransport)

			// Suites CRUD & builds
			bffV1.Any("/suites", proxyHandler)
			bffV1.Any("/suites/*path", proxyHandler)

			// Runner profiles CRUD
			bffV1.Any("/profiles", proxyHandler)
			bffV1.Any("/profiles/*path", proxyHandler)

			// Schedules CRUD
			bffV1.Any("/schedules", proxyHandler)
			bffV1.Any("/schedules/*path", proxyHandler)

			// Runs trigger, listing, and lifecycle
			bffV1.POST("/runs", proxyHandler)
			bffV1.GET("/runs", proxyHandler)
			bffV1.POST("/runs/:id/abort", proxyHandler)
			bffV1.POST("/runs/:id/complete", proxyHandler)
		}
	}

	// Backwards-compatible /api/v1/bff endpoints
	v1 := router.Group("/api/v1/bff")
	{
		v1.GET("/status", handler.GetStatus)
		v1.POST("/sessions", handler.CreateSession)
		v1.GET("/sessions/:id", handler.GetSession)
		v1.GET("/dashboard", handler.GetDashboard)
		v1.GET("/runs/:id", handler.GetRunDetail)
	}

	// SPA & Static file serving
	var spaHandler *SPAHandler
	if len(spaCfg) > 0 {
		spaHandler = NewSPAHandler(spaCfg[0])
	} else {
		distFS, _ := web.GetFS()
		spaHandler = NewSPAHandler(SPAConfig{FileSystem: distFS})
	}

	router.GET("/", spaHandler.Serve)
	router.NoRoute(spaHandler.Serve)

	return router
}
