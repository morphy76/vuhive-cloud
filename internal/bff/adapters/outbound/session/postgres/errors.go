package postgres

import (
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/morphy76/vuhive-cloud/internal/bff/domain/model"
)

const (
	pgErrUniqueViolation = "23505"
)

// MapError translates PostgreSQL driver errors into domain errors.
func MapError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return model.ErrSessionNotFound
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		if pgErr.Code == pgErrUniqueViolation {
			return fmt.Errorf("%w: %s", model.ErrConcurrentSessionModification, pgErr.Detail)
		}
	}
	return err
}
