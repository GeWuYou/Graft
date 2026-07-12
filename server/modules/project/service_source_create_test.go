package project

import (
	"context"
	"os"
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
