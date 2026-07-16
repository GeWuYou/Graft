package project

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	projectcompose "graft/server/modules/project/compose"
	projectcontract "graft/server/modules/project/contract"
)

func TestValidateManagedCreateAcceptsLabbyWorkspaceManifest(t *testing.T) {
	managedRoot := t.TempDir()
	service, err := NewService(&stubProjectRepository{}, WithSystemConfigResolver(stubSystemConfigResolver{value: managedRoot}))
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	compose := "services:\n  labby:\n    image: ghcr.io/samuelloranger/labby:latest\n    restart: unless-stopped\n    ports:\n      - \\\"8080:8080\\\"\n    volumes:\n      - ./config:/app/config\n"
	dashboard := "{\n  \\\"title\\\": \\\"Labby E2E\\\"\n}\n"
	result, err := service.ValidateManagedCreate(context.Background(), ManagedProjectCreateRequest{
		DisplayName:        "拉布",
		RuntimeTargetID:    1,
		ApplicationName:    stringPointer("labby"),
		ComposeFileName:    "compose.yaml",
		ComposeFilePath:    "compose.yaml",
		ComposeFileContent: compose,
		WorkspaceEntries: []ManagedWorkspaceEntry{
			{Path: "config", NodeType: "directory"},
			{Path: "compose.yaml", NodeType: "file", Content: &compose},
			{Path: "config/dashboard.e2e.json", NodeType: "file", Content: &dashboard},
		},
	})
	if err != nil {
		t.Fatalf("validate Labby workspace: %v", err)
	}
	if result.ComposeProjectName != "labby" {
		t.Fatalf("expected labby compose name, got %q", result.ComposeProjectName)
	}
	if _, ok := result.SourceMetadata["managed_env_file_name"]; ok {
		t.Fatalf("expected absent env metadata when no env file is declared, got %#v", result.SourceMetadata)
	}
}

func TestValidateManagedCreateClassifiesManagedRootAndWorkspaceFailures(t *testing.T) {
	request := managedCreateDiagnosticRequest()

	t.Run("invalid managed root", func(t *testing.T) {
		service, err := NewService(&stubProjectRepository{}, WithSystemConfigResolver(stubSystemConfigResolver{value: "relative-root"}))
		if err != nil {
			t.Fatalf("new service: %v", err)
		}
		_, err = service.ValidateManagedCreate(context.Background(), request)
		if !errors.Is(err, errProjectManagedRootInvalid) {
			t.Fatalf("expected managed-root-invalid error, got %v", err)
		}
	})

	t.Run("unsafe existing workspace", func(t *testing.T) {
		managedRoot := t.TempDir()
		if err := os.WriteFile(filepath.Join(managedRoot, "labby"), []byte("not a directory"), 0o600); err != nil {
			t.Fatalf("write unsafe workspace fixture: %v", err)
		}
		service, err := NewService(&stubProjectRepository{}, WithSystemConfigResolver(stubSystemConfigResolver{value: managedRoot}))
		if err != nil {
			t.Fatalf("new service: %v", err)
		}
		request.ReuseExistingWorkspace = true
		_, err = service.ValidateManagedCreate(context.Background(), request)
		if !errors.Is(err, errProjectWorkspaceUnsafe) || !errors.Is(err, errProjectInvalidArgument) {
			t.Fatalf("expected classified unsafe workspace error, got %v", err)
		}
	})
}

func TestValidateManagedCreateClassifiesInvalidCompose(t *testing.T) {
	managedRoot := t.TempDir()
	service, err := NewService(&stubProjectRepository{}, WithSystemConfigResolver(stubSystemConfigResolver{value: managedRoot}))
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	request := managedCreateDiagnosticRequest()
	invalidCompose := "services: ["
	request.ComposeFileContent = invalidCompose
	request.WorkspaceEntries[0].Content = &invalidCompose

	_, err = service.ValidateManagedCreate(context.Background(), request)
	if !errors.Is(err, errProjectInvalidCompose) || !errors.Is(err, errProjectInvalidArgument) {
		t.Fatalf("expected classified invalid compose error, got %v", err)
	}
}

func TestManagedCreationCommandUsesComputedCanonicalNameSource(t *testing.T) {
	command := managedCreationCommand(
		ManagedProjectCreateValidationResult{ComposeProjectName: "labby", WorkspacePath: "/srv/applications/labby"},
		normalizedManagedCreateRequest{DisplayName: "拉布"},
		projectcompose.Result{},
		nil,
	)
	if command.CanonicalProjectNameSource != projectcontract.CanonicalProjectNameSourceComputed.String() {
		t.Fatalf("expected computed canonical name source, got %q", command.CanonicalProjectNameSource)
	}
}

func managedCreateDiagnosticRequest() ManagedProjectCreateRequest {
	compose := "services:\n  labby:\n    image: ghcr.io/samuelloranger/labby:latest\n"
	return ManagedProjectCreateRequest{
		DisplayName:        "拉布",
		RuntimeTargetID:    1,
		ApplicationName:    stringPointer("labby"),
		ComposeFileName:    "compose.yaml",
		ComposeFilePath:    "compose.yaml",
		ComposeFileContent: compose,
		WorkspaceEntries:   []ManagedWorkspaceEntry{{Path: "compose.yaml", NodeType: "file", Content: &compose}},
	}
}
