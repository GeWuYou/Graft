package project

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	projectcontract "graft/server/modules/project/contract"
)

func TestCreateTemplateProjectUsesSharedCreationPipeline(t *testing.T) {
	managedRoot := t.TempDir()
	repository := &stubProjectRepository{}
	service, err := NewService(repository, WithSystemConfigResolver(stubSystemConfigResolver{value: managedRoot}))
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	result, err := service.CreateTemplateProject(context.Background(), TemplateProjectCreateRequest{
		DisplayName: "Template project", CanonicalProjectName: "template-project", RelativeProjectDirectory: "template-project", TemplateKey: defaultTemplateKey, TemplateVersion: defaultTemplateVersion,
	}, nil)
	if err != nil {
		t.Fatalf("create template project: %v", err)
	}
	if result.SourceType != projectcontract.SourceKindTemplate.String() {
		t.Fatalf("expected template source type, got %q", result.SourceType)
	}
	if repository.importInput == nil || repository.importInput.SourceKind != projectcontract.SourceKindTemplate.String() {
		t.Fatalf("expected shared creation aggregate for template, got %#v", repository.importInput)
	}
	if repository.importInput.SourceMetadata["template_key"] != defaultTemplateKey || repository.importInput.SourceMetadata["template_version"] != defaultTemplateVersion {
		t.Fatalf("expected safe template provenance, got %#v", repository.importInput.SourceMetadata)
	}
	if _, err := os.Stat(filepath.Join(managedRoot, "template-project", "compose.yaml")); err != nil {
		t.Fatalf("expected template compose workspace: %v", err)
	}
}

func TestResolveGitWorkspaceRejectsTraversalSubpath(t *testing.T) {
	_, err := normalizeGitComposeSubpath("../outside")
	if err == nil {
		t.Fatal("expected traversal subpath rejection")
	}
}

func TestResolveGitWorkspaceUsesIsolatedLocalCheckout(t *testing.T) {
	repository := t.TempDir()
	runGitTestCommand(t, repository, "init", "-q")
	runGitTestCommand(t, repository, "config", "user.email", "project-test@example.invalid")
	runGitTestCommand(t, repository, "config", "user.name", "Project Test")
	if err := os.MkdirAll(filepath.Join(repository, "deploy"), 0o750); err != nil {
		t.Fatalf("create compose directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repository, "deploy", "compose.yaml"), []byte("services: {}\n"), 0o600); err != nil {
		t.Fatalf("write compose: %v", err)
	}
	runGitTestCommand(t, repository, "add", ".")
	runGitTestCommand(t, repository, "commit", "-qm", "fixture")

	workspace, metadata, err := resolveGitWorkspace(context.Background(), GitProjectCreateRequest{RepositoryURL: repository, ComposeSubpath: "deploy"})
	if err != nil {
		t.Fatalf("resolve git workspace: %v", err)
	}
	if workspace.ComposeFilePath != "compose.yaml" || len(workspace.WorkspaceFiles) != 1 || metadata["git_compose_subpath"] != "deploy" {
		t.Fatalf("expected staged compose workspace and safe provenance, got workspace=%#v metadata=%#v", workspace, metadata)
	}
	if _, err := os.Stat(filepath.Join(repository, "graft-project-git-unexpected")); !os.IsNotExist(err) {
		t.Fatalf("git staging must not occur inside repository: %v", err)
	}
}

func runGitTestCommand(t *testing.T, directory string, args ...string) {
	t.Helper()
	//nolint:gosec // Test fixture invokes only fixed Git subcommands against t.TempDir().
	command := exec.Command("git", append([]string{"-C", directory}, args...)...)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v (%s)", args, err, output)
	}
}
