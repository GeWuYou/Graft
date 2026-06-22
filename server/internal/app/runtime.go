// Package app 组装 Graft 的显式运行时外壳。
package app

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gopkg.in/yaml.v3"
	"go.uber.org/zap"

	"graft/server/internal/buildinfo"
	"graft/server/internal/cachex"
	cachebackend "graft/server/internal/cachex/backend"
	"graft/server/internal/config"
	"graft/server/internal/configregistry"
	"graft/server/internal/container"
	generated "graft/server/internal/contract/openapi/generated"
	healthopenapi "graft/server/internal/contract/openapi/health"
	"graft/server/internal/cronx"
	"graft/server/internal/dashboard"
	"graft/server/internal/database"
	"graft/server/internal/eventbus"
	"graft/server/internal/httpx"
	"graft/server/internal/i18n"
	"graft/server/internal/kvx"
	"graft/server/internal/logger"
	"graft/server/internal/menu"
	"graft/server/internal/module"
	"graft/server/internal/moduleapi"
	"graft/server/internal/moduleregistry"
	"graft/server/internal/moduleruntime"
	moduleruntimelocales "graft/server/internal/moduleruntime/locales"
	"graft/server/internal/permission"
	"graft/server/internal/realtimeauth"
	"graft/server/internal/redisx"
	"graft/server/internal/statex"
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
//   - error: 当配置、数据库、Redis 或核心服务注册失败时返回错误，并尽力回收已创建资源。
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

	runtime.registerCoreRoutes(runtime.server.Engine())

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

// newRuntimeCoreWithDeps initializes all core runtime resources and returns a fully configured Runtime instance with locale resources pre-registered.
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
		eventBus:           eventbus.New(runtimeLogger),
		services:           container.New(),
		menuRegistry:       menu.NewRegistry(),
		permissionRegistry: permission.NewRegistry(),
		cronRegistry:       cronx.NewRegistry(),
		configRegistry:     configregistry.NewRegistry(),
		dashboardRegistry:  dashboard.NewRegistry(),
		moduleManager:      module.NewManager(),
		appLogRepository:   appLogRepo,
	}
	if err := runtime.preregisterOwnerLocaleResources(); err != nil {
		_ = runtime.closeCoreResources()
		return nil, err
	}

	return runtime, nil
}

// normalizeRuntimeCoreDeps replaces nil constructor functions with default implementations.
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

func (r *Runtime) registerCoreAuthenticatedRoutes() error {
	authService, authorizer, err := r.resolveLogExplorerAuth()
	if errors.Is(err, container.ErrServiceNotRegistered) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("resolve log explorer auth service: %w", err)
	}

	if err := r.registerAccessLogExplorerWithAuth(authService, authorizer); err != nil {
		return err
	}
	if err := r.registerAppLogExplorerWithAuth(authService, authorizer); err != nil {
		return err
	}
	if err := r.registerModuleRuntimeWithAuth(authService, authorizer); err != nil {
		return err
	}
	if err := r.registerCoreDashboardWidgets(); err != nil {
		return err
	}
	if err := r.registerDashboardWithAuth(authService, authorizer); err != nil {
		return err
	}

	return nil
}

func (r *Runtime) registerModuleRuntimeWithAuth(
	authService moduleapi.AuthService,
	authorizer moduleapi.Authorizer,
) error {
	if err := moduleruntime.Register(
		moduleruntime.Registration{
			I18n:               r.i18n,
			MenuRegistry:       r.menuRegistry,
			PermissionRegistry: r.permissionRegistry,
			EventBus:           r.eventBus,
			Config:             r.config,
			Specs:              r.moduleRuntimeSpecs(),
		},
		r.server.Engine().Group("/api"),
		authService,
		authorizer,
	); err != nil {
		return fmt.Errorf("register module runtime routes: %w", err)
	}

	return nil
}

