package controlplane_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"testing"

	"github.com/morphy76/vuhive-cloud/internal/bff/adapters/outbound/controlplane"
	"github.com/morphy76/vuhive-cloud/internal/bff/domain/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type roundTripperFunc func(req *http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestClient_CheckHealth(t *testing.T) {
	ctx := context.Background()

	t.Run("successful health check", func(t *testing.T) {
		mockTransport := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			assert.Equal(t, "http://controlplane/healthz", req.URL.String())
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBufferString(`{"status":"ok"}`)),
				Header:     make(http.Header),
			}, nil
		})

		client := controlplane.NewClient(controlplane.Config{
			BaseURL:    "http://controlplane",
			HTTPClient: &http.Client{Transport: mockTransport},
		})

		health, err := client.CheckHealth(ctx)
		require.NoError(t, err)
		assert.Equal(t, "UP", health.Status)
	})

	t.Run("server returns 500 error", func(t *testing.T) {
		mockTransport := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusInternalServerError,
				Body:       io.NopCloser(bytes.NewBufferString(`internal error`)),
				Header:     make(http.Header),
			}, nil
		})

		client := controlplane.NewClient(controlplane.Config{
			BaseURL:    "http://controlplane",
			HTTPClient: &http.Client{Transport: mockTransport},
		})

		_, err := client.CheckHealth(ctx)
		assert.Error(t, err)
		assert.ErrorIs(t, err, model.ErrControlPlaneUnavailable)
	})

	t.Run("server connection refused / transport error", func(t *testing.T) {
		mockTransport := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			return nil, errors.New("dial tcp: connection refused")
		})

		client := controlplane.NewClient(controlplane.Config{
			BaseURL:    "http://controlplane",
			HTTPClient: &http.Client{Transport: mockTransport},
		})

		_, err := client.CheckHealth(ctx)
		assert.Error(t, err)
		assert.ErrorIs(t, err, model.ErrControlPlaneUnavailable)
	})
}

func TestClient_GetVersion(t *testing.T) {
	ctx := context.Background()

	t.Run("successful version check", func(t *testing.T) {
		mockTransport := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			assert.Equal(t, "http://controlplane/version", req.URL.String())
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBufferString(`{"version":"0.0.1","commit":"abc1234","build_time":"2026-09-05T10:00:00Z"}`)),
				Header:     make(http.Header),
			}, nil
		})

		client := controlplane.NewClient(controlplane.Config{
			BaseURL:    "http://controlplane",
			HTTPClient: &http.Client{Transport: mockTransport},
		})

		ver, err := client.GetVersion(ctx)
		require.NoError(t, err)
		assert.Equal(t, "0.0.1", ver.Version)
		assert.Equal(t, "abc1234", ver.Commit)
	})

	t.Run("server returns 404 for version", func(t *testing.T) {
		mockTransport := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusNotFound,
				Body:       io.NopCloser(bytes.NewBufferString(`not found`)),
				Header:     make(http.Header),
			}, nil
		})

		client := controlplane.NewClient(controlplane.Config{
			BaseURL:    "http://controlplane",
			HTTPClient: &http.Client{Transport: mockTransport},
		})

		ver, err := client.GetVersion(ctx)
		require.NoError(t, err)
		assert.Equal(t, "unknown", ver.Version)
	})

	t.Run("server connection error during version fetch", func(t *testing.T) {
		mockTransport := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			return nil, errors.New("network failure")
		})

		client := controlplane.NewClient(controlplane.Config{
			BaseURL:    "http://controlplane",
			HTTPClient: &http.Client{Transport: mockTransport},
		})

		_, err := client.GetVersion(ctx)
		assert.Error(t, err)
		assert.ErrorIs(t, err, model.ErrControlPlaneUnavailable)
	})
}

func TestClient_ConnectionPoolingAndHeaders(t *testing.T) {
	ctx := context.Background()

	t.Run("propagates bearer token if configured", func(t *testing.T) {
		mockTransport := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			assert.Equal(t, "Bearer secret-token", req.Header.Get("Authorization"))
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBufferString(`{"status":"ok"}`)),
				Header:     make(http.Header),
			}, nil
		})

		client := controlplane.NewClient(controlplane.Config{
			BaseURL:    "http://controlplane",
			AuthToken:  "secret-token",
			HTTPClient: &http.Client{Transport: mockTransport},
		})

		health, err := client.CheckHealth(ctx)
		require.NoError(t, err)
		assert.Equal(t, "UP", health.Status)
	})

	t.Run("default client initializes connection pooling transport", func(t *testing.T) {
		client := controlplane.NewClient(controlplane.Config{
			BaseURL: "http://controlplane",
		})
		require.NotNil(t, client)
		transport, ok := client.HTTPClient().Transport.(*http.Transport)
		require.True(t, ok, "expected *http.Transport")
		assert.Equal(t, 100, transport.MaxIdleConns)
		assert.Equal(t, 50, transport.MaxIdleConnsPerHost)
	})
}

