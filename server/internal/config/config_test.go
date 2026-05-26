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
		"GRAFT_REDIS_ADDR=redis:6379",
		"GRAFT_REDIS_DB=2",
		"GRAFT_LOG_LEVEL=debug",
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
	assertEqual(t, "Redis address from .env", cfg.Redis.Addr, "redis:6379")
	assertEqual(t, "Redis DB from .env", cfg.Redis.DB, 2)
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
	assertEqual(t, "default database driver", cfg.Database.Driver, defaultDatabaseDriver)
	assertEqual(t, "default database URL", cfg.Database.URL, defaultDatabaseURL)
	assertEqual(t, "default Redis address", cfg.Redis.Addr, defaultRedisAddr)
	assertEqual(t, "default log level", cfg.Log.Level, defaultLogLevel)
	assertEqual(t, "default locale", cfg.I18n.DefaultLocale, defaultLocale)
	assertEqual(t, "fallback locale", cfg.I18n.FallbackLocale, defaultLocale)
	assertStringSliceEqual(t, "supported locales", cfg.I18n.SupportedLocales, []string{defaultLocale, defaultSecondaryLocale})
	assertEqual(t, "default access token ttl", cfg.Auth.AccessTokenTTL, defaultAccessTokenTTL)
	assertEqual(t, "jwt secret from environment", cfg.Auth.JWTSecret, "runtime-secret")
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
		Database: DatabaseConfig{
			Driver: "postgres",
			URL:    testDatabaseURL(),
		},
		Redis: RedisConfig{
			Addr: "localhost:6379",
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
