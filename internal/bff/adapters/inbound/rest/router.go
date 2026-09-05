package rest

import (
	"github.com/gin-gonic/gin"
	"github.com/morphy76/vuhive-cloud/internal/bff/application/ports/inbound"
)

// SetupRouter configures and returns the Gin HTTP engine with BFF routes and middleware.
func SetupRouter(bffService inbound.BFFService, version string) *gin.Engine {
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

	return router
}
