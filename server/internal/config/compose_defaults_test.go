package config

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	composecli "github.com/compose-spec/compose-go/v2/cli"
	composetypes "github.com/compose-spec/compose-go/v2/types"
)

func TestOfficialComposeConfigurationDefaultsAndOverrides(t *testing.T) {
	for _, test := range []struct {
		name        string
		overrides   string
		environment []string
		database    string
		user        string
	}{
		{name: "minimal existing file", database: "graft", user: "graft"},
		{name: "empty optional values", overrides: "POSTGRES_DB=\nPOSTGRES_USER=\n", database: "graft", user: "graft"},
		{name: "file overrides", overrides: "POSTGRES_DB=custom_database\nPOSTGRES_USER=custom_user\n", database: "custom_database", user: "custom_user"},
		{name: "environment overrides file", overrides: "POSTGRES_DB=file_database\nPOSTGRES_USER=file_user\n",
			environment: []string{"POSTGRES_DB=process_database", "POSTGRES_USER=process_user"}, database: "process_database", user: "process_user"},
	} {
		t.Run(test.name, func(t *testing.T) {
			fixture := "GRAFT_IMAGE_TAG=v0.12.0-beta.2\nGRAFT_EXECUTION_MODE=local\n" +
				"POSTGRES_PASSWORD=configuration-test-password\nGRAFT_AUTH_JWT_SECRET=configuration-test-signing-material\n" +
				"GRAFT_DOCS_ENABLED=false\nGRAFT_REDIS_DB=0\n" + test.overrides
			options, envFile := composeDefaultsOptions(t, fixture, test.environment)
			project, err := options.LoadProject(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			assertComposeDatabaseIdentity(t, project, test.database, test.user)
			if actual := project.Services["server"].Environment["GRAFT_DOCS_ENABLED"]; actual == nil || *actual != "false" {
				t.Fatal("Compose lost an explicit false override")
			}
			if actual := project.Services["server"].Environment["GRAFT_REDIS_DB"]; actual == nil || *actual != "0" {
				t.Fatal("Compose lost an explicit zero override")
			}
			// #nosec G304 -- envFile 由本测试的 t.TempDir 创建，读取用于核对配置解析保持文件原样。
			if actual, err := os.ReadFile(envFile); err != nil || string(actual) != fixture {
				t.Fatalf("Compose resolution modified the existing environment file: %v", err)
			}
		})
	}
}

func TestOfficialComposeRequiresInstanceDatabasePassword(t *testing.T) {
	options, _ := composeDefaultsOptions(t,
		"GRAFT_IMAGE_TAG=v0.12.0-beta.2\nGRAFT_AUTH_JWT_SECRET=configuration-test-signing-material\n", nil)
	if _, err := options.LoadProject(context.Background()); err == nil || !strings.Contains(err.Error(), "POSTGRES_PASSWORD") {
		t.Fatalf("expected missing instance password to fail before startup, got %v", err)
	}
}

func composeDefaultsOptions(t *testing.T, fixture string, environment []string) (*composecli.ProjectOptions, string) {
	t.Helper()
	contents, err := os.ReadFile(filepath.Join("..", "..", "..", "compose.yml"))
	if err != nil {
		t.Fatal(err)
	}
	directory := t.TempDir()
	composeFile := filepath.Join(directory, "compose.yml")
	// #nosec G703 -- 固定文件名位于本测试独占的 t.TempDir，内容来自仓库官方 Compose 模板。
	if err := os.WriteFile(composeFile, contents, 0o600); err != nil {
		t.Fatal(err)
	}
	envFile := filepath.Join(directory, ".env")
	if err := os.WriteFile(envFile, []byte(fixture), 0o600); err != nil {
		t.Fatal(err)
	}
	options, err := composecli.NewProjectOptions([]string{composeFile}, composecli.WithName("graft-configuration-test"),
		composecli.WithWorkingDirectory(directory), composecli.WithEnv(environment), composecli.WithEnvFiles(envFile), composecli.WithDotEnv)
	if err != nil {
		t.Fatal(err)
	}
	return options, envFile
}

func assertComposeDatabaseIdentity(t *testing.T, project *composetypes.Project, database, user string) {
	t.Helper()
	postgres := project.Services["postgres"]
	for key, expected := range map[string]string{"POSTGRES_DB": database, "POSTGRES_USER": user} {
		actual := postgres.Environment[key]
		if actual == nil || *actual != expected {
			t.Fatalf("database configuration %s did not resolve to %q", key, expected)
		}
	}
	if postgres.HealthCheck == nil || len(postgres.HealthCheck.Test) < 2 || postgres.HealthCheck.Test[1] != "pg_isready -U "+user+" -d "+database {
		t.Fatal("PostgreSQL health check does not match the effective database identity")
	}
	for _, name := range []string{"bootstrap", "server"} {
		value := project.Services[name].Environment["GRAFT_DATABASE_URL"]
		if value == nil || !strings.Contains(*value, "dbname="+database+" ") || !strings.Contains(*value, "user="+user+" ") {
			t.Fatalf("%s database connection does not match PostgreSQL configuration", name)
		}
	}
}