func TestClient_RetryMechanism(t *testing.T) {
	ctx := context.Background()

	t.Run("retries on transient 5xx error and succeeds", func(t *testing.T) {
		attempts := 0
		mockTransport := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			attempts++
			if attempts == 1 {
				return &http.Response{
					StatusCode: http.StatusBadGateway,
					Body:       io.NopCloser(bytes.NewBufferString(`bad gateway`)),
					Header:     make(http.Header),
				}, nil
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBufferString(`{"status":"ok"}`)),
				Header:     make(http.Header),
			}, nil
		})

		client := controlplane.NewClient(controlplane.Config{
			BaseURL:    "http://controlplane",
			MaxRetries: 2,
			HTTPClient: &http.Client{Transport: mockTransport},
		})

		health, err := client.CheckHealth(ctx)
		require.NoError(t, err)
		assert.Equal(t, "UP", health.Status)
		assert.Equal(t, 2, attempts)
	})
}

func TestClient_GetActiveRunsCount(t *testing.T) {
	ctx := context.Background()

	t.Run("returns active runs count by querying running and queued runs", func(t *testing.T) {
		mockTransport := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			status := req.URL.Query().Get("status")
			var body string
			switch status {
			case "RUNNING":
				body = `{"runs":[],"count":0,"total":3,"limit":1,"offset":0}`
			case "QUEUED":
				body = `{"runs":[],"count":0,"total":2,"limit":1,"offset":0}`
			default:
				body = `{"runs":[],"count":0,"total":0,"limit":1,"offset":0}`
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBufferString(body)),
				Header:     make(http.Header),
			}, nil
		})

		client := controlplane.NewClient(controlplane.Config{
			BaseURL:    "http://controlplane",
			HTTPClient: &http.Client{Transport: mockTransport},
		})

		count, err := client.GetActiveRunsCount(ctx)
		require.NoError(t, err)
		assert.Equal(t, int64(5), count)
	})
}

func TestClient_ListRecentSuites(t *testing.T) {
	ctx := context.Background()

	t.Run("returns suites from control plane", func(t *testing.T) {
		mockTransport := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			assert.Equal(t, "/api/v1/suites", req.URL.Path)
			body := `{"suites":[{"id":"suite-1","name":"Checkout","description":"Checkout test","state":"ACTIVE","created_at":"2026-09-01T00:00:00Z","updated_at":"2026-09-01T00:00:00Z"}],"count":1}`
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBufferString(body)),
				Header:     make(http.Header),
			}, nil
		})

		client := controlplane.NewClient(controlplane.Config{
			BaseURL:    "http://controlplane",
			HTTPClient: &http.Client{Transport: mockTransport},
		})

		suites, err := client.ListRecentSuites(ctx, 5)
		require.NoError(t, err)
		require.Len(t, suites, 1)
		assert.Equal(t, "suite-1", suites[0].ID)
		assert.Equal(t, "Checkout", suites[0].Name)
	})

	t.Run("returns empty slice gracefully when 404 returned", func(t *testing.T) {
		mockTransport := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusNotFound,
				Body:       io.NopCloser(bytes.NewBufferString(`not found`)),
				Header:     make(http.Header),
			}, nil
		})

		client := controlplane.NewClient(controlplane.Config{
			BaseURL:    "http://controlplane",
			HTTPClient: &http.Client{Transport: mockTransport},
		})

		suites, err := client.ListRecentSuites(ctx, 5)
		require.NoError(t, err)
		assert.Empty(t, suites)
	})
}

