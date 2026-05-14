package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestLoadReadsDotenv 验证 Load 会读取当前目录下的 .env 默认值。
func TestLoadReadsDotenv(t *testing.T) {
	restoreEnv := clearGraftEnv(t)
	t.Cleanup(restoreEnv)
	chdir(t, t.TempDir())

	env := strings.Join([]string{
		"GRAFT_APP_NAME=dotenv-graft",
		"GRAFT_APP_ENV=test",
		"GRAFT_HTTP_ADDR=:18080",
		"GRAFT_DATABASE_DRIVER=postgres",
		"GRAFT_DATABASE_URL=postgres://graft:graft@db:5432/graft?sslmode=disable",
		"GRAFT_REDIS_ADDR=redis:6379",
		"GRAFT_REDIS_DB=2",
		"GRAFT_LOG_LEVEL=debug",
	}, "\n")
	if err := os.WriteFile(".env", []byte(env), 0o600); err != nil {
		t.Fatalf("write .env: %v", err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if cfg.App.Name != "dotenv-graft" {
		t.Fatalf("expected app name from .env, got %q", cfg.App.Name)
	}
	if cfg.HTTP.Addr != ":18080" {
		t.Fatalf("expected HTTP address from .env, got %q", cfg.HTTP.Addr)
	}
	if cfg.Redis.Addr != "redis:6379" {
		t.Fatalf("expected Redis address from .env, got %q", cfg.Redis.Addr)
	}
	if cfg.Redis.DB != 2 {
		t.Fatalf("expected Redis DB from .env, got %d", cfg.Redis.DB)
	}
	if cfg.I18n.DefaultLocale != defaultLocale {
		t.Fatalf("expected default locale %q, got %q", defaultLocale, cfg.I18n.DefaultLocale)
	}
	if cfg.I18n.FallbackLocale != defaultLocale {
		t.Fatalf("expected fallback locale %q, got %q", defaultLocale, cfg.I18n.FallbackLocale)
	}
	if cfg.Auth.AccessTokenTTL != defaultAccessTokenTTL {
		t.Fatalf("expected default access token ttl %s, got %s", defaultAccessTokenTTL, cfg.Auth.AccessTokenTTL)
	}
	if cfg.Auth.RefreshTokenTTL != defaultRefreshTokenTTL {
		t.Fatalf("expected default refresh token ttl %s, got %s", defaultRefreshTokenTTL, cfg.Auth.RefreshTokenTTL)
	}
	if cfg.Auth.JWTSecret != defaultJWTSecret {
		t.Fatalf("expected default jwt secret %q, got %q", defaultJWTSecret, cfg.Auth.JWTSecret)
	}
	if cfg.Auth.SigningKey != defaultSigningKey {
		t.Fatalf("expected default signing key %q, got %q", defaultSigningKey, cfg.Auth.SigningKey)
	}
	if cfg.Auth.RefreshCookieName != defaultRefreshCookieName {
		t.Fatalf("expected default refresh cookie name %q, got %q", defaultRefreshCookieName, cfg.Auth.RefreshCookieName)
	}
	if cfg.Auth.RefreshCookieSecure {
		t.Fatal("expected default refresh cookie secure to be false")
	}
	if cfg.Auth.RefreshCookieSameSite != defaultRefreshCookieSameSite {
		t.Fatalf("expected default refresh cookie same site %q, got %q", defaultRefreshCookieSameSite, cfg.Auth.RefreshCookieSameSite)
	}
	if cfg.Auth.RefreshCookiePath != defaultRefreshCookiePath {
		t.Fatalf("expected default refresh cookie path %q, got %q", defaultRefreshCookiePath, cfg.Auth.RefreshCookiePath)
	}
}

// TestLoadReadsServerDotenvFromRepoRoot 验证从仓库根目录启动时会回退读取 server/.env。
func TestLoadReadsServerDotenvFromRepoRoot(t *testing.T) {
	restoreEnv := clearGraftEnv(t)
	t.Cleanup(restoreEnv)

	root := t.TempDir()
	chdir(t, root)

	if err := os.MkdirAll("server", 0o755); err != nil {
		t.Fatalf("create server directory: %v", err)
	}

	env := strings.Join([]string{
		"GRAFT_APP_NAME=server-dotenv-graft",
		"GRAFT_APP_ENV=local",
		"GRAFT_HTTP_ADDR=:38080",
		"GRAFT_DATABASE_DRIVER=postgres",
		"GRAFT_DATABASE_URL=postgres://graft:graft@db:5432/graft?sslmode=disable",
		"GRAFT_REDIS_ADDR=redis:6379",
		"GRAFT_REDIS_DB=3",
		"GRAFT_LOG_LEVEL=warn",
	}, "\n")
	if err := os.WriteFile(filepath.Join("server", ".env"), []byte(env), 0o600); err != nil {
		t.Fatalf("write server/.env: %v", err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if cfg.App.Name != "server-dotenv-graft" {
		t.Fatalf("expected app name from server/.env, got %q", cfg.App.Name)
	}
	if cfg.HTTP.Addr != ":38080" {
		t.Fatalf("expected HTTP address from server/.env, got %q", cfg.HTTP.Addr)
	}
	if cfg.Redis.DB != 3 {
		t.Fatalf("expected Redis DB from server/.env, got %d", cfg.Redis.DB)
	}
	if cfg.I18n.DefaultLocale != defaultLocale {
		t.Fatalf("expected default locale %q, got %q", defaultLocale, cfg.I18n.DefaultLocale)
	}
}

// TestLoadKeepsRealEnvironmentBeforeDotenv 验证真实环境变量优先于 .env 中的默认值。
func TestLoadKeepsRealEnvironmentBeforeDotenv(t *testing.T) {
	restoreEnv := clearGraftEnv(t)
	t.Cleanup(restoreEnv)
	chdir(t, t.TempDir())

	if err := os.WriteFile(".env", []byte("GRAFT_HTTP_ADDR=:18080\n"), 0o600); err != nil {
		t.Fatalf("write .env: %v", err)
	}
	t.Setenv("GRAFT_HTTP_ADDR", ":28080")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if cfg.HTTP.Addr != ":28080" {
		t.Fatalf("expected real environment to win, got %q", cfg.HTTP.Addr)
	}
}

// TestLoadUsesDefaultsWhenNoEnvironmentAvailable 验证在没有显式环境变量与
// dotenv 文件时，Load 会回退到仓库定义的默认配置。
func TestLoadUsesDefaultsWhenNoEnvironmentAvailable(t *testing.T) {
	restoreEnv := clearGraftEnv(t)
	t.Cleanup(restoreEnv)
	chdir(t, t.TempDir())

	cfg, err := Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if cfg.App.Name != defaultAppName {
		t.Fatalf("expected default app name %q, got %q", defaultAppName, cfg.App.Name)
	}
	if cfg.App.Env != defaultAppEnv {
		t.Fatalf("expected default app env %q, got %q", defaultAppEnv, cfg.App.Env)
	}
	if cfg.HTTP.Addr != defaultHTTPAddr {
		t.Fatalf("expected default HTTP address %q, got %q", defaultHTTPAddr, cfg.HTTP.Addr)
	}
	if cfg.Database.Driver != defaultDatabaseDriver {
		t.Fatalf("expected default database driver %q, got %q", defaultDatabaseDriver, cfg.Database.Driver)
	}
	if cfg.Database.URL != defaultDatabaseURL {
		t.Fatalf("expected default database URL %q, got %q", defaultDatabaseURL, cfg.Database.URL)
	}
	if cfg.Redis.Addr != defaultRedisAddr {
		t.Fatalf("expected default Redis address %q, got %q", defaultRedisAddr, cfg.Redis.Addr)
	}
	if cfg.Log.Level != defaultLogLevel {
		t.Fatalf("expected default log level %q, got %q", defaultLogLevel, cfg.Log.Level)
	}
	if cfg.I18n.DefaultLocale != defaultLocale {
		t.Fatalf("expected default locale %q, got %q", defaultLocale, cfg.I18n.DefaultLocale)
	}
	if cfg.I18n.FallbackLocale != defaultLocale {
		t.Fatalf("expected fallback locale %q, got %q", defaultLocale, cfg.I18n.FallbackLocale)
	}
	if len(cfg.I18n.SupportedLocales) != 1 || cfg.I18n.SupportedLocales[0] != defaultLocale {
		t.Fatalf("expected supported locales [%q], got %#v", defaultLocale, cfg.I18n.SupportedLocales)
	}
	if cfg.Auth.AccessTokenTTL != defaultAccessTokenTTL {
		t.Fatalf("expected default access token ttl %s, got %s", defaultAccessTokenTTL, cfg.Auth.AccessTokenTTL)
	}
}

// TestLoadPrefersExplicitEnvFile 验证显式指定的环境文件会优先于默认
// `.env` / `server/.env` 回退路径加载。
func TestLoadPrefersExplicitEnvFile(t *testing.T) {
	restoreEnv := clearGraftEnv(t)
	t.Cleanup(restoreEnv)
	chdir(t, t.TempDir())

	if err := os.WriteFile(".env", []byte("GRAFT_APP_NAME=from-default-dotenv\nGRAFT_LOG_LEVEL=warn\n"), 0o600); err != nil {
		t.Fatalf("write default .env: %v", err)
	}
	if err := os.WriteFile("custom.env", []byte("GRAFT_APP_NAME=from-explicit-dotenv\nGRAFT_LOG_LEVEL=error\n"), 0o600); err != nil {
		t.Fatalf("write custom env: %v", err)
	}
	t.Setenv("GRAFT_ENV_FILE", "custom.env")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if cfg.App.Name != "from-explicit-dotenv" {
		t.Fatalf("expected explicit env file app name, got %q", cfg.App.Name)
	}
	if cfg.Log.Level != "error" {
		t.Fatalf("expected explicit env file log level, got %q", cfg.Log.Level)
	}
}

// TestLoadReadsI18nLocales 验证 i18n 相关配置会按逗号分隔解析为稳定列表。
func TestLoadReadsI18nLocales(t *testing.T) {
	restoreEnv := clearGraftEnv(t)
	t.Cleanup(restoreEnv)
	chdir(t, t.TempDir())

	env := strings.Join([]string{
		"GRAFT_I18N_DEFAULT_LOCALE=zh-CN",
		"GRAFT_I18N_FALLBACK_LOCALE=zh-CN",
		"GRAFT_I18N_SUPPORTED_LOCALES=zh-CN, en-US ,zh-CN",
	}, "\n")
	if err := os.WriteFile(".env", []byte(env), 0o600); err != nil {
		t.Fatalf("write .env: %v", err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	expected := []string{"zh-CN", "en-US"}
	if len(cfg.I18n.SupportedLocales) != len(expected) {
		t.Fatalf("expected supported locales %v, got %v", expected, cfg.I18n.SupportedLocales)
	}
	for index, locale := range expected {
		if cfg.I18n.SupportedLocales[index] != locale {
			t.Fatalf("expected supported locales %v, got %v", expected, cfg.I18n.SupportedLocales)
		}
	}
}

// TestValidateRejectsUnsupportedDatabaseDriver 验证 Validate 会拒绝非 postgres 驱动。
func TestValidateRejectsUnsupportedDatabaseDriver(t *testing.T) {
	cfg := &Config{
		App: AppConfig{
			Name: "graft",
			Env:  "test",
		},
		HTTP: HTTPConfig{
			Addr: ":8080",
		},
		Database: DatabaseConfig{
			Driver: "sqlite",
			URL:    "postgres://graft:graft@db:5432/graft?sslmode=disable",
		},
		Redis: RedisConfig{
			Addr: "localhost:6379",
		},
		I18n: I18nConfig{
			DefaultLocale:    "zh-CN",
			FallbackLocale:   "zh-CN",
			SupportedLocales: []string{"zh-CN"},
		},
	}

	if err := cfg.Validate(); err == nil {
		t.Fatal("expected unsupported database driver error")
	}
}

// TestValidateRejectsMissingDatabaseURL 验证 Validate 会拒绝缺失数据库连接串的配置。
func TestValidateRejectsMissingDatabaseURL(t *testing.T) {
	cfg := &Config{
		App: AppConfig{
			Name: "graft",
			Env:  "test",
		},
		HTTP: HTTPConfig{
			Addr: ":8080",
		},
		Database: DatabaseConfig{
			Driver: "postgres",
		},
		Redis: RedisConfig{
			Addr: "localhost:6379",
		},
		I18n: I18nConfig{
			DefaultLocale:    "zh-CN",
			FallbackLocale:   "zh-CN",
			SupportedLocales: []string{"zh-CN"},
		},
	}

	if err := cfg.Validate(); err == nil {
		t.Fatal("expected missing database URL error")
	}
}

// TestValidateRejectsMissingSupportedLocales 验证 Validate 会拒绝没有支持语言的配置。
func TestValidateRejectsMissingSupportedLocales(t *testing.T) {
	cfg := &Config{
		App: AppConfig{
			Name: "graft",
			Env:  "test",
		},
		HTTP: HTTPConfig{
			Addr: ":8080",
		},
		Database: DatabaseConfig{
			Driver: "postgres",
			URL:    "postgres://graft:graft@db:5432/graft?sslmode=disable",
		},
		Redis: RedisConfig{
			Addr: "localhost:6379",
		},
		I18n: I18nConfig{
			DefaultLocale:  "zh-CN",
			FallbackLocale: "zh-CN",
		},
	}

	if err := cfg.Validate(); err == nil {
		t.Fatal("expected missing supported locales error")
	}
}

// TestValidateRejectsMissingAuthTokenTTLs 验证 Validate 会拒绝非正数的 token 期限。
func TestValidateRejectsMissingAuthTokenTTLs(t *testing.T) {
	cfg := &Config{
		App: AppConfig{
			Name: "graft",
			Env:  "test",
		},
		HTTP: HTTPConfig{
			Addr: ":8080",
		},
		Database: DatabaseConfig{
			Driver: "postgres",
			URL:    "postgres://graft:graft@db:5432/graft?sslmode=disable",
		},
		Redis: RedisConfig{
			Addr: "localhost:6379",
		},
		I18n: I18nConfig{
			DefaultLocale:    "zh-CN",
			FallbackLocale:   "zh-CN",
			SupportedLocales: []string{"zh-CN"},
		},
		Auth: AuthConfig{
			AccessTokenTTL:        0,
			RefreshTokenTTL:       time.Minute,
			JWTSecret:             "secret",
			SigningKey:            "signing",
			RefreshCookieName:     defaultRefreshCookieName,
			RefreshCookieSameSite: defaultRefreshCookieSameSite,
			RefreshCookiePath:     defaultRefreshCookiePath,
		},
	}

	if err := cfg.Validate(); err == nil {
		t.Fatal("expected missing access token ttl error")
	}
}

// TestValidateRejectsUnsafeCookieMode 验证 SameSite=None 时必须同时开启安全 cookie。
func TestValidateRejectsUnsafeCookieMode(t *testing.T) {
	cfg := &Config{
		App: AppConfig{
			Name: "graft",
			Env:  "test",
		},
		HTTP: HTTPConfig{
			Addr: ":8080",
		},
		Database: DatabaseConfig{
			Driver: "postgres",
			URL:    "postgres://graft:graft@db:5432/graft?sslmode=disable",
		},
		Redis: RedisConfig{
			Addr: "localhost:6379",
		},
		I18n: I18nConfig{
			DefaultLocale:    "zh-CN",
			FallbackLocale:   "zh-CN",
			SupportedLocales: []string{"zh-CN"},
		},
		Auth: AuthConfig{
			AccessTokenTTL:        time.Minute,
			RefreshTokenTTL:       time.Hour,
			JWTSecret:             "secret",
			SigningKey:            "signing",
			RefreshCookieName:     defaultRefreshCookieName,
			RefreshCookieSecure:   false,
			RefreshCookieSameSite: "none",
			RefreshCookiePath:     defaultRefreshCookiePath,
		},
	}

	if err := cfg.Validate(); err == nil {
		t.Fatal("expected insecure same-site none error")
	}
}

func chdir(t *testing.T, dir string) {
	t.Helper()

	previous, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("change working directory: %v", err)
	}

	t.Cleanup(func() {
		if err := os.Chdir(previous); err != nil {
			t.Fatalf("restore working directory to %s: %v", filepath.Clean(previous), err)
		}
	})
}

// clearGraftEnv 隔离当前进程中的 GRAFT_* 环境变量，避免测试彼此污染。
func clearGraftEnv(t *testing.T) func() {
	t.Helper()

	original := make(map[string]string)
	for _, item := range os.Environ() {
		key, value, ok := strings.Cut(item, "=")
		if !ok || !strings.HasPrefix(key, "GRAFT_") {
			continue
		}

		original[key] = value
		if err := os.Unsetenv(key); err != nil {
			t.Fatalf("unset %s: %v", key, err)
		}
	}

	return func() {
		for _, item := range os.Environ() {
			key, _, ok := strings.Cut(item, "=")
			if ok && strings.HasPrefix(key, "GRAFT_") {
				_ = os.Unsetenv(key)
			}
		}
		for key, value := range original {
			_ = os.Setenv(key, value)
		}
	}
}