func (r *Runtime) registerCoreDashboardWidgets() error {
	if r.dashboardRegistry == nil {
		return errors.New("dashboard registry is unavailable")
	}

	if err := r.registerCoreModuleRuntimeDashboard(); err != nil {
		return err
	}
	if err := r.registerCoreAccessLogDashboard(); err != nil {
		return err
	}
	if err := r.registerCoreAppLogDashboard(); err != nil {
		return err
	}

	return nil
}

func (r *Runtime) registerCoreModuleRuntimeDashboard() error {
	if err := r.dashboardRegistry.Register(dashboard.WidgetDefinition{
		ID:             "core.module-runtime-health",
		ModuleKey:      "core",
		TitleKey:       moduleRuntimeHealthTitleKey,
		Title:          r.mustLookupCoreDisplay(moduleRuntimeHealthTitleKey),
		DescriptionKey: moduleRuntimeHealthDescriptionKey,
		Description:    r.mustLookupCoreDisplay(moduleRuntimeHealthDescriptionKey),
		Type:           dashboard.WidgetTypeHealth,
		Size:           dashboard.WidgetSizeMedium,
		Category:       dashboard.WidgetCategorySystem,
		Priority:       dashboard.WidgetPriorityInfo,
		Order:          coreModuleRuntimeHealthWidgetOrder,
		RouteLocation:  moduleruntime.MenuRuntimePath(),
		Action: dashboard.WidgetAction{
			LabelKey: "dashboard.actions.details",
			Label:    r.mustLookupCoreDisplay("dashboard.actions.details"),
			Route:    moduleruntime.MenuRuntimePath(),
		},
		RequiredPermissions: []string{moduleruntime.PermissionRead},
		Loader: dashboard.WidgetLoaderFunc(func(context.Context, dashboard.WidgetRequest) (dashboard.WidgetPayload, error) {
			snapshot := moduleruntime.BuildSnapshot(r.config, r.moduleRuntimeSpecs())
			items := make([]dashboard.HealthItem, 0, len(snapshot.Items))
			for _, item := range snapshot.Items {
				if item.Health == generated.ModuleRuntimeItemHealthHealthy {
					continue
				}
				items = append(items, dashboard.HealthItem{
					Key:            item.ModuleKey,
					LabelKey:       "dashboard.widget.moduleRuntimeHealth.item." + item.ModuleKey,
					Label:          item.ModuleKey,
					Status:         dashboard.HealthStatus(string(item.Health)),
					DescriptionKey: moduleRuntimeStatusDescriptionKey(item.RuntimeStatus),
					Description:    string(item.RuntimeStatus),
					RouteLocation:  moduleruntime.MenuRuntimePath(),
				})
			}

			abnormalModules := snapshot.Summary.DegradedModules + snapshot.Summary.UnknownModules
			summaryStatus := dashboard.HealthStatusHealthy
			widgetState := dashboard.WidgetStateNormal
			widgetPriority := dashboard.WidgetPriorityInfo
			if abnormalModules > 0 {
				summaryStatus = dashboard.HealthStatusDegraded
				widgetState = dashboard.WidgetStateWarning
				widgetPriority = dashboard.WidgetPriorityWarning
			}
			if snapshot.Summary.EnabledModules == 0 && snapshot.Summary.TotalModules > 0 {
				summaryStatus = dashboard.HealthStatusDisabled
				widgetState = dashboard.WidgetStateWarning
				widgetPriority = dashboard.WidgetPriorityWarning
			}

			return dashboard.WidgetPayload{
				"summary": dashboard.HealthSummaryItem{
					Status:   summaryStatus,
					LabelKey: moduleRuntimeHealthSummaryKey,
					Label:    r.mustLookupCoreDisplay(moduleRuntimeHealthSummaryKey),
				},
				"items":             items,
				"healthy_modules":   snapshot.Summary.HealthyModules,
				"enabled_modules":   snapshot.Summary.EnabledModules,
				"abnormal_services": abnormalModules,
				"state":             string(widgetState),
				"priority":          string(widgetPriority),
			}, nil
		}),
	}); err != nil {
		return fmt.Errorf("register core dashboard widget: %w", err)
	}

	return nil
}

