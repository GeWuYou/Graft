package network

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"

	"graft/server/internal/moduleapi"
)

func TestSQLConnectivityStoreAppendsSanitizedReportAndRetainsBoundedHistory(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("open sql mock: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	store, err := NewSQLConnectivityStore(db)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	checkedAt := time.Date(2026, 8, 4, 14, 33, 0, 0, time.UTC)
	report := moduleapi.NewConnectivityReport("github", moduleapi.ConnectivityReportStatusHealthy, checkedAt, 183*time.Millisecond, []moduleapi.ProbeResult{{Kind: moduleapi.ConnectivityProbeHTTP, Status: moduleapi.ProbeStatusSucceeded, Duration: 183 * time.Millisecond, Summary: "HTTP 200"}}, &moduleapi.RouteExplanation{MatchedStrategy: "platform_default", Decision: "direct", Reason: "policy_disabled"}, &moduleapi.ExitIPDisclosure{Masked: "***.***.45.19", Available: true})
	mock.ExpectBegin()
	mock.ExpectQuery("INSERT INTO platform_connectivity_checks").WithArgs("github", "healthy", int64(183), checkedAt).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(42))
	mock.ExpectExec("INSERT INTO platform_connectivity_reports").WithArgs(int64(42), 1, sqlmock.AnyArg()).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("DELETE FROM platform_connectivity_checks").WithArgs("github", maxConnectivityHistoryLimit).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectCommit()
	check, err := store.Append(context.Background(), report)
	if err != nil {
		t.Fatalf("append report: %v", err)
	}
	if check.ID != 42 || check.TargetID != "github" || check.Latency != 183*time.Millisecond {
		t.Fatalf("unexpected check: %#v", check)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet SQL expectations: %v", err)
	}
}

func TestSQLConnectivityStoreRejectsUnmaskedExitIP(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("open sql mock: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	store, err := NewSQLConnectivityStore(db)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	report := moduleapi.ConnectivityReport{SchemaVersion: 1, TargetID: "github", Status: moduleapi.ConnectivityReportStatusHealthy, CheckedAt: time.Now(), ExitIP: &moduleapi.ExitIPDisclosure{Masked: "34.12.45.19", Available: true}}
	if _, err := store.Append(context.Background(), report); err == nil {
		t.Fatal("expected unmasked exit IP to be rejected")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unexpected SQL: %v", err)
	}
}
