package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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
}

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
	}

	if err := cfg.Validate(); err == nil {
		t.Fatal("expected unsupported database driver error")
	}
}

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
	}

	if err := cfg.Validate(); err == nil {
		t.Fatal("expected missing database URL error")
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
