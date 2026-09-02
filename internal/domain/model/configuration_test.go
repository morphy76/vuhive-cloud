package model_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/morphy76/vuhive-cloud/internal/domain/model"
)

func TestNewConfiguration(t *testing.T) {
	validYAML := `vus: 100
duration: 5m
`
	t.Run("successfully create configuration", func(t *testing.T) {
		cfg, err := model.NewConfiguration(
			"suite-1",
			"smoke-test",
			validYAML,
			"vuhive-configs/suite-1/smoke-test.yaml",
			true,
		)
		require.NoError(t, err)
		require.NotNil(t, cfg)

		assert.NotEmpty(t, cfg.ID())
		assert.Equal(t, cfg.ID(), cfg.EntityID())
		assert.Equal(t, "suite-1", cfg.SuiteID())
		assert.Equal(t, "smoke-test", cfg.Name())
		assert.Equal(t, validYAML, cfg.ContentYAML())
		assert.Equal(t, "vuhive-configs/suite-1/smoke-test.yaml", cfg.S3ConfigKey())
		assert.True(t, cfg.IsDefault())
		assert.False(t, cfg.CreatedAt().IsZero())
	})

	t.Run("fail validation on empty required fields", func(t *testing.T) {
		_, err := model.NewConfiguration("", "name", validYAML, "key", false)
		assert.ErrorIs(t, err, model.ErrValidation)

		_, err = model.NewConfiguration("suite-1", "", validYAML, "key", false)
		assert.ErrorIs(t, err, model.ErrEmptyName)

		_, err = model.NewConfiguration("suite-1", "name", "", "key", false)
		assert.ErrorIs(t, err, model.ErrValidation)

		_, err = model.NewConfiguration("suite-1", "name", validYAML, "", false)
		assert.ErrorIs(t, err, model.ErrEmptyS3Key)
	})
}

func TestConfiguration_Updates(t *testing.T) {
	cfg, err := model.NewConfiguration(
		"suite-1", "test-cfg", "initial: yaml", "s3-key", false,
	)
	require.NoError(t, err)

	t.Run("set default", func(t *testing.T) {
		cfg.SetDefault(true)
		assert.True(t, cfg.IsDefault())
		cfg.SetDefault(false)
		assert.False(t, cfg.IsDefault())
	})

	t.Run("update content", func(t *testing.T) {
		err := cfg.UpdateContent("new: content", "new-s3-key")
		require.NoError(t, err)
		assert.Equal(t, "new: content", cfg.ContentYAML())
		assert.Equal(t, "new-s3-key", cfg.S3ConfigKey())
	})

	t.Run("fail update with empty content or key", func(t *testing.T) {
		assert.ErrorIs(t, cfg.UpdateContent("", "key"), model.ErrValidation)
		assert.ErrorIs(t, cfg.UpdateContent("content", ""), model.ErrEmptyS3Key)
	})
}

func TestNewConfigurationWithID(t *testing.T) {
	now := time.Now().UTC()
	cfg, err := model.NewConfigurationWithID(
		"cfg-1", "suite-1", "name", "yaml", "key", true, now,
	)
	require.NoError(t, err)
	assert.Equal(t, "cfg-1", cfg.ID())
}
