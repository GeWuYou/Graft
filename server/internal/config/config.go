package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
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
	maxDurationMilliseconds         = int64((1<<63 - 1) / int64(time.Millisecond))
	defaultSupported                = "zh-CN,en-US"
	defaultAccessTokenTTL           = 15 * time.Minute
	defaultRefreshTokenTTL          = 7 * 24 * time.Hour
	defaultRefreshCookieName        = "graft_refresh_token"
	defaultRefreshCookiePath        = "/"
	defaultRefreshCookieSameSite    = "lax"
	defaultContainerRuntime         = "first-adapter"
	defaultContainerDockerEndpoint  = "unix:///var/run/docker.sock"
	defaultBackupArtifactRoot       = "/var/lib/graft/backups"
	defaultContainerLogsDefaultTail = 200
	defaultContainerLogsMaxTail     = 2000
	defaultRealtimeAllowedOrigins   = ""
	defaultMCPConfirmationTokenTTL  = 5 * time.Minute
	defaultMCPSessionTimeout        = 15 * time.Minute
	defaultMCPRequestTimeout        = 30 * time.Second
	defaultMCPMaxRequestBytes       = int64(1 << 20)
	defaultMCPMaxSessions           = 64
	defaultMCPMaxConcurrentRequests = 32
)

const (
	// EnvAppEnv 是选择运行环境的进程环境变量名。
	EnvAppEnv = "GRAFT_APP_ENV"
	// EnvLogLevel 是选择 zap 日志级别的进程环境变量名。
	EnvLogLevel = "GRAFT_LOG_LEVEL"
	// EnvLogFormat 是选择 console 或 JSON 输出格式的进程环境变量名。
	EnvLogFormat = "GRAFT_LOG_FORMAT"
	// EnvLogColor 是控制日志级别 ANSI 颜色的进程环境变量名。
	EnvLogColor = "GRAFT_LOG_COLOR"
	// EnvLogCategories 是定义受控日志类别开关的聚合环境变量名。
	EnvLogCategories = "GRAFT_LOG_CATEGORIES"
	// EnvGinMode 是选择 Gin 运行模式的进程环境变量名。
	EnvGinMode = "GRAFT_GIN_MODE"
	// EnvAccessLogConsole 是控制访问日志是否输出到进程日志的环境变量名。
	EnvAccessLogConsole = "GRAFT_ACCESS_LOG_CONSOLE"
	// EnvAccessLogSlowThresholdMS 是控制慢访问日志阈值的进程环境变量名。
	EnvAccessLogSlowThresholdMS = "GRAFT_ACCESS_LOG_SLOW_THRESHOLD_MS"
	// EnvAccessLogPersistTimeoutMS 是控制访问日志持久化 deadline 的环境变量名。
	EnvAccessLogPersistTimeoutMS   = "GRAFT_ACCESS_LOG_PERSIST_TIMEOUT_MS"
	defaultAccessLogSlowThreshold  = 1000 * time.Millisecond
	defaultAccessLogPersistTimeout = 1000 * time.Millisecond
)

// LogFormat 描述 zap 输出使用的运行时编码格式。
type LogFormat string

const (
	// LogFormatAuto 让运行时在本地类环境使用 console，其它环境使用 JSON。
	LogFormatAuto LogFormat = "auto"
	// LogFormatConsole 选择 zap console 编码。
	LogFormatConsole LogFormat = "console"
	// LogFormatJSON 选择 zap JSON 编码。
	LogFormatJSON LogFormat = "json"
)

// LogColor 描述 console 日志级别是否包含 ANSI 颜色。
type LogColor string

const (
	// LogColorAuto 仅为本地类环境的 console 输出启用颜色。
	LogColorAuto LogColor = "auto"
	// LogColorAlways 为 console 输出启用 ANSI 颜色。
	LogColorAlways LogColor = "always"
	// LogColorNever 禁用 ANSI 颜色。
	LogColorNever LogColor = "never"
)

