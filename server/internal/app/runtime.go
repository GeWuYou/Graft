// Package app 组装 Graft 的显式运行时外壳。
package app

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

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
	"graft/server/internal/logger"
	"graft/server/internal/menu"
	"graft/server/internal/permission"
	"graft/server/internal/plugin"
	"graft/server/internal/pluginregistry"
	"graft/server/internal/redisx"
	"graft/server/internal/store"
	"graft/server/internal/store/entstore"
)

const pluginShutdownTimeout = 5 * time.Second

// Runtime 持有 MVP 运行时的核心资源与插件生命周期执行入口。
//
// Runtime 把配置、数据库、Redis、HTTP 服务、注册中心和插件管理器集中
// 到一个显式对象中，方便在失败路径和正常关闭路径统一回收资源。
//
// Runtime 本身不承载业务能力；它只负责 core 资源装配、插件生命周期编排
// 和进程级关闭顺序，避免插件把运行时控制逻辑反向塞回 core。
type Runtime struct {
	config             *config.Config
	logger             *zap.Logger
	i18n               *i18n.Service
	database           *database.Resources
	redis              *redis.Client
	server             *httpx.Server
	eventBus           eventbus.Bus
	services           *container.Container
	stores             store.Factory
	menuRegistry       *menu.Registry
	permissionRegistry *permission.Registry
	cronRegistry       *cronx.Registry
	pluginManager      *plugin.Manager
}

// NewRuntime 使用给定插件构造显式的 MVP 运行时外壳。
//
// 参数：
//   - plugins: 需要接入当前进程的插件集合；这里只注册插件元数据，不执行插件生命周期。
//
// 返回：
//   - *Runtime: 已完成 core 资源装配和插件登记的运行时对象。
//   - error: 当配置、数据库、Redis 或核心服务注册失败时返回错误，并尽力回收已创建资源。
func NewRuntime() (*Runtime, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	runtimeLogger, err := logger.New(cfg)
	if err != nil {
		return nil, fmt.Errorf("create logger: %w", err)
	}

	databaseResources, err := database.Open(cfg.Database)
	if err != nil {
		_ = logger.Close(runtimeLogger)
		return nil, err
	}

	redisClient, err := redisx.Open(context.Background(), cfg.Redis)
	if err != nil {
		_ = database.Close(databaseResources)
		_ = logger.Close(runtimeLogger)
		return nil, err
	}

	server := httpx.NewServer()
	eventBus := eventbus.New(runtimeLogger)
	services := container.New()
	stores, err := entstore.NewFactory(databaseResources.Client)
	if err != nil {
		_ = redisClient.Close()
		_ = database.Close(databaseResources)
		_ = logger.Close(runtimeLogger)
		return nil, fmt.Errorf("create ent store factory: %w", err)
	}
	localizer := i18n.New(cfg.I18n)
	menuRegistry := menu.NewRegistry()
	permissionRegistry := permission.NewRegistry()
	cronRegistry := cronx.NewRegistry()
	pluginManager := plugin.NewManager()

	runtime := &Runtime{
		config:             cfg,
		logger:             runtimeLogger,
		i18n:               localizer,
		database:           databaseResources,
		redis:              redisClient,
		server:             server,
		eventBus:           eventBus,
		services:           services,
		stores:             stores,
		menuRegistry:       menuRegistry,
		permissionRegistry: permissionRegistry,
		cronRegistry:       cronRegistry,
		pluginManager:      pluginManager,
	}

	if err := runtime.registerCoreServices(); err != nil {
		_ = runtime.closeCoreResources()
		return nil, err
	}

	runtime.registerCoreRoutes(server.Engine())

	plugins, err := pluginregistry.BuildPlugins(plugin.BuildContext{
		Services: services,
	})
	if err != nil {
		_ = runtime.closeCoreResources()
		return nil, fmt.Errorf("build runtime plugins: %w", err)
	}

	for _, current := range plugins {
		if err := runtime.pluginManager.RegisterPlugin(current); err != nil {
			_ = runtime.closeCoreResources()
			return nil, err
		}
	}

	return runtime, nil
}

