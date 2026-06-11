// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package app

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"graft/server/internal/config"
	"graft/server/internal/container"
	"graft/server/internal/cronx"
	"graft/server/internal/dashboard"
	"graft/server/internal/database"
	"graft/server/internal/eventbus"
	"graft/server/internal/httpx"
	"graft/server/internal/i18n"
	"graft/server/internal/logger"
	"graft/server/internal/menu"
	"graft/server/internal/module"
	"graft/server/internal/moduleregistry"
	"graft/server/internal/moduleruntime"
	"graft/server/internal/permission"
	testent "graft/server/internal/testent"
)

type runtimeAccessLogRecorderRepo struct {
	created []httpx.CreateAccessLogInput
	deleted []time.Time
}

func (r *runtimeAccessLogRecorderRepo) CreateAccessLog(_ context.Context, input httpx.CreateAccessLogInput) (httpx.AccessLog, error) {
	r.created = append(r.created, input)
	return httpx.AccessLog{}, nil
}

func (r *runtimeAccessLogRecorderRepo) CreateAccessLogs(_ context.Context, inputs []httpx.CreateAccessLogInput) ([]httpx.AccessLog, error) {
	r.created = append(r.created, inputs...)
	return []httpx.AccessLog{}, nil
}

func (r *runtimeAccessLogRecorderRepo) DeleteAccessLogsBefore(_ context.Context, cutoff time.Time) (int64, error) {
	r.deleted = append(r.deleted, cutoff)
	return 0, nil
}

func (r *runtimeAccessLogRecorderRepo) DeleteAccessLogsBeforeLimit(_ context.Context, cutoff time.Time, _ int) (int64, error) {
	r.deleted = append(r.deleted, cutoff)
	return 0, nil
}

func (r *runtimeAccessLogRecorderRepo) ListAccessLogs(context.Context, httpx.AccessLogListQuery) (httpx.AccessLogListResult, error) {
	return httpx.AccessLogListResult{}, nil
}

func (r *runtimeAccessLogRecorderRepo) GetAccessLogByID(context.Context, uint64) (httpx.AccessLog, error) {
	return httpx.AccessLog{}, httpx.ErrAccessLogNotFound
}

type runtimeAppLogRecorderRepo struct {
	mu      sync.Mutex
	created []logger.CreateAppLogInput
	deleted []time.Time
}

func (r *runtimeAppLogRecorderRepo) CreateAppLog(_ context.Context, input logger.CreateAppLogInput) (logger.AppLogRecord, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.created = append(r.created, input)
	return logger.AppLogRecord{}, nil
}

func (r *runtimeAppLogRecorderRepo) DeleteAppLogsBefore(_ context.Context, cutoff time.Time) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.deleted = append(r.deleted, cutoff)
	return 0, nil
}

func (r *runtimeAppLogRecorderRepo) DeleteAppLogsBeforeLimit(_ context.Context, cutoff time.Time, _ int) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.deleted = append(r.deleted, cutoff)
	return 0, nil
}

func (r *runtimeAppLogRecorderRepo) ListAppLogs(context.Context, logger.AppLogListQuery) (logger.AppLogListResult, error) {
	return logger.AppLogListResult{}, nil
}

func (r *runtimeAppLogRecorderRepo) GetAppLogByID(context.Context, uint64) (logger.AppLogRecord, error) {
	return logger.AppLogRecord{}, logger.ErrAppLogNotFound
}

type shutdownRecorderModule struct {
	name        string
	shutdownLog *[]string
	err         error
}

func (p shutdownRecorderModule) Register(_ *module.Context) error { return nil }

func (p shutdownRecorderModule) Boot(_ *module.Context) error { return nil }

func (p shutdownRecorderModule) Shutdown(_ *module.Context) error {
	*p.shutdownLog = append(*p.shutdownLog, p.name)
	return p.err
}

type bootRecorderModule struct {
	err    error
	booted *bool
}

func (p bootRecorderModule) Register(_ *module.Context) error { return nil }

func (p bootRecorderModule) Boot(_ *module.Context) error {
	if p.booted != nil {
		*p.booted = true
	}
	return p.err
}

func (p bootRecorderModule) Shutdown(_ *module.Context) error { return nil }