// GinMode 描述创建 Gin engine 前确定的框架运行模式。
type GinMode string

const (
	// GinModeAuto 让运行时根据应用环境选择 debug、test 或 release。
	GinModeAuto GinMode = "auto"
	// GinModeDebug 选择 Gin debug 模式。
	GinModeDebug GinMode = "debug"
	// GinModeRelease 选择 Gin release 模式。
	GinModeRelease GinMode = "release"
	// GinModeTest 选择 Gin test 模式。
	GinModeTest GinMode = "test"
)

// AccessLogConsolePolicy 控制请求事实是否输出到进程日志，同时不改变持久化策略。
type AccessLogConsolePolicy string

const (
	// AccessLogConsoleAuto 让运行时根据应用环境选择较安静的 console 策略。
	AccessLogConsoleAuto AccessLogConsolePolicy = "auto"
	// AccessLogConsoleAlways 将每条访问日志输出到进程日志。
	AccessLogConsoleAlways AccessLogConsolePolicy = "always"
	// AccessLogConsoleNever 抑制进程日志输出，但保留访问日志持久化。
	AccessLogConsoleNever AccessLogConsolePolicy = "never"
	// AccessLogConsoleErrorOnly 仅将错误或慢访问日志输出到进程日志。
	AccessLogConsoleErrorOnly AccessLogConsolePolicy = "error_only"
)

// Config 包含服务启动前一次性解析并校验的运行时配置快照。
//
// core 会把该快照作为只读依赖注入给运行时与模块，避免后续流程再隐式读取环境变量。
type Config struct {
	// DotenvPath 是本次配置加载实际采用的 dotenv 路径；运行时诊断只能消费该快照，避免重复解析环境得到漂移来源。
	DotenvPath          string
	App                 AppConfig
	HTTP                HTTPConfig
	HTTPX               HTTPXConfig
	CredentialVault     CredentialVaultConfig
	Audit               AuditConfig
	Docs                DocsConfig
	Modules             ModulesConfig
	Database            DatabaseConfig
	Redis               RedisConfig
	Log                 LogConfig
	Runtime             RuntimeConfig
	I18n                I18nConfig
	Auth                AuthConfig
	MCP                 MCPConfig
	Container           ContainerConfig
	RegistryCredentials RegistryCredentialSourceConfig
	EnrollmentSecurity  EnrollmentSecurityConfig
	Backup              BackupConfig
	Project             ProjectConfig
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
	AccessLogConsole          AccessLogConsolePolicy
	AccessLogSlowThresholdMS  int64
	AccessLogPersistTimeoutMS int64
	WebSocketAllowedOrigins   []string
	AgentTLS                  AgentTLSConfig
	AgentBootstrapTLS         AgentBootstrapTLSConfig
}

// AgentTLSConfig 描述专用 Agent mTLS 监听器的部署期证书材料位置。
// 私钥和 CA 内容由部署层挂载，运行时只消费绝对文件路径。
type AgentTLSConfig struct {
	Enabled         bool
	Addr            string
	CertificateFile string
	KeyFile         string
	ClientCAFile    string
}

// AgentBootstrapTLSConfig 描述首次 Agent 证书签发的专用 server-authenticated TLS listener。
// 它不接受客户端证书，因新 Agent 尚未拥有 Vault 签发的身份材料。
type AgentBootstrapTLSConfig struct {
	Enabled         bool
	Addr            string
	CertificateFile string
	KeyFile         string
}

// CredentialVaultConfig 描述 Credential Vault 的非秘密接入信息。
// 认证材料必须由部署机器身份提供，禁止通过此配置传递 token、私钥或 PEM。
type CredentialVaultConfig struct {
	Enabled          bool
	Backend          string
	Address          string
	CAFile           string
	Namespace        string
	AuthMount        string
	AuthRole         string
	AuthRoleIDFile   string
	AuthSecretIDFile string
	PKIMount         string
	PKIRole          string
	TrustBundleRef   string
}

