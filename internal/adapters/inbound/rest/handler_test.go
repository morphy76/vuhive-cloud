package rest_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
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
	return m.TriggerBuildWithOptions(ctx, suiteID, platform, source, size, inbound.BuildOptions{})
}

func (m *MockBuildsUseCase) TriggerBuildWithOptions(ctx context.Context, suiteID string, platform *model.Platform, source io.Reader, size int64, opts inbound.BuildOptions) ([]*model.Artifact, error) {
	args := m.Called(ctx, suiteID, platform, source, size, opts)
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

func (m *MockBuildsUseCase) BuildArtifactWithOptions(ctx context.Context, suiteID, artifactID string, opts inbound.BuildOptions) (*model.Artifact, error) {
	args := m.Called(ctx, suiteID, artifactID, opts)
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

func (m *MockBuildsUseCase) CancelBuild(ctx context.Context, suiteID, artifactID, reason string) (*model.Artifact, error) {
	args := m.Called(ctx, suiteID, artifactID, reason)
	if a := args.Get(0); a != nil {
		return a.(*model.Artifact), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockBuildsUseCase) RetryBuild(ctx context.Context, suiteID, artifactID string) (*model.Artifact, error) {
	args := m.Called(ctx, suiteID, artifactID)
	if a := args.Get(0); a != nil {
		return a.(*model.Artifact), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockBuildsUseCase) DeleteArtifact(ctx context.Context, suiteID, artifactID string) error {
	args := m.Called(ctx, suiteID, artifactID)
	return args.Error(0)
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
		router := rest.SetupRouter(mockUC, nil, nil, nil)

		expectedPlatform := model.PlatformLinuxAmd64
		art, err := model.NewArtifact(suiteID, expectedPlatform)
		require.NoError(t, err)

		mockUC.On("TriggerBuildWithOptions", mock.Anything, suiteID, &expectedPlatform, mock.Anything, mock.AnythingOfType("int64"), inbound.BuildOptions{AllowInsecureImports: false}).
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
		router := rest.SetupRouter(mockUC, nil, nil, nil)

		expectedPlatform := model.PlatformLinuxArm64
		art, err := model.NewArtifact(suiteID, expectedPlatform)
		require.NoError(t, err)

		mockUC.On("TriggerBuildWithOptions", mock.Anything, suiteID, &expectedPlatform, mock.Anything, mock.AnythingOfType("int64"), inbound.BuildOptions{AllowInsecureImports: false}).
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
		router := rest.SetupRouter(mockUC, nil, nil, nil)

		art1, _ := model.NewArtifact(suiteID, model.PlatformLinuxAmd64)
		art2, _ := model.NewArtifact(suiteID, model.PlatformLinuxArm64)

		mockUC.On("TriggerBuildWithOptions", mock.Anything, suiteID, (*model.Platform)(nil), mock.Anything, mock.AnythingOfType("int64"), inbound.BuildOptions{AllowInsecureImports: false}).
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
		router := rest.SetupRouter(mockUC, nil, nil, nil)

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
		router := rest.SetupRouter(mockUC, nil, nil, nil)

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
		router := rest.SetupRouter(mockUC, nil, nil, nil)

		mockUC.On("TriggerBuildWithOptions", mock.Anything, "non-existent-suite", (*model.Platform)(nil), mock.Anything, mock.AnythingOfType("int64"), inbound.BuildOptions{AllowInsecureImports: false}).
			Return(nil, model.ErrNotFound)

		req, _ := createMultipartRequest(t, "/api/v1/suites/non-existent-suite/builds", "file", "source.tar.gz", []byte("content"), nil)
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("failure with internal server error", func(t *testing.T) {
		mockUC := new(MockBuildsUseCase)
		router := rest.SetupRouter(mockUC, nil, nil, nil)

		mockUC.On("TriggerBuildWithOptions", mock.Anything, suiteID, (*model.Platform)(nil), mock.Anything, mock.AnythingOfType("int64"), inbound.BuildOptions{AllowInsecureImports: false}).
			Return(nil, errors.New("s3 connection failed"))

		req, _ := createMultipartRequest(t, "/api/v1/suites/"+suiteID+"/builds", "file", "source.tar.gz", []byte("content"), nil)
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusInternalServerError, rec.Code)
	})

	t.Run("failure when pre-build static analysis fails (missing vuhive dependency)", func(t *testing.T) {
		mockUC := new(MockBuildsUseCase)
		router := rest.SetupRouter(mockUC, nil, nil, nil)

		mockUC.On("TriggerBuildWithOptions", mock.Anything, suiteID, (*model.Platform)(nil), mock.Anything, mock.AnythingOfType("int64"), inbound.BuildOptions{AllowInsecureImports: false}).
			Return(nil, model.ErrMissingVuhiveDependency)

		req, _ := createMultipartRequest(t, "/api/v1/suites/"+suiteID+"/builds", "file", "source.tar.gz", []byte("content"), nil)
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
		var errResp rest.ErrorResponse
		err := json.Unmarshal(rec.Body.Bytes(), &errResp)
		require.NoError(t, err)
		assert.Contains(t, errResp.Error, "github.com/morphy76/vuhive")
	})

	t.Run("failure when pre-build static analysis fails (forbidden import)", func(t *testing.T) {
		mockUC := new(MockBuildsUseCase)
		router := rest.SetupRouter(mockUC, nil, nil, nil)

		mockUC.On("TriggerBuildWithOptions", mock.Anything, suiteID, (*model.Platform)(nil), mock.Anything, mock.AnythingOfType("int64"), inbound.BuildOptions{AllowInsecureImports: false}).
			Return(nil, model.ErrForbiddenImport)

		req, _ := createMultipartRequest(t, "/api/v1/suites/"+suiteID+"/builds", "file", "source.tar.gz", []byte("content"), nil)
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
		var errResp rest.ErrorResponse
		err := json.Unmarshal(rec.Body.Bytes(), &errResp)
		require.NoError(t, err)
		assert.Contains(t, errResp.Error, "disallowed package")
	})

	t.Run("failure when insecure override is forbidden by cluster policy", func(t *testing.T) {
		mockUC := new(MockBuildsUseCase)
		router := rest.SetupRouter(mockUC, nil, nil, nil)

		mockUC.On("TriggerBuildWithOptions", mock.Anything, suiteID, (*model.Platform)(nil), mock.Anything, mock.AnythingOfType("int64"), inbound.BuildOptions{AllowInsecureImports: true}).
			Return(nil, model.ErrInsecureOverrideForbidden)

		req, _ := createMultipartRequest(t, "/api/v1/suites/"+suiteID+"/builds", "file", "source.tar.gz", []byte("content"), map[string]string{
			"allow_insecure_imports": "true",
		})
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusForbidden, rec.Code)
		var errResp rest.ErrorResponse
		err := json.Unmarshal(rec.Body.Bytes(), &errResp)
		require.NoError(t, err)
		assert.Contains(t, errResp.Error, "override is forbidden")
	})

	t.Run("success with allow_insecure_imports override flag", func(t *testing.T) {
		mockUC := new(MockBuildsUseCase)
		router := rest.SetupRouter(mockUC, nil, nil, nil)

		art, err := model.NewArtifact(suiteID, model.PlatformLinuxAmd64)
		require.NoError(t, err)

		mockUC.On("TriggerBuildWithOptions", mock.Anything, suiteID, (*model.Platform)(nil), mock.Anything, mock.AnythingOfType("int64"), inbound.BuildOptions{AllowInsecureImports: true}).
			Return([]*model.Artifact{art}, nil)

		req, _ := createMultipartRequest(t, "/api/v1/suites/"+suiteID+"/builds", "file", "source.tar.gz", []byte("content"), map[string]string{
			"allow_insecure_imports": "true",
		})
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusAccepted, rec.Code)
	})

	t.Run("success with go_version specified", func(t *testing.T) {
		mockUC := new(MockBuildsUseCase)
		router := rest.SetupRouter(mockUC, nil, nil, nil)

		art, err := model.NewArtifact(suiteID, model.PlatformLinuxAmd64)
		require.NoError(t, err)

		mockUC.On("TriggerBuildWithOptions", mock.Anything, suiteID, (*model.Platform)(nil), mock.Anything, mock.AnythingOfType("int64"), inbound.BuildOptions{
			GoVersion: "1.27",
		}).Return([]*model.Artifact{art}, nil)

		req, _ := createMultipartRequest(t, "/api/v1/suites/"+suiteID+"/builds", "file", "source.tar.gz", []byte("content"), map[string]string{
			"go_version": "1.27",
		})
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusAccepted, rec.Code)
		mockUC.AssertExpectations(t)
	})

	t.Run("success with go_image specified", func(t *testing.T) {
		mockUC := new(MockBuildsUseCase)
		router := rest.SetupRouter(mockUC, nil, nil, nil)

		art, err := model.NewArtifact(suiteID, model.PlatformLinuxAmd64)
		require.NoError(t, err)

		mockUC.On("TriggerBuildWithOptions", mock.Anything, suiteID, (*model.Platform)(nil), mock.Anything, mock.AnythingOfType("int64"), inbound.BuildOptions{
			GoImage: "custom.registry/golang:1.27-alpine",
		}).Return([]*model.Artifact{art}, nil)

		req, _ := createMultipartRequest(t, "/api/v1/suites/"+suiteID+"/builds", "file", "source.tar.gz", []byte("content"), map[string]string{
			"go_image": "custom.registry/golang:1.27-alpine",
		})
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusAccepted, rec.Code)
		mockUC.AssertExpectations(t)
	})

	t.Run("fails with HTTP 400 when go_version is older than 1.26", func(t *testing.T) {
		mockUC := new(MockBuildsUseCase)
		router := rest.SetupRouter(mockUC, nil, nil, nil)

		req, _ := createMultipartRequest(t, "/api/v1/suites/"+suiteID+"/builds", "file", "source.tar.gz", []byte("content"), map[string]string{
			"go_version": "1.24",
		})
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
		var errResp rest.ErrorResponse
		err := json.Unmarshal(rec.Body.Bytes(), &errResp)
		require.NoError(t, err)
		assert.Contains(t, errResp.Error, "unsupported go version")
	})
}