func TestClient_ListProfiles(t *testing.T) {
	ctx := context.Background()

	t.Run("returns profiles from control plane", func(t *testing.T) {
		mockTransport := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			assert.Equal(t, "/api/v1/profiles", req.URL.Path)
			body := `{"profiles":[{"id":"prof-1","name":"Standard","description":"Standard 1-node","runner_image":"vuhive/runner:latest","cpu_request":"500m","cpu_limit":"1000m","memory_limit":"1Gi","created_at":"2026-09-01T00:00:00Z"}],"count":1}`
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBufferString(body)),
				Header:     make(http.Header),
			}, nil
		})

		client := controlplane.NewClient(controlplane.Config{
			BaseURL:    "http://controlplane",
			HTTPClient: &http.Client{Transport: mockTransport},
		})

		profiles, err := client.ListProfiles(ctx)
		require.NoError(t, err)
		require.Len(t, profiles, 1)
		assert.Equal(t, "prof-1", profiles[0].ID)
		assert.Equal(t, "Standard", profiles[0].Name)
	})
}

func TestClient_GetRun(t *testing.T) {
	ctx := context.Background()

	t.Run("returns run detail successfully", func(t *testing.T) {
		mockTransport := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			assert.Equal(t, "/api/v1/runs/run-123", req.URL.Path)
			body := `{
				"id": "run-123",
				"suite_id": "suite-1",
				"artifact_id": "art-1",
				"runner_profile_id": "prof-1",
				"status": "COMPLETED",
				"metrics": {
					"total_iterations": 1000,
					"total_requests": 5000,
					"avg_tps": 250.5,
					"p50_duration_ms": 12.3,
					"p90_duration_ms": 25.4,
					"p95_duration_ms": 35.1,
					"p99_duration_ms": 48.9,
					"error_rate_pct": 0.05
				},
				"s3_report_key": "runs/run-123/summary.json",
				"s3_logs_key": "runs/run-123/run.log",
				"created_at": "2026-09-01T00:00:00Z"
			}`
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBufferString(body)),
				Header:     make(http.Header),
			}, nil
		})

		client := controlplane.NewClient(controlplane.Config{
			BaseURL:    "http://controlplane",
			HTTPClient: &http.Client{Transport: mockTransport},
		})

		run, err := client.GetRun(ctx, "run-123")
		require.NoError(t, err)
		require.NotNil(t, run)
		assert.Equal(t, "run-123", run.ID)
		assert.Equal(t, "COMPLETED", run.Status)
		assert.Equal(t, int64(1000), run.Metrics.TotalIterations)
		assert.Equal(t, 250.5, run.Metrics.AvgTPS)
		assert.Equal(t, "runs/run-123/summary.json", run.S3ReportKey)
	})

	t.Run("returns ErrRunNotFound on 404", func(t *testing.T) {
		mockTransport := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusNotFound,
				Body:       io.NopCloser(bytes.NewBufferString(`{"error":"run not found"}`)),
				Header:     make(http.Header),
			}, nil
		})

		client := controlplane.NewClient(controlplane.Config{
			BaseURL:    "http://controlplane",
			HTTPClient: &http.Client{Transport: mockTransport},
		})

		_, err := client.GetRun(ctx, "missing-run")
		assert.ErrorIs(t, err, model.ErrRunNotFound)
	})
}

func TestClient_ArtifactPresignedURLs(t *testing.T) {
	ctx := context.Background()

	t.Run("GetRunReportURL fetches presigned url", func(t *testing.T) {
		mockTransport := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			assert.Equal(t, "/api/v1/runs/run-1/report", req.URL.Path)
			assert.Equal(t, "true", req.URL.Query().Get("presign"))
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBufferString(`{"download_url":"https://s3/report.json","expires_in_seconds":900}`)),
				Header:     make(http.Header),
			}, nil
		})

		client := controlplane.NewClient(controlplane.Config{
			BaseURL:    "http://controlplane",
			HTTPClient: &http.Client{Transport: mockTransport},
		})

		url, err := client.GetRunReportURL(ctx, "run-1")
		require.NoError(t, err)
		assert.Equal(t, "https://s3/report.json", url)
	})

	t.Run("GetRunLogsURL fetches presigned url", func(t *testing.T) {
		mockTransport := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			assert.Equal(t, "/api/v1/runs/run-1/logs", req.URL.Path)
			assert.Equal(t, "true", req.URL.Query().Get("presign"))
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBufferString(`{"download_url":"https://s3/run.log","expires_in_seconds":900}`)),
				Header:     make(http.Header),
			}, nil
		})

		client := controlplane.NewClient(controlplane.Config{
			BaseURL:    "http://controlplane",
			HTTPClient: &http.Client{Transport: mockTransport},
		})

		url, err := client.GetRunLogsURL(ctx, "run-1")
		require.NoError(t, err)
		assert.Equal(t, "https://s3/run.log", url)
	})
}
