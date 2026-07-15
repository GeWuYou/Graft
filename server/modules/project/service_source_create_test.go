package project

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"graft/server/internal/moduleapi"
	projectcontract "graft/server/modules/project/contract"
	projectstore "graft/server/modules/project/store"
)

type stubComposeRuntimeTargetReader struct {
	target moduleapi.ComposeRuntimeTargetSummary
	state  moduleapi.ComposeProjectNameState
}

func (s stubComposeRuntimeTargetReader) ReadComposeTarget(_ context.Context, _ *int64) (moduleapi.ComposeRuntimeTargetSummary, error) {
	return s.target, nil
}

func (s stubComposeRuntimeTargetReader) ListComposeTargets(context.Context) ([]moduleapi.ComposeRuntimeTargetSummary, error) {
	return []moduleapi.ComposeRuntimeTargetSummary{s.target}, nil
}

func (s stubComposeRuntimeTargetReader) CheckComposeProjectName(context.Context, int64, string) (moduleapi.ComposeProjectNameAvailability, error) {
	return moduleapi.ComposeProjectNameAvailability{State: s.state}, nil
}

func composeTargetReader(state moduleapi.ComposeProjectNameState) stubComposeRuntimeTargetReader {
	return stubComposeRuntimeTargetReader{target: moduleapi.ComposeRuntimeTargetSummary{ID: 7, Available: state != moduleapi.ComposeProjectNameStateUnavailable}, state: state}
}

func managedWorkspaceEntries(compose string) []ManagedWorkspaceEntry {
	return []ManagedWorkspaceEntry{{Path: "compose.yaml", NodeType: "file", Content: &compose}}
}

func TestCreateTemplateProjectUsesSharedCreationPipeline(t *testing.T) {
	managedRoot := t.TempDir()
	repository := &stubProjectRepository{}
	service, err := NewService(repository, WithSystemConfigResolver(stubSystemConfigResolver{value: managedRoot}))
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	result, err := service.CreateTemplateProject(context.Background(), TemplateProjectCreateRequest{
		DisplayName: "Template project", ApplicationName: stringPointer("template-project"), TemplateKey: defaultTemplateKey, TemplateVersion: defaultTemplateVersion,
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

func TestValidateTemplateProjectDoesNotSeedDefaultTemplate(t *testing.T) {
	root := t.TempDir()
	service, err := NewService(&stubProjectRepository{}, WithSystemConfigResolver(stubSystemConfigResolver{value: root}))
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	_, err = service.ValidateTemplateProject(context.Background(), TemplateProjectCreateRequest{DisplayName: "Template project"})
	if !errors.Is(err, errProjectInvalidArgument) {
		t.Fatalf("expected unavailable template error, got %v", err)
	}
	if _, statErr := os.Stat(filepath.Join(root, "templates")); !os.IsNotExist(statErr) {
		t.Fatalf("validation must not create template files, stat error = %v", statErr)
	}
}

func TestResolveTemplateWorkspaceRejectsUnsupportedVersion(t *testing.T) {
	service, err := NewService(&stubProjectRepository{}, WithSystemConfigResolver(stubSystemConfigResolver{value: t.TempDir()}))
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	_, _, err = service.resolveTemplateWorkspace(context.Background(), TemplateProjectCreateRequest{TemplateVersion: "v2"}, false)
	if !errors.Is(err, errProjectInvalidArgument) {
		t.Fatalf("expected unsupported version rejection, got %v", err)
	}
}

func TestBlankCreatePrefillDefaultTemplatePropagatesConfigErrors(t *testing.T) {
	service, err := NewService(&stubProjectRepository{}, WithSystemConfigResolver(stubSystemConfigResolver{err: errors.New("config unavailable")}))
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	_, err = service.blankCreatePrefillDefaultTemplate(context.Background())
	if !errors.Is(err, errProjectInvalidArgument) {
		t.Fatalf("expected configuration failure to propagate, got %v", err)
	}
}

func TestCreateManagedProjectChecksRuntimeComposeNameWithoutRetainingFailedWorkspace(t *testing.T) {
	managedRoot := t.TempDir()
	repository := &stubProjectRepository{}
	service, err := NewService(repository,
		WithSystemConfigResolver(stubSystemConfigResolver{value: managedRoot}),
		WithRuntimeTargetReader(composeTargetReader(moduleapi.ComposeProjectNameStateOccupied)),
	)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	_, err = service.CreateManagedProject(context.Background(), ManagedProjectCreateRequest{
		DisplayName: "Demo", RuntimeTargetID: 7, ApplicationName: stringPointer("demo"), ComposeFileName: "compose.yaml", ComposeFileContent: "services: {}\n", ComposeFilePath: "compose.yaml", WorkspaceEntries: managedWorkspaceEntries("services: {}\n"),
	}, nil)
	if !errors.Is(err, errProjectComposeNameOccupied) || !errors.Is(err, errProjectConflict) {
		t.Fatalf("expected compose name conflict, got %v", err)
	}
	if repository.importInput != nil {
		t.Fatalf("unexpected registry import: %#v", repository.importInput)
	}
	if _, statErr := os.Stat(filepath.Join(managedRoot, "demo")); !os.IsNotExist(statErr) {
		t.Fatalf("workspace should be cleaned after collision, stat error = %v", statErr)
	}
}

func TestCreateManagedProjectAllowsUnavailableRuntimeTarget(t *testing.T) {
	managedRoot := t.TempDir()
	repository := &stubProjectRepository{}
	service, err := NewService(repository,
		WithSystemConfigResolver(stubSystemConfigResolver{value: managedRoot}),
		WithRuntimeTargetReader(composeTargetReader(moduleapi.ComposeProjectNameStateUnavailable)),
	)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	_, err = service.CreateManagedProject(context.Background(), ManagedProjectCreateRequest{
		DisplayName: "Demo", RuntimeTargetID: 7, ApplicationName: stringPointer("demo"), ComposeFileName: "compose.yaml", ComposeFileContent: "services: {}\n", ComposeFilePath: "compose.yaml", WorkspaceEntries: managedWorkspaceEntries("services: {}\n"),
	}, nil)
	if err != nil {
		t.Fatalf("create managed project: %v", err)
	}
	if repository.importInput == nil {
		t.Fatal("expected registry import for unavailable target")
	}
}

func TestEnsureLifecycleRuntimeTargetAvailableAllowsRegisteredComposeName(t *testing.T) {
	service, err := NewService(&stubProjectRepository{}, WithRuntimeTargetReader(composeTargetReader(moduleapi.ComposeProjectNameStateOccupied)))
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	targetID := uint64(7)
	err = service.ensureLifecycleRuntimeTargetAvailable(context.Background(), projectstore.ProjectAggregate{Project: projectstore.Project{RuntimeTargetID: &targetID, ComposeProjectName: "demo"}})
	if err != nil {
		t.Fatalf("expected registered project lifecycle to ignore its occupied Compose name, got %v", err)
	}
}

func TestEnsureLifecycleRuntimeTargetAvailableRejectsUnavailableTarget(t *testing.T) {
	service, err := NewService(&stubProjectRepository{}, WithRuntimeTargetReader(composeTargetReader(moduleapi.ComposeProjectNameStateUnavailable)))
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	targetID := uint64(7)
	err = service.ensureLifecycleRuntimeTargetAvailable(context.Background(), projectstore.ProjectAggregate{Project: projectstore.Project{RuntimeTargetID: &targetID}})
	if !errors.Is(err, errProjectRuntimeUnavailable) {
		t.Fatalf("expected unavailable runtime target to block lifecycle, got %v", err)
	}
}
