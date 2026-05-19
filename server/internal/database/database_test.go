package database

import (
	"testing"

	"graft/server/internal/config"
)

func TestOpenReturnsSharedSQLPool(t *testing.T) {
	resources, err := Open(config.DatabaseConfig{
		Driver: "postgres",
		URL:    "postgres://graft@localhost:5432/graft?sslmode=disable",
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
}

func TestCloseAllowsNilResources(t *testing.T) {
	if err := Close(nil); err != nil {
		t.Fatalf("expected nil resources close to succeed, got %v", err)
	}
}
