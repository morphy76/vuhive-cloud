package runner

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/morphy76/vuhive-cloud/internal/application/ports/outbound"
	"github.com/morphy76/vuhive-cloud/internal/domain/service"
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
		Str("component", "runner-init").
		Str("op", "RunnerInitializer.Init").
		Str("shared_dir", cfg.SharedDir).
		Str("secrets_dir", cfg.SecretsDir).
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
	configDstPath := filepath.Join(cfg.SharedDir, "vuhive.yaml")
	if cfg.ConfigKey != "" {
		log.Debug().Str("destination", configDstPath).Msg("downloading vuhive.yaml configuration")
		if err := r.downloadToFile(ctx, cfg.ConfigKey, configDstPath, 0644); err != nil {
			log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed to download configuration")
			return fmt.Errorf("failed to download configuration: %w", err)
		}
	}

	// 3. Resolve configuration secrets if present
	if err := r.resolveSecrets(ctx, cfg.SecretsDir, configDstPath); err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed to resolve configuration secrets")
		return fmt.Errorf("failed to resolve configuration secrets: %w", err)
	}

	// 4. Copy runner-wrapper if source path exists
	if cfg.WrapperSourcePath != "" {
		wrapperDstPath := filepath.Join(cfg.SharedDir, "runner-wrapper")
		log.Debug().Str("src", cfg.WrapperSourcePath).Str("dst", wrapperDstPath).Msg("copying runner-wrapper")
		if err := copyFile(cfg.WrapperSourcePath, wrapperDstPath, 0755); err != nil {
			log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed to copy runner-wrapper")
			return fmt.Errorf("failed to copy runner-wrapper: %w", err)
		}
	}

	// 5. Copy entrypoint.sh if source path exists, or generate fallback default
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

func (r *RunnerInitializer) resolveSecrets(ctx context.Context, secretsDir, configPath string) error {
	secrets, err := loadSecrets(secretsDir)
	if err != nil {
		return err
	}
	if len(secrets) == 0 {
		return nil
	}

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil
	}

	configBytes, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read configuration file %q: %w", configPath, err)
	}

	rawConfig := string(configBytes)
	placeholderKeys := service.ExtractPlaceholderKeys(rawConfig)
	resolvedConfig := service.ResolveTemplatePlaceholders(rawConfig, secrets)

	var resolvedCount int
	for _, key := range placeholderKeys {
		if _, ok := secrets[key]; ok {
			resolvedCount++
		}
	}

	stat, err := os.Stat(configPath)
	perm := os.FileMode(0644)
	if err == nil {
		perm = stat.Mode().Perm()
	}

	if err := os.WriteFile(configPath, []byte(resolvedConfig), perm); err != nil {
		return fmt.Errorf("failed to write resolved configuration to %q: %w", configPath, err)
	}

	log := zerolog.Ctx(ctx).With().
		Str("component", "runner-init").
		Str("op", "RunnerInitializer.resolveSecrets").
		Str("config_path", configPath).
		Logger()
	log.Info().
		Int("secrets_resolved", resolvedCount).
		Int("secrets_available", len(secrets)).
		Msg("resolved configuration template placeholders with secrets")

	return nil
}

func loadSecrets(secretsDir string) (map[string]string, error) {
	if secretsDir == "" {
		return nil, nil
	}

	entries, err := os.ReadDir(secretsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read secrets directory %q: %w", secretsDir, err)
	}

	if len(entries) == 0 {
		return nil, nil
	}

	secrets := make(map[string]string)
	for _, entry := range entries {
		// Skip Kubernetes metadata files and directories (e.g. "..data", hidden files)
		if strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		filePath := filepath.Join(secretsDir, entry.Name())
		info, err := os.Stat(filePath)
		if err != nil {
			return nil, fmt.Errorf("failed to stat secret file %q: %w", filePath, err)
		}

		if info.IsDir() {
			continue
		}

		content, err := os.ReadFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("failed to read secret file %q: %w", filePath, err)
		}

		secrets[entry.Name()] = string(content)
	}

	return secrets, nil
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
