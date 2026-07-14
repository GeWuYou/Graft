package project

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateExplicitCanonicalProjectNameRejectsUppercase(t *testing.T) {
	t.Parallel()

	if _, err := validateExplicitCanonicalProjectName("CLIProxyAPI"); err == nil {
		t.Fatal("expected uppercase canonical project name to be rejected")
	}
}

func TestValidateExplicitCanonicalProjectNameNormalizesValidNames(t *testing.T) {
	t.Parallel()

	for input, want := range map[string]string{
		"my-project":     "my-project",
		"orders_dev":     "orders_dev",
		"123abc":         "123abc",
		"  valid-name  ": "valid-name",
	} {
		got, err := validateExplicitCanonicalProjectName(input)
		if err != nil {
			t.Fatalf("validate %q: %v", input, err)
		}
		if got != want {
			t.Fatalf("validate %q returned %q, want %q", input, got, want)
		}
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

func TestChooseWorkspacePathSuggestsNextAvailableSuffix(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "demo"), 0o750); err != nil {
		t.Fatalf("create base workspace: %v", err)
	}
	path, key, err := chooseWorkspacePath(root, stringPointer("demo"), true)
	if !errors.Is(err, errProjectConflict) || !strings.Contains(err.Error(), "suggested=demo-2") || path != "" || key != nil {
		t.Fatalf("chooseWorkspacePath = (%q, %v, %v), want conflict", path, key, err)
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
