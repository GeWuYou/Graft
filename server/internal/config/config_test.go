// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"net/url"
	"os"
	"path/filepath"
	"reflect"
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
		"GRAFT_DATABASE_MAX_OPEN_CONNS=41",
		"GRAFT_DATABASE_MAX_IDLE_CONNS=17",
		"GRAFT_DATABASE_CONN_MAX_LIFETIME=2h",
		"GRAFT_DATABASE_CONN_MAX_IDLE_TIME=20m",
		"GRAFT_REDIS_ADDR=redis:6379",
		"GRAFT_REDIS_DB=2",
		"GRAFT_REDIS_POOL_SIZE=31",
		"GRAFT_REDIS_MIN_IDLE_CONNS=4",
		"GRAFT_REDIS_MAX_IDLE_CONNS=12",
		"GRAFT_REDIS_MAX_ACTIVE_CONNS=45",
		"GRAFT_REDIS_POOL_TIMEOUT=2s",
		"GRAFT_REDIS_CONN_MAX_IDLE_TIME=15m",
		"GRAFT_REDIS_CONN_MAX_LIFETIME=1h",
		"GRAFT_LOG_LEVEL=debug",
		"GRAFT_LOG_FORMAT=json",
		"GRAFT_LOG_COLOR=never",
		"GRAFT_GIN_MODE=release",
		"GRAFT_ACCESS_LOG_CONSOLE=error_only",
		"GRAFT_ACCESS_LOG_SLOW_THRESHOLD_MS=2500",
		"GRAFT_AUTH_JWT_SECRET=dotenv-jwt-secret",
		"GRAFT_AUTH_SIGNING_KEY=dotenv-signing-key",
	}, "\n")
	if err := os.WriteFile(".env", []byte(env), 0o600); err != nil {
		t.Fatalf("write .env: %v", err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	assertEqual(t, "app name from .env", cfg.App.Name, "dotenv-graft")
	assertEqual(t, "HTTP address from .env", cfg.HTTP.Addr, ":18080")
	assertEqual(t, "database max open connections from .env", cfg.Database.MaxOpenConns, 41)
	assertEqual(t, "database max idle connections from .env", cfg.Database.MaxIdleConns, 17)
	assertEqual(t, "database connection max lifetime from .env", cfg.Database.ConnMaxLifetime, 2*time.Hour)
	assertEqual(t, "database connection max idle time from .env", cfg.Database.ConnMaxIdleTime, 20*time.Minute)
	assertEqual(t, "Redis address from .env", cfg.Redis.Addr, "redis:6379")
	assertEqual(t, "Redis DB from .env", cfg.Redis.DB, 2)
	assertEqual(t, "Redis pool size from .env", cfg.Redis.PoolSize, 31)
	assertEqual(t, "Redis min idle connections from .env", cfg.Redis.MinIdleConns, 4)
	assertEqual(t, "Redis max idle connections from .env", cfg.Redis.MaxIdleConns, 12)
	assertEqual(t, "Redis max active connections from .env", cfg.Redis.MaxActiveConns, 45)
	assertEqual(t, "Redis pool timeout from .env", cfg.Redis.PoolTimeout, 2*time.Second)
	assertEqual(t, "Redis connection max idle time from .env", cfg.Redis.ConnMaxIdleTime, 15*time.Minute)
	assertEqual(t, "Redis connection max lifetime from .env", cfg.Redis.ConnMaxLifetime, time.Hour)
	assertEqual(t, "log format from .env", cfg.Log.Format, LogFormatJSON)
	assertEqual(t, "log color from .env", cfg.Log.Color, LogColorNever)
	assertEqual(t, "gin mode from .env", cfg.Runtime.GinMode, GinModeRelease)
	assertEqual(t, "access log console policy from .env", cfg.HTTPX.AccessLogConsole, AccessLogConsoleErrorOnly)
	assertEqual(t, "access log slow threshold from .env", cfg.HTTPX.AccessLogSlowThresholdMS, int64(2500))
	assertEqual(t, "default locale", cfg.I18n.DefaultLocale, defaultLocale)
	assertEqual(t, "fallback locale", cfg.I18n.FallbackLocale, defaultLocale)
	assertStringSliceEqual(t, "supported locales", cfg.I18n.SupportedLocales, []string{defaultLocale, defaultSecondaryLocale})
	assertEqual(t, "default access token ttl", cfg.Auth.AccessTokenTTL, defaultAccessTokenTTL)
	assertEqual(t, "default refresh token ttl", cfg.Auth.RefreshTokenTTL, defaultRefreshTokenTTL)
	assertEqual(t, "jwt secret from .env", cfg.Auth.JWTSecret, "dotenv-jwt-secret")
	assertEqual(t, "signing key from .env", cfg.Auth.SigningKey, "dotenv-signing-key")
	assertEqual(t, "default refresh cookie name", cfg.Auth.RefreshCookieName, defaultRefreshCookieName)
	assertEqual(t, "default refresh cookie secure", cfg.Auth.RefreshCookieSecure, false)
	assertEqual(t, "default refresh cookie same site", cfg.Auth.RefreshCookieSameSite, defaultRefreshCookieSameSite)
	assertEqual(t, "default refresh cookie path", cfg.Auth.RefreshCookiePath, defaultRefreshCookiePath)
	assertEqual(t, "default container runtime enabled", cfg.Container.RuntimeEnabled, false)
	assertEqual(t, "default container runtime", cfg.Container.Runtime, "first-adapter")
	assertEqual(t, "default container endpoint", cfg.Container.DockerEndpoint, "unix:///var/run/docker.sock")
	assertEqual(t, "default container logs tail", cfg.Container.LogsDefaultTail, 200)
	assertEqual(t, "default container logs max tail", cfg.Container.LogsMaxTail, 2000)
	assertEqual(t, "default container dangerous actions", cfg.Container.DangerousActionsEnabled, false)
}

func TestLoadReadsContainerRuntimeConfig(t *testing.T) {
	restoreEnv := clearGraftEnv(t)
	t.Cleanup(restoreEnv)
	chdir(t, t.TempDir())

	t.Setenv("GRAFT_AUTH_JWT_SECRET", "container-config-secret")
	t.Setenv("GRAFT_OPS_CONTAINER_RUNTIME_ENABLED", "true")
	t.Setenv("GRAFT_OPS_CONTAINER_RUNTIME", "docker")
	t.Setenv("GRAFT_OPS_CONTAINER_DOCKER_ENDPOINT", "unix:///tmp/docker.sock")
	t.Setenv("GRAFT_OPS_CONTAINER_LOGS_DEFAULT_TAIL", "50")
	t.Setenv("GRAFT_OPS_CONTAINER_LOGS_MAX_TAIL", "500")
	t.Setenv("GRAFT_OPS_CONTAINER_ACTIONS_DANGEROUS_ENABLED", "true")
	t.Setenv("GRAFT_OPS_CONTAINER_SHELL_ENABLED", "true")
	t.Setenv("GRAFT_HTTPX_WEBSOCKET_ALLOWED_ORIGINS", "http://localhost:3002, http://127.0.0.1:3002")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	assertEqual(t, "container runtime enabled", cfg.Container.RuntimeEnabled, true)
	assertEqual(t, "container runtime", cfg.Container.Runtime, "docker")
	assertEqual(t, "container endpoint", cfg.Container.DockerEndpoint, "unix:///tmp/docker.sock")
	assertEqual(t, "container logs default tail", cfg.Container.LogsDefaultTail, 50)
	assertEqual(t, "container logs max tail", cfg.Container.LogsMaxTail, 500)
	assertEqual(t, "container dangerous actions enabled", cfg.Container.DangerousActionsEnabled, true)
	assertEqual(t, "container shell enabled", cfg.Container.ShellEnabled, true)
	assertStringSliceEqual(
		t,
		"websocket allowed origins",
		cfg.HTTPX.WebSocketAllowedOrigins,
		[]string{"http://localhost:3002", "http://127.0.0.1:3002"},
	)
}

func TestLoadRejectsInvalidWebSocketAllowedOrigin(t *testing.T) {
	restoreEnv := clearGraftEnv(t)
	t.Cleanup(restoreEnv)
	chdir(t, t.TempDir())

	t.Setenv("GRAFT_AUTH_JWT_SECRET", "container-config-secret")
	t.Setenv("GRAFT_OPS_CONTAINER_SHELL_ENABLED", "true")
	t.Setenv("GRAFT_HTTPX_WEBSOCKET_ALLOWED_ORIGINS", "http://localhost:3002/shell")

	_, err := Load()
	if err == nil {
		t.Fatal("expected invalid websocket allowed origin error")
	}
	if !strings.Contains(err.Error(), "GRAFT_HTTPX_WEBSOCKET_ALLOWED_ORIGINS") {
		t.Fatalf("expected websocket allowed origins error, got %v", err)
	}
}

// TestLoadReadsServerDotenvFromRepoRoot 验证从仓库根目录启动时会回退读取 server/.env。
func TestLoadReadsServerDotenvFromRepoRoot(t *testing.T) {
	restoreEnv := clearGraftEnv(t)
	t.Cleanup(restoreEnv)

	root := t.TempDir()
	chdir(t, root)

	if err := os.MkdirAll("server", 0o750); err != nil {
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
		"GRAFT_AUTH_JWT_SECRET=server-dotenv-jwt-secret",
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
	if cfg.Auth.JWTSecret != "server-dotenv-jwt-secret" {
		t.Fatalf("expected jwt secret from server/.env, got %q", cfg.Auth.JWTSecret)
	}
}

// TestLoadReadsServerDotenvFromNestedPackageDir 验证从 server 子目录深层包路径启动时，
// Load 仍会向上回溯并命中仓库规范的 server/.env。
func TestLoadReadsServerDotenvFromNestedPackageDir(t *testing.T) {
	restoreEnv := clearGraftEnv(t)
	t.Cleanup(restoreEnv)

	root := t.TempDir()
	nestedDir := filepath.Join(root, "server", "cmd", "graft")
	if err := os.MkdirAll(nestedDir, 0o750); err != nil {
		t.Fatalf("create nested package directory: %v", err)
	}
	chdir(t, nestedDir)

	env := strings.Join([]string{
		"GRAFT_APP_NAME=nested-server-dotenv-graft",
		"GRAFT_APP_ENV=local",
		"GRAFT_HTTP_ADDR=:48080",
		"GRAFT_DATABASE_DRIVER=postgres",
		"GRAFT_DATABASE_URL=postgres://graft:graft@db:5432/graft?sslmode=disable",
		"GRAFT_REDIS_ADDR=redis:6379",
		"GRAFT_REDIS_DB=4",
		"GRAFT_LOG_LEVEL=warn",
		"GRAFT_AUTH_JWT_SECRET=nested-server-dotenv-jwt-secret",
	}, "\n")
	if err := os.WriteFile(filepath.Join(root, "server", ".env"), []byte(env), 0o600); err != nil {
		t.Fatalf("write server/.env: %v", err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	assertEqual(t, "app name from nested server/.env", cfg.App.Name, "nested-server-dotenv-graft")
	assertEqual(t, "HTTP address from nested server/.env", cfg.HTTP.Addr, ":48080")
	assertEqual(t, "Redis DB from nested server/.env", cfg.Redis.DB, 4)
	assertEqual(t, "jwt secret from nested server/.env", cfg.Auth.JWTSecret, "nested-server-dotenv-jwt-secret")
}

// TestLoadStopsDotenvSearchAtWorkspaceBoundary 验证向上查找不会越过首个项目边界读取外层 .env。
func TestLoadStopsDotenvSearchAtWorkspaceBoundary(t *testing.T) {
	restoreEnv := clearGraftEnv(t)
	t.Cleanup(restoreEnv)

	outer := t.TempDir()
	projectRoot := filepath.Join(outer, "project")
	nestedDir := filepath.Join(projectRoot, "server", "cmd", "graft")
	if err := os.MkdirAll(nestedDir, 0o750); err != nil {
		t.Fatalf("create nested package directory: %v", err)
	}
	chdir(t, nestedDir)

	if err := os.WriteFile(filepath.Join(outer, ".env"), []byte("GRAFT_APP_NAME=outer-dotenv\n"), 0o600); err != nil {
		t.Fatalf("write outer .env: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectRoot, "server", ".env"), []byte("GRAFT_APP_NAME=project-server-dotenv\n"), 0o600); err != nil {
		t.Fatalf("write project server .env: %v", err)
	}
	t.Setenv("GRAFT_AUTH_JWT_SECRET", "workspace-boundary-secret")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	assertEqual(t, "app name from workspace server/.env", cfg.App.Name, "project-server-dotenv")
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
	t.Setenv("GRAFT_AUTH_JWT_SECRET", "runtime-secret")

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
	t.Setenv("GRAFT_AUTH_JWT_SECRET", "runtime-secret")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	assertEqual(t, "default app name", cfg.App.Name, defaultAppName)
	assertEqual(t, "default app env", cfg.App.Env, defaultAppEnv)
	assertEqual(t, "default HTTP address", cfg.HTTP.Addr, defaultHTTPAddr)
	assertEqual(t, "docs enabled in local env by default", cfg.Docs.Enabled, true)
	assertStringSliceEqual(t, "default enabled modules", cfg.Modules.Enabled, []string{})
	assertEqual(t, "default database driver", cfg.Database.Driver, defaultDatabaseDriver)
	assertEqual(t, "default database URL", cfg.Database.URL, defaultDatabaseURL)
	assertEqual(t, "default database max open connections", cfg.Database.MaxOpenConns, defaultDatabaseMaxOpenConns)
	assertEqual(t, "default database max idle connections", cfg.Database.MaxIdleConns, defaultDatabaseMaxIdleConns)
	assertEqual(t, "default database connection max lifetime", cfg.Database.ConnMaxLifetime, defaultDatabaseConnMaxLifetime)
	assertEqual(t, "default database connection max idle time", cfg.Database.ConnMaxIdleTime, defaultDatabaseConnMaxIdleTime)
	assertEqual(t, "default Redis address", cfg.Redis.Addr, defaultRedisAddr)
	assertEqual(t, "default Redis pool size", cfg.Redis.PoolSize, defaultRedisPoolSize)
	assertEqual(t, "default Redis min idle connections", cfg.Redis.MinIdleConns, defaultRedisMinIdleConns)
	assertEqual(t, "default Redis max idle connections", cfg.Redis.MaxIdleConns, defaultRedisMaxIdleConns)
	assertEqual(t, "default Redis max active connections", cfg.Redis.MaxActiveConns, defaultRedisMaxActiveConns)
	assertEqual(t, "default Redis pool timeout", cfg.Redis.PoolTimeout, defaultRedisPoolTimeout)
	assertEqual(t, "default Redis connection max idle time", cfg.Redis.ConnMaxIdleTime, defaultRedisConnMaxIdleTime)
	assertEqual(t, "default Redis connection max lifetime", cfg.Redis.ConnMaxLifetime, defaultRedisConnMaxLifetime)
	assertEqual(t, "default log level", cfg.Log.Level, defaultLogLevel)
	assertEqual(t, "default log format", cfg.Log.Format, LogFormatAuto)
	assertEqual(t, "default log color", cfg.Log.Color, LogColorAuto)
	assertEqual(t, "default gin mode", cfg.Runtime.GinMode, GinModeAuto)
	assertEqual(t, "default access log console", cfg.HTTPX.AccessLogConsole, AccessLogConsoleAuto)
	assertEqual(t, "default access log slow threshold", cfg.HTTPX.AccessLogSlowThresholdMS, int64(1000))
	assertEqual(t, "default locale", cfg.I18n.DefaultLocale, defaultLocale)
	assertEqual(t, "fallback locale", cfg.I18n.FallbackLocale, defaultLocale)
	assertStringSliceEqual(t, "supported locales", cfg.I18n.SupportedLocales, []string{defaultLocale, defaultSecondaryLocale})
	assertEqual(t, "default access token ttl", cfg.Auth.AccessTokenTTL, defaultAccessTokenTTL)
	assertEqual(t, "jwt secret from environment", cfg.Auth.JWTSecret, "runtime-secret")
}

func TestLoadReadsEnabledModules(t *testing.T) {
	restoreEnv := clearGraftEnv(t)
	t.Cleanup(restoreEnv)
	chdir(t, t.TempDir())

	t.Setenv("GRAFT_AUTH_JWT_SECRET", "runtime-secret")
	t.Setenv("GRAFT_MODULES_ENABLED", " user,auth,user ")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	assertStringSliceEqual(t, "enabled modules", cfg.Modules.Enabled, []string{"user", "auth"})
}

func TestLoadDisablesDocsByDefaultInProduction(t *testing.T) {
	restoreEnv := clearGraftEnv(t)
	t.Cleanup(restoreEnv)
	chdir(t, t.TempDir())

	t.Setenv("GRAFT_APP_ENV", "production")
	t.Setenv("GRAFT_AUTH_JWT_SECRET", "runtime-secret")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if cfg.Docs.Enabled {
		t.Fatal("expected docs to stay disabled by default in production")
	}
}

func TestLoadAllowsExplicitDocsOverride(t *testing.T) {
	restoreEnv := clearGraftEnv(t)
	t.Cleanup(restoreEnv)
	chdir(t, t.TempDir())

	t.Setenv("GRAFT_APP_ENV", "production")
	t.Setenv("GRAFT_DOCS_ENABLED", "true")
	t.Setenv("GRAFT_AUTH_JWT_SECRET", "runtime-secret")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if !cfg.Docs.Enabled {
		t.Fatal("expected explicit docs override to enable docs in production")
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
	if err := os.WriteFile("custom.env", []byte("GRAFT_APP_NAME=from-explicit-dotenv\nGRAFT_LOG_LEVEL=error\nGRAFT_AUTH_SIGNING_KEY=explicit-signing-key\n"), 0o600); err != nil {
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
	if cfg.Auth.SigningKey != "explicit-signing-key" {
		t.Fatalf("expected signing key from explicit env file, got %q", cfg.Auth.SigningKey)
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
		"GRAFT_AUTH_JWT_SECRET=i18n-secret",
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

// TestLoadAuthSigningMaterial 验证 Load 会严格校验 auth 签名材料的最小要求。
func TestLoadAuthSigningMaterial(t *testing.T) {
	testCases := []struct {
		name           string
		jwtSecret      string
		signingKey     string
		wantErr        string
		wantJWTSecret  string
		wantSigningKey string
	}{
		{
			name:    "rejects when both jwt secret and signing key are missing",
			wantErr: "GRAFT_AUTH_JWT_SECRET or GRAFT_AUTH_SIGNING_KEY is required",
		},
		{
			name:          "accepts when only jwt secret exists",
			jwtSecret:     testSigningMaterial("jwt"),
			wantJWTSecret: testSigningMaterial("jwt"),
		},
		{
			name:           "accepts when only signing key exists",
			signingKey:     testSigningMaterial("sig"),
			wantSigningKey: testSigningMaterial("sig"),
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			prepareAuthSigningMaterialTest(t, testCase.jwtSecret, testCase.signingKey)
			cfg, err := Load()
			assertAuthSigningMaterialResult(t, cfg, err, testCase.wantErr, testCase.wantJWTSecret, testCase.wantSigningKey)
		})
	}
}

// TestValidateRejectsUnsupportedDatabaseDriver 验证 Validate 会拒绝非 postgres 驱动。
func TestValidateRejectsUnsupportedDatabaseDriver(t *testing.T) {
	cfg := validConfigForValidateTests()
	cfg.Database.Driver = "sqlite"

	assertValidateError(t, cfg, "")
}

// TestValidateRejectsMissingDatabaseURL 验证 Validate 会拒绝缺失数据库连接串的配置。
func TestValidateRejectsMissingDatabaseURL(t *testing.T) {
	cfg := validConfigForValidateTests()
	cfg.Database.URL = ""

	assertValidateError(t, cfg, "")
}

func TestValidateRejectsInvalidPoolConfig(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*Config)
		wantErr string
	}{
		{
			name: "database max open connections",
			mutate: func(cfg *Config) {
				cfg.Database.MaxOpenConns = 0
			},
			wantErr: "GRAFT_DATABASE_MAX_OPEN_CONNS must be greater than zero",
		},
		{
			name: "database max idle connections",
			mutate: func(cfg *Config) {
				cfg.Database.MaxIdleConns = -1
			},
			wantErr: "GRAFT_DATABASE_MAX_IDLE_CONNS must be greater than or equal to zero",
		},
		{
			name: "database connection max lifetime",
			mutate: func(cfg *Config) {
				cfg.Database.ConnMaxLifetime = -time.Second
			},
			wantErr: "GRAFT_DATABASE_CONN_MAX_LIFETIME must be greater than or equal to zero",
		},
		{
			name: "database connection max idle time",
			mutate: func(cfg *Config) {
				cfg.Database.ConnMaxIdleTime = -time.Second
			},
			wantErr: "GRAFT_DATABASE_CONN_MAX_IDLE_TIME must be greater than or equal to zero",
		},
		{
			name: "redis pool size",
			mutate: func(cfg *Config) {
				cfg.Redis.PoolSize = -1
			},
			wantErr: "GRAFT_REDIS_POOL_SIZE must be greater than or equal to zero",
		},
		{
			name: "redis min idle connections",
			mutate: func(cfg *Config) {
				cfg.Redis.MinIdleConns = -1
			},
			wantErr: "GRAFT_REDIS_MIN_IDLE_CONNS must be greater than or equal to zero",
		},
		{
			name: "redis max idle connections",
			mutate: func(cfg *Config) {
				cfg.Redis.MaxIdleConns = -1
			},
			wantErr: "GRAFT_REDIS_MAX_IDLE_CONNS must be greater than or equal to zero",
		},
		{
			name: "redis max active connections",
			mutate: func(cfg *Config) {
				cfg.Redis.MaxActiveConns = -1
			},
			wantErr: "GRAFT_REDIS_MAX_ACTIVE_CONNS must be greater than or equal to zero",
		},
		{
			name: "redis pool timeout",
			mutate: func(cfg *Config) {
				cfg.Redis.PoolTimeout = -time.Second
			},
			wantErr: "GRAFT_REDIS_POOL_TIMEOUT must be greater than or equal to zero",
		},
		{
			name: "redis connection max idle time",
			mutate: func(cfg *Config) {
				cfg.Redis.ConnMaxIdleTime = -time.Second
			},
			wantErr: "GRAFT_REDIS_CONN_MAX_IDLE_TIME must be greater than or equal to zero",
		},
		{
			name: "redis connection max lifetime",
			mutate: func(cfg *Config) {
				cfg.Redis.ConnMaxLifetime = -time.Second
			},
			wantErr: "GRAFT_REDIS_CONN_MAX_LIFETIME must be greater than or equal to zero",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			cfg := validConfigForValidateTests()
			testCase.mutate(cfg)

			assertValidateError(t, cfg, testCase.wantErr)
		})
	}
}

// TestValidateRejectsMissingSupportedLocales 验证 Validate 会拒绝没有支持语言的配置。
func TestValidateRejectsMissingSupportedLocales(t *testing.T) {
	cfg := validConfigForValidateTests()
	cfg.I18n.SupportedLocales = nil

	assertValidateError(t, cfg, "")
}

// TestValidateRejectsMissingRequiredEnglishLocale 验证 Validate 会拒绝缺少
// 当前阶段固定英文 locale 的配置。
func TestValidateRejectsMissingRequiredEnglishLocale(t *testing.T) {
	cfg := validConfigForValidateTests()
	cfg.I18n.SupportedLocales = []string{"zh-CN"}

	assertValidateError(t, cfg, "GRAFT_I18N_SUPPORTED_LOCALES must include \"en-US\"")
}

// TestValidateRejectsDefaultLocaleOutsideSupported 验证 Validate 会拒绝默认
// 语言不在支持列表内的配置。
func TestValidateRejectsDefaultLocaleOutsideSupported(t *testing.T) {
	cfg := validConfigForValidateTests()
	cfg.I18n.DefaultLocale = "fr-FR"

	assertValidateError(t, cfg, "GRAFT_I18N_DEFAULT_LOCALE must be listed in GRAFT_I18N_SUPPORTED_LOCALES")
}

// TestValidateRejectsFallbackLocaleOutsideSupported 验证 Validate 会拒绝回退
// 语言不在支持列表内的配置。
func TestValidateRejectsFallbackLocaleOutsideSupported(t *testing.T) {
	cfg := validConfigForValidateTests()
	cfg.I18n.FallbackLocale = "fr-FR"

	assertValidateError(t, cfg, "GRAFT_I18N_FALLBACK_LOCALE must be listed in GRAFT_I18N_SUPPORTED_LOCALES")
}

// TestValidateNormalizesI18nLocales 验证 Validate 会把 locale 配置收敛到稳定格式，
// 避免校验期通过、运行期仍保留空白和重复值。
func TestValidateNormalizesI18nLocales(t *testing.T) {
	cfg := validConfigForValidateTests()
	cfg.I18n.DefaultLocale = " zh-CN "
	cfg.I18n.FallbackLocale = " en-US "
	cfg.I18n.SupportedLocales = []string{" zh-CN ", "en-US", "zh-CN", " "}

	if err := cfg.Validate(); err != nil {
		t.Fatalf("validate config: %v", err)
	}

	assertEqual(t, "normalized default locale", cfg.I18n.DefaultLocale, "zh-CN")
	assertEqual(t, "normalized fallback locale", cfg.I18n.FallbackLocale, "en-US")
	assertStringSliceEqual(t, "normalized supported locales", cfg.I18n.SupportedLocales, []string{"zh-CN", "en-US"})
}

// TestValidateRejectsMissingAuthTokenTTLs 验证 Validate 会拒绝非正数的 token 期限。
func TestValidateRejectsMissingAuthTokenTTLs(t *testing.T) {
	cfg := validConfigForValidateTests()
	cfg.Auth.AccessTokenTTL = 0

	assertValidateError(t, cfg, "")
}

func TestValidateNormalizesEnabledModules(t *testing.T) {
	cfg := validConfigForValidateTests()
	cfg.Modules.Enabled = []string{" user ", "auth", "user", ""}

	if err := cfg.Validate(); err != nil {
		t.Fatalf("validate config: %v", err)
	}

	assertStringSliceEqual(t, "normalized enabled modules", cfg.Modules.Enabled, []string{"user", "auth"})
}

func TestValidateRejectsNonPositiveAccessLogRetention(t *testing.T) {
	cfg := validConfigForValidateTests()
	cfg.HTTPX.AccessLogRetention = 0

	assertValidateError(t, cfg, "GRAFT_HTTPX_ACCESS_LOG_RETENTION must be greater than zero")
}

func TestValidateRejectsInvalidAccessLogConsolePolicyAndThreshold(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*Config)
		wantErr string
	}{
		{
			name: "console policy",
			mutate: func(cfg *Config) {
				cfg.HTTPX.AccessLogConsole = "sometimes"
			},
			wantErr: `unsupported GRAFT_ACCESS_LOG_CONSOLE value "sometimes"`,
		},
		{
			name: "slow threshold",
			mutate: func(cfg *Config) {
				cfg.HTTPX.AccessLogSlowThresholdMS = 0
			},
			wantErr: "GRAFT_ACCESS_LOG_SLOW_THRESHOLD_MS must be greater than zero",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			cfg := validConfigForValidateTests()
			testCase.mutate(cfg)

			assertValidateError(t, cfg, testCase.wantErr)
		})
	}
}

func TestValidateRejectsNonPositiveAuditLogRetention(t *testing.T) {
	cfg := validConfigForValidateTests()
	cfg.Audit.LogRetention = 0

	assertValidateError(t, cfg, "GRAFT_AUDIT_LOG_RETENTION must be greater than zero")
}

func TestValidateRejectsNonPositiveAppLogRetention(t *testing.T) {
	cfg := validConfigForValidateTests()
	cfg.Log.AppLogPersist = true
	cfg.Log.AppLogRetention = 0

	assertValidateError(t, cfg, "GRAFT_LOG_APP_LOG_RETENTION must be greater than zero")
}

func TestValidateRejectsInvalidLogAndGinEnums(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*Config)
		wantErr string
	}{
		{
			name: "log format",
			mutate: func(cfg *Config) {
				cfg.Log.Format = "pretty"
			},
			wantErr: `unsupported GRAFT_LOG_FORMAT value "pretty"`,
		},
		{
			name: "log color",
			mutate: func(cfg *Config) {
				cfg.Log.Color = "sometimes"
			},
			wantErr: `unsupported GRAFT_LOG_COLOR value "sometimes"`,
		},
		{
			name: "gin mode",
			mutate: func(cfg *Config) {
				cfg.Runtime.GinMode = "trace"
			},
			wantErr: `unsupported GRAFT_GIN_MODE value "trace"`,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			cfg := validConfigForValidateTests()
			testCase.mutate(cfg)

			assertValidateError(t, cfg, testCase.wantErr)
		})
	}
}

