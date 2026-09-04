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

func TestToleration_Validate(t *testing.T) {
	secs := int64(300)
	negSecs := int64(-10)

	tests := []struct {
		name        string
		toleration  model.Toleration
		expectError bool
	}{
		{
			name: "valid Exists operator with no value",
			toleration: model.Toleration{
				Key:      "dedicated",
				Operator: "Exists",
				Effect:   "NoSchedule",
			},
			expectError: false,
		},
		{
			name: "valid Equal operator with key and value",
			toleration: model.Toleration{
				Key:      "dedicated",
				Operator: "Equal",
				Value:    "runner",
				Effect:   "NoSchedule",
			},
			expectError: false,
		},
		{
			name: "valid NoExecute with TolerationSeconds",
			toleration: model.Toleration{
				Key:               "node.kubernetes.io/unreachable",
				Operator:          "Exists",
				Effect:            "NoExecute",
				TolerationSeconds: &secs,
			},
			expectError: false,
		},
		{
			name: "valid empty operator defaults to Equal",
			toleration: model.Toleration{
				Key:    "dedicated",
				Value:  "runner",
				Effect: "PreferNoSchedule",
			},
			expectError: false,
		},
		{
			name: "invalid operator",
			toleration: model.Toleration{
				Key:      "dedicated",
				Operator: "InvalidOperator",
			},
			expectError: true,
		},
		{
			name: "invalid effect",
			toleration: model.Toleration{
				Key:      "dedicated",
				Operator: "Exists",
				Effect:   "InvalidEffect",
			},
			expectError: true,
		},
		{
			name: "invalid Exists operator with non-empty value",
			toleration: model.Toleration{
				Key:      "dedicated",
				Operator: "Exists",
				Value:    "some-value",
			},
			expectError: true,
		},
		{
			name: "invalid Equal operator with empty key",
			toleration: model.Toleration{
				Operator: "Equal",
				Value:    "runner",
			},
			expectError: true,
		},
		{
			name: "invalid TolerationSeconds when effect is not NoExecute",
			toleration: model.Toleration{
				Key:               "dedicated",
				Operator:          "Exists",
				Effect:            "NoSchedule",
				TolerationSeconds: &secs,
			},
			expectError: true,
		},
		{
			name: "invalid negative TolerationSeconds",
			toleration: model.Toleration{
				Key:               "node.kubernetes.io/unreachable",
				Operator:          "Exists",
				Effect:            "NoExecute",
				TolerationSeconds: &negSecs,
			},
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.toleration.Validate()
			if tc.expectError {
				assert.Error(t, err)
				assert.ErrorIs(t, err, model.ErrInvalidToleration)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestAffinity_Validate(t *testing.T) {
	tests := []struct {
		name        string
		affinity    model.Affinity
		expectError bool
	}{
		{
			name: "valid empty affinity",
			affinity: model.Affinity{
				NodeSelectorTerms: nil,
			},
			expectError: false,
		},
		{
			name: "valid In operator with values",
			affinity: model.Affinity{
				NodeSelectorTerms: []model.NodeAffinityTerm{
					{
						Key:      "topology.kubernetes.io/zone",
						Operator: "In",
						Values:   []string{"zone-a", "zone-b"},
					},
				},
			},
			expectError: false,
		},
		{
			name: "valid Exists operator with empty values",
			affinity: model.Affinity{
				NodeSelectorTerms: []model.NodeAffinityTerm{
					{
						Key:      "node-role.kubernetes.io/runner",
						Operator: "Exists",
						Values:   nil,
					},
				},
			},
			expectError: false,
		},
		{
			name: "valid Gt operator with single integer",
			affinity: model.Affinity{
				NodeSelectorTerms: []model.NodeAffinityTerm{
					{
						Key:      "cpu-cores",
						Operator: "Gt",
						Values:   []string{"4"},
					},
				},
			},
			expectError: false,
		},
		{
			name: "invalid empty key",
			affinity: model.Affinity{
				NodeSelectorTerms: []model.NodeAffinityTerm{
					{
						Key:      "",
						Operator: "Exists",
					},
				},
			},
			expectError: true,
		},
		{
			name: "invalid operator",
			affinity: model.Affinity{
				NodeSelectorTerms: []model.NodeAffinityTerm{
					{
						Key:      "zone",
						Operator: "InvalidOp",
						Values:   []string{"zone-a"},
					},
				},
			},
			expectError: true,
		},
		{
			name: "invalid In operator with empty values",
			affinity: model.Affinity{
				NodeSelectorTerms: []model.NodeAffinityTerm{
					{
						Key:      "zone",
						Operator: "In",
						Values:   []string{},
					},
				},
			},
			expectError: true,
		},
		{
			name: "invalid Exists operator with non-empty values",
			affinity: model.Affinity{
				NodeSelectorTerms: []model.NodeAffinityTerm{
					{
						Key:      "zone",
						Operator: "Exists",
						Values:   []string{"zone-a"},
					},
				},
			},
			expectError: true,
		},
		{
			name: "invalid Gt operator with non-integer value",
			affinity: model.Affinity{
				NodeSelectorTerms: []model.NodeAffinityTerm{
					{
						Key:      "cpu-cores",
						Operator: "Gt",
						Values:   []string{"four"},
					},
				},
			},
			expectError: true,
		},
		{
			name: "invalid Gt operator with multiple values",
			affinity: model.Affinity{
				NodeSelectorTerms: []model.NodeAffinityTerm{
					{
						Key:      "cpu-cores",
						Operator: "Gt",
						Values:   []string{"4", "8"},
					},
				},
			},
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.affinity.Validate()
			if tc.expectError {
				assert.Error(t, err)
				assert.ErrorIs(t, err, model.ErrInvalidAffinity)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestRunnerProfile_UpdateDetails(t *testing.T) {
	res1, _ := model.NewResourceRequirements("1000m", "2000m", "1Gi", "2Gi")
	profile, err := model.NewRunnerProfile("initial-name", "desc", "alpine:3.20", res1, nil, model.Affinity{}, nil)
	require.NoError(t, err)
	initialUpdatedAt := profile.UpdatedAt()

	time.Sleep(10 * time.Millisecond)

	res2, err := model.NewResourceRequirements("2000m", "4000m", "2Gi", "4Gi")
	require.NoError(t, err)

	affinity := model.Affinity{
		NodeSelectorTerms: []model.NodeAffinityTerm{
			{Key: "zone", Operator: "In", Values: []string{"zone-a"}},
		},
	}
	tolerations := []model.Toleration{
		{Key: "dedicated", Operator: "Exists", Effect: "NoSchedule"},
	}
	nodeSelector := map[string]string{"env": "perf"}

	t.Run("successfully update profile details", func(t *testing.T) {
		err := profile.UpdateDetails("updated-name", "updated-desc", "custom-runner:v2", res2, nodeSelector, affinity, tolerations)
		require.NoError(t, err)

		assert.Equal(t, "updated-name", profile.Name())
		assert.Equal(t, "updated-desc", profile.Description())
		assert.Equal(t, "custom-runner:v2", profile.RunnerImage())
		assert.Equal(t, res2, profile.Resources())
		assert.Equal(t, nodeSelector, profile.NodeSelector())
		assert.Equal(t, affinity, profile.Affinity())
		assert.Equal(t, tolerations, profile.Tolerations())
		assert.True(t, profile.UpdatedAt().After(initialUpdatedAt))
	})

	t.Run("fail update with empty name", func(t *testing.T) {
		err := profile.UpdateDetails("   ", "desc", "", res2, nil, model.Affinity{}, nil)
		assert.ErrorIs(t, err, model.ErrEmptyName)
	})

	t.Run("fail update with invalid affinity", func(t *testing.T) {
		invalidAffinity := model.Affinity{
			NodeSelectorTerms: []model.NodeAffinityTerm{
				{Key: "", Operator: "In"},
			},
		}
		err := profile.UpdateDetails("name", "desc", "", res2, nil, invalidAffinity, nil)
		assert.ErrorIs(t, err, model.ErrInvalidAffinity)
	})

	t.Run("fail update with invalid toleration", func(t *testing.T) {
		invalidToleration := []model.Toleration{
			{Key: "dedicated", Operator: "InvalidOp"},
		}
		err := profile.UpdateDetails("name", "desc", "", res2, nil, model.Affinity{}, invalidToleration)
		assert.ErrorIs(t, err, model.ErrInvalidToleration)
	})
}