// TestShutdownModulesUsesReverseOrder 验证模块关闭顺序与启动顺序相反，
// 以便后启动的依赖先完成资源释放。
func TestShutdownModulesUsesReverseOrder(t *testing.T) {
	log := make([]string, 0, 3)
	modules := []module.RuntimeModule{
		mustDescribeRuntimeTestModule(module.Spec{ID: "user"}, shutdownRecorderModule{name: "user", shutdownLog: &log}),
		mustDescribeRuntimeTestModule(module.Spec{ID: "rbac"}, shutdownRecorderModule{name: "rbac", shutdownLog: &log}),
		mustDescribeRuntimeTestModule(module.Spec{ID: "audit"}, shutdownRecorderModule{name: "audit", shutdownLog: &log}),
	}

	if err := shutdownModules(&module.Context{}, modules); err != nil {
		t.Fatalf("shutdown modules: %v", err)
	}

	expected := []string{"audit", "rbac", "user"}
	for index, name := range expected {
		if log[index] != name {
			t.Fatalf("expected shutdown order %v, got %v", expected, log)
		}
	}
}

// TestShutdownModulesAggregatesErrors 验证多个模块关闭失败时会聚合错误，
// 避免后续失败被前一个失败覆盖。
//
// 这里直接构造返回固定错误的测试模块，目的是只锁定关闭聚合语义，
// 不把断言耦合到 Register 或 Boot 的其它生命周期分支。
func TestShutdownModulesAggregatesErrors(t *testing.T) {
	userErr := errors.New("user failed")
	rbacErr := errors.New("rbac failed")
	modules := []module.RuntimeModule{
		mustDescribeRuntimeTestModule(module.Spec{ID: "user"}, shutdownRecorderModule{name: "user", shutdownLog: &[]string{}, err: userErr}),
		mustDescribeRuntimeTestModule(module.Spec{ID: "rbac"}, shutdownRecorderModule{name: "rbac", shutdownLog: &[]string{}, err: rbacErr}),
	}

	err := shutdownModules(&module.Context{}, modules)
	if err == nil {
		t.Fatal("expected shutdown error")
	}
	if !errors.Is(err, userErr) {
		t.Fatal("expected joined error to include user failure")
	}
	if !errors.Is(err, rbacErr) {
		t.Fatal("expected joined error to include rbac failure")
	}
}

func TestBootModulesWritesHighSignalAppLogs(t *testing.T) {
	repo := &runtimeAppLogRecorderRepo{}
	runtime := &Runtime{
		logger:           zap.NewNop(),
		services:         container.New(),
		appLogRepository: repo,
	}
	if err := runtime.registerCoreServices(); err != nil {
		t.Fatalf("register core services: %v", err)
	}

	moduleCtx := &module.Context{
		LifecycleContext: context.Background(),
		Services:         runtime.services,
	}
	ordered := []module.RuntimeModule{
		mustDescribeRuntimeTestModule(module.Spec{ID: "user"}, bootRecorderModule{}),
	}

	booted, err := runtime.bootModules(moduleCtx, ordered, nil)
	if err != nil {
		t.Fatalf("boot modules: %v", err)
	}
	if len(booted) != 1 {
		t.Fatalf("expected one booted module, got %d", len(booted))
	}
	assertEventuallyAppLogRecord(t, repo, "module boot completed", "user")
}

func TestBootModulesWritesFailureAppLog(t *testing.T) {
	repo := &runtimeAppLogRecorderRepo{}
	runtime := &Runtime{
		logger:           zap.NewNop(),
		services:         container.New(),
		appLogRepository: repo,
	}
	if err := runtime.registerCoreServices(); err != nil {
		t.Fatalf("register core services: %v", err)
	}

	bootErr := errors.New("boot exploded")
	moduleCtx := &module.Context{
		LifecycleContext: context.Background(),
		Services:         runtime.services,
	}
	ordered := []module.RuntimeModule{
		mustDescribeRuntimeTestModule(module.Spec{ID: "user"}, bootRecorderModule{err: bootErr}),
	}

	if _, err := runtime.bootModules(moduleCtx, ordered, nil); err == nil {
		t.Fatal("expected boot failure")
	}
	assertEventuallyAppLogRecord(t, repo, "module boot failed", "user")
}

func TestInjectedAppLoggerFallsBackToRuntimeLoggerWhenServicesMissing(t *testing.T) {
	repo := &runtimeAppLogRecorderRepo{}
	runtime := &Runtime{
		logger:           zap.NewNop(),
		appLogRepository: repo,
	}

	runtime.injectedAppLogger().Info(context.Background(), "runtime fallback log")

	assertEventuallyAppLogRecord(t, repo, "runtime fallback log", "")
}

func assertEventuallyAppLogRecord(t *testing.T, repo *runtimeAppLogRecorderRepo, message string, moduleName string) {
	t.Helper()

	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		repo.mu.Lock()
		for _, record := range repo.created {
			moduleField, hasModule := record.Fields["module"]
			if record.Message == message && (moduleName == "" || hasModule && moduleField == moduleName) {
				repo.mu.Unlock()
				return
			}
		}
		repo.mu.Unlock()
		time.Sleep(time.Millisecond)
	}
	repo.mu.Lock()
	defer repo.mu.Unlock()
	t.Fatalf("expected app log message %q for module %q, got %#v", message, moduleName, repo.created)
}

