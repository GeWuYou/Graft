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
	"graft/server/internal/module"
	"graft/server/internal/permission"
)

type stopContextRecorderRuntime struct {
	registeredJobs []cronx.Job
	startCtx       context.Context
	stopCtx        context.Context
}

func (r *stopContextRecorderRuntime) RegisterJob(job cronx.Job) error {
	r.registeredJobs = append(r.registeredJobs, job)
	return nil
}

func (r *stopContextRecorderRuntime) RemoveJob(_ string) error { return nil }

func (r *stopContextRecorderRuntime) Start(ctx context.Context) error {
	r.startCtx = ctx
	return nil
}

func (r *stopContextRecorderRuntime) Stop(ctx context.Context) error {
	r.stopCtx = ctx
	return nil
}

func newModuleTestContext() *module.Context {
	return &module.Context{
		Logger:             zap.NewNop(),
		Config:             &config.Config{},
		I18n:               i18n.MustNew(config.I18nConfig{DefaultLocale: "zh-CN", FallbackLocale: "zh-CN", SupportedLocales: []string{"zh-CN"}}),
		EventBus:           eventbus.New(zap.NewNop()),
		Services:           container.New(),
		MenuRegistry:       menu.NewRegistry(),
		PermissionRegistry: permission.NewRegistry(),
		CronRegistry:       cronx.NewRegistry(),
	}
}

// TestBootRejectsInvalidJobs 验证 scheduler 插件会在 Boot 阶段拒绝非法任务声明。
func TestBootRejectsInvalidJobs(t *testing.T) {
	ctx := newModuleTestContext()
	ctx.CronRegistry.Register(cronx.Job{Name: "invalid", Schedule: "*/1 * * * * *"})

	pluginInstance := NewModule()
	if err := pluginInstance.Register(ctx); err != nil {
		t.Fatalf("register plugin: %v", err)
	}

	err := pluginInstance.Boot(&module.Context{LifecycleContext: context.Background(), CronRegistry: ctx.CronRegistry, Logger: ctx.Logger})
	if err == nil {
		t.Fatal("expected invalid job boot to fail")
	}
}

// TestBootRegistersJobsAddedAfterRegister 验证 scheduler 插件会在 Boot 阶段读取最终 registry，
// 而不是在 Register 阶段提前快照。
func TestBootRegistersJobsAddedAfterRegister(t *testing.T) {
	ctx := newModuleTestContext()
	ctx.CronRegistry.Register(cronx.Job{
		Name:     "first",
		Schedule: "*/1 * * * * *",
		Run:      func(context.Context) error { return nil },
	})

	lifecycleCtx := context.Background()
	runtimeRecorder := &stopContextRecorderRuntime{}
	pluginInstance := NewModule()
	pluginInstance.runtime = runtimeRecorder

	if err := pluginInstance.Register(ctx); err != nil {
		t.Fatalf("register plugin: %v", err)
	}

	ctx.CronRegistry.Register(cronx.Job{
		Name:     "second",
		Schedule: "*/1 * * * * *",
		Run:      func(context.Context) error { return nil },
	})

	if err := pluginInstance.Boot(&module.Context{
		LifecycleContext: lifecycleCtx,
		Logger:           ctx.Logger,
		CronRegistry:     ctx.CronRegistry,
	}); err != nil {
		t.Fatalf("boot plugin: %v", err)
	}

	if len(runtimeRecorder.registeredJobs) != 2 {
		t.Fatalf("expected 2 registered jobs, got %d", len(runtimeRecorder.registeredJobs))
	}
	if runtimeRecorder.registeredJobs[0].Name != "first" || runtimeRecorder.registeredJobs[1].Name != "second" {
		t.Fatalf("expected boot to register final registry snapshot, got %q then %q", runtimeRecorder.registeredJobs[0].Name, runtimeRecorder.registeredJobs[1].Name)
	}
	if runtimeRecorder.startCtx != lifecycleCtx {
		t.Fatal("expected boot to pass lifecycle context into scheduler runtime start")
	}
}

// TestBootRunsRegisteredJobs 验证 scheduler 插件会在 Boot 后驱动 registry 中的任务执行。
func TestBootRunsRegisteredJobs(t *testing.T) {
	ctx := newModuleTestContext()
	triggered := make(chan struct{}, 1)
	lifecycleCtx, cancel := context.WithCancel(context.Background())
	defer cancel()
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

	pluginInstance := NewModule()
	if err := pluginInstance.Register(ctx); err != nil {
		t.Fatalf("register plugin: %v", err)
	}
	ctx.LifecycleContext = lifecycleCtx
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
	pluginInstance := NewModule()
	pluginInstance.runtime = runtimeRecorder

	if err := pluginInstance.Shutdown(&module.Context{LifecycleContext: lifecycleCtx}); err != nil {
		t.Fatalf("shutdown plugin: %v", err)
	}
	if runtimeRecorder.stopCtx != lifecycleCtx {
		t.Fatal("expected scheduler shutdown to forward lifecycle context")
	}
}
