package rest

import (
	"github.com/gin-gonic/gin"
	"github.com/morphy76/vuhive-cloud/internal/bff/application/ports/inbound"
	"github.com/morphy76/vuhive-cloud/web"
)

// SetupRouter configures and returns the Gin HTTP engine with BFF routes, middleware, and SPA file serving.
func SetupRouter(bffService inbound.BFFService, version string, spaCfg ...SPAConfig) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()

	router.Use(LoggingMiddleware())
	router.Use(RecoveryMiddleware())

	handler := NewHandler(bffService, version)

	// Liveness and health endpoints
	router.GET("/healthz", handler.Healthz)
	router.GET("/version", handler.Version)

	// BFF API endpoints
	v1 := router.Group("/api/v1/bff")
	{
		v1.GET("/status", handler.GetStatus)
		v1.POST("/sessions", handler.CreateSession)
		v1.GET("/sessions/:id", handler.GetSession)
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
