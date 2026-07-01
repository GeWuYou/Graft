package project

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	generated "graft/server/internal/contract/openapi/generated"
	"graft/server/internal/moduleapi"
	projectstore "graft/server/modules/project/store"
)

type stubProjectRepository struct {
	aggregate        projectstore.ProjectAggregate
	unregisterCalled bool
	importInput      *projectstore.ImportProjectInput
}

func (s *stubProjectRepository) List(context.Context, projectstore.ListQuery) (projectstore.ListResult, error) {
	return projectstore.ListResult{Items: []projectstore.ProjectAggregate{s.aggregate}, Total: 1}, nil
}

func (s *stubProjectRepository) Get(context.Context, uint64) (projectstore.ProjectAggregate, error) {
	if s.aggregate.Project.ID == 0 {
		return projectstore.ProjectAggregate{}, projectstore.ErrProjectNotFound
	}
	return s.aggregate, nil
}

func (s *stubProjectRepository) GetFile(context.Context, uint64, uint64) (projectstore.ProjectFile, error) {
	return projectstore.ProjectFile{}, projectstore.ErrFileNotFound
}

func (s *stubProjectRepository) ImportProject(_ context.Context, input projectstore.ImportProjectInput) (projectstore.ProjectAggregate, error) {
	s.importInput = &input
	if s.aggregate.Project.ID == 0 {
		s.aggregate.Project.ID = 99
	}
	s.aggregate.Project.DisplayName = input.DisplayName
	s.aggregate.Project.CanonicalProjectName = input.CanonicalProjectName
	s.aggregate.Project.CanonicalProjectNameSource = input.CanonicalProjectNameSource
	s.aggregate.Project.SourceKind = input.SourceKind
	s.aggregate.Project.HostScope = input.HostScope
	s.aggregate.Project.WorkingDirectory = input.WorkingDirectory
	s.aggregate.Project.OwnershipMode = input.OwnershipMode
	s.aggregate.Project.LastRefreshStatus = input.LastRefreshStatus
	s.aggregate.Project.LastRefreshAt = input.LastRefreshAt
	s.aggregate.Project.LastRefreshConfigHash = input.LastRefreshConfigHash
	s.aggregate.Project.LastObservedConfigHash = input.LastObservedConfigHash
	s.aggregate.Project.LastDriftCheckedAt = input.LastDriftCheckedAt
	s.aggregate.Project.DriftStatus = input.DriftStatus
	s.aggregate.Files = input.Files
	s.aggregate.Snapshot = input.Snapshot
	return s.aggregate, nil
}

func (s *stubProjectRepository) RefreshProject(context.Context, projectstore.RefreshProjectInput) (projectstore.ProjectAggregate, error) {
	return projectstore.ProjectAggregate{}, errors.New("not implemented")
}

func (s *stubProjectRepository) UnregisterProject(context.Context, projectstore.UnregisterProjectInput) error {
	s.unregisterCalled = true
	return nil
}

type stubRuntimeReader struct {
	summary moduleapi.ContainerProjectRuntimeSummary
}

type stubSystemConfigResolver struct {
	value string
	err   error
}

func (s stubSystemConfigResolver) IsBooleanConfigEnabled(context.Context, string, bool) bool { return false }

func (s stubSystemConfigResolver) ResolveDefaultConfig(context.Context, string) (string, error) {
	if s.err != nil {
		return "", s.err
	}
	payload, err := json.Marshal(s.value)
	if err != nil {
		return "", err
	}
	return string(payload), nil
}

func (s stubRuntimeReader) ListProjectMembers(context.Context, string, string) (moduleapi.ContainerProjectRuntimeSummary, error) {
	return s.summary, nil
}

