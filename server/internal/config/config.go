package config

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

const (
	defaultAppName        = "graft"
	defaultAppEnv         = "local"
	defaultHTTPAddr       = ":8080"
	defaultDatabaseDriver = "postgres"
	defaultDatabaseURL    = "postgres://graft:graft@localhost:5432/graft?sslmode=disable"
	defaultRedisAddr      = "localhost:6379"
	defaultLogLevel       = "info"
	defaultLocale         = "zh-CN"
	defaultSupported      = "zh-CN"
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
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Validate 校验配置是否足以让服务以确定方式启动。
//
// 该方法只验证 core 当前明确依赖的约束，不负责探测数据库或 Redis 的连通性；
// 这些外部资源的真实可用性由对应资源构造阶段继续确认。
func (c *Config) Validate() error {
	if c == nil {
		return errors.New("config is required")
	}

	if strings.TrimSpace(c.App.Name) == "" {
		return errors.New("GRAFT_APP_NAME is required")
	}

	if strings.TrimSpace(c.HTTP.Addr) == "" {
		return errors.New("GRAFT_HTTP_ADDR is required")
	}

	if strings.TrimSpace(c.Database.Driver) != defaultDatabaseDriver {
		return fmt.Errorf("unsupported database driver %q: only postgres is supported", c.Database.Driver)
	}

	if strings.TrimSpace(c.Database.URL) == "" {
		return errors.New("GRAFT_DATABASE_URL is required")
	}

	if strings.TrimSpace(c.Redis.Addr) == "" {
		return errors.New("GRAFT_REDIS_ADDR is required")
	}

	if c.Redis.DB < 0 {
		return errors.New("GRAFT_REDIS_DB must be greater than or equal to zero")
	}

	if strings.TrimSpace(c.I18n.DefaultLocale) == "" {
		return errors.New("GRAFT_I18N_DEFAULT_LOCALE is required")
	}

	if strings.TrimSpace(c.I18n.FallbackLocale) == "" {
		return errors.New("GRAFT_I18N_FALLBACK_LOCALE is required")
	}

	if len(c.I18n.SupportedLocales) == 0 {
		return errors.New("GRAFT_I18N_SUPPORTED_LOCALES must include at least one locale")
	}

	return nil
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
}

func parseLocaleList(raw string) []string {
	items := make([]string, 0)
	seen := make(map[string]struct{})

	for _, part := range strings.Split(raw, ",") {
		locale := strings.TrimSpace(part)
		if locale == "" {
			continue
		}
		if _, ok := seen[locale]; ok {
			continue
		}

		seen[locale] = struct{}{}
		items = append(items, locale)
	}

	return items
}
