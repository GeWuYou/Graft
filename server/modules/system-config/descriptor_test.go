package systemconfig

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"

	"graft/server/internal/cachex"
	cachebackend "graft/server/internal/cachex/backend"
	"graft/server/internal/configregistry"
	"graft/server/internal/container"
	"graft/server/internal/module"
	"graft/server/internal/moduleapi"
)

func TestDescriptorBuildAllowsUserServiceRegistrationAfterBuild(t *testing.T) {
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
	cacheManager, err := cachex.NewManager(cachex.ManagerOptions{
		Backend:   cachebackend.NewMemory(),
		Namespace: "test-runtime",
	})
	if err != nil {
		t.Fatalf("new cache manager: %v", err)
	}
	if err := services.RegisterSingleton((*cachex.Manager)(nil), func(container.Resolver) (any, error) {
		return cacheManager, nil
	}); err != nil {
		t.Fatalf("register cache manager: %v", err)
	}

	descriptor := NewModuleSpec()
	if _, err := descriptor.Build(module.BuildContext{Services: services}); err != nil {
		t.Fatalf("build system-config before user service registration: %v", err)
	}
}

func TestDescriptorBuildWithUserService(t *testing.T) {
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
	cacheManager, err := cachex.NewManager(cachex.ManagerOptions{
		Backend:   cachebackend.NewMemory(),
		Namespace: "test-runtime",
	})
	if err != nil {
		t.Fatalf("new cache manager: %v", err)
	}
	if err := services.RegisterSingleton((*cachex.Manager)(nil), func(container.Resolver) (any, error) {
		return cacheManager, nil
	}); err != nil {
		t.Fatalf("register cache manager: %v", err)
	}
	if err := services.RegisterSingleton((*moduleapi.UserService)(nil), func(container.Resolver) (any, error) {
		return descriptorTestUserService{}, nil
	}); err != nil {
		t.Fatalf("register user service: %v", err)
	}

	descriptor := NewModuleSpec()
	if _, err := descriptor.Build(module.BuildContext{Services: services}); err != nil {
		t.Fatalf("build system-config with user service: %v", err)
	}
}

type descriptorTestUserService struct{}

func (descriptorTestUserService) GetUserByID(context.Context, uint64) (moduleapi.UserSummary, error) {
	return moduleapi.UserSummary{}, moduleapi.ErrUserNotFound
}

func (descriptorTestUserService) CountUsers(context.Context) (int, error) {
	return 0, nil
}
