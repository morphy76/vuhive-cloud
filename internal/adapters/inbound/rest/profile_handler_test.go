package rest_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/morphy76/vuhive-cloud/internal/adapters/inbound/rest"
	"github.com/morphy76/vuhive-cloud/internal/application/ports/inbound"
	"github.com/morphy76/vuhive-cloud/internal/domain/model"
)

// MockProfilesUseCase mocks inbound.ProfilesUseCase
type MockProfilesUseCase struct {
	mock.Mock
}

func (m *MockProfilesUseCase) CreateProfile(ctx context.Context, cmd inbound.CreateProfileCommand) (*model.RunnerProfile, error) {
	args := m.Called(ctx, cmd)
	if p := args.Get(0); p != nil {
		return p.(*model.RunnerProfile), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockProfilesUseCase) GetProfile(ctx context.Context, id string) (*model.RunnerProfile, error) {
	args := m.Called(ctx, id)
	if p := args.Get(0); p != nil {
		return p.(*model.RunnerProfile), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockProfilesUseCase) ListProfiles(ctx context.Context) ([]*model.RunnerProfile, error) {
	args := m.Called(ctx)
	if p := args.Get(0); p != nil {
		return p.([]*model.RunnerProfile), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockProfilesUseCase) UpdateProfile(ctx context.Context, id string, cmd inbound.UpdateProfileCommand) (*model.RunnerProfile, error) {
	args := m.Called(ctx, id, cmd)
	if p := args.Get(0); p != nil {
		return p.(*model.RunnerProfile), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockProfilesUseCase) DeleteProfile(ctx context.Context, id string) error {
	return m.Called(ctx, id).Error(0)
}

var _ inbound.ProfilesUseCase = (*MockProfilesUseCase)(nil)

func createSampleProfile(t *testing.T, id, name string) *model.RunnerProfile {
	t.Helper()
	res, err := model.NewResourceRequirements("1000m", "2000m", "1Gi", "2Gi")
	require.NoError(t, err)

	now := time.Now().UTC()
	term := model.NodeAffinityTerm{Key: "zone", Operator: "In", Values: []string{"zone-a"}}
	affinity := model.Affinity{NodeSelectorTerms: []model.NodeAffinityTerm{term}}
	toleration := model.Toleration{Key: "dedicated", Operator: "Exists", Effect: "NoSchedule"}

	p, err := model.NewRunnerProfileWithID(
		id, name, "Profile description", "alpine:3.20",
		res, map[string]string{"kubernetes.io/os": "linux"},
		affinity, []model.Toleration{toleration},
		now, now,
	)
	require.NoError(t, err)
	return p
}

func TestProfileHandler_CreateProfile(t *testing.T) {
	t.Run("successfully create profile returns 201 Created", func(t *testing.T) {
		mockProfilesUC := new(MockProfilesUseCase)
		router := rest.SetupRouter(nil, mockProfilesUC, nil, nil)

		sampleProfile := createSampleProfile(t, "prof-1", "high-perf")
		mockProfilesUC.On("CreateProfile", mock.Anything, mock.MatchedBy(func(cmd inbound.CreateProfileCommand) bool {
			return cmd.Name == "high-perf" && cmd.RunnerImage == "custom-image:v1"
		})).Return(sampleProfile, nil)

		reqBody := rest.CreateProfileRequest{
			Name:          "high-perf",
			Description:   "Profile description",
			RunnerImage:   "custom-image:v1",
			CPURequest:    "1000m",
			CPULimit:      "2000m",
			MemoryRequest: "1Gi",
			MemoryLimit:   "2Gi",
			NodeSelector:  map[string]string{"kubernetes.io/os": "linux"},
			Affinity: &rest.AffinityDTO{
				NodeSelectorTerms: []rest.NodeAffinityTermDTO{
					{Key: "zone", Operator: "In", Values: []string{"zone-a"}},
				},
			},
			Tolerations: []rest.TolerationDTO{
				{Key: "dedicated", Operator: "Exists", Effect: "NoSchedule"},
			},
		}
		raw, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/profiles", bytes.NewReader(raw))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
		var resp rest.ProfileResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "prof-1", resp.ID)
		assert.Equal(t, "high-perf", resp.Name)
		assert.Equal(t, "1000m", resp.CPURequest)
		mockProfilesUC.AssertExpectations(t)
	})

	t.Run("fail with invalid json body returns 400", func(t *testing.T) {
		mockProfilesUC := new(MockProfilesUseCase)
		router := rest.SetupRouter(nil, mockProfilesUC, nil, nil)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/profiles", bytes.NewReader([]byte("{invalid-json")))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("fail with missing required name returns 400", func(t *testing.T) {
		mockProfilesUC := new(MockProfilesUseCase)
		router := rest.SetupRouter(nil, mockProfilesUC, nil, nil)

		reqBody := map[string]string{"description": "missing name"}
		raw, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/profiles", bytes.NewReader(raw))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("fail with resource quantity error returns 400", func(t *testing.T) {
		mockProfilesUC := new(MockProfilesUseCase)
		router := rest.SetupRouter(nil, mockProfilesUC, nil, nil)

		mockProfilesUC.On("CreateProfile", mock.Anything, mock.Anything).
			Return(nil, model.ErrInvalidResourceQuantity)

		reqBody := rest.CreateProfileRequest{Name: "bad-res"}
		raw, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/profiles", bytes.NewReader(raw))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		mockProfilesUC.AssertExpectations(t)
	})

	t.Run("fail with conflict on duplicate name returns 409", func(t *testing.T) {
		mockProfilesUC := new(MockProfilesUseCase)
		router := rest.SetupRouter(nil, mockProfilesUC, nil, nil)

		mockProfilesUC.On("CreateProfile", mock.Anything, mock.Anything).
			Return(nil, model.ErrConflict)

		reqBody := rest.CreateProfileRequest{Name: "existing-name"}
		raw, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/profiles", bytes.NewReader(raw))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusConflict, w.Code)
		mockProfilesUC.AssertExpectations(t)
	})

	t.Run("fail with internal server error returns 500", func(t *testing.T) {
		mockProfilesUC := new(MockProfilesUseCase)
		router := rest.SetupRouter(nil, mockProfilesUC, nil, nil)

		mockProfilesUC.On("CreateProfile", mock.Anything, mock.Anything).
			Return(nil, errors.New("db failure"))

		reqBody := rest.CreateProfileRequest{Name: "err-profile"}
		raw, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/profiles", bytes.NewReader(raw))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		mockProfilesUC.AssertExpectations(t)
	})
}

func TestProfileHandler_ListProfiles(t *testing.T) {
	t.Run("successfully list profiles returns 200 OK", func(t *testing.T) {
		mockProfilesUC := new(MockProfilesUseCase)
		router := rest.SetupRouter(nil, mockProfilesUC, nil, nil)

		p1 := createSampleProfile(t, "prof-1", "profile-1")
		p2 := createSampleProfile(t, "prof-2", "profile-2")

		mockProfilesUC.On("ListProfiles", mock.Anything).Return([]*model.RunnerProfile{p1, p2}, nil)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/profiles", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp rest.ProfileListResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, 2, resp.Count)
		assert.Len(t, resp.Profiles, 2)
		assert.Equal(t, "prof-1", resp.Profiles[0].ID)
		mockProfilesUC.AssertExpectations(t)
	})

	t.Run("internal error on listing returns 500", func(t *testing.T) {
		mockProfilesUC := new(MockProfilesUseCase)
		router := rest.SetupRouter(nil, mockProfilesUC, nil, nil)

		mockProfilesUC.On("ListProfiles", mock.Anything).Return(nil, errors.New("list failed"))

		req := httptest.NewRequest(http.MethodGet, "/api/v1/profiles", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		mockProfilesUC.AssertExpectations(t)
	})
}

func TestProfileHandler_GetProfile(t *testing.T) {
	t.Run("successfully get profile returns 200 OK", func(t *testing.T) {
		mockProfilesUC := new(MockProfilesUseCase)
		router := rest.SetupRouter(nil, mockProfilesUC, nil, nil)

		p := createSampleProfile(t, "prof-100", "my-profile")
		mockProfilesUC.On("GetProfile", mock.Anything, "prof-100").Return(p, nil)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/profiles/prof-100", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp rest.ProfileResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "prof-100", resp.ID)
		mockProfilesUC.AssertExpectations(t)
	})

	t.Run("not found returns 404", func(t *testing.T) {
		mockProfilesUC := new(MockProfilesUseCase)
		router := rest.SetupRouter(nil, mockProfilesUC, nil, nil)

		mockProfilesUC.On("GetProfile", mock.Anything, "missing").Return(nil, model.ErrNotFound)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/profiles/missing", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
		mockProfilesUC.AssertExpectations(t)
	})
}

func TestProfileHandler_UpdateProfile(t *testing.T) {
	t.Run("successfully update profile returns 200 OK", func(t *testing.T) {
		mockProfilesUC := new(MockProfilesUseCase)
		router := rest.SetupRouter(nil, mockProfilesUC, nil, nil)

		updatedProfile := createSampleProfile(t, "prof-1", "updated-profile")
		mockProfilesUC.On("UpdateProfile", mock.Anything, "prof-1", mock.MatchedBy(func(cmd inbound.UpdateProfileCommand) bool {
			return cmd.Name == "updated-profile"
		})).Return(updatedProfile, nil)

		reqBody := rest.UpdateProfileRequest{
			Name:        "updated-profile",
			Description: "Updated desc",
		}
		raw, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPut, "/api/v1/profiles/prof-1", bytes.NewReader(raw))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp rest.ProfileResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "updated-profile", resp.Name)
		mockProfilesUC.AssertExpectations(t)
	})

	t.Run("update non-existent profile returns 404", func(t *testing.T) {
		mockProfilesUC := new(MockProfilesUseCase)
		router := rest.SetupRouter(nil, mockProfilesUC, nil, nil)

		mockProfilesUC.On("UpdateProfile", mock.Anything, "missing", mock.Anything).
			Return(nil, model.ErrNotFound)

		reqBody := rest.UpdateProfileRequest{Name: "name"}
		raw, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPut, "/api/v1/profiles/missing", bytes.NewReader(raw))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
		mockProfilesUC.AssertExpectations(t)
	})
}

func TestProfileHandler_DeleteProfile(t *testing.T) {
	t.Run("successfully delete profile returns 204 No Content", func(t *testing.T) {
		mockProfilesUC := new(MockProfilesUseCase)
		router := rest.SetupRouter(nil, mockProfilesUC, nil, nil)

		mockProfilesUC.On("DeleteProfile", mock.Anything, "prof-1").Return(nil)

		req := httptest.NewRequest(http.MethodDelete, "/api/v1/profiles/prof-1", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)
		mockProfilesUC.AssertExpectations(t)
	})

	t.Run("delete non-existent profile returns 404 Not Found", func(t *testing.T) {
		mockProfilesUC := new(MockProfilesUseCase)
		router := rest.SetupRouter(nil, mockProfilesUC, nil, nil)

		mockProfilesUC.On("DeleteProfile", mock.Anything, "missing").Return(model.ErrNotFound)

		req := httptest.NewRequest(http.MethodDelete, "/api/v1/profiles/missing", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
		mockProfilesUC.AssertExpectations(t)
	})
}
