package project

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	generated "graft/server/internal/contract/openapi/generated"
	"graft/server/internal/eventbus"
	"graft/server/internal/httpx"
	"graft/server/internal/module"
	"graft/server/internal/moduleapi"
	projectcompose "graft/server/modules/project/compose"
	projectcontract "graft/server/modules/project/contract"
	projectstore "graft/server/modules/project/store"
)

type stubProjectRepository struct {
	aggregate        projectstore.ApplicationAggregate
	listInput        *projectstore.ListQuery
	unregisterCalled bool
	unregisterInput  *projectstore.UnregisterApplicationInput
	unregisterErr    error
	importInput      *projectstore.ImportApplicationInput
	refreshInput     *projectstore.RefreshApplicationInput
	refreshErr       error
	refreshFn        func(projectstore.RefreshApplicationInput) (projectstore.ApplicationAggregate, error)
	getCalls         int
}

func (s *stubProjectRepository) List(_ context.Context, query projectstore.ListQuery) (projectstore.ListResult, error) {
	recorded := query
	s.listInput = &recorded
	return projectstore.ListResult{Items: []projectstore.ApplicationAggregate{s.aggregate}, Total: 1}, nil
}

func (s *stubProjectRepository) BackfillRuntimeTarget(context.Context, uint64) error { return nil }

func (s *stubProjectRepository) Get(context.Context, uint64) (projectstore.ApplicationAggregate, error) {
	s.getCalls++
	if s.aggregate.Application.ApplicationRecordID == 0 {
		return projectstore.ApplicationAggregate{}, projectstore.ErrApplicationNotFound
	}
	return ensureStubApplicationAggregateDefaults(s.aggregate), nil
}

func (s *stubProjectRepository) GetFile(context.Context, uint64, uint64) (projectstore.ApplicationFile, error) {
	return projectstore.ApplicationFile{}, projectstore.ErrFileNotFound
}

func (s *stubProjectRepository) GetByApplicationName(_ context.Context, applicationName string) (projectstore.ApplicationAggregate, error) {
	if s.aggregate.Application.ApplicationName == nil || *s.aggregate.Application.ApplicationName != applicationName || s.aggregate.Application.ApplicationRecordID == 0 {
		return projectstore.ApplicationAggregate{}, projectstore.ErrApplicationNotFound
	}
	return ensureStubApplicationAggregateDefaults(s.aggregate), nil
}

func (s *stubProjectRepository) ImportApplication(_ context.Context, input projectstore.ImportApplicationInput) (projectstore.ApplicationAggregate, error) {
	s.importInput = &input
	if s.aggregate.Application.ApplicationRecordID == 0 {
		s.aggregate.Application.ApplicationRecordID = 99
	}
	s.aggregate.Application.DisplayName = input.DisplayName
	s.aggregate.Application.ComposeProjectName = input.ComposeProjectName
	s.aggregate.Application.ComposeProjectNameSource = input.ComposeProjectNameSource
	s.aggregate.Application.SourceType = input.SourceType
	s.aggregate.Application.WorkspacePath = input.WorkspacePath
	s.aggregate.Application.OwnershipMode = input.OwnershipMode
	s.aggregate.Application.LifecycleStrategyKind = input.LifecycleStrategyKind
	s.aggregate.Application.LifecycleReviewStatus = input.LifecycleReviewStatus
	s.aggregate.Application.LifecycleConfig = input.LifecycleConfig
	s.aggregate.Application.LastObservedConfigHash = input.LastObservedConfigHash
	s.aggregate.Application.LastDriftCheckedAt = input.LastDriftCheckedAt
	s.aggregate.Application.DriftStatus = input.DriftStatus
	files := append([]projectstore.ApplicationFile(nil), input.Files...)
	for index := range files {
		if files[index].ID == 0 {
			files[index].ID = uint64(index + 1)
		}
		files[index].ApplicationRecordID = s.aggregate.Application.ApplicationRecordID
	}
	s.aggregate.Files = files
	s.aggregate.Snapshot = input.Snapshot
	return ensureStubApplicationAggregateDefaults(s.aggregate), nil
}

func (s *stubProjectRepository) RefreshApplication(_ context.Context, input projectstore.RefreshApplicationInput) (projectstore.ApplicationAggregate, error) {
	recorded := input
	s.refreshInput = &recorded
	if s.refreshFn != nil {
		return s.refreshFn(input)
	}
	if s.refreshErr != nil {
		return projectstore.ApplicationAggregate{}, s.refreshErr
	}
	s.aggregate.Application.LastObservedConfigHash = input.LastObservedConfigHash
	s.aggregate.Application.LastDriftCheckedAt = input.LastDriftCheckedAt
	s.aggregate.Application.DriftStatus = input.DriftStatus
	s.aggregate.Files = append([]projectstore.ApplicationFile(nil), input.Files...)
	s.aggregate.Snapshot = input.Snapshot
	return ensureStubApplicationAggregateDefaults(s.aggregate), nil
}

func (s *stubProjectRepository) UpdateLifecycleConfig(
	_ context.Context,
	input projectstore.UpdateLifecycleConfigInput,
) (projectstore.ApplicationAggregate, error) {
	s.aggregate.Application.LifecycleStrategyKind = input.LifecycleStrategyKind
	s.aggregate.Application.LifecycleReviewStatus = input.LifecycleReviewStatus
	s.aggregate.Application.LifecycleConfig = input.LifecycleConfig
	return ensureStubApplicationAggregateDefaults(s.aggregate), nil
}

func (s *stubProjectRepository) UpdateWorkspaceAnnotation(
	_ context.Context,
	input projectstore.UpdateWorkspaceAnnotationInput,
) (projectstore.ApplicationAggregate, error) {
	if s.aggregate.Application.WorkspaceAnnotations == nil {
		s.aggregate.Application.WorkspaceAnnotations = map[string]string{}
	}
	if input.Annotation == nil {
		delete(s.aggregate.Application.WorkspaceAnnotations, input.RelativePath)
	} else {
		s.aggregate.Application.WorkspaceAnnotations[input.RelativePath] = *input.Annotation
	}
	return ensureStubApplicationAggregateDefaults(s.aggregate), nil
}

func (s *stubProjectRepository) UnregisterApplication(_ context.Context, input projectstore.UnregisterApplicationInput) error {
	s.unregisterCalled = true
	recorded := input
	s.unregisterInput = &recorded
	return s.unregisterErr
}

func ensureStubApplicationAggregateDefaults(aggregate projectstore.ApplicationAggregate) projectstore.ApplicationAggregate {
	if aggregate.Application.LifecycleStrategyKind == "" {
		aggregate.Application.LifecycleStrategyKind = projectcontract.LifecycleStrategyKindStandard.String()
	}
	if aggregate.Application.LifecycleReviewStatus == "" {
		aggregate.Application.LifecycleReviewStatus = projectcontract.LifecycleReviewStatusConfirmed.String()
	}
	if len(aggregate.Files) == 0 && aggregate.Application.WorkspacePath != "" {
		aggregate.Files = []projectstore.ApplicationFile{
			{
				ID:                  1,
				ApplicationRecordID: aggregate.Application.ApplicationRecordID,
				Kind:                projectcontract.FileKindCompose.String(),
				Role:                projectcontract.FileRolePrimary.String(),
				AbsolutePath:        filepath.Join(aggregate.Application.WorkspacePath, "compose.yaml"),
				DisplayPath:         "compose.yaml",
				OrderIndex:          0,
			},
		}
	}
	return aggregate
}

type capturedAuditBus struct {
	mu        sync.Mutex
	events    []moduleapi.AuditEvent
	published chan struct{}
	blocked   <-chan struct{}
}

func (b *capturedAuditBus) Subscribe(string, eventbus.Handler) error {
	return nil
}

func (b *capturedAuditBus) Publish(_ context.Context, event eventbus.Event) error {
	if b.blocked != nil {
		<-b.blocked
	}
	auditEvent, ok := event.Payload.(moduleapi.AuditEvent)
	if !ok {
		return nil
	}
	b.mu.Lock()
	b.events = append(b.events, auditEvent)
	b.mu.Unlock()
	if b.published != nil {
		select {
		case b.published <- struct{}{}:
		default:
		}
	}
	return nil
}

