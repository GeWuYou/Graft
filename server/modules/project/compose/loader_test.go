package compose

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestLoadPrefersTopLevelProjectNameAndNormalizesIt(t *testing.T) {
	t.Parallel()

	workingDirectory := t.TempDir()
	composePath := filepath.Join(workingDirectory, "compose.yaml")
	content := []byte("name: CLIProxyAPI\nservices:\n  api:\n    image: nginx:latest\n")
	if err := os.WriteFile(composePath, content, 0o600); err != nil {
		t.Fatalf("write compose file: %v", err)
	}

	result, err := Load(Input{
		WorkspacePath: workingDirectory,
		ComposeFiles:  []string{composePath},
	})
	if err != nil {
		t.Fatalf("load compose project: %v", err)
	}
	if result.ComposeProjectName != "cliproxyapi" {
		t.Fatalf("expected normalized top-level project name, got %q", result.ComposeProjectName)
	}
}

func TestLoadExtractsDependsOnListAndMappingForms(t *testing.T) {
	t.Parallel()

	workingDirectory := t.TempDir()
	composePath := filepath.Join(workingDirectory, "compose.yaml")
	content := []byte("services:\n  api:\n    image: nginx\n    depends_on:\n      - db\n  worker:\n    image: nginx\n    depends_on:\n      db:\n        condition: service_healthy\n      cache:\n        condition: service_started\n  db:\n    image: postgres\n  cache:\n    image: redis\n")
	if err := os.WriteFile(composePath, content, 0o600); err != nil {
		t.Fatalf("write compose file: %v", err)
	}

	result, err := Load(Input{WorkspacePath: workingDirectory, ComposeFiles: []string{composePath}})
	if err != nil {
		t.Fatalf("load compose project: %v", err)
	}
	dependencies := make(map[string][]string, len(result.Services))
	for _, service := range result.Services {
		dependencies[service.ServiceName] = service.DependsOn
	}
	if !slices.Equal(dependencies["api"], []string{"db"}) {
		t.Fatalf("api dependencies = %#v", dependencies["api"])
	}
	if !slices.Equal(dependencies["worker"], []string{"cache", "db"}) {
		t.Fatalf("worker dependencies = %#v", dependencies["worker"])
	}
}

func TestLoadFallsBackToNormalizedWorkspacePathName(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	workingDirectory := filepath.Join(root, "CLI Proxy API")
	if err := os.Mkdir(workingDirectory, 0o700); err != nil {
		t.Fatalf("create working directory: %v", err)
	}
	composePath := filepath.Join(workingDirectory, "compose.yaml")
	if err := os.WriteFile(composePath, []byte("services:\n  api:\n    image: nginx:latest\n"), 0o600); err != nil {
		t.Fatalf("write compose file: %v", err)
	}

	result, err := Load(Input{WorkspacePath: workingDirectory, ComposeFiles: []string{composePath}})
	if err != nil {
		t.Fatalf("load compose project: %v", err)
	}
	if result.ComposeProjectName != "cli-proxy-api" {
		t.Fatalf("expected normalized working-directory fallback, got %q", result.ComposeProjectName)
	}
}
