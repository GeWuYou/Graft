// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

const (
	defaultAppName        = "graft"
	defaultAppEnv         = "local"
	defaultHTTPAddr       = ":8080"
	defaultDatabaseDriver = "postgres"
	// #nosec G101 -- 本地开发默认 DSN 只作为示例值，不代表可用分发凭据。
	defaultDatabaseURL              = "postgres://graft:graft@localhost:5432/graft?sslmode=disable"
	defaultDatabaseMaxOpenConns     = 25
	defaultDatabaseMaxIdleConns     = 10
	defaultDatabaseConnMaxLifetime  = time.Hour
	defaultDatabaseConnMaxIdleTime  = 30 * time.Minute
	defaultRedisAddr                = "localhost:6379"
	defaultRedisPoolSize            = 0
	defaultRedisMinIdleConns        = 0
	defaultRedisMaxIdleConns        = 0
	defaultRedisMaxActiveConns      = 0
	defaultRedisPoolTimeout         = 0
	defaultRedisConnMaxIdleTime     = 0
	defaultRedisConnMaxLifetime     = 0
	defaultLogLevel                 = "info"
	defaultAppLogPersistence        = true
	defaultLocale                   = "zh-CN"
	defaultSecondaryLocale          = "en-US"
	defaultSupported                = "zh-CN,en-US"
	defaultAccessTokenTTL           = 15 * time.Minute
	defaultRefreshTokenTTL          = 7 * 24 * time.Hour
	defaultRefreshCookieName        = "graft_refresh_token"
	defaultRefreshCookiePath        = "/"
	defaultRefreshCookieSameSite    = "lax"
	defaultContainerRuntime         = "first-adapter"
	defaultContainerDockerEndpoint  = "unix:///var/run/docker.sock"
	defaultContainerLogsDefaultTail = 200
	defaultContainerLogsMaxTail     = 2000
	defaultRealtimeAllowedOrigins   = ""
)

const (
	// EnvAppEnv names the process environment variable that selects the runtime environment.
	EnvAppEnv = "GRAFT_APP_ENV"
	// EnvLogLevel names the process environment variable that selects the zap severity threshold.
	EnvLogLevel = "GRAFT_LOG_LEVEL"
	// EnvLogFormat names the process environment variable that selects console or JSON output.
	EnvLogFormat = "GRAFT_LOG_FORMAT"
	// EnvLogColor names the process environment variable that controls ANSI level colors.
	EnvLogColor = "GRAFT_LOG_COLOR"
	// EnvGinMode names the process environment variable that selects the Gin framework mode.
	EnvGinMode = "GRAFT_GIN_MODE"
	// EnvAccessLogConsole names the process environment variable that controls access-log console emission.
	EnvAccessLogConsole = "GRAFT_ACCESS_LOG_CONSOLE"
	// EnvAccessLogSlowThresholdMS names the process environment variable that controls slow access-log visibility.
	EnvAccessLogSlowThresholdMS   = "GRAFT_ACCESS_LOG_SLOW_THRESHOLD_MS"
	defaultAccessLogSlowThreshold = 1000 * time.Millisecond
)

// LogFormat describes the runtime encoder format selected for zap output.
type LogFormat string

const (
	// LogFormatAuto lets the runtime choose console for local-like environments and JSON otherwise.
	LogFormatAuto LogFormat = "auto"
	// LogFormatConsole selects zap console encoding.
	LogFormatConsole LogFormat = "console"
	// LogFormatJSON selects zap JSON encoding.
	LogFormatJSON LogFormat = "json"
)

// LogColor describes whether console log levels should include ANSI color.
type LogColor string

const (
	// LogColorAuto enables colors only for local-like console output.
	LogColorAuto LogColor = "auto"
	// LogColorAlways enables ANSI colors for console output.
	LogColorAlways LogColor = "always"
	// LogColorNever disables ANSI colors.
	LogColorNever LogColor = "never"
)

// GinMode describes the Gin framework mode selected before engine creation.
type GinMode string

