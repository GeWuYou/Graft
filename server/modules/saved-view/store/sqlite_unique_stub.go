//go:build !cgo

package store

func isSQLiteUniqueViolation(error) bool {
	return false
}
