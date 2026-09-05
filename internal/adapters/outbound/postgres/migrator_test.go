package postgres_test

import (
	"context"
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

func TestMigrateUpURL_InvalidURL(t *testing.T) {
	ctx := context.Background()
	err := postgres.MigrateUpURL(ctx, "postgres://invalid-host:99999/invalid-db?sslmode=disable")
	assert.Error(t, err)
}

func TestMigrateUpURL_CanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := postgres.MigrateUpURL(ctx, "postgres://127.0.0.1:5432/vuhive?sslmode=disable")
	assert.Error(t, err)
}
