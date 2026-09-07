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

// RouterConfig encapsulates all use cases and dependencies for the control plane router.
type RouterConfig struct {
	BuildsUC       inbound.BuildsUseCase
	ProfilesUC     inbound.ProfilesUseCase
	SchedulesUC    inbound.SchedulesUseCase
	RunsUC         inbound.RunsUseCase
	BarrierUC      inbound.BarrierUseCase
	HousekeepingUC inbound.HousekeepingUseCase
	SuitesUC       inbound.SuitesUseCase
	ConfigsUC      inbound.ConfigsUseCase
	TokenVerifier  outbound.TokenVerifierPort
}

// SetupRouter initializes and configures the Gin HTTP engine with routes and middleware.
func SetupRouter(
	buildsUC inbound.BuildsUseCase,
	profilesUC inbound.ProfilesUseCase,
	schedulesUC inbound.SchedulesUseCase,
	runsUC inbound.RunsUseCase,
) *gin.Engine {
	return SetupRouterWithConfig(RouterConfig{
		BuildsUC:    buildsUC,
		ProfilesUC:  profilesUC,
		SchedulesUC: schedulesUC,
		RunsUC:      runsUC,
	})
}

// SetupRouterWithBarrier initializes and configures the Gin HTTP engine including optional barrier coordination.
func SetupRouterWithBarrier(
	buildsUC inbound.BuildsUseCase,
	profilesUC inbound.ProfilesUseCase,
	schedulesUC inbound.SchedulesUseCase,
	runsUC inbound.RunsUseCase,
	barrierUC inbound.BarrierUseCase,
) *gin.Engine {
	return SetupRouterWithConfig(RouterConfig{
		BuildsUC:    buildsUC,
		ProfilesUC:  profilesUC,
		SchedulesUC: schedulesUC,
		RunsUC:      runsUC,
		BarrierUC:   barrierUC,
	})
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
	return SetupRouterWithConfig(RouterConfig{
		BuildsUC:       buildsUC,
		ProfilesUC:     profilesUC,
		SchedulesUC:    schedulesUC,
		RunsUC:         runsUC,
		BarrierUC:      barrierUC,
		HousekeepingUC: housekeepingUC,
	})
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
	return SetupRouterWithConfig(RouterConfig{
		BuildsUC:       buildsUC,
		ProfilesUC:     profilesUC,
		SchedulesUC:    schedulesUC,
		RunsUC:         runsUC,
		BarrierUC:      barrierUC,
		HousekeepingUC: housekeepingUC,
		TokenVerifier:  tokenVerifier,
	})
}

