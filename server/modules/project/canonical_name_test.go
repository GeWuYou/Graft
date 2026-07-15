package project

import (
	"context"
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

func TestCheckApplicationNameAvailabilityDistinguishesRegistryAndWorkspace(t *testing.T) {
	managedRoot := t.TempDir()
	repository := &stubProjectRepository{}
	service, err := NewService(repository, WithSystemConfigResolver(stubSystemConfigResolver{value: managedRoot}))
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	result, err := service.CheckApplicationNameAvailability(context.Background(), ApplicationNameAvailabilityRequest{ApplicationName: "demo"})
	assertApplicationNameAvailability(t, result, err, applicationNameAvailabilityAvailable, false, "")
	if err := os.Mkdir(filepath.Join(managedRoot, "demo"), 0o750); err != nil {
		t.Fatalf("create empty workspace: %v", err)
	}
	result, err = service.CheckApplicationNameAvailability(context.Background(), ApplicationNameAvailabilityRequest{ApplicationName: "demo"})
	assertApplicationNameAvailability(t, result, err, applicationNameAvailabilityReusable, false, "")
	compose := "services: {}\n"
	if err := os.WriteFile(filepath.Join(managedRoot, "demo", "compose.yaml"), []byte(compose), 0o600); err != nil {
		t.Fatalf("write reusable compose: %v", err)
	}
	result, err = service.CheckApplicationNameAvailability(context.Background(), ApplicationNameAvailabilityRequest{ApplicationName: "demo"})
	assertApplicationNameAvailability(t, result, err, applicationNameAvailabilityReusable, true, "compose.yaml")
	repository.aggregate.Project.ID = 7
	repository.aggregate.Project.ApplicationName = stringPointer("demo")
	result, err = service.CheckApplicationNameAvailability(context.Background(), ApplicationNameAvailabilityRequest{ApplicationName: "demo"})
	assertApplicationNameAvailability(t, result, err, applicationNameAvailabilityRegistered, false, "")
}

func assertApplicationNameAvailability(t *testing.T, result ApplicationNameAvailabilityResult, err error, status string, nonEmpty bool, composePath string) {
	t.Helper()
	if err != nil {
		t.Fatalf("check application name availability: %v", err)
	}
	if result.Status != status {
		t.Fatalf("availability status = %q, want %q", result.Status, status)
	}
	if result.WorkspaceNonEmpty != nonEmpty {
		t.Fatalf("workspace non-empty = %t, want %t", result.WorkspaceNonEmpty, nonEmpty)
	}
	if composePath == "" {
		if result.ComposeFilePath != nil {
			t.Fatalf("unexpected compose path: %q", *result.ComposeFilePath)
		}
		return
	}
	if result.ComposeFilePath == nil || *result.ComposeFilePath != composePath {
		t.Fatalf("compose path = %v, want %q", result.ComposeFilePath, composePath)
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
