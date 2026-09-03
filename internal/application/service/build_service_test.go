package service_test

import (
	"bytes"
	"context"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/morphy76/vuhive-cloud/internal/application/ports/inbound"
	"github.com/morphy76/vuhive-cloud/internal/application/ports/outbound"
	"github.com/morphy76/vuhive-cloud/internal/application/service"
	"github.com/morphy76/vuhive-cloud/internal/domain/model"
)

// MockArtifactRepository mocks outbound.ArtifactRepository
type MockArtifactRepository struct {
	mock.Mock
}

func (m *MockArtifactRepository) Save(ctx context.Context, artifact *model.Artifact) error {
	args := m.Called(ctx, artifact)
	return args.Error(0)
}

func (m *MockArtifactRepository) FindByID(ctx context.Context, id string) (*model.Artifact, error) {
	args := m.Called(ctx, id)
	if fn, ok := args.Get(0).(func(context.Context, string) *model.Artifact); ok {
		return fn(ctx, id), args.Error(1)
	}
	if a := args.Get(0); a != nil {
		return a.(*model.Artifact), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockArtifactRepository) ListBySuiteID(ctx context.Context, suiteID string) ([]*model.Artifact, error) {
	args := m.Called(ctx, suiteID)
	if a := args.Get(0); a != nil {
		return a.([]*model.Artifact), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockArtifactRepository) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

// MockStoragePort mocks outbound.StoragePort
type MockStoragePort struct {
	mock.Mock
}

func (m *MockStoragePort) Upload(ctx context.Context, key string, content io.Reader, size int64, contentType string) error {
	args := m.Called(ctx, key, content, size, contentType)
	return args.Error(0)
}

func (m *MockStoragePort) Download(ctx context.Context, key string) (io.ReadCloser, error) {
	args := m.Called(ctx, key)
	if r := args.Get(0); r != nil {
		return r.(io.ReadCloser), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockStoragePort) Delete(ctx context.Context, key string) error {
	args := m.Called(ctx, key)
	return args.Error(0)
}

func (m *MockStoragePort) Exists(ctx context.Context, key string) (bool, error) {
	args := m.Called(ctx, key)
	return args.Bool(0), args.Error(1)
}

func (m *MockStoragePort) PresignDownload(ctx context.Context, key string, lifetime time.Duration) (string, error) {
	args := m.Called(ctx, key, lifetime)
	return args.String(0), args.Error(1)
}

func (m *MockStoragePort) PresignUpload(ctx context.Context, key string, lifetime time.Duration) (string, error) {
	args := m.Called(ctx, key, lifetime)
	return args.String(0), args.Error(1)
}

func (m *MockStoragePort) EnsureBucket(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

// MockBuildOrchestratorPort mocks outbound.BuildOrchestratorPort
type MockBuildOrchestratorPort struct {
	mock.Mock
}

func (m *MockBuildOrchestratorPort) DispatchBuildJob(ctx context.Context, opts outbound.BuildJobOptions) (string, error) {
	args := m.Called(ctx, opts)
	return args.String(0), args.Error(1)
}

func (m *MockBuildOrchestratorPort) StreamJobLogs(ctx context.Context, jobName string) (io.ReadCloser, error) {
	args := m.Called(ctx, jobName)
	if r := args.Get(0); r != nil {
		return r.(io.ReadCloser), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockBuildOrchestratorPort) WaitForJob(ctx context.Context, jobName string) (*outbound.BuildJobExecution, error) {
	args := m.Called(ctx, jobName)
	if r := args.Get(0); r != nil {
		return r.(*outbound.BuildJobExecution), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockBuildOrchestratorPort) DeleteJob(ctx context.Context, jobName string) error {
	args := m.Called(ctx, jobName)
	return args.Error(0)
}

func TestBuildService_BuildArtifact_Success(t *testing.T) {
	repo := new(MockArtifactRepository)
	storage := new(MockStoragePort)
	orchestrator := new(MockBuildOrchestratorPort)

	svc := service.NewBuildService(repo, storage, orchestrator)

	suiteID := "suite-1111"
	artifact, err := model.NewArtifact(suiteID, model.PlatformLinuxAmd64)
	require.NoError(t, err)

	ctx := context.Background()
	sourceKey := "suites/suite-1111/sources/source.tar.gz"
	binaryKey := "suites/suite-1111/artifacts/" + artifact.ID() + "/linux-amd64/runner"
	logsKey := "suites/suite-1111/artifacts/" + artifact.ID() + "/build.log"
	jobName := "vuhive-build-" + artifact.ID()
	checksum := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"

	repo.On("FindByID", ctx, artifact.ID()).Return(artifact, nil)
	storage.On("Exists", ctx, sourceKey).Return(true, nil)
	repo.On("Save", ctx, mock.MatchedBy(func(a *model.Artifact) bool {
		return a.Status() == model.ArtifactStatusBuilding
	})).Return(nil)

	storage.On("PresignDownload", ctx, sourceKey, mock.Anything).Return("https://download-source", nil)
	storage.On("PresignUpload", ctx, binaryKey, mock.Anything).Return("https://upload-binary", nil)

	orchestrator.On("DispatchBuildJob", ctx, mock.MatchedBy(func(opts outbound.BuildJobOptions) bool {
		return opts.SuiteID == suiteID && opts.ArtifactID == artifact.ID() && opts.Platform == model.PlatformLinuxAmd64
	})).Return(jobName, nil)

	logOutput := "compiling...\nVUHIVE_ARTIFACT_SHA256=" + checksum + "\ndone"
	orchestrator.On("WaitForJob", ctx, jobName).Return(&outbound.BuildJobExecution{
		JobName:        jobName,
		ExitCode:       0,
		SHA256Checksum: checksum,
		Logs:           io.NopCloser(strings.NewReader(logOutput)),
	}, nil)

	storage.On("Upload", ctx, logsKey, mock.Anything, mock.Anything, "text/plain").Return(nil)

	repo.On("Save", ctx, mock.MatchedBy(func(a *model.Artifact) bool {
		return a.Status() == model.ArtifactStatusReady && a.SHA256Checksum() == checksum
	})).Return(nil)

	res, err := svc.BuildArtifact(ctx, suiteID, artifact.ID())
	require.NoError(t, err)
	assert.Equal(t, model.ArtifactStatusReady, res.Status())
	assert.Equal(t, checksum, res.SHA256Checksum())
	assert.Equal(t, binaryKey, res.S3BinaryKey())

	repo.AssertExpectations(t)
	storage.AssertExpectations(t)
	orchestrator.AssertExpectations(t)
}

func TestBuildService_BuildArtifact_CompilationFailed(t *testing.T) {
	repo := new(MockArtifactRepository)
	storage := new(MockStoragePort)
	orchestrator := new(MockBuildOrchestratorPort)

	svc := service.NewBuildService(repo, storage, orchestrator)

	suiteID := "suite-1111"
	artifact, err := model.NewArtifact(suiteID, model.PlatformLinuxArm64)
	require.NoError(t, err)

	ctx := context.Background()
	sourceKey := "suites/suite-1111/sources/source.tar.gz"
	binaryKey := "suites/suite-1111/artifacts/" + artifact.ID() + "/linux-arm64/runner"
	logsKey := "suites/suite-1111/artifacts/" + artifact.ID() + "/build.log"
	jobName := "vuhive-build-" + artifact.ID()

	repo.On("FindByID", ctx, artifact.ID()).Return(artifact, nil)
	storage.On("Exists", ctx, sourceKey).Return(true, nil)
	repo.On("Save", ctx, mock.MatchedBy(func(a *model.Artifact) bool {
		return a.Status() == model.ArtifactStatusBuilding
	})).Return(nil)

	storage.On("PresignDownload", ctx, sourceKey, mock.Anything).Return("https://download-source", nil)
	storage.On("PresignUpload", ctx, binaryKey, mock.Anything).Return("https://upload-binary", nil)

	orchestrator.On("DispatchBuildJob", ctx, mock.Anything).Return(jobName, nil)

	failLogs := "syntax error in main.go:12:3"
	orchestrator.On("WaitForJob", ctx, jobName).Return(&outbound.BuildJobExecution{
		JobName:  jobName,
		ExitCode: 1,
		Logs:     io.NopCloser(strings.NewReader(failLogs)),
	}, model.ErrBuildFailed)

	storage.On("Upload", ctx, logsKey, mock.Anything, mock.Anything, "text/plain").Return(nil)

	repo.On("Save", ctx, mock.MatchedBy(func(a *model.Artifact) bool {
		return a.Status() == model.ArtifactStatusFailed && a.BuildLogsS3Key() == logsKey
	})).Return(nil)

	res, err := svc.BuildArtifact(ctx, suiteID, artifact.ID())
	assert.ErrorIs(t, err, model.ErrBuildFailed)
	require.NotNil(t, res)
	assert.Equal(t, model.ArtifactStatusFailed, res.Status())

	repo.AssertExpectations(t)
	storage.AssertExpectations(t)
	orchestrator.AssertExpectations(t)
}

func TestBuildService_BuildSuite_BothPlatforms(t *testing.T) {
	repo := new(MockArtifactRepository)
	storage := new(MockStoragePort)
	orchestrator := new(MockBuildOrchestratorPort)

	svc := service.NewBuildService(repo, storage, orchestrator)

	suiteID := "suite-2222"
	ctx := context.Background()
	sourceKey := "suites/suite-2222/sources/source.tar.gz"
	checksum := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"

	// No existing artifacts
	repo.On("ListBySuiteID", ctx, suiteID).Return([]*model.Artifact{}, nil)
	storage.On("Exists", mock.Anything, sourceKey).Return(true, nil)

	// Save newly created artifacts
	savedArtifacts := make(map[string]*model.Artifact)
	var mu sync.Mutex
	repo.On("Save", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
		art := args.Get(1).(*model.Artifact)
		mu.Lock()
		savedArtifacts[art.ID()] = art
		mu.Unlock()
	}).Return(nil)

	repo.On("FindByID", mock.Anything, mock.Anything).Return(func(ctx context.Context, id string) *model.Artifact {
		mu.Lock()
		defer mu.Unlock()
		return savedArtifacts[id]
	}, nil)

	storage.On("PresignDownload", mock.Anything, sourceKey, mock.Anything).Return("https://download", nil)
	storage.On("PresignUpload", mock.Anything, mock.Anything, mock.Anything).Return("https://upload", nil)
	orchestrator.On("DispatchBuildJob", mock.Anything, mock.Anything).Return("vuhive-build-job", nil)
	orchestrator.On("WaitForJob", mock.Anything, mock.Anything).Return(&outbound.BuildJobExecution{
		JobName:        "vuhive-build-job",
		ExitCode:       0,
		SHA256Checksum: checksum,
		Logs:           io.NopCloser(bytes.NewReader([]byte("success"))),
	}, nil)
	storage.On("Upload", mock.Anything, mock.Anything, mock.Anything, mock.Anything, "text/plain").Return(nil)

	artifacts, err := svc.BuildSuite(ctx, suiteID)
	require.NoError(t, err)
	require.Len(t, artifacts, 2)

	platforms := []model.Platform{artifacts[0].Platform(), artifacts[1].Platform()}
	assert.Contains(t, platforms, model.PlatformLinuxAmd64)
	assert.Contains(t, platforms, model.PlatformLinuxArm64)
	assert.Equal(t, model.ArtifactStatusReady, artifacts[0].Status())
	assert.Equal(t, model.ArtifactStatusReady, artifacts[1].Status())
}

func TestBuildService_ValidationAndQueries(t *testing.T) {
	repo := new(MockArtifactRepository)
	storage := new(MockStoragePort)
	orchestrator := new(MockBuildOrchestratorPort)

	svc := service.NewBuildService(repo, storage, orchestrator)
	ctx := context.Background()

	t.Run("GetArtifact successfully", func(t *testing.T) {
		artifact, _ := model.NewArtifact("suite-1", model.PlatformLinuxAmd64)
		repo.On("FindByID", ctx, artifact.ID()).Return(artifact, nil)

		res, err := svc.GetArtifact(ctx, artifact.ID())
		require.NoError(t, err)
		assert.Equal(t, artifact.ID(), res.ID())
	})

	t.Run("GetArtifact with empty ID fails", func(t *testing.T) {
		_, err := svc.GetArtifact(ctx, "")
		assert.ErrorIs(t, err, model.ErrValidation)
	})

	t.Run("ListArtifacts successfully", func(t *testing.T) {
		artifact, _ := model.NewArtifact("suite-1", model.PlatformLinuxAmd64)
		repo.On("ListBySuiteID", ctx, "suite-1").Return([]*model.Artifact{artifact}, nil)

		res, err := svc.ListArtifacts(ctx, "suite-1")
		require.NoError(t, err)
		require.Len(t, res, 1)
	})

	t.Run("ListArtifacts with empty suiteID fails", func(t *testing.T) {
		_, err := svc.ListArtifacts(ctx, "")
		assert.ErrorIs(t, err, model.ErrValidation)
	})

	t.Run("BuildArtifact fails when source tarball missing", func(t *testing.T) {
		artifact, _ := model.NewArtifact("suite-1", model.PlatformLinuxAmd64)
		repo.On("FindByID", ctx, artifact.ID()).Return(artifact, nil)
		storage.On("Exists", ctx, "suites/suite-1/sources/source.tar.gz").Return(false, nil)

		_, err := svc.BuildArtifact(ctx, "suite-1", artifact.ID())
		assert.ErrorIs(t, err, model.ErrNotFound)
	})

	t.Run("BuildArtifact fails when suiteID does not match artifact", func(t *testing.T) {
		artifact, _ := model.NewArtifact("suite-1", model.PlatformLinuxAmd64)
		repo.On("FindByID", ctx, artifact.ID()).Return(artifact, nil)

		_, err := svc.BuildArtifact(ctx, "wrong-suite", artifact.ID())
		assert.ErrorIs(t, err, model.ErrValidation)
	})
}

func TestBuildService_ImplementsUseCase(t *testing.T) {
	var _ inbound.BuildsUseCase = (*service.BuildService)(nil)
	assert.True(t, true)
}
