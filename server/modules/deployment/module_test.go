package deployment

import (
	"errors"
	"strings"
	"testing"

	containerdi "graft/server/internal/container"
	"graft/server/internal/module"
	"graft/server/internal/moduleapi"
)

func TestModuleRegisterAddsDependencyResolutionContext(t *testing.T) {
	err := NewModule().Register(&module.Context{Services: containerdi.New()})
	if !errors.Is(err, containerdi.ErrServiceNotRegistered) || !strings.Contains(err.Error(), "resolve Docker facts provider") {
		t.Fatalf("Register error = %v, want Docker facts resolution context", err)
	}
}

func TestModuleRegisterAddsCapabilityRegistrationContext(t *testing.T) {
	services := containerdi.New()
	if err := services.RegisterSingleton((*moduleapi.DockerFactsProvider)(nil), func(containerdi.Resolver) (any, error) {
		return dockerFactsStub{}, nil
	}); err != nil {
		t.Fatalf("register Docker facts provider: %v", err)
	}
	if err := services.RegisterSingleton((*moduleapi.DeploymentRuntime)(nil), func(containerdi.Resolver) (any, error) {
		return NewRuntime(nil, dockerFactsStub{}), nil
	}); err != nil {
		t.Fatalf("register deployment runtime: %v", err)
	}

	err := NewModule().Register(&module.Context{Services: services})
	if err == nil || !strings.Contains(err.Error(), "register deployment runtime") {
		t.Fatalf("Register error = %v, want deployment runtime registration context", err)
	}
}
