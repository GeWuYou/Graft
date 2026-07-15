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

func TestNormalizeManagedCreateRequestUsesApplicationNameForComposeIdentity(t *testing.T) {
	t.Parallel()

	compose := "services:\n  app:\n    image: nginx:latest\n"
	result, err := normalizeManagedCreateRequest(ManagedProjectCreateRequest{
		DisplayName:        "拉布",
		ApplicationName:    stringPointer("labby"),
		ComposeFileName:    "compose.yaml",
		ComposeFilePath:    "compose.yaml",
		ComposeFileContent: compose,
		WorkspaceEntries: []ManagedWorkspaceEntry{
			{Path: "compose.yaml", NodeType: "file", Content: &compose},
		},
	})
	if err != nil {
		t.Fatalf("normalize managed create request: %v", err)
	}
	if result.DisplayName != "拉布" || result.ApplicationName == nil || *result.ApplicationName != "labby" {
		t.Fatalf("unexpected normalized identity: %#v", result)
	}
	if !strings.Contains(result.ComposeFileContent, "name: labby") {
		t.Fatalf("expected Compose name from application name, got %q", result.ComposeFileContent)
	}
}

func TestNormalizeManagedCreateRequestRequiresApplicationName(t *testing.T) {
	t.Parallel()

	for _, applicationName := range []*string{nil, stringPointer("   ")} {
		_, err := normalizeManagedCreateRequest(ManagedProjectCreateRequest{
			DisplayName:        "拉布",
			ApplicationName:    applicationName,
			ComposeFileName:    "compose.yaml",
			ComposeFileContent: "services: {}\n",
		})
		if !errors.Is(err, errProjectApplicationNameRequired) {
			t.Fatalf("expected required application-name error, got %v", err)
		}
	}
}

func TestChooseWorkspacePathSuggestsNextAvailableSuffix(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "demo"), 0o750); err != nil {
		t.Fatalf("create base workspace: %v", err)
	}
	path, key, err := chooseWorkspacePath(root, stringPointer("demo"), true)
	if !errors.Is(err, errProjectApplicationNameOccupied) || path != "" || key != nil {
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
