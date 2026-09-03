package postgres

import (
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/morphy76/vuhive-cloud/internal/domain/model"
)

// PostgreSQL error codes
const (
	pgErrUniqueViolation           = "23505"
	pgErrForeignKeyViolation       = "23503"
	pgErrInvalidTextRepresentation = "22P02"
)

// MapError translates PostgreSQL and driver-specific errors into domain errors.
func MapError(err error) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, pgx.ErrNoRows) {
		return model.ErrNotFound
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case pgErrUniqueViolation:
			return fmt.Errorf("%w: %s", model.ErrConflict, pgErr.Detail)
		case pgErrForeignKeyViolation:
			return fmt.Errorf("%w: %s", model.ErrValidation, pgErr.Detail)
		case pgErrInvalidTextRepresentation:
			return fmt.Errorf("%w: %s", model.ErrNotFound, pgErr.Message)
		}
	}

	return err
}
