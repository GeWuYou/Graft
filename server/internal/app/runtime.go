// Package app 组装 Graft 的显式运行时外壳。
package app

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"graft/server/internal/buildinfo"
	"graft/server/internal/cachex"
	cachebackend "graft/server/internal/cachex/backend"
	"graft/server/internal/config"
	"graft/server/internal/configregistry"
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
	moduleruntimelocales "graft/server/internal/moduleruntime/locales"
	"graft/server/internal/permission"
	"graft/server/internal/realtime"
	"graft/server/internal/redisx"
)

const moduleShutdownTimeout = 5 * time.Second
const appRuntimeLogComponent = "internal.app.runtime"
const coreServiceRegistrationCapacity = 12
const (
	coreModuleRuntimeHealthWidgetOrder = 10
	moduleRuntimeHealthTitleKey        = "dashboard.widget.moduleRuntimeHealth.title"
	moduleRuntimeHealthDescriptionKey  = "dashboard.widget.moduleRuntimeHealth.description"
	moduleRuntimeHealthSummaryKey      = "dashboard.widget.moduleRuntimeHealth.summary"
	openapiYAMLPath                    = "/openapi.yaml"
)

type runtimeCoreDeps struct {
	newAccessLogRepository func(*sql.DB) (httpx.AccessLogRepository, error)
	newAppLogRepository    func(*sql.DB) (logger.AppLogRepository, error)
	openRedisClient        func(context.Context, config.RedisConfig) (*redis.Client, error)
}

var defaultRuntimeCoreDeps = runtimeCoreDeps{
	newAccessLogRepository: httpx.NewAccessLogRepository,
	newAppLogRepository:    logger.NewAppLogRepository,
	openRedisClient:        redisx.Open,
}

var runtimeEmbeddedLocaleResources = func() []i18n.EmbeddedLocaleResource {
	resources := moduleregistry.EmbeddedLocaleResources()

	moduleRuntimeResources, err := moduleruntimelocales.EmbeddedLocaleResources()
	if err != nil {
		panic(fmt.Sprintf("load module-runtime embedded locale resources: %v", err))
	}

	return append(resources, moduleRuntimeResources...)
}

// Runtime 持有 MVP 运行时的核心资源与模块生命周期执行入口。
//
// Runtime 把配置、数据库、Redis、HTTP 服务、注册中心和模块管理器集中
// 到一个显式对象中，方便在失败路径和正常关闭路径统一回收资源。
//
// Runtime 本身不承载业务能力；它只负责 core 资源装配、模块生命周期编排
// 和进程级关闭顺序，避免模块把运行时控制逻辑反向塞回 core。
type Runtime struct {
	config                    *config.Config
	logger                    *zap.Logger
	i18n                      *i18n.Service
	localeResourcesRegistered bool
	database                  *database.Resources
	redis                     *redis.Client
	cacheManager              *cachex.Manager
	server                    *httpx.Server
	openapiDocs               *openAPIDocsAssets
	eventBus                  eventbus.Bus
	realtimeHub               realtime.Hub
	realtimeTopicIssuers      realtime.TopicIssuerRegistry
	services                  *container.Container
	menuRegistry              *menu.Registry
	permissionRegistry        *permission.Registry
	cronRegistry              *cronx.Registry
	configRegistry            *configregistry.Registry
	dashboardRegistry         *dashboard.Registry
	moduleManager             *module.Manager
	runtimeMetadata           module.RuntimeMetadata
	appLogRepository          logger.AppLogRepository
}

