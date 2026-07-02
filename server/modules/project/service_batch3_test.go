package project

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	generated "graft/server/internal/contract/openapi/generated"
	"graft/server/internal/moduleapi"
	projectcontract "graft/server/modules/project/contract"
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
	files := append([]projectstore.ProjectFile(nil), input.Files...)
	for index := range files {
		if files[index].ID == 0 {
			files[index].ID = uint64(index + 1)
		}
		files[index].ProjectID = s.aggregate.Project.ID
	}
	s.aggregate.Files = files
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
	summary          moduleapi.ContainerProjectRuntimeSummary
	candidates       []moduleapi.ContainerProjectRuntimeCandidate
	candidateMembers []moduleapi.ContainerProjectMember
}

type stubSystemConfigResolver struct {
	value string
	err   error
}

func (s stubSystemConfigResolver) IsBooleanConfigEnabled(context.Context, string, bool) bool {
	return false
}

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

func (s stubRuntimeReader) ListImportCandidates(context.Context, string) ([]moduleapi.ContainerProjectRuntimeCandidate, error) {
	return append([]moduleapi.ContainerProjectRuntimeCandidate(nil), s.candidates...), nil
}

func (s stubRuntimeReader) ListImportCandidateMembers(
	context.Context,
	string,
	moduleapi.ContainerProjectRuntimeCandidate,
) ([]moduleapi.ContainerProjectMember, error) {
	return append([]moduleapi.ContainerProjectMember(nil), s.candidateMembers...), nil
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

func TestListRuntimeImportCandidatesMarksBrokenCompose(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	composePath := filepath.Join(tempDir, "compose.yaml")
	if err := os.WriteFile(composePath, []byte("services:\n  web:\n    image: [\n"), 0o600); err != nil {
		t.Fatalf("write compose file: %v", err)
	}

	service, err := NewService(&stubProjectRepository{}, WithRuntimeReader(stubRuntimeReader{
		candidates: []moduleapi.ContainerProjectRuntimeCandidate{
			{
				CandidateKey:           "runtime_demo",
				CanonicalProjectName:   "demo",
				Status:                 importRuntimeCandidateStatusReady,
				Importable:             true,
				RuntimeType:            "docker",
				WorkingDirectory:       tempDir,
				WorkingDirectorySource: "runtime_label",
				ConfigFiles:            []string{composePath},
				ServiceNames:           []string{"web"},
				ContainerCounts:        moduleapi.ContainerProjectRuntimeContainerCounts{Running: 1, Total: 1},
			},
		},
	}))
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	result, err := service.ListRuntimeImportCandidates(context.Background(), RuntimeImportCandidateListQuery{})
	if err != nil {
		t.Fatalf("list runtime import candidates: %v", err)
	}
	if len(result.Items) != 1 {
		t.Fatalf("expected 1 candidate, got %d", len(result.Items))
	}
	candidate := result.Items[0]
	if candidate.Status != importRuntimeCandidateStatusBrokenCompose || candidate.Importable {
		t.Fatalf("expected broken compose candidate, got %#v", candidate)
	}
	if len(candidate.StatusReasonCodes) != 1 || candidate.StatusReasonCodes[0] != importRuntimeReasonComposeParseFailed {
		t.Fatalf("unexpected status reason codes %#v", candidate.StatusReasonCodes)
	}
}

func TestListRuntimeImportCandidatesDedupesCandidateKeys(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	composePath := filepath.Join(tempDir, "compose.yaml")
	if err := os.WriteFile(composePath, []byte("services:\n  web:\n    image: nginx:latest\n"), 0o600); err != nil {
		t.Fatalf("write compose file: %v", err)
	}

	service, err := NewService(&stubProjectRepository{}, WithRuntimeReader(stubRuntimeReader{
		candidates: []moduleapi.ContainerProjectRuntimeCandidate{
			{
				CandidateKey:           "runtime_demo",
				CanonicalProjectName:   "demo",
				Status:                 importRuntimeCandidateStatusBrokenCompose,
				StatusReasonCodes:      []string{importRuntimeReasonComposeParseFailed},
				Importable:             false,
				RuntimeType:            "docker",
				WorkingDirectory:       tempDir,
				WorkingDirectorySource: "runtime_label",
				ConfigFiles:            []string{composePath},
				ServiceNames:           []string{"web"},
				ContainerCounts:        moduleapi.ContainerProjectRuntimeContainerCounts{Running: 1, Total: 1},
			},
			{
				CandidateKey:           "runtime_demo",
				CanonicalProjectName:   "demo",
				Status:                 importRuntimeCandidateStatusBrokenCompose,
				StatusReasonCodes:      []string{importRuntimeReasonComposeParseFailed},
				Importable:             false,
				RuntimeType:            "docker",
				WorkingDirectory:       tempDir,
				WorkingDirectorySource: "runtime_label",
				ConfigFiles:            []string{composePath},
				ServiceNames:           []string{"web"},
				ContainerCounts:        moduleapi.ContainerProjectRuntimeContainerCounts{Running: 1, Total: 1},
			},
		},
	}))
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	result, err := service.ListRuntimeImportCandidates(context.Background(), RuntimeImportCandidateListQuery{})
	if err != nil {
		t.Fatalf("list runtime import candidates: %v", err)
	}
	if len(result.Items) != 1 {
		t.Fatalf("expected duplicate candidate keys to collapse to 1 item, got %#v", result.Items)
	}
	if result.Total != 1 {
		t.Fatalf("expected deduped total 1, got %d", result.Total)
	}
	if result.FilterCounts.All != 1 || result.FilterCounts.Unavailable != 1 {
		t.Fatalf("expected deduped filter counts, got %#v", result.FilterCounts)
	}
}

func assertRuntimeInspectResult(
	t *testing.T,
	result RuntimeImportInspectResult,
	expectedWorkingDirectory string,
	expectedComposePath string,
	expectedEnvPath string,
) {
	t.Helper()
	if result.CandidateKey != "runtime_demo" {
		t.Fatalf("expected candidate key to round-trip, got %#v", result)
	}
	if result.ResolvedWorkingDirectory != expectedWorkingDirectory {
		t.Fatalf("expected working directory %q, got %q", expectedWorkingDirectory, result.ResolvedWorkingDirectory)
	}
	if len(result.ComposeFiles) != 1 || result.ComposeFiles[0].AbsolutePath != expectedComposePath {
		t.Fatalf("unexpected compose files %#v", result.ComposeFiles)
	}
	if len(result.EnvFiles) != 1 || result.EnvFiles[0].AbsolutePath != expectedEnvPath {
		t.Fatalf("unexpected env files %#v", result.EnvFiles)
	}
	if result.ValidationStatus != "ready" {
		t.Fatalf("expected ready validation status, got %q", result.ValidationStatus)
	}
	assertRuntimeInspectMembers(t, result.RuntimeMembers)
	assertRuntimeInspectNetworks(t, result.NetworkResources)
	assertRuntimeInspectVolumes(t, result.VolumeResources)
	if !slices.Contains(result.Warnings, "working_directory_derived_from_config_files") {
		t.Fatalf("expected candidate warning in inspect result, got %#v", result.Warnings)
	}
}

func assertRuntimeInspectMembers(t *testing.T, members []RuntimeImportMember) {
	t.Helper()
	if len(members) != 2 {
		t.Fatalf("expected two runtime members, got %#v", members)
	}
	if members[0].ServiceName != "web" || members[0].ContainerID != "c1" {
		t.Fatalf("unexpected runtime members %#v", members)
	}
	if members[1].ServiceName != "worker" || members[1].ContainerID != "c2" {
		t.Fatalf("unexpected runtime members %#v", members)
	}
}

func assertRuntimeInspectNetworks(t *testing.T, resources []RuntimeImportNetworkResource) {
	t.Helper()
	if len(resources) != 2 {
		t.Fatalf("expected two network resources, got %#v", resources)
	}
	assertBackendNetworkResource(t, resources[0])
	assertFrontendNetworkResource(t, resources[1])
}

func assertBackendNetworkResource(t *testing.T, resource RuntimeImportNetworkResource) {
	t.Helper()
	if resource.Name != "backend" || resource.ServiceCount != 2 {
		t.Fatalf("unexpected backend network resource %#v", resource)
	}
	if resource.Driver == nil || *resource.Driver != "overlay" {
		t.Fatalf("expected backend network driver overlay, got %#v", resource.Driver)
	}
	if resource.Internal != nil {
		t.Fatalf("expected backend network internal to remain unknown, got %#v", resource.Internal)
	}
	if !slices.Equal(resource.Containers, []string{"demo-web-1", "demo-worker-1"}) {
		t.Fatalf("unexpected backend network containers %#v", resource.Containers)
	}
	if !slices.Equal(resource.Services, []string{"web", "worker"}) {
		t.Fatalf("unexpected backend network services %#v", resource.Services)
	}
}

func assertFrontendNetworkResource(t *testing.T, resource RuntimeImportNetworkResource) {
	t.Helper()
	if resource.Name != "frontend" || resource.ServiceCount != 1 {
		t.Fatalf("unexpected frontend network resource %#v", resource)
	}
	if resource.Driver == nil || *resource.Driver != "bridge" {
		t.Fatalf("expected frontend network driver bridge, got %#v", resource.Driver)
	}
	if resource.Internal == nil || !*resource.Internal {
		t.Fatalf("expected frontend network internal=true, got %#v", resource.Internal)
	}
}

func assertRuntimeInspectVolumes(t *testing.T, resources []RuntimeImportVolumeResource) {
	t.Helper()
	if len(resources) != 2 {
		t.Fatalf("expected two volume resources, got %#v", resources)
	}
	assertAnonymousVolumeResource(t, resources[0])
	assertNamedVolumeResource(t, resources[1])
}

func assertAnonymousVolumeResource(t *testing.T, resource RuntimeImportVolumeResource) {
	t.Helper()
	if resource.Name != "/tmp/cache" || !resource.Anonymous {
		t.Fatalf("unexpected anonymous volume resource %#v", resource)
	}
	if !slices.Equal(resource.MountedBy, []string{"web", "worker"}) {
		t.Fatalf("unexpected anonymous volume mounted_by %#v", resource.MountedBy)
	}
}

func assertNamedVolumeResource(t *testing.T, resource RuntimeImportVolumeResource) {
	t.Helper()
	if resource.Name != "data" || resource.Anonymous {
		t.Fatalf("unexpected named volume resource %#v", resource)
	}
	if resource.Driver == nil || *resource.Driver != "local" {
		t.Fatalf("expected named volume driver local, got %#v", resource.Driver)
	}
	if resource.MountTarget != "/data" {
		t.Fatalf("unexpected named volume mount target %#v", resource)
	}
	if !slices.Equal(resource.Containers, []string{"demo-web-1", "demo-worker-1"}) {
		t.Fatalf("unexpected named volume containers %#v", resource.Containers)
	}
}

func TestInspectRuntimeCandidateReusesInspectPipeline(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	composePath := filepath.Join(tempDir, "compose.yaml")
	envPath := filepath.Join(tempDir, ".env")
	composeContent := "" +
		"services:\n" +
		"  web:\n" +
		"    image: nginx:latest\n" +
		"    networks:\n" +
		"      - frontend\n" +
		"      - backend\n" +
		"    volumes:\n" +
		"      - data:/data\n" +
		"      - /tmp/cache\n" +
		"  worker:\n" +
		"    image: busybox:latest\n" +
		"    networks:\n" +
		"      - backend\n" +
		"    volumes:\n" +
		"      - data:/data\n" +
		"      - type=volume,target=/tmp/cache\n" +
		"networks:\n" +
		"  frontend:\n" +
		"    driver: bridge\n" +
		"    internal: true\n" +
		"  backend:\n" +
		"    driver: overlay\n" +
		"volumes:\n" +
		"  data:\n" +
		"    driver: local\n"
	if err := os.WriteFile(composePath, []byte(composeContent), 0o600); err != nil {
		t.Fatalf("write compose file: %v", err)
	}
	if err := os.WriteFile(envPath, []byte("FOO=bar\n"), 0o600); err != nil {
		t.Fatalf("write env file: %v", err)
	}

	repo := &stubProjectRepository{
		aggregate: projectstore.ProjectAggregate{
			Project: projectstore.Project{ID: 1, CanonicalProjectName: "existing", WorkingDirectory: "/srv/existing"},
		},
	}
	service, err := NewService(repo, WithRuntimeReader(stubRuntimeReader{
		candidates: []moduleapi.ContainerProjectRuntimeCandidate{
			{
				CandidateKey:           "runtime_demo",
				CanonicalProjectName:   "demo",
				Status:                 importRuntimeCandidateStatusReady,
				Importable:             true,
				RuntimeType:            "docker",
				RuntimeVersion:         "27.0.1",
				WorkingDirectory:       tempDir,
				WorkingDirectorySource: "runtime_label",
				ConfigFiles:            []string{composePath},
				ServiceNames:           []string{"web", "worker"},
				ContainerCounts:        moduleapi.ContainerProjectRuntimeContainerCounts{Running: 2, Total: 2},
				Warnings:               []string{"working_directory_derived_from_config_files"},
			},
		},
		candidateMembers: []moduleapi.ContainerProjectMember{
			{ContainerID: "c1", ContainerName: "demo-web-1", ServiceName: "web", CanonicalState: "running"},
			{ContainerID: "c2", ContainerName: "demo-worker-1", ServiceName: "worker", CanonicalState: "running"},
		},
	}))
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	result, err := service.InspectRuntimeCandidate(context.Background(), RuntimeImportInspectRequest{
		CandidateKey: "runtime_demo",
	})
	if err != nil {
		t.Fatalf("inspect runtime candidate: %v", err)
	}
	assertRuntimeInspectResult(t, result, tempDir, composePath, envPath)
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

	item := toProjectListItemWithManagedRoot(projectstore.ProjectAggregate{
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
	}, "")
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

func TestImportDirectorySourcesIncludeManagedRootAndAllowlistedRoot(t *testing.T) {
	t.Parallel()

	managedRoot := t.TempDir()
	service, err := NewService(&stubProjectRepository{}, WithSystemConfigResolver(stubCompositeConfigResolver{
		values: map[string]string{
			"ops.project.managed.root_directory": `"` + managedRoot + `"`,
			"ops.project.import.allowed_roots":   `[{"id":"srv","label":"Srv","path":"/srv"}]`,
		},
	}))
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	result, err := service.ImportDirectorySources(context.Background())
	if err != nil {
		t.Fatalf("import directory sources: %v", err)
	}
	if len(result.Items) != 2 {
		t.Fatalf("expected 2 roots, got %d", len(result.Items))
	}
	if result.Items[0].RootID != importManagedRootSourceID || !result.Items[0].Managed {
		t.Fatalf("expected managed root first, got %#v", result.Items[0])
	}
}

func TestImportDirectorySourcesDecodeSystemConfigStringValue(t *testing.T) {
	t.Parallel()

	managedRoot := t.TempDir()
	service, err := NewService(&stubProjectRepository{}, WithSystemConfigResolver(stubSystemConfigResolver{
		value: `[{"id":"srv","label":"Srv","path":"` + managedRoot + `"}]`,
	}))
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	result, err := service.ImportDirectorySources(context.Background())
	if err != nil {
		t.Fatalf("import directory sources: %v", err)
	}
	if len(result.Items) != 1 {
		t.Fatalf("expected 1 decoded root, got %d", len(result.Items))
	}
	if result.Items[0].RootID != "srv" || result.Items[0].Path != managedRoot {
		t.Fatalf("unexpected decoded root: %#v", result.Items[0])
	}
}

func TestImportDirectorySourcesFallbackToCurrentServiceDirectory(t *testing.T) {
	t.Parallel()

	service, err := NewService(&stubProjectRepository{})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	result, err := service.ImportDirectorySources(context.Background())
	if err != nil {
		t.Fatalf("import directory sources: %v", err)
	}
	if len(result.Items) != 0 {
		t.Fatalf("expected no fallback roots without explicit authority, got %#v", result.Items)
	}
}

func TestImportDirectorySourcesFallbackUsesConfiguredDefaultPath(t *testing.T) {
	customPath := filepath.Join(string(filepath.Separator), "workspace", "compose")
	t.Setenv("GRAFT_PROJECT_IMPORT_DEFAULT_PATH", customPath)

	service, err := NewService(&stubProjectRepository{})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	result, err := service.ImportDirectorySources(context.Background())
	if err != nil {
		t.Fatalf("import directory sources: %v", err)
	}
	if len(result.Items) != 1 {
		t.Fatalf("expected one explicit fallback root, got %d items", len(result.Items))
	}
	if result.Items[0].RootID != importServiceRootSourceID {
		t.Fatalf("expected service-root fallback id, got %#v", result.Items[0])
	}
	if result.Items[0].Path != customPath {
		t.Fatalf("expected configured fallback root path %q, got %q", customPath, result.Items[0].Path)
	}
	if result.Items[0].InitialPath != "" {
		t.Fatalf("expected root-scoped initial path for explicit fallback root, got %q", result.Items[0].InitialPath)
	}
}

func TestImportDirectorySourcesFallbackUsesContainerPathWhenDefaultPathMissing(t *testing.T) {
	t.Setenv("GRAFT_PROJECT_IMPORT_CONTAINER_PATH", filepath.Join(string(filepath.Separator), "srv", "graft-imports"))

	service, err := NewService(&stubProjectRepository{})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	result, err := service.ImportDirectorySources(context.Background())
	if err != nil {
		t.Fatalf("import directory sources: %v", err)
	}
	if len(result.Items) != 1 {
		t.Fatalf("expected one explicit fallback root, got %d items", len(result.Items))
	}
	if result.Items[0].Path != filepath.Join(string(filepath.Separator), "srv", "graft-imports") {
		t.Fatalf("unexpected container-path fallback root: %#v", result.Items[0])
	}
}

func TestToProjectDetailResponsePreservesNestedManagedRelativeDirectory(t *testing.T) {
	t.Parallel()

	managedRoot := filepath.Join(string(filepath.Separator), "srv", "managed")
	aggregate := projectstore.ProjectAggregate{
		Project: projectstore.Project{
			ID:                         7,
			DisplayName:                "Orders",
			CanonicalProjectName:       "orders",
			CanonicalProjectNameSource: "computed",
			SourceKind:                 projectcontract.SourceKindManaged.String(),
			HostScope:                  projectcontract.HostScopeLocal.String(),
			OwnershipMode:              projectcontract.OwnershipModeManagedRootDedicated.String(),
			WorkingDirectory:           filepath.Join(managedRoot, "team-a", "orders"),
			LastRefreshStatus:          "success",
			DriftStatus:                "clean",
		},
	}

	detail := toProjectDetailResponseWithManagedRoot(aggregate, managedRoot)
	if detail.SourceMetadata == nil || detail.SourceMetadata.ManagedRelativeDirectory == nil {
		t.Fatalf("expected managed source metadata with relative directory")
	}
	if *detail.SourceMetadata.ManagedRelativeDirectory != "team-a/orders" {
		t.Fatalf("expected nested managed relative directory, got %q", *detail.SourceMetadata.ManagedRelativeDirectory)
	}
}

func TestDiscoverImportFilesExcludesDirectoriesFromComposeCandidates(t *testing.T) {
	t.Parallel()

	workingDirectory := t.TempDir()
	if err := os.Mkdir(filepath.Join(workingDirectory, "compose.yaml"), 0o750); err != nil {
		t.Fatalf("mkdir fake compose dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(workingDirectory, "compose.yml"), []byte("services:\n  api:\n    image: nginx:latest\n"), 0o600); err != nil {
		t.Fatalf("write compose file: %v", err)
	}

	discovered, err := discoverImportFiles(workingDirectory)
	if err != nil {
		t.Fatalf("discover import files: %v", err)
	}
	if len(discovered.composeFiles) != 1 || discovered.composeFiles[0] != "compose.yml" {
		t.Fatalf("expected only the regular compose file candidate, got %#v", discovered.composeFiles)
	}
}

func TestDeleteManagedWorkingDirectoryRemovesOnlyTargetDirectory(t *testing.T) {
	t.Parallel()

	parent := t.TempDir()
	workingDirectory := filepath.Join(parent, "orders")
	sibling := filepath.Join(parent, "shared")
	if err := os.MkdirAll(filepath.Join(workingDirectory, "nested"), 0o750); err != nil {
		t.Fatalf("mkdir working tree: %v", err)
	}
	if err := os.WriteFile(filepath.Join(workingDirectory, "nested", "compose.yaml"), []byte("services:{}\n"), 0o600); err != nil {
		t.Fatalf("write working file: %v", err)
	}
	if err := os.MkdirAll(sibling, 0o750); err != nil {
		t.Fatalf("mkdir sibling: %v", err)
	}

	if err := deleteManagedWorkingDirectory(workingDirectory); err != nil {
		t.Fatalf("delete managed working directory: %v", err)
	}
	if _, err := os.Stat(workingDirectory); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected working directory removed, got %v", err)
	}
	if _, err := os.Stat(parent); err != nil {
		t.Fatalf("expected parent directory preserved: %v", err)
	}
	if _, err := os.Stat(sibling); err != nil {
		t.Fatalf("expected sibling directory preserved: %v", err)
	}
}

func TestCleanupManagedCreateRemovesCreatedDirectoryWithinParentBoundary(t *testing.T) {
	t.Parallel()

	parent := t.TempDir()
	createdDir := filepath.Join(parent, "orders")
	createdFile := filepath.Join(createdDir, "compose.yaml")
	if err := os.MkdirAll(createdDir, 0o750); err != nil {
		t.Fatalf("mkdir created dir: %v", err)
	}
	if err := os.WriteFile(createdFile, []byte("services:\n  api:\n    image: nginx:latest\n"), 0o600); err != nil {
		t.Fatalf("write created file: %v", err)
	}

	if err := cleanupManagedCreate(createdDir, []string{createdFile}); err != nil {
		t.Fatalf("cleanup managed create: %v", err)
	}
	if _, err := os.Stat(createdDir); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected created directory removed, got %v", err)
	}
	if _, err := os.Stat(parent); err != nil {
		t.Fatalf("expected parent directory preserved: %v", err)
	}
}

func TestProjectErrorMessageKeyUsesProjectCode(t *testing.T) {
	t.Parallel()

	if got := projectErrorMessageKey(projectcontract.ProjectConflict.String()); got != projectcontract.ProjectConflict.String() {
		t.Fatalf("expected project code as message key, got %q", got)
	}
	if got := projectErrorMessageKey(" "); got != "common.invalid_argument" {
		t.Fatalf("expected common invalid argument fallback, got %q", got)
	}
}

func TestBrowseImportDirectoriesStaysRootRelative(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "apps", "orders"), 0o750); err != nil {
		t.Fatalf("mkdir nested dir: %v", err)
	}
	service, err := NewService(&stubProjectRepository{}, WithSystemConfigResolver(stubCompositeConfigResolver{
		values: map[string]string{
			"ops.project.import.allowed_roots": `[{"id":"apps","label":"Apps","path":"` + root + `"}]`,
		},
	}))
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	result, err := service.BrowseImportDirectories(context.Background(), ImportDirectoryBrowseQuery{
		Provider: importProviderLocal,
		RootID:   "apps",
		Path:     "apps",
		Limit:    20,
	})
	if err != nil {
		t.Fatalf("browse import directories: %v", err)
	}
	if len(result.Items) != 1 || result.Items[0].Path != "apps/orders" {
		t.Fatalf("unexpected browse result: %#v", result.Items)
	}
}

