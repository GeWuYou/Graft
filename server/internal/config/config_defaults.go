package config

import (
	"time"

	"github.com/spf13/viper"
)

// 它会为应用标识、HTTP、WebSocket、模块开关、数据库、Redis、日志、Gin 模式、i18n、鉴权以及容器运维相关配置设置默认值。
func setDefaults(reader *viper.Viper) {
	reader.SetDefault("app.name", defaultAppName)
	reader.SetDefault("app.env", defaultAppEnv)
	reader.SetDefault("http.addr", defaultHTTPAddr)
	reader.SetDefault("httpx.access_log_retention", defaultAccessLogRetentionForEnv(reader.GetString("app.env")))
	reader.SetDefault("access_log.console", string(AccessLogConsoleAuto))
	reader.SetDefault("access_log.slow_threshold_ms", defaultAccessLogSlowThreshold/time.Millisecond)
	reader.SetDefault("httpx.websocket.allowed_origins", defaultRealtimeAllowedOrigins)
	reader.SetDefault("audit.log_retention", defaultAuditLogRetentionForEnv(reader.GetString("app.env")))
	reader.SetDefault("modules.enabled", "")
	reader.SetDefault("database.driver", defaultDatabaseDriver)
	reader.SetDefault("database.url", defaultDatabaseURL)
	reader.SetDefault("database.max_open_conns", defaultDatabaseMaxOpenConns)
	reader.SetDefault("database.max_idle_conns", defaultDatabaseMaxIdleConns)
	reader.SetDefault("database.conn_max_lifetime", defaultDatabaseConnMaxLifetime)
	reader.SetDefault("database.conn_max_idle_time", defaultDatabaseConnMaxIdleTime)
	reader.SetDefault("redis.addr", defaultRedisAddr)
	reader.SetDefault("redis.password", "")
	reader.SetDefault("redis.db", 0)
	reader.SetDefault("redis.pool_size", defaultRedisPoolSize)
	reader.SetDefault("redis.min_idle_conns", defaultRedisMinIdleConns)
	reader.SetDefault("redis.max_idle_conns", defaultRedisMaxIdleConns)
	reader.SetDefault("redis.max_active_conns", defaultRedisMaxActiveConns)
	reader.SetDefault("redis.pool_timeout", defaultRedisPoolTimeout)
	reader.SetDefault("redis.conn_max_idle_time", defaultRedisConnMaxIdleTime)
	reader.SetDefault("redis.conn_max_lifetime", defaultRedisConnMaxLifetime)
	reader.SetDefault("log.level", defaultLogLevel)
	reader.SetDefault("log.format", string(LogFormatAuto))
	reader.SetDefault("log.color", string(LogColorAuto))
	reader.SetDefault("log.app_log_persist", defaultAppLogPersistence)
	reader.SetDefault("log.app_log_retention", defaultAppLogRetentionForEnv(reader.GetString("app.env")))
	reader.SetDefault("gin.mode", string(GinModeAuto))
	reader.SetDefault("runtime.dev_allow_dirty_migration_bootstrap", defaultDevAllowDirtyMigrationBootstrapForEnv(reader.GetString("app.env")))
	reader.SetDefault("i18n.default_locale", defaultLocale)
	reader.SetDefault("i18n.fallback_locale", defaultLocale)
	reader.SetDefault("i18n.supported_locales", defaultSupported)
	reader.SetDefault("auth.access_token_ttl", defaultAccessTokenTTL)
	reader.SetDefault("auth.refresh_token_ttl", defaultRefreshTokenTTL)
	reader.SetDefault("auth.refresh_cookie_name", defaultRefreshCookieName)
	reader.SetDefault("auth.refresh_cookie_secure", false)
	reader.SetDefault("auth.refresh_cookie_same_site", defaultRefreshCookieSameSite)
	reader.SetDefault("auth.refresh_cookie_path", defaultRefreshCookiePath)
	reader.SetDefault("ops.container.runtime.enabled", false)
	reader.SetDefault("ops.container.runtime", defaultContainerRuntime)
	reader.SetDefault("ops.container.docker.endpoint", defaultContainerDockerEndpoint)
	reader.SetDefault("ops.container.logs.default_tail", defaultContainerLogsDefaultTail)
	reader.SetDefault("ops.container.logs.max_tail", defaultContainerLogsMaxTail)
	reader.SetDefault("ops.container.actions.dangerous_enabled", false)
	reader.SetDefault("ops.container.shell.enabled", false)
}

// resolveDocsEnabled 根据显式配置或环境默认值确定是否启用文档。
// 当未设置 `docs.enabled` 时，使用 `app.env` 对应的默认值；如果 `reader` 为空，则使用默认应用环境的默认值。
func resolveDocsEnabled(reader *viper.Viper) bool {
	if reader == nil {
		return defaultDocsEnabledForEnv(defaultAppEnv)
	}

	if reader.IsSet("docs.enabled") {
		return reader.GetBool("docs.enabled")
	}

	return defaultDocsEnabledForEnv(reader.GetString("app.env"))
}