// NewRuntime 使用给定模块构造显式的 MVP 运行时外壳。
//
// 参数：
//   - modules: 需要接入当前进程的模块集合；这里只注册模块元数据，不执行模块生命周期。
//
// 返回：
//   - *Runtime: 已完成 core 资源装配和模块登记的运行时对象。
//
// NewRuntime 加载配置并构建运行时实例，完成核心资源、服务、路由和模块注册。
// 任一步骤失败时返回错误，部分失败场景下会尽力回收已创建的核心资源。
func NewRuntime() (*Runtime, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	runtime, err := newRuntimeCore(cfg)
	if err != nil {
		return nil, err
	}

	if err := runtime.loadOptionalDocsAssets(); err != nil {
		_ = runtime.closeCoreResources()
		return nil, err
	}

	if err := runtime.registerCoreServices(); err != nil {
		_ = runtime.closeCoreResources()
		return nil, err
	}

	if err := runtime.registerCoreConfigDefinitions(); err != nil {
		return nil, err
	}

	if err := runtime.registerRetentionJobs(); err != nil {
		return nil, err
	}

	if err := runtime.registerCoreRoutes(runtime.server.Engine()); err != nil {
		_ = runtime.closeCoreResources()
		return nil, err
	}

	if err := runtime.registerRuntimeModules(cfg.Modules.Enabled); err != nil {
		return nil, err
	}

	return runtime, nil
}

func (r *Runtime) registerRetentionJobs() error {
	if err := r.registerAccessLogRetentionJob(); err != nil {
		_ = r.closeCoreResources()
		return fmt.Errorf("register access-log retention job: %w", err)
	}
	if err := r.registerAppLogRetentionJob(); err != nil {
		_ = r.closeCoreResources()
		return fmt.Errorf("register app-log retention job: %w", err)
	}
	return nil
}

func (r *Runtime) registerCoreConfigDefinitions() error {
	if r == nil {
		return errors.New("runtime is unavailable")
	}
	if err := dashboard.RegisterQuickActionsConfigMessages(r.i18n); err != nil {
		_ = r.closeCoreResources()
		return fmt.Errorf("register dashboard quick-actions config messages: %w", err)
	}
	if err := dashboard.RegisterQuickActionsConfigDefinitions(r.configRegistry); err != nil {
		_ = r.closeCoreResources()
		return fmt.Errorf("register dashboard quick-actions config definitions: %w", err)
	}
	return nil
}

func (r *Runtime) registerRuntimeModules(enabledModules []string) error {
	orderedDescriptors, err := moduleregistry.FilteredOrderedModuleSpecs(enabledModules)
	if err != nil {
		_ = r.closeCoreResources()
		return fmt.Errorf("order runtime module descriptors: %w", err)
	}
	r.runtimeMetadata = module.NewRuntimeMetadata(orderedDescriptors, buildinfo.Current())

	modules, err := moduleregistry.BuildModules(module.BuildContext{Services: r.services}, enabledModules)
	if err != nil {
		_ = r.closeCoreResources()
		return fmt.Errorf("build runtime modules: %w", err)
	}

	for _, current := range modules {
		if err := r.moduleManager.RegisterModule(current); err != nil {
			_ = r.closeCoreResources()
			return err
		}
	}

	return nil
}

func newRuntimeCore(cfg *config.Config) (*Runtime, error) {
	return newRuntimeCoreWithDeps(cfg, defaultRuntimeCoreDeps)
}

