package app

import (
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
	"graft/server/internal/i18n"
	"graft/server/internal/menu"
	"graft/server/internal/permission"
	"graft/server/internal/plugin"
	"graft/server/internal/store"
)

type shutdownRecorderPlugin struct {
	name        string
	shutdownLog *[]string
	err         error
}

func (p shutdownRecorderPlugin) Name() string { return p.name }

func (p shutdownRecorderPlugin) Version() string { return "test" }

func (p shutdownRecorderPlugin) DependsOn() []string { return nil }

func (p shutdownRecorderPlugin) Register(ctx *plugin.Context) error { return nil }

func (p shutdownRecorderPlugin) Boot(ctx *plugin.Context) error { return nil }

func (p shutdownRecorderPlugin) Shutdown(ctx *plugin.Context) error {
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

type runtimeTestStoreFactory struct{}

func (runtimeTestStoreFactory) Users() store.UserRepository {
	return nil
}

// TestRegisterCoreServicesExposesRuntimeSingletons 验证 core 装配会把配置、
// store factory 与 Redis 客户端注册到运行时容器中。
func TestRegisterCoreServicesExposesRuntimeSingletons(t *testing.T) {
	redisClient := redis.NewClient(&redis.Options{Addr: "127.0.0.1:6379"})
	t.Cleanup(func() {
		_ = redisClient.Close()
	})
	runtimeLogger := zap.NewNop()

	cfg := &config.Config{
		App: config.AppConfig{Name: "graft", Env: "test"},
		HTTP: config.HTTPConfig{
			Addr: ":8080",
		},
		Database: config.DatabaseConfig{
			Driver: "postgres",
			URL:    "postgres://graft:graft@localhost:5432/graft?sslmode=disable",
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
	stores := &runtimeTestStoreFactory{}
	runtime := &Runtime{
		config:   cfg,
		logger:   runtimeLogger,
		i18n:     localizer,
		redis:    redisClient,
		services: container.New(),
		stores:   stores,
	}

	if err := runtime.registerCoreServices(); err != nil {
		t.Fatalf("register core services: %v", err)
	}

	resolvedConfig, err := runtime.services.Resolve((*config.Config)(nil))
	if err != nil {
		t.Fatalf("resolve config: %v", err)
	}
	if resolvedConfig != cfg {
		t.Fatalf("expected resolved config to reuse runtime pointer")
	}

	resolvedLogger, err := runtime.services.Resolve((*zap.Logger)(nil))
	if err != nil {
		t.Fatalf("resolve logger: %v", err)
	}
	if resolvedLogger != runtimeLogger {
		t.Fatal("expected resolved logger to reuse runtime instance")
	}

	resolvedI18n, err := runtime.services.Resolve((*i18n.Service)(nil))
	if err != nil {
		t.Fatalf("resolve i18n service: %v", err)
	}
	if resolvedI18n != localizer {
		t.Fatal("expected resolved i18n service to reuse runtime instance")
	}

	resolvedStores, err := runtime.services.Resolve((*store.Factory)(nil))
	if err != nil {
		t.Fatalf("resolve store factory: %v", err)
	}
	if resolvedStores.(store.Factory) != stores {
		t.Fatal("expected resolved store factory to reuse runtime instance")
	}

	resolvedRedis, err := runtime.services.Resolve((*redis.Client)(nil))
	if err != nil {
		t.Fatalf("resolve redis client: %v", err)
	}
	if resolvedRedis != redisClient {
		t.Fatal("expected resolved redis client to reuse runtime instance")
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