func TestArtifactHandler_ListArtifacts(t *testing.T) {
	suiteID := "suite-100"

	t.Run("success with available artifacts and checksums", func(t *testing.T) {
		mockUC := new(MockBuildsUseCase)
		router := rest.SetupRouter(mockUC, nil, nil, nil)

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
		router := rest.SetupRouter(mockUC, nil, nil, nil)

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
		router := rest.SetupRouter(mockUC, nil, nil, nil)

		mockUC.On("ListArtifacts", mock.Anything, "non-existent").Return(nil, model.ErrNotFound)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/suites/non-existent/artifacts", nil)
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("failure with internal server error", func(t *testing.T) {
		mockUC := new(MockBuildsUseCase)
		router := rest.SetupRouter(mockUC, nil, nil, nil)

		mockUC.On("ListArtifacts", mock.Anything, suiteID).Return(nil, errors.New("db error"))

		req := httptest.NewRequest(http.MethodGet, "/api/v1/suites/"+suiteID+"/artifacts", nil)
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusInternalServerError, rec.Code)
	})
}

func TestHealthCheck(t *testing.T) {
	mockUC := new(MockBuildsUseCase)
	router := rest.SetupRouter(mockUC, nil, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	var body map[string]string
	err := json.Unmarshal(rec.Body.Bytes(), &body)
	require.NoError(t, err)
	assert.Equal(t, "ok", body["status"])
}

func TestArtifactHandler_CancelBuild(t *testing.T) {
	suiteID := "suite-123"
	artifactID := "art-456"

	t.Run("success: cancels build and returns 200 with cancelled artifact", func(t *testing.T) {
		mockUC := new(MockBuildsUseCase)
		router := rest.SetupRouter(mockUC, nil, nil, nil)

		cancelledArt, _ := model.NewArtifact(suiteID, model.PlatformLinuxAmd64)
		_ = cancelledArt.Cancel("user stopped build")

		mockUC.On("CancelBuild", mock.Anything, suiteID, artifactID, "user stopped build").Return(cancelledArt, nil)

		body := `{"reason":"user stopped build"}`
		req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/suites/%s/artifacts/%s/cancel", suiteID, artifactID), strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		var resp rest.ArtifactResponse
		err := json.Unmarshal(rec.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "CANCELLED", resp.Status)
		assert.Equal(t, "user stopped build", resp.ErrorMessage)
	})

	t.Run("failure: returns 409 conflict when artifact in terminal state", func(t *testing.T) {
		mockUC := new(MockBuildsUseCase)
		router := rest.SetupRouter(mockUC, nil, nil, nil)

		mockUC.On("CancelBuild", mock.Anything, suiteID, artifactID, "").Return(nil, model.ErrTerminalState)

		req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/suites/%s/artifacts/%s/cancel", suiteID, artifactID), nil)
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusConflict, rec.Code)
	})

	t.Run("failure: returns 404 when artifact not found", func(t *testing.T) {
		mockUC := new(MockBuildsUseCase)
		router := rest.SetupRouter(mockUC, nil, nil, nil)

		mockUC.On("CancelBuild", mock.Anything, suiteID, artifactID, "").Return(nil, model.ErrNotFound)

		req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/suites/%s/artifacts/%s/cancel", suiteID, artifactID), nil)
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusNotFound, rec.Code)
	})
}