func (b *capturedAuditBus) snapshot() []moduleapi.AuditEvent {
	b.mu.Lock()
	defer b.mu.Unlock()
	return append([]moduleapi.AuditEvent(nil), b.events...)
}

func (b *capturedAuditBus) waitForEvents(t *testing.T, count int, timeout time.Duration) []moduleapi.AuditEvent {
	t.Helper()
	deadline := time.After(timeout)
	for {
		events := b.snapshot()
		if len(events) >= count {
			return events
		}
		if b.published == nil {
			t.Fatalf("expected %d audit events, got %d", count, len(events))
		}
		select {
		case <-b.published:
		case <-deadline:
			t.Fatalf("timed out waiting for %d audit events, got %d", count, len(events))
		}
	}
}

func authenticatedApplicationActionContext() context.Context {
	ctx := context.Background()
	ctx = moduleapi.WithRequestAuthContext(ctx, moduleapi.RequestAuthContext{
		User: &moduleapi.CurrentUser{ID: 7, Username: "alice", DisplayName: "Alice"},
	})
	return httpx.WithRequestAuditContext(ctx, httpx.RequestAuditContext{
		RequestID: "req-project-1",
		TraceID:   "trace-project-1",
		Route:     "/api/projects/:id/action",
		Method:    "POST",
		ClientIP:  "127.0.0.1",
		UserAgent: "project-test",
	})
}

type stubRuntimeReader struct {
	summary          moduleapi.ContainerProjectRuntimeSummary
	candidates       []moduleapi.ContainerProjectRuntimeCandidate
	candidateMembers []moduleapi.ContainerProjectMember
}

type stubResourceReader struct {
	summary moduleapi.ContainerProjectResourceSummary
	err     error
}

