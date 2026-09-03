package postgres_test

import (
	"testing"

	"github.com/morphy76/vuhive-cloud/internal/adapters/outbound/postgres"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEmbeddedMigrations(t *testing.T) {
	entries, err := postgres.MigrationFS.ReadDir("migrations")
	require.NoError(t, err)
	assert.NotEmpty(t, entries)

	foundInit := false
	for _, entry := range entries {
		if entry.Name() == "000001_init_schema.sql" {
			foundInit = true
			break
		}
	}
	assert.True(t, foundInit, "expected 000001_init_schema.sql to be embedded")
}
