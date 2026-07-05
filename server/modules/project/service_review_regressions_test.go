package project

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"

	"graft/server/internal/moduleapi"
	"graft/server/internal/realtime"
	projectcompose "graft/server/modules/project/compose"
	projectcontract "graft/server/modules/project/contract"
	projectstore "graft/server/modules/project/store"
)

func TestConfigurationDeployAggregateUsesPreparedComposeFiles(t *testing.T) {
	t.Parallel()

	aggregate := projectstore.ProjectAggregate{
		Project: projectstore.Project{
			ID:                    1,
			HostScope:             projectcontract.HostScopeLocal.String(),
			WorkingDirectory:      "/srv/demo",
			CanonicalProjectName:  "demo",
			LastRefreshStatus:     projectcontract.RefreshStatusSuccess.String(),
			LifecycleReviewStatus: projectcontract.LifecycleReviewStatusConfirmed.String(),
		},
		Files: []projectstore.ProjectFile{
			{
				Kind:         projectcontract.FileKindCompose.String(),
				Role:         projectcontract.FileRolePrimary.String(),
				AbsolutePath: "/srv/demo/compose.old.yaml",
				DisplayPath:  "compose.old.yaml",
			},
		},
	}
	parseResult := projectcompose.Result{
		ComposeFiles: []projectcompose.FileProjection{
			{
				Kind:         projectcontract.FileKindCompose.String(),
				Role:         projectcontract.FileRolePrimary.String(),
				AbsolutePath: "/srv/demo/compose.new.yaml",
				DisplayPath:  "compose.new.yaml",
			},
		},
	}

	deployAggregate := configurationDeployAggregate(aggregate, parseResult)
	args, err := lifecycleUpArgs(deployAggregate, lifecycleConfigurationFromAggregate(deployAggregate))
	if err != nil {
		t.Fatalf("build deploy lifecycle args: %v", err)
	}
	assertContainsArg(t, args, "/srv/demo/compose.new.yaml")
	assertNotContainsArg(t, args, "/srv/demo/compose.old.yaml")
}

func TestUpdateLifecycleConfigurationReturnsRepositoryAggregate(t *testing.T) {
	t.Parallel()

	repo := &stubProjectRepository{
		aggregate: projectstore.ProjectAggregate{
			Project: projectstore.Project{
				ID: 77,
			},
		},
	}
	service, err := NewService(repo)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	result, err := service.UpdateLifecycleConfiguration(context.Background(), 77, LifecycleStandardConfig{
		Profiles:           []string{"blue", "blue", " green "},
		DownBeforeRedeploy: true,
	}, nil)
	if err != nil {
		t.Fatalf("update lifecycle configuration: %v", err)
	}
	if repo.getCalls != 0 {
		t.Fatalf("expected no follow-up Get call, got %d", repo.getCalls)
	}
	if result.Project.ID != 77 {
		t.Fatalf("expected repository aggregate to be returned, got %#v", result)
	}
	if len(result.Project.LifecycleConfig.Profiles) != 2 || result.Project.LifecycleConfig.Profiles[0] != "blue" || result.Project.LifecycleConfig.Profiles[1] != "green" {
		t.Fatalf("expected normalized profiles in returned aggregate, got %#v", result.Project.LifecycleConfig.Profiles)
	}
}

func TestListProjectConflictScanItemsPaginatesBeyondFirstPage(t *testing.T) {
	t.Parallel()

	repo := &pagedConflictRepository{total: projectConflictScanSize + 1}
	items, err := listProjectConflictScanItems(context.Background(), repo)
	if err != nil {
		t.Fatalf("list conflict scan items: %v", err)
	}
	if len(items) != projectConflictScanSize+1 {
		t.Fatalf("expected %d items, got %d", projectConflictScanSize+1, len(items))
	}
	if len(repo.listCalls) != 2 {
		t.Fatalf("expected two paged List calls, got %#v", repo.listCalls)
	}
	if repo.listCalls[0] != (projectstore.ListQuery{Limit: projectConflictScanSize, Offset: 0}) {
		t.Fatalf("unexpected first query %#v", repo.listCalls[0])
	}
	if repo.listCalls[1] != (projectstore.ListQuery{Limit: projectConflictScanSize, Offset: projectConflictScanSize}) {
		t.Fatalf("unexpected second query %#v", repo.listCalls[1])
	}
}

