package rest

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/morphy76/vuhive-cloud/api"
	"github.com/morphy76/vuhive-cloud/internal/application/ports/inbound"
	"github.com/morphy76/vuhive-cloud/internal/application/ports/outbound"
	"github.com/morphy76/vuhive-cloud/internal/domain/model"
	"github.com/morphy76/vuhive-cloud/internal/version"
)

// SetupRouter initializes and configures the Gin HTTP engine with routes and middleware.
func SetupRouter(
	buildsUC inbound.BuildsUseCase,
	profilesUC inbound.ProfilesUseCase,
	schedulesUC inbound.SchedulesUseCase,
	runsUC inbound.RunsUseCase,
) *gin.Engine {
	return SetupRouterWithBarrier(buildsUC, profilesUC, schedulesUC, runsUC, nil)
}

// SetupRouterWithBarrier initializes and configures the Gin HTTP engine including optional barrier coordination.
func SetupRouterWithBarrier(
	buildsUC inbound.BuildsUseCase,
	profilesUC inbound.ProfilesUseCase,
	schedulesUC inbound.SchedulesUseCase,
	runsUC inbound.RunsUseCase,
	barrierUC inbound.BarrierUseCase,
) *gin.Engine {
	return SetupRouterWithAll(buildsUC, profilesUC, schedulesUC, runsUC, barrierUC, nil)
}

// SetupRouterWithAll initializes and configures the Gin HTTP engine with all use cases including housekeeping.
func SetupRouterWithAll(
	buildsUC inbound.BuildsUseCase,
	profilesUC inbound.ProfilesUseCase,
	schedulesUC inbound.SchedulesUseCase,
	runsUC inbound.RunsUseCase,
	barrierUC inbound.BarrierUseCase,
	housekeepingUC inbound.HousekeepingUseCase,
) *gin.Engine {
	return SetupRouterWithAuth(buildsUC, profilesUC, schedulesUC, runsUC, barrierUC, housekeepingUC, nil)
}

