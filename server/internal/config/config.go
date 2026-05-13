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
)

// Config 包含服务启动时加载的完整运行时配置。
type Config struct {
	App      AppConfig
	HTTP     HTTPConfig
	Database DatabaseConfig
	Redis    RedisConfig
	Log      LogConfig
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

// Load 先读取可选的 .env 默认值，再解析最终生效的环境配置。
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
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Validate 校验运行时配置是否足以让服务以确定方式启动。
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
}
