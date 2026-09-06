package rest_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/morphy76/vuhive-cloud/internal/adapters/inbound/rest"
	"github.com/morphy76/vuhive-cloud/internal/application/ports/inbound"
	"github.com/morphy76/vuhive-cloud/internal/domain/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type MockHousekeepingUseCase struct {
	mock.Mock
}

func (m *MockHousekeepingUseCase) RunHousekeeping(ctx context.Context, cmd inbound.HousekeepingCommand) (*inbound.HousekeepingResult, error) {
	args := m.Called(ctx, cmd)
	if r := args.Get(0); r != nil {
		return r.(*inbound.HousekeepingResult), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockHousekeepingUseCase) GetGlobalPolicy(ctx context.Context) model.RetentionPolicy {
	args := m.Called(ctx)
	return args.Get(0).(model.RetentionPolicy)
}

func (m *MockHousekeepingUseCase) ConfigureBucketLifecycle(ctx context.Context) error {
	return m.Called(ctx).Error(0)
}

func TestHousekeepingHandler_RunHousekeeping(t *testing.T) {
	mockUC := new(MockHousekeepingUseCase)
	router := rest.SetupRouterWithAll(nil, nil, nil, nil, nil, mockUC)

	t.Run("execute housekeeping successfully", func(t *testing.T) {
		expectedResult := &inbound.HousekeepingResult{
			PurgedLogsCount:      5,
			PurgedReportsCount:   2,
			PrunedRunsCount:      3,
			ArchivedRunsCount:    2,
			PurgedArtifactsCount: 1,
			DurationMs:           12,
		}

		mockUC.On("RunHousekeeping", mock.Anything, inbound.HousekeepingCommand{DryRun: false}).
			Return(expectedResult, nil).
			Once()

		req, err := http.NewRequest(http.MethodPost, "/api/v1/system/housekeeping", nil)
		require.NoError(t, err)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp rest.HousekeepingResponse
		err = json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, int64(5), resp.PurgedLogsCount)
		assert.Equal(t, int64(2), resp.PurgedReportsCount)
		assert.Equal(t, int64(3), resp.PrunedRunsCount)
		mockUC.AssertExpectations(t)
	})

	t.Run("execute housekeeping with dry_run query param", func(t *testing.T) {
		mockUC.On("RunHousekeeping", mock.Anything, inbound.HousekeepingCommand{DryRun: true}).
			Return(&inbound.HousekeepingResult{PurgedLogsCount: 10}, nil).
			Once()

		req, err := http.NewRequest(http.MethodPost, "/api/v1/system/housekeeping?dry_run=true", nil)
		require.NoError(t, err)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp rest.HousekeepingResponse
		err = json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, int64(10), resp.PurgedLogsCount)
		mockUC.AssertExpectations(t)
	})

	t.Run("execute housekeeping with json body", func(t *testing.T) {
		suiteID := "suite-123"
		body := rest.HousekeepingRequest{
			SuiteID: &suiteID,
			DryRun:  true,
		}
		jsonBytes, err := json.Marshal(body)
		require.NoError(t, err)

		mockUC.On("RunHousekeeping", mock.Anything, inbound.HousekeepingCommand{
			SuiteID: &suiteID,
			DryRun:  true,
		}).Return(&inbound.HousekeepingResult{PurgedLogsCount: 1}, nil).Once()

		req, err := http.NewRequest(http.MethodPost, "/api/v1/system/housekeeping", bytes.NewReader(jsonBytes))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		mockUC.AssertExpectations(t)
	})
}

func TestHousekeepingHandler_GetPolicy(t *testing.T) {
	mockUC := new(MockHousekeepingUseCase)
	router := rest.SetupRouterWithAll(nil, nil, nil, nil, nil, mockUC)

	t.Run("get global retention policy successfully", func(t *testing.T) {
		policy := model.DefaultRetentionPolicy()
		mockUC.On("GetGlobalPolicy", mock.Anything).Return(policy).Once()

		req, err := http.NewRequest(http.MethodGet, "/api/v1/system/housekeeping/policy", nil)
		require.NoError(t, err)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp rest.RetentionPolicyResponse
		err = json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, 14, resp.LogsTTLDays)
		assert.Equal(t, 90, resp.ReportsTTLDays)
		assert.Equal(t, 180, resp.RunsTTLDays)
		mockUC.AssertExpectations(t)
	})
}
