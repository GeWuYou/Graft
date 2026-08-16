package project

import (
	"context"
	"strings"
	"testing"

	"graft/server/internal/module"
)

func TestProjectModuleBootRequiresLifecycleContext(t *testing.T) {
	projectModule := NewModule(&Service{})

	err := projectModule.Boot(&module.Context{})
	if err == nil || !strings.Contains(err.Error(), "project lifecycle context is required") {
		t.Fatalf("boot error = %v, want missing lifecycle context", err)
	}

	if err := projectModule.Boot(&module.Context{LifecycleContext: context.Background()}); err != nil {
		t.Fatalf("boot with lifecycle context: %v", err)
	}
}