func TestListRuntimeImportCandidatesMarksAlreadyImportedBeyondFirstPage(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	composePath := filepath.Join(tempDir, "compose.yaml")
	if err := os.WriteFile(composePath, []byte("services:\n  web:\n    image: nginx:latest\n"), 0o600); err != nil {
		t.Fatalf("write compose file: %v", err)
	}
	repo := &pagedConflictRepository{
		total: projectConflictScanSize + 1,
		override: map[int]projectstore.ProjectAggregate{
			projectConflictScanSize: {
				Project: projectstore.Project{
					ID:                   uint64(projectConflictScanSize + 1),
					CanonicalProjectName: "demo",
					WorkingDirectory:     tempDir,
				},
			},
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
		t.Fatalf("expected one runtime candidate, got %#v", result.Items)
	}
	if result.Items[0].Status != importRuntimeCandidateStatusAlreadyImported {
		t.Fatalf("expected paged conflict to mark candidate imported, got %#v", result.Items[0])
	}
}

type countingTopicMonitorHub struct {
	realtime.Hub
	detailRegisters atomic.Int32
	logRegisters    atomic.Int32
}

func newCountingTopicMonitorHub() *countingTopicMonitorHub {
	return &countingTopicMonitorHub{Hub: realtime.NewHub()}
}

func (h *countingTopicMonitorHub) RegisterTopicObserver(
	topic string,
	onActive func(string),
	onInactive func(string),
) (func(), error) {
	switch topic {
	case "projects.detail.1":
		h.detailRegisters.Add(1)
	case "projects.logs.1":
		h.logRegisters.Add(1)
	}
	monitor, ok := h.Hub.(realtime.TopicSubscriptionMonitor)
	if !ok {
		return nil, nil
	}
	return monitor.RegisterTopicObserver(topic, onActive, onInactive)
}

func TestRealtimeTopicStreamingInitializersRegisterObserverOnce(t *testing.T) {
	t.Parallel()

	repo := &stubProjectRepository{
		aggregate: projectstore.ProjectAggregate{
			Project: projectstore.Project{
				ID:                   1,
				CanonicalProjectName: "demo",
				HostScope:            "local",
				WorkingDirectory:     "/srv/demo",
			},
		},
	}
	hub := newCountingTopicMonitorHub()
	service, err := NewService(
		repo,
		WithLogReader(stubLogReader{}),
	)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	service.realtimeHub = hub

	start := make(chan struct{})
	var wg sync.WaitGroup
	for range 16 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			if err := service.ensureProjectDetailTopicStreaming("projects.detail.1", 1); err != nil {
				t.Errorf("ensure detail topic: %v", err)
			}
		}()
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			if err := service.ensureProjectLogsTopicStreaming("projects.logs.1", 1, LogQuery{Tail: 10, Stdout: true}); err != nil {
				t.Errorf("ensure logs topic: %v", err)
			}
		}()
	}
	close(start)
	wg.Wait()

	if got := hub.detailRegisters.Load(); got != 1 {
		t.Fatalf("expected one detail observer registration, got %d", got)
	}
	if got := hub.logRegisters.Load(); got != 1 {
		t.Fatalf("expected one log observer registration, got %d", got)
	}
}

type pagedConflictRepository struct {
	total     int
	override  map[int]projectstore.ProjectAggregate
	listCalls []projectstore.ListQuery
}

func (r *pagedConflictRepository) List(_ context.Context, query projectstore.ListQuery) (projectstore.ListResult, error) {
	r.listCalls = append(r.listCalls, query)
	if query.Offset >= r.total {
		return projectstore.ListResult{Items: nil, Total: r.total}, nil
	}
	end := minInt(query.Offset+query.Limit, r.total)
	items := make([]projectstore.ProjectAggregate, 0, end-query.Offset)
	for index := query.Offset; index < end; index++ {
		if aggregate, ok := r.override[index]; ok {
			items = append(items, aggregate)
			continue
		}
		items = append(items, projectstore.ProjectAggregate{
			Project: projectstore.Project{
				ID:                   1,
				CanonicalProjectName: "project-" + string(rune('a'+(index%26))),
				WorkingDirectory:     filepath.Join("/srv/projects", "project"),
			},
		})
	}
	return projectstore.ListResult{Items: items, Total: r.total}, nil
}

func (r *pagedConflictRepository) Get(context.Context, uint64) (projectstore.ProjectAggregate, error) {
	return projectstore.ProjectAggregate{}, projectstore.ErrProjectNotFound
}

func (r *pagedConflictRepository) GetFile(context.Context, uint64, uint64) (projectstore.ProjectFile, error) {
	return projectstore.ProjectFile{}, projectstore.ErrFileNotFound
}

func (r *pagedConflictRepository) ImportProject(context.Context, projectstore.ImportProjectInput) (projectstore.ProjectAggregate, error) {
	return projectstore.ProjectAggregate{}, projectstore.ErrInvalidInput
}

func (r *pagedConflictRepository) RefreshProject(context.Context, projectstore.RefreshProjectInput) (projectstore.ProjectAggregate, error) {
	return projectstore.ProjectAggregate{}, projectstore.ErrInvalidInput
}

func (r *pagedConflictRepository) UpdateLifecycleConfig(context.Context, projectstore.UpdateLifecycleConfigInput) (projectstore.ProjectAggregate, error) {
	return projectstore.ProjectAggregate{}, projectstore.ErrInvalidInput
}

func (r *pagedConflictRepository) UnregisterProject(context.Context, projectstore.UnregisterProjectInput) error {
	return projectstore.ErrInvalidInput
}

func assertContainsArg(t *testing.T, args []string, want string) {
	t.Helper()
	for _, arg := range args {
		if arg == want {
			return
		}
	}
	t.Fatalf("expected args %#v to contain %q", args, want)
}

func assertNotContainsArg(t *testing.T, args []string, want string) {
	t.Helper()
	for _, arg := range args {
		if arg == want {
			t.Fatalf("expected args %#v not to contain %q", args, want)
		}
	}
}
