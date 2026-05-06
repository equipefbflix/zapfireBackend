package repository

import (
	"errors"

	"github.com/jackc/pgx/v5"
)

func isNoRowsError(err error) bool {
	return errors.Is(err, pgx.ErrNoRows)
}