// AuditConfig 预留 core 提供的审计启动配置边界。
type AuditConfig struct{}

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
	Level         string
	Format        LogFormat
	Color         LogColor
	Categories    string
	AppLogPersist bool
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

// MCPConfig 描述产品 MCP runtime 的进程级 transport 开关与安全生命周期参数。
//
// 这些值在模块和 System Config 尚未可用时决定是否装配公开 `/mcp` 入口，
// 因此属于部署配置而不是可热更新的管理员策略。
type MCPConfig struct {
	Enabled               bool
	ConfirmationTokenTTL  time.Duration
	SessionTimeout        time.Duration
	RequestTimeout        time.Duration
	MaxRequestBytes       int64
	MaxSessions           int
	MaxConcurrentRequests int
}

// ContainerConfig 描述容器管理模块的部署配置。
//
// Provider 和 endpoint 由部署环境决定；管理员运行时策略由 System Config 拥有。
type ContainerConfig struct {
	Runtime        string
	DockerEndpoint string
}

// RegistryCredentialSourceConfig 描述 core 读取 Registry 凭据文件的位置。
// 文件内容只由 CredentialProvider 消费，不能进入 Config 快照之外的控制面。
type RegistryCredentialSourceConfig struct {
	File string
}

// EnrollmentSecurityConfig 描述安装级 Agent enrollment 秘密文件的位置。
// Pepper 内容只能由受控的 security provider 读取，不能作为模块配置或持久化事实。
type EnrollmentSecurityConfig struct {
	PepperFile string
}

// BackupConfig 描述 Backup 模块可写入的受控工件根目录。
//
// 该目录由部署层挂载和权限控制，不能由 HTTP 请求或 System Config 覆盖。
type BackupConfig struct {
	ArtifactRoot string
}

// ProjectConfig 描述随 core 配置快照加载的 project 模块诊断开关。
type ProjectConfig struct {
	LogDebug           bool
	ManagedCreateDebug bool
}

// Load 按“真实环境变量优先、.env 兜底”的顺序加载配置并返回校验后的快照。
//
// 失败语义：
//   - 当显式指定的 `GRAFT_ENV_FILE` 无法读取时直接返回错误，避免启动时误用过期默认值。
//
// Load 读取环境配置并返回经过校验的配置快照。
// 当 dotenv 载入失败或配置不满足校验要求时返回错误。
func Load() (*Config, error) {
	dotenvPath, err := loadDotenv()
	if err != nil {
		return nil, err
	}

	reader := viper.New()
	reader.SetEnvPrefix("GRAFT")
	reader.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	reader.AutomaticEnv()

	setDefaults(reader)

	cfg := readConfig(reader)
	cfg.DotenvPath = dotenvPath

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// DefaultDiskUsagePath 根据当前 GOOS 解析运行时磁盘根目录。
func DefaultDiskUsagePath(goos string) string {
	return DefaultDiskUsagePathForGOOS(goos, os.Getenv)
}

// DefaultDiskUsagePathForGOOS 根据指定 GOOS 解析运行时磁盘根目录。
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
		validateCredentialVaultConfig,
		validateAuditConfig,
		validateLogConfig,
		validateRuntimeConfig,
		validateModulesConfig,
		validateDatabaseConfig,
		validateRedisConfig,
		validateI18nConfig,
		validateAuthConfig,
		validateMCPConfig,
		validateContainerConfig,
		validateRegistryCredentialSourceConfig,
		validateEnrollmentSecurityConfig,
		validateBackupConfig,
	}
	for _, validate := range validators {
		if err := validate(c); err != nil {
			return err
		}
	}
	return nil
}

func validateRegistryCredentialSourceConfig(c *Config) error {
	file := strings.TrimSpace(c.RegistryCredentials.File)
	if file == "" {
		return nil
	}
	if !filepath.IsAbs(file) {
		return errors.New("GRAFT_REGISTRY_CREDENTIALS_FILE must be an absolute path")
	}
	c.RegistryCredentials.File = filepath.Clean(file)
	return nil
}

