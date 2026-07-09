package project

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateExplicitCanonicalProjectNameRejectsUppercase(t *testing.T) {
	t.Parallel()

	if _, err := validateExplicitCanonicalProjectName("CLIProxyAPI"); err == nil {
		t.Fatal("expected uppercase canonical project name to be rejected")
	}
}

func TestNormalizeManagedCreateRequestRejectsInvalidCanonicalProjectName(t *testing.T) {
	t.Parallel()

	_, err := normalizeManagedCreateRequest(ManagedProjectCreateRequest{
		DisplayName:              "Demo",
		CanonicalProjectName:     "CLIProxyAPI",
		RelativeProjectDirectory: "demo",
		ComposeFileName:          "compose.yaml",
		ComposeFileContent:       "services:\n  app:\n    image: nginx:latest\n",
	})
	if err == nil {
		t.Fatal("expected managed create request to reject invalid canonical project name")
	}
}

func TestParseImportRequestRejectsInvalidCanonicalProjectNameOverride(t *testing.T) {
	t.Parallel()

	workingDirectory := t.TempDir()
	composePath := filepath.Join(workingDirectory, "compose.yaml")
	content := []byte("services:\n  api:\n    image: nginx:latest\n")
	if err := os.WriteFile(composePath, content, 0o600); err != nil {
		t.Fatalf("write compose file: %v", err)
	}

	override := "CLIProxyAPI"
	service := Service{}
	_, _, err := service.parseImportRequest(ImportRequest{
		WorkingDirectory:             workingDirectory,
		ComposeFiles:                 []string{composePath},
		CanonicalProjectNameOverride: &override,
	})
	if err == nil {
		t.Fatal("expected import override to reject invalid canonical project name")
	}
}

func TestDefaultImportedDisplayNamePreservesDirectoryBasename(t *testing.T) {
	t.Parallel()

	got := defaultImportedDisplayName(nil, "/srv/CLIProxyAPI", "cliproxyapi")
	if got != "CLIProxyAPI" {
		t.Fatalf("expected original directory basename, got %q", got)
	}
}