func (r *Runtime) registerCoreAccessLogDashboard() error {
	if repo := r.server.AccessLogRepository(); repo != nil {
		if err := r.dashboardRegistry.Register(dashboard.WidgetDefinition{
			ID:             httpx.AccessLogDashboardWidgetID,
			ModuleKey:      httpx.AccessLogDashboardModuleKey(),
			TitleKey:       "dashboard.widget.accessLogRequestAttention.title",
			Title:          r.mustLookupCoreDisplay("dashboard.widget.accessLogRequestAttention.title"),
			DescriptionKey: "dashboard.widget.accessLogRequestAttention.description",
			Description:    r.mustLookupCoreDisplay("dashboard.widget.accessLogRequestAttention.description"),
			Type:           dashboard.WidgetTypeAlertList,
			Size:           dashboard.WidgetSizeMedium,
			Category:       dashboard.WidgetCategoryOperation,
			Priority:       dashboard.WidgetPriorityWarning,
			Order:          httpx.AccessLogDashboardWidgetOrder,
			RouteLocation:  httpx.AccessLogDashboardRouteLocation(),
			Action: dashboard.WidgetAction{
				LabelKey: "dashboard.actions.details",
				Label:    r.mustLookupCoreDisplay("dashboard.actions.details"),
				Route:    httpx.AccessLogDashboardRouteLocation(),
			},
			RequiredPermissions: []string{httpx.AccessLogReadPermission},
			Loader: dashboard.WidgetLoaderFunc(func(ctx context.Context, _ dashboard.WidgetRequest) (dashboard.WidgetPayload, error) {
				return httpx.LoadAccessLogRequestAttentionPayload(ctx, repo)
			}),
		}); err != nil {
			return fmt.Errorf("register access-log dashboard widget: %w", err)
		}
	}
	return nil
}

func (r *Runtime) registerCoreAppLogDashboard() error {
	return nil
}

func (r *Runtime) mustLookupCoreDisplay(key string) string {
	if r == nil || r.i18n == nil {
		panic("core display localization requires i18n service")
	}
	if len(r.i18n.RegisteredMessageResources(i18n.LocaleTag(r.i18n.DefaultLocale()), i18n.MessageKey(key))) == 0 {
		panic("core display localization key missing: " + key)
	}

	return r.i18n.Lookup(i18n.LookupRequest{
		Locale: i18n.LocaleTag(r.i18n.DefaultLocale()),
		Key:    i18n.MessageKey(key),
	})
}

func moduleRuntimeStatusDescriptionKey(status generated.ModuleRuntimeItemRuntimeStatus) string {
	switch status {
	case generated.ModuleRuntimeItemRuntimeStatusRegistered:
		return "dashboard.widget.moduleRuntimeHealth.runtimeStatus.registered"
	case generated.ModuleRuntimeItemRuntimeStatusDisabled:
		return "dashboard.widget.moduleRuntimeHealth.runtimeStatus.disabled"
	case generated.ModuleRuntimeItemRuntimeStatusDegraded:
		return "dashboard.widget.moduleRuntimeHealth.runtimeStatus.degraded"
	default:
		return "dashboard.widget.moduleRuntimeHealth.runtimeStatus.unknown"
	}
}

func (r *Runtime) registerDashboardWithAuth(
	authService moduleapi.AuthService,
	authorizer moduleapi.Authorizer,
) error {
	if err := dashboard.Register(
		dashboard.Registration{
			I18n:     r.i18n,
			Config:   r.config,
			Registry: r.dashboardRegistry,
			Logger:   r.injectedAppLogger(),
			ModuleRuntimeSummary: func() generated.ModuleRuntimeSummary {
				return moduleruntime.BuildSnapshot(r.config, r.moduleRuntimeSpecs()).Summary
			},
		},
		r.server.Engine().Group("/api"),
		authService,
		authorizer,
	); err != nil {
		return fmt.Errorf("register dashboard routes: %w", err)
	}

	return nil
}