// newRuntimeCoreWithDeps 初始化核心运行时资源，并返回已完成配置且预注册了本地化资源的 Runtime 实例。
// 它会按顺序创建日志、数据库、Redis、i18n、仓储、缓存管理器、HTTP 服务器以及各类注册表，并在任一步失败时回收已创建资源。
func newRuntimeCoreWithDeps(cfg *config.Config, deps runtimeCoreDeps) (*Runtime, error) {
	deps = normalizeRuntimeCoreDeps(deps)
	applyGinMode(cfg)

	runtimeLogger, err := logger.New(cfg)
	if err != nil {
		return nil, fmt.Errorf("create logger: %w", err)
	}

	databaseResources, err := database.Open(cfg.Database)
	if err != nil {
		_ = logger.Close(runtimeLogger)
		return nil, fmt.Errorf("open database resources: %w", err)
	}

	redisClient, err := deps.openRedisClient(context.Background(), cfg.Redis)
	if err != nil {
		_ = database.Close(databaseResources)
		_ = logger.Close(runtimeLogger)
		return nil, fmt.Errorf("open redis client: %w", err)
	}

	localizer, err := i18n.New(cfg.I18n)
	if err != nil {
		_ = redisClient.Close()
		_ = database.Close(databaseResources)
		_ = logger.Close(runtimeLogger)
		return nil, fmt.Errorf("create i18n service: %w", err)
	}

	accessLogRepo, err := deps.newAccessLogRepository(databaseResources.SQL)
	if err != nil {
		_ = redisClient.Close()
		_ = database.Close(databaseResources)
		_ = logger.Close(runtimeLogger)
		return nil, fmt.Errorf("create access log repository: %w", err)
	}

	appLogRepo, err := newOptionalAppLogRepository(cfg, deps, databaseResources.SQL)
	if err != nil {
		_ = redisClient.Close()
		_ = database.Close(databaseResources)
		_ = logger.Close(runtimeLogger)
		return nil, err
	}

	cacheManager, err := newRuntimeCacheManager(cfg, redisClient)
	if err != nil {
		_ = redisClient.Close()
		_ = database.Close(databaseResources)
		_ = logger.Close(runtimeLogger)
		return nil, fmt.Errorf("create cache manager: %w", err)
	}

	runtime := &Runtime{
		config:       cfg,
		logger:       runtimeLogger,
		i18n:         localizer,
		database:     databaseResources,
		redis:        redisClient,
		cacheManager: cacheManager,
		server: httpx.NewServerWithOptions(runtimeLogger, httpx.ServerOptions{
			AccessLog: httpx.AccessLogOptions{
				ConsolePolicy: config.ResolveAccessLogConsolePolicy(cfg.App.Env, cfg.HTTPX.AccessLogConsole),
				SlowThreshold: time.Duration(cfg.HTTPX.AccessLogSlowThresholdMS) * time.Millisecond,
			},
		}, accessLogRepo),
		eventBus:             eventbus.New(runtimeLogger),
		realtimeHub:          realtime.NewHub(),
		realtimeTopicIssuers: realtime.NewTopicIssuerRegistry(),
		services:             container.New(),
		menuRegistry:         menu.NewRegistry(),
		permissionRegistry:   permission.NewRegistry(),
		cronRegistry:         cronx.NewRegistry(),
		configRegistry:       configregistry.NewRegistry(),
		dashboardRegistry:    dashboard.NewRegistry(),
		moduleManager:        module.NewManager(),
		appLogRepository:     appLogRepo,
	}
	if err := runtime.preregisterOwnerLocaleResources(); err != nil {
		_ = runtime.closeCoreResources()
		return nil, err
	}

	return runtime, nil
}

// normalizeRuntimeCoreDeps 为缺失的构造函数和打开函数填充默认实现。
func normalizeRuntimeCoreDeps(deps runtimeCoreDeps) runtimeCoreDeps {
	if deps.newAccessLogRepository == nil {
		deps.newAccessLogRepository = httpx.NewAccessLogRepository
	}
	if deps.newAppLogRepository == nil {
		deps.newAppLogRepository = logger.NewAppLogRepository
	}
	if deps.openRedisClient == nil {
		deps.openRedisClient = redisx.Open
	}
	return deps
}

// newRuntimeCacheManager 创建一个以 Redis 为后端的缓存管理器，并使用应用名称作为命名空间，默认回落为 "graft"。
// 缓存键前缀为命名空间加上 ":cache"。
func newRuntimeCacheManager(cfg *config.Config, client *redis.Client) (*cachex.Manager, error) {
	namespace := "graft"
	if cfg != nil {
		appName := strings.TrimSpace(cfg.App.Name)
		if appName != "" {
			namespace = appName
		}
	}

	redisBackend, err := cachebackend.NewRedis(client, cachebackend.RedisOptions{
		Prefix: namespace + ":cache",
	})
	if err != nil {
		return nil, err
	}

	return cachex.NewManager(cachex.ManagerOptions{
		Backend:   redisBackend,
		Metrics:   cachex.NopMetrics(),
		Namespace: namespace,
	})
}

// applyGinMode 根据配置设置 Gin 运行模式。
func applyGinMode(cfg *config.Config) {
	appEnv := ""
	mode := config.GinModeAuto
	if cfg != nil {
		appEnv = cfg.App.Env
		mode = cfg.Runtime.GinMode
	}
	gin.SetMode(string(config.ResolveGinMode(appEnv, mode)))
}

