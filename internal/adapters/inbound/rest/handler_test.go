package rest_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime/multipart"
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

// MockBuildsUseCase mocks inbound.BuildsUseCase
type MockBuildsUseCase struct {
	mock.Mock
}

func (m *MockBuildsUseCase) TriggerBuild(ctx context.Context, suiteID string, platform *model.Platform, source io.Reader, size int64) ([]*model.Artifact, error) {
	args := m.Called(ctx, suiteID, platform, source, size)
	if a := args.Get(0); a != nil {
		return a.([]*model.Artifact), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockBuildsUseCase) BuildArtifact(ctx context.Context, suiteID, artifactID string) (*model.Artifact, error) {
	args := m.Called(ctx, suiteID, artifactID)
	if a := args.Get(0); a != nil {
		return a.(*model.Artifact), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockBuildsUseCase) BuildSuite(ctx context.Context, suiteID string) ([]*model.Artifact, error) {
	args := m.Called(ctx, suiteID)
	if a := args.Get(0); a != nil {
		return a.([]*model.Artifact), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockBuildsUseCase) GetArtifact(ctx context.Context, id string) (*model.Artifact, error) {
	args := m.Called(ctx, id)
	if a := args.Get(0); a != nil {
		return a.(*model.Artifact), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockBuildsUseCase) ListArtifacts(ctx context.Context, suiteID string) ([]*model.Artifact, error) {
	args := m.Called(ctx, suiteID)
	if a := args.Get(0); a != nil {
		return a.([]*model.Artifact), args.Error(1)
	}
	return nil, args.Error(1)
}

var _ inbound.BuildsUseCase = (*MockBuildsUseCase)(nil)

func createMultipartRequest(t *testing.T, targetURL, fieldName, filename string, fileContent []byte, extraFields map[string]string) (*http.Request, string) {
	t.Helper()
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	if filename != "" {
		part, err := writer.CreateFormFile(fieldName, filename)
		require.NoError(t, err)
		_, err = part.Write(fileContent)
		require.NoError(t, err)
	}

	for k, v := range extraFields {
		err := writer.WriteField(k, v)
		require.NoError(t, err)
	}

	err := writer.Close()
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, targetURL, body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	return req, writer.FormDataContentType()
}

func TestArtifactHandler_UploadAndBuild(t *testing.T) {
	suiteID := "suite-100"

	t.Run("success with specific platform", func(t *testing.T) {
		mockUC := new(MockBuildsUseCase)
		router := rest.SetupRouter(mockUC, nil)

		expectedPlatform := model.PlatformLinuxAmd64
		art, err := model.NewArtifact(suiteID, expectedPlatform)
		require.NoError(t, err)

		mockUC.On("TriggerBuild", mock.Anything, suiteID, &expectedPlatform, mock.Anything, mock.AnythingOfType("int64")).
			Return([]*model.Artifact{art}, nil)

		req, _ := createMultipartRequest(t, "/api/v1/suites/"+suiteID+"/builds", "file", "source.tar.gz", []byte("fake-tarball"), map[string]string{
			"platform": "linux/amd64",
		})
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusAccepted, rec.Code)

		var resp rest.BuildTriggerResponse
		err = json.Unmarshal(rec.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "build triggered successfully", resp.Message)
		require.Len(t, resp.Artifacts, 1)
		assert.Equal(t, art.ID(), resp.Artifacts[0].ID)
		assert.Equal(t, suiteID, resp.Artifacts[0].SuiteID)
		assert.Equal(t, "linux/amd64", resp.Artifacts[0].Platform)
		assert.Equal(t, "PENDING", resp.Artifacts[0].Status)
	})

	t.Run("success with arch parameter alias", func(t *testing.T) {
		mockUC := new(MockBuildsUseCase)
		router := rest.SetupRouter(mockUC, nil)

		expectedPlatform := model.PlatformLinuxArm64
		art, err := model.NewArtifact(suiteID, expectedPlatform)
		require.NoError(t, err)

		mockUC.On("TriggerBuild", mock.Anything, suiteID, &expectedPlatform, mock.Anything, mock.AnythingOfType("int64")).
			Return([]*model.Artifact{art}, nil)

		req, _ := createMultipartRequest(t, "/api/v1/suites/"+suiteID+"/builds", "file", "source.tar.gz", []byte("fake-tarball"), map[string]string{
			"arch": "linux/arm64",
		})
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusAccepted, rec.Code)
	})

	t.Run("success with multi-arch when arch is omitted or all", func(t *testing.T) {
		mockUC := new(MockBuildsUseCase)
		router := rest.SetupRouter(mockUC, nil)

		art1, _ := model.NewArtifact(suiteID, model.PlatformLinuxAmd64)
		art2, _ := model.NewArtifact(suiteID, model.PlatformLinuxArm64)

		mockUC.On("TriggerBuild", mock.Anything, suiteID, (*model.Platform)(nil), mock.Anything, mock.AnythingOfType("int64")).
			Return([]*model.Artifact{art1, art2}, nil)

		req, _ := createMultipartRequest(t, "/api/v1/suites/"+suiteID+"/builds", "file", "source.tar.gz", []byte("fake-tarball"), nil)
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusAccepted, rec.Code)

		var resp rest.BuildTriggerResponse
		err := json.Unmarshal(rec.Body.Bytes(), &resp)
		require.NoError(t, err)
		require.Len(t, resp.Artifacts, 2)
	})

	t.Run("failure when file is missing in multipart form", func(t *testing.T) {
		mockUC := new(MockBuildsUseCase)
		router := rest.SetupRouter(mockUC, nil)

		req, _ := createMultipartRequest(t, "/api/v1/suites/"+suiteID+"/builds", "wrong_field", "", nil, map[string]string{
			"platform": "linux/amd64",
		})
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
		var errResp rest.ErrorResponse
		err := json.Unmarshal(rec.Body.Bytes(), &errResp)
		require.NoError(t, err)
		assert.Contains(t, errResp.Error, "file")
	})

	t.Run("failure with unsupported platform", func(t *testing.T) {
		mockUC := new(MockBuildsUseCase)
		router := rest.SetupRouter(mockUC, nil)

		req, _ := createMultipartRequest(t, "/api/v1/suites/"+suiteID+"/builds", "file", "source.tar.gz", []byte("content"), map[string]string{
			"platform": "windows/amd64",
		})
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
		var errResp rest.ErrorResponse
		err := json.Unmarshal(rec.Body.Bytes(), &errResp)
		require.NoError(t, err)
		assert.Contains(t, errResp.Error, "unsupported platform")
	})

	t.Run("failure when suite is not found", func(t *testing.T) {
		mockUC := new(MockBuildsUseCase)
		router := rest.SetupRouter(mockUC, nil)

		mockUC.On("TriggerBuild", mock.Anything, "non-existent-suite", (*model.Platform)(nil), mock.Anything, mock.AnythingOfType("int64")).
			Return(nil, model.ErrNotFound)

		req, _ := createMultipartRequest(t, "/api/v1/suites/non-existent-suite/builds", "file", "source.tar.gz", []byte("content"), nil)
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("failure with internal server error", func(t *testing.T) {
		mockUC := new(MockBuildsUseCase)
		router := rest.SetupRouter(mockUC, nil)

		mockUC.On("TriggerBuild", mock.Anything, suiteID, (*model.Platform)(nil), mock.Anything, mock.AnythingOfType("int64")).
			Return(nil, errors.New("s3 connection failed"))

		req, _ := createMultipartRequest(t, "/api/v1/suites/"+suiteID+"/builds", "file", "source.tar.gz", []byte("content"), nil)
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusInternalServerError, rec.Code)
	})
}

func TestArtifactHandler_ListArtifacts(t *testing.T) {
	suiteID := "suite-100"

	t.Run("success with available artifacts and checksums", func(t *testing.T) {
		mockUC := new(MockBuildsUseCase)
		router := rest.SetupRouter(mockUC, nil)

		art1, err := model.NewArtifactWithID(
			"art-1", suiteID, model.PlatformLinuxAmd64,
			"suites/suite-100/artifacts/art-1/linux-amd64/runner",
			"e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
			"suites/suite-100/artifacts/art-1/build.log",
			model.ArtifactStatusReady, "", time.Now().UTC(),
		)
		require.NoError(t, err)

		art2, err := model.NewArtifactWithID(
			"art-2", suiteID, model.PlatformLinuxArm64,
			"suites/suite-100/artifacts/art-2/linux-arm64/runner",
			"a591a6d40bf420404a011733cfb7b190d62c65bf0bcda32b57b277d9ad9f146e",
			"suites/suite-100/artifacts/art-2/build.log",
			model.ArtifactStatusReady, "", time.Now().UTC(),
		)
		require.NoError(t, err)

		mockUC.On("ListArtifacts", mock.Anything, suiteID).Return([]*model.Artifact{art1, art2}, nil)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/suites/"+suiteID+"/artifacts", nil)
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)

		var resp rest.ArtifactListResponse
		err = json.Unmarshal(rec.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, 2, resp.Count)
		require.Len(t, resp.Artifacts, 2)
		assert.Equal(t, "art-1", resp.Artifacts[0].ID)
		assert.Equal(t, "linux/amd64", resp.Artifacts[0].Platform)
		assert.Equal(t, "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855", resp.Artifacts[0].SHA256Checksum)
		assert.Equal(t, "suites/suite-100/artifacts/art-1/linux-amd64/runner", resp.Artifacts[0].S3BinaryKey)
		assert.Equal(t, "READY", resp.Artifacts[0].Status)
	})

	t.Run("success with empty artifact list", func(t *testing.T) {
		mockUC := new(MockBuildsUseCase)
		router := rest.SetupRouter(mockUC, nil)

		mockUC.On("ListArtifacts", mock.Anything, suiteID).Return([]*model.Artifact{}, nil)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/suites/"+suiteID+"/artifacts", nil)
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)

		var resp rest.ArtifactListResponse
		err := json.Unmarshal(rec.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, 0, resp.Count)
		assert.NotNil(t, resp.Artifacts)
		assert.Len(t, resp.Artifacts, 0)
	})

	t.Run("failure when suite not found", func(t *testing.T) {
		mockUC := new(MockBuildsUseCase)
		router := rest.SetupRouter(mockUC, nil)

		mockUC.On("ListArtifacts", mock.Anything, "non-existent").Return(nil, model.ErrNotFound)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/suites/non-existent/artifacts", nil)
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("failure with internal server error", func(t *testing.T) {
		mockUC := new(MockBuildsUseCase)
		router := rest.SetupRouter(mockUC, nil)

		mockUC.On("ListArtifacts", mock.Anything, suiteID).Return(nil, errors.New("db error"))

		req := httptest.NewRequest(http.MethodGet, "/api/v1/suites/"+suiteID+"/artifacts", nil)
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusInternalServerError, rec.Code)
	})
}

func TestHealthCheck(t *testing.T) {
	mockUC := new(MockBuildsUseCase)
	router := rest.SetupRouter(mockUC, nil)

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	var body map[string]string
	err := json.Unmarshal(rec.Body.Bytes(), &body)
	require.NoError(t, err)
	assert.Equal(t, "ok", body["status"])
}
