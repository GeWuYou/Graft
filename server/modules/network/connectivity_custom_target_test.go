package network

import (
	"context"
	"database/sql"
	"errors"
	"net/netip"
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

func TestResolvePublicEndpointRejectsLiteralPrivateAddress(t *testing.T) {
	if _, err := resolvePublicEndpoint(context.Background(), nil, "127.0.0.1"); err == nil {
		t.Fatal("expected loopback literal to be rejected before DNS resolution")
	}
}

func TestResolvePublicEndpointRejectsPrivateDNSResult(t *testing.T) {
	resolver := connectivityResolverStub{addresses: []netip.Addr{netip.MustParseAddr("10.0.0.1")}}
	if _, err := resolvePublicEndpoint(context.Background(), resolver, "example.com"); err == nil {
		t.Fatal("expected private DNS result to be rejected")
	}
}

type connectivityResolverStub struct {
	addresses []netip.Addr
	err       error
}

func (s connectivityResolverStub) LookupNetIP(context.Context, string, string) ([]netip.Addr, error) {
	return s.addresses, s.err
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

func TestSQLConnectivityStoreCustomTargetReadAndDeleteNotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("open SQL mock: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	store, err := NewSQLConnectivityStore(db)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	createdAt := time.Date(2026, 8, 4, 14, 33, 0, 0, time.UTC)
	mock.ExpectQuery("SELECT target_id, display_name, endpoint, enabled, created_at FROM platform_connectivity_custom_targets WHERE deleted_at = 0").WithArgs(maxConnectivityTargetListSize).WillReturnRows(sqlmock.NewRows([]string{"target_id", "display_name", "endpoint", "enabled", "created_at"}).AddRow("custom-status", "Status", "https://example.com/status", true, createdAt))
	targets, err := store.ListCustomTargets(context.Background())
	if err != nil || len(targets) != 1 || targets[0].ID != "custom-status" || !targets[0].CreatedAt.Equal(createdAt) {
		t.Fatalf("unexpected custom target list %#v, %v", targets, err)
	}
	mock.ExpectQuery("SELECT target_id, display_name, endpoint, enabled, created_at FROM platform_connectivity_custom_targets WHERE target_id").WithArgs("missing").WillReturnError(sql.ErrNoRows)
	if _, err := store.CustomTarget(context.Background(), "missing"); !errors.Is(err, errCustomConnectivityTargetNotFound) {
		t.Fatalf("expected target not found, got %v", err)
	}
	mock.ExpectExec("UPDATE platform_connectivity_custom_targets").WithArgs("missing", uint64(9)).WillReturnResult(sqlmock.NewResult(0, 0))
	if err := store.DeleteCustomTarget(context.Background(), "missing", 9); !errors.Is(err, errCustomConnectivityTargetNotFound) {
		t.Fatalf("expected delete not found, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet SQL expectations: %v", err)
	}
}
