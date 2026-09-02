package model_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/morphy76/vuhive-cloud/internal/domain/model"
)

func TestResourceRequirements(t *testing.T) {
	t.Run("valid resource requirements", func(t *testing.T) {
		res, err := model.NewResourceRequirements("1000m", "2000m", "1Gi", "2Gi")
		require.NoError(t, err)
		assert.Equal(t, "1000m", res.CPURequest())
		assert.Equal(t, "2000m", res.CPULimit())
		assert.Equal(t, "1Gi", res.MemoryRequest())
		assert.Equal(t, "2Gi", res.MemoryLimit())
	})

	t.Run("valid with whole cores and megabytes", func(t *testing.T) {
		res, err := model.NewResourceRequirements("1", "2", "512Mi", "1024Mi")
		require.NoError(t, err)
		assert.Equal(t, "1", res.CPURequest())
		assert.Equal(t, "2", res.CPULimit())
	})

	t.Run("fail when CPU request exceeds CPU limit", func(t *testing.T) {
		_, err := model.NewResourceRequirements("2000m", "1000m", "1Gi", "2Gi")
		assert.ErrorIs(t, err, model.ErrInvalidResourceQuantity)
	})

	t.Run("fail when Memory request exceeds Memory limit", func(t *testing.T) {
		_, err := model.NewResourceRequirements("1000m", "2000m", "4Gi", "2Gi")
		assert.ErrorIs(t, err, model.ErrInvalidResourceQuantity)
	})

	t.Run("fail with invalid CPU quantity syntax", func(t *testing.T) {
		_, err := model.NewResourceRequirements("invalid", "2000m", "1Gi", "2Gi")
		assert.ErrorIs(t, err, model.ErrInvalidResourceQuantity)
	})

	t.Run("fail with invalid Memory quantity syntax", func(t *testing.T) {
		_, err := model.NewResourceRequirements("1000m", "2000m", "1000Xyz", "2Gi")
		assert.ErrorIs(t, err, model.ErrInvalidResourceQuantity)
	})
}

func TestNewRunnerProfile(t *testing.T) {
	resources, err := model.NewResourceRequirements("1000m", "2000m", "1Gi", "2Gi")
	require.NoError(t, err)

	tolerations := []model.Toleration{
		{
			Key:      "vuhive.io/load-generator",
			Operator: "Exists",
			Effect:   "NoSchedule",
		},
	}

	affinity := model.Affinity{
		NodeSelectorTerms: []model.NodeAffinityTerm{
			{
				Key:      "role",
				Operator: "In",
				Values:   []string{"load-generator"},
			},
		},
	}

	nodeSelector := map[string]string{
		"kubernetes.io/os": "linux",
	}

	t.Run("successfully create runner profile with default image", func(t *testing.T) {
		profile, err := model.NewRunnerProfile(
			"high-performance",
			"Dedicated 2-core load generator",
			"",
			resources,
			nodeSelector,
			affinity,
			tolerations,
		)
		require.NoError(t, err)
		require.NotNil(t, profile)

		assert.NotEmpty(t, profile.ID())
		assert.Equal(t, profile.ID(), profile.EntityID())
		assert.Equal(t, "high-performance", profile.Name())
		assert.Equal(t, "Dedicated 2-core load generator", profile.Description())
		assert.Equal(t, "alpine:3.20", profile.RunnerImage())
		assert.Equal(t, resources, profile.Resources())
		assert.Equal(t, nodeSelector, profile.NodeSelector())
		assert.Equal(t, affinity, profile.Affinity())
		assert.Equal(t, tolerations, profile.Tolerations())
		assert.False(t, profile.CreatedAt().IsZero())
	})

	t.Run("fail with empty name", func(t *testing.T) {
		_, err := model.NewRunnerProfile(
			"",
			"desc",
			"custom-image:latest",
			resources,
			nodeSelector,
			affinity,
			tolerations,
		)
		assert.ErrorIs(t, err, model.ErrEmptyName)
	})

	t.Run("custom runner image is preserved", func(t *testing.T) {
		profile, err := model.NewRunnerProfile(
			"custom-profile",
			"desc",
			"my-registry/runner:v1.0",
			resources,
			nodeSelector,
			affinity,
			tolerations,
		)
		require.NoError(t, err)
		assert.Equal(t, "my-registry/runner:v1.0", profile.RunnerImage())
	})
}

func TestRunnerProfile_UpdateResources(t *testing.T) {
	res1, _ := model.NewResourceRequirements("1000m", "2000m", "1Gi", "2Gi")
	profile, err := model.NewRunnerProfile(
		"profile", "desc", "", res1, nil, model.Affinity{}, nil,
	)
	require.NoError(t, err)

	res2, err := model.NewResourceRequirements("2000m", "4000m", "2Gi", "4Gi")
	require.NoError(t, err)

	err = profile.UpdateResources(res2)
	require.NoError(t, err)
	assert.Equal(t, res2, profile.Resources())
}

func TestNewRunnerProfileWithID(t *testing.T) {
	res, _ := model.NewResourceRequirements("1000m", "2000m", "1Gi", "2Gi")
	now := time.Now()

	profile, err := model.NewRunnerProfileWithID(
		"prof-123", "name", "desc", "alpine:3.20",
		res, nil, model.Affinity{}, nil, now, now,
	)
	require.NoError(t, err)
	assert.Equal(t, "prof-123", profile.ID())
}
