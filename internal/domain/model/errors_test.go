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
	assert.NotNil(t, model.ErrNoScenariosDefined)
	assert.Equal(t, "no scenarios defined in configuration", model.ErrNoScenariosDefined.Error())
	assert.NotNil(t, model.ErrMissingGoMod)
	assert.Equal(t, "missing go.mod in source archive", model.ErrMissingGoMod.Error())
	assert.NotNil(t, model.ErrMissingVuhiveDependency)
	assert.Equal(t, "go.mod must declare github.com/morphy76/vuhive as a direct dependency", model.ErrMissingVuhiveDependency.Error())
	assert.NotNil(t, model.ErrForbiddenImport)
	assert.Equal(t, "source code imports disallowed package", model.ErrForbiddenImport.Error())
	assert.NotNil(t, model.ErrForbiddenPackageMain)
	assert.Equal(t, "user source code must declare package scenario and must not define package main or func main()", model.ErrForbiddenPackageMain.Error())
	assert.NotNil(t, model.ErrMissingScenarioContract)
	assert.Equal(t, "source package does not declare or implement a valid vuhive.Scenario contract", model.ErrMissingScenarioContract.Error())
	assert.NotNil(t, model.ErrInvalidArchive)
	assert.Equal(t, "source package is not a valid tar.gz archive", model.ErrInvalidArchive.Error())
	assert.NotNil(t, model.ErrInsecureOverrideForbidden)
	assert.Equal(t, "import blocklist override is forbidden by cluster policy", model.ErrInsecureOverrideForbidden.Error())
}

