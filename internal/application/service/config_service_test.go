package service_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/morphy76/vuhive-cloud/internal/application/ports/inbound"
	"github.com/morphy76/vuhive-cloud/internal/application/service"
	"github.com/morphy76/vuhive-cloud/internal/domain/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockConfigurationRepository mocks outbound.ConfigurationRepository
type MockConfigurationRepository struct {
	mock.Mock
}

func (m *MockConfigurationRepository) Save(ctx context.Context, config *model.Configuration) error {
	return m.Called(ctx, config).Error(0)
}

func (m *MockConfigurationRepository) FindByID(ctx context.Context, id string) (*model.Configuration, error) {
	args := m.Called(ctx, id)
	if c := args.Get(0); c != nil {
		return c.(*model.Configuration), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockConfigurationRepository) ListBySuiteID(ctx context.Context, suiteID string) ([]*model.Configuration, error) {
	args := m.Called(ctx, suiteID)
	if c := args.Get(0); c != nil {
		return c.([]*model.Configuration), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockConfigurationRepository) Delete(ctx context.Context, id string) error {
	return m.Called(ctx, id).Error(0)
}

func TestConfigService_CreateConfig(t *testing.T) {
	ctx := context.Background()

	t.Run("successfully creates config and uploads to storage", func(t *testing.T) {
		suite, _ := model.NewTestSuite("suite-1", "desc")
		suiteRepo := new(MockTestSuiteRepository)
		suiteRepo.On("FindByID", mock.Anything, suite.ID()).Return(suite, nil).Once()

		storage := new(MockStoragePort)
		storage.On("Upload", mock.Anything, mock.MatchedBy(func(key string) bool {
			return strings.HasPrefix(key, "suites/"+suite.ID()+"/configs/") && strings.HasSuffix(key, ".yaml")
		}), mock.Anything, int64(len("scenario: test")), "application/x-yaml").Return(nil).Once()

		configRepo := new(MockConfigurationRepository)
		configRepo.On("Save", mock.Anything, mock.MatchedBy(func(c *model.Configuration) bool {
			return c.SuiteID() == suite.ID() && c.Name() == "default" && c.ContentYAML() == "scenario: test" && c.IsDefault()
		})).Return(nil).Once()

		svc := service.NewConfigService(suiteRepo, configRepo, storage)
		cfg, err := svc.CreateConfig(ctx, inbound.CreateConfigCommand{
			SuiteID:     suite.ID(),
			Name:        "default",
			ContentYAML: "scenario: test",
			IsDefault:   true,
		})

		require.NoError(t, err)
		require.NotNil(t, cfg)
		assert.Equal(t, suite.ID(), cfg.SuiteID())
		assert.Equal(t, "default", cfg.Name())
		assert.Equal(t, "scenario: test", cfg.ContentYAML())
		assert.True(t, cfg.IsDefault())
		assert.NotEmpty(t, cfg.S3ConfigKey())

		suiteRepo.AssertExpectations(t)
		storage.AssertExpectations(t)
		configRepo.AssertExpectations(t)
	})

	t.Run("fails when parent suite does not exist", func(t *testing.T) {
		suiteRepo := new(MockTestSuiteRepository)
		suiteRepo.On("FindByID", mock.Anything, "missing-suite").Return(nil, model.ErrNotFound).Once()

		configRepo := new(MockConfigurationRepository)
		storage := new(MockStoragePort)

		svc := service.NewConfigService(suiteRepo, configRepo, storage)
		cfg, err := svc.CreateConfig(ctx, inbound.CreateConfigCommand{
			SuiteID:     "missing-suite",
			Name:        "default",
			ContentYAML: "scenario: test",
		})

		require.Error(t, err)
		assert.ErrorIs(t, err, model.ErrNotFound)
		assert.Nil(t, cfg)
	})

	t.Run("fails when name is empty", func(t *testing.T) {
		suite, _ := model.NewTestSuite("suite-1", "desc")
		suiteRepo := new(MockTestSuiteRepository)
		suiteRepo.On("FindByID", mock.Anything, suite.ID()).Return(suite, nil).Once()

		configRepo := new(MockConfigurationRepository)
		storage := new(MockStoragePort)

		svc := service.NewConfigService(suiteRepo, configRepo, storage)
		cfg, err := svc.CreateConfig(ctx, inbound.CreateConfigCommand{
			SuiteID:     suite.ID(),
			Name:        "   ",
			ContentYAML: "scenario: test",
		})

		require.Error(t, err)
		assert.ErrorIs(t, err, model.ErrEmptyName)
		assert.Nil(t, cfg)
	})

	t.Run("fails when content YAML is empty", func(t *testing.T) {
		suite, _ := model.NewTestSuite("suite-1", "desc")
		suiteRepo := new(MockTestSuiteRepository)
		suiteRepo.On("FindByID", mock.Anything, suite.ID()).Return(suite, nil).Once()

		configRepo := new(MockConfigurationRepository)
		storage := new(MockStoragePort)

		svc := service.NewConfigService(suiteRepo, configRepo, storage)
		cfg, err := svc.CreateConfig(ctx, inbound.CreateConfigCommand{
			SuiteID:     suite.ID(),
			Name:        "default",
			ContentYAML: "   ",
		})

		require.Error(t, err)
		assert.ErrorIs(t, err, model.ErrValidation)
		assert.Nil(t, cfg)
	})

	t.Run("fails when storage upload fails", func(t *testing.T) {
		suite, _ := model.NewTestSuite("suite-1", "desc")
		suiteRepo := new(MockTestSuiteRepository)
		suiteRepo.On("FindByID", mock.Anything, suite.ID()).Return(suite, nil).Once()

		storage := new(MockStoragePort)
		storage.On("Upload", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Return(errors.New("s3 upload failed")).Once()

		configRepo := new(MockConfigurationRepository)
		svc := service.NewConfigService(suiteRepo, configRepo, storage)

		cfg, err := svc.CreateConfig(ctx, inbound.CreateConfigCommand{
			SuiteID:     suite.ID(),
			Name:        "default",
			ContentYAML: "scenario: test",
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "s3 upload failed")
		assert.Nil(t, cfg)
	})
}

func TestConfigService_GetConfig(t *testing.T) {
	ctx := context.Background()

	t.Run("successfully retrieves configuration belonging to suite", func(t *testing.T) {
		cfg, _ := model.NewConfiguration("suite-1", "default", "scenario: test", "s3/key", true)

		configRepo := new(MockConfigurationRepository)
		configRepo.On("FindByID", mock.Anything, cfg.ID()).Return(cfg, nil).Once()

		svc := service.NewConfigService(nil, configRepo, nil)
		found, err := svc.GetConfig(ctx, "suite-1", cfg.ID())

		require.NoError(t, err)
		assert.Equal(t, cfg.ID(), found.ID())
		assert.Equal(t, "default", found.Name())
	})

	t.Run("returns ErrNotFound when configuration belongs to another suite", func(t *testing.T) {
		cfg, _ := model.NewConfiguration("suite-1", "default", "scenario: test", "s3/key", true)

		configRepo := new(MockConfigurationRepository)
		configRepo.On("FindByID", mock.Anything, cfg.ID()).Return(cfg, nil).Once()

		svc := service.NewConfigService(nil, configRepo, nil)
		found, err := svc.GetConfig(ctx, "different-suite", cfg.ID())

		require.Error(t, err)
		assert.ErrorIs(t, err, model.ErrNotFound)
		assert.Nil(t, found)
	})

	t.Run("returns ErrNotFound when configuration does not exist", func(t *testing.T) {
		configRepo := new(MockConfigurationRepository)
		configRepo.On("FindByID", mock.Anything, "missing-id").Return(nil, model.ErrNotFound).Once()

		svc := service.NewConfigService(nil, configRepo, nil)
		found, err := svc.GetConfig(ctx, "suite-1", "missing-id")

		require.Error(t, err)
		assert.ErrorIs(t, err, model.ErrNotFound)
		assert.Nil(t, found)
	})
}

func TestConfigService_ListConfigs(t *testing.T) {
	ctx := context.Background()

	t.Run("successfully lists configurations for valid suite", func(t *testing.T) {
		suite, _ := model.NewTestSuite("suite-1", "desc")
		suiteRepo := new(MockTestSuiteRepository)
		suiteRepo.On("FindByID", mock.Anything, suite.ID()).Return(suite, nil).Once()

		c1, _ := model.NewConfiguration(suite.ID(), "c1", "yaml1", "k1", true)
		c2, _ := model.NewConfiguration(suite.ID(), "c2", "yaml2", "k2", false)

		configRepo := new(MockConfigurationRepository)
		configRepo.On("ListBySuiteID", mock.Anything, suite.ID()).Return([]*model.Configuration{c1, c2}, nil).Once()

		svc := service.NewConfigService(suiteRepo, configRepo, nil)
		configs, err := svc.ListConfigs(ctx, suite.ID())

		require.NoError(t, err)
		assert.Len(t, configs, 2)
		assert.Equal(t, "c1", configs[0].Name())
		assert.Equal(t, "c2", configs[1].Name())
	})

	t.Run("fails when parent suite does not exist", func(t *testing.T) {
		suiteRepo := new(MockTestSuiteRepository)
		suiteRepo.On("FindByID", mock.Anything, "missing-suite").Return(nil, model.ErrNotFound).Once()

		configRepo := new(MockConfigurationRepository)
		svc := service.NewConfigService(suiteRepo, configRepo, nil)

		configs, err := svc.ListConfigs(ctx, "missing-suite")
		require.Error(t, err)
		assert.ErrorIs(t, err, model.ErrNotFound)
		assert.Nil(t, configs)
	})
}

func TestConfigService_DeleteConfig(t *testing.T) {
	ctx := context.Background()

	t.Run("successfully deletes configuration and removes S3 object", func(t *testing.T) {
		cfg, _ := model.NewConfiguration("suite-1", "default", "scenario: test", "suites/suite-1/configs/c1.yaml", true)

		configRepo := new(MockConfigurationRepository)
		configRepo.On("FindByID", mock.Anything, cfg.ID()).Return(cfg, nil).Once()
		configRepo.On("Delete", mock.Anything, cfg.ID()).Return(nil).Once()

		storage := new(MockStoragePort)
		storage.On("Delete", mock.Anything, cfg.S3ConfigKey()).Return(nil).Once()

		svc := service.NewConfigService(nil, configRepo, storage)
		err := svc.DeleteConfig(ctx, "suite-1", cfg.ID())

		require.NoError(t, err)
		configRepo.AssertExpectations(t)
		storage.AssertExpectations(t)
	})

	t.Run("returns ErrNotFound when configuration belongs to another suite", func(t *testing.T) {
		cfg, _ := model.NewConfiguration("suite-1", "default", "scenario: test", "s3/key", true)

		configRepo := new(MockConfigurationRepository)
		configRepo.On("FindByID", mock.Anything, cfg.ID()).Return(cfg, nil).Once()

		svc := service.NewConfigService(nil, configRepo, nil)
		err := svc.DeleteConfig(ctx, "different-suite", cfg.ID())

		require.Error(t, err)
		assert.ErrorIs(t, err, model.ErrNotFound)
	})

	t.Run("returns ErrNotFound when configuration does not exist", func(t *testing.T) {
		configRepo := new(MockConfigurationRepository)
		configRepo.On("FindByID", mock.Anything, "missing-id").Return(nil, model.ErrNotFound).Once()

		svc := service.NewConfigService(nil, configRepo, nil)
		err := svc.DeleteConfig(ctx, "suite-1", "missing-id")

		require.Error(t, err)
		assert.ErrorIs(t, err, model.ErrNotFound)
	})
}

func TestConfigService_UpdateConfig(t *testing.T) {
	ctx := context.Background()

	t.Run("successfully updates configuration name, yaml content, and isDefault", func(t *testing.T) {
		suite, _ := model.NewTestSuite("suite-1", "desc")
		suiteRepo := new(MockTestSuiteRepository)
		suiteRepo.On("FindByID", mock.Anything, suite.ID()).Return(suite, nil).Once()

		cfg, _ := model.NewConfiguration(suite.ID(), "old-name", "duration: 1m", "suites/"+suite.ID()+"/configs/c1.yaml", false)
		configRepo := new(MockConfigurationRepository)
		configRepo.On("FindByID", mock.Anything, cfg.ID()).Return(cfg, nil).Once()

		storage := new(MockStoragePort)
		storage.On("Upload", mock.Anything, cfg.S3ConfigKey(), mock.Anything, int64(len("duration: 5m\nconcurrency: 100\n")), "application/x-yaml").Return(nil).Once()

		configRepo.On("Save", mock.Anything, mock.MatchedBy(func(c *model.Configuration) bool {
			return c.ID() == cfg.ID() && c.Name() == "new-name" && c.ContentYAML() == "duration: 5m\nconcurrency: 100\n" && c.IsDefault()
		})).Return(nil).Once()

		svc := service.NewConfigService(suiteRepo, configRepo, storage)
		isDefault := true
		updated, err := svc.UpdateConfig(ctx, inbound.UpdateConfigCommand{
			SuiteID:     suite.ID(),
			ConfigID:    cfg.ID(),
			Name:        "new-name",
			ContentYAML: "duration: 5m\nconcurrency: 100\n",
			IsDefault:   &isDefault,
		})

		require.NoError(t, err)
		require.NotNil(t, updated)
		assert.Equal(t, "new-name", updated.Name())
		assert.Equal(t, "duration: 5m\nconcurrency: 100\n", updated.ContentYAML())
		assert.True(t, updated.IsDefault())

		suiteRepo.AssertExpectations(t)
		configRepo.AssertExpectations(t)
		storage.AssertExpectations(t)
	})

	t.Run("returns ErrNotFound when configuration belongs to another suite", func(t *testing.T) {
		suite, _ := model.NewTestSuite("suite-1", "desc")
		suiteRepo := new(MockTestSuiteRepository)
		suiteRepo.On("FindByID", mock.Anything, suite.ID()).Return(suite, nil).Once()

		cfg, _ := model.NewConfiguration("other-suite", "cfg-name", "duration: 1m", "key", false)
		configRepo := new(MockConfigurationRepository)
		configRepo.On("FindByID", mock.Anything, cfg.ID()).Return(cfg, nil).Once()

		svc := service.NewConfigService(suiteRepo, configRepo, nil)
		_, err := svc.UpdateConfig(ctx, inbound.UpdateConfigCommand{
			SuiteID:     suite.ID(),
			ConfigID:    cfg.ID(),
			Name:        "new-name",
			ContentYAML: "duration: 2m",
		})

		require.Error(t, err)
		assert.ErrorIs(t, err, model.ErrNotFound)
	})

	t.Run("fails when validation fails on empty fields", func(t *testing.T) {
		suite, _ := model.NewTestSuite("suite-1", "desc")
		suiteRepo := new(MockTestSuiteRepository)
		suiteRepo.On("FindByID", mock.Anything, suite.ID()).Return(suite, nil).Once()

		cfg, _ := model.NewConfiguration(suite.ID(), "old-name", "duration: 1m", "key", false)
		configRepo := new(MockConfigurationRepository)
		configRepo.On("FindByID", mock.Anything, cfg.ID()).Return(cfg, nil).Once()

		svc := service.NewConfigService(suiteRepo, configRepo, nil)
		_, err := svc.UpdateConfig(ctx, inbound.UpdateConfigCommand{
			SuiteID:     suite.ID(),
			ConfigID:    cfg.ID(),
			Name:        "",
			ContentYAML: "duration: 2m",
		})

		require.Error(t, err)
		assert.ErrorIs(t, err, model.ErrEmptyName)
	})
}
