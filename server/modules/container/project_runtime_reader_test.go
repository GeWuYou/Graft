package container

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"graft/server/internal/moduleapi"
	"graft/server/modules/container/terminal"
)

type stubProjectReaderRuntime struct {
	items []Summary
}

func (s stubProjectReaderRuntime) Info(context.Context) (RuntimeInfo, error) {
	return RuntimeInfo{Runtime: runtimeNameDocker, ServerVersion: "27.0.1"}, nil
}
func (s stubProjectReaderRuntime) List(context.Context, ListQuery) ([]Summary, error) {
	return s.items, nil
}
func (s stubProjectReaderRuntime) Detail(context.Context, Ref) (Detail, error)  { return Detail{}, nil }
func (s stubProjectReaderRuntime) Mounts(context.Context, Ref) ([]Mount, error) { return nil, nil }
func (s stubProjectReaderRuntime) MountUsage(context.Context, Ref, string) (MountUsage, error) {
	return MountUsage{}, nil
}
func (s stubProjectReaderRuntime) Logs(context.Context, Ref, LogQuery) (Logs, error) {
	return Logs{}, nil
}
func (s stubProjectReaderRuntime) StreamLogs(context.Context, Ref, LogQuery, func(LogChunk) error) error {
	return nil
}
func (s stubProjectReaderRuntime) Shell(context.Context, Ref, string) (terminal.Session, error) {
	return nil, nil
}
func (s stubProjectReaderRuntime) Start(context.Context, Ref) (ActionResult, error) {
	return ActionResult{}, nil
}
func (s stubProjectReaderRuntime) Stop(context.Context, Ref) (ActionResult, error) {
	return ActionResult{}, nil
}
func (s stubProjectReaderRuntime) Restart(context.Context, Ref) (ActionResult, error) {
	return ActionResult{}, nil
}
func (s stubProjectReaderRuntime) Remove(context.Context, Ref, RemoveOptions) (ActionResult, error) {
	return ActionResult{}, nil
}
func (s stubProjectReaderRuntime) Close() error { return nil }

func TestContainerProjectRuntimeReaderMapsComposeMembers(t *testing.T) {
	t.Parallel()

	reader := containerProjectRuntimeReader{
		service: &service{
			runtime: stubProjectReaderRuntime{
				items: []Summary{
					{ID: "1", Name: "demo-web-1", State: "running", ComposeProject: "demo", ComposeService: "web"},
					{ID: "2", Name: "demo-worker-1", State: "exited", ComposeProject: "demo", ComposeService: "worker"},
					{ID: "3", Name: "other-api-1", State: "running", ComposeProject: "other", ComposeService: "api"},
				},
			},
			enabled: true,
		},
	}

	summary, err := reader.ListProjectMembers(context.Background(), "local", "demo")
	if err != nil {
		t.Fatalf("list project members: %v", err)
	}
	if summary.RunningCount != 1 || summary.StoppedCount != 1 {
		t.Fatalf("unexpected counts: %#v", summary)
	}
	if len(summary.Members) != 2 {
		t.Fatalf("expected 2 members, got %d", len(summary.Members))
	}
	if summary.Members[0].ServiceName != "web" && summary.Members[1].ServiceName != "web" {
		t.Fatalf("expected web member in %#v", summary.Members)
	}
}

type pagingProjectReaderRuntime struct {
	pages [][]Summary
	seen  []ListQuery
}