// SetupRouterWithConfig initializes and configures the Gin HTTP engine based on RouterConfig.
func SetupRouterWithConfig(cfg RouterConfig) *gin.Engine {
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
		if cfg.TokenVerifier == nil {
			return func(c *gin.Context) { c.Next() }
		}
		return RequireRole(roles...)
	}

	// API v1 routes
	v1 := router.Group("/api/v1")
	if cfg.TokenVerifier != nil {
		v1.Use(AuthMiddleware(cfg.TokenVerifier, false))
	}
	{
		if cfg.SuitesUC != nil || cfg.ConfigsUC != nil || cfg.BuildsUC != nil {
			suites := v1.Group("/suites")
			{
				if cfg.SuitesUC != nil {
					suiteHandler := NewSuiteHandler(cfg.SuitesUC)
					suites.POST("", roleGuard(model.RoleDeveloper, model.RoleAdmin), suiteHandler.CreateSuite)
					suites.GET("", roleGuard(model.RoleViewer), suiteHandler.ListSuites)
					suites.GET("/:id", roleGuard(model.RoleViewer), suiteHandler.GetSuite)
					suites.PUT("/:id", roleGuard(model.RoleDeveloper, model.RoleAdmin), suiteHandler.UpdateSuite)
					suites.DELETE("/:id", roleGuard(model.RoleDeveloper, model.RoleAdmin), suiteHandler.DeleteSuite)
				}

				if cfg.ConfigsUC != nil {
					configHandler := NewConfigHandler(cfg.ConfigsUC)
					suites.POST("/:id/configs", roleGuard(model.RoleDeveloper, model.RoleAdmin), configHandler.CreateConfig)
					suites.GET("/:id/configs", roleGuard(model.RoleViewer), configHandler.ListConfigs)
					suites.GET("/:id/configs/:configId", roleGuard(model.RoleViewer), configHandler.GetConfig)
					suites.DELETE("/:id/configs/:configId", roleGuard(model.RoleDeveloper, model.RoleAdmin), configHandler.DeleteConfig)
				}

				if cfg.BuildsUC != nil {
					artifactHandler := NewArtifactHandler(cfg.BuildsUC)
					suites.POST("/:id/builds", roleGuard(model.RoleDeveloper, model.RoleAdmin), artifactHandler.UploadAndBuild)
					suites.GET("/:id/artifacts", roleGuard(model.RoleViewer, model.RoleDeveloper, model.RoleDeployer, model.RoleAdmin), artifactHandler.ListArtifacts)
				}
			}
		}

		if cfg.ProfilesUC != nil {
			profileHandler := NewProfileHandler(cfg.ProfilesUC)
			profiles := v1.Group("/profiles")
			{
				profiles.POST("", roleGuard(model.RoleAdmin), profileHandler.CreateProfile)
				profiles.GET("", roleGuard(model.RoleViewer), profileHandler.ListProfiles)
				profiles.GET("/:id", roleGuard(model.RoleViewer), profileHandler.GetProfile)
				profiles.PUT("/:id", roleGuard(model.RoleAdmin), profileHandler.UpdateProfile)
				profiles.DELETE("/:id", roleGuard(model.RoleAdmin), profileHandler.DeleteProfile)
			}
		}

		if cfg.SchedulesUC != nil {
			scheduleHandler := NewScheduleHandler(cfg.SchedulesUC)
			schedules := v1.Group("/schedules")
			{
				schedules.POST("", roleGuard(model.RoleDeployer, model.RoleAdmin), scheduleHandler.CreateSchedule)
				schedules.GET("", roleGuard(model.RoleViewer), scheduleHandler.ListSchedules)
				schedules.GET("/:id", roleGuard(model.RoleViewer), scheduleHandler.GetSchedule)
				schedules.PUT("/:id", roleGuard(model.RoleDeployer, model.RoleAdmin), scheduleHandler.UpdateSchedule)
				schedules.DELETE("/:id", roleGuard(model.RoleDeployer, model.RoleAdmin), scheduleHandler.DeleteSchedule)
			}
		}

		if cfg.RunsUC != nil {
			runHandler := NewRunHandler(cfg.RunsUC)
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

		if cfg.BarrierUC != nil {
			barrierHandler := NewBarrierHandler(cfg.BarrierUC)
			runs := v1.Group("/runs")
			{
				runs.POST("/:id/barrier/await", roleGuard(model.RoleRunner, model.RoleAdmin), barrierHandler.AwaitBarrier)
				runs.POST("/:id/barrier/abort", roleGuard(model.RoleRunner, model.RoleAdmin), barrierHandler.AbortBarrier)
				runs.GET("/:id/barrier", roleGuard(model.RoleRunner, model.RoleViewer, model.RoleAdmin), barrierHandler.GetBarrier)
			}
		}

		if cfg.HousekeepingUC != nil {
			hkHandler := NewHousekeepingHandler(cfg.HousekeepingUC)
			sys := v1.Group("/system")
			{
				sys.POST("/housekeeping", roleGuard(model.RoleAdmin), hkHandler.RunHousekeeping)
				sys.GET("/housekeeping/policy", roleGuard(model.RoleViewer, model.RoleAdmin), hkHandler.GetPolicy)
			}
		}
	}

	return router
}
