package postgres_test

import (
	"testing"

	"github.com/morphy76/vuhive-cloud/internal/adapters/outbound/postgres"
	"github.com/morphy76/vuhive-cloud/internal/application/ports/outbound"
	"github.com/stretchr/testify/assert"
)

func TestRepositories_InterfaceSatisfaction(t *testing.T) {
	var suiteRepo outbound.TestSuiteRepository = postgres.NewTestSuiteRepository(nil)
	assert.NotNil(t, suiteRepo)

	var artifactRepo outbound.ArtifactRepository = postgres.NewArtifactRepository(nil)
	assert.NotNil(t, artifactRepo)

	var configRepo outbound.ConfigurationRepository = postgres.NewConfigurationRepository(nil)
	assert.NotNil(t, configRepo)

	var profileRepo outbound.RunnerProfileRepository = postgres.NewRunnerProfileRepository(nil)
	assert.NotNil(t, profileRepo)

	var scheduleRepo outbound.ScheduleRepository = postgres.NewScheduleRepository(nil)
	assert.NotNil(t, scheduleRepo)

	var runRepo outbound.TestRunRepository = postgres.NewTestRunRepository(nil)
	assert.NotNil(t, runRepo)
}
