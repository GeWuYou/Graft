package network

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"

	"graft/server/internal/moduleapi"
)

func TestValidateCustomConnectivityEndpointRejectsSSRFAddressesAndAuthorities(t *testing.T) {
	for _, endpoint := range []string{
		"ftp://example.com", "http://user:password@example.com", "http://127.0.0.1", "https://[::1]",
		"http://169.254.169.254", "http://10.0.0.1", "http://192.168.1.1", "http://100.64.0.1",
		"http://198.51.100.1", "http://example.com:8080", "http://localhost", "http://example.com#fragment",
	} {
		if _, err := validateCustomConnectivityEndpoint(endpoint); err == nil {
			t.Fatalf("expected unsafe endpoint %q to be rejected", endpoint)
		}
	}
	if _, err := validateCustomConnectivityEndpoint("https://example.com/health"); err != nil {
		t.Fatalf("expected public HTTPS endpoint to be accepted: %v", err)
	}
}

func TestResolvePublicEndpointRejectsResolvedPrivateAddress(t *testing.T) {
	if _, err := resolvePublicEndpoint(context.Background(), nil, "127.0.0.1"); err == nil {
		t.Fatal("expected loopback literal to be rejected before DNS resolution")
	}
}

func TestSQLConnectivityStoreCustomTargetLifecycle(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("open SQL mock: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	store, err := NewSQLConnectivityStore(db)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	target := CustomConnectivityTarget{ID: "custom-status", DisplayName: "Status", Endpoint: "https://example.com/status", Enabled: true}
	mock.ExpectQuery("INSERT INTO platform_connectivity_custom_targets").WithArgs("custom-status", "Status", "https://example.com/status", uint64(9)).WillReturnRows(sqlmock.NewRows([]string{"target_id", "display_name", "endpoint", "enabled", "created_at"}).AddRow("custom-status", "Status", "https://example.com/status", true, time.Date(2026, 8, 4, 14, 33, 0, 0, time.UTC)))
	created, err := store.CreateCustomTarget(context.Background(), target, 9)
	if err != nil || created.ID != target.ID || !created.Enabled {
		t.Fatalf("unexpected create result %#v, %v", created, err)
	}
	mock.ExpectExec("UPDATE platform_connectivity_custom_targets").WithArgs("custom-status", uint64(9)).WillReturnResult(sqlmock.NewResult(0, 1))
	if err := store.DeleteCustomTarget(context.Background(), moduleapi.ConnectivityTargetID("custom-status"), 9); err != nil {
		t.Fatalf("delete custom target: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet SQL expectations: %v", err)
	}
}