func newOptionalAppLogRepository(
	cfg *config.Config,
	deps runtimeCoreDeps,
	db *sql.DB,
) (logger.AppLogRepository, error) {
	if cfg == nil || !cfg.Log.AppLogPersist {
		return nil, nil
	}

	appLogRepo, err := deps.newAppLogRepository(db)
	if err != nil {
		return nil, fmt.Errorf("create app log repository: %w", err)
	}
	return appLogRepo, nil
}

// Run 先执行模块注册与启动，再启动 HTTP 服务。
//
// 如果任一阶段失败，Run 会按已启动的实际范围反向释放模块与核心资源，
// 避免把半初始化状态泄漏到调用方。
//
// 参数：
//   - runCtx: 绑定当前进程运行期的上下文；取消后会触发 HTTP 服务停止，并继续进入模块与 core 资源清理。
//
// 返回：
//   - error: 返回注册、启动、监听、关闭阶段的首个失败，并按需要聚合模块关闭或 core 资源回收错误。
func (r *Runtime) Run(runCtx context.Context) error {
	moduleCtx := r.newModuleContext(runCtx)

	ordered, err := r.moduleManager.Ordered()
	if err != nil {
		return err
	}

	booted, err := r.prepareModules(runCtx, moduleCtx, ordered)
	if err != nil {
		return err
	}
	r.appLogger().Info(runCtx, "app runtime boot completed",
		logger.StringField(logger.FieldOperation, "runtime_boot"),
		logger.IntField("modules", len(booted)),
	)

	return r.runServerAndShutdown(runCtx, moduleCtx, booted)
}

func (r *Runtime) prepareModules(
	runCtx context.Context,
	moduleCtx *module.Context,
	ordered []module.RuntimeModule,
) ([]module.RuntimeModule, error) {
	booted := make([]module.RuntimeModule, 0, len(ordered))
	if err := r.ensureLifecycleActive(runCtx, moduleCtx, booted); err != nil {
		return nil, err
	}
	if err := r.assertOwnerLocaleResourcesRegistered(moduleCtx, booted); err != nil {
		return nil, err
	}
	if err := r.registerModules(moduleCtx, ordered, booted); err != nil {
		return nil, err
	}
	if err := r.prepareCoreRegistries(runCtx, moduleCtx, booted); err != nil {
		return nil, err
	}
	return r.bootModules(moduleCtx, ordered, booted)
}

func (r *Runtime) prepareCoreRegistries(
	runCtx context.Context,
	moduleCtx *module.Context,
	booted []module.RuntimeModule,
) error {
	if err := r.ensureLifecycleActive(runCtx, moduleCtx, booted); err != nil {
		return err
	}
	if err := r.registerCoreAuthenticatedRoutes(); err != nil {
		return r.cleanupAfterFailure(moduleCtx, booted, err)
	}
	if err := r.ensureLifecycleActive(runCtx, moduleCtx, booted); err != nil {
		return err
	}
	if err := r.i18n.Freeze(); err != nil {
		return r.cleanupAfterFailure(moduleCtx, booted, fmt.Errorf("freeze i18n registry: %w", err))
	}
	return r.ensureLifecycleActive(runCtx, moduleCtx, booted)
}

func (r *Runtime) preregisterOwnerLocaleResources() error {
	if r == nil || r.i18n == nil {
		return errors.New("runtime i18n service is unavailable")
	}
	if r.localeResourcesRegistered {
		return nil
	}

	resources := runtimeEmbeddedLocaleResources()
	if len(resources) == 0 {
		r.localeResourcesRegistered = true
		return nil
	}
	if err := r.i18n.RegisterEmbeddedLocaleResources(resources); err != nil {
		return fmt.Errorf("pre-register locale resources: %w", err)
	}
	r.localeResourcesRegistered = true
	return nil
}

func (r *Runtime) assertOwnerLocaleResourcesRegistered(
	moduleCtx *module.Context,
	booted []module.RuntimeModule,
) error {
	if r == nil || r.i18n == nil {
		return r.cleanupAfterFailure(moduleCtx, booted, errors.New("runtime i18n service is unavailable"))
	}
	if !r.localeResourcesRegistered {
		return r.cleanupAfterFailure(moduleCtx, booted, errors.New("runtime owner-local locale resources were not pre-registered"))
	}
	return nil
}

