package compose

import (
	"os"
	"path/filepath"
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
		WorkingDirectory: workingDirectory,
		ComposeFiles:     []string{composePath},
	})
	if err != nil {
		t.Fatalf("load compose project: %v", err)
	}
	if result.CanonicalProjectName != "cliproxyapi" {
		t.Fatalf("expected normalized top-level project name, got %q", result.CanonicalProjectName)
	}
}

func TestLoadFallsBackToNormalizedWorkingDirectoryName(t *testing.T) {
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

	result, err := Load(Input{WorkingDirectory: workingDirectory, ComposeFiles: []string{composePath}})
	if err != nil {
		t.Fatalf("load compose project: %v", err)
	}
	if result.CanonicalProjectName != "cli-proxy-api" {
		t.Fatalf("expected normalized working-directory fallback, got %q", result.CanonicalProjectName)
	}
}
