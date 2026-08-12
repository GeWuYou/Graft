package store

import (
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
)

func TestRegistryWriteErrorMapsUniqueViolationToConflict(t *testing.T) {
	if !errors.Is(registryWriteError(&pgconn.PgError{Code: "23505"}), ErrConflict) {
		t.Fatal("unique violation was not mapped to ErrConflict")
	}
}
