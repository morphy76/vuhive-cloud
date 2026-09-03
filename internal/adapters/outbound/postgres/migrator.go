package postgres

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/rs/zerolog"
)

//go:embed migrations/*.sql
var MigrationFS embed.FS

const migrationsDir = "migrations"

// MigrateUp executes all pending database migrations.
func MigrateUp(ctx context.Context, db *sql.DB) error {
	start := time.Now()
	log := zerolog.Ctx(ctx).With().Str("op", "MigrateUp").Logger()
	log.Debug().Msg("starting database migration up")

	goose.SetBaseFS(MigrationFS)
	if err := goose.SetDialect("postgres"); err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed to set dialect for goose")
		return fmt.Errorf("failed to set goose dialect: %w", err)
	}

	if err := goose.UpContext(ctx, db, migrationsDir); err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("database migration up failed")
		return fmt.Errorf("goose up failed: %w", err)
	}

	log.Info().Dur("duration_ms", time.Since(start)).Msg("completed database migration up")
	return nil
}

// MigrateDown rolls back the most recent database migration.
func MigrateDown(ctx context.Context, db *sql.DB) error {
	start := time.Now()
	log := zerolog.Ctx(ctx).With().Str("op", "MigrateDown").Logger()
	log.Debug().Msg("starting database migration down")

	goose.SetBaseFS(MigrationFS)
	if err := goose.SetDialect("postgres"); err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed to set dialect for goose")
		return fmt.Errorf("failed to set goose dialect: %w", err)
	}

	if err := goose.DownContext(ctx, db, migrationsDir); err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("database migration down failed")
		return fmt.Errorf("goose down failed: %w", err)
	}

	log.Info().Dur("duration_ms", time.Since(start)).Msg("completed database migration down")
	return nil
}

// MigrateReset rolls back all database migrations.
func MigrateReset(ctx context.Context, db *sql.DB) error {
	start := time.Now()
	log := zerolog.Ctx(ctx).With().Str("op", "MigrateReset").Logger()
	log.Debug().Msg("starting database migration reset")

	goose.SetBaseFS(MigrationFS)
	if err := goose.SetDialect("postgres"); err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("failed to set dialect for goose")
		return fmt.Errorf("failed to set goose dialect: %w", err)
	}

	if err := goose.ResetContext(ctx, db, migrationsDir); err != nil {
		log.Error().Err(err).Dur("duration_ms", time.Since(start)).Msg("database migration reset failed")
		return fmt.Errorf("goose reset failed: %w", err)
	}

	log.Info().Dur("duration_ms", time.Since(start)).Msg("completed database migration reset")
	return nil
}
