package scheduler

import (
	"context"
	"testing"
	"time"

	"go.uber.org/zap"

	"graft/server/internal/config"
	"graft/server/internal/container"
	"graft/server/internal/cronx"
	"graft/server/internal/eventbus"
	"graft/server/internal/i18n"
	"graft/server/internal/menu"
	"graft/server/internal/permission"
	"graft/server/internal/plugin"
	"graft/server/internal/store"
)

type pluginTestStoreFactory struct{}

func (pluginTestStoreFactory) Audit() store.AuditRepository { return nil }
func (pluginTestStoreFactory) Users() store.UserRepository  { return nil }
func (pluginTestStoreFactory) Auth() store.AuthRepository   { return nil }
func (pluginTestStoreFactory) RBAC() store.RBACRepository   { return nil }

type stopContextRecorderRuntime struct {
	stopCtx context.Context
}

func (r *stopContextRecorderRuntime) RegisterJob(job cronx.Job) error { return nil }

func (r *stopContextRecorderRuntime) RemoveJob(name string) error { return nil }

func (r *stopContextRecorderRuntime) Start() error { return nil }

func (r *stopContextRecorderRuntime) Stop(ctx context.Context) error {
	r.stopCtx = ctx
	return nil
}

func newPluginTestContext() *plugin.Context {
	return &plugin.Context{
		Logger:             zap.NewNop(),
		Config:             &config.Config{},
		I18n:               i18n.New(config.I18nConfig{DefaultLocale: "zh-CN", FallbackLocale: "zh-CN", SupportedLocales: []string{"zh-CN"}}),
		EventBus:           eventbus.New(zap.NewNop()),
		Services:           container.New(),
		Stores:             pluginTestStoreFactory{},
		MenuRegistry:       menu.NewRegistry(),
		PermissionRegistry: permission.NewRegistry(),
		CronRegistry:       cronx.NewRegistry(),
	}
}

// TestRegisterRejectsInvalidJobs 验证 scheduler 插件会在 Register 阶段阻止非法任务声明进入运行态。
func TestRegisterRejectsInvalidJobs(t *testing.T) {
	ctx := newPluginTestContext()
	ctx.CronRegistry.Register(cronx.Job{Name: "invalid", Schedule: "*/1 * * * * *"})

	err := NewPlugin().Register(ctx)
	if err == nil {
		t.Fatal("expected invalid job registration to fail")
	}
}

// TestBootRunsRegisteredJobs 验证 scheduler 插件会在 Boot 后驱动 registry 中的任务执行。
func TestBootRunsRegisteredJobs(t *testing.T) {
	ctx := newPluginTestContext()
	triggered := make(chan struct{}, 1)
	ctx.CronRegistry.Register(cronx.Job{
		Name:     "heartbeat",
		Schedule: "*/1 * * * * *",
		Plugin:   "test",
		Run: func(context.Context) error {
			select {
			case triggered <- struct{}{}:
			default:
			}
			return nil
		},
	})

	pluginInstance := NewPlugin()
	if err := pluginInstance.Register(ctx); err != nil {
		t.Fatalf("register plugin: %v", err)
	}
	if err := pluginInstance.Boot(ctx); err != nil {
		t.Fatalf("boot plugin: %v", err)
	}
	defer func() {
		_ = pluginInstance.Shutdown(ctx)
	}()

	select {
	case <-triggered:
	case <-time.After(1500 * time.Millisecond):
		t.Fatal("expected scheduled job to run after boot")
	}
}

// TestShutdownUsesLifecycleContext 验证 scheduler 插件会把生命周期关闭上下文
// 传递给底层 runtime，而不是回退到脱离宿主约束的全新 Background。
func TestShutdownUsesLifecycleContext(t *testing.T) {
	lifecycleCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	runtimeRecorder := &stopContextRecorderRuntime{}
	pluginInstance := NewPlugin()
	pluginInstance.runtime = runtimeRecorder

	if err := pluginInstance.Shutdown(&plugin.Context{LifecycleContext: lifecycleCtx}); err != nil {
		t.Fatalf("shutdown plugin: %v", err)
	}
	if runtimeRecorder.stopCtx != lifecycleCtx {
		t.Fatal("expected scheduler shutdown to forward lifecycle context")
	}
}