type eventBusRecorderModule struct {
	registerEventBus eventbus.Bus
	bootEventBus     eventbus.Bus
	cancelOnBoot     context.CancelFunc
}

func (p *eventBusRecorderModule) Register(ctx *module.Context) error {
	p.registerEventBus = ctx.EventBus
	return nil
}

func (p *eventBusRecorderModule) Boot(ctx *module.Context) error {
	p.bootEventBus = ctx.EventBus
	if p.cancelOnBoot != nil {
		p.cancelOnBoot()
	}
	return nil
}

func (p *eventBusRecorderModule) Shutdown(_ *module.Context) error { return nil }

type lifecycleContextRecorderModule struct {
	registerLifecycleContext context.Context
	bootLifecycleContext     context.Context
	shutdownLifecycleContext context.Context
	shutdownLifecycleErr     error
	cancelOnBoot             context.CancelFunc
}

func (p *lifecycleContextRecorderModule) Register(ctx *module.Context) error {
	p.registerLifecycleContext = ctx.LifecycleContext
	return nil
}

func (p *lifecycleContextRecorderModule) Boot(ctx *module.Context) error {
	p.bootLifecycleContext = ctx.LifecycleContext
	if p.cancelOnBoot != nil {
		p.cancelOnBoot()
	}
	return nil
}

func (p *lifecycleContextRecorderModule) Shutdown(ctx *module.Context) error {
	p.shutdownLifecycleContext = ctx.LifecycleContext
	if ctx.LifecycleContext != nil {
		p.shutdownLifecycleErr = ctx.LifecycleContext.Err()
	}
	return nil
}

type i18nFreezeRecorderModule struct {
	registerFrozen  bool
	bootFrozen      bool
	bootRegisterErr error
	cancelOnBoot    context.CancelFunc
}

func (p *i18nFreezeRecorderModule) Register(ctx *module.Context) error {
	p.registerFrozen = ctx.I18n.IsFrozen()
	return ctx.I18n.RegisterMessages(i18n.Registration{
		Namespace: "test-module",
		Locale:    i18n.LocaleZHCN,
		Messages: []i18n.MessageResource{
			{Key: "boot.message", Text: "注册阶段文案"},
		},
	})
}

func (p *i18nFreezeRecorderModule) Boot(ctx *module.Context) error {
	p.bootFrozen = ctx.I18n.IsFrozen()
	p.bootRegisterErr = ctx.I18n.RegisterMessages(i18n.Registration{
		Namespace: "test-module",
		Locale:    i18n.LocaleZHCN,
		Messages: []i18n.MessageResource{
			{Key: "late.message", Text: "启动阶段文案"},
		},
	})
	if p.cancelOnBoot != nil {
		p.cancelOnBoot()
	}
	return nil
}

func (p *i18nFreezeRecorderModule) Shutdown(_ *module.Context) error { return nil }

func mustDescribeRuntimeTestModule(spec module.Spec, instance module.Module) module.RuntimeModule {
	builtModule, err := module.Spec{
		ID:           spec.ID,
		Dependencies: append([]string(nil), spec.Dependencies...),
		Builder: module.BuilderFunc(func(module.BuildContext) (module.Module, error) {
			return instance, nil
		}),
	}.Build(module.BuildContext{})
	if err != nil {
		panic(err)
	}
	return builtModule
}

// TestRegisterCoreServicesExposesRuntimeSingletons 验证 core 装配会把配置、
// event bus、共享 SQL 连接池与 Redis 客户端注册到运行时容器中。
func TestRegisterCoreServicesExposesRuntimeSingletons(t *testing.T) {
	redisClient := redis.NewClient(&redis.Options{Addr: "127.0.0.1:6379"})
	t.Cleanup(func() {
		_ = redisClient.Close()
	})
	runtimeLogger := zap.NewNop()
	runtimeEventBus := eventbus.New(runtimeLogger)
	sqlDB := &sql.DB{}

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
			Level:           "info",
			AppLogRetention: 3 * 24 * time.Hour,
		},
		I18n: config.I18nConfig{
			DefaultLocale:    "zh-CN",
			FallbackLocale:   "zh-CN",
			SupportedLocales: []string{"zh-CN", "en-US"},
		},
	}
	localizer := i18n.MustNew(cfg.I18n)
	runtime := &Runtime{
		config:           cfg,
		logger:           runtimeLogger,
		i18n:             localizer,
		database:         &database.Resources{SQL: sqlDB},
		redis:            redisClient,
		eventBus:         runtimeEventBus,
		services:         container.New(),
		appLogRepository: &runtimeAppLogRecorderRepo{},
	}

	if err := runtime.registerCoreServices(); err != nil {
		t.Fatalf("register core services: %v", err)
	}

	assertResolvedService(t, runtime.services, (*config.Config)(nil), cfg, "config")
	assertResolvedService(t, runtime.services, (*zap.Logger)(nil), runtimeLogger, "logger")
	assertResolvedService(t, runtime.services, (*i18n.Service)(nil), localizer, "i18n service")
	assertResolvedService(t, runtime.services, (*eventbus.Bus)(nil), runtimeEventBus, "event bus")
	assertResolvedService(t, runtime.services, (*sql.DB)(nil), sqlDB, "sql db")
	assertResolvedService(t, runtime.services, (*redis.Client)(nil), redisClient, "redis client")
	assertAppLoggerRegistered(t, runtime.services)
	assertResolvedService(t, runtime.services, (*logger.AppLogRepository)(nil), runtime.appLogRepository, "app log repository")
	assertServiceKeyNotRegistered(t, runtime.services, (*testent.Client)(nil), "*ent.Client")
}

