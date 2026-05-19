package app

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"graft/server/internal/config"
	"graft/server/internal/container"
	"graft/server/internal/cronx"
	"graft/server/internal/database"
	"graft/server/internal/ent"
	"graft/server/internal/eventbus"
	"graft/server/internal/httpx"
	"graft/server/internal/i18n"
	"graft/server/internal/menu"
	"graft/server/internal/permission"
	"graft/server/internal/plugin"
)

type shutdownRecorderPlugin struct {
	name        string
	shutdownLog *[]string
	err         error
}

func (p shutdownRecorderPlugin) Name() string { return p.name }

func (p shutdownRecorderPlugin) Version() string { return "test" }

func (p shutdownRecorderPlugin) DependsOn() []string { return nil }

func (p shutdownRecorderPlugin) Register(_ *plugin.Context) error { return nil }

func (p shutdownRecorderPlugin) Boot(_ *plugin.Context) error { return nil }

func (p shutdownRecorderPlugin) Shutdown(_ *plugin.Context) error {
	*p.shutdownLog = append(*p.shutdownLog, p.name)
	return p.err
}

// TestShutdownPluginsUsesReverseOrder 验证插件关闭顺序与启动顺序相反，
// 以便后启动的依赖先完成资源释放。
func TestShutdownPluginsUsesReverseOrder(t *testing.T) {
	log := make([]string, 0, 3)
	plugins := []plugin.Plugin{
		shutdownRecorderPlugin{name: "user", shutdownLog: &log},
		shutdownRecorderPlugin{name: "rbac", shutdownLog: &log},
		shutdownRecorderPlugin{name: "audit", shutdownLog: &log},
	}

	if err := shutdownPlugins(&plugin.Context{}, plugins); err != nil {
		t.Fatalf("shutdown plugins: %v", err)
	}

	expected := []string{"audit", "rbac", "user"}
	for index, name := range expected {
		if log[index] != name {
			t.Fatalf("expected shutdown order %v, got %v", expected, log)
		}
	}
}

// TestShutdownPluginsAggregatesErrors 验证多个插件关闭失败时会聚合错误，
// 避免后续失败被前一个失败覆盖。
//
// 这里直接构造返回固定错误的测试插件，目的是只锁定关闭聚合语义，
// 不把断言耦合到 Register 或 Boot 的其它生命周期分支。
func TestShutdownPluginsAggregatesErrors(t *testing.T) {
	plugins := []plugin.Plugin{
		shutdownRecorderPlugin{name: "user", shutdownLog: &[]string{}, err: errors.New("user failed")},
		shutdownRecorderPlugin{name: "rbac", shutdownLog: &[]string{}, err: errors.New("rbac failed")},
	}

	err := shutdownPlugins(&plugin.Context{}, plugins)
	if err == nil {
		t.Fatal("expected shutdown error")
	}
	if !errors.Is(err, plugins[0].(shutdownRecorderPlugin).err) {
		t.Fatal("expected joined error to include user failure")
	}
	if !errors.Is(err, plugins[1].(shutdownRecorderPlugin).err) {
		t.Fatal("expected joined error to include rbac failure")
	}
}

type eventBusRecorderPlugin struct {
	registerEventBus eventbus.Bus
	bootEventBus     eventbus.Bus
}

func (p *eventBusRecorderPlugin) Name() string { return "eventbus-recorder" }

func (p *eventBusRecorderPlugin) Version() string { return "test" }

func (p *eventBusRecorderPlugin) DependsOn() []string { return nil }

func (p *eventBusRecorderPlugin) Register(ctx *plugin.Context) error {
	p.registerEventBus = ctx.EventBus
	return nil
}

func (p *eventBusRecorderPlugin) Boot(ctx *plugin.Context) error {
	p.bootEventBus = ctx.EventBus
	return nil
}

func (p *eventBusRecorderPlugin) Shutdown(_ *plugin.Context) error { return nil }

type lifecycleContextRecorderPlugin struct {
	registerLifecycleContext context.Context
	bootLifecycleContext     context.Context
	shutdownLifecycleContext context.Context
	shutdownLifecycleErr     error
}

func (p *lifecycleContextRecorderPlugin) Name() string { return "lifecycle-context-recorder" }

func (p *lifecycleContextRecorderPlugin) Version() string { return "test" }

func (p *lifecycleContextRecorderPlugin) DependsOn() []string { return nil }

