package database

import (
	"testing"
	"time"

	"graft/server/internal/config"
)

func TestOpenReturnsSharedSQLPool(t *testing.T) {
	resources, err := Open(config.DatabaseConfig{
		Driver:          "postgres",
		URL:             "postgres://graft@localhost:5432/graft?sslmode=disable",
		MaxOpenConns:    25,
		MaxIdleConns:    10,
		ConnMaxLifetime: time.Hour,
		ConnMaxIdleTime: 30 * time.Minute,
	})
	if err != nil {
		t.Fatalf("open database resources: %v", err)
	}
	t.Cleanup(func() {
		if closeErr := Close(resources); closeErr != nil {
			t.Fatalf("close database resources: %v", closeErr)
		}
	})

	if resources == nil {
		t.Fatal("expected database resources")
	}
	if resources.SQL == nil {
		t.Fatal("expected shared sql pool")
	}
	if got := resources.SQL.Stats().MaxOpenConnections; got != 25 {
		t.Fatalf("expected max open connections 25, got %d", got)
	}
}

func TestCloseAllowsNilResources(t *testing.T) {
	if err := Close(nil); err != nil {
		t.Fatalf("expected nil resources close to succeed, got %v", err)
	}
}