// Run 先执行插件注册与启动，再启动 HTTP 服务。
//
// 如果任一阶段失败，Run 会按已启动的实际范围反向释放插件与核心资源，
// 避免把半初始化状态泄漏到调用方。
//
// 参数：
//   - runCtx: 绑定当前进程运行期的上下文；取消后会触发 HTTP 服务停止，并继续进入插件与 core 资源清理。
//
// 返回：
//   - error: 返回注册、启动、监听、关闭阶段的首个失败，并按需要聚合插件关闭或 core 资源回收错误。
func (r *Runtime) Run(runCtx context.Context) error {
	pluginCtx := &plugin.Context{
		LifecycleContext:   runCtx,
		Config:             r.config,
		Logger:             r.logger,
		I18n:               r.i18n,
		EventBus:           r.eventBus,
		Redis:              r.redis,
		Router:             r.server.Engine().Group("/api"),
		Services:           r.services,
		MenuRegistry:       r.menuRegistry,
		PermissionRegistry: r.permissionRegistry,
		CronRegistry:       r.cronRegistry,
	}

	ordered, err := r.pluginManager.Ordered()
	if err != nil {
		return err
	}

	booted := make([]plugin.Plugin, 0, len(ordered))
	for _, p := range ordered {
		// Register 阶段只允许声明能力，不应启动长期运行行为；一旦失败，
		// 当前插件及其后续插件都不再继续，避免部分注册状态继续扩散。
		if err := p.Register(pluginCtx); err != nil {
			return r.cleanupAfterFailure(pluginCtx, booted, fmt.Errorf("register plugin %s: %w", p.Name(), err))
		}
	}

	if err := r.i18n.Freeze(); err != nil {
		return r.cleanupAfterFailure(pluginCtx, booted, fmt.Errorf("freeze i18n registry: %w", err))
	}

	for _, p := range ordered {
		// 只有完成 Register 的插件才会进入 Boot。booted 只记录真正成功启动
		// 的插件，确保失败清理不会误关未启动插件。
		if err := p.Boot(pluginCtx); err != nil {
			return r.cleanupAfterFailure(pluginCtx, booted, fmt.Errorf("boot plugin %s: %w", p.Name(), err))
		}
		booted = append(booted, p)
	}

	if err := r.server.Run(runCtx, r.config.HTTP.Addr); err != nil {
		return r.cleanupAfterFailure(pluginCtx, booted, err)
	}

	if err := shutdownPlugins(pluginCtx, booted); err != nil {
		return r.cleanupAfterFailure(pluginCtx, nil, err)
	}

	if err := r.closeCoreResources(); err != nil {
		return err
	}

	return nil
}

func (r *Runtime) registerCoreRoutes(engine *gin.Engine) {
	engine.GET("/healthz", func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, gin.H{
			"status":         "ok",
			"defaultLocale":  r.i18n.DefaultLocale(),
			"fallbackLocale": r.i18n.FallbackLocale(),
			"menus":          len(r.menuRegistry.Items()),
			"permissions":    len(r.permissionRegistry.Items()),
			"jobs":           len(r.cronRegistry.Items()),
		})
	})
}

func (r *Runtime) registerCoreServices() error {
	registrations := []struct {
		key      any
		provider func() (any, error)
	}{
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
		{
			key: (*ent.Client)(nil),
			provider: func() (any, error) {
				if r.database == nil || r.database.Client == nil {
					return nil, errors.New("ent client is unavailable")
				}
				return r.database.Client, nil
			},
		},
		{
			key: (*store.UserRepository)(nil),
			provider: func() (any, error) {
				if r.stores == nil {
					return nil, errors.New("user repository is unavailable")
				}
				return r.stores.Users(), nil
			},
		},
		{
			key: (*store.AuthRepository)(nil),
			provider: func() (any, error) {
				if r.stores == nil {
					return nil, errors.New("auth repository is unavailable")
				}
				return r.stores.Auth(), nil
			},
		},
		{
			key: (*redis.Client)(nil),
			provider: func() (any, error) {
				return r.redis, nil
			},
		},
	}

	for _, registration := range registrations {
		if err := r.registerSingleton(registration.key, registration.provider); err != nil {
			return err
		}
	}

	return nil
}

func (r *Runtime) registerSingleton(key any, provider func() (any, error)) error {
	return r.services.RegisterSingleton(key, func(_ container.Resolver) (any, error) {
		return provider()
	})
}

// shutdownPlugins 按启动逆序关闭插件，并聚合所有关闭错误。
//
// 这里不在首个失败处提前返回，因为关闭阶段的目标是尽最大努力释放资源，
// 而不是维持“全部成功或立即退出”的启动语义。
func shutdownPlugins(ctx *plugin.Context, ordered []plugin.Plugin) error {
	shutdownCtx, cancel := withPluginShutdownContext(ctx)
	defer cancel()

	var shutdownErr error
	for i := len(ordered) - 1; i >= 0; i-- {
		// 关闭顺序必须与启动顺序相反，避免后启动的依赖还未释放时，上游
		// 插件先被销毁，导致清理逻辑访问失效资源。
		if err := ordered[i].Shutdown(shutdownCtx); err != nil {
			shutdownErr = errors.Join(shutdownErr, fmt.Errorf("shutdown plugin %s: %w", ordered[i].Name(), err))
		}
	}

	return shutdownErr
}

func withPluginShutdownContext(ctx *plugin.Context) (*plugin.Context, context.CancelFunc) {
	shutdownCtx, cancel := context.WithTimeout(context.Background(), pluginShutdownTimeout)
	if ctx == nil {
		return &plugin.Context{LifecycleContext: shutdownCtx}, cancel
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
// 这里保留原始失败原因，并把插件关闭和 core 资源回收错误聚合到同一个
// 返回值中，方便调用方看到完整失败路径。
func (r *Runtime) cleanupAfterFailure(ctx *plugin.Context, booted []plugin.Plugin, cause error) error {
	err := cause
	if shutdownErr := shutdownPlugins(ctx, booted); shutdownErr != nil {
		err = errors.Join(err, shutdownErr)
	}
	if closeErr := r.closeCoreResources(); closeErr != nil {
		err = errors.Join(err, closeErr)
	}
	return err
}
