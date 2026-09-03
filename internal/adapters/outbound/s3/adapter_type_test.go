package s3_test

import (
	"testing"

	"github.com/morphy76/vuhive-cloud/internal/adapters/outbound/s3"
	"github.com/morphy76/vuhive-cloud/internal/application/ports/outbound"
	"github.com/stretchr/testify/assert"
)

func TestAdapter_InterfaceSatisfaction(t *testing.T) {
	var port outbound.StoragePort = (*s3.Adapter)(nil)
	assert.Nil(t, port)
}