func TestValidateAllowsNonPositiveAppLogRetentionWhenPersistenceDisabled(t *testing.T) {
	cfg := validConfigForValidateTests()
	cfg.Log.AppLogPersist = false
	cfg.Log.AppLogRetention = 0

	if err := cfg.Validate(); err != nil {
		t.Fatalf("validate config with disabled app log persistence: %v", err)
	}
}

func TestValidateRejectsMissingContainerDockerEndpoint(t *testing.T) {
	cfg := validConfigForValidateTests()
	cfg.Container.DockerEndpoint = ""

	assertValidateError(t, cfg, "GRAFT_OPS_CONTAINER_DOCKER_ENDPOINT is required")
}

func TestValidateRejectsEnabledContainerShellWithoutWebSocketAllowedOrigins(t *testing.T) {
	cfg := validConfigForValidateTests()
	cfg.Container.ShellEnabled = true
	cfg.HTTPX.WebSocketAllowedOrigins = nil

	assertValidateError(
		t,
		cfg,
		"GRAFT_HTTPX_WEBSOCKET_ALLOWED_ORIGINS is required when GRAFT_OPS_CONTAINER_SHELL_ENABLED is true",
	)
}

func TestResolveLogFormatColorAndGinMode(t *testing.T) {
	logFormatCases := []struct {
		name   string
		env    string
		format LogFormat
		want   LogFormat
	}{
		{name: "auto local", env: "local", format: LogFormatAuto, want: LogFormatConsole},
		{name: "auto test", env: "test", format: LogFormatAuto, want: LogFormatConsole},
		{name: "auto production", env: "production", format: LogFormatAuto, want: LogFormatJSON},
		{name: "auto staging", env: "stage", format: LogFormatAuto, want: LogFormatJSON},
		{name: "auto unknown", env: "preview", format: LogFormatAuto, want: LogFormatJSON},
		{name: "explicit console", env: "production", format: LogFormatConsole, want: LogFormatConsole},
		{name: "explicit json", env: "local", format: LogFormatJSON, want: LogFormatJSON},
	}
	for _, testCase := range logFormatCases {
		t.Run("format/"+testCase.name, func(t *testing.T) {
			if got := ResolveLogFormat(testCase.env, testCase.format); got != testCase.want {
				t.Fatalf("expected log format %q, got %q", testCase.want, got)
			}
		})
	}

	logColorCases := []struct {
		name   string
		env    string
		format LogFormat
		color  LogColor
		want   bool
	}{
		{name: "auto local console", env: "local", format: LogFormatConsole, color: LogColorAuto, want: true},
		{name: "auto test console", env: "test", format: LogFormatConsole, color: LogColorAuto, want: true},
		{name: "auto production console", env: "production", format: LogFormatConsole, color: LogColorAuto, want: false},
		{name: "always console", env: "production", format: LogFormatConsole, color: LogColorAlways, want: true},
		{name: "always json", env: "local", format: LogFormatJSON, color: LogColorAlways, want: false},
		{name: "never console", env: "local", format: LogFormatConsole, color: LogColorNever, want: false},
	}
	for _, testCase := range logColorCases {
		t.Run("color/"+testCase.name, func(t *testing.T) {
			if got := ResolveLogColor(testCase.env, testCase.format, testCase.color); got != testCase.want {
				t.Fatalf("expected log color %v, got %v", testCase.want, got)
			}
		})
	}

	assertResolvedGinMode(t, "auto local", "local", GinModeAuto, GinModeDebug)
	assertResolvedGinMode(t, "auto dev", "dev", GinModeAuto, GinModeDebug)
	assertResolvedGinMode(t, "auto test", "test", GinModeAuto, GinModeTest)
	assertResolvedGinMode(t, "auto production", "prod", GinModeAuto, GinModeRelease)
	assertResolvedGinMode(t, "auto staging", "staging", GinModeAuto, GinModeRelease)
	assertResolvedGinMode(t, "auto unknown", "preview", GinModeAuto, GinModeRelease)
	assertResolvedGinMode(t, "explicit debug", "production", GinModeDebug, GinModeDebug)
}

