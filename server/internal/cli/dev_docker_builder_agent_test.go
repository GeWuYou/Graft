package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/joho/godotenv"
)

func TestReadLocalDockerBuilderServerDatabaseURL(t *testing.T) {
	envFile := filepath.Join(t.TempDir(), ".env")
	const databaseURL = "postgres://graft@127.0.0.1:5432/graft?sslmode=disable"
	if err := os.WriteFile(envFile, []byte("GRAFT_DATABASE_URL="+databaseURL+"\n"), 0o600); err != nil {
		t.Fatalf("write dotenv: %v", err)
	}

	actual, err := readLocalDockerBuilderServerDatabaseURL(envFile)
	if err != nil {
		t.Fatalf("read database URL: %v", err)
	}
	if actual != databaseURL {
		t.Fatalf("database URL = %q, want %q", actual, databaseURL)
	}
}

func TestReadLocalDockerBuilderServerDatabaseURLRequiresDatabaseURL(t *testing.T) {
	envFile := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(envFile, []byte("GRAFT_APP_ENV=local\n"), 0o600); err != nil {
		t.Fatalf("write dotenv: %v", err)
	}

	_, err := readLocalDockerBuilderServerDatabaseURL(envFile)
	if err == nil || !strings.Contains(err.Error(), "GRAFT_DATABASE_URL is required") {
		t.Fatalf("expected missing database URL error, got %v", err)
	}
}

func TestResolveLocalDockerBuilderDatabaseURL(t *testing.T) {
	repositoryRoot := t.TempDir()
	root := filepath.Join(repositoryRoot, ".data", "docker-builder-agent-dev")
	serverDir := filepath.Join(repositoryRoot, "server")
	if err := os.MkdirAll(serverDir, 0o750); err != nil {
		t.Fatalf("mkdir server dir: %v", err)
	}
	const sharedDatabaseURL = "postgres://graft@127.0.0.1:5432/graft?sslmode=disable"
	if err := os.WriteFile(filepath.Join(serverDir, ".env"), []byte("GRAFT_DATABASE_URL="+sharedDatabaseURL+"\n"), 0o600); err != nil {
		t.Fatalf("write server dotenv: %v", err)
	}

	shared, err := resolveLocalDockerBuilderDatabaseURL(root, localDockerBuilderDatabaseModeShared)
	if err != nil {
		t.Fatalf("resolve shared database URL: %v", err)
	}
	if shared != sharedDatabaseURL {
		t.Fatalf("shared database URL = %q, want %q", shared, sharedDatabaseURL)
	}

	isolate, err := resolveLocalDockerBuilderDatabaseURL(root, localDockerBuilderDatabaseModeIsolated)
	if err != nil {
		t.Fatalf("resolve isolated database URL: %v", err)
	}
	if isolate != localDockerBuilderIsolatedDatabaseURL {
		t.Fatalf("isolated database URL = %q, want %q", isolate, localDockerBuilderIsolatedDatabaseURL)
	}
}

func TestWriteLocalDockerBuilderServerEnvPreservesServerConfiguration(t *testing.T) {
	repositoryRoot := t.TempDir()
	root := filepath.Join(repositoryRoot, ".data", "docker-builder-agent-dev")
	serverDir := filepath.Join(repositoryRoot, "server")
	if err := os.MkdirAll(serverDir, 0o750); err != nil {
		t.Fatalf("mkdir server dir: %v", err)
	}
	const databaseURL = "postgres://graft@127.0.0.1:5432/graft?sslmode=disable"
	const websocketOrigins = "http://localhost:3002,http://127.0.0.1:3002"
	contents := "GRAFT_APP_ENV=local\nGRAFT_DATABASE_URL=" + databaseURL + "\nGRAFT_HTTPX_WEBSOCKET_ALLOWED_ORIGINS=" + websocketOrigins + "\nGRAFT_AUTH_JWT_SECRET=server-secret\n"
	if err := os.WriteFile(filepath.Join(serverDir, ".env"), []byte(contents), 0o600); err != nil {
		t.Fatalf("write server dotenv: %v", err)
	}

	if err := writeLocalDockerBuilderServerEnv(root, localDockerBuilderDatabaseModeShared); err != nil {
		t.Fatalf("write local Server integration environment: %v", err)
	}

	values, err := godotenv.Read(localDockerBuilderServerEnvFile(root))
	if err != nil {
		t.Fatalf("read generated Server environment: %v", err)
	}
	if values["GRAFT_HTTPX_WEBSOCKET_ALLOWED_ORIGINS"] != websocketOrigins {
		t.Fatalf("websocket origins = %q, want %q", values["GRAFT_HTTPX_WEBSOCKET_ALLOWED_ORIGINS"], websocketOrigins)
	}
	if values["GRAFT_AUTH_JWT_SECRET"] != "server-secret" {
		t.Fatalf("JWT secret was not preserved")
	}
	if values["GRAFT_DATABASE_URL"] != databaseURL {
		t.Fatalf("database URL = %q, want %q", values["GRAFT_DATABASE_URL"], databaseURL)
	}
	if values["GRAFT_CREDENTIAL_VAULT_ENABLED"] != "true" {
		t.Fatalf("Vault integration was not enabled")
	}
}

