package project

import (
	"context"
	"strings"
	"testing"

	"graft/server/internal/module"
)

// TestProjectModuleBootRequiresLifecycleContext 验证项目模块启动必须绑定生命周期上下文，避免订阅实时流脱离模块关闭边界。
func TestProjectModuleBootRequiresLifecycleContext(t *testing.T) {
	projectModule := NewModule(&Service{})

	err := projectModule.Boot(&module.Context{})
	if err == nil || !strings.Contains(err.Error(), "project lifecycle context is required") {
		t.Fatalf("boot error = %v, want missing lifecycle context", err)
	}

	lifecycleContext, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := projectModule.Boot(&module.Context{LifecycleContext: lifecycleContext}); err != nil {
		t.Fatalf("boot with lifecycle context: %v", err)
	}
	if actual := projectModule.service.realtimeStreamContext(); actual != lifecycleContext {
		t.Fatalf("stored lifecycle context = %v, want supplied context", actual)
	}
}