func assertResolvedGinMode(t *testing.T, name string, env string, mode GinMode, want GinMode) {
	t.Helper()

	t.Run("gin/"+name, func(t *testing.T) {
		if got := ResolveGinMode(env, mode); got != want {
			t.Fatalf("expected gin mode %q, got %q", want, got)
		}
	})
}

func TestResolveAccessLogConsolePolicy(t *testing.T) {
	testCases := []struct {
		name   string
		env    string
		policy AccessLogConsolePolicy
		want   AccessLogConsolePolicy
	}{
		{name: "auto local", env: "local", policy: AccessLogConsoleAuto, want: AccessLogConsoleErrorOnly},
		{name: "auto dev", env: "dev", policy: AccessLogConsoleAuto, want: AccessLogConsoleErrorOnly},
		{name: "auto development", env: "development", policy: AccessLogConsoleAuto, want: AccessLogConsoleErrorOnly},
		{name: "auto production", env: "production", policy: AccessLogConsoleAuto, want: AccessLogConsoleNever},
		{name: "auto prod", env: "prod", policy: AccessLogConsoleAuto, want: AccessLogConsoleNever},
		{name: "auto staging", env: "staging", policy: AccessLogConsoleAuto, want: AccessLogConsoleNever},
		{name: "auto stage", env: "stage", policy: AccessLogConsoleAuto, want: AccessLogConsoleNever},
		{name: "auto test", env: "test", policy: AccessLogConsoleAuto, want: AccessLogConsoleNever},
		{name: "auto unknown", env: "preview", policy: AccessLogConsoleAuto, want: AccessLogConsoleNever},
		{name: "explicit always", env: "production", policy: AccessLogConsoleAlways, want: AccessLogConsoleAlways},
		{name: "explicit never", env: "local", policy: AccessLogConsoleNever, want: AccessLogConsoleNever},
		{name: "explicit error only", env: "production", policy: AccessLogConsoleErrorOnly, want: AccessLogConsoleErrorOnly},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			if got := ResolveAccessLogConsolePolicy(testCase.env, testCase.policy); got != testCase.want {
				t.Fatalf("expected access log console policy %q, got %q", testCase.want, got)
			}
		})
	}
}

