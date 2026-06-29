package config

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

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
	GinMode                         GinMode
	DevAllowDirtyMigrationBootstrap bool
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
//
// Load 读取环境配置并返回经过校验的配置快照。
// 当 dotenv 载入失败或配置不满足校验要求时返回错误。
func Load() (*Config, error) {
	if err := loadDotenv(); err != nil {
		return nil, err
	}

	reader := viper.New()
	reader.SetEnvPrefix("GRAFT")
	reader.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	reader.AutomaticEnv()

	setDefaults(reader)

	cfg := readConfig(reader)

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
	if err := validateWebSocketAllowedOrigins(c.HTTPX.WebSocketAllowedOrigins); err != nil {
		return err
	}

	return nil
}

// validateAuditConfig 校验审计日志保留时长配置。
// 当 GRAFT_AUDIT_LOG_RETENTION 小于或等于 0 时返回错误；否则返回 nil。
// @returns 校验失败时返回错误，校验通过时返回 nil。
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

func defaultDevAllowDirtyMigrationBootstrapForEnv(appEnv string) bool {
	return strings.EqualFold(strings.TrimSpace(appEnv), defaultAppEnv)
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

// validateI18nConfig 校验并规范化 i18n 配置。
// 它要求默认语言、回退语言和支持语言列表都已配置，并确保默认语言、回退语言以及内置必需语言都包含在支持列表中。
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

// normalizeLocaleList 规范化语言区域列表并返回去重集合。
// 返回规范化后的区域列表及其去重映射。
func normalizeLocaleList(locales []string) ([]string, map[string]struct{}) {
	return normalizeIndexedStringList(locales)
}

// normalizeModuleList 规范化模块 ID 列表，并返回去重后的结果及索引集合。
// 返回规范化后的模块 ID 列表，以及以规范化值为键的集合。
func normalizeModuleList(modules []string) ([]string, map[string]struct{}) {
	return normalizeIndexedStringList(modules)
}

// validateAuthConfig 检查认证相关配置是否有效。
// 当访问令牌或刷新令牌的 TTL 无效，JWT 密钥缺失，刷新 Cookie 策略不合法，或刷新 Cookie 名称/路径为空时返回错误。
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

// validateRefreshCookiePolicy 验证刷新 Cookie 的 SameSite 和 Secure 组合约束。
// 当 SameSite 为 none 时，要求 Secure 为 true。
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

// defaultDocsEnabledForEnv 根据应用环境判断是否启用文档页面。
// @returns 在本地类环境或测试环境下返回 true，其他环境返回 false。
func defaultDocsEnabledForEnv(env string) bool {
	switch classifyAppEnv(env) {
	case appEnvLocalLike, appEnvTest:
		return true
	default:
		return false
	}
}

// defaultAccessLogRetentionForEnv 根据应用环境返回默认的访问日志保留时长。
// 本地类和测试环境返回 3 天，预发布环境返回 7 天，生产环境返回 30 天，其余环境返回 7 天。
func defaultAccessLogRetentionForEnv(env string) time.Duration {
	return durationByAppEnv(env, 3*24*time.Hour, 7*24*time.Hour, 30*24*time.Hour, 7*24*time.Hour)
}

// defaultAuditLogRetentionForEnv 根据应用环境返回审计日志的默认保留时长。
// 本地/测试环境为 30 天，预发布环境为 90 天，生产环境为 180 天，其它环境为 90 天。
func defaultAuditLogRetentionForEnv(env string) time.Duration {
	return durationByAppEnv(env, 30*24*time.Hour, 90*24*time.Hour, 180*24*time.Hour, 90*24*time.Hour)
}

// defaultAppLogRetentionForEnv 返回给定应用环境下的应用日志保留时长。
// 本地类和测试环境为 3 天，预发环境为 7 天，生产环境为 14 天，其它环境为 7 天。
// 返回对应环境的应用日志保留时长。
func defaultAppLogRetentionForEnv(env string) time.Duration {
	return durationByAppEnv(env, 3*24*time.Hour, 7*24*time.Hour, 14*24*time.Hour, 7*24*time.Hour)
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

// ResolveGinMode 根据应用环境和请求策略确定 Gin 的实际运行模式。
// 显式指定为 debug、release 或 test 时返回对应模式；否则根据应用环境在 debug、test 和 release 之间选择。
func ResolveGinMode(appEnv string, mode GinMode) GinMode {
	switch normalizeGinMode(mode) {
	case GinModeDebug:
		return GinModeDebug
	case GinModeRelease:
		return GinModeRelease
	case GinModeTest:
		return GinModeTest
	default:
		switch classifyAppEnv(appEnv) {
		case appEnvLocalLike:
			return GinModeDebug
		case appEnvTest:
			return GinModeTest
		default:
			return GinModeRelease
		}
	}
}

// ResolveAccessLogConsolePolicy returns the effective access-log console policy.
// 当未显式指定策略时，局部环境返回 error_only，其它环境返回 never。
func ResolveAccessLogConsolePolicy(appEnv string, policy AccessLogConsolePolicy) AccessLogConsolePolicy {
	switch normalizeAccessLogConsolePolicy(policy) {
	case AccessLogConsoleAlways:
		return AccessLogConsoleAlways
	case AccessLogConsoleNever:
		return AccessLogConsoleNever
	case AccessLogConsoleErrorOnly:
		return AccessLogConsoleErrorOnly
	default:
		switch classifyAppEnv(appEnv) {
		case appEnvLocalLike:
			return AccessLogConsoleErrorOnly
		default:
			return AccessLogConsoleNever
		}
	}
}

// normalizeAppEnv 将应用环境字符串转换为小写并去除首尾空白。
func normalizeAppEnv(env string) string {
	return strings.ToLower(strings.TrimSpace(env))
}

// normalizeLogFormat 规范化日志格式配置并返回有效值。
// 当输入匹配 `auto`、`console` 或 `json` 时返回对应值，否则返回 `auto`。
func normalizeLogFormat(format LogFormat) LogFormat {
	return normalizeStringEnum(format, LogFormatAuto, LogFormatConsole, LogFormatJSON)
}

// normalizeLogColor 规范化日志颜色策略，返回受支持的取值或默认值。
func normalizeLogColor(color LogColor) LogColor {
	return normalizeStringEnum(color, LogColorAuto, LogColorAlways, LogColorNever)
}

// normalizeGinMode 将输入规范化为支持的 Gin 模式值。
// @returns 规范化后的 GinMode；当输入不匹配任何支持值时返回 `auto`。
func normalizeGinMode(mode GinMode) GinMode {
	return normalizeStringEnum(mode, GinModeAuto, GinModeDebug, GinModeRelease, GinModeTest)
}

// normalizeAccessLogConsolePolicy 将访问日志控制台策略归一为受支持的取值。
// 无法识别的值会回退为 `auto`。
func normalizeAccessLogConsolePolicy(policy AccessLogConsolePolicy) AccessLogConsolePolicy {
	return normalizeStringEnum(policy, AccessLogConsoleAuto, AccessLogConsoleAlways, AccessLogConsoleNever, AccessLogConsoleErrorOnly)
}

// isLocalLikeEnv 判断应用环境是否属于本地开发类或测试类环境。
func isLocalLikeEnv(env string) bool {
	return classifyAppEnv(env) == appEnvLocalLike || classifyAppEnv(env) == appEnvTest
}

type appEnvClass uint8

const (
	appEnvOther appEnvClass = iota
	appEnvLocalLike
	appEnvTest
	appEnvStaging
	appEnvProduction
)

// normalizeIndexedStringList 规范化字符串列表并返回去重索引集。
//
// @param items 待规范化的字符串列表。
// @returns 规范化后的字符串列表，以及以规范化值为键的集合。
func normalizeIndexedStringList(items []string) ([]string, map[string]struct{}) {
	normalized := normalizeStringList(items)
	seen := make(map[string]struct{}, len(normalized))
	for _, item := range normalized {
		seen[item] = struct{}{}
	}
	return normalized, seen
}

// durationByAppEnv 根据应用环境分类返回对应的时长。
// 本地类和测试环境返回 localLike，预发环境返回 staging，生产环境返回 production，其它环境返回 fallback。
func durationByAppEnv(env string, localLike, staging, production, fallback time.Duration) time.Duration {
	switch classifyAppEnv(env) {
	case appEnvLocalLike, appEnvTest:
		return localLike
	case appEnvStaging:
		return staging
	case appEnvProduction:
		return production
	default:
		return fallback
	}
}

// classifyAppEnv 将应用环境归类为本地类、测试、预发布、生产或其他类别。
func classifyAppEnv(env string) appEnvClass {
	switch normalizeAppEnv(env) {
	case "", "local", "development", "dev":
		return appEnvLocalLike
	case "test":
		return appEnvTest
	case "staging", "stage":
		return appEnvStaging
	case "prod", "production":
		return appEnvProduction
	default:
		return appEnvOther
	}
}

// normalizeStringEnum 规范化字符串枚举值，并在不匹配允许值时返回回退值。
// 它会对输入进行去首尾空白和小写化处理后再进行匹配。
func normalizeStringEnum[T ~string](raw T, fallback T, allowed ...T) T {
	value := T(strings.ToLower(strings.TrimSpace(string(raw))))
	for _, candidate := range allowed {
		if value == candidate {
			return candidate
		}
	}
	return fallback
}
