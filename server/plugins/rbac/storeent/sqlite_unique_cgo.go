//go:build cgo

package storeent

import (
	"errors"

	sqlite3 "github.com/mattn/go-sqlite3"
)

func isSQLiteUniqueViolation(err error) bool {
	var sqliteErr sqlite3.Error
	return errors.As(err, &sqliteErr) &&
		(sqliteErr.ExtendedCode == sqlite3.ErrConstraintUnique ||
			sqliteErr.ExtendedCode == sqlite3.ErrConstraintPrimaryKey)
}
