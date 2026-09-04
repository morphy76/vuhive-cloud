package runner

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/morphy76/vuhive-cloud/internal/application/ports/outbound"
	"github.com/rs/zerolog"
)

// RunnerInitializer handles downloading test runner binaries and configs into the pod's shared volume.
type RunnerInitializer struct {
	storage outbound.StoragePort
}

// NewRunnerInitializer creates a new RunnerInitializer using the given storage adapter.
func NewRunnerInitializer(storage outbound.StoragePort) *RunnerInitializer {
	return &RunnerInitializer{
		storage: storage,
	}
}

// Init downloads the target runner binary and configuration from S3 into the configured shared directory.
func (r *RunnerInitializer) Init(ctx context.Context, cfg InitConfig) error {
	start := time.Now()
	log := zerolog.Ctx(ctx).With().
		Str("op", "RunnerInitializer.Init").
		Str("shared_dir", cfg.SharedDir).
		Str("binary_key", cfg.BinaryKey).
		Str("config_key", cfg.ConfigKey).
		Logger()
	log.Debug().Msg("starting runner pod initialization")

	if err := cfg.Validate(); err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("invalid init configuration")
		return err
	}

	if err := os.MkdirAll(cfg.SharedDir, 0755); err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed to create shared directory")
		return fmt.Errorf("failed to create shared directory %q: %w", cfg.SharedDir, err)
	}

	// 1. Download runner binary
	runnerDstPath := filepath.Join(cfg.SharedDir, "runner")
	log.Debug().Str("destination", runnerDstPath).Msg("downloading runner binary")
	if err := r.downloadToFile(ctx, cfg.BinaryKey, runnerDstPath, 0755); err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed to download runner binary")
		return fmt.Errorf("failed to download runner binary: %w", err)
	}

	// 2. Download configuration YAML if specified
	if cfg.ConfigKey != "" {
		configDstPath := filepath.Join(cfg.SharedDir, "vuhive.yaml")
		log.Debug().Str("destination", configDstPath).Msg("downloading vuhive.yaml configuration")
		if err := r.downloadToFile(ctx, cfg.ConfigKey, configDstPath, 0644); err != nil {
			log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed to download configuration")
			return fmt.Errorf("failed to download configuration: %w", err)
		}
	}

	// 3. Copy runner-wrapper if source path exists
	if cfg.WrapperSourcePath != "" {
		wrapperDstPath := filepath.Join(cfg.SharedDir, "runner-wrapper")
		log.Debug().Str("src", cfg.WrapperSourcePath).Str("dst", wrapperDstPath).Msg("copying runner-wrapper")
		if err := copyFile(cfg.WrapperSourcePath, wrapperDstPath, 0755); err != nil {
			log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed to copy runner-wrapper")
			return fmt.Errorf("failed to copy runner-wrapper: %w", err)
		}
	}

	// 4. Copy entrypoint.sh if source path exists, or generate fallback default
	entrypointDstPath := filepath.Join(cfg.SharedDir, "entrypoint.sh")
	if cfg.EntrypointSourcePath != "" {
		log.Debug().Str("src", cfg.EntrypointSourcePath).Str("dst", entrypointDstPath).Msg("copying entrypoint.sh")
		if err := copyFile(cfg.EntrypointSourcePath, entrypointDstPath, 0755); err != nil {
			log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed to copy entrypoint.sh")
			return fmt.Errorf("failed to copy entrypoint.sh: %w", err)
		}
	} else if _, err := os.Stat(entrypointDstPath); os.IsNotExist(err) {
		log.Debug().Msg("writing default entrypoint.sh")
		if err := os.WriteFile(entrypointDstPath, []byte(defaultEntrypointScript), 0755); err != nil {
			log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed to write default entrypoint.sh")
			return fmt.Errorf("failed to write default entrypoint.sh: %w", err)
		}
	}

	log.Info().Dur("duration_ms", time.Since(start)).Msg("completed runner pod initialization")
	return nil
}

func (r *RunnerInitializer) downloadToFile(ctx context.Context, s3Key, dstPath string, perm os.FileMode) error {
	reader, err := r.storage.Download(ctx, s3Key)
	if err != nil {
		return err
	}
	defer func() { _ = reader.Close() }()

	// Use temporary file in destination directory for atomic write
	tmpFile, err := os.CreateTemp(filepath.Dir(dstPath), "tmp-download-*")
	if err != nil {
		return fmt.Errorf("failed to create temporary download file: %w", err)
	}
	tmpName := tmpFile.Name()
	defer func() { _ = os.Remove(tmpName) }()

	if _, err := io.Copy(tmpFile, reader); err != nil {
		_ = tmpFile.Close()
		return fmt.Errorf("failed writing downloaded bytes: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("failed closing temp file: %w", err)
	}

	if err := os.Chmod(tmpName, perm); err != nil {
		return fmt.Errorf("failed setting file permissions: %w", err)
	}

	if err := os.Rename(tmpName, dstPath); err != nil {
		return fmt.Errorf("failed renaming downloaded file to destination %q: %w", dstPath, err)
	}

	return nil
}

func copyFile(src, dst string, perm os.FileMode) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = srcFile.Close() }()

	dstFile, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, perm)
	if err != nil {
		return err
	}
	defer func() { _ = dstFile.Close() }()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return err
	}

	return os.Chmod(dst, perm)
}

const defaultEntrypointScript = `#!/bin/sh
set -e

# If runner-wrapper binary exists, delegate to it
if [ -x "/shared/runner-wrapper" ]; then
    exec /shared/runner-wrapper "$@"
fi

RUNNER_BIN="/shared/runner"
SUMMARY_FILE="/shared/summary.json"
LOG_FILE="/shared/run.log"
CONFIG_FILE="/shared/vuhive.yaml"

ARGS="--summary-export=${SUMMARY_FILE}"
if [ -f "${CONFIG_FILE}" ]; then
    ARGS="${ARGS} --config=${CONFIG_FILE}"
fi

echo "Starting runner execution..."
EXIT_CODE=0
"${RUNNER_BIN}" ${ARGS} "$@" 2>&1 | tee "${LOG_FILE}" || EXIT_CODE=$?

echo "Runner execution finished with exit code: ${EXIT_CODE}"
exit ${EXIT_CODE}
`

// Compile-time interface assertion
var _ Initializer = (*RunnerInitializer)(nil)