// SetupRouterWithAuth initializes and configures the Gin HTTP engine with all use cases and optional OIDC JWT auth.
func SetupRouterWithAuth(
	buildsUC inbound.BuildsUseCase,
	profilesUC inbound.ProfilesUseCase,
	schedulesUC inbound.SchedulesUseCase,
	runsUC inbound.RunsUseCase,
	barrierUC inbound.BarrierUseCase,
	housekeepingUC inbound.HousekeepingUseCase,
	tokenVerifier outbound.TokenVerifierPort,
) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()

	router.Use(LoggingMiddleware())
	router.Use(RecoveryMiddleware())
	router.Use(CORSMiddleware())

	// Health and liveness probes
	healthHandler := func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	}
	router.GET("/healthz", healthHandler)
	router.GET("/api/v1/health", healthHandler)

	// Runtime version endpoint
	router.GET("/version", func(c *gin.Context) {
		c.JSON(http.StatusOK, VersionResponse{
			Version:   version.Version,
			Commit:    version.Commit,
			BuildTime: version.BuildTime,
		})
	})

	// Machine-readable OpenAPI 3.1 specifications
	router.GET("/openapi.yaml", func(c *gin.Context) {
		c.Data(http.StatusOK, "application/yaml", api.OpenAPISpecYAML)
	})
	router.GET("/openapi.json", func(c *gin.Context) {
		c.Data(http.StatusOK, "application/json", api.OpenAPISpecJSON)
	})

	// Helper to enforce role guards if auth is configured
	roleGuard := func(roles ...string) gin.HandlerFunc {
		if tokenVerifier == nil {
			return func(c *gin.Context) { c.Next() }
		}
		return RequireRole(roles...)
	}

	// API v1 routes
	v1 := router.Group("/api/v1")
	if tokenVerifier != nil {
		v1.Use(AuthMiddleware(tokenVerifier, false))
	}
	{
		if buildsUC != nil {
			artifactHandler := NewArtifactHandler(buildsUC)
			suites := v1.Group("/suites")
			{
				suites.POST("/:id/builds", roleGuard(model.RoleDeveloper, model.RoleAdmin), artifactHandler.UploadAndBuild)
				suites.GET("/:id/artifacts", roleGuard(model.RoleViewer, model.RoleDeveloper, model.RoleDeployer, model.RoleAdmin), artifactHandler.ListArtifacts)
			}
		}

		if profilesUC != nil {
			profileHandler := NewProfileHandler(profilesUC)
			profiles := v1.Group("/profiles")
			{
				profiles.POST("", roleGuard(model.RoleAdmin), profileHandler.CreateProfile)
				profiles.GET("", roleGuard(model.RoleViewer), profileHandler.ListProfiles)
				profiles.GET("/:id", roleGuard(model.RoleViewer), profileHandler.GetProfile)
				profiles.PUT("/:id", roleGuard(model.RoleAdmin), profileHandler.UpdateProfile)
				profiles.DELETE("/:id", roleGuard(model.RoleAdmin), profileHandler.DeleteProfile)
			}
		}

		if schedulesUC != nil {
			scheduleHandler := NewScheduleHandler(schedulesUC)
			schedules := v1.Group("/schedules")
			{
				schedules.POST("", roleGuard(model.RoleDeployer, model.RoleAdmin), scheduleHandler.CreateSchedule)
				schedules.GET("", roleGuard(model.RoleViewer), scheduleHandler.ListSchedules)
				schedules.GET("/:id", roleGuard(model.RoleViewer), scheduleHandler.GetSchedule)
				schedules.PUT("/:id", roleGuard(model.RoleDeployer, model.RoleAdmin), scheduleHandler.UpdateSchedule)
				schedules.DELETE("/:id", roleGuard(model.RoleDeployer, model.RoleAdmin), scheduleHandler.DeleteSchedule)
			}
		}

		if runsUC != nil {
			runHandler := NewRunHandler(runsUC)
			runs := v1.Group("/runs")
			{
				runs.POST("", roleGuard(model.RoleDeployer, model.RoleAdmin), runHandler.TriggerRun)
				runs.GET("", roleGuard(model.RoleViewer), runHandler.ListRuns)
				runs.GET("/:id", roleGuard(model.RoleViewer), runHandler.GetRun)
				runs.GET("/:id/report", roleGuard(model.RoleViewer), runHandler.GetRunReport)
				runs.GET("/:id/logs", roleGuard(model.RoleViewer), runHandler.GetRunLogs)
				runs.POST("/:id/abort", roleGuard(model.RoleDeployer, model.RoleAdmin), runHandler.AbortRun)
				runs.POST("/:id/complete", roleGuard(model.RoleRunner, model.RoleAdmin), runHandler.CompleteRun)
				runs.POST("/complete", roleGuard(model.RoleRunner, model.RoleAdmin), runHandler.CompleteRun)
			}
		}

		if barrierUC != nil {
			barrierHandler := NewBarrierHandler(barrierUC)
			runs := v1.Group("/runs")
			{
				runs.POST("/:id/barrier/await", roleGuard(model.RoleRunner, model.RoleAdmin), barrierHandler.AwaitBarrier)
				runs.POST("/:id/barrier/abort", roleGuard(model.RoleRunner, model.RoleAdmin), barrierHandler.AbortBarrier)
				runs.GET("/:id/barrier", roleGuard(model.RoleRunner, model.RoleViewer, model.RoleAdmin), barrierHandler.GetBarrier)
			}
		}

		if housekeepingUC != nil {
			hkHandler := NewHousekeepingHandler(housekeepingUC)
			sys := v1.Group("/system")
			{
				sys.POST("/housekeeping", roleGuard(model.RoleAdmin), hkHandler.RunHousekeeping)
				sys.GET("/housekeeping/policy", roleGuard(model.RoleViewer, model.RoleAdmin), hkHandler.GetPolicy)
			}
		}
	}

	return router
}
