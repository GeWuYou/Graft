//go:build cgo

package store

import (
	"errors"

	"github.com/mattn/go-sqlite3"
)

func isSQLiteUniqueViolation(err error) bool {
	var sqliteErr sqlite3.Error
	return errors.As(err, &sqliteErr) && sqliteErr.ExtendedCode == sqlite3.ErrConstraintUnique
}