func TestDefaultLogRetentionForEnv(t *testing.T) {
	testCases := []struct {
		env        string
		wantAccess time.Duration
		wantAudit  time.Duration
		wantApp    time.Duration
	}{
		{env: "development", wantAccess: 3 * 24 * time.Hour, wantAudit: 30 * 24 * time.Hour, wantApp: 3 * 24 * time.Hour},
		{env: "staging", wantAccess: 7 * 24 * time.Hour, wantAudit: 90 * 24 * time.Hour, wantApp: 7 * 24 * time.Hour},
		{env: "production", wantAccess: 30 * 24 * time.Hour, wantAudit: 180 * 24 * time.Hour, wantApp: 14 * 24 * time.Hour},
		{env: "local", wantAccess: 3 * 24 * time.Hour, wantAudit: 30 * 24 * time.Hour, wantApp: 3 * 24 * time.Hour},
	}

	for _, testCase := range testCases {
		if got := defaultAccessLogRetentionForEnv(testCase.env); got != testCase.wantAccess {
			t.Fatalf("env %q: expected access retention %s, got %s", testCase.env, testCase.wantAccess, got)
		}
		if got := defaultAuditLogRetentionForEnv(testCase.env); got != testCase.wantAudit {
			t.Fatalf("env %q: expected audit retention %s, got %s", testCase.env, testCase.wantAudit, got)
		}
		if got := defaultAppLogRetentionForEnv(testCase.env); got != testCase.wantApp {
			t.Fatalf("env %q: expected app retention %s, got %s", testCase.env, testCase.wantApp, got)
		}
	}
}

