package runner_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/morphy76/vuhive-cloud/internal/application/ports/outbound"
	"github.com/morphy76/vuhive-cloud/internal/runner"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockStoragePort implements outbound.StoragePort for testing.
type mockStoragePort struct {
	downloadFunc func(ctx context.Context, key string) (io.ReadCloser, error)
	uploadFunc   func(ctx context.Context, key string, content io.Reader, size int64, contentType string) error
	deleteFunc   func(ctx context.Context, key string) error
	existsFunc   func(ctx context.Context, key string) (bool, error)
}

var _ outbound.StoragePort = (*mockStoragePort)(nil)

func (m *mockStoragePort) Download(ctx context.Context, key string) (io.ReadCloser, error) {
	if m.downloadFunc != nil {
		return m.downloadFunc(ctx, key)
	}
	return nil, errors.New("download not implemented")
}

func (m *mockStoragePort) Upload(ctx context.Context, key string, content io.Reader, size int64, contentType string) error {
	if m.uploadFunc != nil {
		return m.uploadFunc(ctx, key, content, size, contentType)
	}
	return nil
}

func (m *mockStoragePort) Delete(ctx context.Context, key string) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, key)
	}
	return nil
}

func (m *mockStoragePort) Exists(ctx context.Context, key string) (bool, error) {
	if m.existsFunc != nil {
		return m.existsFunc(ctx, key)
	}
	return true, nil
}

func (m *mockStoragePort) PresignDownload(ctx context.Context, key string, lifetime time.Duration) (string, error) {
	return "", nil
}

func (m *mockStoragePort) PresignUpload(ctx context.Context, key string, lifetime time.Duration) (string, error) {
	return "", nil
}

func (m *mockStoragePort) EnsureBucket(ctx context.Context) error {
	return nil
}

func TestRunnerInitializer_Success(t *testing.T) {
	tempDir := t.TempDir()
	sharedDir := filepath.Join(tempDir, "shared")

	// Create fake wrapper and entrypoint to copy
	srcWrapper := filepath.Join(tempDir, "dummy-wrapper")
	require.NoError(t, os.WriteFile(srcWrapper, []byte("#!/bin/sh\necho wrapper\n"), 0755))

	srcEntrypoint := filepath.Join(tempDir, "dummy-entrypoint.sh")
	require.NoError(t, os.WriteFile(srcEntrypoint, []byte("#!/bin/sh\necho entrypoint\n"), 0755))

	fakeBinaryData := []byte("ELF-fake-runner-binary-content")
	fakeConfigData := []byte("version: 1\nscenarios: []\n")

	mockStorage := &mockStoragePort{
		downloadFunc: func(ctx context.Context, key string) (io.ReadCloser, error) {
			switch key {
			case "suites/suite-1/artifacts/art-1/linux-amd64/runner":
				return io.NopCloser(bytes.NewReader(fakeBinaryData)), nil
			case "suites/suite-1/configs/cfg-1.yaml":
				return io.NopCloser(bytes.NewReader(fakeConfigData)), nil
			default:
				return nil, errors.New("key not found: " + key)
			}
		},
	}

	initializer := runner.NewRunnerInitializer(mockStorage)
	cfg := runner.InitConfig{
		SharedDir:            sharedDir,
		BinaryKey:            "suites/suite-1/artifacts/art-1/linux-amd64/runner",
		ConfigKey:            "suites/suite-1/configs/cfg-1.yaml",
		WrapperSourcePath:    srcWrapper,
		EntrypointSourcePath: srcEntrypoint,
	}

	err := initializer.Init(context.Background(), cfg)
	require.NoError(t, err)

	// Check runner binary downloaded and executable
	runnerBinary := filepath.Join(sharedDir, "runner")
	assert.FileExists(t, runnerBinary)
	binContent, err := os.ReadFile(runnerBinary)
	require.NoError(t, err)
	assert.Equal(t, fakeBinaryData, binContent)
	info, err := os.Stat(runnerBinary)
	require.NoError(t, err)
	assert.True(t, info.Mode()&0111 != 0, "runner binary should be executable")

	// Check config downloaded
	configFile := filepath.Join(sharedDir, "vuhive.yaml")
	assert.FileExists(t, configFile)
	cfgContent, err := os.ReadFile(configFile)
	require.NoError(t, err)
	assert.Equal(t, fakeConfigData, cfgContent)

	// Check runner-wrapper copied and executable
	destWrapper := filepath.Join(sharedDir, "runner-wrapper")
	assert.FileExists(t, destWrapper)
	wrapperInfo, err := os.Stat(destWrapper)
	require.NoError(t, err)
	assert.True(t, wrapperInfo.Mode()&0111 != 0, "runner-wrapper should be executable")

	// Check entrypoint copied and executable
	destEntrypoint := filepath.Join(sharedDir, "entrypoint.sh")
	assert.FileExists(t, destEntrypoint)
	entrypointInfo, err := os.Stat(destEntrypoint)
	require.NoError(t, err)
	assert.True(t, entrypointInfo.Mode()&0111 != 0, "entrypoint.sh should be executable")
}

func TestRunnerInitializer_NoConfigKey(t *testing.T) {
	tempDir := t.TempDir()
	sharedDir := filepath.Join(tempDir, "shared")

	fakeBinaryData := []byte("fake-binary")
	mockStorage := &mockStoragePort{
		downloadFunc: func(ctx context.Context, key string) (io.ReadCloser, error) {
			if key == "binary-key" {
				return io.NopCloser(bytes.NewReader(fakeBinaryData)), nil
			}
			return nil, errors.New("unexpected key: " + key)
		},
	}

	initializer := runner.NewRunnerInitializer(mockStorage)
	cfg := runner.InitConfig{
		SharedDir: sharedDir,
		BinaryKey: "binary-key",
	}

	err := initializer.Init(context.Background(), cfg)
	require.NoError(t, err)

	assert.FileExists(t, filepath.Join(sharedDir, "runner"))
	assert.NoFileExists(t, filepath.Join(sharedDir, "vuhive.yaml"))
}

func TestRunnerInitializer_ValidationErrors(t *testing.T) {
	mockStorage := &mockStoragePort{}
	initializer := runner.NewRunnerInitializer(mockStorage)

	err := initializer.Init(context.Background(), runner.InitConfig{
		BinaryKey: "", // empty
	})
	require.Error(t, err)
}

func TestRunnerInitializer_StorageDownloadFailure(t *testing.T) {
	mockStorage := &mockStoragePort{
		downloadFunc: func(ctx context.Context, key string) (io.ReadCloser, error) {
			return nil, errors.New("s3 connection timeout")
		},
	}

	initializer := runner.NewRunnerInitializer(mockStorage)
	cfg := runner.InitConfig{
		SharedDir: t.TempDir(),
		BinaryKey: "binary-key",
	}

	err := initializer.Init(context.Background(), cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to download runner binary")
}
