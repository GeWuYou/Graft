//go:build !cgo

package storeent

func isSQLiteUniqueViolation(error) bool {
	return false
}