// TestValidateRejectsUnsafeCookieMode 验证 SameSite=None 时必须同时开启安全 cookie。
func TestValidateRejectsUnsafeCookieMode(t *testing.T) {
	cfg := validConfigForValidateTests()
	cfg.Auth.RefreshCookieSecure = false
	cfg.Auth.RefreshCookieSameSite = "none"

	assertValidateError(t, cfg, "")
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

func assertEqual[T comparable](t *testing.T, label string, got T, want T) {
	t.Helper()

	if got != want {
		t.Fatalf("expected %s %v, got %v", label, want, got)
	}
}

func assertStringSliceEqual(t *testing.T, label string, got []string, want []string) {
	t.Helper()

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %s %v, got %v", label, want, got)
	}
}

func validConfigForValidateTests() *Config {
	return &Config{
		App: AppConfig{
			Name: "graft",
			Env:  "test",
		},
		HTTP: HTTPConfig{
			Addr: ":8080",
		},
		HTTPX: HTTPXConfig{
			AccessLogRetention:       3 * 24 * time.Hour,
			AccessLogConsole:         AccessLogConsoleAuto,
			AccessLogSlowThresholdMS: 1000,
		},
		Audit: AuditConfig{
			LogRetention: 30 * 24 * time.Hour,
		},
		Log: LogConfig{
			Level:           "info",
			Format:          LogFormatAuto,
			Color:           LogColorAuto,
			AppLogPersist:   true,
			AppLogRetention: 3 * 24 * time.Hour,
		},
		Runtime: RuntimeConfig{
			GinMode: GinModeAuto,
		},
		Database: DatabaseConfig{
			Driver:          "postgres",
			URL:             testDatabaseURL(),
			MaxOpenConns:    defaultDatabaseMaxOpenConns,
			MaxIdleConns:    defaultDatabaseMaxIdleConns,
			ConnMaxLifetime: defaultDatabaseConnMaxLifetime,
			ConnMaxIdleTime: defaultDatabaseConnMaxIdleTime,
		},
		Redis: RedisConfig{
			Addr:            "localhost:6379",
			PoolSize:        defaultRedisPoolSize,
			MinIdleConns:    defaultRedisMinIdleConns,
			MaxIdleConns:    defaultRedisMaxIdleConns,
			MaxActiveConns:  defaultRedisMaxActiveConns,
			PoolTimeout:     defaultRedisPoolTimeout,
			ConnMaxIdleTime: defaultRedisConnMaxIdleTime,
			ConnMaxLifetime: defaultRedisConnMaxLifetime,
		},
		I18n: I18nConfig{
			DefaultLocale:    "zh-CN",
			FallbackLocale:   "zh-CN",
			SupportedLocales: []string{"zh-CN", "en-US"},
		},
		Auth: AuthConfig{
			AccessTokenTTL:        time.Minute,
			RefreshTokenTTL:       time.Hour,
			JWTSecret:             "secret",
			SigningKey:            "signing",
			RefreshCookieName:     defaultRefreshCookieName,
			RefreshCookieSecure:   true,
			RefreshCookieSameSite: defaultRefreshCookieSameSite,
			RefreshCookiePath:     defaultRefreshCookiePath,
		},
		Container: ContainerConfig{
			Runtime:                 "first-adapter",
			DockerEndpoint:          "unix:///var/run/docker.sock",
			LogsDefaultTail:         200,
			LogsMaxTail:             2000,
			RuntimeEnabled:          false,
			DangerousActionsEnabled: false,
			ShellEnabled:            false,
		},
	}
}