func (r *Runtime) moduleRuntimeSpecs() []module.Spec {
	ordered, err := moduleregistry.OrderedModuleSpecs()
	if err != nil {
		r.appLogger().Warn(context.Background(), "module runtime spec ordering failed",
			logger.StringField(logger.FieldOperation, "module_runtime_specs"),
			logger.ErrorField(err),
		)
		return moduleregistry.ModuleSpecs()
	}

	return ordered
}

func (r *Runtime) registerAccessLogExplorerWithAuth(
	authService moduleapi.AuthService,
	authorizer moduleapi.Authorizer,
) error {
	if err := httpx.RegisterAccessLogExplorer(
		httpx.AccessLogExplorerRegistration{
			I18n:               r.i18n,
			MenuRegistry:       r.menuRegistry,
			PermissionRegistry: r.permissionRegistry,
			EventBus:           r.eventBus,
		},
		r.server.Engine().Group("/api"),
		r.server.AccessLogRepository(),
		authService,
		authorizer,
	); err != nil {
		return fmt.Errorf("register access-log explorer: %w", err)
	}

	return nil
}

func (r *Runtime) registerAppLogExplorerWithAuth(
	authService moduleapi.AuthService,
	authorizer moduleapi.Authorizer,
) error {
	if r.appLogRepository == nil {
		return nil
	}

	if err := logger.RegisterAppLogExplorer(
		logger.AppLogExplorerRegistration{
			I18n:               r.i18n,
			MenuRegistry:       r.menuRegistry,
			PermissionRegistry: r.permissionRegistry,
			EventBus:           r.eventBus,
		},
		r.server.Engine().Group("/api"),
		r.appLogRepository,
		authService,
		authorizer,
	); err != nil {
		return fmt.Errorf("register app-log explorer: %w", err)
	}

	return nil
}

func (r *Runtime) resolveLogExplorerAuth() (moduleapi.AuthService, moduleapi.Authorizer, error) {
	authService, err := r.resolveLogExplorerAuthService()
	if err != nil {
		return nil, nil, err
	}

	authorizer, err := r.resolveLogExplorerAuthorizer()
	if err != nil {
		return nil, nil, err
	}

	return authService, authorizer, nil
}

func (r *Runtime) resolveLogExplorerAuthService() (moduleapi.AuthService, error) {
	authResolved, err := r.services.Resolve((*moduleapi.AuthService)(nil))
	if err != nil {
		return nil, err
	}

	authService, ok := authResolved.(moduleapi.AuthService)
	if !ok {
		return nil, fmt.Errorf("unexpected type %T", authResolved)
	}

	return authService, nil
}

func (r *Runtime) resolveLogExplorerAuthorizer() (moduleapi.Authorizer, error) {
	authorizerResolved, err := r.services.Resolve((*moduleapi.Authorizer)(nil))
	if err != nil {
		return nil, fmt.Errorf("resolve access-log authorizer: %w", err)
	}

	authorizer, ok := authorizerResolved.(moduleapi.Authorizer)
	if !ok {
		return nil, fmt.Errorf("resolve access-log authorizer: unexpected type %T", authorizerResolved)
	}

	return authorizer, nil
}

func (r *Runtime) loadOptionalDocsAssets() error {
	if r.config == nil || !r.config.Docs.Enabled {
		return nil
	}

	docsAssets, err := loadOpenAPIDocsAssets()
	if err != nil {
		return fmt.Errorf("load openapi docs assets: %w", err)
	}

	r.openapiDocs = docsAssets
	return nil
}

