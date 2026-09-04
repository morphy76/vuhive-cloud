package main

import (
	"context"
	"flag"
	"fmt"
	"os"
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
	sharedDirFlag := flag.String("shared-dir", "", "Path to shared emptyDir directory (defaults to SHARED_DIR or /shared)")
	binaryKeyFlag := flag.String("binary-key", "", "S3 key for target runner binary (defaults to S3_BINARY_KEY)")
	configKeyFlag := flag.String("config-key", "", "S3 key for target vuhive.yaml (defaults to S3_CONFIG_KEY)")
	wrapperSrcFlag := flag.String("wrapper-src", "", "Source path for runner-wrapper to copy to shared directory")
	entrypointSrcFlag := flag.String("entrypoint-src", "", "Source path for entrypoint.sh to copy to shared directory")
	flag.Parse()

	if *showVersion {
		fmt.Printf("vuhive-runner-init %s (commit: %s, built: %s)\n", version.Version, version.Commit, version.BuildTime)
		os.Exit(0)
	}

	zerolog.TimeFieldFormat = time.RFC3339
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339})

	ctx := log.Logger.WithContext(context.Background())

	sharedDir := *sharedDirFlag
	if sharedDir == "" {
		sharedDir = os.Getenv("SHARED_DIR")
	}
	if sharedDir == "" {
		sharedDir = runner.DefaultSharedDir
	}

	binaryKey := *binaryKeyFlag
	if binaryKey == "" {
		binaryKey = os.Getenv("S3_BINARY_KEY")
	}

	configKey := *configKeyFlag
	if configKey == "" {
		configKey = os.Getenv("S3_CONFIG_KEY")
	}

	wrapperSrc := *wrapperSrcFlag
	if wrapperSrc == "" {
		wrapperSrc = os.Getenv("WRAPPER_SOURCE_PATH")
	}
	if wrapperSrc == "" {
		candidate := "/usr/local/bin/runner-wrapper"
		if _, err := os.Stat(candidate); err == nil {
			wrapperSrc = candidate
		}
	}

	entrypointSrc := *entrypointSrcFlag
	if entrypointSrc == "" {
		entrypointSrc = os.Getenv("ENTRYPOINT_SOURCE_PATH")
	}
	if entrypointSrc == "" {
		candidate := "/usr/local/bin/entrypoint.sh"
		if _, err := os.Stat(candidate); err == nil {
			entrypointSrc = candidate
		}
	}

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

	initCfg := runner.InitConfig{
		SharedDir:            sharedDir,
		BinaryKey:            strings.TrimSpace(binaryKey),
		ConfigKey:            strings.TrimSpace(configKey),
		WrapperSourcePath:    wrapperSrc,
		EntrypointSourcePath: entrypointSrc,
		S3Config:             s3Cfg,
	}

	initializer := runner.NewRunnerInitializer(s3Client)
	if err := initializer.Init(ctx, initCfg); err != nil {
		log.Fatal().Err(err).Msg("runner-init failed")
	}

	log.Info().Msg("runner pod initialization completed successfully")
}