func (s *pagingProjectReaderRuntime) Info(context.Context) (RuntimeInfo, error) {
	return RuntimeInfo{Runtime: runtimeNameDocker, ServerVersion: "27.0.1"}, nil
}
func (s *pagingProjectReaderRuntime) List(_ context.Context, query ListQuery) ([]Summary, error) {
	s.seen = append(s.seen, query)
	index := query.Offset / maxContainerListLimit
	if index < 0 || index >= len(s.pages) {
		return []Summary{}, nil
	}
	return s.pages[index], nil
}
func (s *pagingProjectReaderRuntime) Detail(context.Context, Ref) (Detail, error) {
	return Detail{}, nil
}
func (s *pagingProjectReaderRuntime) Mounts(context.Context, Ref) ([]Mount, error) { return nil, nil }
func (s *pagingProjectReaderRuntime) MountUsage(context.Context, Ref, string) (MountUsage, error) {
	return MountUsage{}, nil
}
func (s *pagingProjectReaderRuntime) Logs(context.Context, Ref, LogQuery) (Logs, error) {
	return Logs{}, nil
}
func (s *pagingProjectReaderRuntime) StreamLogs(context.Context, Ref, LogQuery, func(LogChunk) error) error {
	return nil
}
func (s *pagingProjectReaderRuntime) Shell(context.Context, Ref, string) (terminal.Session, error) {
	return nil, nil
}
func (s *pagingProjectReaderRuntime) Start(context.Context, Ref) (ActionResult, error) {
	return ActionResult{}, nil
}
func (s *pagingProjectReaderRuntime) Stop(context.Context, Ref) (ActionResult, error) {
	return ActionResult{}, nil
}
func (s *pagingProjectReaderRuntime) Restart(context.Context, Ref) (ActionResult, error) {
	return ActionResult{}, nil
}
func (s *pagingProjectReaderRuntime) Remove(context.Context, Ref, RemoveOptions) (ActionResult, error) {
	return ActionResult{}, nil
}
func (s *pagingProjectReaderRuntime) Close() error { return nil }

func TestContainerProjectRuntimeReaderListsAllPages(t *testing.T) {
	t.Parallel()

	firstPage := make([]Summary, 0, maxContainerListLimit)
	for i := 0; i < maxContainerListLimit; i++ {
		firstPage = append(firstPage, Summary{
			ID:             "id-first-" + string(rune('a'+(i%26))),
			Name:           "demo-web",
			State:          "running",
			ComposeProject: "demo",
			ComposeService: "svc",
		})
	}
	runtime := &pagingProjectReaderRuntime{
		pages: [][]Summary{
			firstPage,
			{
				{ID: "tail", Name: "demo-tail", State: "exited", ComposeProject: "demo", ComposeService: "worker"},
			},
		},
	}
	reader := containerProjectRuntimeReader{
		service: &service{
			runtime: runtime,
			enabled: true,
		},
	}

	summary, err := reader.ListProjectMembers(context.Background(), "local", "demo")
	if err != nil {
		t.Fatalf("list project members: %v", err)
	}
	if len(summary.Members) != maxContainerListLimit+1 {
		t.Fatalf("expected %d members, got %d", maxContainerListLimit+1, len(summary.Members))
	}
	if summary.RunningCount != maxContainerListLimit || summary.StoppedCount != 1 {
		t.Fatalf("unexpected counts: %#v", summary)
	}
	if len(runtime.seen) != 2 {
		t.Fatalf("expected 2 list calls, got %d", len(runtime.seen))
	}
	if runtime.seen[0].Offset != 0 || runtime.seen[1].Offset != maxContainerListLimit {
		t.Fatalf("unexpected paging offsets: %#v", runtime.seen)
	}
}

