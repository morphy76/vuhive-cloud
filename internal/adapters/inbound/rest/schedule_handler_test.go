package rest_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/morphy76/vuhive-cloud/internal/adapters/inbound/rest"
	"github.com/morphy76/vuhive-cloud/internal/domain/model"
)

type mockSchedulesUseCase struct {
	createFunc func(ctx context.Context, suiteID, artifactID string, configID *string, runnerProfileID, name, cronExpr string) (*model.Schedule, error)
	getFunc    func(ctx context.Context, id string) (*model.Schedule, error)
	listFunc   func(ctx context.Context) ([]*model.Schedule, error)
	updateFunc func(ctx context.Context, id string, cronExpr *string, isActive *bool) (*model.Schedule, error)
	deleteFunc func(ctx context.Context, id string) error
}

func (m *mockSchedulesUseCase) CreateSchedule(ctx context.Context, suiteID, artifactID string, configID *string, runnerProfileID, name, cronExpr string) (*model.Schedule, error) {
	if m.createFunc != nil {
		return m.createFunc(ctx, suiteID, artifactID, configID, runnerProfileID, name, cronExpr)
	}
	return nil, nil
}

func (m *mockSchedulesUseCase) GetSchedule(ctx context.Context, id string) (*model.Schedule, error) {
	if m.getFunc != nil {
		return m.getFunc(ctx, id)
	}
	return nil, nil
}

func (m *mockSchedulesUseCase) ListSchedules(ctx context.Context) ([]*model.Schedule, error) {
	if m.listFunc != nil {
		return m.listFunc(ctx)
	}
	return nil, nil
}

func (m *mockSchedulesUseCase) UpdateSchedule(ctx context.Context, id string, cronExpr *string, isActive *bool) (*model.Schedule, error) {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, id, cronExpr, isActive)
	}
	return nil, nil
}

func (m *mockSchedulesUseCase) DeleteSchedule(ctx context.Context, id string) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, id)
	}
	return nil
}

func sampleSchedule(t *testing.T) *model.Schedule {
	s, err := model.NewSchedule("suite-1", "art-1", nil, "prof-1", "nightly", "0 2 * * *")
	require.NoError(t, err)
	return s
}

func TestScheduleHandler_CreateSchedule(t *testing.T) {
	t.Run("successfully creates schedule", func(t *testing.T) {
		s := sampleSchedule(t)
		mockUC := &mockSchedulesUseCase{
			createFunc: func(_ context.Context, suiteID, artifactID string, configID *string, runnerProfileID, name, cronExpr string) (*model.Schedule, error) {
				return s, nil
			},
		}

		router := rest.SetupRouter(nil, nil, mockUC, nil)

		payload := rest.CreateScheduleRequest{
			SuiteID:         "suite-1",
			ArtifactID:      "art-1",
			RunnerProfileID: "prof-1",
			Name:            "nightly",
			CronExpression:  "0 2 * * *",
		}
		body, err := json.Marshal(payload)
		require.NoError(t, err)

		req, err := http.NewRequest(http.MethodPost, "/api/v1/schedules", bytes.NewReader(body))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusCreated, rec.Code)

		var resp rest.ScheduleResponse
		err = json.Unmarshal(rec.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, s.ID(), resp.ID)
		assert.Equal(t, "nightly", resp.Name)
		assert.Equal(t, "0 2 * * *", resp.CronExpression)
	})

	t.Run("returns 400 on invalid JSON or missing fields", func(t *testing.T) {
		mockUC := &mockSchedulesUseCase{}
		router := rest.SetupRouter(nil, nil, mockUC, nil)

		req, err := http.NewRequest(http.MethodPost, "/api/v1/schedules", bytes.NewReader([]byte(`{"invalid":`)))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("returns 404 when dependency is not found", func(t *testing.T) {
		mockUC := &mockSchedulesUseCase{
			createFunc: func(_ context.Context, _, _ string, _ *string, _, _, _ string) (*model.Schedule, error) {
				return nil, model.ErrNotFound
			},
		}
		router := rest.SetupRouter(nil, nil, mockUC, nil)

		payload := rest.CreateScheduleRequest{
			SuiteID:         "suite-1",
			ArtifactID:      "art-1",
			RunnerProfileID: "prof-1",
			Name:            "nightly",
			CronExpression:  "0 2 * * *",
		}
		body, _ := json.Marshal(payload)

		req, err := http.NewRequest(http.MethodPost, "/api/v1/schedules", bytes.NewReader(body))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusNotFound, rec.Code)
	})
}

func TestScheduleHandler_GetSchedule(t *testing.T) {
	t.Run("successfully retrieves schedule by ID", func(t *testing.T) {
		s := sampleSchedule(t)
		mockUC := &mockSchedulesUseCase{
			getFunc: func(_ context.Context, id string) (*model.Schedule, error) {
				if id == s.ID() {
					return s, nil
				}
				return nil, model.ErrNotFound
			},
		}

		router := rest.SetupRouter(nil, nil, mockUC, nil)

		req, err := http.NewRequest(http.MethodGet, "/api/v1/schedules/"+s.ID(), nil)
		require.NoError(t, err)

		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)

		var resp rest.ScheduleResponse
		err = json.Unmarshal(rec.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, s.ID(), resp.ID)
	})

	t.Run("returns 404 when schedule not found", func(t *testing.T) {
		mockUC := &mockSchedulesUseCase{
			getFunc: func(_ context.Context, _ string) (*model.Schedule, error) {
				return nil, model.ErrNotFound
			},
		}

		router := rest.SetupRouter(nil, nil, mockUC, nil)

		req, err := http.NewRequest(http.MethodGet, "/api/v1/schedules/unknown", nil)
		require.NoError(t, err)

		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusNotFound, rec.Code)
	})
}

