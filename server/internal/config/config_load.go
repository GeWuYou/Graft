package config

import "github.com/spf13/viper"

// 它会填充应用、HTTP、HTTPX、审计、文档、模块、数据库、Redis、日志、运行时、i18n、鉴权和容器相关配置。
//
// readConfig 从 Viper 实例读取配置并组装为 Config；调用方应先完成默认值和环境变量绑定。
//
//nolint:funlen // Config 映射集中保留键到运行时字段的一一对应关系。
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
			AccessLogConsole:          AccessLogConsolePolicy(reader.GetString("access_log.console")),
			AccessLogSlowThresholdMS:  reader.GetInt64("access_log.slow_threshold_ms"),
			AccessLogPersistTimeoutMS: reader.GetInt64("access_log.persist_timeout_ms"),
			WebSocketAllowedOrigins:   parseCommaSeparatedList(reader.GetString("httpx.websocket.allowed_origins")),
			AgentTLS: AgentTLSConfig{
				Enabled:         reader.GetBool("httpx.agent_tls.enabled"),
				Addr:            reader.GetString("httpx.agent_tls.addr"),
				CertificateFile: reader.GetString("httpx.agent_tls.certificate_file"),
				KeyFile:         reader.GetString("httpx.agent_tls.key_file"),
				ClientCAFile:    reader.GetString("httpx.agent_tls.client_ca_file"),
			},
		},
		CredentialVault: CredentialVaultConfig{
			Enabled:        reader.GetBool("credential_vault.enabled"),
			Backend:        reader.GetString("credential_vault.backend"),
			Address:        reader.GetString("credential_vault.address"),
			Namespace:      reader.GetString("credential_vault.namespace"),
			AuthMount:      reader.GetString("credential_vault.auth_mount"),
			AuthRole:       reader.GetString("credential_vault.auth_role"),
			PKIMount:       reader.GetString("credential_vault.pki_mount"),
			PKIRole:        reader.GetString("credential_vault.pki_role"),
			TrustBundleRef: reader.GetString("credential_vault.trust_bundle_ref"),
		},
		Audit: AuditConfig{},
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
			Level:         reader.GetString("log.level"),
			Format:        LogFormat(reader.GetString("log.format")),
			Color:         LogColor(reader.GetString("log.color")),
			Categories:    reader.GetString("log.categories"),
			AppLogPersist: reader.GetBool("log.app_log_persist"),
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
		MCP: MCPConfig{
			Enabled:               reader.GetBool("mcp.enabled"),
			ConfirmationTokenTTL:  reader.GetDuration("mcp.confirmation_token_ttl"),
			SessionTimeout:        reader.GetDuration("mcp.session_timeout"),
			RequestTimeout:        reader.GetDuration("mcp.request_timeout"),
			MaxRequestBytes:       reader.GetInt64("mcp.max_request_bytes"),
			MaxSessions:           reader.GetInt("mcp.max_sessions"),
			MaxConcurrentRequests: reader.GetInt("mcp.max_concurrent_requests"),
		},
		Container: ContainerConfig{
			Runtime:        reader.GetString("ops.container.runtime"),
			DockerEndpoint: reader.GetString("ops.container.docker.endpoint"),
		},
		RegistryCredentials: RegistryCredentialSourceConfig{
			File: reader.GetString("registry.credentials_file"),
		},
		Backup: BackupConfig{
			ArtifactRoot: reader.GetString("backup.artifact_root"),
		},
		Project: ProjectConfig{
			LogDebug:           reader.GetBool("project.log_debug"),
			ManagedCreateDebug: reader.GetBool("project.managed_create_debug"),
		},
	}
}