func validateEnrollmentSecurityConfig(c *Config) error {
	file := strings.TrimSpace(c.EnrollmentSecurity.PepperFile)
	if file == "" {
		return nil
	}
	if !filepath.IsAbs(file) {
		return errors.New("GRAFT_ENROLLMENT_PEPPER_FILE must be an absolute path")
	}
	c.EnrollmentSecurity.PepperFile = filepath.Clean(file)
	return nil
}

func validateBackupConfig(c *Config) error {
	root := strings.TrimSpace(c.Backup.ArtifactRoot)
	if root == "" {
		root = defaultBackupArtifactRoot
	}
	if !filepath.IsAbs(root) {
		return errors.New("GRAFT_BACKUP_ARTIFACT_ROOT must be an absolute path")
	}
	c.Backup.ArtifactRoot = filepath.Clean(root)
	return nil
}

func validateAppConfig(c *Config) error {
	// 应用名参与日志、模块标识和运行时诊断，禁止以空白值启动以避免产生不可追踪实例。
	if strings.TrimSpace(c.App.Name) == "" {
		return errors.New("GRAFT_APP_NAME is required")
	}

	return nil
}

// validateHTTPConfig 验证 HTTP 服务监听地址已配置。
func validateHTTPConfig(c *Config) error {
	if strings.TrimSpace(c.HTTP.Addr) == "" {
		return errors.New("GRAFT_HTTP_ADDR is required")
	}

	return nil
}

// 验证访问日志控制台策略、慢请求阈值和 WebSocket 允许来源列表。
func validateHTTPXConfig(c *Config) error {
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
	if c.HTTPX.AccessLogPersistTimeoutMS <= 0 {
		return errors.New("GRAFT_ACCESS_LOG_PERSIST_TIMEOUT_MS must be greater than zero")
	}
	if c.HTTPX.AccessLogPersistTimeoutMS > maxDurationMilliseconds {
		return fmt.Errorf("GRAFT_ACCESS_LOG_PERSIST_TIMEOUT_MS must be no greater than %d", maxDurationMilliseconds)
	}
	c.HTTPX.WebSocketAllowedOrigins = normalizeStringList(c.HTTPX.WebSocketAllowedOrigins)
	if err := validateWebSocketAllowedOrigins(c.HTTPX.WebSocketAllowedOrigins); err != nil {
		return err
	}
	if err := validateAgentTLSConfig(&c.HTTPX.AgentTLS); err != nil {
		return err
	}
	if err := validateAgentBootstrapTLSConfig(&c.HTTPX.AgentBootstrapTLS); err != nil {
		return err
	}

	return nil
}

func validateAgentTLSConfig(agentTLS *AgentTLSConfig) error {
	if agentTLS == nil || !agentTLS.Enabled {
		return nil
	}
	if strings.TrimSpace(agentTLS.Addr) == "" {
		return errors.New("GRAFT_HTTPX_AGENT_TLS_ADDR is required when GRAFT_HTTPX_AGENT_TLS_ENABLED is true")
	}
	for _, field := range []struct {
		name  string
		value *string
	}{
		{name: "GRAFT_HTTPX_AGENT_TLS_CERTIFICATE_FILE", value: &agentTLS.CertificateFile},
		{name: "GRAFT_HTTPX_AGENT_TLS_KEY_FILE", value: &agentTLS.KeyFile},
		{name: "GRAFT_HTTPX_AGENT_TLS_CLIENT_CA_FILE", value: &agentTLS.ClientCAFile},
	} {
		if strings.TrimSpace(*field.value) == "" || !filepath.IsAbs(*field.value) {
			return fmt.Errorf("%s must be an absolute path when GRAFT_HTTPX_AGENT_TLS_ENABLED is true", field.name)
		}
		*field.value = filepath.Clean(*field.value)
	}
	return nil
}

