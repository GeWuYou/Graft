package store

import (
	"context"
	"errors"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestRegistryWriteErrorMapsUniqueViolationToConflict(t *testing.T) {
	if !errors.Is(registryWriteError(&pgconn.PgError{Code: "23505"}), ErrConflict) {
		t.Fatal("unique violation was not mapped to ErrConflict")
	}
}

func TestCreateRepositoryRejectsMissingCreatorBeforeRepositoryWrite(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sql mock: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT EXISTS (SELECT 1 FROM users WHERE id = $1)`)).
		WithArgs(uint64(9)).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))
	mock.ExpectRollback()

	repository, err := NewSQLRepository(db)
	if err != nil {
		t.Fatalf("new repository: %v", err)
	}
	_, err = repository.CreateRepository(context.Background(), "registry:primary", RepositoryInput{RepositoryRef: "team/app", DisplayName: "Application", AllowPull: true, AllowPush: true, GrantCreatorUse: true}, 9)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("create repository error = %v, want ErrNotFound", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unexpected database operation: %v", err)
	}
}
