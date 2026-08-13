package registry

import (
	"database/sql"
	"testing"

	containerdi "graft/server/internal/container"
	"graft/server/internal/module"
)

// TestRegistryModuleSpecBuildDoesNotRequireRegisteredUserCapabilities 验证构造阶段不依赖其它模块在 Register 阶段才发布的能力。
func TestRegistryModuleSpecBuildDoesNotRequireRegisteredUserCapabilities(t *testing.T) {
	services := containerdi.New()
	if err := services.RegisterSingleton((*sql.DB)(nil), func(containerdi.Resolver) (any, error) {
		return &sql.DB{}, nil
	}); err != nil {
		t.Fatal(err)
	}

	built, err := NewModuleSpec().Build(module.BuildContext{Services: services})
	if err != nil {
		t.Fatalf("build registry module without user capability: %v", err)
	}
	if built == nil {
		t.Fatal("built registry module is nil")
	}
}
