package postgres_test

import (
	"context"
	"fmt"
	"strings"
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

func TestEmbeddedMigrations_UniqueSequentialVersions(t *testing.T) {
	entries, err := postgres.MigrationFS.ReadDir("migrations")
	require.NoError(t, err)
	require.NotEmpty(t, entries)

	seenVersions := make(map[int]string)
	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasSuffix(name, ".sql") {
			continue
		}
		parts := strings.SplitN(name, "_", 2)
		require.Len(t, parts, 2, "migration filename must contain an underscore separator: %s", name)

		var version int
		_, err := fmt.Sscanf(parts[0], "%d", &version)
		require.NoError(t, err, "migration prefix must be numeric: %s", name)

		prevFile, exists := seenVersions[version]
		assert.Falsef(t, exists, "duplicate migration version %06d found in files %s and %s", version, prevFile, name)
		seenVersions[version] = name
	}

	// Verify continuous sequential ordering without gaps starting from 1
	for v := 1; v <= len(seenVersions); v++ {
		_, ok := seenVersions[v]
		assert.Truef(t, ok, "missing migration version %06d in sequence", v)
	}
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