func TestNewRuntimeCoreWiresAccessLogRepositoryIntoServer(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Cleanup(func() {
		gin.SetMode(gin.TestMode)
	})

	recorderRepo := &runtimeAccessLogRecorderRepo{}
	appLogRepo := &runtimeAppLogRecorderRepo{}
	deps := runtimeCoreDeps{
		newAccessLogRepository: func(db *sql.DB) (httpx.AccessLogRepository, error) {
			if db == nil {
				t.Fatal("expected runtime assembly to pass shared sql db into access log repository factory")
			}
			return recorderRepo, nil
		},
		newAppLogRepository: func(db *sql.DB) (logger.AppLogRepository, error) {
			if db == nil {
				t.Fatal("expected runtime assembly to pass shared sql db into app log repository factory")
			}
			return appLogRepo, nil
		},
		openRedisClient: func(context.Context, config.RedisConfig) (*redis.Client, error) {
			return redis.NewClient(&redis.Options{Addr: "127.0.0.1:6379"}), nil
		},
	}

	runtime, err := newRuntimeCoreWithDeps(&config.Config{
		App: config.AppConfig{Name: "graft", Env: "test"},
		HTTP: config.HTTPConfig{
			Addr: "127.0.0.1:0",
		},
		Database: config.DatabaseConfig{
			Driver: "postgres",
			URL:    "postgres://graft@localhost:5432/graft?sslmode=disable",
		},
		Redis: config.RedisConfig{
			Addr: "127.0.0.1:6379",
		},
		Log: config.LogConfig{
			Level:           "info",
			Format:          config.LogFormatAuto,
			Color:           config.LogColorAuto,
			AppLogPersist:   true,
			AppLogRetention: 3 * 24 * time.Hour,
		},
		Runtime: config.RuntimeConfig{
			GinMode: config.GinModeAuto,
		},
		I18n: config.I18nConfig{
			DefaultLocale:    "zh-CN",
			FallbackLocale:   "zh-CN",
			SupportedLocales: []string{"zh-CN", "en-US"},
		},
	}, deps)
	if err != nil {
		t.Fatalf("new runtime core: %v", err)
	}
	t.Cleanup(func() {
		_ = runtime.closeCoreResources()
	})

	runtime.server.Engine().GET("/access-log-check", func(ctx *gin.Context) {
		ctx.Status(http.StatusNoContent)
	})

	request := httptest.NewRequest(http.MethodGet, "/access-log-check", nil)
	request.RemoteAddr = "203.0.113.10:3456"
	recorder := httptest.NewRecorder()
	runtime.server.Engine().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, recorder.Code)
	}
	if len(recorderRepo.created) != 1 {
		t.Fatalf("expected runtime server to persist one access log, got %d", len(recorderRepo.created))
	}
	if recorderRepo.created[0].Path != "/access-log-check" || recorderRepo.created[0].Route != "/access-log-check" {
		t.Fatalf("expected canonical access-log route fields, got %#v", recorderRepo.created[0])
	}
	if runtime.appLogRepository != appLogRepo {
		t.Fatal("expected runtime to retain logger-owned app log repository")
	}
}