func (p *lifecycleContextRecorderPlugin) Register(ctx *plugin.Context) error {
	p.registerLifecycleContext = ctx.LifecycleContext
	return nil
}

func (p *lifecycleContextRecorderPlugin) Boot(ctx *plugin.Context) error {
	p.bootLifecycleContext = ctx.LifecycleContext
	return nil
}

func (p *lifecycleContextRecorderPlugin) Shutdown(ctx *plugin.Context) error {
	p.shutdownLifecycleContext = ctx.LifecycleContext
	if ctx.LifecycleContext != nil {
		p.shutdownLifecycleErr = ctx.LifecycleContext.Err()
	}
	return nil
}

type i18nFreezeRecorderPlugin struct {
	registerFrozen  bool
	bootFrozen      bool
	bootRegisterErr error
}

func (p *i18nFreezeRecorderPlugin) Name() string { return "i18n-freeze-recorder" }

func (p *i18nFreezeRecorderPlugin) Version() string { return "test" }

func (p *i18nFreezeRecorderPlugin) DependsOn() []string { return nil }

func (p *i18nFreezeRecorderPlugin) Register(ctx *plugin.Context) error {
	p.registerFrozen = ctx.I18n.IsFrozen()
	return ctx.I18n.RegisterMessages(i18n.Registration{
		Namespace: "test-plugin",
		Locale:    i18n.LocaleZHCN,
		Messages: []i18n.MessageResource{
			{Key: "boot.message", Text: "注册阶段文案"},
		},
	})
}

func (p *i18nFreezeRecorderPlugin) Boot(ctx *plugin.Context) error {
	p.bootFrozen = ctx.I18n.IsFrozen()
	p.bootRegisterErr = ctx.I18n.RegisterMessages(i18n.Registration{
		Namespace: "test-plugin",
		Locale:    i18n.LocaleZHCN,
		Messages: []i18n.MessageResource{
			{Key: "late.message", Text: "启动阶段文案"},
		},
	})
	return nil
}

func (p *i18nFreezeRecorderPlugin) Shutdown(_ *plugin.Context) error { return nil }

// TestRegisterCoreServicesExposesRuntimeSingletons 验证 core 装配会把配置、
// event bus、Ent client 与 Redis 客户端注册到运行时容器中。
func TestRegisterCoreServicesExposesRuntimeSingletons(t *testing.T) {
	redisClient := redis.NewClient(&redis.Options{Addr: "127.0.0.1:6379"})
	t.Cleanup(func() {
		_ = redisClient.Close()
	})
	runtimeLogger := zap.NewNop()
	runtimeEventBus := eventbus.New(runtimeLogger)
	entClient := ent.NewClient()

	cfg := &config.Config{
		App: config.AppConfig{Name: "graft", Env: "test"},
		HTTP: config.HTTPConfig{
			Addr: ":8080",
		},
		Database: config.DatabaseConfig{
			Driver: "postgres",
			URL:    "postgres://graft@localhost:5432/graft?sslmode=disable",
		},
		Redis: config.RedisConfig{
			Addr: "localhost:6379",
		},
		Log: config.LogConfig{
			Level: "info",
		},
		I18n: config.I18nConfig{
			DefaultLocale:    "zh-CN",
			FallbackLocale:   "zh-CN",
			SupportedLocales: []string{"zh-CN", "en-US"},
		},
	}
	localizer := i18n.New(cfg.I18n)
	runtime := &Runtime{
		config:   cfg,
		logger:   runtimeLogger,
		i18n:     localizer,
		database: &database.Resources{Client: entClient},
		redis:    redisClient,
		eventBus: runtimeEventBus,
		services: container.New(),
	}

	if err := runtime.registerCoreServices(); err != nil {
		t.Fatalf("register core services: %v", err)
	}

	assertResolvedService(t, runtime.services, (*config.Config)(nil), cfg, "config")
	assertResolvedService(t, runtime.services, (*zap.Logger)(nil), runtimeLogger, "logger")
	assertResolvedService(t, runtime.services, (*i18n.Service)(nil), localizer, "i18n service")
	assertResolvedService(t, runtime.services, (*eventbus.Bus)(nil), runtimeEventBus, "event bus")
	assertResolvedService(t, runtime.services, (*ent.Client)(nil), entClient, "ent client")
	assertResolvedService(t, runtime.services, (*redis.Client)(nil), redisClient, "redis client")
}

