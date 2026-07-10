package project

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"sync"
	"sync/atomic"
	"testing"
	"time"

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
		Snapshot: &projectstore.Snapshot{ProjectID: 1, ConfigHash: "cfg-demo", RefreshedAt: time.Now().UTC()},
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

	deployAggregate := aggregate
	deployAggregate.Files = toStoreFiles(parseResult.ComposeFiles, parseResult.EnvFiles)
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

func TestResolveWorkspaceTooltipDisabledRuleDoesNotClearEarlierEnabledMatch(t *testing.T) {
	t.Parallel()

	tooltip, source := resolveWorkspaceTooltip(
		".env.local",
		false,
		"",
		[]workspaceTooltipRule{
			{Enabled: true, regex: mustWorkspaceTooltipRegex(t, `^\.env(?:\..+)?$`), Tooltip: "env"},
			{Enabled: false, regex: mustWorkspaceTooltipRegex(t, `^\.env\.local$`)},
		},
		nil,
	)
	if tooltip != "env" || source != "default-rule" {
		t.Fatalf("expected earlier enabled tooltip to survive disabled match, got tooltip=%q source=%q", tooltip, source)
	}
}

func mustWorkspaceTooltipRegex(t *testing.T, pattern string) *regexp.Regexp {
	t.Helper()
	regex, err := regexp.Compile(pattern)
	if err != nil {
		t.Fatalf("compile regex: %v", err)
	}
	return regex
}

type countingTopicMonitorHub struct {
	realtime.Hub
	runtimeRegisters atomic.Int32
	logRegisters     atomic.Int32
}

type countingProjectRuntimeReader struct {
	listProjectMemberCalls atomic.Int32
}

func (r *countingProjectRuntimeReader) ListProjectMembers(
	context.Context,
	string,
	string,
) (moduleapi.ContainerProjectRuntimeSummary, error) {
	r.listProjectMemberCalls.Add(1)
	return moduleapi.ContainerProjectRuntimeSummary{}, nil
}

func (*countingProjectRuntimeReader) ListImportCandidates(
	context.Context,
	string,
) ([]moduleapi.ContainerProjectRuntimeCandidate, error) {
	return nil, nil
}

func (*countingProjectRuntimeReader) ListImportCandidateMembers(
	context.Context,
	string,
	moduleapi.ContainerProjectRuntimeCandidate,
) ([]moduleapi.ContainerProjectMember, error) {
	return nil, nil
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
	case projectcontract.ProjectRuntimeTopicPrefix + "1":
		h.runtimeRegisters.Add(1)
	case projectcontract.ProjectLogsTopicPrefix + "1":
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
			if err := service.ensureProjectRuntimeTopicStreaming(projectcontract.ProjectRuntimeTopicPrefix+"1", 1); err != nil {
				t.Errorf("ensure runtime topic: %v", err)
			}
		}()
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			if err := service.ensureProjectLogsTopicStreaming(projectcontract.ProjectLogsTopicPrefix+"1", 1, LogQuery{Tail: 10, Stdout: true}); err != nil {
				t.Errorf("ensure logs topic: %v", err)
			}
		}()
	}
	close(start)
	wg.Wait()

	if got := hub.runtimeRegisters.Load(); got != 1 {
		t.Fatalf("expected one runtime observer registration, got %d", got)
	}
	if got := hub.logRegisters.Load(); got != 1 {
		t.Fatalf("expected one log observer registration, got %d", got)
	}
}

func TestLifecycleConfigRealtimePayloadDoesNotReadRuntimeSummary(t *testing.T) {
	t.Parallel()

	repo := &stubProjectRepository{aggregate: projectstore.ProjectAggregate{Project: projectstore.Project{
		ID:                   1,
		CanonicalProjectName: "demo",
		HostScope:            projectcontract.HostScopeLocal.String(),
		WorkingDirectory:     "/srv/demo",
	}}}
	runtimeReader := &countingProjectRuntimeReader{}
	service, err := NewService(repo, WithRuntimeReader(runtimeReader))
	if err != nil {
		t.Fatalf("new project service: %v", err)
	}

	payload, err := service.buildProjectLifecycleConfigRealtimePayload(
		context.Background(),
		projectcontract.ProjectLifecycleConfigTopicPrefix+"1",
		1,
	)
	if err != nil {
		t.Fatalf("build lifecycle configuration realtime payload: %v", err)
	}
	if payload.Detail.Id != 1 {
		t.Fatalf("expected payload detail for project 1, got %#v", payload.Detail)
	}
	if got := runtimeReader.listProjectMemberCalls.Load(); got != 0 {
		t.Fatalf("expected no runtime summary reads, got %d", got)
	}
}

type blockingTopicMonitorHub struct {
	realtime.Hub
	registerStarted chan struct{}
	releaseRegister chan struct{}
	unregisterCalls atomic.Int32
}