const (
	// GinModeAuto lets the runtime choose debug, test, or release from the app environment.
	GinModeAuto GinMode = "auto"
	// GinModeDebug selects Gin debug mode.
	GinModeDebug GinMode = "debug"
	// GinModeRelease selects Gin release mode.
	GinModeRelease GinMode = "release"
	// GinModeTest selects Gin test mode.
	GinModeTest GinMode = "test"
)

// AccessLogConsolePolicy controls whether request facts are emitted to the process logger.
type AccessLogConsolePolicy string

const (
	// AccessLogConsoleAuto lets the runtime choose a quiet console policy from the app environment.
	AccessLogConsoleAuto AccessLogConsolePolicy = "auto"
	// AccessLogConsoleAlways emits every access log to the process logger.
	AccessLogConsoleAlways AccessLogConsolePolicy = "always"
	// AccessLogConsoleNever suppresses process-log emission while keeping persistence.
	AccessLogConsoleNever AccessLogConsolePolicy = "never"
	// AccessLogConsoleErrorOnly emits only error or slow access logs to the process logger.
	AccessLogConsoleErrorOnly AccessLogConsolePolicy = "error_only"
)

// Config 包含服务启动前一次性解析并校验的运行时配置快照。
//
// core 会把该快照作为只读依赖注入给运行时与模块，避免后续流程再隐式读取环境变量。
type Config struct {
	App       AppConfig
	HTTP      HTTPConfig
	HTTPX     HTTPXConfig
	Audit     AuditConfig
	Docs      DocsConfig
	Modules   ModulesConfig
	Database  DatabaseConfig
	Redis     RedisConfig
	Log       LogConfig
	Runtime   RuntimeConfig
	I18n      I18nConfig
	Auth      AuthConfig
	Container ContainerConfig
}

// AppConfig 描述进程级应用标识配置。
type AppConfig struct {
	Name string
	Env  string
}

// HTTPConfig 控制 core 持有的公开 HTTP 监听配置。
type HTTPConfig struct {
	Addr string
}

// HTTPXConfig 描述 core-owned httpx 运行时配置。
type HTTPXConfig struct {
	AccessLogRetention       time.Duration
	AccessLogConsole         AccessLogConsolePolicy
	AccessLogSlowThresholdMS int64
	WebSocketAllowedOrigins  []string
}

// AuditConfig describes audit-module-owned runtime policy configuration.
type AuditConfig struct {
	LogRetention time.Duration
}

// DocsConfig 控制 OpenAPI 文档与文档页面的公开策略。
type DocsConfig struct {
	Enabled bool
}

// ModulesConfig 描述 compile-time modules 在当前运行时的启用集合。
//
// 空集合表示“不做过滤，启用全部已编译模块”；非空时仅启用列出的模块。
type ModulesConfig struct {
	Enabled []string
}

// DatabaseConfig 描述 Ent 与 Atlas 共用的 PostgreSQL 连接配置。
type DatabaseConfig struct {
	Driver          string
	URL             string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
}

// RedisConfig 描述 core 服务与模块共享的 Redis 连接配置。
type RedisConfig struct {
	Addr            string
	Password        string
	DB              int
	PoolSize        int
	MinIdleConns    int
	MaxIdleConns    int
	MaxActiveConns  int
	PoolTimeout     time.Duration
	ConnMaxIdleTime time.Duration
	ConnMaxLifetime time.Duration
}

// LogConfig 描述日志核心服务接入后的日志行为配置。
type LogConfig struct {
	Level           string
	Format          LogFormat
	Color           LogColor
	AppLogPersist   bool
	AppLogRetention time.Duration
}

// RuntimeConfig 描述 core runtime 启动前必须冻结的进程级框架行为。
type RuntimeConfig struct {
	GinMode GinMode
}

// I18nConfig 描述平台级语言解析与消息回退配置。
type I18nConfig struct {
	DefaultLocale    string
	FallbackLocale   string
	SupportedLocales []string
}

// AuthConfig 描述认证模块和 HTTP 会话相关的最小稳定配置。
//
// 该配置只保留 token 和 refresh cookie 所需的基础参数，不承载 OAuth、SSO、MFA 或缓存策略。
type AuthConfig struct {
	AccessTokenTTL        time.Duration
	RefreshTokenTTL       time.Duration
	JWTSecret             string
	SigningKey            string
	RefreshCookieName     string
	RefreshCookieSecure   bool
	RefreshCookieSameSite string
	RefreshCookiePath     string
}