func TestRegisterCoreDashboardWidgetsIncludesAccessLogSystemCapability(t *testing.T) {
	repo := &runtimeAccessLogRecorderRepo{}
	runtime := &Runtime{
		config:            &config.Config{},
		server:            httpx.NewServer(zap.NewNop(), repo),
		dashboardRegistry: dashboard.NewRegistry(),
	}

	if err := runtime.registerCoreDashboardWidgets(); err != nil {
		t.Fatalf("register core dashboard widgets: %v", err)
	}

	moduleRuntimeWidget, ok := runtime.dashboardRegistry.Get("core.module-runtime-health")
	if !ok {
		t.Fatalf("expected module runtime health widget to be registered")
	}
	if len(moduleRuntimeWidget.RequiredPermissions) != 1 || moduleRuntimeWidget.RequiredPermissions[0] != moduleruntime.PermissionRead {
		t.Fatalf("unexpected module runtime permissions: %#v", moduleRuntimeWidget.RequiredPermissions)
	}

	accessLogWidget, ok := runtime.dashboardRegistry.Get(httpx.AccessLogDashboardWidgetID)
	if !ok {
		t.Fatalf("expected access-log dashboard widget to be registered")
	}
	if accessLogWidget.ModuleKey != httpx.AccessLogDashboardModuleKey() {
		t.Fatalf("expected access-log system owner, got %q", accessLogWidget.ModuleKey)
	}
	if accessLogWidget.Type != dashboard.WidgetTypeAlertList {
		t.Fatalf("expected access-log alert-list widget, got %q", accessLogWidget.Type)
	}
	if len(accessLogWidget.RequiredPermissions) != 1 || accessLogWidget.RequiredPermissions[0] != httpx.AccessLogReadPermission {
		t.Fatalf("unexpected access-log permissions: %#v", accessLogWidget.RequiredPermissions)
	}
}

func TestAccessLogIsNotRegisteredAsModule(t *testing.T) {
	for _, spec := range moduleregistry.ModuleSpecs() {
		if spec.Name() == "access-log" || spec.Name() == httpx.AccessLogDashboardModuleKey() {
			t.Fatalf("access-log must remain a core/httpx system capability, got module spec %q", spec.Name())
		}
	}
}

func TestApplyGinModeUsesResolvedRuntimeConfig(t *testing.T) {
	originalMode := gin.Mode()
	t.Cleanup(func() {
		gin.SetMode(originalMode)
	})

	applyGinMode(&config.Config{
		App: config.AppConfig{Env: "local"},
		Runtime: config.RuntimeConfig{
			GinMode: config.GinModeAuto,
		},
	})
	if got := gin.Mode(); got != gin.DebugMode {
		t.Fatalf("expected local auto gin mode %q, got %q", gin.DebugMode, got)
	}

	applyGinMode(&config.Config{
		App: config.AppConfig{Env: "production"},
		Runtime: config.RuntimeConfig{
			GinMode: config.GinModeAuto,
		},
	})
	if got := gin.Mode(); got != gin.ReleaseMode {
		t.Fatalf("expected production auto gin mode %q, got %q", gin.ReleaseMode, got)
	}

	applyGinMode(&config.Config{
		App: config.AppConfig{Env: "production"},
		Runtime: config.RuntimeConfig{
			GinMode: config.GinModeTest,
		},
	})
	if got := gin.Mode(); got != gin.TestMode {
		t.Fatalf("expected explicit test gin mode %q, got %q", gin.TestMode, got)
	}
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

func assertAppLoggerRegistered(t *testing.T, resolver container.Resolver) {
	t.Helper()

	resolved, err := resolver.Resolve((*logger.AppLogger)(nil))
	if err != nil {
		t.Fatalf("resolve app logger: %v", err)
	}

	appLogger, ok := resolved.(logger.AppLogger)
	if !ok {
		t.Fatalf("expected app logger, got %T", resolved)
	}
	if appLogger == nil {
		t.Fatal("expected non-nil app logger")
	}
}

func assertServiceKeyNotRegistered(t *testing.T, resolver container.Resolver, key any, name string) {
	t.Helper()

	_, err := resolver.Resolve(key)
	if err == nil {
		t.Fatalf("expected runtime services to omit %s", name)
	}
	if !errors.Is(err, container.ErrServiceNotRegistered) {
		t.Fatalf("expected %s to be unregistered, got %v", name, err)
	}
}

// TestRunPassesEventBusIntoModuleContext 验证 Runtime 在 Register 与 Boot
// 阶段向模块注入同一个事件总线实例，避免模块各自持有漂移的协作边界。
func TestRunPassesEventBusIntoModuleContext(t *testing.T) {
	gin.SetMode(gin.TestMode)

	runCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	recorder := &eventBusRecorderModule{cancelOnBoot: cancel}
	manager := module.NewManager()
	if err := manager.RegisterModule(mustDescribeRuntimeTestModule(module.Spec{ID: "eventbus-recorder"}, recorder)); err != nil {
		t.Fatalf("register module: %v", err)
	}

	runtimeEventBus := eventbus.New(zap.NewNop())
	runtime := &Runtime{
		config: &config.Config{
			HTTP: config.HTTPConfig{Addr: "127.0.0.1:0"},
		},
		logger:             zap.NewNop(),
		i18n:               i18n.MustNew(config.I18nConfig{DefaultLocale: "zh-CN", FallbackLocale: "zh-CN", SupportedLocales: []string{"zh-CN"}}),
		server:             httpx.NewServer(zap.NewNop()),
		eventBus:           runtimeEventBus,
		services:           container.New(),
		menuRegistry:       menu.NewRegistry(),
		permissionRegistry: permission.NewRegistry(),
		cronRegistry:       cronx.NewRegistry(),
		moduleManager:      manager,
	}

	if err := runtime.Run(runCtx); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected canceled runtime lifecycle, got %v", err)
	}
	if recorder.registerEventBus != runtimeEventBus {
		t.Fatal("expected register phase to receive runtime event bus instance")
	}
	if recorder.bootEventBus != runtimeEventBus {
		t.Fatal("expected boot phase to receive runtime event bus instance")
	}
}