func assertResolvedService[T comparable](t *testing.T, resolver container.Resolver, key any, expected T, name string) {
	t.Helper()

	resolvedAny, err := resolver.Resolve(key)
	if err != nil {
		t.Fatalf("resolve %s: %v", name, err)
	}

	resolved, ok := resolvedAny.(T)
	if !ok {
		t.Fatalf("expected resolved %s to have type %T, got %T", name, expected, resolvedAny)
	}
	if resolved != expected {
		t.Fatalf("expected resolved %s to reuse runtime instance", name)
	}
}

// TestRunPassesEventBusIntoPluginContext 验证 Runtime 在 Register 与 Boot
// 阶段向插件注入同一个事件总线实例，避免插件各自持有漂移的协作边界。
func TestRunPassesEventBusIntoPluginContext(t *testing.T) {
	gin.SetMode(gin.TestMode)

	recorder := &eventBusRecorderPlugin{}
	manager := plugin.NewManager()
	if err := manager.RegisterPlugin(recorder); err != nil {
		t.Fatalf("register plugin: %v", err)
	}

	runtimeEventBus := eventbus.New(zap.NewNop())
	runtime := &Runtime{
		config: &config.Config{
			HTTP: config.HTTPConfig{Addr: "127.0.0.1:0"},
		},
		logger:             zap.NewNop(),
		i18n:               i18n.New(config.I18nConfig{DefaultLocale: "zh-CN", FallbackLocale: "zh-CN", SupportedLocales: []string{"zh-CN"}}),
		server:             httpx.NewServer(),
		eventBus:           runtimeEventBus,
		services:           container.New(),
		menuRegistry:       menu.NewRegistry(),
		permissionRegistry: permission.NewRegistry(),
		cronRegistry:       cronx.NewRegistry(),
		pluginManager:      manager,
	}

	runCtx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := runtime.Run(runCtx); err != nil {
		t.Fatalf("run runtime: %v", err)
	}
	if recorder.registerEventBus != runtimeEventBus {
		t.Fatal("expected register phase to receive runtime event bus instance")
	}
	if recorder.bootEventBus != runtimeEventBus {
		t.Fatal("expected boot phase to receive runtime event bus instance")
	}
}

// TestRunPassesLifecycleContextIntoPluginPhases 验证 Runtime 会在插件生命周期内
// 注入显式上下文，并在 Shutdown 阶段切换到独立的有界关闭上下文。
func TestRunPassesLifecycleContextIntoPluginPhases(t *testing.T) {
	gin.SetMode(gin.TestMode)

	recorder := &lifecycleContextRecorderPlugin{}
	manager := plugin.NewManager()
	if err := manager.RegisterPlugin(recorder); err != nil {
		t.Fatalf("register plugin: %v", err)
	}

	runtime := &Runtime{
		config: &config.Config{
			HTTP: config.HTTPConfig{Addr: "127.0.0.1:0"},
		},
		logger:             zap.NewNop(),
		i18n:               i18n.New(config.I18nConfig{DefaultLocale: "zh-CN", FallbackLocale: "zh-CN", SupportedLocales: []string{"zh-CN"}}),
		server:             httpx.NewServer(),
		eventBus:           eventbus.New(zap.NewNop()),
		services:           container.New(),
		menuRegistry:       menu.NewRegistry(),
		permissionRegistry: permission.NewRegistry(),
		cronRegistry:       cronx.NewRegistry(),
		pluginManager:      manager,
	}

	runCtx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := runtime.Run(runCtx); err != nil {
		t.Fatalf("run runtime: %v", err)
	}
	if recorder.registerLifecycleContext != runCtx {
		t.Fatal("expected register phase to receive runtime run context")
	}
	if recorder.bootLifecycleContext != runCtx {
		t.Fatal("expected boot phase to receive runtime run context")
	}
	if recorder.shutdownLifecycleContext == nil {
		t.Fatal("expected shutdown phase to receive lifecycle context")
	}
	if recorder.shutdownLifecycleContext == runCtx {
		t.Fatal("expected shutdown phase to receive bounded shutdown context instead of canceled run context")
	}
	if recorder.shutdownLifecycleErr != nil {
		t.Fatalf("expected shutdown lifecycle context to remain usable, got %v", recorder.shutdownLifecycleErr)
	}
}

