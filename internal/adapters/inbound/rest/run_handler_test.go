package rest_test

import (
	"bytes"
	"context"
	"encoding/json"
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
	triggerRunFunc  func(ctx context.Context, cmd inbound.TriggerRunCommand) (*model.TestRun, error)
	getRunFunc      func(ctx context.Context, id string) (*model.TestRun, error)
	listRunsFunc    func(ctx context.Context, suiteID string, status model.RunStatus) ([]*model.TestRun, error)
	abortRunFunc    func(ctx context.Context, id string, reason string) error
	completeRunFunc func(ctx context.Context, cmd inbound.CompleteRunCommand) (*model.TestRun, error)
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
func (m *mockRunsUseCase) ListRuns(ctx context.Context, suiteID string, status model.RunStatus) ([]*model.TestRun, error) {
	if m.listRunsFunc != nil {
		return m.listRunsFunc(ctx, suiteID, status)
	}
	return nil, nil
}
func (m *mockRunsUseCase) AbortRun(ctx context.Context, id string, reason string) error {
	if m.abortRunFunc != nil {
		return m.abortRunFunc(ctx, id, reason)
	}
	return nil
}
func (m *mockRunsUseCase) CompleteRun(ctx context.Context, cmd inbound.CompleteRunCommand) (*model.TestRun, error) {
	if m.completeRunFunc != nil {
		return m.completeRunFunc(ctx, cmd)
	}
	return nil, nil
}

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
