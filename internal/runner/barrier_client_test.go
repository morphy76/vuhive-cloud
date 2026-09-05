package runner_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/morphy76/vuhive-cloud/internal/runner"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHTTPBarrierClient_PreflightFailure(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	nonExistentRunner := filepath.Join(tmpDir, "missing-runner")

	aborted := false
	mockClient := &mockHTTPClient{
		doFunc: func(req *http.Request) (*http.Response, error) {
			if req.URL.Path == "/barrier/abort" {
				aborted = true
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewReader([]byte(`{"status":"ABORTED"}`))),
				}, nil
			}
			return &http.Response{
				StatusCode: http.StatusNotFound,
				Body:       io.NopCloser(bytes.NewReader(nil)),
			}, nil
		},
	}

	cfg := runner.WrapperConfig{
		RunnerPath:     nonExistentRunner,
		CoordinatorURL: "http://vuhive.local/barrier",
		WorkerID:       "worker-fail",
		WorkerCount:    2,
		BarrierEnabled: true,
	}

	client := runner.NewHTTPBarrierClient(mockClient)
	err := client.Rendezvous(context.Background(), cfg)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "preflight check failed")
	assert.True(t, aborted, "must notify coordinator of abort on preflight failure")
}

func TestHTTPBarrierClient_RendezvousSuccess(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	validRunner := filepath.Join(tmpDir, "runner")
	require.NoError(t, os.WriteFile(validRunner, []byte("#!/bin/sh\nexit 0\n"), 0755))

	mockClient := &mockHTTPClient{
		doFunc: func(req *http.Request) (*http.Response, error) {
			if req.URL.Path == "/barrier/await" {
				targetTime := time.Now().UTC().Add(80 * time.Millisecond).Format(time.RFC3339Nano)
				resp := map[string]interface{}{
					"status":            "RELEASED",
					"target_start_time": targetTime,
					"run_id":            "run-123",
					"worker_id":         "worker-1",
					"total_workers":     2,
				}
				data, _ := json.Marshal(resp)
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewReader(data)),
				}, nil
			}
			return &http.Response{
				StatusCode: http.StatusNotFound,
				Body:       io.NopCloser(bytes.NewReader(nil)),
			}, nil
		},
	}

	cfg := runner.WrapperConfig{
		RunnerPath:     validRunner,
		CoordinatorURL: "http://vuhive.local/barrier",
		WorkerID:       "worker-1",
		WorkerCount:    2,
		BarrierEnabled: true,
	}

	client := runner.NewHTTPBarrierClient(mockClient)
	start := time.Now()
	err := client.Rendezvous(context.Background(), cfg)

	require.NoError(t, err)
	assert.True(t, time.Since(start) >= 50*time.Millisecond, "client must synchronize and sleep until target start time")
}

func TestHTTPBarrierClient_RendezvousAborted(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	validRunner := filepath.Join(tmpDir, "runner")
	require.NoError(t, os.WriteFile(validRunner, []byte("#!/bin/sh\nexit 0\n"), 0755))

	mockClient := &mockHTTPClient{
		doFunc: func(req *http.Request) (*http.Response, error) {
			if req.URL.Path == "/barrier/await" {
				return &http.Response{
					StatusCode: http.StatusFailedDependency,
					Body:       io.NopCloser(bytes.NewReader([]byte(`{"error":"barrier rendezvous aborted"}`))),
				}, nil
			}
			return &http.Response{
				StatusCode: http.StatusNotFound,
				Body:       io.NopCloser(bytes.NewReader(nil)),
			}, nil
		},
	}

	cfg := runner.WrapperConfig{
		RunnerPath:     validRunner,
		CoordinatorURL: "http://vuhive.local/barrier",
		WorkerID:       "worker-1",
		WorkerCount:    2,
		BarrierEnabled: true,
	}

	client := runner.NewHTTPBarrierClient(mockClient)
	err := client.Rendezvous(context.Background(), cfg)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "aborted")
}
