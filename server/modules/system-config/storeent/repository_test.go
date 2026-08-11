package storeent

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	_ "github.com/mattn/go-sqlite3"

	systemconfigstore "graft/server/modules/system-config/store"
)

func TestRepositoryCompareAndSwapOverrideWrapsUserIDConversionError(t *testing.T) {
	t.Parallel()

	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			t.Fatalf("close sqlite db: %v", err)
		}
	}()

	repo := &repository{db: db}
	overflow := uint64(math.MaxInt64) + 1
	_, err = repo.CompareAndSwapOverride(context.Background(), "scheduler.timeout", json.RawMessage(`"60s"`), &overflow, 0)
	if err == nil {
		t.Fatalf("expected user id range error")
	}
	if !strings.Contains(err.Error(), "compare and swap system config override:") {
		t.Fatalf("expected compare and swap operation context, got %v", err)
	}
	if !strings.Contains(err.Error(), "system config override user id exceeds database range") {
		t.Fatalf("expected user id conversion error, got %v", err)
	}
	if _, ok := err.(interface{ Unwrap() error }); !ok {
		t.Fatalf("expected wrapped error, got %T", err)
	}
}

func TestRepositoryCompareAndSwapAndResetUseVersionConditions(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("new sql mock: %v", err)
	}
	defer func() {
		_ = db.Close()
	}()
	repo := &repository{db: db}
	columns := []string{"key", "override_value", "version", "created_at", "created_by", "updated_at", "updated_by"}
	createdAt := time.Date(2026, time.August, 4, 0, 0, 0, 0, time.UTC)

	mock.ExpectQuery("INSERT INTO system_config_values").
		WithArgs("network.outbound", json.RawMessage(`{"enabled":true}`), sql.NullInt64{}).
		WillReturnRows(sqlmock.NewRows(columns).AddRow("network.outbound", []byte(`{"enabled":true}`), int64(1), createdAt, nil, createdAt, nil))
	updated, err := repo.CompareAndSwapOverride(context.Background(), "network.outbound", json.RawMessage(`{"enabled":true}`), nil, 0)
	if err != nil || updated.Version != 1 || string(updated.Value) != `{"enabled":true}` {
		t.Fatalf("expected first CAS write, got %#v, %v", updated, err)
	}

	mock.ExpectQuery("INSERT INTO system_config_values").
		WithArgs("network.outbound", json.RawMessage(`{"enabled":false}`), sql.NullInt64{}).
		WillReturnError(sql.ErrNoRows)
	if _, err := repo.CompareAndSwapOverride(context.Background(), "network.outbound", json.RawMessage(`{"enabled":false}`), nil, 0); !errors.Is(err, systemconfigstore.ErrVersionConflict) {
		t.Fatalf("expected stale CAS conflict, got %v", err)
	}

	mock.ExpectQuery("UPDATE system_config_values").
		WithArgs("network.outbound", json.RawMessage(`{"enabled":false}`), sql.NullInt64{}, int64(1)).
		WillReturnRows(sqlmock.NewRows(columns).AddRow("network.outbound", []byte(`{"enabled":false}`), int64(2), createdAt, nil, createdAt, nil))
	updated, err = repo.CompareAndSwapOverride(context.Background(), "network.outbound", json.RawMessage(`{"enabled":false}`), nil, 1)
	if err != nil || updated.Version != 2 || string(updated.Value) != `{"enabled":false}` {
		t.Fatalf("expected version-matched CAS update, got %#v, %v", updated, err)
	}

	mock.ExpectQuery("UPDATE system_config_values").
		WithArgs("network.outbound", json.RawMessage(`{"enabled":false}`), sql.NullInt64{}, int64(1)).
		WillReturnError(sql.ErrNoRows)
	if _, err := repo.CompareAndSwapOverride(context.Background(), "network.outbound", json.RawMessage(`{"enabled":false}`), nil, 1); !errors.Is(err, systemconfigstore.ErrVersionConflict) {
		t.Fatalf("expected stale CAS update conflict, got %v", err)
	}

	mock.ExpectQuery("UPDATE system_config_values").
		WithArgs("network.outbound", sql.NullInt64{}, int64(2)).
		WillReturnRows(sqlmock.NewRows(columns).AddRow("network.outbound", nil, int64(3), createdAt, nil, createdAt, nil))
	reset, err := repo.ResetOverride(context.Background(), "network.outbound", nil, 2)
	if err != nil || reset.Version != 3 || reset.Value != nil {
		t.Fatalf("expected version-retaining reset tombstone, got %#v, %v", reset, err)
	}

	mock.ExpectQuery("UPDATE system_config_values").
		WithArgs("network.outbound", sql.NullInt64{}, int64(2)).
		WillReturnError(sql.ErrNoRows)
	if _, err := repo.ResetOverride(context.Background(), "network.outbound", nil, 2); !errors.Is(err, systemconfigstore.ErrVersionConflict) {
		t.Fatalf("expected stale reset conflict, got %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("verify sql expectations: %v", err)
	}
}

func TestRepositoryRepairsLegacyVersionZeroRows(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("new sql mock: %v", err)
	}
	defer func() { _ = db.Close() }()

	repo := &repository{db: db}
	columns := []string{"key", "override_value", "version", "created_at", "created_by", "updated_at", "updated_by"}
	updatedAt := time.Date(2026, time.August, 11, 0, 0, 0, 0, time.UTC)
	mock.ExpectQuery("INSERT INTO system_config_values").
		WithArgs("notification.enabled", json.RawMessage(`false`), sql.NullInt64{}).
		WillReturnRows(sqlmock.NewRows(columns).AddRow("notification.enabled", []byte(`false`), int64(1), updatedAt, nil, updatedAt, nil))
	updated, err := repo.CompareAndSwapOverride(context.Background(), "notification.enabled", json.RawMessage(`false`), nil, 0)
	if err != nil || updated.Version != 1 || string(updated.Value) != `false` {
		t.Fatalf("expected legacy version zero row repair, got %#v, %v", updated, err)
	}

	mock.ExpectQuery("INSERT INTO system_config_values").
		WithArgs("notification.enabled", sql.NullInt64{}).
		WillReturnRows(sqlmock.NewRows(columns).AddRow("notification.enabled", nil, int64(1), updatedAt, nil, updatedAt, nil))
	reset, err := repo.ResetOverride(context.Background(), "notification.enabled", nil, 0)
	if err != nil || reset.Version != 1 || reset.Value != nil {
		t.Fatalf("expected legacy version zero reset repair, got %#v, %v", reset, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("verify sql expectations: %v", err)
	}
}

var _ systemconfigstore.Repository = (*repository)(nil)
