package rest_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/morphy76/vuhive-cloud/internal/adapters/inbound/rest"
	"github.com/morphy76/vuhive-cloud/internal/application/ports/inbound"
	"github.com/morphy76/vuhive-cloud/internal/domain/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockBarrierUseCase struct {
	awaitFn  func(ctx context.Context, cmd inbound.AwaitRendezvousCommand) (*inbound.RendezvousResult, error)
	abortFn  func(ctx context.Context, cmd inbound.SignalAbortCommand) error
	statusFn func(ctx context.Context, runID string) (*model.BarrierSession, error)
}

func (m *mockBarrierUseCase) AwaitRendezvous(ctx context.Context, cmd inbound.AwaitRendezvousCommand) (*inbound.RendezvousResult, error) {
	if m.awaitFn != nil {
		return m.awaitFn(ctx, cmd)
	}
	return nil, nil
}

func (m *mockBarrierUseCase) SignalAbort(ctx context.Context, cmd inbound.SignalAbortCommand) error {
	if m.abortFn != nil {
		return m.abortFn(ctx, cmd)
	}
	return nil
}

func (m *mockBarrierUseCase) GetBarrierStatus(ctx context.Context, runID string) (*model.BarrierSession, error) {
	if m.statusFn != nil {
		return m.statusFn(ctx, runID)
	}
	return nil, nil
}

var _ inbound.BarrierUseCase = (*mockBarrierUseCase)(nil)

func setupTestRouterWithBarrier(mockUC inbound.BarrierUseCase) *gin.Engine {
	gin.SetMode(gin.TestMode)
	return rest.SetupRouterWithBarrier(nil, nil, nil, nil, mockUC)
}

func TestBarrierHandler_AwaitBarrier(t *testing.T) {
	t.Parallel()

	t.Run("successfully awaits barrier and returns release response", func(t *testing.T) {
		t.Parallel()

		targetTime := time.Now().Add(300 * time.Millisecond)
		mockUC := &mockBarrierUseCase{
			awaitFn: func(ctx context.Context, cmd inbound.AwaitRendezvousCommand) (*inbound.RendezvousResult, error) {
				assert.Equal(t, "run-123", cmd.RunID)
				assert.Equal(t, "worker-1", cmd.WorkerID)
				assert.Equal(t, 2, cmd.TotalWorkers)
				return &inbound.RendezvousResult{
					Status:          model.BarrierStatusReleased,
					RunID:           cmd.RunID,
					WorkerID:        cmd.WorkerID,
					TotalWorkers:    cmd.TotalWorkers,
					TargetStartTime: targetTime,
					StartIn:         250 * time.Millisecond,
				}, nil
			},
		}

		router := setupTestRouterWithBarrier(mockUC)

		body, _ := json.Marshal(rest.BarrierAwaitRequest{
			WorkerID:     "worker-1",
			TotalWorkers: 2,
		})

		req := httptest.NewRequest(http.MethodPost, "/api/v1/runs/run-123/barrier/await", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		var resp rest.BarrierResponse
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
		assert.Equal(t, "RELEASED", resp.Status)
		assert.Equal(t, "run-123", resp.RunID)
		assert.Equal(t, "worker-1", resp.WorkerID)
		assert.Equal(t, 2, resp.TotalWorkers)
		assert.NotNil(t, resp.TargetStartTime)
	})

	t.Run("returns 424 Failed Dependency when barrier aborted", func(t *testing.T) {
		t.Parallel()

		mockUC := &mockBarrierUseCase{
			awaitFn: func(ctx context.Context, cmd inbound.AwaitRendezvousCommand) (*inbound.RendezvousResult, error) {
				return nil, model.ErrBarrierAborted
			},
		}

		router := setupTestRouterWithBarrier(mockUC)

		body, _ := json.Marshal(rest.BarrierAwaitRequest{
			WorkerID:     "worker-1",
			TotalWorkers: 2,
		})

		req := httptest.NewRequest(http.MethodPost, "/api/v1/runs/run-123/barrier/await", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusFailedDependency, rec.Code)
	})

	t.Run("returns 408 Request Timeout when barrier times out", func(t *testing.T) {
		t.Parallel()

		mockUC := &mockBarrierUseCase{
			awaitFn: func(ctx context.Context, cmd inbound.AwaitRendezvousCommand) (*inbound.RendezvousResult, error) {
				return nil, model.ErrBarrierTimeout
			},
		}

		router := setupTestRouterWithBarrier(mockUC)

		body, _ := json.Marshal(rest.BarrierAwaitRequest{
			WorkerID:     "worker-1",
			TotalWorkers: 2,
		})

		req := httptest.NewRequest(http.MethodPost, "/api/v1/runs/run-123/barrier/await", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusRequestTimeout, rec.Code)
	})
}

func TestBarrierHandler_AbortBarrier(t *testing.T) {
	t.Parallel()

	mockUC := &mockBarrierUseCase{
		abortFn: func(ctx context.Context, cmd inbound.SignalAbortCommand) error {
			assert.Equal(t, "run-abort", cmd.RunID)
			assert.Equal(t, "worker-x", cmd.WorkerID)
			assert.Equal(t, "preflight check failed", cmd.Reason)
			return nil
		},
	}

	router := setupTestRouterWithBarrier(mockUC)

	body, _ := json.Marshal(rest.BarrierAbortRequest{
		WorkerID: "worker-x",
		Reason:   "preflight check failed",
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/runs/run-abort/barrier/abort", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestBarrierHandler_GetBarrier(t *testing.T) {
	t.Parallel()

	session, err := model.NewBarrierSession("run-info", 3, 10*time.Second, 200*time.Millisecond)
	require.NoError(t, err)
	require.NoError(t, session.RegisterWorker("w-1"))

	mockUC := &mockBarrierUseCase{
		statusFn: func(ctx context.Context, runID string) (*model.BarrierSession, error) {
			assert.Equal(t, "run-info", runID)
			return session, nil
		},
	}

	router := setupTestRouterWithBarrier(mockUC)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/runs/run-info/barrier", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	var resp rest.BarrierResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "run-info", resp.RunID)
	assert.Equal(t, 3, resp.TotalWorkers)
	assert.Equal(t, "PENDING", resp.Status)
	assert.Len(t, resp.Participants, 1)
}
