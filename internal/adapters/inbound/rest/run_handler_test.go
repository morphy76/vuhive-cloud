package rest_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/morphy76/vuhive-cloud/internal/adapters/inbound/rest"
	"github.com/morphy76/vuhive-cloud/internal/application/ports/inbound"
	"github.com/morphy76/vuhive-cloud/internal/domain/model"
)

type mockRunsUseCase struct {
	triggerRunFunc      func(ctx context.Context, cmd inbound.TriggerRunCommand) (*model.TestRun, error)
	getRunFunc          func(ctx context.Context, id string) (*model.TestRun, error)
	listRunsFunc        func(ctx context.Context, filter model.RunFilter) ([]*model.TestRun, int64, error)
	abortRunFunc        func(ctx context.Context, id string, reason string) (*model.TestRun, error)
	completeRunFunc     func(ctx context.Context, cmd inbound.CompleteRunCommand) (*model.TestRun, error)
	getRunReportFunc    func(ctx context.Context, id string) (io.ReadCloser, error)
	getRunReportURLFunc func(ctx context.Context, id string, lifetime time.Duration) (string, error)
	getRunLogsFunc      func(ctx context.Context, id string) (io.ReadCloser, error)
	getRunLogsURLFunc   func(ctx context.Context, id string, lifetime time.Duration) (string, error)
}

func (m *mockRunsUseCase) TriggerRun(ctx context.Context, cmd inbound.TriggerRunCommand) (*model.TestRun, error) {
	if m.triggerRunFunc != nil {
		return m.triggerRunFunc(ctx, cmd)
	}
	return nil, nil
}
func (m *mockRunsUseCase) GetRun(ctx context.Context, id string) (*model.TestRun, error) {
	if m.getRunFunc != nil {
		return m.getRunFunc(ctx, id)
	}
	return nil, nil
}
func (m *mockRunsUseCase) ListRuns(ctx context.Context, filter model.RunFilter) ([]*model.TestRun, int64, error) {
	if m.listRunsFunc != nil {
		return m.listRunsFunc(ctx, filter)
	}
	return nil, 0, nil
}
func (m *mockRunsUseCase) AbortRun(ctx context.Context, id string, reason string) (*model.TestRun, error) {
	if m.abortRunFunc != nil {
		return m.abortRunFunc(ctx, id, reason)
	}
	return nil, nil
}
func (m *mockRunsUseCase) CompleteRun(ctx context.Context, cmd inbound.CompleteRunCommand) (*model.TestRun, error) {
	if m.completeRunFunc != nil {
		return m.completeRunFunc(ctx, cmd)
	}
	return nil, nil
}
func (m *mockRunsUseCase) GetRunReport(ctx context.Context, id string) (io.ReadCloser, error) {
	if m.getRunReportFunc != nil {
		return m.getRunReportFunc(ctx, id)
	}
	return nil, nil
}
func (m *mockRunsUseCase) GetRunReportURL(ctx context.Context, id string, lifetime time.Duration) (string, error) {
	if m.getRunReportURLFunc != nil {
		return m.getRunReportURLFunc(ctx, id, lifetime)
	}
	return "", nil
}
func (m *mockRunsUseCase) GetRunLogs(ctx context.Context, id string) (io.ReadCloser, error) {
	if m.getRunLogsFunc != nil {
		return m.getRunLogsFunc(ctx, id)
	}
	return nil, nil
}
func (m *mockRunsUseCase) GetRunLogsURL(ctx context.Context, id string, lifetime time.Duration) (string, error) {
	if m.getRunLogsURLFunc != nil {
		return m.getRunLogsURLFunc(ctx, id, lifetime)
	}
	return "", nil
}

var _ inbound.RunsUseCase = (*mockRunsUseCase)(nil)

