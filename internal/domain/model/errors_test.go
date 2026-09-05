package model_test

import (
	"testing"

	"github.com/morphy76/vuhive-cloud/internal/domain/model"
	"github.com/stretchr/testify/assert"
)

func TestDomainErrors_SentinelValues(t *testing.T) {
	assert.NotNil(t, model.ErrNotFound)
	assert.NotNil(t, model.ErrConflict)
	assert.Equal(t, "resource already exists", model.ErrConflict.Error())
	assert.NotNil(t, model.ErrTimeout)
	assert.Equal(t, "operation timed out", model.ErrTimeout.Error())
	assert.NotNil(t, model.ErrBuildFailed)
	assert.Equal(t, "build compilation failed", model.ErrBuildFailed.Error())
	assert.NotNil(t, model.ErrInvalidWorkerCount)
	assert.Equal(t, "worker count must be at least 1", model.ErrInvalidWorkerCount.Error())
	assert.NotNil(t, model.ErrInvalidWorkerIndex)
	assert.Equal(t, "worker index out of bounds", model.ErrInvalidWorkerIndex.Error())
	assert.NotNil(t, model.ErrNoScenariosDefined)
	assert.Equal(t, "no scenarios defined in configuration", model.ErrNoScenariosDefined.Error())
}

