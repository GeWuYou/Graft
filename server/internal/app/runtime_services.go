package app

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"go.uber.org/zap"

	"graft/server/internal/cachex"
	"graft/server/internal/config"
	"graft/server/internal/configregistry"
	"graft/server/internal/container"
	"graft/server/internal/database"
	"graft/server/internal/eventbus"
	"graft/server/internal/httpx"
	"graft/server/internal/i18n"
	"graft/server/internal/kvx"
	"graft/server/internal/logger"
	"graft/server/internal/module"
	"graft/server/internal/realtime"
	"graft/server/internal/realtimeauth"
	"graft/server/internal/redisx"
	"graft/server/internal/statex"
)

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
		{
			key: (*realtime.Hub)(nil),
			provider: func() (any, error) {
				return r.realtimeHub, nil
			},
		},
		{
			key: (*realtime.TopicIssuerRegistry)(nil),
			provider: func() (any, error) {
				return r.realtimeTopicIssuers, nil
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
