package runner_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/morphy76/vuhive-cloud/internal/runner"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type uploadedFile struct {
	key         string
	content     []byte
	size        int64
	contentType string
}

func TestRunnerWrapper_Success(t *testing.T) {
	tempDir := t.TempDir()
	runnerPath := filepath.Join(tempDir, "runner")
	summaryPath := filepath.Join(tempDir, "summary.json")
	logPath := filepath.Join(tempDir, "run.log")

	// Create mock runner script
	script := fmt.Sprintf(`#!/bin/sh
echo "runner starting..."
echo '{"status":"PASS","iterations":100,"avg_tps":50.5}' > "%s"
echo "runner finished successfully"
exit 0
`, summaryPath)
	require.NoError(t, os.WriteFile(runnerPath, []byte(script), 0755))

	var mu sync.Mutex
	uploads := make(map[string]uploadedFile)
	mockStorage := &mockStoragePort{
		uploadFunc: func(ctx context.Context, key string, content io.Reader, size int64, contentType string) error {
			data, err := io.ReadAll(content)
			if err != nil {
				return err
			}
			mu.Lock()
			uploads[key] = uploadedFile{
				key:         key,
				content:     data,
				size:        size,
				contentType: contentType,
			}
			mu.Unlock()
			return nil
		},
	}

	wrapper := runner.NewRunnerWrapper(mockStorage)
	cfg := runner.WrapperConfig{
		RunnerPath:  runnerPath,
		SummaryPath: summaryPath,
		LogPath:     logPath,
		ReportKey:   "runs/run-123/summary.json",
		LogsKey:     "runs/run-123/run.log",
	}

	exitCode, err := wrapper.Run(context.Background(), cfg, []string{"--extra-arg=true"})
	require.NoError(t, err)
	assert.Equal(t, 0, exitCode)

	// Verify log file exists and contains runner output
	assert.FileExists(t, logPath)
	logData, err := os.ReadFile(logPath)
	require.NoError(t, err)
	assert.Contains(t, string(logData), "runner starting...")
	assert.Contains(t, string(logData), "runner finished successfully")

	// Verify S3 uploads
	mu.Lock()
	defer mu.Unlock()
	require.Contains(t, uploads, "runs/run-123/summary.json")
	assert.Contains(t, string(uploads["runs/run-123/summary.json"].content), `"status":"PASS"`)

	require.Contains(t, uploads, "runs/run-123/run.log")
	assert.Contains(t, string(uploads["runs/run-123/run.log"].content), "runner starting...")
}

func TestRunnerWrapper_NonZeroExitCode_WithSummary(t *testing.T) {
	tempDir := t.TempDir()
	runnerPath := filepath.Join(tempDir, "runner")
	summaryPath := filepath.Join(tempDir, "summary.json")
	logPath := filepath.Join(tempDir, "run.log")

	// Mock runner script that writes summary but exits with failure exit code 42
	script := fmt.Sprintf(`#!/bin/sh
echo "runner running with failure..."
echo '{"status":"FAILED","sla_passed":false}' > "%s"
exit 42
`, summaryPath)
	require.NoError(t, os.WriteFile(runnerPath, []byte(script), 0755))

	var mu sync.Mutex
	uploads := make(map[string]uploadedFile)
	mockStorage := &mockStoragePort{
		uploadFunc: func(ctx context.Context, key string, content io.Reader, size int64, contentType string) error {
			data, err := io.ReadAll(content)
			if err != nil {
				return err
			}
			mu.Lock()
			uploads[key] = uploadedFile{
				key:         key,
				content:     data,
				contentType: contentType,
			}
			mu.Unlock()
			return nil
		},
	}

	wrapper := runner.NewRunnerWrapper(mockStorage)
	cfg := runner.WrapperConfig{
		RunnerPath:  runnerPath,
		SummaryPath: summaryPath,
		LogPath:     logPath,
		ReportKey:   "runs/run-456/summary.json",
		LogsKey:     "runs/run-456/run.log",
	}

	exitCode, err := wrapper.Run(context.Background(), cfg, nil)
	require.NoError(t, err)
	assert.Equal(t, 42, exitCode)

	mu.Lock()
	defer mu.Unlock()
	// Report and logs MUST still be uploaded
	require.Contains(t, uploads, "runs/run-456/summary.json")
	assert.Contains(t, string(uploads["runs/run-456/summary.json"].content), `"status":"FAILED"`)
	require.Contains(t, uploads, "runs/run-456/run.log")
	assert.Contains(t, string(uploads["runs/run-456/run.log"].content), "runner running with failure...")
}

