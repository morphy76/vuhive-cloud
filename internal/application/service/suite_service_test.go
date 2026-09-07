package service_test

import (
	"context"
	"errors"
	"testing"

	"github.com/morphy76/vuhive-cloud/internal/application/ports/inbound"
	"github.com/morphy76/vuhive-cloud/internal/application/service"
	"github.com/morphy76/vuhive-cloud/internal/domain/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestSuiteService_CreateSuite(t *testing.T) {
	ctx := context.Background()

	t.Run("successfully creates a new suite in DRAFT state", func(t *testing.T) {
		repo := new(MockTestSuiteRepository)
		repo.On("Save", mock.Anything, mock.MatchedBy(func(s *model.TestSuite) bool {
			return s.Name() == "load-test-1" && s.Description() == "Main service test" && s.State() == model.TestSuiteStateDraft
		})).Return(nil).Once()

		svc := service.NewSuiteService(repo)
		suite, err := svc.CreateSuite(ctx, "load-test-1", "Main service test")

		require.NoError(t, err)
		require.NotNil(t, suite)
		assert.Equal(t, "load-test-1", suite.Name())
		assert.Equal(t, "Main service test", suite.Description())
		assert.Equal(t, model.TestSuiteStateDraft, suite.State())
		repo.AssertExpectations(t)
	})

	t.Run("fails when name is empty or whitespace", func(t *testing.T) {
		repo := new(MockTestSuiteRepository)
		svc := service.NewSuiteService(repo)

		suite, err := svc.CreateSuite(ctx, "   ", "some desc")
		require.Error(t, err)
		assert.ErrorIs(t, err, model.ErrEmptyName)
		assert.Nil(t, suite)
	})

	t.Run("fails when repository Save returns an error", func(t *testing.T) {
		repo := new(MockTestSuiteRepository)
		repo.On("Save", mock.Anything, mock.Anything).Return(errors.New("db error")).Once()

		svc := service.NewSuiteService(repo)
		suite, err := svc.CreateSuite(ctx, "valid-name", "desc")
		require.Error(t, err)
		assert.Nil(t, suite)
		repo.AssertExpectations(t)
	})
}

func TestSuiteService_GetSuite(t *testing.T) {
	ctx := context.Background()

	t.Run("successfully retrieves an existing suite", func(t *testing.T) {
		existing, err := model.NewTestSuite("suite-1", "desc")
		require.NoError(t, err)

		repo := new(MockTestSuiteRepository)
		repo.On("FindByID", mock.Anything, existing.ID()).Return(existing, nil).Once()

		svc := service.NewSuiteService(repo)
		found, err := svc.GetSuite(ctx, existing.ID())

		require.NoError(t, err)
		require.NotNil(t, found)
		assert.Equal(t, existing.ID(), found.ID())
		assert.Equal(t, "suite-1", found.Name())
		repo.AssertExpectations(t)
	})

	t.Run("returns ErrNotFound when suite does not exist", func(t *testing.T) {
		repo := new(MockTestSuiteRepository)
		repo.On("FindByID", mock.Anything, "non-existent").Return(nil, model.ErrNotFound).Once()

		svc := service.NewSuiteService(repo)
		found, err := svc.GetSuite(ctx, "non-existent")

		require.Error(t, err)
		assert.ErrorIs(t, err, model.ErrNotFound)
		assert.Nil(t, found)
		repo.AssertExpectations(t)
	})

	t.Run("returns ErrValidation when id is empty", func(t *testing.T) {
		repo := new(MockTestSuiteRepository)
		svc := service.NewSuiteService(repo)

		found, err := svc.GetSuite(ctx, "  ")
		require.Error(t, err)
		assert.ErrorIs(t, err, model.ErrValidation)
		assert.Nil(t, found)
	})
}

func TestSuiteService_ListSuites(t *testing.T) {
	ctx := context.Background()

	t.Run("successfully lists all suites", func(t *testing.T) {
		s1, _ := model.NewTestSuite("suite-1", "desc1")
		s2, _ := model.NewTestSuite("suite-2", "desc2")

		repo := new(MockTestSuiteRepository)
		repo.On("List", mock.Anything).Return([]*model.TestSuite{s1, s2}, nil).Once()

		svc := service.NewSuiteService(repo)
		suites, err := svc.ListSuites(ctx)

		require.NoError(t, err)
		assert.Len(t, suites, 2)
		assert.Equal(t, "suite-1", suites[0].Name())
		assert.Equal(t, "suite-2", suites[1].Name())
		repo.AssertExpectations(t)
	})

	t.Run("returns empty slice when no suites exist", func(t *testing.T) {
		repo := new(MockTestSuiteRepository)
		repo.On("List", mock.Anything).Return([]*model.TestSuite{}, nil).Once()

		svc := service.NewSuiteService(repo)
		suites, err := svc.ListSuites(ctx)

		require.NoError(t, err)
		assert.Empty(t, suites)
		repo.AssertExpectations(t)
	})
}

