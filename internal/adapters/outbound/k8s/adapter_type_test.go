package k8s_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/morphy76/vuhive-cloud/internal/adapters/outbound/k8s"
	"github.com/morphy76/vuhive-cloud/internal/application/ports/outbound"
)

func TestBuildOrchestrator_ImplementsPort(t *testing.T) {
	var _ outbound.BuildOrchestratorPort = (*k8s.BuildOrchestrator)(nil)
	assert.True(t, true)
}