// TestRunPassesLifecycleContextIntoModulePhases 验证 Runtime 会在模块生命周期内
// 注入显式上下文，并在 Shutdown 阶段切换到独立的有界关闭上下文。
func TestRunPassesLifecycleContextIntoModulePhases(t *testing.T) {
	gin.SetMode(gin.TestMode)

	runCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	recorder := &lifecycleContextRecorderModule{cancelOnBoot: cancel}
	manager := module.NewManager()
	if err := manager.RegisterModule(mustDescribeRuntimeTestModule(module.Spec{ID: "lifecycle-context-recorder"}, recorder)); err != nil {
		t.Fatalf("register module: %v", err)
	}

	runtime := &Runtime{
		config: &config.Config{
			HTTP: config.HTTPConfig{Addr: "127.0.0.1:0"},
		},
		logger:             zap.NewNop(),
		i18n:               i18n.MustNew(config.I18nConfig{DefaultLocale: "zh-CN", FallbackLocale: "zh-CN", SupportedLocales: []string{"zh-CN"}}),
		server:             httpx.NewServer(zap.NewNop()),
		eventBus:           eventbus.New(zap.NewNop()),
		services:           container.New(),
		menuRegistry:       menu.NewRegistry(),
		permissionRegistry: permission.NewRegistry(),
		cronRegistry:       cronx.NewRegistry(),
		moduleManager:      manager,
	}

	if err := runtime.Run(runCtx); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected canceled runtime lifecycle, got %v", err)
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

// TestRunStopsBeforeBootWhenLifecycleContextAlreadyCanceled 验证启动上下文已经取消时，
// Runtime 不会继续进入会访问数据库或外部资源的模块 Boot 阶段。
func TestRunStopsBeforeBootWhenLifecycleContextAlreadyCanceled(t *testing.T) {
	gin.SetMode(gin.TestMode)

	booted := false
	manager := module.NewManager()
	if err := manager.RegisterModule(mustDescribeRuntimeTestModule(
		module.Spec{ID: "canceled-boot-recorder"},
		bootRecorderModule{booted: &booted},
	)); err != nil {
		t.Fatalf("register module: %v", err)
	}

	runtime := &Runtime{
		config: &config.Config{
			HTTP: config.HTTPConfig{Addr: "127.0.0.1:0"},
		},
		logger:             zap.NewNop(),
		i18n:               i18n.MustNew(config.I18nConfig{DefaultLocale: "zh-CN", FallbackLocale: "zh-CN", SupportedLocales: []string{"zh-CN"}}),
		server:             httpx.NewServer(zap.NewNop()),
		eventBus:           eventbus.New(zap.NewNop()),
		services:           container.New(),
		menuRegistry:       menu.NewRegistry(),
		permissionRegistry: permission.NewRegistry(),
		cronRegistry:       cronx.NewRegistry(),
		moduleManager:      manager,
	}

	runCtx, cancel := context.WithCancel(context.Background())
	cancel()

	err := runtime.Run(runCtx)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected canceled lifecycle error, got %v", err)
	}
	if booted {
		t.Fatal("expected canceled lifecycle to stop before module boot")
	}
}