type stubLogReader struct {
	snapshot moduleapi.ContainerProjectLogSnapshot
	err      error
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

func (s stubResourceReader) ReadProjectResourceSummary(context.Context, string, string) (moduleapi.ContainerProjectResourceSummary, error) {
	return s.summary, s.err
}

func (s stubLogReader) ReadProjectLogs(context.Context, string, string, moduleapi.ContainerProjectLogQuery) (moduleapi.ContainerProjectLogSnapshot, error) {
	return s.snapshot, s.err
}

func (s stubLogReader) StreamProjectLogs(
	context.Context,
	string,
	string,
	moduleapi.ContainerProjectLogQuery,
	func(moduleapi.ContainerProjectLogEntry) error,
) error {
	return s.err
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
		aggregate: projectstore.ApplicationAggregate{
			Application: projectstore.Application{
				ApplicationRecordID: 1,
				ApplicationID:       "app_01ARZ3NDEKTSV4RRFFQ69G5FAV",
				ComposeProjectName:  "demo",
				WorkspacePath:       tempDir,
				OwnershipMode:       "external",
				DriftStatus:         "clean",
			},
			Files: []projectstore.ApplicationFile{
				{
					ID:                  1,
					ApplicationRecordID: 1,
					Kind:                "compose",
					Role:                "primary",
					AbsolutePath:        composePath,
					DisplayPath:         composePath,
					OrderIndex:          0,
				},
			},
			Snapshot: &projectstore.Snapshot{
				ApplicationRecordID:  1,
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

func TestOverviewAggregatesRuntimeResources(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	composePath := filepath.Join(tempDir, "compose.yaml")
	content := []byte("services:\n  web:\n    image: nginx:latest\n    networks:\n      - frontend\n    volumes:\n      - data:/data\n  worker:\n    image: busybox\n    networks:\n      - backend\n")
	if err := os.WriteFile(composePath, content, 0o600); err != nil {
		t.Fatalf("write compose file: %v", err)
	}
	now := time.Now().UTC()
	repo := &stubProjectRepository{
		aggregate: projectstore.ApplicationAggregate{
			Application: projectstore.Application{
				ApplicationRecordID: 9,
				ComposeProjectName:  "demo",
				WorkspacePath:       tempDir,
			},
			Files: []projectstore.ApplicationFile{
				{Kind: "compose", Role: "primary", AbsolutePath: composePath, DisplayPath: "compose.yaml", OrderIndex: 1},
			},
			Snapshot: &projectstore.Snapshot{ApplicationRecordID: 9, ConfigHash: "cfg-demo", RefreshedAt: now},
		},
	}
	service, err := NewService(
		repo,
		WithResourceReader(stubResourceReader{
			summary: moduleapi.ContainerProjectResourceSummary{
				CanonicalProjectName:         "demo",
				CollectedAt:                  now.Format(time.RFC3339),
				StatsAvailable:               true,
				StatsAvailableContainerCount: 2,
				HealthyContainerCount:        1,
				UnhealthyContainerCount:      1,
				StartingContainerCount:       0,
				RestartCount:                 3,
				CPUPercent:                   12.5,
				MemoryUsageBytes:             300,
				MemoryLimitBytes:             600,
				RxBytes:                      1024,
				TxBytes:                      2048,
				Services: []moduleapi.ContainerProjectServiceResourceSummary{
					{
						ServiceName:                  "web",
						ContainerCount:               1,
						RunningCount:                 1,
						HealthyContainerCount:        1,
						StatsAvailable:               true,
						StatsAvailableContainerCount: 1,
						CPUPercent:                   10,
						MemoryUsageBytes:             200,
						MemoryLimitBytes:             400,
					},
					{
						ServiceName:                  "worker",
						ContainerCount:               1,
						IssueCount:                   1,
						UnhealthyContainerCount:      1,
						RestartCount:                 3,
						StatsAvailable:               true,
						StatsAvailableContainerCount: 1,
						CPUPercent:                   2.5,
						MemoryUsageBytes:             100,
						MemoryLimitBytes:             200,
					},
				},
			},
		}),
	)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	response, err := service.Overview(context.Background(), 9)
	if err != nil {
		t.Fatalf("overview: %v", err)
	}
	if response.Health.HealthyServiceCount != 1 {
		t.Fatalf("expected one healthy service, got %#v", response.Health)
	}
	if response.Health.NetworksCount != 2 || response.Health.VolumesCount != 1 {
		t.Fatalf("expected derived topology counts, got %#v", response.Health)
	}
	if response.Resources.CpuPercent != 12.5 || response.Resources.MemoryUsageBytes != 300 || response.Resources.TxBytes != 2048 {
		t.Fatalf("unexpected overview resources %#v", response.Resources)
	}
	if len(response.Services) != 2 {
		t.Fatalf("expected two overview services, got %#v", response.Services)
	}
	if response.Services[0].ServiceName != "web" || response.Services[0].Status != generated.ApplicationOverviewServiceItemStatus("running") {
		t.Fatalf("unexpected first overview service %#v", response.Services[0])
	}
	if response.Services[1].ServiceName != "worker" || response.Services[1].Health != generated.ApplicationOverviewServiceItemHealth("attention") {
		t.Fatalf("unexpected second overview service %#v", response.Services[1])
	}
}

func TestDestroyBlocksExternalWorkspacePathDeletion(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	repo := &stubProjectRepository{
		aggregate: projectstore.ApplicationAggregate{
			Application: projectstore.Application{
				ApplicationRecordID: 1,
				ApplicationID:       "app_01ARZ3NDEKTSV4RRFFQ69G5FAV",
				ComposeProjectName:  "demo",
				WorkspacePath:       tempDir,
				OwnershipMode:       "external",
			},
			Snapshot: &projectstore.Snapshot{ApplicationRecordID: 1, ConfigHash: "cfg-demo", RefreshedAt: time.Now().UTC()},
		},
	}
	service, err := NewService(repo)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	result, err := service.Destroy(authenticatedApplicationActionContext(), 1, DestroyRequest{
		DeleteWorkspacePath:       true,
		ConfirmComposeProjectName: "app_01ARZ3NDEKTSV4RRFFQ69G5FAV",
	})
	if !errors.Is(err, errProjectDestroyBlocked) {
		t.Fatalf("expected destroy blocked, got %v", err)
	}
	if result.Result != generated.ApplicationActionResponseResultApplicationActionResultBlocked {
		t.Fatalf("expected blocked result, got %s", result.Result)
	}
	if repo.unregisterCalled {
		t.Fatalf("unregister should not be called when destroy is blocked")
	}
}

func TestUnregisterUsesRequestActorAndPublishesAudit(t *testing.T) {
	t.Parallel()

	repo := &stubProjectRepository{
		aggregate: projectstore.ApplicationAggregate{
			Application: projectstore.Application{
				ApplicationRecordID: 1,
				ApplicationID:       "app_01ARZ3NDEKTSV4RRFFQ69G5FAV",
				ComposeProjectName:  "demo",
				WorkspacePath:       t.TempDir(),
			},
			Snapshot: &projectstore.Snapshot{ApplicationRecordID: 1, ConfigHash: "cfg-demo", RefreshedAt: time.Now().UTC()},
		},
	}
	service, err := NewService(repo)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	auditBus := &capturedAuditBus{}
	service.SetAuditPublisher(auditBus, nil, moduleID)

	result, err := service.Unregister(authenticatedApplicationActionContext(), 1, nil)
	if err != nil {
		t.Fatalf("unregister: %v", err)
	}
	if result.Result != generated.ApplicationActionResponseResultApplicationActionResultCompleted {
		t.Fatalf("expected completed result, got %#v", result)
	}
	if repo.unregisterInput == nil || repo.unregisterInput.ActorID == nil || *repo.unregisterInput.ActorID != 7 {
		t.Fatalf("expected unregister actor id 7, got %#v", repo.unregisterInput)
	}
	events := auditBus.snapshot()
	if len(events) != 1 {
		t.Fatalf("expected one audit event, got %d", len(events))
	}
	if events[0].Action != projectcontract.ApplicationAuditActionUnregister.String() {
		t.Fatalf("expected unregister audit action, got %#v", events[0])
	}
	if events[0].Operator == nil || events[0].Operator.ID != 7 {
		t.Fatalf("expected operator id 7, got %#v", events[0].Operator)
	}
}

func TestUnregisterFailsClosedWithoutRequestActor(t *testing.T) {
	t.Parallel()

	repo := &stubProjectRepository{
		aggregate: projectstore.ApplicationAggregate{
			Application: projectstore.Application{
				ApplicationRecordID: 1,
				ApplicationID:       "app_01ARZ3NDEKTSV4RRFFQ69G5FAV",
				ComposeProjectName:  "demo",
				WorkspacePath:       t.TempDir(),
			},
			Snapshot: &projectstore.Snapshot{ApplicationRecordID: 1, ConfigHash: "cfg-demo", RefreshedAt: time.Now().UTC()},
		},
	}
	service, err := NewService(repo)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	actorID := uint64(7)

	result, err := service.Unregister(context.Background(), 1, &actorID)
	if !errors.Is(err, errProjectActorAttribution) {
		t.Fatalf("expected actor attribution error, got %v", err)
	}
	if result.Result != generated.ApplicationActionResponseResultApplicationActionResultBlocked {
		t.Fatalf("expected blocked result, got %#v", result)
	}
	if len(result.GuardResults) != 1 || result.GuardResults[0].Code != "actor_attribution_required" {
		t.Fatalf("expected actor attribution guard, got %#v", result.GuardResults)
	}
	if repo.unregisterCalled {
		t.Fatalf("unregister should fail closed before repository mutation")
	}
}

func TestBatchActionFailsClosedWithoutRequestActor(t *testing.T) {
	t.Parallel()

	service, err := NewService(&stubProjectRepository{})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	actorID := uint64(7)

	result, err := service.BatchAction(context.Background(), BatchActionRequest{
		Action:               generated.ApplicationBatchActionRequestActionStart,
		ApplicationRecordIDs: []uint64{1, 2},
		ActorID:              &actorID,
	})
	if err != nil {
		t.Fatalf("batch action should fail closed through result semantics, got %v", err)
	}
	if result.BlockedCount != 2 || len(result.Items) != 2 {
		t.Fatalf("expected two blocked items, got %#v", result)
	}
	for _, item := range result.Items {
		if item.Result != generated.ApplicationActionResponseResultApplicationActionResultBlocked {
			t.Fatalf("expected blocked item, got %#v", item)
		}
		if len(item.GuardResults) != 1 || item.GuardResults[0].Code != "actor_attribution_required" {
			t.Fatalf("expected actor attribution guard, got %#v", item.GuardResults)
		}
	}
}

func TestBatchActionKeepsBlockedLifecycleItems(t *testing.T) {
	t.Parallel()

	repo := &stubProjectRepository{
		aggregate: projectstore.ApplicationAggregate{
			Application: projectstore.Application{
				ApplicationRecordID: 1,
				ApplicationID:       "app_01ARZ3NDEKTSV4RRFFQ69G5FAV",
				ComposeProjectName:  "demo",
				WorkspacePath:       filepath.Join(t.TempDir(), "missing"),
			},
			Snapshot: &projectstore.Snapshot{ApplicationRecordID: 1, ConfigHash: "cfg-demo", RefreshedAt: time.Now().UTC()},
		},
	}
	service, err := NewService(repo)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	result, err := service.BatchAction(authenticatedApplicationActionContext(), BatchActionRequest{
		Action:               generated.ApplicationBatchActionRequestActionStart,
		ApplicationRecordIDs: []uint64{1, 1},
	})
	if err != nil {
		t.Fatalf("batch action: %v", err)
	}
	if result.BlockedCount != 2 || len(result.Items) != 2 {
		t.Fatalf("expected two blocked items, got %#v", result)
	}
	for _, item := range result.Items {
		if item.Result != generated.ApplicationActionResponseResultApplicationActionResultBlocked {
			t.Fatalf("expected blocked item, got %#v", item)
		}
	}
}

func TestBatchDestroyRequiresExplicitConfirmation(t *testing.T) {
	t.Parallel()

	repo := &stubProjectRepository{
		aggregate: projectstore.ApplicationAggregate{
			Application: projectstore.Application{
				ApplicationRecordID: 1,
				ApplicationID:       "app_01ARZ3NDEKTSV4RRFFQ69G5FAV",
				ComposeProjectName:  "demo",
				WorkspacePath:       t.TempDir(),
				OwnershipMode:       projectcontract.OwnershipModeExternal.String(),
			},
			Snapshot: &projectstore.Snapshot{ApplicationRecordID: 1, ConfigHash: "cfg-demo", RefreshedAt: time.Now().UTC()},
		},
	}
	service, err := NewService(repo)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	result, err := service.BatchAction(authenticatedApplicationActionContext(), BatchActionRequest{
		Action:               generated.ApplicationBatchActionRequestActionDestroy,
		ApplicationRecordIDs: []uint64{1},
	})
	if err != nil {
		t.Fatalf("batch destroy: %v", err)
	}
	if !result.Items[0].Skipped {
		t.Fatalf("expected skipped destroy item, got %#v", result.Items[0])
	}
	if repo.unregisterCalled {
		t.Fatalf("destroy without confirmation should not unregister project")
	}
}

func TestSkipBatchRestartForStatusAllowsStoppedProjects(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name       string
		status     generated.ApplicationRuntimeStatus
		wantReason string
		wantSkip   bool
	}{
		{name: "running", status: generated.ApplicationRuntimeStatusRunning, wantReason: "", wantSkip: false},
		{name: "degraded", status: generated.ApplicationRuntimeStatusDegraded, wantReason: "", wantSkip: false},
		{name: "stopped", status: generated.ApplicationRuntimeStatusStopped, wantReason: "", wantSkip: false},
		{name: "transitioning", status: generated.ApplicationRuntimeStatusTransitioning, wantReason: "currently_transitioning", wantSkip: true},
		{name: "unknown", status: generated.ApplicationRuntimeStatusUnknown, wantReason: "runtime_status_unknown", wantSkip: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gotReason, gotSkip := skipBatchRestartForStatus(tc.status)
			if gotReason != tc.wantReason || gotSkip != tc.wantSkip {
				t.Fatalf("skipBatchRestartForStatus(%q) = (%q, %v), want (%q, %v)", tc.status, gotReason, gotSkip, tc.wantReason, tc.wantSkip)
			}
		})
	}
}

func TestBatchDestroyReturnsBlockedItemOnComposeFailure(t *testing.T) {
	t.Parallel()

	repo := &stubProjectRepository{
		aggregate: projectstore.ApplicationAggregate{
			Application: projectstore.Application{
				ApplicationRecordID: 1,
				ApplicationID:       "app_01ARZ3NDEKTSV4RRFFQ69G5FAV",
				ComposeProjectName:  "demo",
				WorkspacePath:       filepath.Join(t.TempDir(), "missing"),
				OwnershipMode:       projectcontract.OwnershipModeExternal.String(),
			},
			Snapshot: &projectstore.Snapshot{ApplicationRecordID: 1, ConfigHash: "cfg-demo", RefreshedAt: time.Now().UTC()},
		},
	}
	service, err := NewService(repo)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	confirmName := "app_01ARZ3NDEKTSV4RRFFQ69G5FAV"

	result, err := service.BatchAction(authenticatedApplicationActionContext(), BatchActionRequest{
		Action:                    generated.ApplicationBatchActionRequestActionDestroy,
		ApplicationRecordIDs:      []uint64{1},
		ConfirmComposeProjectName: &confirmName,
	})
	if err != nil {
		t.Fatalf("batch destroy: %v", err)
	}
	if result.BlockedCount != 1 || len(result.Items) != 1 {
		t.Fatalf("expected one blocked destroy item, got %#v", result)
	}
	for _, guard := range result.Items[0].GuardResults {
		if guard.Code == "compose_down_completed" {
			t.Fatalf("unexpected success guard on failed destroy: %#v", result.Items[0].GuardResults)
		}
	}
}

func TestBatchDestroyReturnsBlockedItemOnWorkspacePathDeleteFailure(t *testing.T) {
	dockerBinDir := t.TempDir()
	if err := os.Symlink("/bin/sh", filepath.Join(dockerBinDir, "docker")); err != nil {
		t.Fatalf("symlink docker stub: %v", err)
	}
	t.Setenv("PATH", dockerBinDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	workingDirectory := filepath.Join(t.TempDir(), "managed-root", "demo")
	if err := os.MkdirAll(workingDirectory, 0o750); err != nil {
		t.Fatalf("mkdir working directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(workingDirectory, "compose"), []byte("rm -rf \"$(dirname \"$PWD\")\"\nexit 0\n"), 0o600); err != nil {
		t.Fatalf("write compose stub: %v", err)
	}

	repo := &stubProjectRepository{
		aggregate: projectstore.ApplicationAggregate{
			Application: projectstore.Application{
				ApplicationRecordID: 1,
				ApplicationID:       "app_01ARZ3NDEKTSV4RRFFQ69G5FAV",
				ComposeProjectName:  "demo",
				WorkspacePath:       workingDirectory,
				OwnershipMode:       projectcontract.OwnershipModeManagedRootDedicated.String(),
			},
			Snapshot: &projectstore.Snapshot{ApplicationRecordID: 1, ConfigHash: "cfg-demo", RefreshedAt: time.Now().UTC()},
		},
	}
	service, err := NewService(repo)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	confirmName := "app_01ARZ3NDEKTSV4RRFFQ69G5FAV"

	result, err := service.BatchAction(authenticatedApplicationActionContext(), BatchActionRequest{
		Action:                    generated.ApplicationBatchActionRequestActionDestroy,
		ApplicationRecordIDs:      []uint64{1},
		DeleteWorkspacePath:       true,
		ConfirmComposeProjectName: &confirmName,
	})
	if err != nil {
		t.Fatalf("batch destroy: %v", err)
	}
	if result.BlockedCount != 1 || len(result.Items) != 1 {
		t.Fatalf("expected one blocked destroy item, got %#v", result)
	}
	if result.Items[0].Result != generated.ApplicationActionResponseResultApplicationActionResultBlocked {
		t.Fatalf("expected blocked destroy result, got %#v", result.Items[0])
	}
	if !slices.ContainsFunc(result.Items[0].GuardResults, func(guard GuardResult) bool {
		return guard.Code == "compose_down_completed"
	}) {
		t.Fatalf("expected compose-down guard after partial destroy, got %#v", result.Items[0].GuardResults)
	}
	if !slices.ContainsFunc(result.Items[0].GuardResults, func(guard GuardResult) bool {
		return guard.Code == "workspace_path_delete_failed" && guard.Detail != nil && *guard.Detail == "filesystem_error"
	}) {
		t.Fatalf("expected working-directory delete failure guard, got %#v", result.Items[0].GuardResults)
	}
}

func TestBatchDestroyReturnsBlockedItemOnUnregisterFailure(t *testing.T) {
	dockerBinDir := t.TempDir()
	if err := os.Symlink("/bin/sh", filepath.Join(dockerBinDir, "docker")); err != nil {
		t.Fatalf("symlink docker stub: %v", err)
	}
	t.Setenv("PATH", dockerBinDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	workingDirectory := filepath.Join(t.TempDir(), "demo")
	if err := os.MkdirAll(workingDirectory, 0o750); err != nil {
		t.Fatalf("mkdir working directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(workingDirectory, "compose"), []byte("exit 0\n"), 0o600); err != nil {
		t.Fatalf("write compose stub: %v", err)
	}

	repo := &stubProjectRepository{
		aggregate: projectstore.ApplicationAggregate{
			Application: projectstore.Application{
				ApplicationRecordID: 1,
				ApplicationID:       "app_01ARZ3NDEKTSV4RRFFQ69G5FAV",
				ComposeProjectName:  "demo",
				WorkspacePath:       workingDirectory,
				OwnershipMode:       projectcontract.OwnershipModeExternal.String(),
			},
			Snapshot: &projectstore.Snapshot{ApplicationRecordID: 1, ConfigHash: "cfg-demo", RefreshedAt: time.Now().UTC()},
		},
		unregisterErr: projectstore.ErrApplicationConflict,
	}
	service, err := NewService(repo)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	confirmName := "app_01ARZ3NDEKTSV4RRFFQ69G5FAV"

	result, err := service.BatchAction(authenticatedApplicationActionContext(), BatchActionRequest{
		Action:                    generated.ApplicationBatchActionRequestActionDestroy,
		ApplicationRecordIDs:      []uint64{1},
		AutoUnregister:            true,
		ConfirmComposeProjectName: &confirmName,
	})
	if err != nil {
		t.Fatalf("batch destroy: %v", err)
	}
	if result.BlockedCount != 1 || len(result.Items) != 1 {
		t.Fatalf("expected one blocked destroy item, got %#v", result)
	}
	if !repo.unregisterCalled {
		t.Fatalf("expected unregister to be attempted")
	}
	if result.Items[0].Result != generated.ApplicationActionResponseResultApplicationActionResultBlocked {
		t.Fatalf("expected blocked destroy result, got %#v", result.Items[0])
	}
	if !slices.ContainsFunc(result.Items[0].GuardResults, func(guard GuardResult) bool {
		return guard.Code == "compose_down_completed"
	}) {
		t.Fatalf("expected compose-down guard after partial destroy, got %#v", result.Items[0].GuardResults)
	}
	if !slices.ContainsFunc(result.Items[0].GuardResults, func(guard GuardResult) bool {
		return guard.Code == "registry_delete_failed" && guard.Detail != nil && *guard.Detail == "persistence_error"
	}) {
		t.Fatalf("expected registry delete failure guard, got %#v", result.Items[0].GuardResults)
	}
}

func TestBatchRedeployReusesLoadedAggregate(t *testing.T) {
	t.Parallel()

	repo := &stubProjectRepository{
		aggregate: projectstore.ApplicationAggregate{
			Application: projectstore.Application{
				ApplicationRecordID: 1,
				ComposeProjectName:  "demo",
				WorkspacePath:       filepath.Join(t.TempDir(), "missing"),
			},
			Snapshot: &projectstore.Snapshot{ApplicationRecordID: 1, ConfigHash: "cfg-demo", RefreshedAt: time.Now().UTC()},
		},
	}
	service, err := NewService(repo)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	result, err := service.BatchAction(authenticatedApplicationActionContext(), BatchActionRequest{
		Action:               generated.ApplicationBatchActionRequestActionRedeploy,
		ApplicationRecordIDs: []uint64{1},
	})
	if err != nil {
		t.Fatalf("batch redeploy: %v", err)
	}
	if result.BlockedCount != 1 {
		t.Fatalf("expected blocked redeploy result, got %#v", result)
	}
	if repo.getCalls != 1 {
		t.Fatalf("expected one aggregate lookup, got %d", repo.getCalls)
	}
}

func TestBatchActionDoesNotWaitForBatchAuditPublish(t *testing.T) {
	t.Parallel()

	repo := &stubProjectRepository{
		aggregate: projectstore.ApplicationAggregate{
			Application: projectstore.Application{
				ApplicationRecordID: 1,
				ComposeProjectName:  "demo",
				WorkspacePath:       filepath.Join(t.TempDir(), "missing"),
			},
			Snapshot: &projectstore.Snapshot{ApplicationRecordID: 1, ConfigHash: "cfg-demo", RefreshedAt: time.Now().UTC()},
		},
	}
	service, err := NewService(repo)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	releaseAudit := make(chan struct{})
	auditBus := &capturedAuditBus{
		published: make(chan struct{}, 1),
		blocked:   releaseAudit,
	}
	service.SetAuditPublisher(auditBus, nil, moduleID)

	resultCh := make(chan BatchActionResult, 1)
	errCh := make(chan error, 1)
	go func() {
		result, batchErr := service.BatchAction(authenticatedApplicationActionContext(), BatchActionRequest{
			Action:               generated.ApplicationBatchActionRequestActionStart,
			ApplicationRecordIDs: []uint64{1},
		})
		resultCh <- result
		errCh <- batchErr
	}()

	select {
	case result := <-resultCh:
		if err := <-errCh; err != nil {
			t.Fatalf("batch action: %v", err)
		}
		if result.BlockedCount != 1 || len(result.Items) != 1 {
			t.Fatalf("expected one blocked item, got %#v", result)
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatal("batch action should not wait for batch audit publish")
	}

	close(releaseAudit)
	events := auditBus.waitForEvents(t, 1, time.Second)
	if events[0].Action != projectcontract.ApplicationAuditActionBatchStart.String() {
		t.Fatalf("expected batch-start audit action, got %#v", events[0])
	}
}

func TestCreateManagedApplicationWritesFilesAndPersistsRegistry(t *testing.T) {
	t.Parallel()

	managedRoot := t.TempDir()
	repo := &stubProjectRepository{}
	service, err := NewService(repo, WithSystemConfigResolver(stubSystemConfigResolver{value: managedRoot}))
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	envName := ".env"
	composeContent := "services:\n  web:\n    image: nginx:latest\n"
	envContent := "FOO=bar\n"
	result, err := service.CreateManagedApplication(context.Background(), ManagedApplicationCreateRequest{
		DisplayName:        "Demo",
		ApplicationName:    stringPointer("demo"),
		ComposeFileName:    "compose.yaml",
		ComposeFileContent: composeContent,
		EnvFileName:        &envName,
		EnvFileContent:     &envContent,
		ComposeFilePath:    "compose.yaml",
		WorkspaceEntries:   []ManagedWorkspaceEntry{{Path: "compose.yaml", NodeType: "file", Content: &composeContent}, {Path: ".env", NodeType: "file", Content: &envContent}},
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
	if repo.importInput.SourceType != "managed" {
		t.Fatalf("expected managed source kind, got %q", repo.importInput.SourceType)
	}
	if result.ApplicationRecordID == 0 {
		t.Fatalf("expected created project id")
	}
	if result.DeclaredServiceCount != 1 {
		t.Fatalf("expected one declared service, got %d", result.DeclaredServiceCount)
	}
}

func TestCreateManagedApplicationRejectsManagedRootBaseDirectory(t *testing.T) {
	t.Parallel()

	managedRoot := t.TempDir()
	repo := &stubProjectRepository{}
	service, err := NewService(repo, WithSystemConfigResolver(stubSystemConfigResolver{value: managedRoot}))
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	_, err = service.CreateManagedApplication(context.Background(), ManagedApplicationCreateRequest{
		DisplayName:        "Demo",
		ApplicationName:    stringPointer("demo"),
		ComposeFileName:    "compose.yaml",
		ComposeFileContent: "services:\n  web:\n    image: nginx:latest\n",
	}, nil)
	if !errors.Is(err, errProjectInvalidArgument) {
		t.Fatalf("expected invalid argument, got %v", err)
	}
}

func TestCreateManagedApplicationMaterializesNestedWorkspaceFiles(t *testing.T) {
	t.Parallel()
	managedRoot := t.TempDir()
	service, err := NewService(&stubProjectRepository{}, WithSystemConfigResolver(stubSystemConfigResolver{value: managedRoot}))
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	_, err = service.CreateManagedApplication(context.Background(), ManagedApplicationCreateRequest{
		DisplayName: "Demo", ApplicationName: stringPointer("demo"), ComposeFileName: "compose.yaml", ComposeFileContent: "services:\n  web:\n    image: nginx:latest\n",
		ComposeFilePath:  "compose.yaml",
		WorkspaceEntries: []ManagedWorkspaceEntry{{Path: "compose.yaml", NodeType: "file", Content: stringPointer("services:\n  web:\n    image: nginx:latest\n")}, {Path: "nginx/nginx.conf", NodeType: "file", Content: stringPointer("events {}\n")}, {Path: ".env.production", NodeType: "file", Content: stringPointer("MODE=production\n")}},
	}, nil)
	if err != nil {
		t.Fatalf("create managed workspace: %v", err)
	}
	for _, path := range []string{"nginx/nginx.conf", ".env.production"} {
		if _, statErr := os.Stat(filepath.Join(managedRoot, "demo", path)); statErr != nil {
			t.Fatalf("expected workspace file %s: %v", path, statErr)
		}
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
	if result.FilterCounts.All != 1 || result.FilterCounts.Imported != 0 || result.FilterCounts.Unavailable != 1 {
		t.Fatalf("expected deduped filter counts, got %#v", result.FilterCounts)
	}
}

func TestListRuntimeImportCandidatesMarksAlreadyImported(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	composePath := filepath.Join(tempDir, "compose.yaml")
	if err := os.WriteFile(composePath, []byte("services:\n  web:\n    image: nginx:latest\n"), 0o600); err != nil {
		t.Fatalf("write compose file: %v", err)
	}

	repo := &stubProjectRepository{
		aggregate: projectstore.ApplicationAggregate{
			Application: projectstore.Application{
				ApplicationRecordID: 42,
				ComposeProjectName:  "demo",
				WorkspacePath:       tempDir,
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
		t.Fatalf("expected 1 candidate, got %d", len(result.Items))
	}
	candidate := result.Items[0]
	if candidate.Status != importRuntimeCandidateStatusAlreadyImported || candidate.Importable {
		t.Fatalf("expected already imported candidate, got %#v", candidate)
	}
	if len(candidate.StatusReasonCodes) != 1 || candidate.StatusReasonCodes[0] != importRuntimeReasonAlreadyImported {
		t.Fatalf("unexpected status reason codes %#v", candidate.StatusReasonCodes)
	}
	if result.FilterCounts.Imported != 1 || result.FilterCounts.Ready != 0 || result.FilterCounts.Unavailable != 0 {
		t.Fatalf("unexpected filter counts %#v", result.FilterCounts)
	}
}

func TestInspectRuntimeCandidateRejectsAlreadyImportedCandidate(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	composePath := filepath.Join(tempDir, "compose.yaml")
	if err := os.WriteFile(composePath, []byte("services:\n  web:\n    image: nginx:latest\n"), 0o600); err != nil {
		t.Fatalf("write compose file: %v", err)
	}

	repo := &stubProjectRepository{
		aggregate: projectstore.ApplicationAggregate{
			Application: projectstore.Application{
				ApplicationRecordID: 7,
				ComposeProjectName:  "demo",
				WorkspacePath:       tempDir,
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

	_, err = service.InspectRuntimeCandidate(context.Background(), RuntimeImportInspectRequest{CandidateKey: "runtime_demo"})
	if !errors.Is(err, errProjectConflict) {
		t.Fatalf("expected project conflict, got %v", err)
	}
}

func TestRuntimeImportCandidateNormalizationTrimsLookupAndOutput(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	composePath := filepath.Join(tempDir, "compose.yaml")
	envPath := filepath.Join(tempDir, ".env")
	if err := os.WriteFile(composePath, []byte("services:\n  web:\n    image: nginx:latest\n"), 0o600); err != nil {
		t.Fatalf("write compose file: %v", err)
	}
	if err := os.WriteFile(envPath, []byte("FOO=bar\n"), 0o600); err != nil {
		t.Fatalf("write env file: %v", err)
	}

	service, err := NewService(&stubProjectRepository{}, WithRuntimeReader(stubRuntimeReader{
		candidates: []moduleapi.ContainerProjectRuntimeCandidate{
			{
				CandidateKey:           "  runtime_demo  ",
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
		candidateMembers: []moduleapi.ContainerProjectMember{
			{ContainerID: "c1", ContainerName: "demo-web-1", ServiceName: "web", CanonicalState: "running"},
		},
	}))
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	listResult, err := service.ListRuntimeImportCandidates(context.Background(), RuntimeImportCandidateListQuery{})
	if err != nil {
		t.Fatalf("list runtime import candidates: %v", err)
	}
	if len(listResult.Items) != 1 || listResult.Items[0].CandidateKey != "runtime_demo" {
		t.Fatalf("expected trimmed candidate key in list result, got %#v", listResult.Items)
	}

	inspectResult, err := service.InspectRuntimeCandidate(context.Background(), RuntimeImportInspectRequest{
		CandidateKey: "runtime_demo",
	})
	if err != nil {
		t.Fatalf("inspect runtime candidate: %v", err)
	}
	if inspectResult.CandidateKey != "runtime_demo" {
		t.Fatalf("expected trimmed candidate key in inspect result, got %#v", inspectResult)
	}
	if len(inspectResult.ComposeFiles) != 1 || inspectResult.ComposeFiles[0].AbsolutePath != composePath {
		t.Fatalf("unexpected compose files %#v", inspectResult.ComposeFiles)
	}
	if len(inspectResult.EnvFiles) != 1 || inspectResult.EnvFiles[0].AbsolutePath != envPath {
		t.Fatalf("unexpected env files %#v", inspectResult.EnvFiles)
	}
}

func assertRuntimeInspectResult(
	t *testing.T,
	result RuntimeImportInspectResult,
	expectedWorkspacePath string,
	expectedComposePath string,
	expectedEnvPath string,
) {
	t.Helper()
	if result.CandidateKey != "runtime_demo" {
		t.Fatalf("expected candidate key to round-trip, got %#v", result)
	}
	if result.ResolvedWorkspacePath != expectedWorkspacePath {
		t.Fatalf("expected working directory %q, got %q", expectedWorkspacePath, result.ResolvedWorkspacePath)
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
	if !slices.Contains(result.Warnings, "workspace_path_derived_from_config_files") {
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
		aggregate: projectstore.ApplicationAggregate{
			Application: projectstore.Application{ApplicationRecordID: 1, ComposeProjectName: "existing", WorkspacePath: "/srv/existing"},
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
				Warnings:               []string{"workspace_path_derived_from_config_files"},
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
		aggregate: projectstore.ApplicationAggregate{
			Application: projectstore.Application{
				ApplicationRecordID: 1,
				DisplayName:         "Orders",
				ComposeProjectName:  "orders",
				WorkspacePath:       projectDir,
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

func TestCreationMethodCatalogExposesSupportedMethods(t *testing.T) {
	t.Parallel()

	managedRoot := t.TempDir()
	service, err := NewService(&stubProjectRepository{}, WithSystemConfigResolver(stubSystemConfigResolver{value: managedRoot}))
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	result, err := service.CreationMethodCatalog(context.Background())
	if err != nil {
		t.Fatalf("creation method catalog: %v", err)
	}
	if len(result.Items) != 3 {
		t.Fatalf("expected 3 creation methods, got %d", len(result.Items))
	}
	if result.Items[0].Method != generated.ApplicationCreationMethodTypeBlank {
		t.Fatalf("expected blank creation method, got %q", result.Items[0].Method)
	}
	if result.Items[0].Availability != generated.ApplicationCreationMethodAvailabilityReady {
		t.Fatalf("expected ready blank method, got %q", result.Items[0].Availability)
	}
	if result.Items[1].Method != generated.ApplicationCreationMethodTypeTemplate {
		t.Fatalf("expected template creation method, got %q", result.Items[1].Method)
	}
	if result.Items[2].Method != generated.ApplicationCreationMethodTypeImport {
		t.Fatalf("expected import creation method, got %q", result.Items[2].Method)
	}
}

func TestCreationMethodCatalogBlocksBlankWhenManagedRootIsUnavailable(t *testing.T) {
	t.Parallel()

	service, err := NewService(&stubProjectRepository{})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	result, err := service.CreationMethodCatalog(context.Background())
	if err != nil {
		t.Fatalf("creation method catalog: %v", err)
	}
	if result.Items[0].Availability != generated.ApplicationCreationMethodAvailabilityBlocked {
		t.Fatalf("expected blocked blank method, got %q", result.Items[0].Availability)
	}
	if result.Items[0].BlockedReason == nil || *result.Items[0].BlockedReason != "managed_root_invalid" {
		t.Fatalf("expected managed-root-invalid reason, got %#v", result.Items[0].BlockedReason)
	}
}

func TestProjectListItemUsesFrontendActivityAuthorityForLocalProjects(t *testing.T) {
	t.Parallel()

	item := toProjectListItemWithManagedRoot(projectstore.ApplicationAggregate{
		Application: projectstore.Application{
			ApplicationRecordID:      1,
			DisplayName:              "Orders",
			ComposeProjectName:       "orders",
			ComposeProjectNameSource: "computed",
			SourceType:               "managed",
			OwnershipMode:            "managed-root-dedicated",
			WorkspacePath:            "/tmp/orders",
			DriftStatus:              "clean",
		},
	}, "", nil, nil)
	if item.ActivityAuthority != generated.ApplicationActivityAuthority("frontend-fanout") {
		t.Fatalf("expected frontend-fanout activity authority, got %q", item.ActivityAuthority)
	}
}

func TestProjectOverviewHealthyServiceCountIgnoresAttentionHealth(t *testing.T) {
	t.Parallel()

	item, healthy := toProjectOverviewServiceItem(
		projectcompose.ServiceProjection{ServiceName: "web"},
		moduleapi.ContainerProjectServiceResourceSummary{
			ServiceName:            "web",
			ContainerCount:         2,
			RunningCount:           1,
			StartingContainerCount: 1,
		},
	)
	if item.Status != generated.ApplicationOverviewServiceItemStatus("running") {
		t.Fatalf("expected running status, got %#v", item)
	}
	if item.Health != generated.ApplicationOverviewServiceItemHealth("attention") {
		t.Fatalf("expected attention health, got %#v", item)
	}
	if healthy {
		t.Fatalf("expected attention health not to count as healthy")
	}
}

func TestProjectLogsNilServiceReturnsRuntimeUnavailable(t *testing.T) {
	t.Parallel()

	var service *Service
	_, err := service.Logs(context.Background(), 1, LogQuery{})
	if !errors.Is(err, errProjectRuntimeUnavailable) {
		t.Fatalf("expected runtime unavailable, got %v", err)
	}
}

func TestToContainerProjectLogFollowQuerySuppressesReplay(t *testing.T) {
	t.Parallel()

	query := toContainerProjectLogFollowQuery(LogQuery{Tail: 200, Since: "1h", Stdout: true, Timestamps: true})
	if !query.FollowOnly || query.Tail != 200 || query.Since != "1h" || !query.Stdout || !query.Timestamps {
		t.Fatalf("unexpected follow query %#v", query)
	}
}

func TestImportDirectorySourcesIncludeManagedRootAndAllowlistedRoot(t *testing.T) {
	t.Parallel()

	managedRoot := t.TempDir()
	service, err := NewService(&stubProjectRepository{}, WithSystemConfigResolver(stubCompositeConfigResolver{
		values: map[string]string{
			"ops.application.root_directory":       `"` + managedRoot + `"`,
			"ops.application.import.allowed_roots": `[{"id":"srv","label":"Srv","path":"/srv"}]`,
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
	aggregate := projectstore.ApplicationAggregate{
		Application: projectstore.Application{
			ApplicationRecordID:      7,
			DisplayName:              "Orders",
			ComposeProjectName:       "orders",
			ComposeProjectNameSource: "computed",
			SourceType:               projectcontract.SourceTypeManaged.String(),
			OwnershipMode:            projectcontract.OwnershipModeManagedRootDedicated.String(),
			WorkspacePath:            filepath.Join(managedRoot, "team-a", "orders"),
			DriftStatus:              "clean",
		},
	}

	detail := toProjectDetailResponseWithManagedRoot(aggregate, managedRoot, nil, nil)
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

func TestDeleteManagedWorkspacePathRemovesOnlyTargetDirectory(t *testing.T) {
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

	if err := deleteManagedWorkspacePath(workingDirectory); err != nil {
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

func TestCleanupManagedCreateRemovesNestedWorkspaceDirectories(t *testing.T) {
	parent := t.TempDir()
	createdDir := filepath.Join(parent, "orders")
	if err := os.MkdirAll(filepath.Join(createdDir, "nested", "empty"), 0o750); err != nil {
		t.Fatalf("create nested workspace: %v", err)
	}
	if err := cleanupManagedCreate(createdDir, nil); err != nil {
		t.Fatalf("cleanup nested managed create: %v", err)
	}
	if _, err := os.Stat(createdDir); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected nested created directory removed, got %v", err)
	}
}

func TestProjectErrorMessageKeyUsesProjectCode(t *testing.T) {
	t.Parallel()

	if got := projectErrorMessageKey(projectcontract.ApplicationConflict.String()); got != projectcontract.ApplicationConflict.String() {
		t.Fatalf("expected project code as message key, got %q", got)
	}
	if got := projectErrorMessageKey(" "); got != "common.invalid_argument" {
		t.Fatalf("expected common invalid argument fallback, got %q", got)
	}
}

func TestComputeConflictsFlagsIndependentWorkspacePathAndCanonicalMatches(t *testing.T) {
	t.Parallel()

	repo := &stubProjectRepository{
		aggregate: projectstore.ApplicationAggregate{
			Application: projectstore.Application{
				ApplicationRecordID: 1,
				DisplayName:         "Orders",
				ComposeProjectName:  "orders",
				WorkspacePath:       "/srv/orders",
			},
		},
	}

	service := &Service{}
	conflicts, err := service.computeConflicts(context.Background(), repo, ImportValidationResult{
		WorkspacePath:      "/srv/orders",
		ComposeProjectName: "orders",
	})
	if err != nil {
		t.Fatalf("compute conflicts: %v", err)
	}
	expected := []string{"compose_project_name", "workspace_path"}
	if !slices.Equal(conflicts, expected) {
		t.Fatalf("expected conflicts %#v, got %#v", expected, conflicts)
	}
}

func TestToStoreFilesPersistsNormalizedBaselineHash(t *testing.T) {
	t.Parallel()

	files := toStoreFiles(
		[]projectcompose.FileProjection{
			{
				AbsolutePath: "/srv/orders/compose.yaml",
				DisplayPath:  "compose.yaml",
				Kind:         projectcontract.FileKindCompose.String(),
				Role:         projectcontract.FileRolePrimary.String(),
				OrderIndex:   0,
				Content:      []byte("services:\r\n  api:  \r\n    image: nginx:latest\r\n"),
				Hash:         "hash-compose",
				Exists:       true,
			},
		},
		nil,
	)

	if len(files) != 1 {
		t.Fatalf("expected one file, got %d", len(files))
	}
	if files[0].LastObservedHash != hashString("services:\n  api:\n    image: nginx:latest\n") {
		t.Fatalf("unexpected normalized baseline hash %q", files[0].LastObservedHash)
	}
}

func TestBrowseProjectFilesReturnsFileNotFoundForMissingDirectory(t *testing.T) {
	t.Parallel()

	repo := &stubProjectRepository{
		aggregate: projectstore.ApplicationAggregate{
			Application: projectstore.Application{
				ApplicationRecordID: 1,
				WorkspacePath:       t.TempDir(),
			},
		},
	}
	service, err := NewService(repo)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	_, err = service.browseProjectFiles(context.Background(), 1, workspaceFileBrowseQuery{Path: "missing"})
	if !errors.Is(err, errProjectFileNotFound) {
		t.Fatalf("expected file not found, got %v", err)
	}
}

func TestSaveProjectFileContentRejectsDirectoryPath(t *testing.T) {
	t.Parallel()

	workingDirectory := t.TempDir()
	if err := os.Mkdir(filepath.Join(workingDirectory, "config.yaml"), 0o750); err != nil {
		t.Fatalf("mkdir config directory: %v", err)
	}
	repo := &stubProjectRepository{
		aggregate: projectstore.ApplicationAggregate{
			Application: projectstore.Application{
				ApplicationRecordID: 1,
				WorkspacePath:       workingDirectory,
			},
		},
	}
	service, err := NewService(repo)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	_, err = service.saveProjectFileContent(context.Background(), 1, "config.yaml", workspaceFileSaveRequest{Content: "key: value\n"})
	if !errors.Is(err, errProjectInvalidArgument) {
		t.Fatalf("expected invalid argument for directory save, got %v", err)
	}
}

func TestWorkspaceHiddenDirectoriesFallsBackToDefaultOnInvalidConfig(t *testing.T) {
	t.Parallel()

	service, err := NewService(&stubProjectRepository{}, WithSystemConfigResolver(stubCompositeConfigResolver{
		values: map[string]string{
			projectcontract.ApplicationWorkspaceHiddenDirectoriesConfig.String(): `invalid-json`,
		},
	}))
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	hiddenDirectories, err := service.workspaceHiddenDirectories(context.Background())
	if err != nil {
		t.Fatalf("workspace hidden directories: %v", err)
	}
	if !slices.Contains(hiddenDirectories, "node_modules") || !slices.Contains(hiddenDirectories, ".git") {
		t.Fatalf("expected default hidden directories fallback, got %#v", hiddenDirectories)
	}
}

func TestWriteRouteErrorMapsProjectFileNotFoundTo404(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ginCtx, _ := gin.CreateTestContext(recorder)
	ginCtx.Request = httptest.NewRequest(http.MethodGet, "/api/projects/1/files?path=missing", nil)

	runtime := routeRuntime{ctx: &module.Context{}}
	runtime.writeRouteError(ginCtx, errProjectFileNotFound)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", recorder.Code)
	}
	var response httpx.ErrorResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode error response: %v", err)
	}
	if response.Code != projectcontract.ApplicationInvalidFileID.String() {
		t.Fatalf("expected file-not-found code %q, got %#v", projectcontract.ApplicationInvalidFileID.String(), response)
	}
}

func TestRestoreManagedDraftOnFailureRestoresOnlyWhenErrSet(t *testing.T) {
	t.Parallel()

	workingDirectory := t.TempDir()
	composePath := filepath.Join(workingDirectory, "compose.yaml")
	if err := os.WriteFile(composePath, []byte("services:\n  api:\n    image: nginx:latest\n"), 0o600); err != nil {
		t.Fatalf("write compose file: %v", err)
	}

	restoreItems, err := writeManagedDraft(workingDirectory, managedDraftProposal{
		ComposePath:    composePath,
		ComposeContent: "services:\n  api:\n    image: caddy:latest\n",
	})
	if err != nil {
		t.Fatalf("write managed draft: %v", err)
	}

	restoreManagedDraftOnFailure(workingDirectory, restoreItems, nil)
	// #nosec G304 -- composePath 由本测试的 t.TempDir() 构造，不受外部输入控制。
	content, err := os.ReadFile(composePath)
	if err != nil {
		t.Fatalf("read drafted compose file: %v", err)
	}
	if string(content) != "services:\n  api:\n    image: caddy:latest\n" {
		t.Fatalf("expected draft content to remain after nil error, got %q", string(content))
	}

	resultErr := errors.New("deploy failed")
	originalErr := resultErr
	restoreManagedDraftOnFailure(workingDirectory, restoreItems, &resultErr)
	// #nosec G304 -- composePath 由本测试的 t.TempDir() 构造，不受外部输入控制。
	restoredContent, err := os.ReadFile(composePath)
	if err != nil {
		t.Fatalf("read restored compose file: %v", err)
	}
	if string(restoredContent) != "services:\n  api:\n    image: nginx:latest\n" {
		t.Fatalf("expected original content restored on failure, got %q", string(restoredContent))
	}
	if !errors.Is(resultErr, originalErr) {
		t.Fatalf("expected original error to be preserved, got %v", resultErr)
	}
}

func TestWithComposeCommandTimeoutAddsFallbackDeadline(t *testing.T) {
	t.Parallel()

	start := time.Now()
	ctx, cancel := withComposeCommandTimeout(context.Background())
	defer cancel()

	deadline, ok := ctx.Deadline()
	if !ok {
		t.Fatalf("expected fallback deadline")
	}
	if deadline.Before(start.Add(projectComposeTimeout-time.Second)) || deadline.After(start.Add(projectComposeTimeout+time.Second)) {
		t.Fatalf("expected deadline near fallback timeout, got %v", deadline.Sub(start))
	}
}

func TestWithComposeCommandTimeoutPreservesExistingDeadline(t *testing.T) {
	t.Parallel()

	parent, parentCancel := context.WithTimeout(context.Background(), time.Second)
	defer parentCancel()
	parentDeadline, ok := parent.Deadline()
	if !ok {
		t.Fatalf("expected parent deadline")
	}

	ctx, cancel := withComposeCommandTimeout(parent)
	defer cancel()
	deadline, ok := ctx.Deadline()
	if !ok {
		t.Fatalf("expected derived deadline")
	}
	if !deadline.Equal(parentDeadline) {
		t.Fatalf("expected deadline %v, got %v", parentDeadline, deadline)
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
			"ops.application.import.allowed_roots": `[{"id":"apps","label":"Apps","path":"` + root + `"}]`,
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
			"ops.application.import.allowed_roots": `[{"id":"apps","label":"Apps","path":"` + root + `"}]`,
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

	imported, err := service.ImportByInspection(context.Background(), ImportExecuteRequest{
		InspectionID:           inspect.InspectionID,
		LifecycleConfiguration: lifecycleConfigPointer(importedLifecycleConfigFixture()),
	})
	if err != nil {
		t.Fatalf("import by inspection: %v", err)
	}
	if imported.Application.ComposeProjectName != "orders" {
		t.Fatalf("unexpected imported project: %#v", imported.Application)
	}
	assertImportedCreationPipelinePersisted(t, repo.importInput, projectDir)
	assertImportedLifecycleConfigPersisted(t, repo.importInput)
}

func assertImportedCreationPipelinePersisted(t *testing.T, input *projectstore.ImportApplicationInput, workingDirectory string) {
	t.Helper()
	if input == nil {
		t.Fatal("expected imported creation pipeline input")
	}
	if input.SourceType != projectcontract.SourceTypeImported.String() || input.OwnershipMode != projectcontract.OwnershipModeExternal.String() {
		t.Fatalf("expected imported source ownership metadata, got source=%q ownership=%q", input.SourceType, input.OwnershipMode)
	}
	if input.WorkspacePath != workingDirectory || input.DriftStatus != projectcontract.DriftStatusClean.String() || input.LastObservedConfigHash == "" || input.LastDriftCheckedAt == nil {
		t.Fatalf("expected shared creation aggregate fields, got %#v", input)
	}
	if len(input.Files) != 2 || input.Snapshot == nil || input.Snapshot.ConfigHash != input.LastObservedConfigHash || input.Snapshot.DeclaredServiceCount != 1 || input.Snapshot.RefreshedAt.IsZero() {
		t.Fatalf("expected imported workspace snapshot and files from shared pipeline, got files=%#v snapshot=%#v", input.Files, input.Snapshot)
	}
}

func importedLifecycleConfigFixture() LifecycleStandardConfig {
	config := defaultLifecycleStandardConfig()
	config.Profiles = []string{"production"}
	config.PullBeforeRedeploy = true
	config.BuildBeforeUp = true
	config.WaitAfterUp = true
	config.WaitTimeoutSeconds = 75
	config.AdditionalArgs = []string{"--ansi", "never"}
	return config
}

func assertImportedLifecycleConfigPersisted(t *testing.T, input *projectstore.ImportApplicationInput) {
	t.Helper()
	if input == nil {
		t.Fatal("expected persisted import input")
	}
	if input.LifecycleReviewStatus != projectcontract.LifecycleReviewStatusConfirmed.String() {
		t.Fatalf("expected imported lifecycle configuration to be confirmed, got %q", input.LifecycleReviewStatus)
	}
	config := input.LifecycleConfig
	if !config.DownBeforeRedeploy || !config.RemoveOrphans || !config.PullBeforeRedeploy || !config.BuildBeforeUp || !config.WaitAfterUp || config.WaitTimeoutSeconds != 75 || !slices.Equal(config.Profiles, []string{"production"}) || !slices.Equal(config.AdditionalArgs, []string{"--ansi", "never"}) {
		t.Fatalf("expected supplied lifecycle configuration to be persisted, got %#v", config)
	}
}

func TestImportByInspectionReturnsInspectionStaleOnFileHashMismatch(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	projectDir := filepath.Join(root, "orders")
	if err := os.MkdirAll(projectDir, 0o750); err != nil {
		t.Fatalf("mkdir project dir: %v", err)
	}
	composePath := filepath.Join(projectDir, "compose.yaml")
	if err := os.WriteFile(composePath, []byte("services:\n  api:\n    image: nginx:latest\n"), 0o600); err != nil {
		t.Fatalf("write compose file: %v", err)
	}

	service, err := NewService(&stubProjectRepository{}, WithSystemConfigResolver(stubCompositeConfigResolver{
		values: map[string]string{
			"ops.application.import.allowed_roots": `[{"id":"apps","label":"Apps","path":"` + root + `"}]`,
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

	if err := os.WriteFile(composePath, []byte("services:\n  api:\n    image: caddy:latest\n"), 0o600); err != nil {
		t.Fatalf("rewrite compose file: %v", err)
	}

	_, err = service.ImportByInspection(context.Background(), ImportExecuteRequest{
		InspectionID:           inspect.InspectionID,
		LifecycleConfiguration: lifecycleConfigPointer(defaultLifecycleStandardConfig()),
	})
	if !errors.Is(err, errProjectInspectionStale) {
		t.Fatalf("expected inspection stale error, got %v", err)
	}
}

type stubCompositeConfigResolver struct {
	values map[string]string
}

func lifecycleConfigPointer(config LifecycleStandardConfig) *LifecycleStandardConfig {
	return &config
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
