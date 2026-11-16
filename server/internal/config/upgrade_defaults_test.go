package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExistingEnvironmentUsesCompatibleDefaults(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".env")
	contents := "GRAFT_AUTH_JWT_SECRET=upgrade-configuration-test\nGRAFT_LOG_LEVEL=warn\nGRAFT_HTTP_ADDR=:19090\n"
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
	report, err := ResolveAndValidate(ResolveOptions{EnvFile: path, Environment: map[string]string{}})
	if err != nil {
		t.Fatalf("existing environment must inherit optional defaults: %v\n%s", err, report.FormatText())
	}
	for key, expected := range map[string]string{
		"GRAFT_CONFIG_SCHEMA_VERSION": "1",
		"GRAFT_EXECUTION_MODE":        ExecutionModeAgent,
		"GRAFT_BACKUP_ARTIFACT_ROOT":  defaultBackupArtifactRoot,
	} {
		value := report.Values[key]
		if value.Value != expected || value.Source != SourceDefault {
			t.Fatalf("%s must use its compatible default, got %#v", key, value)
		}
	}
	if value := report.Values["GRAFT_LOG_LEVEL"]; value.Value != "warn" || value.Source != SourceEnvFile {
		t.Fatalf("existing override was lost: %#v", value)
	}
	// #nosec G304 -- path 位于本测试的 t.TempDir，读取用于核对已有配置文件保持原样。
	if actual, err := os.ReadFile(path); err != nil || string(actual) != contents {
		t.Fatalf("configuration resolution modified the existing file: %v", err)
	}
}

func TestConfigurationVersionDefaultKeepsExplicitValidation(t *testing.T) {
	for _, version := range []string{"0", "2", "invalid"} {
		t.Run(version, func(t *testing.T) {
			report, err := ResolveAndValidate(ResolveOptions{Environment: map[string]string{
				"GRAFT_AUTH_JWT_SECRET":       strings.Repeat("test-key-", 4),
				"GRAFT_CONFIG_SCHEMA_VERSION": version,
			}})
			if err == nil || report.ErrorCount() == 0 {
				t.Fatal("explicit incompatible configuration version was accepted")
			}
		})
	}
	report, err := ResolveAndValidate(ResolveOptions{Environment: map[string]string{}})
	if err == nil || report.ErrorCount() != 1 || report.Findings[0].Code != "required_any_of" {
		t.Fatalf("signing material must remain required: %#v, %v", report.Findings, err)
	}
}

func TestLoadMinimalEnvironmentPreservesOverridesAndDefaults(t *testing.T) {
	restoreEnv := clearGraftEnv(t)
	t.Cleanup(restoreEnv)
	chdir(t, t.TempDir())
	contents := strings.Join([]string{
		"GRAFT_AUTH_JWT_SECRET=upgrade-configuration-test",
		"GRAFT_HTTP_ADDR=:19090",
		"GRAFT_LOG_LEVEL=warn",
		"GRAFT_DOCS_ENABLED=false",
		"GRAFT_LOG_APP_LOG_PERSIST=false",
		"GRAFT_DATABASE_MAX_IDLE_CONNS=0",
	}, "\n")
	if err := os.WriteFile(".env", []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GRAFT_HTTP_ADDR", ":19091")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.HTTP.Addr != ":19091" || cfg.Log.Level != "warn" {
		t.Fatalf("environment and file overrides were lost: HTTP=%q level=%q", cfg.HTTP.Addr, cfg.Log.Level)
	}
	if cfg.Docs.Enabled || cfg.Log.AppLogPersist || cfg.Database.MaxIdleConns != 0 {
		t.Fatal("explicit false or zero was replaced by a default")
	}
	if cfg.MCP.ConfirmationTokenTTL != defaultMCPConfirmationTokenTTL || cfg.Backup.ArtifactRoot != defaultBackupArtifactRoot || cfg.Execution.Mode != ExecutionModeAgent {
		t.Fatal("missing optional configuration did not inherit its default")
	}
	if actual, err := os.ReadFile(".env"); err != nil || string(actual) != contents {
		t.Fatalf("runtime loading modified the existing configuration file: %v", err)
	}
}