func TestRunHandler_CompleteRun(t *testing.T) {
	now := time.Now().UTC()
	exitCode := 0
	slaPassed := true
	metrics := model.RunMetrics{
		TotalIterations: 100,
		TotalRequests:   500,
		AvgTPS:          50.0,
		P50DurationMs:   10.0,
		P90DurationMs:   20.0,
		P95DurationMs:   30.0,
		P99DurationMs:   45.0,
		ErrorRatePct:    0.5,
	}

	completedRun, err := model.NewTestRunWithID(
		"run-123", "suite-1", "art-1", nil, "prof-1", nil,
		model.RunStatusCompleted, "vuhive-run-job", "vuhive-runners",
		&now, &now, &exitCode, &slaPassed,
		metrics, "runs/run-123/summary.json", "runs/run-123/run.log",
		[]byte(`{"status":"PASS"}`), "", now,
	)
	require.NoError(t, err)

	t.Run("successful run completion via POST /api/v1/runs/:id/complete", func(t *testing.T) {
		mockUC := &mockRunsUseCase{
			completeRunFunc: func(ctx context.Context, cmd inbound.CompleteRunCommand) (*model.TestRun, error) {
				assert.Equal(t, "run-123", cmd.RunID)
				assert.Equal(t, 0, *cmd.ExitCode)
				assert.Equal(t, "runs/run-123/summary.json", cmd.ReportKey)
				return completedRun, nil
			},
		}

		router := rest.SetupRouter(nil, nil, nil, mockUC)

		body := map[string]interface{}{
			"exit_code":  0,
			"report_key": "runs/run-123/summary.json",
			"logs_key":   "runs/run-123/run.log",
		}
		raw, _ := json.Marshal(body)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/runs/run-123/complete", bytes.NewReader(raw))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)

		var res rest.RunResponse
		require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &res))
		assert.Equal(t, "run-123", res.ID)
		assert.Equal(t, "COMPLETED", res.Status)
		require.NotNil(t, res.SLAPassed)
		assert.True(t, *res.SLAPassed)
		assert.Equal(t, int64(100), res.Metrics.TotalIterations)
		assert.Equal(t, int64(500), res.Metrics.TotalRequests)
		assert.InDelta(t, 10.0, res.Metrics.P50DurationMs, 0.01)
	})

	t.Run("run not found returns 404", func(t *testing.T) {
		mockUC := &mockRunsUseCase{
			completeRunFunc: func(ctx context.Context, cmd inbound.CompleteRunCommand) (*model.TestRun, error) {
				return nil, model.ErrNotFound
			},
		}

		router := rest.SetupRouter(nil, nil, nil, mockUC)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/runs/missing/complete", bytes.NewReader([]byte(`{}`)))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusNotFound, resp.Code)
	})

	t.Run("terminal run returns 409 conflict", func(t *testing.T) {
		mockUC := &mockRunsUseCase{
			completeRunFunc: func(ctx context.Context, cmd inbound.CompleteRunCommand) (*model.TestRun, error) {
				return nil, model.ErrTerminalState
			},
		}

		router := rest.SetupRouter(nil, nil, nil, mockUC)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/runs/run-123/complete", bytes.NewReader([]byte(`{}`)))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusConflict, resp.Code)
	})

	t.Run("malformed body returns 400 bad request", func(t *testing.T) {
		mockUC := &mockRunsUseCase{}
		router := rest.SetupRouter(nil, nil, nil, mockUC)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/runs/run-123/complete", bytes.NewReader([]byte(`{invalid json`)))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusBadRequest, resp.Code)
	})
}

