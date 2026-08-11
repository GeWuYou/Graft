package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
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
