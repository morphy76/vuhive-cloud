package model_test

import (
	"errors"
	"testing"

	"github.com/morphy76/vuhive-cloud/internal/bff/domain/model"
	"github.com/stretchr/testify/assert"
)

func TestDomainErrors(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{
			name:     "ErrSessionNotFound",
			err:      model.ErrSessionNotFound,
			expected: "bff session not found",
		},
		{
			name:     "ErrControlPlaneUnavailable",
			err:      model.ErrControlPlaneUnavailable,
			expected: "control plane service is unavailable",
		},
		{
			name:     "ErrInvalidParameter",
			err:      model.ErrInvalidParameter,
			expected: "invalid parameter provided",
		},
		{
			name:     "ErrUnauthorized",
			err:      model.ErrUnauthorized,
			expected: "unauthorized request",
		},
		{
			name:     "ErrRunNotFound",
			err:      model.ErrRunNotFound,
			expected: "test run not found",
		},
		{
			name:     "ErrSuiteNotFound",
			err:      model.ErrSuiteNotFound,
			expected: "test suite not found",
		},
		{
			name:     "ErrProfileNotFound",
			err:      model.ErrProfileNotFound,
			expected: "runner profile not found",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Error(t, tc.err)
			assert.Equal(t, tc.expected, tc.err.Error())
		})
	}
}

func TestDomainErrorWrapping(t *testing.T) {
	customErr := errors.New("upstream timeout")
	wrapped := model.NewDomainError(model.ErrControlPlaneUnavailable, customErr)

	assert.True(t, errors.Is(wrapped, model.ErrControlPlaneUnavailable))
	assert.Contains(t, wrapped.Error(), "upstream timeout")
}