func TestRunHandler_AbortRun(t *testing.T) {
	now := time.Now().UTC()

	abortedRun, err := model.NewTestRunWithID(
		"run-123", "suite-1", "art-1", nil, "prof-1", nil,
		model.RunStatusAborted, "vuhive-run-job", "vuhive-runners",
		&now, &now, nil, nil,
		model.RunMetrics{}, "", "",
		nil, "too much latency (requested by: operator-bob)", now,
	)
	require.NoError(t, err)

	t.Run("successful abort via POST /api/v1/runs/:id/abort with body", func(t *testing.T) {
		mockUC := &mockRunsUseCase{
			abortRunFunc: func(ctx context.Context, id string, reason string) (*model.TestRun, error) {
				assert.Equal(t, "run-123", id)
				assert.Equal(t, "too much latency (requested by: operator-bob)", reason)
				return abortedRun, nil
			},
		}

		router := rest.SetupRouter(nil, nil, nil, mockUC)

		body := map[string]interface{}{
			"reason":       "too much latency",
			"requested_by": "operator-bob",
		}
		raw, _ := json.Marshal(body)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/runs/run-123/abort", bytes.NewReader(raw))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)

		var res rest.RunResponse
		require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &res))
		assert.Equal(t, "run-123", res.ID)
		assert.Equal(t, "ABORTED", res.Status)
		assert.Equal(t, "too much latency (requested by: operator-bob)", res.AbortReason)
	})

	t.Run("successful abort with empty body defaults reason", func(t *testing.T) {
		defaultAborted, err := model.NewTestRunWithID(
			"run-456", "suite-1", "art-1", nil, "prof-1", nil,
			model.RunStatusAborted, "vuhive-run-job", "vuhive-runners",
			&now, &now, nil, nil,
			model.RunMetrics{}, "", "",
			nil, "manual cancellation via API", now,
		)
		require.NoError(t, err)

		mockUC := &mockRunsUseCase{
			abortRunFunc: func(ctx context.Context, id string, reason string) (*model.TestRun, error) {
				assert.Equal(t, "run-456", id)
				assert.Equal(t, "manual cancellation via API", reason)
				return defaultAborted, nil
			},
		}

		router := rest.SetupRouter(nil, nil, nil, mockUC)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/runs/run-456/abort", nil)
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)

		var res rest.RunResponse
		require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &res))
		assert.Equal(t, "run-456", res.ID)
		assert.Equal(t, "ABORTED", res.Status)
		assert.Equal(t, "manual cancellation via API", res.AbortReason)
	})

	t.Run("terminal run returns 409 conflict", func(t *testing.T) {
		mockUC := &mockRunsUseCase{
			abortRunFunc: func(ctx context.Context, id string, reason string) (*model.TestRun, error) {
				return nil, model.ErrTerminalState
			},
		}

		router := rest.SetupRouter(nil, nil, nil, mockUC)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/runs/run-123/abort", nil)
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusConflict, resp.Code)
	})

	t.Run("run not found returns 404", func(t *testing.T) {
		mockUC := &mockRunsUseCase{
			abortRunFunc: func(ctx context.Context, id string, reason string) (*model.TestRun, error) {
				return nil, model.ErrNotFound
			},
		}

		router := rest.SetupRouter(nil, nil, nil, mockUC)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/runs/missing/abort", nil)
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusNotFound, resp.Code)
	})

	t.Run("validation error returns 400 bad request", func(t *testing.T) {
		mockUC := &mockRunsUseCase{
			abortRunFunc: func(ctx context.Context, id string, reason string) (*model.TestRun, error) {
				return nil, model.ErrValidation
			},
		}

		router := rest.SetupRouter(nil, nil, nil, mockUC)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/runs/invalid/abort", nil)
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusBadRequest, resp.Code)
	})
}

func TestRunHandler_GetRun(t *testing.T) {
	now := time.Now().UTC()
	exitCode := 0
	slaPassed := true
	metrics := model.RunMetrics{
		TotalIterations: 1000,
		TotalRequests:   5000,
		AvgTPS:          250.0,
		P50DurationMs:   5.0,
		P90DurationMs:   10.0,
		P95DurationMs:   15.0,
		P99DurationMs:   25.0,
		ErrorRatePct:    0.1,
	}

	completedRun, err := model.NewTestRunWithID(
		"run-xyz", "suite-1", "art-1", nil, "prof-1", nil,
		model.RunStatusCompleted, "vuhive-job", "vuhive-runners",
		&now, &now, &exitCode, &slaPassed,
		metrics, "runs/run-xyz/summary.json", "runs/run-xyz/run.log",
		[]byte(`{}`), "", now,
	)
	require.NoError(t, err)

	t.Run("success returns 200 with RunResponse and KPIs", func(t *testing.T) {
		mockUC := &mockRunsUseCase{
			getRunFunc: func(ctx context.Context, id string) (*model.TestRun, error) {
				assert.Equal(t, "run-xyz", id)
				return completedRun, nil
			},
		}

		router := rest.SetupRouter(nil, nil, nil, mockUC)
		req := httptest.NewRequest(http.MethodGet, "/api/v1/runs/run-xyz", nil)
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)

		var res rest.RunResponse
		err := json.Unmarshal(resp.Body.Bytes(), &res)
		require.NoError(t, err)
		assert.Equal(t, "run-xyz", res.ID)
		assert.Equal(t, "COMPLETED", res.Status)
		assert.Equal(t, 250.0, res.Metrics.AvgTPS)
		assert.Equal(t, 0.1, res.Metrics.ErrorRatePct)
		assert.Equal(t, 0, *res.ExitCode)
		assert.True(t, *res.SLAPassed)
	})

	t.Run("not found returns 404", func(t *testing.T) {
		mockUC := &mockRunsUseCase{
			getRunFunc: func(ctx context.Context, id string) (*model.TestRun, error) {
				return nil, model.ErrNotFound
			},
		}

		router := rest.SetupRouter(nil, nil, nil, mockUC)
		req := httptest.NewRequest(http.MethodGet, "/api/v1/runs/not-found", nil)
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusNotFound, resp.Code)
	})
}

