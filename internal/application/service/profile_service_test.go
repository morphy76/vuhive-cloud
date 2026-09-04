package service_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/morphy76/vuhive-cloud/internal/application/ports/inbound"
	"github.com/morphy76/vuhive-cloud/internal/application/service"
	"github.com/morphy76/vuhive-cloud/internal/domain/model"
)

// MockRunnerProfileRepository mocks outbound.RunnerProfileRepository
type MockRunnerProfileRepository struct {
	mock.Mock
}

func (m *MockRunnerProfileRepository) Save(ctx context.Context, profile *model.RunnerProfile) error {
	return m.Called(ctx, profile).Error(0)
}

func (m *MockRunnerProfileRepository) FindByID(ctx context.Context, id string) (*model.RunnerProfile, error) {
	args := m.Called(ctx, id)
	if p := args.Get(0); p != nil {
		return p.(*model.RunnerProfile), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockRunnerProfileRepository) FindByName(ctx context.Context, name string) (*model.RunnerProfile, error) {
	args := m.Called(ctx, name)
	if p := args.Get(0); p != nil {
		return p.(*model.RunnerProfile), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockRunnerProfileRepository) List(ctx context.Context) ([]*model.RunnerProfile, error) {
	args := m.Called(ctx)
	if p := args.Get(0); p != nil {
		return p.([]*model.RunnerProfile), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockRunnerProfileRepository) Delete(ctx context.Context, id string) error {
	return m.Called(ctx, id).Error(0)
}

func TestProfileService_CreateProfile(t *testing.T) {
	ctx := context.Background()

	t.Run("successfully create runner profile with custom values", func(t *testing.T) {
		repo := new(MockRunnerProfileRepository)
		svc := service.NewProfileService(repo)

		cmd := inbound.CreateProfileCommand{
			Name:          "perf-c6i",
			Description:   "Compute optimized runner",
			RunnerImage:   "my-registry.io/runner:v1",
			CPURequest:    "2000m",
			CPULimit:      "4000m",
			MemoryRequest: "2Gi",
			MemoryLimit:   "4Gi",
			NodeSelector:  map[string]string{"node.kubernetes.io/instance-type": "c6i.xlarge"},
			Affinity: model.Affinity{
				NodeSelectorTerms: []model.NodeAffinityTerm{
					{Key: "topology.kubernetes.io/zone", Operator: "In", Values: []string{"eu-west-1a"}},
				},
			},
			Tolerations: []model.Toleration{
				{Key: "dedicated", Operator: "Exists", Effect: "NoSchedule"},
			},
		}

		repo.On("Save", ctx, mock.MatchedBy(func(p *model.RunnerProfile) bool {
			return p.Name() == "perf-c6i" &&
				p.RunnerImage() == "my-registry.io/runner:v1" &&
				p.Resources().CPURequest() == "2000m" &&
				p.Resources().CPULimit() == "4000m" &&
				p.Resources().MemoryRequest() == "2Gi" &&
				p.Resources().MemoryLimit() == "4Gi"
		})).Return(nil)

		profile, err := svc.CreateProfile(ctx, cmd)
		require.NoError(t, err)
		require.NotNil(t, profile)
		assert.Equal(t, "perf-c6i", profile.Name())
		assert.Equal(t, "my-registry.io/runner:v1", profile.RunnerImage())
		repo.AssertExpectations(t)
	})

	t.Run("successfully create runner profile applying schema defaults", func(t *testing.T) {
		repo := new(MockRunnerProfileRepository)
		svc := service.NewProfileService(repo)

		cmd := inbound.CreateProfileCommand{
			Name: "default-profile",
		}

		repo.On("Save", ctx, mock.MatchedBy(func(p *model.RunnerProfile) bool {
			return p.Name() == "default-profile" &&
				p.RunnerImage() == "alpine:3.20" &&
				p.Resources().CPURequest() == "1000m" &&
				p.Resources().CPULimit() == "2000m" &&
				p.Resources().MemoryRequest() == "1Gi" &&
				p.Resources().MemoryLimit() == "2Gi"
		})).Return(nil)

		profile, err := svc.CreateProfile(ctx, cmd)
		require.NoError(t, err)
		require.NotNil(t, profile)
		assert.Equal(t, "alpine:3.20", profile.RunnerImage())
		assert.Equal(t, "1000m", profile.Resources().CPURequest())
		assert.Equal(t, "2000m", profile.Resources().CPULimit())
		assert.Equal(t, "1Gi", profile.Resources().MemoryRequest())
		assert.Equal(t, "2Gi", profile.Resources().MemoryLimit())
		repo.AssertExpectations(t)
	})

	t.Run("fail when name is empty", func(t *testing.T) {
		repo := new(MockRunnerProfileRepository)
		svc := service.NewProfileService(repo)

		cmd := inbound.CreateProfileCommand{
			Name: "   ",
		}

		_, err := svc.CreateProfile(ctx, cmd)
		assert.ErrorIs(t, err, model.ErrEmptyName)
	})

	t.Run("fail when resource quantities violate k8s limits", func(t *testing.T) {
		repo := new(MockRunnerProfileRepository)
		svc := service.NewProfileService(repo)

		cmd := inbound.CreateProfileCommand{
			Name:       "invalid-cpu",
			CPURequest: "4000m",
			CPULimit:   "2000m",
		}

		_, err := svc.CreateProfile(ctx, cmd)
		assert.ErrorIs(t, err, model.ErrInvalidResourceQuantity)
	})

	t.Run("fail when affinity violates k8s schema", func(t *testing.T) {
		repo := new(MockRunnerProfileRepository)
		svc := service.NewProfileService(repo)

		cmd := inbound.CreateProfileCommand{
			Name: "invalid-affinity",
			Affinity: model.Affinity{
				NodeSelectorTerms: []model.NodeAffinityTerm{
					{Key: "", Operator: "In", Values: []string{"zone-a"}},
				},
			},
		}

		_, err := svc.CreateProfile(ctx, cmd)
		assert.ErrorIs(t, err, model.ErrInvalidAffinity)
	})

	t.Run("fail when toleration violates k8s schema", func(t *testing.T) {
		repo := new(MockRunnerProfileRepository)
		svc := service.NewProfileService(repo)

		cmd := inbound.CreateProfileCommand{
			Name: "invalid-toleration",
			Tolerations: []model.Toleration{
				{Key: "key", Operator: "BadOp"},
			},
		}

		_, err := svc.CreateProfile(ctx, cmd)
		assert.ErrorIs(t, err, model.ErrInvalidToleration)
	})

	t.Run("propagate repository save error", func(t *testing.T) {
		repo := new(MockRunnerProfileRepository)
		svc := service.NewProfileService(repo)

		cmd := inbound.CreateProfileCommand{
			Name: "collision-profile",
		}

		repo.On("Save", ctx, mock.Anything).Return(model.ErrConflict)

		_, err := svc.CreateProfile(ctx, cmd)
		assert.ErrorIs(t, err, model.ErrConflict)
		repo.AssertExpectations(t)
	})
}

func TestProfileService_GetProfile(t *testing.T) {
	ctx := context.Background()

	t.Run("successfully retrieve profile", func(t *testing.T) {
		repo := new(MockRunnerProfileRepository)
		svc := service.NewProfileService(repo)

		res, _ := model.NewResourceRequirements("1000m", "2000m", "1Gi", "2Gi")
		profile, err := model.NewRunnerProfile("found-profile", "desc", "", res, nil, model.Affinity{}, nil)
		require.NoError(t, err)

		repo.On("FindByID", ctx, profile.ID()).Return(profile, nil)

		found, err := svc.GetProfile(ctx, profile.ID())
		require.NoError(t, err)
		assert.Equal(t, profile.ID(), found.ID())
		assert.Equal(t, "found-profile", found.Name())
		repo.AssertExpectations(t)
	})

	t.Run("fail with empty ID", func(t *testing.T) {
		repo := new(MockRunnerProfileRepository)
		svc := service.NewProfileService(repo)

		_, err := svc.GetProfile(ctx, "   ")
		assert.ErrorIs(t, err, model.ErrValidation)
	})

	t.Run("profile not found", func(t *testing.T) {
		repo := new(MockRunnerProfileRepository)
		svc := service.NewProfileService(repo)

		repo.On("FindByID", ctx, "nonexistent").Return(nil, model.ErrNotFound)

		_, err := svc.GetProfile(ctx, "nonexistent")
		assert.ErrorIs(t, err, model.ErrNotFound)
		repo.AssertExpectations(t)
	})
}

func TestProfileService_ListProfiles(t *testing.T) {
	ctx := context.Background()

	t.Run("successfully list profiles", func(t *testing.T) {
		repo := new(MockRunnerProfileRepository)
		svc := service.NewProfileService(repo)

		res, _ := model.NewResourceRequirements("1000m", "2000m", "1Gi", "2Gi")
		p1, _ := model.NewRunnerProfile("prof1", "desc1", "", res, nil, model.Affinity{}, nil)
		p2, _ := model.NewRunnerProfile("prof2", "desc2", "", res, nil, model.Affinity{}, nil)

		repo.On("List", ctx).Return([]*model.RunnerProfile{p1, p2}, nil)

		list, err := svc.ListProfiles(ctx)
		require.NoError(t, err)
		assert.Len(t, list, 2)
		repo.AssertExpectations(t)
	})

	t.Run("propagate list error", func(t *testing.T) {
		repo := new(MockRunnerProfileRepository)
		svc := service.NewProfileService(repo)

		repo.On("List", ctx).Return(nil, errors.New("db query error"))

		_, err := svc.ListProfiles(ctx)
		assert.Error(t, err)
		repo.AssertExpectations(t)
	})
}

func TestProfileService_UpdateProfile(t *testing.T) {
	ctx := context.Background()

	t.Run("successfully update profile", func(t *testing.T) {
		repo := new(MockRunnerProfileRepository)
		svc := service.NewProfileService(repo)

		res, _ := model.NewResourceRequirements("1000m", "2000m", "1Gi", "2Gi")
		existing, _ := model.NewRunnerProfile("orig-name", "orig-desc", "", res, nil, model.Affinity{}, nil)

		cmd := inbound.UpdateProfileCommand{
			Name:          "new-name",
			Description:   "new-desc",
			RunnerImage:   "new-image:v2",
			CPURequest:    "2000m",
			CPULimit:      "4000m",
			MemoryRequest: "2Gi",
			MemoryLimit:   "4Gi",
		}

		repo.On("FindByID", ctx, existing.ID()).Return(existing, nil)
		repo.On("Save", ctx, mock.MatchedBy(func(p *model.RunnerProfile) bool {
			return p.ID() == existing.ID() &&
				p.Name() == "new-name" &&
				p.RunnerImage() == "new-image:v2" &&
				p.Resources().CPURequest() == "2000m" &&
				p.Resources().CPULimit() == "4000m"
		})).Return(nil)

		updated, err := svc.UpdateProfile(ctx, existing.ID(), cmd)
		require.NoError(t, err)
		assert.Equal(t, "new-name", updated.Name())
		assert.Equal(t, "new-image:v2", updated.RunnerImage())
		repo.AssertExpectations(t)
	})

	t.Run("fail with empty ID", func(t *testing.T) {
		repo := new(MockRunnerProfileRepository)
		svc := service.NewProfileService(repo)

		_, err := svc.UpdateProfile(ctx, " ", inbound.UpdateProfileCommand{Name: "valid"})
		assert.ErrorIs(t, err, model.ErrValidation)
	})

	t.Run("fail when profile not found", func(t *testing.T) {
		repo := new(MockRunnerProfileRepository)
		svc := service.NewProfileService(repo)

		repo.On("FindByID", ctx, "missing").Return(nil, model.ErrNotFound)

		_, err := svc.UpdateProfile(ctx, "missing", inbound.UpdateProfileCommand{Name: "valid"})
		assert.ErrorIs(t, err, model.ErrNotFound)
		repo.AssertExpectations(t)
	})

	t.Run("fail when update validation fails", func(t *testing.T) {
		repo := new(MockRunnerProfileRepository)
		svc := service.NewProfileService(repo)

		res, _ := model.NewResourceRequirements("1000m", "2000m", "1Gi", "2Gi")
		existing, _ := model.NewRunnerProfile("orig-name", "orig-desc", "", res, nil, model.Affinity{}, nil)

		repo.On("FindByID", ctx, existing.ID()).Return(existing, nil)

		_, err := svc.UpdateProfile(ctx, existing.ID(), inbound.UpdateProfileCommand{
			Name:       "new-name",
			CPURequest: "5000m",
			CPULimit:   "2000m", // Request > Limit
		})
		assert.ErrorIs(t, err, model.ErrInvalidResourceQuantity)
	})
}

func TestProfileService_DeleteProfile(t *testing.T) {
	ctx := context.Background()

	t.Run("successfully delete profile", func(t *testing.T) {
		repo := new(MockRunnerProfileRepository)
		svc := service.NewProfileService(repo)

		repo.On("Delete", ctx, "prof-123").Return(nil)

		err := svc.DeleteProfile(ctx, "prof-123")
		require.NoError(t, err)
		repo.AssertExpectations(t)
	})

	t.Run("fail with empty ID", func(t *testing.T) {
		repo := new(MockRunnerProfileRepository)
		svc := service.NewProfileService(repo)

		err := svc.DeleteProfile(ctx, "")
		assert.ErrorIs(t, err, model.ErrValidation)
	})

	t.Run("not found on delete", func(t *testing.T) {
		repo := new(MockRunnerProfileRepository)
		svc := service.NewProfileService(repo)

		repo.On("Delete", ctx, "missing").Return(model.ErrNotFound)

		err := svc.DeleteProfile(ctx, "missing")
		assert.ErrorIs(t, err, model.ErrNotFound)
		repo.AssertExpectations(t)
	})
}
