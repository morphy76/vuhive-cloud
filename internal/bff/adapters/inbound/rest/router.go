package rest

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/morphy76/vuhive-cloud/internal/bff/application/ports/inbound"
	"github.com/morphy76/vuhive-cloud/internal/bff/application/ports/outbound"
	"github.com/morphy76/vuhive-cloud/web"
)

// RouterConfig holds options for configuring the BFF HTTP router.
type RouterConfig struct {
	BFFService      inbound.BFFService
	Version         string
	ControlPlaneURL string
	ProxyTransport  http.RoundTripper
	SPAConfig       *SPAConfig
	AuthHandler     *AuthHandler
	SessionService  inbound.SessionService
	OIDCClient      outbound.OIDCClient
	CookieName      string
	AuthEnabled     bool
}

// SetupRouter configures and returns the Gin HTTP engine with BFF routes, middleware, and SPA file serving.
func SetupRouter(bffService inbound.BFFService, version string, spaCfg ...SPAConfig) *gin.Engine {
	return SetupRouterWithProxy(bffService, version, "http://localhost:8080", nil, spaCfg...)
}

// SetupRouterWithProxy configures the Gin HTTP engine with control plane proxying capabilities.
func SetupRouterWithProxy(bffService inbound.BFFService, version string, controlPlaneURL string, proxyTransport http.RoundTripper, spaCfg ...SPAConfig) *gin.Engine {
	var sc *SPAConfig
	if len(spaCfg) > 0 {
		sc = &spaCfg[0]
	}
	return SetupRouterWithConfig(RouterConfig{
		BFFService:      bffService,
		Version:         version,
		ControlPlaneURL: controlPlaneURL,
		ProxyTransport:  proxyTransport,
		SPAConfig:       sc,
	})
}

// SetupRouterWithConfig constructs the Gin HTTP engine based on the comprehensive RouterConfig.
func SetupRouterWithConfig(cfg RouterConfig) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()

	router.Use(LoggingMiddleware())
	router.Use(RecoveryMiddleware())

	handler := NewHandler(cfg.BFFService, cfg.Version)

	// Liveness and health endpoints
	router.GET("/healthz", handler.Healthz)
	router.GET("/version", handler.Version)

	// Session middleware if auth is enabled and session service is present
	var sessionMiddleware gin.HandlerFunc
	if cfg.AuthEnabled && cfg.SessionService != nil {
		sessionMiddleware = SessionMiddleware(cfg.SessionService, cfg.OIDCClient, cfg.CookieName)
	}

	// Primary BFF API endpoints (/api/bff/v1)
	bffV1 := router.Group("/api/bff/v1")
	{
		bffV1.GET("/status", handler.GetStatus)
		bffV1.POST("/sessions", handler.CreateSession)
		bffV1.GET("/sessions/:id", handler.GetSession)
		bffV1.GET("/events", handler.Events)

		if cfg.AuthHandler != nil {
			bffV1.GET("/auth/login", cfg.AuthHandler.Login)
			bffV1.GET("/auth/callback", cfg.AuthHandler.Callback)
			bffV1.POST("/auth/logout", cfg.AuthHandler.Logout)
			bffV1.POST("/auth/backchannel-logout", cfg.AuthHandler.BackchannelLogout)
			if sessionMiddleware != nil {
				bffV1.GET("/auth/me", sessionMiddleware, cfg.AuthHandler.Me)
			} else {
				bffV1.GET("/auth/me", cfg.AuthHandler.Me)
			}
		}

		// Aggregate routes (optionally protected by session middleware)
		if sessionMiddleware != nil {
			bffV1.GET("/dashboard", sessionMiddleware, handler.GetDashboard)
			bffV1.GET("/runs/:id", sessionMiddleware, handler.GetRunDetail)
		} else {
			bffV1.GET("/dashboard", handler.GetDashboard)
			bffV1.GET("/runs/:id", handler.GetRunDetail)
		}

		// Transparent reverse proxy to control plane server for entity CRUD operations
		if cfg.ControlPlaneURL != "" {
			proxyHandler := NewControlPlaneProxy(cfg.ControlPlaneURL, cfg.ProxyTransport)

			var proxyGroup gin.IRoutes = bffV1
			if sessionMiddleware != nil {
				// Apply session middleware so Bearer token is injected before proxying
				proxyGroup = bffV1.Group("", sessionMiddleware)
			}

			// Suites CRUD & builds
			proxyGroup.Any("/suites", proxyHandler)
			proxyGroup.Any("/suites/*path", proxyHandler)

			// Runner profiles CRUD
			proxyGroup.Any("/profiles", proxyHandler)
			proxyGroup.Any("/profiles/*path", proxyHandler)

			// Schedules CRUD
			proxyGroup.Any("/schedules", proxyHandler)
			proxyGroup.Any("/schedules/*path", proxyHandler)

			// Runs trigger, listing, and lifecycle
			proxyGroup.POST("/runs", proxyHandler)
			proxyGroup.GET("/runs", proxyHandler)
			proxyGroup.POST("/runs/:id/abort", proxyHandler)
			proxyGroup.POST("/runs/:id/complete", proxyHandler)
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
		v1.GET("/events", handler.Events)

		if cfg.AuthHandler != nil {
			v1.GET("/auth/login", cfg.AuthHandler.Login)
			v1.GET("/auth/callback", cfg.AuthHandler.Callback)
			v1.POST("/auth/logout", cfg.AuthHandler.Logout)
			v1.POST("/auth/backchannel-logout", cfg.AuthHandler.BackchannelLogout)
			if sessionMiddleware != nil {
				v1.GET("/auth/me", sessionMiddleware, cfg.AuthHandler.Me)
			} else {
				v1.GET("/auth/me", cfg.AuthHandler.Me)
			}
		}
	}

	// SPA & Static file serving
	var spaHandler *SPAHandler
	if cfg.SPAConfig != nil {
		spaHandler = NewSPAHandler(*cfg.SPAConfig)
	} else {
		distFS, _ := web.GetFS()
		spaHandler = NewSPAHandler(SPAConfig{FileSystem: distFS})
	}

	router.GET("/", spaHandler.Serve)
	router.NoRoute(spaHandler.Serve)

	return router
}