func TestRunHandler_ListRuns(t *testing.T) {
	now := time.Now().UTC()
	run1, err := model.NewTestRunWithID(
		"run-1", "suite-1", "art-1", nil, "prof-1", nil,
		model.RunStatusCompleted, "", "", &now, &now, nil, nil,
		model.RunMetrics{}, "", "", nil, "", now,
	)
	require.NoError(t, err)

	t.Run("success returns 200 with RunListResponse", func(t *testing.T) {
		mockUC := &mockRunsUseCase{
			listRunsFunc: func(ctx context.Context, filter model.RunFilter) ([]*model.TestRun, int64, error) {
				assert.Equal(t, "suite-1", filter.SuiteID)
				assert.Equal(t, model.RunStatusCompleted, filter.Status)
				assert.Equal(t, 10, filter.Limit)
				assert.Equal(t, 20, filter.Offset)
				return []*model.TestRun{run1}, 1, nil
			},
		}

		router := rest.SetupRouter(nil, nil, nil, mockUC)
		req := httptest.NewRequest(http.MethodGet, "/api/v1/runs?suite_id=suite-1&status=COMPLETED&limit=10&offset=20", nil)
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)

		var res rest.RunListResponse
		err := json.Unmarshal(resp.Body.Bytes(), &res)
		require.NoError(t, err)
		assert.Equal(t, 1, res.Count)
		assert.Equal(t, int64(1), res.Total)
		assert.Equal(t, 10, res.Limit)
		assert.Equal(t, 20, res.Offset)
		require.Len(t, res.Runs, 1)
		assert.Equal(t, "run-1", res.Runs[0].ID)
	})
}

func TestRunHandler_GetRunReport(t *testing.T) {
	t.Run("returns raw json report", func(t *testing.T) {
		reportData := `{"summary":"ok","avg_tps":1200}`
		mockUC := &mockRunsUseCase{
			getRunReportFunc: func(ctx context.Context, id string) (io.ReadCloser, error) {
				assert.Equal(t, "run-100", id)
				return io.NopCloser(bytes.NewReader([]byte(reportData))), nil
			},
		}

		router := rest.SetupRouter(nil, nil, nil, mockUC)
		req := httptest.NewRequest(http.MethodGet, "/api/v1/runs/run-100/report", nil)
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
		assert.Equal(t, "application/json", resp.Header().Get("Content-Type"))
		assert.Equal(t, reportData, resp.Body.String())
	})

	t.Run("returns presigned URL when presign=true", func(t *testing.T) {
		mockUC := &mockRunsUseCase{
			getRunReportURLFunc: func(ctx context.Context, id string, lifetime time.Duration) (string, error) {
				assert.Equal(t, "run-100", id)
				return "https://s3.amazonaws.com/bucket/runs/run-100/summary.json?signature=xyz", nil
			},
		}

		router := rest.SetupRouter(nil, nil, nil, mockUC)
		req := httptest.NewRequest(http.MethodGet, "/api/v1/runs/run-100/report?presign=true", nil)
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
		var res rest.PresignedURLResponse
		err := json.Unmarshal(resp.Body.Bytes(), &res)
		require.NoError(t, err)
		assert.Contains(t, res.DownloadURL, "https://s3.amazonaws.com")
	})

	t.Run("in-flight run returns 409 Conflict", func(t *testing.T) {
		mockUC := &mockRunsUseCase{
			getRunReportFunc: func(ctx context.Context, id string) (io.ReadCloser, error) {
				return nil, model.ErrRunInFlight
			},
		}

		router := rest.SetupRouter(nil, nil, nil, mockUC)
		req := httptest.NewRequest(http.MethodGet, "/api/v1/runs/run-inflight/report", nil)
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusConflict, resp.Code)
	})

	t.Run("missing report returns 404 Not Found", func(t *testing.T) {
		mockUC := &mockRunsUseCase{
			getRunReportFunc: func(ctx context.Context, id string) (io.ReadCloser, error) {
				return nil, model.ErrReportNotFound
			},
		}

		router := rest.SetupRouter(nil, nil, nil, mockUC)
		req := httptest.NewRequest(http.MethodGet, "/api/v1/runs/run-missing/report", nil)
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusNotFound, resp.Code)
	})
}