func TestServicesMergesRuntimeMembers(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	composePath := filepath.Join(tempDir, "compose.yaml")
	content := []byte("services:\n  web:\n    image: nginx:latest\n  worker:\n    image: busybox\n")
	if err := os.WriteFile(composePath, content, 0o600); err != nil {
		t.Fatalf("write compose file: %v", err)
	}
	now := time.Now().UTC()
	repo := &stubProjectRepository{
		aggregate: projectstore.ProjectAggregate{
			Project: projectstore.Project{
				ID:                   1,
				CanonicalProjectName: "demo",
				HostScope:            "local",
				WorkingDirectory:     tempDir,
				OwnershipMode:        "external",
				LastRefreshStatus:    "success",
				LastRefreshAt:        &now,
				DriftStatus:          "clean",
			},
			Files: []projectstore.ProjectFile{
				{
					ID:           1,
					ProjectID:    1,
					Kind:         "compose",
					Role:         "primary",
					AbsolutePath: composePath,
					DisplayPath:  composePath,
					OrderIndex:   0,
				},
			},
			Snapshot: &projectstore.Snapshot{
				ProjectID:            1,
				ConfigHash:           "hash",
				DeclaredServiceCount: 2,
				RefreshedAt:          now,
			},
		},
	}
	service, err := NewService(repo, WithRuntimeReader(stubRuntimeReader{
		summary: moduleapi.ContainerProjectRuntimeSummary{
			CanonicalProjectName: "demo",
			RunningCount:         1,
			StoppedCount:         1,
			Members: []moduleapi.ContainerProjectMember{
				{ContainerID: "c1", ContainerName: "demo-web-1", ServiceName: "web", CanonicalState: "running"},
				{ContainerID: "c2", ContainerName: "demo-worker-1", ServiceName: "worker", CanonicalState: "exited"},
			},
		},
	}))
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	result, err := service.Services(context.Background(), 1)
	if err != nil {
		t.Fatalf("services: %v", err)
	}
	if len(result.Items) != 2 {
		t.Fatalf("expected 2 services, got %d", len(result.Items))
	}
	if result.Items[0].RunningCount+result.Items[1].RunningCount != 1 {
		t.Fatalf("expected one running member, got %#v", result.Items)
	}
	if result.Items[0].StoppedCount+result.Items[1].StoppedCount != 1 {
		t.Fatalf("expected one stopped member, got %#v", result.Items)
	}
}

func TestDestroyBlocksExternalWorkingDirectoryDeletion(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	repo := &stubProjectRepository{
		aggregate: projectstore.ProjectAggregate{
			Project: projectstore.Project{
				ID:                   1,
				CanonicalProjectName: "demo",
				HostScope:            "local",
				WorkingDirectory:     tempDir,
				OwnershipMode:        "external",
				LastRefreshStatus:    "success",
			},
		},
	}
	service, err := NewService(repo)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	result, err := service.Destroy(context.Background(), 1, DestroyRequest{
		DeleteWorkingDirectory:      true,
		ConfirmCanonicalProjectName: "demo",
	})
	if !errors.Is(err, errProjectDestroyBlocked) {
		t.Fatalf("expected destroy blocked, got %v", err)
	}
	if result.Result != generated.ProjectActionResultBlocked {
		t.Fatalf("expected blocked result, got %s", result.Result)
	}
	if repo.unregisterCalled {
		t.Fatalf("unregister should not be called when destroy is blocked")
	}
}

func TestCreateManagedProjectWritesFilesAndPersistsRegistry(t *testing.T) {
	t.Parallel()

	managedRoot := t.TempDir()
	repo := &stubProjectRepository{}
	service, err := NewService(repo, WithSystemConfigResolver(stubSystemConfigResolver{value: managedRoot}))
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	envName := ".env"
	result, err := service.CreateManagedProject(context.Background(), ManagedProjectCreateRequest{
		DisplayName:              "Demo",
		CanonicalProjectName:     "demo",
		RelativeProjectDirectory: "demo",
		ComposeFileName:          "compose.yaml",
		ComposeFileContent:       "services:\n  web:\n    image: nginx:latest\n",
		EnvFileName:              &envName,
		EnvFileContent:           stringPointer("FOO=bar\n"),
	}, nil)
	if err != nil {
		t.Fatalf("create managed project: %v", err)
	}

	composePath := filepath.Join(managedRoot, "demo", "compose.yaml")
	if _, err := os.Stat(composePath); err != nil {
		t.Fatalf("expected compose file written: %v", err)
	}
	envPath := filepath.Join(managedRoot, "demo", ".env")
	if _, err := os.Stat(envPath); err != nil {
		t.Fatalf("expected env file written: %v", err)
	}
	if repo.importInput == nil {
		t.Fatalf("expected repository import input to be recorded")
	}
	if repo.importInput.SourceKind != "managed" {
		t.Fatalf("expected managed source kind, got %q", repo.importInput.SourceKind)
	}
	if result.ProjectID == 0 {
		t.Fatalf("expected created project id")
	}
	if result.DeclaredServiceCount != 1 {
		t.Fatalf("expected one declared service, got %d", result.DeclaredServiceCount)
	}
}