func TestSuiteService_UpdateSuite(t *testing.T) {
	ctx := context.Background()

	t.Run("successfully updates details and state", func(t *testing.T) {
		suite, _ := model.NewTestSuite("old-name", "old-desc")
		repo := new(MockTestSuiteRepository)
		repo.On("FindByID", mock.Anything, suite.ID()).Return(suite, nil).Once()
		repo.On("Save", mock.Anything, mock.MatchedBy(func(s *model.TestSuite) bool {
			return s.Name() == "new-name" && s.Description() == "new-desc" && s.State() == model.TestSuiteStateActive
		})).Return(nil).Once()

		activeState := model.TestSuiteStateActive
		svc := service.NewSuiteService(repo)
		updated, err := svc.UpdateSuite(ctx, suite.ID(), inbound.UpdateSuiteCommand{
			Name:        "new-name",
			Description: "new-desc",
			State:       &activeState,
		})

		require.NoError(t, err)
		require.NotNil(t, updated)
		assert.Equal(t, "new-name", updated.Name())
		assert.Equal(t, "new-desc", updated.Description())
		assert.Equal(t, model.TestSuiteStateActive, updated.State())
		repo.AssertExpectations(t)
	})

	t.Run("fails when suite not found", func(t *testing.T) {
		repo := new(MockTestSuiteRepository)
		repo.On("FindByID", mock.Anything, "missing-id").Return(nil, model.ErrNotFound).Once()

		svc := service.NewSuiteService(repo)
		updated, err := svc.UpdateSuite(ctx, "missing-id", inbound.UpdateSuiteCommand{
			Name: "valid-name",
		})

		require.Error(t, err)
		assert.ErrorIs(t, err, model.ErrNotFound)
		assert.Nil(t, updated)
		repo.AssertExpectations(t)
	})

	t.Run("fails with empty name", func(t *testing.T) {
		suite, _ := model.NewTestSuite("old-name", "old-desc")
		repo := new(MockTestSuiteRepository)
		repo.On("FindByID", mock.Anything, suite.ID()).Return(suite, nil).Once()

		svc := service.NewSuiteService(repo)
		updated, err := svc.UpdateSuite(ctx, suite.ID(), inbound.UpdateSuiteCommand{
			Name: "   ",
		})

		require.Error(t, err)
		assert.ErrorIs(t, err, model.ErrEmptyName)
		assert.Nil(t, updated)
	})

	t.Run("fails with invalid state", func(t *testing.T) {
		suite, _ := model.NewTestSuite("old-name", "old-desc")
		repo := new(MockTestSuiteRepository)
		repo.On("FindByID", mock.Anything, suite.ID()).Return(suite, nil).Once()

		invalidState := model.TestSuiteState("BAD_STATE")
		svc := service.NewSuiteService(repo)
		updated, err := svc.UpdateSuite(ctx, suite.ID(), inbound.UpdateSuiteCommand{
			Name:  "valid-name",
			State: &invalidState,
		})

		require.Error(t, err)
		assert.ErrorIs(t, err, model.ErrInvalidStateTransition)
		assert.Nil(t, updated)
	})
}

func TestSuiteService_DeleteSuite(t *testing.T) {
	ctx := context.Background()

	t.Run("successfully deletes a suite", func(t *testing.T) {
		repo := new(MockTestSuiteRepository)
		repo.On("Delete", mock.Anything, "suite-to-delete").Return(nil).Once()

		svc := service.NewSuiteService(repo)
		err := svc.DeleteSuite(ctx, "suite-to-delete")

		require.NoError(t, err)
		repo.AssertExpectations(t)
	})

	t.Run("returns ErrValidation when id is empty", func(t *testing.T) {
		repo := new(MockTestSuiteRepository)
		svc := service.NewSuiteService(repo)

		err := svc.DeleteSuite(ctx, "  ")
		require.Error(t, err)
		assert.ErrorIs(t, err, model.ErrValidation)
	})

	t.Run("returns ErrNotFound when suite to delete is not found", func(t *testing.T) {
		repo := new(MockTestSuiteRepository)
		repo.On("Delete", mock.Anything, "missing-id").Return(model.ErrNotFound).Once()

		svc := service.NewSuiteService(repo)
		err := svc.DeleteSuite(ctx, "missing-id")

		require.Error(t, err)
		assert.ErrorIs(t, err, model.ErrNotFound)
		repo.AssertExpectations(t)
	})
}

func TestSuiteService_ArchiveSuite(t *testing.T) {
	ctx := context.Background()

	t.Run("successfully archives a suite", func(t *testing.T) {
		suite, _ := model.NewTestSuite("active-suite", "desc")
		_ = suite.Activate()

		repo := new(MockTestSuiteRepository)
		repo.On("FindByID", mock.Anything, suite.ID()).Return(suite, nil).Once()
		repo.On("Save", mock.Anything, mock.MatchedBy(func(s *model.TestSuite) bool {
			return s.State() == model.TestSuiteStateArchived
		})).Return(nil).Once()

		svc := service.NewSuiteService(repo)
		err := svc.ArchiveSuite(ctx, suite.ID())

		require.NoError(t, err)
		assert.Equal(t, model.TestSuiteStateArchived, suite.State())
		repo.AssertExpectations(t)
	})
}