func TestArtifactHandler_RetryBuild(t *testing.T) {
	suiteID := "suite-123"
	artifactID := "art-456"

	t.Run("success: retries build and returns 202 accepted with pending artifact", func(t *testing.T) {
		mockUC := new(MockBuildsUseCase)
		router := rest.SetupRouter(mockUC, nil, nil, nil)

		retriedArt, _ := model.NewArtifact(suiteID, model.PlatformLinuxAmd64)

		mockUC.On("RetryBuild", mock.Anything, suiteID, artifactID).Return(retriedArt, nil)

		req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/suites/%s/artifacts/%s/retry", suiteID, artifactID), nil)
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusAccepted, rec.Code)
		var resp rest.ArtifactResponse
		err := json.Unmarshal(rec.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, "PENDING", resp.Status)
	})

	t.Run("failure: returns 400 when artifact cannot be retried", func(t *testing.T) {
		mockUC := new(MockBuildsUseCase)
		router := rest.SetupRouter(mockUC, nil, nil, nil)

		mockUC.On("RetryBuild", mock.Anything, suiteID, artifactID).Return(nil, model.ErrInvalidStateTransition)

		req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/suites/%s/artifacts/%s/retry", suiteID, artifactID), nil)
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
}

func TestArtifactHandler_DeleteArtifact(t *testing.T) {
	suiteID := "suite-123"
	artifactID := "art-456"

	t.Run("success: deletes artifact and returns 204 no content", func(t *testing.T) {
		mockUC := new(MockBuildsUseCase)
		router := rest.SetupRouter(mockUC, nil, nil, nil)

		mockUC.On("DeleteArtifact", mock.Anything, suiteID, artifactID).Return(nil)

		req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/suites/%s/artifacts/%s", suiteID, artifactID), nil)
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusNoContent, rec.Code)
	})

	t.Run("failure: returns 409 conflict when artifact is currently building", func(t *testing.T) {
		mockUC := new(MockBuildsUseCase)
		router := rest.SetupRouter(mockUC, nil, nil, nil)

		mockUC.On("DeleteArtifact", mock.Anything, suiteID, artifactID).Return(model.ErrConflict)

		req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/suites/%s/artifacts/%s", suiteID, artifactID), nil)
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusConflict, rec.Code)
	})

	t.Run("failure: returns 404 when artifact not found", func(t *testing.T) {
		mockUC := new(MockBuildsUseCase)
		router := rest.SetupRouter(mockUC, nil, nil, nil)

		mockUC.On("DeleteArtifact", mock.Anything, suiteID, artifactID).Return(model.ErrNotFound)

		req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/suites/%s/artifacts/%s", suiteID, artifactID), nil)
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusNotFound, rec.Code)
	})
}
