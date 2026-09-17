package model_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/morphy76/vuhive-cloud/internal/domain/model"
)

func TestNewSecret(t *testing.T) {
	t.Run("successfully create secret", func(t *testing.T) {
		s, err := model.NewSecret("suite-1", "DB_PASSWORD", []byte("encrypted-value"))
		require.NoError(t, err)
		require.NotNil(t, s)

		assert.NotEmpty(t, s.ID())
		assert.Equal(t, s.ID(), s.EntityID())
		assert.Equal(t, "suite-1", s.SuiteID())
		assert.Equal(t, "DB_PASSWORD", s.Key())
		assert.Equal(t, []byte("encrypted-value"), s.EncryptedValue())
		assert.False(t, s.CreatedAt().IsZero())
		assert.False(t, s.UpdatedAt().IsZero())
	})

	t.Run("trims whitespace from suite ID and key", func(t *testing.T) {
		s, err := model.NewSecret("  suite-1  ", "  API_KEY  ", []byte("val"))
		require.NoError(t, err)
		assert.Equal(t, "suite-1", s.SuiteID())
		assert.Equal(t, "API_KEY", s.Key())
	})

	t.Run("fail on empty suite ID", func(t *testing.T) {
		_, err := model.NewSecret("", "KEY", []byte("val"))
		assert.ErrorIs(t, err, model.ErrValidation)
	})

	t.Run("fail on empty key", func(t *testing.T) {
		_, err := model.NewSecret("suite-1", "", []byte("val"))
		assert.ErrorIs(t, err, model.ErrInvalidSecretKey)
	})

	t.Run("fail on whitespace-only key", func(t *testing.T) {
		_, err := model.NewSecret("suite-1", "   ", []byte("val"))
		assert.ErrorIs(t, err, model.ErrInvalidSecretKey)
	})

	t.Run("fail on nil encrypted value", func(t *testing.T) {
		_, err := model.NewSecret("suite-1", "KEY", nil)
		assert.ErrorIs(t, err, model.ErrValidation)
	})

	t.Run("fail on empty encrypted value", func(t *testing.T) {
		_, err := model.NewSecret("suite-1", "KEY", []byte{})
		assert.ErrorIs(t, err, model.ErrValidation)
	})
}

func TestValidateSecretKey(t *testing.T) {
	t.Run("valid keys", func(t *testing.T) {
		validKeys := []string{
			"DB_PASSWORD",
			"API_KEY",
			"MY_SECRET_123",
			"A",
			"A1_B2_C3",
			"UPPER_CASE_ONLY",
		}
		for _, key := range validKeys {
			assert.NoError(t, model.ValidateSecretKey(key), "expected valid: %s", key)
		}
	})

	t.Run("invalid keys", func(t *testing.T) {
		invalidKeys := []string{
			"",
			"   ",
			"lower_case",
			"has space",
			"has-dash",
			"has.dot",
			"123_STARTS_WITH_NUMBER",
			"_STARTS_WITH_UNDERSCORE",
			"HAS SPACE",
			"with/slash",
		}
		for _, key := range invalidKeys {
			assert.ErrorIs(t, model.ValidateSecretKey(key), model.ErrInvalidSecretKey, "expected invalid: %q", key)
		}
	})
}

func TestNewSecretWithID(t *testing.T) {
	now := time.Now().UTC()
	s, err := model.NewSecretWithID("sec-1", "suite-1", "MY_KEY", []byte("encrypted"), now, now)
	require.NoError(t, err)
	assert.Equal(t, "sec-1", s.ID())
	assert.Equal(t, "suite-1", s.SuiteID())
	assert.Equal(t, "MY_KEY", s.Key())
	assert.Equal(t, []byte("encrypted"), s.EncryptedValue())
	assert.Equal(t, now, s.CreatedAt())
	assert.Equal(t, now, s.UpdatedAt())
}

func TestSecret_UpdateValue(t *testing.T) {
	s, err := model.NewSecret("suite-1", "KEY", []byte("original"))
	require.NoError(t, err)
	originalUpdatedAt := s.UpdatedAt()

	t.Run("successfully update value", func(t *testing.T) {
		err := s.UpdateValue([]byte("new-encrypted"))
		require.NoError(t, err)
		assert.Equal(t, []byte("new-encrypted"), s.EncryptedValue())
		assert.True(t, s.UpdatedAt().After(originalUpdatedAt) || s.UpdatedAt().Equal(originalUpdatedAt))
	})

	t.Run("fail on nil value", func(t *testing.T) {
		err := s.UpdateValue(nil)
		assert.ErrorIs(t, err, model.ErrValidation)
	})

	t.Run("fail on empty value", func(t *testing.T) {
		err := s.UpdateValue([]byte{})
		assert.ErrorIs(t, err, model.ErrValidation)
	})
}
