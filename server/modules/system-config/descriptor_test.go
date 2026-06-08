package systemconfig

import (
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"

	"graft/server/internal/configregistry"
	"graft/server/internal/container"
	"graft/server/internal/module"
)

func TestDescriptorBuildAllowsMissingUserService(t *testing.T) {
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

	services := container.New()
	if err := services.RegisterSingleton((*sql.DB)(nil), func(container.Resolver) (any, error) {
		return db, nil
	}); err != nil {
		t.Fatalf("register sql db: %v", err)
	}
	registry := configregistry.NewRegistry()
	if err := services.RegisterSingleton((*configregistry.Registry)(nil), func(container.Resolver) (any, error) {
		return registry, nil
	}); err != nil {
		t.Fatalf("register config registry: %v", err)
	}

	descriptor := NewModuleSpec()
	if _, err := descriptor.Build(module.BuildContext{Services: services}); err != nil {
		t.Fatalf("build system-config without user service: %v", err)
	}
}