func TestContainerProjectRuntimeReaderListsImportCandidates(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	composePath := filepath.Join(tempDir, "compose.yaml")
	overridePath := filepath.Join(tempDir, "docker-compose.override.yml")
	if err := os.WriteFile(composePath, []byte("services:\n  web:\n    image: nginx:latest\n"), 0o600); err != nil {
		t.Fatalf("write compose file: %v", err)
	}
	if err := os.WriteFile(overridePath, []byte("services:\n  worker:\n    image: busybox\n"), 0o600); err != nil {
		t.Fatalf("write override file: %v", err)
	}

	reader := containerProjectRuntimeReader{
		service: &service{
			runtime: stubProjectReaderRuntime{
				items: []Summary{
					{
						ID:             "1",
						Name:           "demo-web-1",
						State:          "running",
						Runtime:        runtimeNameDocker,
						ComposeProject: "demo",
						ComposeService: "web",
						Orchestrator: OrchestratorInfo{
							Type:        containerOrchestratorCompose,
							Project:     "demo",
							Service:     "web",
							ConfigFiles: []string{composePath, overridePath},
						},
					},
					{
						ID:             "2",
						Name:           "demo-worker-1",
						State:          "exited",
						Runtime:        runtimeNameDocker,
						ComposeProject: "demo",
						ComposeService: "worker",
						Orchestrator: OrchestratorInfo{
							Type:        containerOrchestratorCompose,
							Project:     "demo",
							Service:     "worker",
							ConfigFiles: []string{composePath, overridePath},
						},
					},
				},
			},
			enabled: true,
		},
	}

	candidates, err := reader.ListImportCandidates(context.Background(), "local")
	if err != nil {
		t.Fatalf("list import candidates: %v", err)
	}
	if len(candidates) != 1 {
		t.Fatalf("expected 1 candidate, got %d", len(candidates))
	}
	candidate := candidates[0]
	if candidate.Status != projectRuntimeCandidateStatusReady || !candidate.Importable {
		t.Fatalf("expected ready candidate, got %#v", candidate)
	}
	if candidate.WorkingDirectory != tempDir {
		t.Fatalf("expected derived working directory %q, got %q", tempDir, candidate.WorkingDirectory)
	}
	if candidate.WorkingDirectorySource != projectRuntimeWorkingDirSourceDerivedConfig {
		t.Fatalf("expected derived working directory source, got %q", candidate.WorkingDirectorySource)
	}
	if !slices.Equal(candidate.ConfigFiles, []string{composePath, overridePath}) {
		t.Fatalf("unexpected config files %#v", candidate.ConfigFiles)
	}
	if !slices.Equal(candidate.ServiceNames, []string{"web", "worker"}) {
		t.Fatalf("unexpected service names %#v", candidate.ServiceNames)
	}
	if candidate.ContainerCounts != (moduleapi.ContainerProjectRuntimeContainerCounts{Running: 1, Stopped: 1, Total: 2}) {
		t.Fatalf("unexpected counts %#v", candidate.ContainerCounts)
	}
	if !slices.Contains(candidate.Warnings, projectRuntimeWarningWorkingDirDerived) {
		t.Fatalf("expected derived working directory warning, got %#v", candidate.Warnings)
	}
}

func TestContainerProjectRuntimeReaderMarksIncompleteMetadataCandidates(t *testing.T) {
	t.Parallel()

	reader := containerProjectRuntimeReader{
		service: &service{
			runtime: stubProjectReaderRuntime{
				items: []Summary{
					{
						ID:             "1",
						Name:           "demo-web-1",
						State:          "running",
						Runtime:        runtimeNameDocker,
						ComposeProject: "demo",
						ComposeService: "web",
						Orchestrator: OrchestratorInfo{
							Type:    containerOrchestratorCompose,
							Project: "demo",
							Service: "web",
						},
					},
				},
			},
			enabled: true,
		},
	}

	candidates, err := reader.ListImportCandidates(context.Background(), "local")
	if err != nil {
		t.Fatalf("list import candidates: %v", err)
	}
	if len(candidates) != 1 {
		t.Fatalf("expected 1 candidate, got %d", len(candidates))
	}
	candidate := candidates[0]
	if candidate.Status != projectRuntimeCandidateStatusIncompleteMetadata || candidate.Importable {
		t.Fatalf("expected incomplete metadata candidate, got %#v", candidate)
	}
	if !slices.Contains(candidate.StatusReasonCodes, projectRuntimeReasonMissingConfigFiles) {
		t.Fatalf("expected missing config files reason, got %#v", candidate.StatusReasonCodes)
	}
}