func (r *Runtime) registerCoreRoutes(engine *gin.Engine) {
	engine.GET("/healthz", func(ctx *gin.Context) {
		coreHealthGeneratedHandler{}.GetHealthz()
		ctx.JSON(http.StatusOK, gin.H{
			"status":         "ok",
			"defaultLocale":  r.i18n.DefaultLocale(),
			"fallbackLocale": r.i18n.FallbackLocale(),
			"menus":          len(r.menuRegistry.Items()),
			"permissions":    len(r.permissionRegistry.Items()),
			"jobs":           len(r.cronRegistry.Items()),
		})
	})

	if r.config == nil || !r.config.Docs.Enabled || r.openapiDocs == nil {
		return
	}

	engine.GET(openapiJSONPath, func(ctx *gin.Context) {
		ctx.Data(http.StatusOK, "application/json; charset=utf-8", r.openapiDocs.json)
	})
	engine.GET(openapiYAMLPath, func(ctx *gin.Context) {
		yamlSpec, err := buildLegacyOpenAPIYAML(r.openapiDocs.json)
		if err != nil {
			if r.logger != nil {
				r.appLogger().
					Error(ctx.Request.Context(), "build legacy openapi yaml", logger.ErrorField(err))
			}
			ctx.String(http.StatusInternalServerError, "failed to render openapi yaml")
			return
		}
		ctx.Data(http.StatusOK, "application/yaml; charset=utf-8", yamlSpec)
	})
	engine.GET(openapiDocsPath, func(ctx *gin.Context) {
		html, err := renderScalarDocsHTML(openapiJSONPath)
		if err != nil {
			if r.logger != nil {
				r.appLogger().
					Error(ctx.Request.Context(), "render docs page", logger.ErrorField(err))
			}
			ctx.String(http.StatusInternalServerError, "failed to render docs page")
			return
		}
		ctx.Data(http.StatusOK, "text/html; charset=utf-8", html)
	})
}

var _ healthopenapi.ServerInterface = coreHealthGeneratedHandler{}

type coreHealthGeneratedHandler struct{}

func (h coreHealthGeneratedHandler) GetHealthz() {
	_ = h
}

func buildLegacyOpenAPIYAML(spec []byte) ([]byte, error) {
	if len(spec) == 0 {
		return nil, fmt.Errorf("generated bundled openapi spec is empty")
	}

	var document any
	if err := json.Unmarshal(spec, &document); err != nil {
		return nil, fmt.Errorf("decode generated bundled openapi json: %w", err)
	}

	yamlSpec, err := yaml.Marshal(document)
	if err != nil {
		return nil, fmt.Errorf("encode generated bundled openapi yaml: %w", err)
	}
	return yamlSpec, nil
}

func (r *Runtime) registerCoreServices() error {
	registrations := r.coreServiceRegistrations()

	for _, registration := range registrations {
		if err := r.registerSingleton(registration.key, registration.provider); err != nil {
			return err
		}
	}

	return nil
}

type serviceRegistration struct {
	key      any
	provider func() (any, error)
}

func (r *Runtime) coreServiceRegistrations() []serviceRegistration {
	registrations := make([]serviceRegistration, 0, coreServiceRegistrationCapacity)
	registrations = append(registrations, r.foundationServiceRegistrations()...)
	registrations = append(registrations, r.runtimeDataServiceRegistrations()...)
	registrations = append(registrations, r.redisBackedServiceRegistrations()...)
	return registrations
}

func (r *Runtime) foundationServiceRegistrations() []serviceRegistration {
	return []serviceRegistration{
		{
			key: (*configregistry.Registry)(nil),
			provider: func() (any, error) {
				return r.configRegistry, nil
			},
		},
		{
			key: (*config.Config)(nil),
			provider: func() (any, error) {
				return r.config, nil
			},
		},
		{
			key: (*zap.Logger)(nil),
			provider: func() (any, error) {
				return r.logger, nil
			},
		},
		{
			key: (*logger.AppLogger)(nil),
			provider: func() (any, error) {
				return r.newAppLogger(), nil
			},
		},
		{
			key: (*logger.AppLogRepository)(nil),
			provider: func() (any, error) {
				if r.appLogRepository == nil {
					return nil, errors.New("app log repository is unavailable")
				}
				return r.appLogRepository, nil
			},
		},
		{
			key: (*i18n.Service)(nil),
			provider: func() (any, error) {
				return r.i18n, nil
			},
		},
		{
			key: (*eventbus.Bus)(nil),
			provider: func() (any, error) {
				return r.eventBus, nil
			},
		},
	}
}