func assertValidateError(t *testing.T, cfg *Config, wantErr string) {
	t.Helper()

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected validate error")
	}
	if wantErr != "" && err.Error() != wantErr {
		t.Fatalf("expected validation error %q, got %q", wantErr, err.Error())
	}
}

func testDatabaseURL() string {
	return (&url.URL{
		Scheme: "postgres",
		User:   url.UserPassword("graft", strings.Repeat("g", 5)),
		Host:   "db:5432",
		Path:   "graft",
		RawQuery: url.Values{
			"sslmode": []string{"disable"},
		}.Encode(),
	}).String()
}

func testSigningMaterial(prefix string) string {
	return prefix + "-material-for-config-tests"
}

func prepareAuthSigningMaterialTest(t *testing.T, jwtSecret string, signingKey string) {
	t.Helper()

	restoreEnv := clearGraftEnv(t)
	t.Cleanup(restoreEnv)
	chdir(t, t.TempDir())

	if jwtSecret != "" {
		t.Setenv("GRAFT_AUTH_JWT_SECRET", jwtSecret)
	}
	if signingKey != "" {
		t.Setenv("GRAFT_AUTH_SIGNING_KEY", signingKey)
	}
}

func assertAuthSigningMaterialResult(
	t *testing.T,
	cfg *Config,
	err error,
	wantErr string,
	wantJWTSecret string,
	wantSigningKey string,
) {
	t.Helper()

	if wantErr != "" {
		if err == nil {
			t.Fatal("expected missing auth signing material error")
		}
		if !strings.Contains(err.Error(), wantErr) {
			t.Fatalf("expected error containing %q, got %q", wantErr, err.Error())
		}
		return
	}

	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.Auth.JWTSecret != wantJWTSecret {
		t.Fatalf("expected jwt secret %q, got %q", wantJWTSecret, cfg.Auth.JWTSecret)
	}
	if cfg.Auth.SigningKey != wantSigningKey {
		t.Fatalf("expected signing key %q, got %q", wantSigningKey, cfg.Auth.SigningKey)
	}
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
