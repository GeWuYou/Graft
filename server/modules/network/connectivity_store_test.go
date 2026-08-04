package network

import (
	"context"
	"regexp"
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
	httpStatus := 200
	report := moduleapi.NewConnectivityReport("github", moduleapi.ConnectivityReportStatusHealthy, checkedAt, 183*time.Millisecond, []moduleapi.ProbeResult{{Kind: moduleapi.ConnectivityProbeHTTP, Status: moduleapi.ProbeStatusSucceeded, Duration: 183 * time.Millisecond, HTTPStatus: &httpStatus, Summary: "HTTP 200"}}, &moduleapi.RouteExplanation{MatchedStrategy: "platform_default", Decision: "direct", Reason: "policy_disabled"}, &moduleapi.ExitIPDisclosure{Masked: "***.***.45.19", Available: true})
	mock.ExpectBegin()
	mock.ExpectQuery("INSERT INTO platform_connectivity_checks").WithArgs("github", "healthy", int64(183), 200, checkedAt).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(42))
	mock.ExpectExec("INSERT INTO platform_connectivity_reports").WithArgs(int64(42), 1, sqlmock.AnyArg()).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("DELETE FROM platform_connectivity_checks").WithArgs("github", maxConnectivityHistoryLimit).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectCommit()
	check, err := store.Append(context.Background(), report)
	if err != nil {
		t.Fatalf("append report: %v", err)
	}
	if check.ID != 42 || check.TargetID != "github" || check.Latency != 183*time.Millisecond || check.HTTPStatus == nil || *check.HTTPStatus != 200 {
		t.Fatalf("unexpected check: %#v", check)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet SQL expectations: %v", err)
	}
}

func TestConnectivityReportHTTPStatusUsesLastHTTPProbeAndRejectsInvalidValues(t *testing.T) {
	first := 200
	last := 503
	invalid := 700
	report := moduleapi.NewConnectivityReport("github", moduleapi.ConnectivityReportStatusDegraded, time.Now(), time.Millisecond, []moduleapi.ProbeResult{
		{Kind: moduleapi.ConnectivityProbeHTTP, Status: moduleapi.ProbeStatusSucceeded, HTTPStatus: &first},
		{Kind: moduleapi.ConnectivityProbeTLS, Status: moduleapi.ProbeStatusSucceeded, HTTPStatus: &invalid},
		{Kind: moduleapi.ConnectivityProbeHTTP, Status: moduleapi.ProbeStatusFailed, HTTPStatus: &last},
	}, nil, nil)
	status := connectivityReportHTTPStatus(report)
	if status == nil || *status != last {
		t.Fatalf("expected last HTTP response status, got %#v", status)
	}
	if report.Probes[1].HTTPStatus != nil {
		t.Fatalf("expected non-HTTP status to be removed, got %#v", report.Probes[1])
	}
	invalidHTTP := 700
	invalidReport := moduleapi.NewConnectivityReport("github", moduleapi.ConnectivityReportStatusFailed, time.Now(), time.Millisecond, []moduleapi.ProbeResult{{Kind: moduleapi.ConnectivityProbeHTTP, Status: moduleapi.ProbeStatusFailed, HTTPStatus: &invalidHTTP}}, nil, nil)
	if status := connectivityReportHTTPStatus(invalidReport); status != nil {
		t.Fatalf("expected invalid HTTP response status to be rejected, got %#v", status)
	}

	withoutResponse := moduleapi.NewConnectivityReport("smtp", moduleapi.ConnectivityReportStatusFailed, time.Now(), time.Millisecond, []moduleapi.ProbeResult{{Kind: moduleapi.ConnectivityProbeHTTP, Status: moduleapi.ProbeStatusFailed}}, nil, nil)
	if status := connectivityReportHTTPStatus(withoutResponse); status != nil {
		t.Fatalf("expected missing HTTP response to remain unavailable, got %#v", status)
	}
}

func TestSQLConnectivityStoreLatestProjectsHTTPStatusAndUnavailableValue(t *testing.T) {
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
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT DISTINCT ON (target_id) id, target_id, status, latency_ms, http_status, checked_at
		FROM platform_connectivity_checks ORDER BY target_id, checked_at DESC, id DESC`)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "target_id", "status", "latency_ms", "http_status", "checked_at"}).
			AddRow(int64(42), "github", "healthy", int64(183), 200, checkedAt).
			AddRow(int64(43), "smtp", "failed", int64(42), nil, checkedAt))

	checks, err := store.Latest(context.Background())
	if err != nil {
		t.Fatalf("load latest checks: %v", err)
	}
	if len(checks) != 2 || checks[0].HTTPStatus == nil || *checks[0].HTTPStatus != 200 || checks[1].HTTPStatus != nil {
		t.Fatalf("expected HTTP and unavailable projections, got %#v", checks)
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
