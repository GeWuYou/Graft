package store

import (
	"context"
	"database/sql/driver"
	"errors"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

func TestRegistryWriteErrorMapsUniqueViolationToConflict(t *testing.T) {
	if !errors.Is(registryWriteError(&pgconn.PgError{Code: "23505"}), ErrConflict) {
		t.Fatal("unique violation was not mapped to ErrConflict")
	}
}

func TestCreateRepositoryRejectsMissingCreatorBeforeRepositoryWrite(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.ValueConverterOption(registrySQLValueConverter{}))
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

func TestReplaceAssignmentsRejectsInvalidUserInLargeSetBeforeClearingAssignments(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sql mock: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT r.id FROM artifact_repositories r JOIN registry_connections c ON c.id = r.connection_id AND c.deleted_at = 0 WHERE c.connection_ref = $1 AND r.repository_ref = $2 AND r.deleted_at = 0`)).
		WithArgs("registry:primary", "team/app").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(uint64(5)))
	userIDs := make([]uint64, 101)
	for index := range userIDs {
		userIDs[index] = uint64(index + 1)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT EXISTS (SELECT 1 FROM users WHERE id = $1)`)).
			WithArgs(userIDs[index]).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(index < 100))
	}
	mock.ExpectRollback()

	repository, err := NewSQLRepository(db)
	if err != nil {
		t.Fatalf("new repository: %v", err)
	}
	_, err = repository.ReplaceAssignments(context.Background(), "registry:primary", "team/app", userIDs, 7)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("replace assignments error = %v, want ErrNotFound", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unexpected database operation: %v", err)
	}
}

func TestAddAssignmentsAddsOnlyMissingPairsAfterValidatingAllTargets(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.ValueConverterOption(registrySQLValueConverter{}))
	if err != nil {
		t.Fatalf("create sql mock: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT COUNT(*) FROM artifact_repositories r
JOIN registry_connections c ON c.id = r.connection_id AND c.deleted_at = 0
WHERE c.connection_ref = $1 AND r.repository_ref = ANY($2) AND r.deleted_at = 0`)).
		WithArgs("registry:primary", "registry-string-array").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT COUNT(*) FROM users WHERE id = ANY($1)`)).
		WithArgs("registry-uint64-array").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO artifact_repository_user_assignments (repository_id, user_id, created_by, updated_by)`)).
		WithArgs("registry:primary", "registry-string-array", "registry-uint64-array", uint64(7)).
		WillReturnResult(sqlmock.NewResult(0, 3))
	mock.ExpectCommit()

	repository, err := NewSQLRepository(db)
	if err != nil {
		t.Fatalf("new repository: %v", err)
	}
	result, err := repository.AddAssignments(context.Background(), "registry:primary", AssignmentBatchAddInput{RepositoryRefs: []string{"app", "team/app"}, UserIDs: []uint64{7, 9}}, 7)
	if err != nil {
		t.Fatalf("add assignments: %v", err)
	}
	if result != (AssignmentBatchAddResult{Total: 4, AddedCount: 3, AlreadyAssignedCount: 1}) {
		t.Fatalf("batch result = %#v", result)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unexpected database operation: %v", err)
	}
}

type registrySQLValueConverter struct{}

func (registrySQLValueConverter) ConvertValue(value any) (driver.Value, error) {
	switch value.(type) {
	case pgtype.FlatArray[string]:
		return "registry-string-array", nil
	case pgtype.FlatArray[uint64]:
		return "registry-uint64-array", nil
	default:
		return driver.DefaultParameterConverter.ConvertValue(value)
	}
}