func TestLocalDockerBuilderAgentConfigFileUsesAgentRoot(t *testing.T) {
	root := filepath.Join(t.TempDir(), ".data", "docker-builder-agent-dev")
	if got, want := localDockerBuilderAgentConfigFile(root), filepath.Join(filepath.Dir(filepath.Dir(root)), "server", "agents", "docker-builder-agent", "agent.json"); got != want {
		t.Fatalf("Agent config path = %q, want %q", got, want)
	}
}

func TestMigrateLocalDockerBuilderAgentConfigMovesLegacyFileToAgentRoot(t *testing.T) {
	root := filepath.Join(t.TempDir(), ".data", "docker-builder-agent-dev")
	legacyFile := legacyLocalDockerBuilderAgentConfigFile(root)
	if err := os.MkdirAll(filepath.Dir(legacyFile), 0o750); err != nil {
		t.Fatalf("mkdir legacy Agent config directory: %v", err)
	}
	if err := os.WriteFile(legacyFile, []byte(`{"agent_id":"local"}`), 0o600); err != nil {
		t.Fatalf("write legacy Agent config: %v", err)
	}

	if err := migrateLocalDockerBuilderAgentConfig(root); err != nil {
		t.Fatalf("migrate Agent config: %v", err)
	}
	if _, err := os.Stat(legacyFile); !os.IsNotExist(err) {
		t.Fatalf("legacy Agent config still exists: %v", err)
	}
	contents, err := os.ReadFile(localDockerBuilderAgentConfigFile(root))
	if err != nil {
		t.Fatalf("read migrated Agent config: %v", err)
	}
	if string(contents) != `{"agent_id":"local"}` {
		t.Fatalf("migrated Agent config = %q", contents)
	}
}

func TestLocalDockerBuilderComposeDependencyServices(t *testing.T) {
	tests := []struct {
		name string
		mode localDockerBuilderDatabaseMode
		want []string
	}{
		{name: "shared only starts shared dependencies", mode: localDockerBuilderDatabaseModeShared, want: []string{"up", "-d", "redis", "vault"}},
		{name: "isolated starts its PostgreSQL dependency", mode: localDockerBuilderDatabaseModeIsolated, want: []string{"up", "-d", "redis", "vault", "postgres"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := localDockerBuilderComposeDependencyServices(test.mode)
			if strings.Join(got, ",") != strings.Join(test.want, ",") {
				t.Fatalf("dependency services = %v, want %v", got, test.want)
			}
		})
	}
}

func TestReadLocalDockerBuilderServerDatabaseURLRejectsNonDevelopmentEnvironment(t *testing.T) {
	envFile := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(envFile, []byte("GRAFT_APP_ENV=production\nGRAFT_DATABASE_URL=postgres://graft@127.0.0.1:5432/graft?sslmode=disable\n"), 0o600); err != nil {
		t.Fatalf("write dotenv: %v", err)
	}

	_, err := readLocalDockerBuilderServerDatabaseURL(envFile)
	if err == nil || !strings.Contains(err.Error(), "requires local/test GRAFT_APP_ENV") {
		t.Fatalf("expected non-development environment error, got %v", err)
	}
}

func TestNormalizeLocalDockerBuilderDatabaseMode(t *testing.T) {
	mode, err := normalizeLocalDockerBuilderDatabaseMode(" ISOLATED ")
	if err != nil {
		t.Fatalf("normalize database mode: %v", err)
	}
	if mode != localDockerBuilderDatabaseModeIsolated {
		t.Fatalf("database mode = %q, want %q", mode, localDockerBuilderDatabaseModeIsolated)
	}

	_, err = normalizeLocalDockerBuilderDatabaseMode("temporary")
	if err == nil || !strings.Contains(err.Error(), "expected shared or isolated") {
		t.Fatalf("expected unsupported mode error, got %v", err)
	}
}