func validateAgentBootstrapTLSConfig(bootstrapTLS *AgentBootstrapTLSConfig) error {
	if bootstrapTLS == nil || !bootstrapTLS.Enabled {
		return nil
	}
	if strings.TrimSpace(bootstrapTLS.Addr) == "" {
		return errors.New("GRAFT_HTTPX_AGENT_BOOTSTRAP_TLS_ADDR is required when GRAFT_HTTPX_AGENT_BOOTSTRAP_TLS_ENABLED is true")
	}
	for _, field := range []struct {
		name  string
		value *string
	}{
		{name: "GRAFT_HTTPX_AGENT_BOOTSTRAP_TLS_CERTIFICATE_FILE", value: &bootstrapTLS.CertificateFile},
		{name: "GRAFT_HTTPX_AGENT_BOOTSTRAP_TLS_KEY_FILE", value: &bootstrapTLS.KeyFile},
	} {
		if strings.TrimSpace(*field.value) == "" || !filepath.IsAbs(*field.value) {
			return fmt.Errorf("%s must be an absolute path when GRAFT_HTTPX_AGENT_BOOTSTRAP_TLS_ENABLED is true", field.name)
		}
		*field.value = filepath.Clean(*field.value)
	}
	return nil
}

func validateCredentialVaultConfig(c *Config) error {
	if c == nil || !c.CredentialVault.Enabled {
		return nil
	}
	vault := &c.CredentialVault
	if strings.TrimSpace(vault.Backend) != "vault-pki" {
		return errors.New("GRAFT_CREDENTIAL_VAULT_BACKEND must be vault-pki when GRAFT_CREDENTIAL_VAULT_ENABLED is true")
	}
	for _, field := range []struct {
		name  string
		value *string
	}{
		{name: "GRAFT_CREDENTIAL_VAULT_ADDRESS", value: &vault.Address},
		{name: "GRAFT_CREDENTIAL_VAULT_CA_FILE", value: &vault.CAFile},
		{name: "GRAFT_CREDENTIAL_VAULT_AUTH_MOUNT", value: &vault.AuthMount},
		{name: "GRAFT_CREDENTIAL_VAULT_AUTH_ROLE", value: &vault.AuthRole},
		{name: "GRAFT_CREDENTIAL_VAULT_AUTH_ROLE_ID_FILE", value: &vault.AuthRoleIDFile},
		{name: "GRAFT_CREDENTIAL_VAULT_AUTH_SECRET_ID_FILE", value: &vault.AuthSecretIDFile},
		{name: "GRAFT_CREDENTIAL_VAULT_PKI_MOUNT", value: &vault.PKIMount},
		{name: "GRAFT_CREDENTIAL_VAULT_PKI_ROLE", value: &vault.PKIRole},
		{name: "GRAFT_CREDENTIAL_VAULT_TRUST_BUNDLE_REF", value: &vault.TrustBundleRef},
	} {
		*field.value = strings.TrimSpace(*field.value)
		if *field.value == "" {
			return fmt.Errorf("%s is required when GRAFT_CREDENTIAL_VAULT_ENABLED is true", field.name)
		}
	}
	if err := validateCredentialVaultAddress(vault.Address); err != nil {
		return err
	}
	for _, field := range []struct {
		name  string
		value *string
	}{
		{name: "GRAFT_CREDENTIAL_VAULT_CA_FILE", value: &vault.CAFile},
		{name: "GRAFT_CREDENTIAL_VAULT_AUTH_ROLE_ID_FILE", value: &vault.AuthRoleIDFile},
		{name: "GRAFT_CREDENTIAL_VAULT_AUTH_SECRET_ID_FILE", value: &vault.AuthSecretIDFile},
	} {
		*field.value = strings.TrimSpace(*field.value)
		if *field.value == "" || !filepath.IsAbs(*field.value) {
			return fmt.Errorf("%s must be an absolute secret file path when GRAFT_CREDENTIAL_VAULT_ENABLED is true", field.name)
		}
		*field.value = filepath.Clean(*field.value)
	}
	vault.Namespace = strings.TrimSpace(vault.Namespace)
	return nil
}