func TestCreateManagedProjectRejectsManagedRootBaseDirectory(t *testing.T) {
	t.Parallel()

	managedRoot := t.TempDir()
	repo := &stubProjectRepository{}
	service, err := NewService(repo, WithSystemConfigResolver(stubSystemConfigResolver{value: managedRoot}))
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	_, err = service.CreateManagedProject(context.Background(), ManagedProjectCreateRequest{
		DisplayName:              "Demo",
		CanonicalProjectName:     "demo",
		RelativeProjectDirectory: ".",
		ComposeFileName:          "compose.yaml",
		ComposeFileContent:       "services:\n  web:\n    image: nginx:latest\n",
	}, nil)
	if !errors.Is(err, errProjectInvalidArgument) {
		t.Fatalf("expected invalid argument, got %v", err)
	}
}

func TestDiscoveryCandidatesScansManagedRootWithoutRegistering(t *testing.T) {
	t.Parallel()

	managedRoot := t.TempDir()
	projectDir := filepath.Join(managedRoot, "orders")
	if err := os.MkdirAll(projectDir, 0o750); err != nil {
		t.Fatalf("mkdir project dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectDir, "compose.yaml"), []byte("services:\n  api:\n    image: nginx:latest\n"), 0o600); err != nil {
		t.Fatalf("write compose file: %v", err)
	}

	repo := &stubProjectRepository{}
	service, err := NewService(repo, WithSystemConfigResolver(stubSystemConfigResolver{value: managedRoot}))
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	result, err := service.DiscoveryCandidates(context.Background())
	if err != nil {
		t.Fatalf("discovery candidates: %v", err)
	}
	if !result.SupportsScan || !result.SupportsAutoDiscovery {
		t.Fatalf("expected discovery support, got %#v", result)
	}
	if len(result.Items) != 1 {
		t.Fatalf("expected 1 candidate, got %d", len(result.Items))
	}
	item := result.Items[0]
	if item.CandidateKind != "directory-scan" {
		t.Fatalf("expected directory-scan candidate, got %q", item.CandidateKind)
	}
	if item.Status != "ready" {
		t.Fatalf("expected ready candidate, got %q", item.Status)
	}
	if item.RecommendedAction != "import" {
		t.Fatalf("expected import action, got %q", item.RecommendedAction)
	}
	if item.SourceMetadata["managed_relative_directory"] != "orders" {
		t.Fatalf("expected managed relative directory metadata, got %#v", item.SourceMetadata)
	}
}

func TestDiscoveryCandidatesMarksConflictWhenProjectAlreadyRegistered(t *testing.T) {
	t.Parallel()

	managedRoot := t.TempDir()
	projectDir := filepath.Join(managedRoot, "orders")
	if err := os.MkdirAll(projectDir, 0o750); err != nil {
		t.Fatalf("mkdir project dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectDir, "compose.yaml"), []byte("services:\n  api:\n    image: nginx:latest\n"), 0o600); err != nil {
		t.Fatalf("write compose file: %v", err)
	}

	repo := &stubProjectRepository{
		aggregate: projectstore.ProjectAggregate{
			Project: projectstore.Project{
				ID:                   1,
				DisplayName:          "Orders",
				CanonicalProjectName: "orders",
				WorkingDirectory:     projectDir,
			},
		},
	}
	service, err := NewService(repo, WithSystemConfigResolver(stubSystemConfigResolver{value: managedRoot}))
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	result, err := service.DiscoveryCandidates(context.Background())
	if err != nil {
		t.Fatalf("discovery candidates: %v", err)
	}
	if len(result.Items) != 1 {
		t.Fatalf("expected 1 candidate, got %d", len(result.Items))
	}
	item := result.Items[0]
	if item.Status != "conflict" {
		t.Fatalf("expected conflict status, got %q", item.Status)
	}
	if item.RecommendedAction != "review" {
		t.Fatalf("expected review action, got %q", item.RecommendedAction)
	}
	if len(item.Conflicts) == 0 {
		t.Fatalf("expected conflict details")
	}
}

func TestSourceCatalogAddsRemoteHostBoundary(t *testing.T) {
	t.Parallel()

	managedRoot := t.TempDir()
	service, err := NewService(&stubProjectRepository{}, WithSystemConfigResolver(stubSystemConfigResolver{value: managedRoot}))
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	result, err := service.SourceCatalog(context.Background())
	if err != nil {
		t.Fatalf("source catalog: %v", err)
	}
	if len(result.Items) != 4 {
		t.Fatalf("expected 4 source entries, got %d", len(result.Items))
	}
	remote := result.Items[3]
	if remote.Type != generated.ProjectSourceEntryType("remote-host") {
		t.Fatalf("expected remote-host source type, got %q", remote.Type)
	}
	if remote.HostScope != generated.ProjectHostScope("remote") {
		t.Fatalf("expected remote host scope, got %q", remote.HostScope)
	}
	if remote.RoutePath != "/ops/projects/create/remote-host" {
		t.Fatalf("unexpected remote-host route path: %q", remote.RoutePath)
	}
	if remote.Status != generated.ProjectSourceEntryStatus("planned") {
		t.Fatalf("expected planned remote-host status, got %q", remote.Status)
	}
}

func TestProjectListItemUsesFrontendActivityAuthorityForLocalProjects(t *testing.T) {
	t.Parallel()

	item := toProjectListItem(projectstore.ProjectAggregate{
		Project: projectstore.Project{
			ID:                         1,
			DisplayName:                "Orders",
			CanonicalProjectName:       "orders",
			CanonicalProjectNameSource: "computed",
			SourceKind:                 "managed",
			HostScope:                  "local",
			OwnershipMode:              "managed-root-dedicated",
			WorkingDirectory:           "/tmp/orders",
			LastRefreshStatus:          "success",
			DriftStatus:                "clean",
		},
	})
	if item.ActivityAuthority != generated.ProjectActivityAuthority("frontend-fanout") {
		t.Fatalf("expected frontend-fanout activity authority, got %q", item.ActivityAuthority)
	}
}

func TestProjectDetailUsesBackendPlannedActivityAuthorityForRemoteScope(t *testing.T) {
	t.Parallel()

	detail := toProjectDetailResponse(projectstore.ProjectAggregate{
		Project: projectstore.Project{
			ID:                         2,
			DisplayName:                "Remote Orders",
			CanonicalProjectName:       "orders-remote",
			CanonicalProjectNameSource: "computed",
			SourceKind:                 "remote-host",
			HostScope:                  "remote",
			OwnershipMode:              "external",
			WorkingDirectory:           "/remote/orders",
			LastRefreshStatus:          "never",
			DriftStatus:                "unknown",
		},
	})
	if detail.ActivityAuthority != generated.ProjectActivityAuthority("backend-planned") {
		t.Fatalf("expected backend-planned activity authority, got %q", detail.ActivityAuthority)
	}
	if detail.SourceMetadata == nil || detail.SourceMetadata.ActivityRollupScope == nil {
		t.Fatalf("expected remote-host source metadata activity rollup scope")
	}
}
