package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/morphy76/vuhive-cloud/internal/adapters/inbound/rest"
	k8sadapter "github.com/morphy76/vuhive-cloud/internal/adapters/outbound/k8s"
	pgadapter "github.com/morphy76/vuhive-cloud/internal/adapters/outbound/postgres"
	s3adapter "github.com/morphy76/vuhive-cloud/internal/adapters/outbound/s3"
	"github.com/morphy76/vuhive-cloud/internal/application/ports/outbound"
	"github.com/morphy76/vuhive-cloud/internal/application/service"
	"github.com/morphy76/vuhive-cloud/internal/version"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"k8s.io/client-go/kubernetes"
	k8srest "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {
	showVersion := flag.Bool("version", false, "Print version information and exit")
	portFlag := flag.String("port", "", "Server HTTP port (defaults to PORT env or 8080)")
	flag.Parse()

	if *showVersion {
		fmt.Printf("vuhive-cloud server %s (commit: %s, built: %s)\n", version.Version, version.Commit, version.BuildTime)
		os.Exit(0)
	}

	// Configure structured logging with zerolog
	zerolog.TimeFieldFormat = time.RFC3339
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339})

	port := *portFlag
	if port == "" {
		port = os.Getenv("PORT")
	}
	if port == "" {
		port = "8080"
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	log.Info().
		Str("version", version.Version).
		Str("commit", version.Commit).
		Str("build_time", version.BuildTime).
		Str("port", port).
		Msg("starting vuhive-cloud control plane server")

	// Outbound dependencies initialization
	var suiteRepo outbound.TestSuiteRepository
	var artifactRepo outbound.ArtifactRepository
	var configRepo outbound.ConfigurationRepository
	var profileRepo outbound.RunnerProfileRepository
	var runRepo outbound.TestRunRepository
	var scheduleRepo outbound.ScheduleRepository
	var storageAdapter outbound.StoragePort
	var buildOrchestrator outbound.BuildOrchestratorPort
	var runnerOrchestrator outbound.RunnerOrchestratorPort
	var scheduleOrchestrator outbound.ScheduleOrchestratorPort

	// 1. PostgreSQL Database
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = os.Getenv("POSTGRES_URL")
	}
	if dbURL != "" {
		poolConfig, err := pgxpool.ParseConfig(dbURL)
		if err != nil {
			log.Warn().Err(err).Msg("failed parsing database url, running without postgres")
		} else {
			pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
			if err != nil {
				log.Warn().Err(err).Msg("failed connecting to postgres, running without postgres")
			} else {
				defer pool.Close()
				suiteRepo = pgadapter.NewTestSuiteRepository(pool)
				artifactRepo = pgadapter.NewArtifactRepository(pool)
				configRepo = pgadapter.NewConfigurationRepository(pool)
				profileRepo = pgadapter.NewRunnerProfileRepository(pool)
				runRepo = pgadapter.NewTestRunRepository(pool)
				scheduleRepo = pgadapter.NewScheduleRepository(pool)
				log.Info().Msg("connected to postgresql repository")
			}
		}
	} else {
		log.Warn().Msg("DATABASE_URL not set; database repositories unavailable")
	}

	// 2. S3 / MinIO Storage
	s3Bucket := os.Getenv("S3_BUCKET")
	if s3Bucket != "" {
		accessKey := os.Getenv("S3_ACCESS_KEY_ID")
		if accessKey == "" {
			accessKey = os.Getenv("AWS_ACCESS_KEY_ID")
		}
		secretKey := os.Getenv("S3_SECRET_ACCESS_KEY")
		if secretKey == "" {
			secretKey = os.Getenv("AWS_SECRET_ACCESS_KEY")
		}
		s3Cfg := s3adapter.Config{
			Endpoint:        os.Getenv("S3_ENDPOINT"),
			Region:          os.Getenv("S3_REGION"),
			Bucket:          s3Bucket,
			AccessKeyID:     accessKey,
			SecretAccessKey: secretKey,
		}
		s3Client, err := s3adapter.NewAdapter(ctx, s3Cfg)
		if err != nil {
			log.Warn().Err(err).Msg("failed initializing s3 adapter")
		} else {
			storageAdapter = s3Client
			log.Info().Str("bucket", s3Bucket).Msg("connected to s3 storage adapter")
		}
	} else {
		log.Warn().Msg("S3_BUCKET not set; s3 storage adapter unavailable")
	}

	// 3. Kubernetes Orchestrator
	k8sConfig, err := k8srest.InClusterConfig()
	if err != nil {
		kubeconfigPath := os.Getenv("KUBECONFIG")
		if kubeconfigPath == "" {
			kubeconfigPath = clientcmd.RecommendedHomeFile
		}
		k8sConfig, err = clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	}
	if err == nil && k8sConfig != nil {
		k8sClientset, err := kubernetes.NewForConfig(k8sConfig)
		if err != nil {
			log.Warn().Err(err).Msg("failed creating kubernetes clientset")
		} else {
			k8sCfg := k8sadapter.DefaultConfig()
			if runnerNs := os.Getenv("RUNNER_NAMESPACE"); runnerNs != "" {
				k8sCfg.RunnerNamespace = runnerNs
			}
			if builderNs := os.Getenv("BUILDER_NAMESPACE"); builderNs != "" {
				k8sCfg.Namespace = builderNs
			} else if podNs := os.Getenv("POD_NAMESPACE"); podNs != "" {
				k8sCfg.Namespace = podNs
			}
			if builderImg := os.Getenv("BUILDER_IMAGE"); builderImg != "" {
				k8sCfg.BuilderImage = builderImg
			}
			if runnerInitImg := os.Getenv("RUNNER_INIT_IMAGE"); runnerInitImg != "" {
				k8sCfg.RunnerInitImage = runnerInitImg
			}
			if runnerDefaultImg := os.Getenv("RUNNER_DEFAULT_IMAGE"); runnerDefaultImg != "" {
				k8sCfg.RunnerDefaultImage = runnerDefaultImg
			}
			if s3Bucket != "" {
				k8sCfg.S3Endpoint = os.Getenv("S3_ENDPOINT")
				k8sCfg.S3Region = os.Getenv("S3_REGION")
				k8sCfg.S3Bucket = s3Bucket
				k8sCfg.S3AccessKeyID = os.Getenv("S3_ACCESS_KEY_ID")
				if k8sCfg.S3AccessKeyID == "" {
					k8sCfg.S3AccessKeyID = os.Getenv("AWS_ACCESS_KEY_ID")
				}
				k8sCfg.S3SecretAccessKey = os.Getenv("S3_SECRET_ACCESS_KEY")
				if k8sCfg.S3SecretAccessKey == "" {
					k8sCfg.S3SecretAccessKey = os.Getenv("AWS_SECRET_ACCESS_KEY")
				}
				k8sCfg.S3UsePathStyle = os.Getenv("S3_USE_PATH_STYLE") == "true" || k8sCfg.S3Endpoint != ""
			}
			k8sCfg.APICallbackURL = os.Getenv("API_CALLBACK_URL")

			buildOrchestrator = k8sadapter.NewBuildOrchestrator(k8sClientset, k8sCfg)
			runnerOrchestrator = k8sadapter.NewRunnerOrchestrator(k8sClientset, k8sCfg)
			scheduleOrchestrator = k8sadapter.NewScheduleOrchestrator(k8sClientset, k8sCfg)
			log.Info().Msg("connected to kubernetes orchestrator")

			if runRepo != nil {
				runnerWatcher := k8sadapter.NewRunnerJobWatcher(k8sClientset, runRepo, scheduleRepo, k8sCfg)
				go func() {
					if err := runnerWatcher.Start(ctx); err != nil && !errors.Is(err, context.Canceled) {
						log.Error().Err(err).Msg("runner job informer watcher failed")
					}
				}()
			}
		}
	} else {
		log.Warn().Msg("kubernetes cluster configuration not found; orchestrator unavailable")
	}

	// Application services wiring
	buildService := service.NewBuildService(suiteRepo, artifactRepo, storageAdapter, buildOrchestrator)
	profileService := service.NewProfileService(profileRepo)
	runService := service.NewRunService(suiteRepo, artifactRepo, configRepo, profileRepo, runRepo, runnerOrchestrator)
	_ = runService
	scheduleService := service.NewScheduleService(suiteRepo, artifactRepo, configRepo, profileRepo, scheduleRepo, scheduleOrchestrator)

	// Router setup
	router := rest.SetupRouter(buildService, profileService, scheduleService)

	server := &http.Server{
		Addr:         ":" + port,
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	// Listen for OS interrupt and termination signals for graceful teardown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Info().Str("addr", server.Addr).Msg("listening for incoming HTTP requests")
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal().Err(err).Msg("failed starting http server")
		}
	}()

	sig := <-sigChan
	log.Info().Str("signal", sig.String()).Msg("received shutdown signal, shutting down gracefully")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Error().Err(err).Msg("server graceful shutdown encountered error")
	} else {
		log.Info().Msg("server gracefully stopped")
	}
}
