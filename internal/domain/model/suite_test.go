package model_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/morphy76/vuhive-cloud/internal/domain/model"
)

func TestNewTestSuite(t *testing.T) {
	t.Run("successfully create a new test suite in DRAFT state", func(t *testing.T) {
		suite, err := model.NewTestSuite("checkout-load-test", "E-commerce checkout stress test")
		require.NoError(t, err)
		require.NotNil(t, suite)

		assert.NotEmpty(t, suite.ID())
		assert.Equal(t, "checkout-load-test", suite.Name())
		assert.Equal(t, "E-commerce checkout stress test", suite.Description())
		assert.Equal(t, model.TestSuiteStateDraft, suite.State())
		assert.False(t, suite.CreatedAt().IsZero())
		assert.False(t, suite.UpdatedAt().IsZero())
		assert.Equal(t, "checkout-load-test", suite.Name())
		assert.Equal(t, "TestSuite", suite.AggregateType())
		assert.Equal(t, suite.ID(), suite.EntityID())
	})

	t.Run("fail when name is empty or only whitespace", func(t *testing.T) {
		suite, err := model.NewTestSuite("", "Description")
		assert.ErrorIs(t, err, model.ErrEmptyName)
		assert.Nil(t, suite)

		suite, err = model.NewTestSuite("   ", "Description")
		assert.ErrorIs(t, err, model.ErrEmptyName)
		assert.Nil(t, suite)
	})
}

func TestTestSuite_StateTransitions(t *testing.T) {
	t.Run("transition from DRAFT to ACTIVE", func(t *testing.T) {
		suite, err := model.NewTestSuite("test-suite", "desc")
		require.NoError(t, err)
		assert.Equal(t, model.TestSuiteStateDraft, suite.State())

		err = suite.Activate()
		require.NoError(t, err)
		assert.Equal(t, model.TestSuiteStateActive, suite.State())
	})

	t.Run("fail activating an already ACTIVE suite", func(t *testing.T) {
		suite, err := model.NewTestSuite("test-suite", "desc")
		require.NoError(t, err)
		require.NoError(t, suite.Activate())

		err = suite.Activate()
		assert.ErrorIs(t, err, model.ErrInvalidStateTransition)
		assert.Equal(t, model.TestSuiteStateActive, suite.State())
	})

	t.Run("transition from ACTIVE to ARCHIVED", func(t *testing.T) {
		suite, err := model.NewTestSuite("test-suite", "desc")
		require.NoError(t, err)
		require.NoError(t, suite.Activate())

		err = suite.Archive()
		require.NoError(t, err)
		assert.Equal(t, model.TestSuiteStateArchived, suite.State())
	})

	t.Run("transition from DRAFT directly to ARCHIVED", func(t *testing.T) {
		suite, err := model.NewTestSuite("test-suite", "desc")
		require.NoError(t, err)

		err = suite.Archive()
		require.NoError(t, err)
		assert.Equal(t, model.TestSuiteStateArchived, suite.State())
	})

	t.Run("fail archiving an already ARCHIVED suite", func(t *testing.T) {
		suite, err := model.NewTestSuite("test-suite", "desc")
		require.NoError(t, err)
		require.NoError(t, suite.Archive())

		err = suite.Archive()
		assert.ErrorIs(t, err, model.ErrInvalidStateTransition)
	})

	t.Run("reactivate from ARCHIVED to ACTIVE", func(t *testing.T) {
		suite, err := model.NewTestSuite("test-suite", "desc")
		require.NoError(t, err)
		require.NoError(t, suite.Archive())

		err = suite.Activate()
		require.NoError(t, err)
		assert.Equal(t, model.TestSuiteStateActive, suite.State())
	})
}

func TestTestSuite_UpdateDetails(t *testing.T) {
	t.Run("update name and description successfully", func(t *testing.T) {
		suite, err := model.NewTestSuite("old-name", "old-desc")
		require.NoError(t, err)

		beforeUpdate := suite.UpdatedAt()
		time.Sleep(10 * time.Millisecond)

		err = suite.UpdateDetails("new-name", "new-desc")
		require.NoError(t, err)
		assert.Equal(t, "new-name", suite.Name())
		assert.Equal(t, "new-desc", suite.Description())
		assert.True(t, suite.UpdatedAt().After(beforeUpdate))
	})

	t.Run("fail update with empty name", func(t *testing.T) {
		suite, err := model.NewTestSuite("valid-name", "desc")
		require.NoError(t, err)

		err = suite.UpdateDetails("", "new-desc")
		assert.ErrorIs(t, err, model.ErrEmptyName)
	})

	t.Run("fail update when suite is archived", func(t *testing.T) {
		suite, err := model.NewTestSuite("valid-name", "desc")
		require.NoError(t, err)
		require.NoError(t, suite.Archive())

		err = suite.UpdateDetails("new-name", "new-desc")
		assert.ErrorIs(t, err, model.ErrInvalidStateTransition)
	})
}

func TestNewTestSuiteWithID(t *testing.T) {
	t.Run("reconstitute valid test suite", func(t *testing.T) {
		now := time.Now()
		suite, err := model.NewTestSuiteWithID("custom-id", "suite-name", "desc", model.TestSuiteStateActive, now, now)
		require.NoError(t, err)
		assert.Equal(t, "custom-id", suite.ID())
		assert.Equal(t, model.TestSuiteStateActive, suite.State())
	})

	t.Run("reconstitute with invalid state fails", func(t *testing.T) {
		now := time.Now()
		_, err := model.NewTestSuiteWithID("custom-id", "suite-name", "desc", "UNKNOWN_STATE", now, now)
		assert.ErrorIs(t, err, model.ErrInvalidStateTransition)
	})
}