func (r *Runtime) runtimeDataServiceRegistrations() []serviceRegistration {
	return []serviceRegistration{
		{
			key: (*sql.DB)(nil),
			provider: func() (any, error) {
				if r.database == nil || r.database.SQL == nil {
					return nil, errors.New("database sql pool is unavailable")
				}
				return r.database.SQL, nil
			},
		},
		{
			key: (*cachex.Manager)(nil),
			provider: func() (any, error) {
				if r.cacheManager == nil {
					return nil, errors.New("cache manager is unavailable")
				}
				return r.cacheManager, nil
			},
		},
	}
}

func (r *Runtime) redisBackedServiceRegistrations() []serviceRegistration {
	return []serviceRegistration{
		{
			key: (*realtimeauth.Service)(nil),
			provider: func() (any, error) {
				if r.redis == nil {
					return nil, errors.New("redis client is unavailable")
				}

				namespace := "graft"
				if r.config != nil {
					appName := strings.TrimSpace(r.config.App.Name)
					if appName != "" {
						namespace = appName
					}
				}

				store, err := kvx.NewRedis(r.redis, kvx.RedisOptions{
					Prefix: namespace + ":kv:realtimeauth",
				})
				if err != nil {
					return nil, fmt.Errorf("create realtime ticket kv store: %w", err)
				}

				service, err := realtimeauth.NewService(store)
				if err != nil {
					return nil, fmt.Errorf("create realtime ticket service: %w", err)
				}

				return service, nil
			},
		},
		{
			key: (*redisx.HealthReporter)(nil),
			provider: func() (any, error) {
				return redisx.NewHealthReporter(r.redis), nil
			},
		},
		{
			key: (*statex.TimeSeriesStore)(nil),
			provider: func() (any, error) {
				return statex.NewRedisTimeSeriesStore(r.redis)
			},
		},
	}
}

// newRuntimeCacheManager creates a cache manager backed by Redis, with namespace derived from the application name in the configuration or defaulting to "graft".
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

func (r *Runtime) registerAccessLogRetentionJob() error {
	if r == nil || r.server == nil {
		return errors.New("runtime server is unavailable")
	}
	if err := httpx.RegisterAccessLogRetentionConfigMessages(r.i18n); err != nil {
		return fmt.Errorf("register access-log retention config messages: %w", err)
	}
	if err := httpx.RegisterAccessLogRetentionConfigDefinition(r.configRegistry); err != nil {
		return fmt.Errorf("register access-log retention config definition: %w", err)
	}

	if err := httpx.RegisterAccessLogRetentionCleanupJob(
		r.cronRegistry,
		r.logger,
		r.server.AccessLogRepository(),
		r.config.HTTPX,
	); err != nil {
		return fmt.Errorf("register access-log retention cleanup job: %w", err)
	}
	return nil
}

func (r *Runtime) registerAppLogRetentionJob() error {
	if r == nil {
		return errors.New("runtime is unavailable")
	}
	if r.appLogRepository == nil {
		return nil
	}
	if err := logger.RegisterAppLogRetentionConfigMessages(r.i18n); err != nil {
		return fmt.Errorf("register app-log retention config messages: %w", err)
	}
	if err := logger.RegisterAppLogRetentionConfigDefinition(r.configRegistry); err != nil {
		return fmt.Errorf("register app-log retention config definition: %w", err)
	}

	if err := logger.RegisterAppLogRetentionCleanupJob(
		r.cronRegistry,
		r.logger,
		r.injectedAppLogger(),
		r.appLogRepository,
		r.config.Log,
	); err != nil {
		return fmt.Errorf("register app-log retention cleanup job: %w", err)
	}
	return nil
}