func TestRunnerWrapper_NonZeroExitCode_WithoutSummary(t *testing.T) {
	tempDir := t.TempDir()
	runnerPath := filepath.Join(tempDir, "runner")
	summaryPath := filepath.Join(tempDir, "summary.json")
	logPath := filepath.Join(tempDir, "run.log")

	// Mock runner script that crashes without creating summary.json
	script := `#!/bin/sh
echo "runner fatal error: panic" >&2
exit 137
`
	require.NoError(t, os.WriteFile(runnerPath, []byte(script), 0755))

	var mu sync.Mutex
	uploads := make(map[string]uploadedFile)
	mockStorage := &mockStoragePort{
		uploadFunc: func(ctx context.Context, key string, content io.Reader, size int64, contentType string) error {
			data, err := io.ReadAll(content)
			if err != nil {
				return err
			}
			mu.Lock()
			uploads[key] = uploadedFile{
				key:         key,
				content:     data,
				contentType: contentType,
			}
			mu.Unlock()
			return nil
		},
	}

	wrapper := runner.NewRunnerWrapper(mockStorage)
	cfg := runner.WrapperConfig{
		RunnerPath:  runnerPath,
		SummaryPath: summaryPath,
		LogPath:     logPath,
		ReportKey:   "runs/run-panic/summary.json",
		LogsKey:     "runs/run-panic/run.log",
	}

	exitCode, err := wrapper.Run(context.Background(), cfg, nil)
	require.NoError(t, err)
	assert.Equal(t, 137, exitCode)

	mu.Lock()
	defer mu.Unlock()
	// Guaranteed fallback report MUST be uploaded
	require.Contains(t, uploads, "runs/run-panic/summary.json")
	var fallback map[string]interface{}
	require.NoError(t, json.Unmarshal(uploads["runs/run-panic/summary.json"].content, &fallback))
	assert.Equal(t, "FAILED", fallback["status"])
	assert.Equal(t, float64(137), fallback["exit_code"])
	assert.NotEmpty(t, fallback["error"])

	// Logs must also be uploaded
	require.Contains(t, uploads, "runs/run-panic/run.log")
	assert.Contains(t, string(uploads["runs/run-panic/run.log"].content), "runner fatal error: panic")
}

func TestRunnerWrapper_WithConfigFile(t *testing.T) {
	tempDir := t.TempDir()
	runnerPath := filepath.Join(tempDir, "runner")
	summaryPath := filepath.Join(tempDir, "summary.json")
	logPath := filepath.Join(tempDir, "run.log")
	configPath := filepath.Join(tempDir, "vuhive.yaml")

	require.NoError(t, os.WriteFile(configPath, []byte("target: http://example.com\n"), 0644))

	// Mock runner checks if --config flag was passed
	script := fmt.Sprintf(`#!/bin/sh
has_config=false
for arg in "$@"; do
    case "$arg" in
        --config=*) has_config=true ;;
    esac
done

if [ "$has_config" = "false" ]; then
    echo "missing --config flag" >&2
    exit 2
fi

echo '{}' > "%s"
echo "config verified"
exit 0
`, summaryPath)
	require.NoError(t, os.WriteFile(runnerPath, []byte(script), 0755))

	mockStorage := &mockStoragePort{}
	wrapper := runner.NewRunnerWrapper(mockStorage)
	cfg := runner.WrapperConfig{
		RunnerPath:  runnerPath,
		ConfigPath:  configPath,
		SummaryPath: summaryPath,
		LogPath:     logPath,
		ReportKey:   "runs/test/summary.json",
		LogsKey:     "runs/test/run.log",
	}

	exitCode, err := wrapper.Run(context.Background(), cfg, nil)
	require.NoError(t, err)
	assert.Equal(t, 0, exitCode)
}

func TestRunnerWrapper_APICallback(t *testing.T) {
	tempDir := t.TempDir()
	runnerPath := filepath.Join(tempDir, "runner")
	summaryPath := filepath.Join(tempDir, "summary.json")
	logPath := filepath.Join(tempDir, "run.log")

	script := fmt.Sprintf(`#!/bin/sh
echo '{}' > "%s"
exit 0
`, summaryPath)
	require.NoError(t, os.WriteFile(runnerPath, []byte(script), 0755))

	var callbackReceived bool
	var callbackBody map[string]interface{}
	mockClient := &mockHTTPClient{
		doFunc: func(req *http.Request) (*http.Response, error) {
			callbackReceived = true
			_ = json.NewDecoder(req.Body).Decode(&callbackBody)
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewReader([]byte(`{"status":"acknowledged"}`))),
			}, nil
		},
	}

	mockStorage := &mockStoragePort{}
	wrapper := runner.NewRunnerWrapper(mockStorage)
	wrapper.SetHTTPClient(mockClient)
	cfg := runner.WrapperConfig{
		RunnerPath:     runnerPath,
		SummaryPath:    summaryPath,
		LogPath:        logPath,
		ReportKey:      "runs/run-callback/summary.json",
		LogsKey:        "runs/run-callback/run.log",
		APICallbackURL: "http://vuhive-control-plane:8080/api/v1/runs/run-callback/complete",
	}

	exitCode, err := wrapper.Run(context.Background(), cfg, nil)
	require.NoError(t, err)
	assert.Equal(t, 0, exitCode)
	assert.True(t, callbackReceived)
	assert.Equal(t, float64(0), callbackBody["exit_code"])
	assert.Equal(t, "runs/run-callback/summary.json", callbackBody["report_key"])
	assert.Equal(t, "runs/run-callback/run.log", callbackBody["logs_key"])
}

type mockHTTPClient struct {
	doFunc func(req *http.Request) (*http.Response, error)
}

func (m *mockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	if m.doFunc != nil {
		return m.doFunc(req)
	}
	return nil, errors.New("unhandled request")
}
