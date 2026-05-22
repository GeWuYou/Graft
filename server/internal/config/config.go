package config

import (
	"errors"
	"fmt"
	"os"
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
	defaultDatabaseURL           = "postgres://graft:graft@localhost:5432/graft?sslmode=disable"
	defaultRedisAddr             = "localhost:6379"
	defaultLogLevel              = "info"
	defaultLocale                = "zh-CN"
	defaultSecondaryLocale       = "en-US"
	defaultSupported             = "zh-CN,en-US"
	defaultAccessTokenTTL        = 15 * time.Minute
	defaultRefreshTokenTTL       = 7 * 24 * time.Hour
	defaultRefreshCookieName     = "graft_refresh_token"
	defaultRefreshCookiePath     = "/"
	defaultRefreshCookieSameSite = "lax"
)

// Config 包含服务启动前一次性解析并校验的运行时配置快照。
//
// core 会把该快照作为只读依赖注入给运行时与插件，避免后续流程再隐式读取环境变量。
type Config struct {
	App      AppConfig
	HTTP     HTTPConfig
	Database DatabaseConfig
	Redis    RedisConfig
	Log      LogConfig
	I18n     I18nConfig
	Auth     AuthConfig
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

// DatabaseConfig 描述 Ent 与 Atlas 共用的 PostgreSQL 连接配置。
type DatabaseConfig struct {
	Driver string
	URL    string
}

// RedisConfig 描述 core 服务与插件共享的 Redis 连接配置。
type RedisConfig struct {
	Addr     string
	Password string
	DB       int
}

// LogConfig 描述日志核心服务接入后的日志行为配置。
type LogConfig struct {
	Level string
}

// I18nConfig 描述平台级语言解析与消息回退配置。
type I18nConfig struct {
	DefaultLocale    string
	FallbackLocale   string
	SupportedLocales []string
}

// AuthConfig 描述认证插件和 HTTP 会话相关的最小稳定配置。
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

// Load 按“真实环境变量优先、.env 兜底”的顺序加载配置并返回校验后的快照。
//
// 失败语义：
//   - 当显式指定的 `GRAFT_ENV_FILE` 无法读取时直接返回错误，避免启动时误用过期默认值。
//   - 当最终配置不满足运行时最小要求时返回 Validate 的校验错误。
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
		Database: DatabaseConfig{
			Driver: reader.GetString("database.driver"),
			URL:    reader.GetString("database.url"),
		},
		Redis: RedisConfig{
			Addr:     reader.GetString("redis.addr"),
			Password: reader.GetString("redis.password"),
			DB:       reader.GetInt("redis.db"),
		},
		Log: LogConfig{
			Level: reader.GetString("log.level"),
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
		validateDatabaseConfig,
		validateRedisConfig,
		validateI18nConfig,
		validateAuthConfig,
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

func validateDatabaseConfig(c *Config) error {
	if strings.TrimSpace(c.Database.Driver) != defaultDatabaseDriver {
		return fmt.Errorf("unsupported database driver %q: only postgres is supported", c.Database.Driver)
	}
	if strings.TrimSpace(c.Database.URL) == "" {
		return errors.New("GRAFT_DATABASE_URL is required")
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

	if _, err := os.Stat(".env"); err == nil {
		return godotenv.Load(".env")
	}

	if _, err := os.Stat("server/.env"); err == nil {
		return godotenv.Load("server/.env")
	}

	return nil
}

func setDefaults(reader *viper.Viper) {
	reader.SetDefault("app.name", defaultAppName)
	reader.SetDefault("app.env", defaultAppEnv)
	reader.SetDefault("http.addr", defaultHTTPAddr)
	reader.SetDefault("database.driver", defaultDatabaseDriver)
	reader.SetDefault("database.url", defaultDatabaseURL)
	reader.SetDefault("redis.addr", defaultRedisAddr)
	reader.SetDefault("redis.password", "")
	reader.SetDefault("redis.db", 0)
	reader.SetDefault("log.level", defaultLogLevel)
	reader.SetDefault("i18n.default_locale", defaultLocale)
	reader.SetDefault("i18n.fallback_locale", defaultLocale)
	reader.SetDefault("i18n.supported_locales", defaultSupported)
	reader.SetDefault("auth.access_token_ttl", defaultAccessTokenTTL)
	reader.SetDefault("auth.refresh_token_ttl", defaultRefreshTokenTTL)
	reader.SetDefault("auth.refresh_cookie_name", defaultRefreshCookieName)
	reader.SetDefault("auth.refresh_cookie_secure", false)
	reader.SetDefault("auth.refresh_cookie_same_site", defaultRefreshCookieSameSite)
	reader.SetDefault("auth.refresh_cookie_path", defaultRefreshCookiePath)
}

func parseLocaleList(raw string) []string {
	items, _ := normalizeLocaleList(strings.Split(raw, ","))
	return items
}