func TestRunFreezesI18nRegistryAfterRegisterBeforeBoot(t *testing.T) {
	gin.SetMode(gin.TestMode)

	recorder := &i18nFreezeRecorderPlugin{}
	manager := plugin.NewManager()
	if err := manager.RegisterPlugin(recorder); err != nil {
		t.Fatalf("register plugin: %v", err)
	}

	localizer := i18n.New(config.I18nConfig{
		DefaultLocale:    "zh-CN",
		FallbackLocale:   "zh-CN",
		SupportedLocales: []string{"zh-CN", "en-US"},
	})
	runtime := &Runtime{
		config: &config.Config{
			HTTP: config.HTTPConfig{Addr: "127.0.0.1:0"},
		},
		logger:             zap.NewNop(),
		i18n:               localizer,
		server:             httpx.NewServer(),
		eventBus:           eventbus.New(zap.NewNop()),
		services:           container.New(),
		menuRegistry:       menu.NewRegistry(),
		permissionRegistry: permission.NewRegistry(),
		cronRegistry:       cronx.NewRegistry(),
		pluginManager:      manager,
	}

	runCtx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := runtime.Run(runCtx); err != nil {
		t.Fatalf("run runtime: %v", err)
	}
	if recorder.registerFrozen {
		t.Fatal("expected i18n registry to remain writable during Register")
	}
	if !recorder.bootFrozen {
		t.Fatal("expected i18n registry to be frozen before Boot")
	}
	if recorder.bootRegisterErr == nil {
		t.Fatal("expected Boot phase registration to be rejected after freeze")
	}

	message := localizer.Lookup(i18n.LookupRequest{
		Namespace: "test-plugin",
		Locale:    i18n.LocaleZHCN,
		Key:       "boot.message",
	})
	if message != "注册阶段文案" {
		t.Fatalf("expected register-time message registration to persist, got %q", message)
	}
}

// TestRegisterCoreRoutesHealthzReportsRegistryCounts 验证健康检查接口会返回
// core 注册表当前快照，便于后续观测运行时装配结果。
func TestRegisterCoreRoutesHealthzReportsRegistryCounts(t *testing.T) {
	gin.SetMode(gin.TestMode)

	menuRegistry := menu.NewRegistry()
	menuRegistry.Register(menu.Item{Code: "dashboard", Path: "/dashboard"})
	menuRegistry.Register(menu.Item{Code: "users", Path: "/users"})

	permissionRegistry := permission.NewRegistry()
	permissionRegistry.Register(permission.Item{Code: "dashboard.view"})

	cronRegistry := cronx.NewRegistry()
	cronRegistry.Register(cronx.Job{Name: "cleanup", Schedule: "0 * * * *"})
	cronRegistry.Register(cronx.Job{Name: "sync", Schedule: "*/5 * * * *"})
	cronRegistry.Register(cronx.Job{Name: "audit", Schedule: "0 0 * * *"})

	runtime := &Runtime{
		i18n:               i18n.New(config.I18nConfig{DefaultLocale: "zh-CN", FallbackLocale: "en-US", SupportedLocales: []string{"zh-CN", "en-US"}}),
		menuRegistry:       menuRegistry,
		permissionRegistry: permissionRegistry,
		cronRegistry:       cronRegistry,
	}

	engine := gin.New()
	runtime.registerCoreRoutes(engine)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	engine.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}

	var payload struct {
		Status         string `json:"status"`
		DefaultLocale  string `json:"defaultLocale"`
		FallbackLocale string `json:"fallbackLocale"`
		Menus          int    `json:"menus"`
		Permissions    int    `json:"permissions"`
		Jobs           int    `json:"jobs"`
	}
	if err := json.NewDecoder(recorder.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if payload.Status != "ok" {
		t.Fatalf("expected status ok, got %q", payload.Status)
	}
	if payload.DefaultLocale != "zh-CN" || payload.FallbackLocale != "en-US" {
		t.Fatalf("expected locale snapshot zh-CN/en-US, got %s/%s", payload.DefaultLocale, payload.FallbackLocale)
	}
	if payload.Menus != 2 || payload.Permissions != 1 || payload.Jobs != 3 {
		t.Fatalf("expected registry counts 2/1/3, got %d/%d/%d", payload.Menus, payload.Permissions, payload.Jobs)
	}
}