func validateCredentialVaultAddress(value string) error {
	address, err := url.Parse(value)
	if err != nil || address.Scheme != "https" || address.Host == "" || address.User != nil || address.RawQuery != "" || address.Fragment != "" {
		return errors.New("GRAFT_CREDENTIAL_VAULT_ADDRESS must be an HTTPS endpoint without credentials, query, or fragment when GRAFT_CREDENTIAL_VAULT_ENABLED is true")
	}
	return nil
}

// validateAuditConfig 保留审计启动配置边界；当前配置没有额外校验约束。
func validateAuditConfig(_ *Config) error {
	return nil
}

// 如果配置值无效，则返回错误。
func validateLogConfig(c *Config) error {
	c.Log.Level = strings.ToLower(strings.TrimSpace(c.Log.Level))
	switch c.Log.Level {
	case "trace", "debug", "info", "warn", "warning", "error", "dpanic", "panic", "fatal":
	default:
		return fmt.Errorf("unsupported GRAFT_LOG_LEVEL value %q", c.Log.Level)
	}
	c.Log.Categories = strings.TrimSpace(c.Log.Categories)

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

	return nil
}

func validateRuntimeConfig(c *Config) error {
	// 先规范化环境输入，再将 Gin 模式限制在显式支持的枚举内，避免框架自行解释未知值。
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
	// 数据库配置只校验启动所需的静态约束；连接可用性由数据库资源构造阶段负责确认。
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
	// Redis 连接池边界在启动前固定，负数值会导致客户端行为不可预测，因此在配置阶段拒绝。
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

// validateMCPConfig 拒绝无效的确认 Token 生命周期，避免开启 MCP 后出现不可预测的确认窗口。
func validateMCPConfig(c *Config) error {
	if !c.MCP.Enabled {
		return nil
	}
	if c.MCP.ConfirmationTokenTTL <= 0 {
		return errors.New("GRAFT_MCP_CONFIRMATION_TOKEN_TTL must be greater than zero")
	}
	if c.MCP.SessionTimeout <= 0 || c.MCP.RequestTimeout <= 0 {
		return errors.New("GRAFT_MCP_SESSION_TIMEOUT and GRAFT_MCP_REQUEST_TIMEOUT must be greater than zero")
	}
	if c.MCP.MaxRequestBytes <= 0 || c.MCP.MaxSessions <= 0 || c.MCP.MaxConcurrentRequests <= 0 {
		return errors.New("GRAFT_MCP request, session, and concurrency limits must be greater than zero")
	}
	return nil
}

// validateContainerConfig 校验并规范化容器运行时和 Docker endpoint；
// 运行时不受支持或 endpoint 为空时返回错误。
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
// defaultDocsEnabledForEnv 根据应用环境判断是否启用文档。
// 返回 true 表示应启用文档，false 表示不应启用文档。
func defaultDocsEnabledForEnv(env string) bool {
	switch classifyAppEnv(env) {
	case appEnvLocalLike, appEnvTest:
		return true
	default:
		return false
	}
}

// ResolveLogFormat 保留显式指定的 console 或 JSON 格式；自动选择时，本地类环境使用 console，其它环境使用 JSON。
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

// ResolveLogColor 根据有效日志格式、显式颜色策略和应用环境判断 console 编码器是否应使用 ANSI 颜色。
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

// ResolveAccessLogConsolePolicy 返回有效的访问日志控制台输出策略。
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

// normalizeGinMode 将输入规范化为支持的 Gin 模式值；无法识别时返回 auto。
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
// normalizeIndexedStringList 规范化字符串列表，并构建以规范化值为键的集合。返回规范化后的列表及其集合。
func normalizeIndexedStringList(items []string) ([]string, map[string]struct{}) {
	normalized := normalizeStringList(items)
	seen := make(map[string]struct{}, len(normalized))
	for _, item := range normalized {
		seen[item] = struct{}{}
	}
	return normalized, seen
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
