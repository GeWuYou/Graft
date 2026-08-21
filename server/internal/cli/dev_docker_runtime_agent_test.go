package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/joho/godotenv"
)

func TestReadLocalDockerRuntimeServerDatabaseURL(t *testing.T) {
	envFile := filepath.Join(t.TempDir(), ".env")
	const databaseURL = "postgres://graft@127.0.0.1:5432/graft?sslmode=disable"
	if err := os.WriteFile(envFile, []byte("GRAFT_APP_ENV=local\nGRAFT_DATABASE_URL="+databaseURL+"\n"), 0o600); err != nil {
		t.Fatalf("write dotenv: %v", err)
	}

	actual, err := readLocalDockerRuntimeServerDatabaseURL(envFile)
	if err != nil {
		t.Fatalf("read database URL: %v", err)
	}
	if actual != databaseURL {
		t.Fatalf("database URL = %q, want %q", actual, databaseURL)
	}
}

func TestReadLocalDockerRuntimeServerDatabaseURLRequiresDatabaseURL(t *testing.T) {
	envFile := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(envFile, []byte("GRAFT_APP_ENV=local\n"), 0o600); err != nil {
		t.Fatalf("write dotenv: %v", err)
	}

	_, err := readLocalDockerRuntimeServerDatabaseURL(envFile)
	if err == nil || !strings.Contains(err.Error(), "GRAFT_DATABASE_URL is required") {
		t.Fatalf("expected missing database URL error, got %v", err)
	}
}

func TestResolveLocalDockerRuntimeDatabaseURL(t *testing.T) {
	repositoryRoot := t.TempDir()
	root := filepath.Join(repositoryRoot, ".data", "docker-runtime-agent-dev")
	serverDir := filepath.Join(repositoryRoot, "server")
	if err := os.MkdirAll(serverDir, 0o750); err != nil {
		t.Fatalf("mkdir server dir: %v", err)
	}
	const sharedDatabaseURL = "postgres://graft@127.0.0.1:5432/graft?sslmode=disable"
	if err := os.WriteFile(filepath.Join(serverDir, ".env"), []byte("GRAFT_APP_ENV=local\nGRAFT_DATABASE_URL="+sharedDatabaseURL+"\n"), 0o600); err != nil {
		t.Fatalf("write server dotenv: %v", err)
	}

	shared, err := resolveLocalDockerRuntimeDatabaseURL(root, localDockerRuntimeDatabaseModeShared)
	if err != nil {
		t.Fatalf("resolve shared database URL: %v", err)
	}
	if shared != sharedDatabaseURL {
		t.Fatalf("shared database URL = %q, want %q", shared, sharedDatabaseURL)
	}

	isolate, err := resolveLocalDockerRuntimeDatabaseURL(root, localDockerRuntimeDatabaseModeIsolated)
	if err != nil {
		t.Fatalf("resolve isolated database URL: %v", err)
	}
	if isolate != localDockerRuntimeIsolatedDatabaseURL {
		t.Fatalf("isolated database URL = %q, want %q", isolate, localDockerRuntimeIsolatedDatabaseURL)
	}
}

func TestWriteLocalDockerRuntimeServerEnvPreservesServerConfiguration(t *testing.T) {
	repositoryRoot := t.TempDir()
	root := filepath.Join(repositoryRoot, ".data", "docker-runtime-agent-dev")
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

	if err := writeLocalDockerRuntimeServerEnv(root, localDockerRuntimeDatabaseModeShared); err != nil {
		t.Fatalf("write local Server integration environment: %v", err)
	}

	values, err := godotenv.Read(localDockerRuntimeServerEnvFile(root))
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

func TestLocalDockerRuntimeAgentConfigFileUsesAgentRoot(t *testing.T) {
	root := filepath.Join(t.TempDir(), ".data", "docker-runtime-agent-dev")
	if got, want := localDockerRuntimeAgentConfigFile(root), filepath.Join(filepath.Dir(filepath.Dir(root)), "server", "agents", "docker-runtime-agent", "agent.json"); got != want {
		t.Fatalf("Agent config path = %q, want %q", got, want)
	}
}

func TestMigrateLocalDockerRuntimeAgentConfigMovesLegacyFileToAgentRoot(t *testing.T) {
	root := filepath.Join(t.TempDir(), ".data", "docker-runtime-agent-dev")
	legacyFile := legacyLocalDockerRuntimeAgentConfigFile(root)
	if err := os.MkdirAll(filepath.Dir(legacyFile), 0o750); err != nil {
		t.Fatalf("mkdir legacy Agent config directory: %v", err)
	}
	if err := os.WriteFile(legacyFile, []byte(`{"agent_id":"local"}`), 0o600); err != nil {
		t.Fatalf("write legacy Agent config: %v", err)
	}

	if err := migrateLocalDockerRuntimeAgentConfig(root); err != nil {
		t.Fatalf("migrate Agent config: %v", err)
	}
	if _, err := os.Stat(legacyFile); !os.IsNotExist(err) {
		t.Fatalf("legacy Agent config still exists: %v", err)
	}
	contents, err := os.ReadFile(localDockerRuntimeAgentConfigFile(root))
	if err != nil {
		t.Fatalf("read migrated Agent config: %v", err)
	}
	if string(contents) != `{"agent_id":"local"}` {
		t.Fatalf("migrated Agent config = %q", contents)
	}
}

func TestLocalDockerRuntimeComposeDependencyServices(t *testing.T) {
	tests := []struct {
		name string
		mode localDockerRuntimeDatabaseMode
		want []string
	}{
		{name: "shared only starts shared dependencies", mode: localDockerRuntimeDatabaseModeShared, want: []string{"up", "-d", "redis", "vault"}},
		{name: "isolated starts its PostgreSQL dependency", mode: localDockerRuntimeDatabaseModeIsolated, want: []string{"up", "-d", "redis", "vault", "postgres"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := localDockerRuntimeComposeDependencyServices(test.mode)
			if strings.Join(got, ",") != strings.Join(test.want, ",") {
				t.Fatalf("dependency services = %v, want %v", got, test.want)
			}
		})
	}
}

func TestReadLocalDockerRuntimeServerDatabaseURLRejectsNonDevelopmentEnvironment(t *testing.T) {
	envFile := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(envFile, []byte("GRAFT_APP_ENV=production\nGRAFT_DATABASE_URL=postgres://graft@127.0.0.1:5432/graft?sslmode=disable\n"), 0o600); err != nil {
		t.Fatalf("write dotenv: %v", err)
	}

	_, err := readLocalDockerRuntimeServerDatabaseURL(envFile)
	if err == nil || !strings.Contains(err.Error(), "requires local/test GRAFT_APP_ENV") {
		t.Fatalf("expected non-development environment error, got %v", err)
	}
}

func TestNormalizeLocalDockerRuntimeDatabaseMode(t *testing.T) {
	mode, err := normalizeLocalDockerRuntimeDatabaseMode(" ISOLATED ")
	if err != nil {
		t.Fatalf("normalize database mode: %v", err)
	}
	if mode != localDockerRuntimeDatabaseModeIsolated {
		t.Fatalf("database mode = %q, want %q", mode, localDockerRuntimeDatabaseModeIsolated)
	}

	_, err = normalizeLocalDockerRuntimeDatabaseMode("temporary")
	if err == nil || !strings.Contains(err.Error(), "expected shared or isolated") {
		t.Fatalf("expected unsupported mode error, got %v", err)
	}
}