func TestRunFreezesI18nRegistryAfterRegisterBeforeBoot(t *testing.T) {
	gin.SetMode(gin.TestMode)

	runCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	recorder := &i18nFreezeRecorderModule{cancelOnBoot: cancel}
	manager := module.NewManager()
	if err := manager.RegisterModule(mustDescribeRuntimeTestModule(module.Spec{ID: "i18n-freeze-recorder"}, recorder)); err != nil {
		t.Fatalf("register module: %v", err)
	}

	localizer := i18n.MustNew(config.I18nConfig{
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
		server:             httpx.NewServer(zap.NewNop()),
		eventBus:           eventbus.New(zap.NewNop()),
		services:           container.New(),
		menuRegistry:       menu.NewRegistry(),
		permissionRegistry: permission.NewRegistry(),
		cronRegistry:       cronx.NewRegistry(),
		moduleManager:      manager,
	}

	if err := runtime.Run(runCtx); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected canceled runtime lifecycle, got %v", err)
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
		Namespace: "test-module",
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
		i18n:               i18n.MustNew(config.I18nConfig{DefaultLocale: "zh-CN", FallbackLocale: "en-US", SupportedLocales: []string{"zh-CN", "en-US"}}),
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

func TestRegisterCoreRoutesServesOpenAPIDocsWhenEnabled(t *testing.T) {
	gin.SetMode(gin.TestMode)

	docsAssets, err := loadOpenAPIDocsAssets()
	if err != nil {
		t.Fatalf("load openapi docs assets: %v", err)
	}

	runtime := &Runtime{
		config:             &config.Config{Docs: config.DocsConfig{Enabled: true}},
		i18n:               i18n.MustNew(config.I18nConfig{DefaultLocale: "zh-CN", FallbackLocale: "en-US", SupportedLocales: []string{"zh-CN", "en-US"}}),
		menuRegistry:       menu.NewRegistry(),
		permissionRegistry: permission.NewRegistry(),
		cronRegistry:       cronx.NewRegistry(),
		openapiDocs:        docsAssets,
	}

	engine := gin.New()
	runtime.registerCoreRoutes(engine)

	assertOpenAPIJSONResponse(t, engine)
	assertOpenAPIYAMLResponse(t, engine)
	assertDocsHTMLResponse(t, engine)
}

func TestRegisterCoreRoutesSkipsOpenAPIDocsWhenDisabled(t *testing.T) {
	gin.SetMode(gin.TestMode)

	runtime := &Runtime{
		config:             &config.Config{Docs: config.DocsConfig{Enabled: false}},
		i18n:               i18n.MustNew(config.I18nConfig{DefaultLocale: "zh-CN", FallbackLocale: "en-US", SupportedLocales: []string{"zh-CN", "en-US"}}),
		menuRegistry:       menu.NewRegistry(),
		permissionRegistry: permission.NewRegistry(),
		cronRegistry:       cronx.NewRegistry(),
	}

	engine := gin.New()
	runtime.registerCoreRoutes(engine)

	for _, path := range []string{openapiJSONPath, openapiYAMLPath, openapiDocsPath} {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, path, nil)
		engine.ServeHTTP(recorder, request)

		if recorder.Code != http.StatusNotFound {
			t.Fatalf("%s: expected status %d, got %d", path, http.StatusNotFound, recorder.Code)
		}
	}
}

func TestNewRuntimeCoreRegistersAccessLogRetentionJobWithoutRunningCleanup(t *testing.T) {
	recorderRepo := &runtimeAccessLogRecorderRepo{}
	appLogRepo := &runtimeAppLogRecorderRepo{}
	deps := runtimeCoreDeps{
		newAccessLogRepository: func(_ *sql.DB) (httpx.AccessLogRepository, error) {
			return recorderRepo, nil
		},
		newAppLogRepository: func(_ *sql.DB) (logger.AppLogRepository, error) {
			return appLogRepo, nil
		},
		openRedisClient: func(context.Context, config.RedisConfig) (*redis.Client, error) {
			return redis.NewClient(&redis.Options{Addr: "127.0.0.1:6379"}), nil
		},
	}

	runtime, err := newRuntimeCoreWithDeps(&config.Config{
		App:   config.AppConfig{Name: "graft", Env: "test"},
		HTTP:  config.HTTPConfig{Addr: ":8080"},
		HTTPX: config.HTTPXConfig{AccessLogRetention: 3 * 24 * time.Hour},
		Database: config.DatabaseConfig{
			Driver: "postgres",
			URL:    "postgres://graft@localhost:5432/graft?sslmode=disable",
		},
		Redis: config.RedisConfig{Addr: "localhost:6379"},
		Log:   config.LogConfig{Level: "info", AppLogPersist: true, AppLogRetention: 3 * 24 * time.Hour},
		I18n:  config.I18nConfig{DefaultLocale: "zh-CN", FallbackLocale: "zh-CN", SupportedLocales: []string{"zh-CN", "en-US"}},
		Auth: config.AuthConfig{
			AccessTokenTTL:        time.Minute,
			RefreshTokenTTL:       time.Hour,
			JWTSecret:             "secret",
			RefreshCookieName:     "refresh",
			RefreshCookiePath:     "/",
			RefreshCookieSameSite: "lax",
		},
	}, deps)
	if err != nil {
		t.Fatalf("new runtime core: %v", err)
	}
	t.Cleanup(func() {
		_ = runtime.closeCoreResources()
	})

	if err := runtime.registerAccessLogRetentionJob(); err != nil {
		t.Fatalf("register retention job: %v", err)
	}
	if err := runtime.registerAppLogRetentionJob(); err != nil {
		t.Fatalf("register app-log retention job: %v", err)
	}

	items := runtime.cronRegistry.Items()
	if len(items) != 2 {
		t.Fatalf("expected two registered retention jobs, got %d", len(items))
	}
	definitions := runtime.configRegistry.Items()
	if len(definitions) != 2 {
		t.Fatalf("expected two registered config definitions, got %d", len(definitions))
	}
	if items[0].Name != "httpx.access-log-retention-cleanup" {
		t.Fatalf("expected retention job name, got %q", items[0].Name)
	}
	if items[1].Name != "logger.app-log-retention-cleanup" {
		t.Fatalf("expected app-log retention job name, got %q", items[1].Name)
	}
	if definitions[0].Key != "httpx.access-log-retention-cleanup" || definitions[1].Key != "logger.app-log-retention-cleanup" {
		t.Fatalf("unexpected retention config definitions: %#v", definitions)
	}
	if len(recorderRepo.deleted) != 0 {
		t.Fatalf("expected startup registration to avoid cleanup execution, got %d deletions", len(recorderRepo.deleted))
	}
	if len(appLogRepo.deleted) != 0 {
		t.Fatalf("expected startup registration to avoid app-log cleanup execution, got %d deletions", len(appLogRepo.deleted))
	}
}

func assertOpenAPIJSONResponse(t *testing.T, engine *gin.Engine) {
	t.Helper()

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, openapiJSONPath, nil)
	engine.ServeHTTP(recorder, request)

	assertResponseStatusAndType(t, recorder, openapiJSONPath, http.StatusOK, "application/json; charset=utf-8")

	jsonBody := recorder.Body.String()
	for _, unexpected := range []string{"./paths/", "./components/"} {
		if strings.Contains(jsonBody, unexpected) {
			t.Fatalf("%s: expected bundled json to omit external ref fragment %q", openapiJSONPath, unexpected)
		}
	}

	var document map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &document); err != nil {
		t.Fatalf("%s: decode response: %v", openapiJSONPath, err)
	}

	paths := mustDecodeJSONObject(t, document["paths"], "paths")
	loginPath := mustDecodeJSONObject(t, paths["/api/auth/login"], "/api/auth/login")
	if _, ok := loginPath["post"]; !ok {
		t.Fatalf("%s: expected bundled json to expose POST /api/auth/login", openapiJSONPath)
	}
}