// ContainerConfig 描述容器管理模块的启动期进程配置。
//
// 运行期系统配置仍拥有 feature gate 的有效值；这些字段只提供无法通过当前
// SystemConfigResolver 表达的 adapter、endpoint 和日志上限启动默认值。
type ContainerConfig struct {
	Runtime                 string
	DockerEndpoint          string
	LogsDefaultTail         int
	LogsMaxTail             int
	RuntimeEnabled          bool
	DangerousActionsEnabled bool
	ShellEnabled            bool
}

// Load 按“真实环境变量优先、.env 兜底”的顺序加载配置并返回校验后的快照。
//
// 失败语义：
//   - 当显式指定的 `GRAFT_ENV_FILE` 无法读取时直接返回错误，避免启动时误用过期默认值。
// 当最终配置不满足运行时最小要求时返回验证错误。
func Load() (*Config, error) {
	if err := loadDotenv(); err != nil {
		return nil, err
	}

	reader := viper.New()
	reader.SetEnvPrefix("GRAFT")
	reader.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	reader.AutomaticEnv()

	setDefaults(reader)

	cfg := &Config{
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
			GinMode: GinMode(reader.GetString("gin.mode")),
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

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// DefaultDiskUsagePath resolves the runtime disk root for the current GOOS.
func DefaultDiskUsagePath(goos string) string {
	return DefaultDiskUsagePathForGOOS(goos, os.Getenv)
}

// DefaultDiskUsagePathForGOOS resolves the runtime disk root for a specific GOOS.
func DefaultDiskUsagePathForGOOS(goos string, lookupEnv func(string) string) string {
	if goos != "windows" {
		return "/"
	}

	if lookupEnv == nil {
		lookupEnv = func(string) string { return "" }
	}

	drive := strings.TrimSpace(lookupEnv("SystemDrive"))
	if drive == "" {
		drive = "C:"
	}
	if !strings.HasSuffix(drive, "\\") {
		drive += "\\"
	}

	return drive
}

// Validate 校验配置是否足以让服务以确定方式启动。
//
// 该方法只验证 core 当前明确依赖的约束，不负责探测数据库或 Redis 的连通性；
// 这些外部资源的真实可用性由对应资源构造阶段继续确认。
func (c *Config) Validate() error {
	if c == nil {
		return errors.New("config is required")
	}

	validators := []func(*Config) error{
		validateAppConfig,
		validateHTTPConfig,
		validateHTTPXConfig,
		validateAuditConfig,
		validateLogConfig,
		validateRuntimeConfig,
		validateModulesConfig,
		validateDatabaseConfig,
		validateRedisConfig,
		validateI18nConfig,
		validateAuthConfig,
		validateContainerConfig,
	}
	for _, validate := range validators {
		if err := validate(c); err != nil {
			return err
		}
	}
	return nil
}

func validateAppConfig(c *Config) error {
	if strings.TrimSpace(c.App.Name) == "" {
		return errors.New("GRAFT_APP_NAME is required")
	}

	return nil
}

func validateHTTPConfig(c *Config) error {
	if strings.TrimSpace(c.HTTP.Addr) == "" {
		return errors.New("GRAFT_HTTP_ADDR is required")
	}

	return nil
}

// validateHTTPXConfig 验证 HTTPX 配置。检查访问日志保留时间和慢查询阈值大于零，规范化并验证访问日志控制台策略为支持的值之一，并规范化 WebSocket 允许来源列表。
func validateHTTPXConfig(c *Config) error {
	if c.HTTPX.AccessLogRetention <= 0 {
		return errors.New("GRAFT_HTTPX_ACCESS_LOG_RETENTION must be greater than zero")
	}
	c.HTTPX.AccessLogConsole = AccessLogConsolePolicy(strings.ToLower(strings.TrimSpace(string(c.HTTPX.AccessLogConsole))))
	if c.HTTPX.AccessLogConsole == "" {
		c.HTTPX.AccessLogConsole = AccessLogConsoleAuto
	}
	switch c.HTTPX.AccessLogConsole {
	case AccessLogConsoleAuto, AccessLogConsoleAlways, AccessLogConsoleNever, AccessLogConsoleErrorOnly:
	default:
		return fmt.Errorf("unsupported GRAFT_ACCESS_LOG_CONSOLE value %q", c.HTTPX.AccessLogConsole)
	}
	if c.HTTPX.AccessLogSlowThresholdMS <= 0 {
		return errors.New("GRAFT_ACCESS_LOG_SLOW_THRESHOLD_MS must be greater than zero")
	}
	c.HTTPX.WebSocketAllowedOrigins = normalizeStringList(c.HTTPX.WebSocketAllowedOrigins)

	return nil
}

func validateAuditConfig(c *Config) error {
	if c.Audit.LogRetention <= 0 {
		return errors.New("GRAFT_AUDIT_LOG_RETENTION must be greater than zero")
	}

	return nil
}

func validateLogConfig(c *Config) error {
	c.Log.Format = LogFormat(strings.ToLower(strings.TrimSpace(string(c.Log.Format))))
	if c.Log.Format == "" {
		c.Log.Format = LogFormatAuto
	}
	switch c.Log.Format {
	case LogFormatAuto, LogFormatConsole, LogFormatJSON:
	default:
		return fmt.Errorf("unsupported GRAFT_LOG_FORMAT value %q", c.Log.Format)
	}

	c.Log.Color = LogColor(strings.ToLower(strings.TrimSpace(string(c.Log.Color))))
	if c.Log.Color == "" {
		c.Log.Color = LogColorAuto
	}
	switch c.Log.Color {
	case LogColorAuto, LogColorAlways, LogColorNever:
	default:
		return fmt.Errorf("unsupported GRAFT_LOG_COLOR value %q", c.Log.Color)
	}

	if !c.Log.AppLogPersist {
		return nil
	}
	if c.Log.AppLogRetention <= 0 {
		return errors.New("GRAFT_LOG_APP_LOG_RETENTION must be greater than zero")
	}

	return nil
}

func validateRuntimeConfig(c *Config) error {
	c.Runtime.GinMode = GinMode(strings.ToLower(strings.TrimSpace(string(c.Runtime.GinMode))))
	if c.Runtime.GinMode == "" {
		c.Runtime.GinMode = GinModeAuto
	}
	switch c.Runtime.GinMode {
	case GinModeAuto, GinModeDebug, GinModeRelease, GinModeTest:
		return nil
	default:
		return fmt.Errorf("unsupported GRAFT_GIN_MODE value %q", c.Runtime.GinMode)
	}
}

func validateModulesConfig(c *Config) error {
	normalized, seen := normalizeModuleList(c.Modules.Enabled)
	c.Modules.Enabled = normalized

	for _, moduleID := range normalized {
		if _, ok := seen[moduleID]; !ok {
			return fmt.Errorf("invalid module id %q", moduleID)
		}
	}

	return nil
}

func validateDatabaseConfig(c *Config) error {
	if strings.TrimSpace(c.Database.Driver) != defaultDatabaseDriver {
		return fmt.Errorf("unsupported database driver %q: only postgres is supported", c.Database.Driver)
	}
	if strings.TrimSpace(c.Database.URL) == "" {
		return errors.New("GRAFT_DATABASE_URL is required")
	}
	if c.Database.MaxOpenConns <= 0 {
		return errors.New("GRAFT_DATABASE_MAX_OPEN_CONNS must be greater than zero")
	}
	if c.Database.MaxIdleConns < 0 {
		return errors.New("GRAFT_DATABASE_MAX_IDLE_CONNS must be greater than or equal to zero")
	}
	if c.Database.ConnMaxLifetime < 0 {
		return errors.New("GRAFT_DATABASE_CONN_MAX_LIFETIME must be greater than or equal to zero")
	}
	if c.Database.ConnMaxIdleTime < 0 {
		return errors.New("GRAFT_DATABASE_CONN_MAX_IDLE_TIME must be greater than or equal to zero")
	}

	return nil
}

func validateRedisConfig(c *Config) error {
	if strings.TrimSpace(c.Redis.Addr) == "" {
		return errors.New("GRAFT_REDIS_ADDR is required")
	}
	if c.Redis.DB < 0 {
		return errors.New("GRAFT_REDIS_DB must be greater than or equal to zero")
	}
	if c.Redis.PoolSize < 0 {
		return errors.New("GRAFT_REDIS_POOL_SIZE must be greater than or equal to zero")
	}
	if c.Redis.MinIdleConns < 0 {
		return errors.New("GRAFT_REDIS_MIN_IDLE_CONNS must be greater than or equal to zero")
	}
	if c.Redis.MaxIdleConns < 0 {
		return errors.New("GRAFT_REDIS_MAX_IDLE_CONNS must be greater than or equal to zero")
	}
	if c.Redis.MaxActiveConns < 0 {
		return errors.New("GRAFT_REDIS_MAX_ACTIVE_CONNS must be greater than or equal to zero")
	}
	if c.Redis.PoolTimeout < 0 {
		return errors.New("GRAFT_REDIS_POOL_TIMEOUT must be greater than or equal to zero")
	}
	if c.Redis.ConnMaxIdleTime < 0 {
		return errors.New("GRAFT_REDIS_CONN_MAX_IDLE_TIME must be greater than or equal to zero")
	}
	if c.Redis.ConnMaxLifetime < 0 {
		return errors.New("GRAFT_REDIS_CONN_MAX_LIFETIME must be greater than or equal to zero")
	}

	return nil
}

func validateI18nConfig(c *Config) error {
	defaultLocaleValue := strings.TrimSpace(c.I18n.DefaultLocale)
	if defaultLocaleValue == "" {
		return errors.New("GRAFT_I18N_DEFAULT_LOCALE is required")
	}
	fallbackLocaleValue := strings.TrimSpace(c.I18n.FallbackLocale)
	if fallbackLocaleValue == "" {
		return errors.New("GRAFT_I18N_FALLBACK_LOCALE is required")
	}

	c.I18n.DefaultLocale = defaultLocaleValue
	c.I18n.FallbackLocale = fallbackLocaleValue

	normalizedLocales, supportedLocales := normalizeLocaleList(c.I18n.SupportedLocales)
	c.I18n.SupportedLocales = normalizedLocales
	if len(c.I18n.SupportedLocales) == 0 {
		return errors.New("GRAFT_I18N_SUPPORTED_LOCALES must include at least one locale")
	}
	if _, ok := supportedLocales[defaultLocaleValue]; !ok {
		return errors.New("GRAFT_I18N_DEFAULT_LOCALE must be listed in GRAFT_I18N_SUPPORTED_LOCALES")
	}
	if _, ok := supportedLocales[fallbackLocaleValue]; !ok {
		return errors.New("GRAFT_I18N_FALLBACK_LOCALE must be listed in GRAFT_I18N_SUPPORTED_LOCALES")
	}
	for _, locale := range []string{defaultLocale, defaultSecondaryLocale} {
		if _, ok := supportedLocales[locale]; !ok {
			return fmt.Errorf("GRAFT_I18N_SUPPORTED_LOCALES must include %q", locale)
		}
	}

	return nil
}

func normalizeLocaleList(locales []string) ([]string, map[string]struct{}) {
	items := make([]string, 0, len(locales))
	seen := make(map[string]struct{}, len(locales))

	for _, raw := range locales {
		locale := strings.TrimSpace(raw)
		if locale == "" {
			continue
		}
		if _, ok := seen[locale]; ok {
			continue
		}

		seen[locale] = struct{}{}
		items = append(items, locale)
	}

	return items, seen
}

func normalizeModuleList(modules []string) ([]string, map[string]struct{}) {
	items := make([]string, 0, len(modules))
	seen := make(map[string]struct{}, len(modules))

	for _, raw := range modules {
		moduleID := strings.TrimSpace(raw)
		if moduleID == "" {
			continue
		}
		if _, ok := seen[moduleID]; ok {
			continue
		}

		seen[moduleID] = struct{}{}
		items = append(items, moduleID)
	}

	return items, seen
}

func validateAuthConfig(c *Config) error {
	if c.Auth.AccessTokenTTL <= 0 {
		return errors.New("GRAFT_AUTH_ACCESS_TOKEN_TTL must be greater than zero")
	}
	if c.Auth.RefreshTokenTTL <= 0 {
		return errors.New("GRAFT_AUTH_REFRESH_TOKEN_TTL must be greater than zero")
	}
	if strings.TrimSpace(c.Auth.JWTSecret) == "" && strings.TrimSpace(c.Auth.SigningKey) == "" {
		return errors.New("GRAFT_AUTH_JWT_SECRET or GRAFT_AUTH_SIGNING_KEY is required")
	}
	if err := validateRefreshCookiePolicy(c.Auth); err != nil {
		return err
	}
	if strings.TrimSpace(c.Auth.RefreshCookieName) == "" {
		return errors.New("GRAFT_AUTH_REFRESH_COOKIE_NAME is required")
	}
	if strings.TrimSpace(c.Auth.RefreshCookiePath) == "" {
		return errors.New("GRAFT_AUTH_REFRESH_COOKIE_PATH is required")
	}

	return nil
}

// validateContainerConfig validates container configuration fields.
// Returns nil if the configuration is valid, or an error describing the validation failure.
func validateContainerConfig(c *Config) error {
	c.Container.Runtime = strings.TrimSpace(c.Container.Runtime)
	if c.Container.Runtime == "" {
		c.Container.Runtime = defaultContainerRuntime
	}
	switch c.Container.Runtime {
	case defaultContainerRuntime, "docker":
	default:
		return fmt.Errorf("unsupported GRAFT_OPS_CONTAINER_RUNTIME value %q", c.Container.Runtime)
	}
	if strings.TrimSpace(c.Container.DockerEndpoint) == "" {
		return errors.New("GRAFT_OPS_CONTAINER_DOCKER_ENDPOINT is required")
	}
	if c.Container.LogsDefaultTail <= 0 {
		return errors.New("GRAFT_OPS_CONTAINER_LOGS_DEFAULT_TAIL must be greater than zero")
	}
	if c.Container.LogsMaxTail <= 0 {
		return errors.New("GRAFT_OPS_CONTAINER_LOGS_MAX_TAIL must be greater than zero")
	}
	if c.Container.LogsDefaultTail > c.Container.LogsMaxTail {
		return errors.New("GRAFT_OPS_CONTAINER_LOGS_DEFAULT_TAIL must be less than or equal to GRAFT_OPS_CONTAINER_LOGS_MAX_TAIL")
	}
	if c.Container.ShellEnabled && len(c.HTTPX.WebSocketAllowedOrigins) == 0 {
		return errors.New("GRAFT_HTTPX_WEBSOCKET_ALLOWED_ORIGINS is required when GRAFT_OPS_CONTAINER_SHELL_ENABLED is true")
	}
	return nil
}

func validateRefreshCookiePolicy(cfg AuthConfig) error {
	switch strings.ToLower(strings.TrimSpace(cfg.RefreshCookieSameSite)) {
	case "lax", "strict":
		return nil
	case "none":
		if !cfg.RefreshCookieSecure {
			return errors.New("GRAFT_AUTH_REFRESH_COOKIE_SECURE must be true when GRAFT_AUTH_REFRESH_COOKIE_SAME_SITE is none")
		}
		return nil
	default:
		return fmt.Errorf("unsupported GRAFT_AUTH_REFRESH_COOKIE_SAME_SITE value %q", cfg.RefreshCookieSameSite)
	}
}

func loadDotenv() error {
	if explicit := strings.TrimSpace(os.Getenv("GRAFT_ENV_FILE")); explicit != "" {
		if err := godotenv.Load(explicit); err != nil {
			return fmt.Errorf("load %s: %w", explicit, err)
		}
		return nil
	}

	dotenvPath, err := findDotenvPath()
	if err != nil {
		return err
	}
	if dotenvPath != "" {
		return godotenv.Load(dotenvPath)
	}

	return nil
}

func findDotenvPath() (string, error) {
	workingDir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("resolve working directory: %w", err)
	}

	for _, dir := range dotenvSearchDirs(workingDir) {
		for _, candidate := range []string{
			filepath.Join(dir, ".env"),
			filepath.Join(dir, "server", ".env"),
		} {
			if _, err := os.Stat(candidate); err == nil {
				return candidate, nil
			} else if err != nil && !errors.Is(err, os.ErrNotExist) {
				return "", fmt.Errorf("stat dotenv candidate %s: %w", candidate, err)
			}
		}
	}

	return "", nil
}

func dotenvSearchDirs(start string) []string {
	if strings.TrimSpace(start) == "" {
		return nil
	}

	dirs := []string{}
	current := filepath.Clean(start)
	for {
		dirs = append(dirs, current)

		if isDotenvSearchBoundary(current) {
			return dirs
		}

		parent := filepath.Dir(current)
		if parent == current {
			return dirs
		}
		current = parent
	}
}

func isDotenvSearchBoundary(dir string) bool {
	if filepath.Base(dir) == "server" {
		return true
	}

	for _, marker := range []string{".git", "server"} {
		info, err := os.Stat(filepath.Join(dir, marker))
		if err != nil {
			continue
		}
		if marker == "server" && !info.IsDir() {
			continue
		}
		return true
	}

	return false
}

// setDefaults sets default configuration values for all supported configuration keys, including environment-dependent retention durations for logging and auditing.
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

// parseLocaleList 将逗号分隔的地域字符串解析为规范化的地域值。
// 结果已去除首尾空白、空项和重复值。
func parseLocaleList(raw string) []string {
	items, _ := normalizeLocaleList(strings.Split(raw, ","))
	return items
}

// parseCommaSeparatedList parses a comma-separated string into a normalized slice, trimming whitespace, removing empty values, and deduplicating.
func parseCommaSeparatedList(raw string) []string {
	return normalizeStringList(strings.Split(raw, ","))
}

// parseModuleList parses a comma-separated string into a normalized list of module identifiers.
func parseModuleList(raw string) []string {
	items, _ := normalizeModuleList(strings.Split(raw, ","))
	return items
}

// normalizeStringList removes empty strings and deduplicates the input list while preserving the order of first occurrence.
func normalizeStringList(items []string) []string {
	normalized := make([]string, 0, len(items))
	seen := make(map[string]struct{}, len(items))
	for _, raw := range items {
		value := strings.TrimSpace(raw)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		normalized = append(normalized, value)
	}
	return normalized
}

// resolveDocsEnabled determines whether documentation should be enabled based on explicit configuration or environment default.
func resolveDocsEnabled(reader *viper.Viper) bool {
	if reader == nil {
		return defaultDocsEnabledForEnv(defaultAppEnv)
	}

	if reader.IsSet("docs.enabled") {
		return reader.GetBool("docs.enabled")
	}

	return defaultDocsEnabledForEnv(reader.GetString("app.env"))
}

func defaultDocsEnabledForEnv(env string) bool {
	switch normalizeAppEnv(env) {
	case "", "local", "development", "dev", "test":
		return true
	case "prod", "production":
		return false
	default:
		return false
	}
}

func defaultAccessLogRetentionForEnv(env string) time.Duration {
	switch normalizeAppEnv(env) {
	case "prod", "production":
		return 30 * 24 * time.Hour
	case "staging", "stage":
		return 7 * 24 * time.Hour
	case "", "local", "development", "dev", "test":
		return 3 * 24 * time.Hour
	default:
		return 7 * 24 * time.Hour
	}
}

func defaultAuditLogRetentionForEnv(env string) time.Duration {
	switch normalizeAppEnv(env) {
	case "prod", "production":
		return 180 * 24 * time.Hour
	case "staging", "stage":
		return 90 * 24 * time.Hour
	case "", "local", "development", "dev", "test":
		return 30 * 24 * time.Hour
	default:
		return 90 * 24 * time.Hour
	}
}

func defaultAppLogRetentionForEnv(env string) time.Duration {
	switch normalizeAppEnv(env) {
	case "prod", "production":
		return 14 * 24 * time.Hour
	case "staging", "stage":
		return 7 * 24 * time.Hour
	case "", "local", "development", "dev", "test":
		return 3 * 24 * time.Hour
	default:
		return 7 * 24 * time.Hour
	}
}

// ResolveLogFormat returns the concrete zap encoder format for the app environment and requested policy.
func ResolveLogFormat(appEnv string, format LogFormat) LogFormat {
	switch normalizeLogFormat(format) {
	case LogFormatConsole:
		return LogFormatConsole
	case LogFormatJSON:
		return LogFormatJSON
	default:
		if isLocalLikeEnv(appEnv) {
			return LogFormatConsole
		}
		return LogFormatJSON
	}
}

// ResolveLogColor reports whether the effective console encoder should colorize log levels.
func ResolveLogColor(appEnv string, format LogFormat, color LogColor) bool {
	if ResolveLogFormat(appEnv, format) != LogFormatConsole {
		return false
	}

	switch normalizeLogColor(color) {
	case LogColorAlways:
		return true
	case LogColorNever:
		return false
	default:
		return isLocalLikeEnv(appEnv)
	}
}

// ResolveGinMode returns the concrete Gin mode for the app environment and requested policy.
func ResolveGinMode(appEnv string, mode GinMode) GinMode {
	switch normalizeGinMode(mode) {
	case GinModeDebug:
		return GinModeDebug
	case GinModeRelease:
		return GinModeRelease
	case GinModeTest:
		return GinModeTest
	default:
		switch normalizeAppEnv(appEnv) {
		case "local", "development", "dev":
			return GinModeDebug
		case "test":
			return GinModeTest
		default:
			return GinModeRelease
		}
	}
}

// ResolveAccessLogConsolePolicy returns the concrete request-log console policy for the app environment.
func ResolveAccessLogConsolePolicy(appEnv string, policy AccessLogConsolePolicy) AccessLogConsolePolicy {
	switch normalizeAccessLogConsolePolicy(policy) {
	case AccessLogConsoleAlways:
		return AccessLogConsoleAlways
	case AccessLogConsoleNever:
		return AccessLogConsoleNever
	case AccessLogConsoleErrorOnly:
		return AccessLogConsoleErrorOnly
	default:
		switch normalizeAppEnv(appEnv) {
		case "local", "development", "dev":
			return AccessLogConsoleErrorOnly
		default:
			return AccessLogConsoleNever
		}
	}
}

func normalizeAppEnv(env string) string {
	return strings.ToLower(strings.TrimSpace(env))
}

func normalizeLogFormat(format LogFormat) LogFormat {
	switch LogFormat(strings.ToLower(strings.TrimSpace(string(format)))) {
	case LogFormatConsole:
		return LogFormatConsole
	case LogFormatJSON:
		return LogFormatJSON
	default:
		return LogFormatAuto
	}
}

func normalizeLogColor(color LogColor) LogColor {
	switch LogColor(strings.ToLower(strings.TrimSpace(string(color)))) {
	case LogColorAlways:
		return LogColorAlways
	case LogColorNever:
		return LogColorNever
	default:
		return LogColorAuto
	}
}

func normalizeGinMode(mode GinMode) GinMode {
	switch GinMode(strings.ToLower(strings.TrimSpace(string(mode)))) {
	case GinModeDebug:
		return GinModeDebug
	case GinModeRelease:
		return GinModeRelease
	case GinModeTest:
		return GinModeTest
	default:
		return GinModeAuto
	}
}

func normalizeAccessLogConsolePolicy(policy AccessLogConsolePolicy) AccessLogConsolePolicy {
	switch AccessLogConsolePolicy(strings.ToLower(strings.TrimSpace(string(policy)))) {
	case AccessLogConsoleAlways:
		return AccessLogConsoleAlways
	case AccessLogConsoleNever:
		return AccessLogConsoleNever
	case AccessLogConsoleErrorOnly:
		return AccessLogConsoleErrorOnly
	default:
		return AccessLogConsoleAuto
	}
}

func isLocalLikeEnv(env string) bool {
	switch normalizeAppEnv(env) {
	case "", "local", "development", "dev", "test":
		return true
	default:
		return false
	}
}