func TestRunHandler_GetRunLogs(t *testing.T) {
	t.Run("returns raw text log", func(t *testing.T) {
		logData := "2026-09-05 Test started...\nPass!\n"
		mockUC := &mockRunsUseCase{
			getRunLogsFunc: func(ctx context.Context, id string) (io.ReadCloser, error) {
				assert.Equal(t, "run-200", id)
				return io.NopCloser(bytes.NewReader([]byte(logData))), nil
			},
		}

		router := rest.SetupRouter(nil, nil, nil, mockUC)
		req := httptest.NewRequest(http.MethodGet, "/api/v1/runs/run-200/logs", nil)
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
		assert.Contains(t, resp.Header().Get("Content-Type"), "text/plain")
		assert.Equal(t, logData, resp.Body.String())
	})

	t.Run("returns presigned URL when presign=true", func(t *testing.T) {
		mockUC := &mockRunsUseCase{
			getRunLogsURLFunc: func(ctx context.Context, id string, lifetime time.Duration) (string, error) {
				assert.Equal(t, "run-200", id)
				return "https://s3.amazonaws.com/bucket/runs/run-200/run.log?signature=xyz", nil
			},
		}

		router := rest.SetupRouter(nil, nil, nil, mockUC)
		req := httptest.NewRequest(http.MethodGet, "/api/v1/runs/run-200/logs?presign=true", nil)
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
		var res rest.PresignedURLResponse
		err := json.Unmarshal(resp.Body.Bytes(), &res)
		require.NoError(t, err)
		assert.Contains(t, res.DownloadURL, "https://s3.amazonaws.com")
	})

	t.Run("in-flight run returns 409 Conflict", func(t *testing.T) {
		mockUC := &mockRunsUseCase{
			getRunLogsFunc: func(ctx context.Context, id string) (io.ReadCloser, error) {
				return nil, model.ErrRunInFlight
			},
		}

		router := rest.SetupRouter(nil, nil, nil, mockUC)
		req := httptest.NewRequest(http.MethodGet, "/api/v1/runs/run-inflight/logs", nil)
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusConflict, resp.Code)
	})

	t.Run("missing logs returns 404 Not Found", func(t *testing.T) {
		mockUC := &mockRunsUseCase{
			getRunLogsFunc: func(ctx context.Context, id string) (io.ReadCloser, error) {
				return nil, model.ErrLogsNotFound
			},
		}

		router := rest.SetupRouter(nil, nil, nil, mockUC)
		req := httptest.NewRequest(http.MethodGet, "/api/v1/runs/run-missing/logs", nil)
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusNotFound, resp.Code)
	})
}

