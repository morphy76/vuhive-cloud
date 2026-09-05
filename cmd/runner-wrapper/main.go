package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	s3adapter "github.com/morphy76/vuhive-cloud/internal/adapters/outbound/s3"
	"github.com/morphy76/vuhive-cloud/internal/runner"
	"github.com/morphy76/vuhive-cloud/internal/version"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	showVersion := flag.Bool("version", false, "Print version information and exit")
	runnerPathFlag := flag.String("runner-path", "", "Path to compiled runner executable (defaults to RUNNER_PATH or /shared/runner)")
	configPathFlag := flag.String("config-path", "", "Path to vuhive.yaml (defaults to CONFIG_PATH or /shared/vuhive.yaml)")
	summaryPathFlag := flag.String("summary-path", "", "Path to summary.json (defaults to SUMMARY_PATH or /shared/summary.json)")
	logPathFlag := flag.String("log-path", "", "Path to run.log (defaults to LOG_PATH or /shared/run.log)")
	runIDFlag := flag.String("run-id", "", "Test run UUID (defaults to VUHIVE_RUN_ID or RUN_ID)")
	reportKeyFlag := flag.String("report-key", "", "S3 key to upload summary.json (defaults to S3_REPORT_KEY)")
	logsKeyFlag := flag.String("logs-key", "", "S3 key to upload run.log (defaults to S3_LOGS_KEY)")
	callbackURLFlag := flag.String("callback-url", "", "API callback endpoint to notify upon run completion (defaults to API_CALLBACK_URL)")
	workerIDFlag := flag.String("worker-id", "", "Worker participant identifier (defaults to VUHIVE_WORKER_ID or HOSTNAME)")
	workerCountFlag := flag.Int("worker-count", 0, "Total number of participating worker pods (defaults to VUHIVE_WORKER_COUNT)")
	barrierEnabledFlag := flag.Bool("barrier-enabled", false, "Enable distributed start barrier (defaults to VUHIVE_BARRIER_ENABLED)")
	barrierTimeoutFlag := flag.Duration("barrier-timeout", 0, "Distributed barrier rendezvous timeout (defaults to VUHIVE_BARRIER_TIMEOUT)")
	releaseDelayFlag := flag.Duration("release-delay", 0, "Rendezvous synchronized start delay (defaults to VUHIVE_RELEASE_DELAY)")
	coordinatorURLFlag := flag.String("coordinator-url", "", "Coordinator barrier base URL (defaults to VUHIVE_COORDINATOR_URL)")
	flag.Parse()

	if *showVersion {
		fmt.Printf("vuhive-runner-wrapper %s (commit: %s, built: %s)\n", version.Version, version.Commit, version.BuildTime)
		os.Exit(0)
	}

	zerolog.TimeFieldFormat = time.RFC3339
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339})

	ctx := log.Logger.WithContext(context.Background())

	runnerPath := *runnerPathFlag
	if runnerPath == "" {
		runnerPath = os.Getenv("RUNNER_PATH")
	}
	if runnerPath == "" {
		runnerPath = runner.DefaultRunnerPath
	}

	configPath := *configPathFlag
	if configPath == "" {
		configPath = os.Getenv("CONFIG_PATH")
	}
	if configPath == "" {
		configPath = runner.DefaultConfigPath
	}

	summaryPath := *summaryPathFlag
	if summaryPath == "" {
		summaryPath = os.Getenv("SUMMARY_PATH")
	}
	if summaryPath == "" {
		summaryPath = runner.DefaultSummaryPath
	}

	logPath := *logPathFlag
	if logPath == "" {
		logPath = os.Getenv("LOG_PATH")
	}
	if logPath == "" {
		logPath = runner.DefaultLogPath
	}

	runID := *runIDFlag
	if runID == "" {
		runID = os.Getenv("VUHIVE_RUN_ID")
	}
	if runID == "" {
		runID = os.Getenv("RUN_ID")
	}

	reportKey := *reportKeyFlag
	if reportKey == "" {
		reportKey = os.Getenv("S3_REPORT_KEY")
	}

	logsKey := *logsKeyFlag
	if logsKey == "" {
		logsKey = os.Getenv("S3_LOGS_KEY")
	}

	callbackURL := *callbackURLFlag
	if callbackURL == "" {
		callbackURL = os.Getenv("API_CALLBACK_URL")
	}
	if callbackURL == "" {
		callbackURL = os.Getenv("VUHIVE_API_URL")
	}

	workerID := *workerIDFlag
	if workerID == "" {
		workerID = os.Getenv("VUHIVE_WORKER_ID")
	}
	if workerID == "" {
		workerID = os.Getenv("HOSTNAME")
	}

	workerCount := *workerCountFlag
	if workerCount <= 0 {
		if envCount := os.Getenv("VUHIVE_WORKER_COUNT"); envCount != "" {
			if parsed, err := strconv.Atoi(envCount); err == nil && parsed > 0 {
				workerCount = parsed
			}
		}
	}
	if workerCount <= 0 {
		workerCount = 1
	}

	barrierEnabled := *barrierEnabledFlag
	if !barrierEnabled {
		barrierEnabled = os.Getenv("VUHIVE_BARRIER_ENABLED") == "true"
	}
	if workerCount > 1 {
		barrierEnabled = true
	}

	barrierTimeout := *barrierTimeoutFlag
	if barrierTimeout <= 0 {
		if envTimeout := os.Getenv("VUHIVE_BARRIER_TIMEOUT"); envTimeout != "" {
			if parsed, err := time.ParseDuration(envTimeout); err == nil && parsed > 0 {
				barrierTimeout = parsed
			}
		}
	}

	releaseDelay := *releaseDelayFlag
	if releaseDelay <= 0 {
		if envDelay := os.Getenv("VUHIVE_RELEASE_DELAY"); envDelay != "" {
			if parsed, err := time.ParseDuration(envDelay); err == nil && parsed > 0 {
				releaseDelay = parsed
			}
		}
	}

	coordinatorURL := *coordinatorURLFlag
	if coordinatorURL == "" {
		coordinatorURL = os.Getenv("VUHIVE_COORDINATOR_URL")
	}

	// S3 storage configuration
	s3Bucket := os.Getenv("S3_BUCKET")
	accessKey := os.Getenv("S3_ACCESS_KEY_ID")
	if accessKey == "" {
		accessKey = os.Getenv("AWS_ACCESS_KEY_ID")
	}
	secretKey := os.Getenv("S3_SECRET_ACCESS_KEY")
	if secretKey == "" {
		secretKey = os.Getenv("AWS_SECRET_ACCESS_KEY")
	}
	endpoint := os.Getenv("S3_ENDPOINT")
	region := os.Getenv("S3_REGION")
	usePathStyle := os.Getenv("S3_USE_PATH_STYLE") == "true" || endpoint != ""

	s3Cfg := s3adapter.Config{
		Endpoint:        endpoint,
		Region:          region,
		Bucket:          s3Bucket,
		AccessKeyID:     accessKey,
		SecretAccessKey: secretKey,
		UsePathStyle:    usePathStyle,
	}

	s3Client, err := s3adapter.NewAdapter(ctx, s3Cfg)
	if err != nil {
		log.Fatal().Err(err).Msg("failed initializing S3 storage adapter")
	}

	wrapperCfg := runner.WrapperConfig{
		RunnerPath:     strings.TrimSpace(runnerPath),
		ConfigPath:     strings.TrimSpace(configPath),
		SummaryPath:    strings.TrimSpace(summaryPath),
		LogPath:        strings.TrimSpace(logPath),
		RunID:          strings.TrimSpace(runID),
		ReportKey:      strings.TrimSpace(reportKey),
		LogsKey:        strings.TrimSpace(logsKey),
		APICallbackURL: strings.TrimSpace(callbackURL),
		WorkerID:       strings.TrimSpace(workerID),
		WorkerCount:    workerCount,
		BarrierEnabled: barrierEnabled,
		BarrierTimeout: barrierTimeout,
		ReleaseDelay:   releaseDelay,
		CoordinatorURL: strings.TrimSpace(coordinatorURL),
		S3Config:       s3Cfg,
	}

	wrapper := runner.NewRunnerWrapper(s3Client)
	extraArgs := flag.Args()

	exitCode, err := wrapper.Run(ctx, wrapperCfg, extraArgs)
	if err != nil {
		log.Error().Err(err).Int("exit_code", exitCode).Msg("runner-wrapper encountered execution error")
	}

	os.Exit(exitCode)
}
