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