func TestRunHandler_TriggerRun(t *testing.T) {
	now := time.Now().UTC()
	configID := "cfg-456"

	queuedRun, err := model.NewTestRunWithID(
		"run-999", "suite-100", "art-200", &configID, "prof-300", nil,
		model.RunStatusQueued, "vuhive-job-999", "vuhive-runners",
		nil, nil, nil, nil,
		model.RunMetrics{}, "", "", nil, "", now,
	)
	require.NoError(t, err)

	queuedRunNoConfig, err := model.NewTestRunWithID(
		"run-1000", "suite-100", "art-200", nil, "prof-300", nil,
		model.RunStatusQueued, "vuhive-job-1000", "vuhive-runners",
		nil, nil, nil, nil,
		model.RunMetrics{}, "", "", nil, "", now,
	)
	require.NoError(t, err)

	t.Run("successful trigger run with all fields returns HTTP 201 Created", func(t *testing.T) {
		timeoutOverride := int64(1800)
		mockUC := &mockRunsUseCase{
			triggerRunFunc: func(ctx context.Context, cmd inbound.TriggerRunCommand) (*model.TestRun, error) {
				assert.Equal(t, "suite-100", cmd.SuiteID)
				assert.Equal(t, "art-200", cmd.ArtifactID)
				require.NotNil(t, cmd.ConfigurationID)
				assert.Equal(t, "cfg-456", *cmd.ConfigurationID)
				assert.Equal(t, "prof-300", cmd.RunnerProfileID)
				require.NotNil(t, cmd.ActiveDeadlineSeconds)
				assert.Equal(t, int64(1800), *cmd.ActiveDeadlineSeconds)
				return queuedRun, nil
			},
		}

		router := rest.SetupRouter(nil, nil, nil, mockUC)

		reqBody := rest.TriggerRunRequest{
			SuiteID:               "suite-100",
			ArtifactID:            "art-200",
			ConfigurationID:       &configID,
			RunnerProfileID:       "prof-300",
			ActiveDeadlineSeconds: &timeoutOverride,
		}
		raw, err := json.Marshal(reqBody)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/runs", bytes.NewReader(raw))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusCreated, resp.Code)

		var res rest.RunResponse
		require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &res))
		assert.Equal(t, "run-999", res.ID)
		assert.Equal(t, "suite-100", res.SuiteID)
		assert.Equal(t, "art-200", res.ArtifactID)
		require.NotNil(t, res.ConfigurationID)
		assert.Equal(t, "cfg-456", *res.ConfigurationID)
		assert.Equal(t, "prof-300", res.RunnerProfileID)
		assert.Equal(t, "QUEUED", res.Status)
		assert.Equal(t, "vuhive-job-999", res.K8sJobName)
	})

	t.Run("successful trigger run with optional runner_namespace passes override to use case", func(t *testing.T) {
		mockUC := &mockRunsUseCase{
			triggerRunFunc: func(ctx context.Context, cmd inbound.TriggerRunCommand) (*model.TestRun, error) {
				assert.Equal(t, "suite-100", cmd.SuiteID)
				assert.Equal(t, "art-200", cmd.ArtifactID)
				assert.Equal(t, "prof-300", cmd.RunnerProfileID)
				require.NotNil(t, cmd.RunnerNamespace)
				assert.Equal(t, "ephemeral-test-ns", *cmd.RunnerNamespace)
				return queuedRunNoConfig, nil
			},
		}

		router := rest.SetupRouter(nil, nil, nil, mockUC)

		body := map[string]interface{}{
			"suite_id":          "suite-100",
			"artifact_id":       "art-200",
			"runner_profile_id": "prof-300",
			"runner_namespace":  "ephemeral-test-ns",
		}
		raw, err := json.Marshal(body)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/runs", bytes.NewReader(raw))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)
		assert.Equal(t, http.StatusCreated, resp.Code)
	})

	t.Run("successful trigger run without optional configuration_id returns HTTP 201 Created", func(t *testing.T) {
		mockUC := &mockRunsUseCase{
			triggerRunFunc: func(ctx context.Context, cmd inbound.TriggerRunCommand) (*model.TestRun, error) {
				assert.Equal(t, "suite-100", cmd.SuiteID)
				assert.Equal(t, "art-200", cmd.ArtifactID)
				assert.Nil(t, cmd.ConfigurationID)
				assert.Equal(t, "prof-300", cmd.RunnerProfileID)
				return queuedRunNoConfig, nil
			},
		}

		router := rest.SetupRouter(nil, nil, nil, mockUC)

		body := map[string]interface{}{
			"suite_id":          "suite-100",
			"artifact_id":       "art-200",
			"runner_profile_id": "prof-300",
		}
		raw, err := json.Marshal(body)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/runs", bytes.NewReader(raw))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusCreated, resp.Code)

		var res rest.RunResponse
		require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &res))
		assert.Equal(t, "run-1000", res.ID)
		assert.Nil(t, res.ConfigurationID)
		assert.Equal(t, "QUEUED", res.Status)
	})

	t.Run("missing required fields returns HTTP 400 Bad Request", func(t *testing.T) {
		router := rest.SetupRouter(nil, nil, nil, &mockRunsUseCase{})

		testCases := []struct {
			name string
			body map[string]interface{}
		}{
			{
				name: "missing suite_id",
				body: map[string]interface{}{
					"artifact_id":       "art-200",
					"runner_profile_id": "prof-300",
				},
			},
			{
				name: "missing artifact_id",
				body: map[string]interface{}{
					"suite_id":          "suite-100",
					"runner_profile_id": "prof-300",
				},
			},
			{
				name: "missing runner_profile_id",
				body: map[string]interface{}{
					"suite_id":    "suite-100",
					"artifact_id": "art-200",
				},
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				raw, err := json.Marshal(tc.body)
				require.NoError(t, err)

				req := httptest.NewRequest(http.MethodPost, "/api/v1/runs", bytes.NewReader(raw))
				req.Header.Set("Content-Type", "application/json")
				resp := httptest.NewRecorder()

				router.ServeHTTP(resp, req)

				assert.Equal(t, http.StatusBadRequest, resp.Code)
				var errRes rest.ErrorResponse
				require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &errRes))
				assert.NotEmpty(t, errRes.Error)
			})
		}
	})

	t.Run("malformed json payload returns HTTP 400 Bad Request", func(t *testing.T) {
		router := rest.SetupRouter(nil, nil, nil, &mockRunsUseCase{})

		req := httptest.NewRequest(http.MethodPost, "/api/v1/runs", bytes.NewReader([]byte(`{not-valid-json`)))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusBadRequest, resp.Code)
	})

	t.Run("inactive suite returns HTTP 400 Bad Request", func(t *testing.T) {
		mockUC := &mockRunsUseCase{
			triggerRunFunc: func(ctx context.Context, cmd inbound.TriggerRunCommand) (*model.TestRun, error) {
				return nil, model.ErrInvalidStateTransition
			},
		}

		router := rest.SetupRouter(nil, nil, nil, mockUC)

		reqBody := rest.TriggerRunRequest{
			SuiteID:         "suite-100",
			ArtifactID:      "art-200",
			RunnerProfileID: "prof-300",
		}
		raw, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/runs", bytes.NewReader(raw))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusBadRequest, resp.Code)
	})

	t.Run("suite or artifact not found returns HTTP 404 Not Found", func(t *testing.T) {
		mockUC := &mockRunsUseCase{
			triggerRunFunc: func(ctx context.Context, cmd inbound.TriggerRunCommand) (*model.TestRun, error) {
				return nil, model.ErrNotFound
			},
		}

		router := rest.SetupRouter(nil, nil, nil, mockUC)

		reqBody := rest.TriggerRunRequest{
			SuiteID:         "suite-100",
			ArtifactID:      "art-200",
			RunnerProfileID: "prof-300",
		}
		raw, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/runs", bytes.NewReader(raw))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusNotFound, resp.Code)
	})

	t.Run("conflict error returns HTTP 409 Conflict", func(t *testing.T) {
		mockUC := &mockRunsUseCase{
			triggerRunFunc: func(ctx context.Context, cmd inbound.TriggerRunCommand) (*model.TestRun, error) {
				return nil, model.ErrConflict
			},
		}

		router := rest.SetupRouter(nil, nil, nil, mockUC)

		reqBody := rest.TriggerRunRequest{
			SuiteID:         "suite-100",
			ArtifactID:      "art-200",
			RunnerProfileID: "prof-300",
		}
		raw, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/runs", bytes.NewReader(raw))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusConflict, resp.Code)
	})
}