func TestScheduleHandler_ListSchedules(t *testing.T) {
	t.Run("successfully lists active schedules", func(t *testing.T) {
		s := sampleSchedule(t)
		mockUC := &mockSchedulesUseCase{
			listFunc: func(_ context.Context) ([]*model.Schedule, error) {
				return []*model.Schedule{s}, nil
			},
		}

		router := rest.SetupRouter(nil, nil, mockUC, nil)

		req, err := http.NewRequest(http.MethodGet, "/api/v1/schedules", nil)
		require.NoError(t, err)

		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)

		var resp rest.ScheduleListResponse
		err = json.Unmarshal(rec.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, 1, resp.Count)
		assert.Equal(t, s.ID(), resp.Schedules[0].ID)
	})
}

func TestScheduleHandler_UpdateSchedule(t *testing.T) {
	t.Run("successfully updates schedule cron expression", func(t *testing.T) {
		s := sampleSchedule(t)
		mockUC := &mockSchedulesUseCase{
			updateFunc: func(_ context.Context, id string, cronExpr *string, isActive *bool) (*model.Schedule, error) {
				if cronExpr != nil {
					_ = s.UpdateCronExpression(*cronExpr)
				}
				return s, nil
			},
		}

		router := rest.SetupRouter(nil, nil, mockUC, nil)

		cronExpr := "*/10 * * * *"
		payload := rest.UpdateScheduleRequest{
			CronExpression: &cronExpr,
		}
		body, _ := json.Marshal(payload)

		req, err := http.NewRequest(http.MethodPut, "/api/v1/schedules/"+s.ID(), bytes.NewReader(body))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)

		var resp rest.ScheduleResponse
		err = json.Unmarshal(rec.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "*/10 * * * *", resp.CronExpression)
	})

	t.Run("successfully pauses and resumes schedule via is_active", func(t *testing.T) {
		s := sampleSchedule(t)
		mockUC := &mockSchedulesUseCase{
			updateFunc: func(_ context.Context, id string, cronExpr *string, isActive *bool) (*model.Schedule, error) {
				if isActive != nil {
					if *isActive {
						_ = s.Activate()
					} else {
						_ = s.Deactivate()
					}
				}
				return s, nil
			},
		}

		router := rest.SetupRouter(nil, nil, mockUC, nil)

		// Pause
		activeFalse := false
		body, _ := json.Marshal(rest.UpdateScheduleRequest{IsActive: &activeFalse})
		req, _ := http.NewRequest(http.MethodPut, "/api/v1/schedules/"+s.ID(), bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		var resp rest.ScheduleResponse
		_ = json.Unmarshal(rec.Body.Bytes(), &resp)
		assert.False(t, resp.IsActive)
	})

	t.Run("returns 400 when neither cron_expression nor is_active is provided", func(t *testing.T) {
		mockUC := &mockSchedulesUseCase{}
		router := rest.SetupRouter(nil, nil, mockUC, nil)

		payload := rest.UpdateScheduleRequest{}
		body, _ := json.Marshal(payload)

		req, _ := http.NewRequest(http.MethodPut, "/api/v1/schedules/sched-123", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("returns 404 when updating non-existent schedule", func(t *testing.T) {
		mockUC := &mockSchedulesUseCase{
			updateFunc: func(_ context.Context, _ string, _ *string, _ *bool) (*model.Schedule, error) {
				return nil, model.ErrNotFound
			},
		}

		router := rest.SetupRouter(nil, nil, mockUC, nil)

		cronExpr := "0 0 * * *"
		payload := rest.UpdateScheduleRequest{CronExpression: &cronExpr}
		body, _ := json.Marshal(payload)

		req, err := http.NewRequest(http.MethodPut, "/api/v1/schedules/unknown", bytes.NewReader(body))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusNotFound, rec.Code)
	})
}

func TestScheduleHandler_DeleteSchedule(t *testing.T) {
	t.Run("successfully deletes schedule", func(t *testing.T) {
		mockUC := &mockSchedulesUseCase{
			deleteFunc: func(_ context.Context, _ string) error {
				return nil
			},
		}

		router := rest.SetupRouter(nil, nil, mockUC, nil)

		req, err := http.NewRequest(http.MethodDelete, "/api/v1/schedules/sched-123", nil)
		require.NoError(t, err)

		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusNoContent, rec.Code)
	})

	t.Run("returns 404 when deleting non-existent schedule", func(t *testing.T) {
		mockUC := &mockSchedulesUseCase{
			deleteFunc: func(_ context.Context, _ string) error {
				return model.ErrNotFound
			},
		}

		router := rest.SetupRouter(nil, nil, mockUC, nil)

		req, err := http.NewRequest(http.MethodDelete, "/api/v1/schedules/unknown", nil)
		require.NoError(t, err)

		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusNotFound, rec.Code)
	})
}