func TestInspectAndImportByInspection(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	projectDir := filepath.Join(root, "orders")
	if err := os.MkdirAll(projectDir, 0o750); err != nil {
		t.Fatalf("mkdir project dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectDir, "compose.yaml"), []byte("services:\n  api:\n    image: nginx:latest\nnetworks:\n  default: {}\nvolumes:\n  data: {}\n"), 0o600); err != nil {
		t.Fatalf("write compose file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectDir, ".env"), []byte("FOO=bar\n"), 0o600); err != nil {
		t.Fatalf("write env file: %v", err)
	}
	repo := &stubProjectRepository{}
	service, err := NewService(repo, WithSystemConfigResolver(stubCompositeConfigResolver{
		values: map[string]string{
			"ops.project.import.allowed_roots": `[{"id":"apps","label":"Apps","path":"` + root + `"}]`,
		},
	}))
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	inspect, err := service.InspectImportDirectory(context.Background(), ImportInspectRequest{
		DirectoryRef: ImportDirectoryReference{Provider: importProviderLocal, RootID: "apps", Path: "orders"},
	})
	if err != nil {
		t.Fatalf("inspect import directory: %v", err)
	}
	if inspect.InspectionID == "" || len(inspect.NetworkNames) != 1 || len(inspect.VolumeNames) != 1 {
		t.Fatalf("unexpected inspect result: %#v", inspect)
	}

	imported, err := service.ImportByInspection(context.Background(), ImportExecuteRequest{InspectionID: inspect.InspectionID})
	if err != nil {
		t.Fatalf("import by inspection: %v", err)
	}
	if imported.Project.CanonicalProjectName != "orders" {
		t.Fatalf("unexpected imported project: %#v", imported.Project)
	}
	if repo.importInput == nil {
		t.Fatalf("expected persisted import input")
	}
}

type stubCompositeConfigResolver struct {
	values map[string]string
}

func (s stubCompositeConfigResolver) IsBooleanConfigEnabled(context.Context, string, bool) bool {
	return false
}

func (s stubCompositeConfigResolver) ResolveDefaultConfig(_ context.Context, key string) (string, error) {
	value, ok := s.values[key]
	if !ok {
		return `""`, nil
	}
	return value, nil
}
