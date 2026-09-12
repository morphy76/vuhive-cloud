package model_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/morphy76/vuhive-cloud/internal/domain/model"
)

func TestValidateGoVersion(t *testing.T) {
	tests := []struct {
		name        string
		version     string
		expectError bool
		expected    string
	}{
		{
			name:        "empty version is allowed (defaults to cluster default)",
			version:     "",
			expectError: false,
			expected:    "",
		},
		{
			name:        "whitespace only is treated as empty",
			version:     "   ",
			expectError: false,
			expected:    "",
		},
		{
			name:        "Go 1.26 is valid",
			version:     "1.26",
			expectError: false,
			expected:    "1.26",
		},
		{
			name:        "Go 1.26.0 patch version is valid",
			version:     "1.26.0",
			expectError: false,
			expected:    "1.26.0",
		},
		{
			name:        "go1.26 prefix is stripped and valid",
			version:     "go1.26",
			expectError: false,
			expected:    "1.26",
		},
		{
			name:        "Go 1.27 is valid",
			version:     "1.27",
			expectError: false,
			expected:    "1.27",
		},
		{
			name:        "Go 1.28 is valid",
			version:     "1.28",
			expectError: false,
			expected:    "1.28",
		},
		{
			name:        "Go 1.25 is rejected as too old",
			version:     "1.25",
			expectError: true,
		},
		{
			name:        "Go 1.24 is rejected as too old",
			version:     "1.24",
			expectError: true,
		},
		{
			name:        "Go 1.22 is rejected as too old",
			version:     "1.22",
			expectError: true,
		},
		{
			name:        "go1.21 prefix is rejected as too old",
			version:     "go1.21",
			expectError: true,
		},
		{
			name:        "invalid version string is rejected",
			version:     "invalid-version",
			expectError: true,
		},
		{
			name:        "Go 2.0 is valid forward compatibility",
			version:     "2.0",
			expectError: false,
			expected:    "2.0",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			normalized, err := model.ValidateGoVersion(tc.version)
			if tc.expectError {
				require.Error(t, err)
				assert.ErrorIs(t, err, model.ErrUnsupportedGoVersion)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.expected, normalized)
			}
		})
	}
}
