package config

import "github.com/spf13/viper"

// readConfig 从 viper 配置读取并组装 Config。
//
// 它会填充应用、HTTP、HTTPX、审计、文档、模块、数据库、Redis、日志、运行时、i18n、鉴权和容器相关配置。
//
// @returns 读取并组装后的 Config 指针。
func readConfig(reader *viper.Viper) *Config {
	return &Config{
		App: AppConfig{
			Name: reader.GetString("app.name"),
			Env:  reader.GetString("app.env"),
		},
		HTTP: HTTPConfig{
			Addr: reader.GetString("http.addr"),
		},
		HTTPX: HTTPXConfig{
			AccessLogRetention:       reader.GetDuration("httpx.access_log_retention"),
			AccessLogConsole:         AccessLogConsolePolicy(reader.GetString("access_log.console")),
			AccessLogSlowThresholdMS: reader.GetInt64("access_log.slow_threshold_ms"),
			WebSocketAllowedOrigins:  parseCommaSeparatedList(reader.GetString("httpx.websocket.allowed_origins")),
		},
		Audit: AuditConfig{
			LogRetention: reader.GetDuration("audit.log_retention"),
		},
		Docs: DocsConfig{
			Enabled: resolveDocsEnabled(reader),
		},
		Modules: ModulesConfig{
			Enabled: parseModuleList(reader.GetString("modules.enabled")),
		},
		Database: DatabaseConfig{
			Driver:          reader.GetString("database.driver"),
			URL:             reader.GetString("database.url"),
			MaxOpenConns:    reader.GetInt("database.max_open_conns"),
			MaxIdleConns:    reader.GetInt("database.max_idle_conns"),
			ConnMaxLifetime: reader.GetDuration("database.conn_max_lifetime"),
			ConnMaxIdleTime: reader.GetDuration("database.conn_max_idle_time"),
		},
		Redis: RedisConfig{
			Addr:            reader.GetString("redis.addr"),
			Password:        reader.GetString("redis.password"),
			DB:              reader.GetInt("redis.db"),
			PoolSize:        reader.GetInt("redis.pool_size"),
			MinIdleConns:    reader.GetInt("redis.min_idle_conns"),
			MaxIdleConns:    reader.GetInt("redis.max_idle_conns"),
			MaxActiveConns:  reader.GetInt("redis.max_active_conns"),
			PoolTimeout:     reader.GetDuration("redis.pool_timeout"),
			ConnMaxIdleTime: reader.GetDuration("redis.conn_max_idle_time"),
			ConnMaxLifetime: reader.GetDuration("redis.conn_max_lifetime"),
		},
		Log: LogConfig{
			Level:           reader.GetString("log.level"),
			Format:          LogFormat(reader.GetString("log.format")),
			Color:           LogColor(reader.GetString("log.color")),
			AppLogPersist:   reader.GetBool("log.app_log_persist"),
			AppLogRetention: reader.GetDuration("log.app_log_retention"),
		},
		Runtime: RuntimeConfig{
			GinMode:                         GinMode(reader.GetString("gin.mode")),
			DevAllowDirtyMigrationBootstrap: reader.GetBool("runtime.dev_allow_dirty_migration_bootstrap"),
		},
		I18n: I18nConfig{
			DefaultLocale:    reader.GetString("i18n.default_locale"),
			FallbackLocale:   reader.GetString("i18n.fallback_locale"),
			SupportedLocales: parseLocaleList(reader.GetString("i18n.supported_locales")),
		},
		Auth: AuthConfig{
			AccessTokenTTL:        reader.GetDuration("auth.access_token_ttl"),
			RefreshTokenTTL:       reader.GetDuration("auth.refresh_token_ttl"),
			JWTSecret:             reader.GetString("auth.jwt_secret"),
			SigningKey:            reader.GetString("auth.signing_key"),
			RefreshCookieName:     reader.GetString("auth.refresh_cookie_name"),
			RefreshCookieSecure:   reader.GetBool("auth.refresh_cookie_secure"),
			RefreshCookieSameSite: reader.GetString("auth.refresh_cookie_same_site"),
			RefreshCookiePath:     reader.GetString("auth.refresh_cookie_path"),
		},
		Container: ContainerConfig{
			Runtime:                 reader.GetString("ops.container.runtime"),
			DockerEndpoint:          reader.GetString("ops.container.docker.endpoint"),
			LogsDefaultTail:         reader.GetInt("ops.container.logs.default_tail"),
			LogsMaxTail:             reader.GetInt("ops.container.logs.max_tail"),
			RuntimeEnabled:          reader.GetBool("ops.container.runtime.enabled"),
			DangerousActionsEnabled: reader.GetBool("ops.container.actions.dangerous_enabled"),
			ShellEnabled:            reader.GetBool("ops.container.shell.enabled"),
		},
	}
}
