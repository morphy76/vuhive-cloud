package rest

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/morphy76/vuhive-cloud/internal/application/ports/inbound"
)

// SetupRouter initializes and configures the Gin HTTP engine with routes and middleware.
func SetupRouter(buildsUC inbound.BuildsUseCase) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()

	router.Use(LoggingMiddleware())
	router.Use(RecoveryMiddleware())

	// Health and liveness probes
	healthHandler := func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	}
	router.GET("/healthz", healthHandler)
	router.GET("/api/v1/health", healthHandler)

	// API v1 routes
	v1 := router.Group("/api/v1")
	{
		artifactHandler := NewArtifactHandler(buildsUC)

		suites := v1.Group("/suites")
		{
			suites.POST("/:id/builds", artifactHandler.UploadAndBuild)
			suites.GET("/:id/artifacts", artifactHandler.ListArtifacts)
		}
	}

	return router
}