func (r *Runtime) newAppLogger() logger.AppLogger {
	if r == nil {
		return logger.NewAppLogger(nil)
	}
	if r.appLogRepository == nil {
		return logger.NewAppLogger(r.logger)
	}
	return logger.NewAppLogger(r.logger, logger.WithAppLogRepository(r.appLogRepository))
}

func (r *Runtime) appLogger() logger.AppLogger {
	return r.injectedAppLogger().Named(appRuntimeLogComponent)
}

func (r *Runtime) injectedAppLogger() logger.AppLogger {
	if r == nil {
		return logger.NewAppLogger(nil)
	}
	if r.services == nil {
		return r.newAppLogger()
	}

	resolved, err := r.services.Resolve((*logger.AppLogger)(nil))
	if err != nil {
		return r.newAppLogger()
	}

	appLogger, ok := resolved.(logger.AppLogger)
	if !ok || appLogger == nil {
		return r.newAppLogger()
	}

	return appLogger
}

func (r *Runtime) registerSingleton(key any, provider func() (any, error)) error {
	return r.services.RegisterSingleton(key, func(_ container.Resolver) (any, error) {
		return provider()
	})
}

// shutdownModules 按启动逆序关闭模块，并聚合所有关闭错误。
//
// 这里不在首个失败处提前返回，因为关闭阶段的目标是尽最大努力释放资源，
// 而不是维持“全部成功或立即退出”的启动语义。
func shutdownModules(ctx *module.Context, ordered []module.RuntimeModule) error {
	shutdownCtx, cancel := withModuleShutdownContext(ctx)
	defer cancel()

	var shutdownErr error
	for i := len(ordered) - 1; i >= 0; i-- {
		// 关闭顺序必须与启动顺序相反，避免后启动的依赖还未释放时，上游
		// 模块先被销毁，导致清理逻辑访问失效资源。
		if err := ordered[i].Shutdown(shutdownCtx); err != nil {
			shutdownErr = errors.Join(shutdownErr, fmt.Errorf("shutdown module %s: %w", ordered[i].Name(), err))
		}
	}

	return shutdownErr
}

func withModuleShutdownContext(ctx *module.Context) (*module.Context, context.CancelFunc) {
	shutdownCtx, cancel := context.WithTimeout(context.Background(), moduleShutdownTimeout)
	if ctx == nil {
		return &module.Context{LifecycleContext: shutdownCtx}, cancel
	}

	cloned := *ctx
	cloned.LifecycleContext = shutdownCtx
	return &cloned, cancel
}

// closeCoreResources 释放 Runtime 持有的 core 级外部资源。
//
// 关闭失败会被聚合返回，但函数仍会继续尝试释放剩余资源，避免前一个
// 资源的错误掩盖后续必需的清理动作。
func (r *Runtime) closeCoreResources() error {
	var closeErr error
	if err := logger.Close(r.logger); err != nil {
		closeErr = errors.Join(closeErr, err)
	}
	r.logger = nil

	if r.redis != nil {
		if err := r.redis.Close(); err != nil {
			closeErr = errors.Join(closeErr, fmt.Errorf("close redis: %w", err))
		}
		r.redis = nil
	}

	if r.database != nil {
		if err := database.Close(r.database); err != nil {
			closeErr = errors.Join(closeErr, err)
		}
		r.database = nil
	}

	return closeErr
}

// cleanupAfterFailure 在启动或关闭中途失败后执行统一清理。
//
// 这里保留原始失败原因，并把模块关闭和 core 资源回收错误聚合到同一个
// 返回值中，方便调用方看到完整失败路径。
func (r *Runtime) cleanupAfterFailure(ctx *module.Context, booted []module.RuntimeModule, cause error) error {
	err := cause
	if shutdownErr := shutdownModules(ctx, booted); shutdownErr != nil {
		err = errors.Join(err, shutdownErr)
	}
	if closeErr := r.closeCoreResources(); closeErr != nil {
		err = errors.Join(err, closeErr)
	}
	return err
}