func assertOpenAPIYAMLResponse(t *testing.T, engine *gin.Engine) {
	t.Helper()

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, openapiYAMLPath, nil)
	engine.ServeHTTP(recorder, request)

	assertResponseStatusAndType(t, recorder, openapiYAMLPath, http.StatusOK, "application/yaml; charset=utf-8")
	if !strings.Contains(recorder.Body.String(), "openapi: 3.1.0") {
		t.Fatalf("%s: expected body to contain root spec marker", openapiYAMLPath)
	}
}

func assertDocsHTMLResponse(t *testing.T, engine *gin.Engine) {
	t.Helper()

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, openapiDocsPath, nil)
	engine.ServeHTTP(recorder, request)

	assertResponseStatusAndType(t, recorder, openapiDocsPath, http.StatusOK, "text/html; charset=utf-8")
	if !strings.Contains(recorder.Body.String(), `data-url="/openapi.json"`) {
		t.Fatalf("%s: expected body to contain Scalar spec url", openapiDocsPath)
	}
	if !strings.Contains(recorder.Body.String(), `src="`+scalarDocsScriptURL+`"`) {
		t.Fatalf("%s: expected body to pin the Scalar docs script url", openapiDocsPath)
	}
	if !strings.Contains(recorder.Body.String(), `integrity="`+scalarDocsScriptIntegrity+`"`) {
		t.Fatalf("%s: expected body to contain the pinned Scalar docs integrity", openapiDocsPath)
	}
}

func assertResponseStatusAndType(t *testing.T, recorder *httptest.ResponseRecorder, path string, status int, contentType string) {
	t.Helper()

	if recorder.Code != status {
		t.Fatalf("%s: expected status %d, got %d", path, status, recorder.Code)
	}
	if actual := recorder.Header().Get("Content-Type"); actual != contentType {
		t.Fatalf("%s: expected content type %q, got %q", path, contentType, actual)
	}
}

func mustDecodeJSONObject(t *testing.T, value any, name string) map[string]any {
	t.Helper()

	object, ok := value.(map[string]any)
	if !ok {
		t.Fatalf("expected %s to decode as object", name)
	}
	return object
}