func newBlockingTopicMonitorHub() *blockingTopicMonitorHub {
	return &blockingTopicMonitorHub{
		Hub:             realtime.NewHub(),
		registerStarted: make(chan struct{}),
		releaseRegister: make(chan struct{}),
	}
}

func (h *blockingTopicMonitorHub) RegisterTopicObserver(
	_ string,
	_ func(string),
	_ func(string),
) (func(), error) {
	close(h.registerStarted)
	<-h.releaseRegister
	return func() {
		h.unregisterCalls.Add(1)
	}, nil
}

func TestProjectRuntimeTopicStreamerCloseUnregistersLateObserver(t *testing.T) {
	t.Parallel()

	hub := newBlockingTopicMonitorHub()
	streamer, err := newProjectRuntimeTopicStreamer(hub, nil, &Service{})
	if err != nil {
		t.Fatalf("new runtime topic streamer: %v", err)
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- streamer.EnsureTopic(projectcontract.ProjectRuntimeTopicPrefix+"1", 1)
	}()

	<-hub.registerStarted
	if err := streamer.Close(context.Background()); err != nil {
		t.Fatalf("close runtime topic streamer: %v", err)
	}
	close(hub.releaseRegister)

	if err := <-errCh; err != nil {
		t.Fatalf("ensure runtime topic: %v", err)
	}
	if got := hub.unregisterCalls.Load(); got != 1 {
		t.Fatalf("expected one late unregister for runtime topic, got %d", got)
	}
}

//nolint:dupl // The runtime and lifecycle stream types require independent regression coverage.
func TestProjectRuntimeTopicStreamerCloseKeepsTimedOutStreamRegistered(t *testing.T) {
	t.Parallel()

	streamCtx, cancelStream := context.WithCancel(context.Background())
	defer cancelStream()
	unregisterCalls := atomic.Int32{}
	topic := projectcontract.ProjectRuntimeTopicPrefix + "1"
	streamer := &projectRuntimeTopicStreamer{
		streams: map[string]*projectRuntimeTopicStream{
			topic: {
				topic:              topic,
				cancel:             cancelStream,
				done:               make(chan struct{}),
				unregisterObserver: func() { unregisterCalls.Add(1) },
			},
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := streamer.Close(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected canceled close error, got %v", err)
	}
	if streamer.streams[topic] == nil {
		t.Fatal("expected timed-out runtime stream to remain registered")
	}
	if got := unregisterCalls.Load(); got != 0 {
		t.Fatalf("expected no unregister after timed-out stop, got %d", got)
	}
	if streamCtx.Err() == nil {
		t.Fatal("expected stream cancellation to be requested")
	}
}

//nolint:dupl // The runtime and lifecycle stream types require independent regression coverage.
func TestProjectLifecycleConfigTopicStreamerCloseKeepsTimedOutStreamRegistered(t *testing.T) {
	t.Parallel()

	streamCtx, cancelStream := context.WithCancel(context.Background())
	defer cancelStream()
	unregisterCalls := atomic.Int32{}
	topic := projectcontract.ProjectLifecycleConfigTopicPrefix + "1"
	streamer := &projectLifecycleConfigTopicStreamer{
		streams: map[string]*projectLifecycleConfigTopicStream{
			topic: {
				topic:              topic,
				cancel:             cancelStream,
				done:               make(chan struct{}),
				unregisterObserver: func() { unregisterCalls.Add(1) },
			},
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := streamer.Close(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected canceled close error, got %v", err)
	}
	if streamer.streams[topic] == nil {
		t.Fatal("expected timed-out lifecycle configuration stream to remain registered")
	}
	if got := unregisterCalls.Load(); got != 0 {
		t.Fatalf("expected no unregister after timed-out stop, got %d", got)
	}
	if streamCtx.Err() == nil {
		t.Fatal("expected stream cancellation to be requested")
	}
}

func TestProjectLogTopicStreamerCloseUnregistersLateObserver(t *testing.T) {
	t.Parallel()

	hub := newBlockingTopicMonitorHub()
	streamer, err := newProjectLogTopicStreamer(hub, nil, &Service{})
	if err != nil {
		t.Fatalf("new log topic streamer: %v", err)
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- streamer.EnsureTopic(projectcontract.ProjectLogsTopicPrefix+"1", 1, LogQuery{Tail: 10, Stdout: true})
	}()

	<-hub.registerStarted
	if err := streamer.Close(context.Background()); err != nil {
		t.Fatalf("close log topic streamer: %v", err)
	}
	close(hub.releaseRegister)

	if err := <-errCh; err != nil {
		t.Fatalf("ensure log topic: %v", err)
	}
	if got := hub.unregisterCalls.Load(); got != 1 {
		t.Fatalf("expected one late unregister for log topic, got %d", got)
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

func (r *pagedConflictRepository) UpdateWorkspaceAnnotation(context.Context, projectstore.UpdateWorkspaceAnnotationInput) (projectstore.ProjectAggregate, error) {
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
