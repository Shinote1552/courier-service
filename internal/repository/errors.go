package repository

import (
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
)

// https://www.postgresql.org/docs/current/errcodes-appendix.html#23505:~:text=foreign_key_violation-,23505,-unique_violation
const PgErrUniqueViolation = "23505"

func IsPgErrorWithCode(err error, code string) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == code
	}
	return false
}
