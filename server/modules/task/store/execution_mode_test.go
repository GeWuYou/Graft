package store

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestPostgresExecutionOwnerRejectsAnotherServerBeforeRecovery(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	repository, err := NewSQLRepository(db, SQLDialectPostgres)
	if err != nil {
		t.Fatal(err)
	}
	mock.ExpectQuery("SELECT pg_try_advisory_lock").WithArgs(executionOwnershipKey).WillReturnRows(sqlmock.NewRows([]string{"acquired"}).AddRow(false))
	if _, err := repository.AcquireExecutionOwnership(context.Background()); err == nil {
		t.Fatal("second active execution owner accepted")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestPostgresExecutionOwnerReportsLostDatabaseSession(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	repository, err := NewSQLRepository(db, SQLDialectPostgres)
	if err != nil {
		t.Fatal(err)
	}
	mock.ExpectQuery("SELECT pg_try_advisory_lock").WithArgs(executionOwnershipKey).WillReturnRows(sqlmock.NewRows([]string{"acquired"}).AddRow(true))
	mock.ExpectPing().WillReturnError(errors.New("lost ownership connection"))
	ownership, err := repository.AcquireExecutionOwnership(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	select {
	case <-ownership.Lost:
	case <-time.After(3 * time.Second):
		t.Fatal("connection loss was not reported")
	}
	mock.ExpectExec("SELECT pg_advisory_unlock").WithArgs(executionOwnershipKey).WillReturnResult(sqlmock.NewResult(0, 1))
	if err := ownership.Close(); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
