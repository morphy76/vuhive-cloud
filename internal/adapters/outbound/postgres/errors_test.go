package postgres_test

import (
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/morphy76/vuhive-cloud/internal/adapters/outbound/postgres"
	"github.com/morphy76/vuhive-cloud/internal/domain/model"
	"github.com/stretchr/testify/assert"
)

func TestMapError(t *testing.T) {
	t.Run("nil error returns nil", func(t *testing.T) {
		assert.Nil(t, postgres.MapError(nil))
	})

	t.Run("pgx.ErrNoRows translates to model.ErrNotFound", func(t *testing.T) {
		err := postgres.MapError(pgx.ErrNoRows)
		assert.True(t, errors.Is(err, model.ErrNotFound))
	})

	t.Run("pgconn.PgError with 23505 translates to model.ErrConflict", func(t *testing.T) {
		pgErr := &pgconn.PgError{Code: "23505", Message: "duplicate key value violates unique constraint"}
		err := postgres.MapError(pgErr)
		assert.True(t, errors.Is(err, model.ErrConflict))
	})

	t.Run("pgconn.PgError with 23503 translates to model.ErrValidation", func(t *testing.T) {
		pgErr := &pgconn.PgError{Code: "23503", Message: "insert or update on table violates foreign key constraint"}
		err := postgres.MapError(pgErr)
		assert.True(t, errors.Is(err, model.ErrValidation))
	})

	t.Run("pgconn.PgError with 22P02 translates to model.ErrNotFound", func(t *testing.T) {
		pgErr := &pgconn.PgError{Code: "22P02", Message: "invalid input syntax for type uuid"}
		err := postgres.MapError(pgErr)
		assert.True(t, errors.Is(err, model.ErrNotFound))
	})

	t.Run("generic error is preserved", func(t *testing.T) {
		customErr := errors.New("connection failed")
		err := postgres.MapError(customErr)
		assert.True(t, errors.Is(err, customErr))
	})
}