func (r *Runtime) runServerAndShutdown(
	runCtx context.Context,
	moduleCtx *module.Context,
	booted []module.RuntimeModule,
) error {
	if err := r.ensureLifecycleActive(runCtx, moduleCtx, booted); err != nil {
		return err
	}
	if err := r.server.Run(runCtx, r.config.HTTP.Addr); err != nil {
		return r.cleanupAfterFailure(moduleCtx, booted, err)
	}

	if err := shutdownModules(moduleCtx, booted); err != nil {
		r.appLogger().Error(moduleCtx.LifecycleContext, "app runtime shutdown failed",
			logger.StringField(logger.FieldOperation, "runtime_shutdown"),
			logger.ErrorField(err),
		)
		return r.cleanupAfterFailure(moduleCtx, nil, err)
	}

	if err := r.closeCoreResources(); err != nil {
		return err
	}

	return nil
}

func (r *Runtime) ensureLifecycleActive(
	ctx context.Context,
	moduleCtx *module.Context,
	booted []module.RuntimeModule,
) error {
	if err := lifecycleCanceled(ctx); err != nil {
		return r.cleanupAfterFailure(moduleCtx, booted, err)
	}
	return nil
}

func (r *Runtime) registerModules(moduleCtx *module.Context, ordered []module.RuntimeModule, booted []module.RuntimeModule) error {
	for _, p := range ordered {
		// Register 阶段只允许声明能力，不应启动长期运行行为；一旦失败，
		// 当前模块及其后续模块都不再继续，避免部分注册状态继续扩散。
		if err := p.Register(moduleCtx); err != nil {
			return r.cleanupAfterFailure(moduleCtx, booted, fmt.Errorf("register module %s: %w", p.Name(), err))
		}
	}

	return nil
}

func (r *Runtime) bootModules(
	moduleCtx *module.Context,
	ordered []module.RuntimeModule,
	booted []module.RuntimeModule,
) ([]module.RuntimeModule, error) {
	for _, p := range ordered {
		if err := lifecycleCanceled(moduleCtx.LifecycleContext); err != nil {
			return nil, r.cleanupAfterFailure(moduleCtx, booted, err)
		}
		// 只有完成 Register 的模块才会进入 Boot。booted 只记录真正成功启动
		// 的模块，确保失败清理不会误关未启动模块。
		if err := p.Boot(moduleCtx); err != nil {
			if lifecycleErr := lifecycleCanceled(moduleCtx.LifecycleContext); lifecycleErr != nil &&
				errors.Is(err, lifecycleErr) {
				return nil, r.cleanupAfterFailure(moduleCtx, booted, lifecycleErr)
			}
			r.appLogger().Error(moduleCtx.LifecycleContext, "module boot failed",
				logger.StringField(logger.FieldOperation, "module_boot"),
				logger.StringField("module", p.Name()),
				logger.ErrorField(err),
			)
			return nil, r.cleanupAfterFailure(moduleCtx, booted, fmt.Errorf("boot module %s: %w", p.Name(), err))
		}
		booted = append(booted, p)
		r.appLogger().Info(moduleCtx.LifecycleContext, "module boot completed",
			logger.StringField(logger.FieldOperation, "module_boot"),
			logger.StringField("module", p.Name()),
		)
	}

	return booted, nil
}

func lifecycleCanceled(ctx context.Context) error {
	if ctx == nil {
		return nil
	}
	return ctx.Err()
}

func (r *Runtime) newModuleContext(runCtx context.Context) *module.Context {
	return &module.Context{
		LifecycleContext:   runCtx,
		Config:             r.config,
		Logger:             r.logger,
		I18n:               r.i18n,
		EventBus:           r.eventBus,
		Realtime:           r.realtimeHub,
		Router:             r.server.Engine().Group("/api"),
		Services:           r.services,
		RuntimeMetadata:    r.runtimeMetadata,
		MenuRegistry:       r.menuRegistry,
		PermissionRegistry: r.permissionRegistry,
		CronRegistry:       r.cronRegistry,
		ConfigRegistry:     r.configRegistry,
		DashboardRegistry:  r.dashboardRegistry,
	}
}
