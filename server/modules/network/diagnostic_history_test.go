package network

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"

	"graft/server/internal/moduleapi"
)

func TestSQLDiagnosticHistoryStoreAppendsAndListsBoundedResults(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("open sql mock: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	store, err := NewSQLDiagnosticHistoryStore(db)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	testedAt := time.Date(2026, 8, 3, 12, 0, 0, 0, time.UTC)
	mock.ExpectExec("INSERT INTO platform_network_diagnostic_history").
		WithArgs("platform-update-release", true, int64(184), 200, nil, testedAt).
		WillReturnResult(sqlmock.NewResult(1, 1))
	if err := store.Append(context.Background(), "platform-update-release", moduleapi.OutboundDiagnosticResult{Connected: true, Latency: 184 * time.Millisecond, HTTPStatus: 200, TestedAt: testedAt}); err != nil {
		t.Fatalf("append history: %v", err)
	}
	mock.ExpectQuery("SELECT connected, latency_ms, http_status, error_message, tested_at").
		WithArgs("platform-update-release", 20).
		WillReturnRows(sqlmock.NewRows([]string{"connected", "latency_ms", "http_status", "error_message", "tested_at"}).AddRow(false, int64(42), nil, "outbound request failed", testedAt))
	items, err := store.List(context.Background(), "platform-update-release", 20)
	if err != nil {
		t.Fatalf("list history: %v", err)
	}
	if len(items) != 1 || items[0].Connected || items[0].Latency != 42*time.Millisecond || items[0].Message != "outbound request failed" || !items[0].TestedAt.Equal(testedAt) {
		t.Fatalf("unexpected history items: %#v", items)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet SQL expectations: %v", err)
	}
}

func TestSQLDiagnosticHistoryStoreRejectsUnboundedList(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("open sql mock: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	store, err := NewSQLDiagnosticHistoryStore(db)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	if _, err := store.List(context.Background(), "platform-update-release", maxDiagnosticHistoryLimit+1); err == nil {
		t.Fatal("expected oversized history limit to fail")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unexpected SQL for invalid list: %v", err)
	}
}
