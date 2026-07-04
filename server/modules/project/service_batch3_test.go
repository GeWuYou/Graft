package project

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"testing"
	"time"

	generated "graft/server/internal/contract/openapi/generated"
	"graft/server/internal/eventbus"
	"graft/server/internal/httpx"
	"graft/server/internal/moduleapi"
	projectcontract "graft/server/modules/project/contract"
	projectstore "graft/server/modules/project/store"
)

type stubProjectRepository struct {
	aggregate        projectstore.ProjectAggregate
	unregisterCalled bool
	unregisterInput  *projectstore.UnregisterProjectInput
	unregisterErr    error
	importInput      *projectstore.ImportProjectInput
	getCalls         int
}

func (s *stubProjectRepository) List(context.Context, projectstore.ListQuery) (projectstore.ListResult, error) {
	return projectstore.ListResult{Items: []projectstore.ProjectAggregate{s.aggregate}, Total: 1}, nil
}

func (s *stubProjectRepository) Get(context.Context, uint64) (projectstore.ProjectAggregate, error) {
	s.getCalls++
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

func (s *stubProjectRepository) UnregisterProject(_ context.Context, input projectstore.UnregisterProjectInput) error {
	s.unregisterCalled = true
	recorded := input
	s.unregisterInput = &recorded
	return s.unregisterErr
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

func authenticatedProjectActionContext() context.Context {
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
	result, err := service.Destroy(authenticatedProjectActionContext(), 1, DestroyRequest{
		DeleteWorkingDirectory:      true,
		ConfirmCanonicalProjectName: "demo",
	})
	if !errors.Is(err, errProjectDestroyBlocked) {
		t.Fatalf("expected destroy blocked, got %v", err)
	}
	if result.Result != generated.ProjectActionResponseResultProjectActionResultBlocked {
		t.Fatalf("expected blocked result, got %s", result.Result)
	}
	if repo.unregisterCalled {
		t.Fatalf("unregister should not be called when destroy is blocked")
	}
}

func TestUnregisterUsesRequestActorAndPublishesAudit(t *testing.T) {
	t.Parallel()

	repo := &stubProjectRepository{
		aggregate: projectstore.ProjectAggregate{
			Project: projectstore.Project{
				ID:                   1,
				CanonicalProjectName: "demo",
				HostScope:            projectcontract.HostScopeLocal.String(),
				WorkingDirectory:     t.TempDir(),
				LastRefreshStatus:    projectcontract.RefreshStatusSuccess.String(),
			},
		},
	}
	service, err := NewService(repo)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	auditBus := &capturedAuditBus{}
	service.SetAuditPublisher(auditBus, nil, moduleID)

	result, err := service.Unregister(authenticatedProjectActionContext(), 1, nil)
	if err != nil {
		t.Fatalf("unregister: %v", err)
	}
	if result.Result != generated.ProjectActionResponseResultProjectActionResultCompleted {
		t.Fatalf("expected completed result, got %#v", result)
	}
	if repo.unregisterInput == nil || repo.unregisterInput.ActorID == nil || *repo.unregisterInput.ActorID != 7 {
		t.Fatalf("expected unregister actor id 7, got %#v", repo.unregisterInput)
	}
	events := auditBus.snapshot()
	if len(events) != 1 {
		t.Fatalf("expected one audit event, got %d", len(events))
	}
	if events[0].Action != projectcontract.ProjectAuditActionUnregister.String() {
		t.Fatalf("expected unregister audit action, got %#v", events[0])
	}
	if events[0].Operator == nil || events[0].Operator.ID != 7 {
		t.Fatalf("expected operator id 7, got %#v", events[0].Operator)
	}
}

func TestUnregisterFailsClosedWithoutRequestActor(t *testing.T) {
	t.Parallel()

	repo := &stubProjectRepository{
		aggregate: projectstore.ProjectAggregate{
			Project: projectstore.Project{
				ID:                   1,
				CanonicalProjectName: "demo",
				HostScope:            projectcontract.HostScopeLocal.String(),
				WorkingDirectory:     t.TempDir(),
				LastRefreshStatus:    projectcontract.RefreshStatusSuccess.String(),
			},
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
	if result.Result != generated.ProjectActionResponseResultProjectActionResultBlocked {
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
		Action:     generated.ProjectBatchActionRequestActionStart,
		ProjectIDs: []uint64{1, 2},
		ActorID:    &actorID,
	})
	if err != nil {
		t.Fatalf("batch action should fail closed through result semantics, got %v", err)
	}
	if result.BlockedCount != 2 || len(result.Items) != 2 {
		t.Fatalf("expected two blocked items, got %#v", result)
	}
	for _, item := range result.Items {
		if item.Result != generated.ProjectActionResponseResultProjectActionResultBlocked {
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
		aggregate: projectstore.ProjectAggregate{
			Project: projectstore.Project{
				ID:                   1,
				CanonicalProjectName: "demo",
				HostScope:            projectcontract.HostScopeLocal.String(),
				WorkingDirectory:     filepath.Join(t.TempDir(), "missing"),
				LastRefreshStatus:    projectcontract.RefreshStatusSuccess.String(),
			},
		},
	}
	service, err := NewService(repo)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	result, err := service.BatchAction(authenticatedProjectActionContext(), BatchActionRequest{
		Action:     generated.ProjectBatchActionRequestActionStart,
		ProjectIDs: []uint64{1, 1},
	})
	if err != nil {
		t.Fatalf("batch action: %v", err)
	}
	if result.BlockedCount != 2 || len(result.Items) != 2 {
		t.Fatalf("expected two blocked items, got %#v", result)
	}
	for _, item := range result.Items {
		if item.Result != generated.ProjectActionResponseResultProjectActionResultBlocked {
			t.Fatalf("expected blocked item, got %#v", item)
		}
	}
}

func TestBatchDestroyRequiresExplicitConfirmation(t *testing.T) {
	t.Parallel()

	repo := &stubProjectRepository{
		aggregate: projectstore.ProjectAggregate{
			Project: projectstore.Project{
				ID:                   1,
				CanonicalProjectName: "demo",
				HostScope:            projectcontract.HostScopeLocal.String(),
				WorkingDirectory:     t.TempDir(),
				OwnershipMode:        projectcontract.OwnershipModeExternal.String(),
				LastRefreshStatus:    projectcontract.RefreshStatusSuccess.String(),
			},
		},
	}
	service, err := NewService(repo)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	result, err := service.BatchAction(authenticatedProjectActionContext(), BatchActionRequest{
		Action:     generated.ProjectBatchActionRequestActionDestroy,
		ProjectIDs: []uint64{1},
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

func TestBatchDestroyReturnsBlockedItemOnComposeFailure(t *testing.T) {
	t.Parallel()

	repo := &stubProjectRepository{
		aggregate: projectstore.ProjectAggregate{
			Project: projectstore.Project{
				ID:                   1,
				CanonicalProjectName: "demo",
				HostScope:            projectcontract.HostScopeLocal.String(),
				WorkingDirectory:     filepath.Join(t.TempDir(), "missing"),
				OwnershipMode:        projectcontract.OwnershipModeExternal.String(),
				LastRefreshStatus:    projectcontract.RefreshStatusSuccess.String(),
			},
		},
	}
	service, err := NewService(repo)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	confirmName := "demo"

	result, err := service.BatchAction(authenticatedProjectActionContext(), BatchActionRequest{
		Action:                      generated.ProjectBatchActionRequestActionDestroy,
		ProjectIDs:                  []uint64{1},
		ConfirmCanonicalProjectName: &confirmName,
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

func TestBatchDestroyReturnsBlockedItemOnWorkingDirectoryDeleteFailure(t *testing.T) {
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
		aggregate: projectstore.ProjectAggregate{
			Project: projectstore.Project{
				ID:                   1,
				CanonicalProjectName: "demo",
				HostScope:            projectcontract.HostScopeLocal.String(),
				WorkingDirectory:     workingDirectory,
				OwnershipMode:        projectcontract.OwnershipModeManagedRootDedicated.String(),
				LastRefreshStatus:    projectcontract.RefreshStatusSuccess.String(),
			},
		},
	}
	service, err := NewService(repo)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	confirmName := "demo"

	result, err := service.BatchAction(authenticatedProjectActionContext(), BatchActionRequest{
		Action:                      generated.ProjectBatchActionRequestActionDestroy,
		ProjectIDs:                  []uint64{1},
		DeleteWorkingDirectory:      true,
		ConfirmCanonicalProjectName: &confirmName,
	})
	if err != nil {
		t.Fatalf("batch destroy: %v", err)
	}
	if result.BlockedCount != 1 || len(result.Items) != 1 {
		t.Fatalf("expected one blocked destroy item, got %#v", result)
	}
	if result.Items[0].Result != generated.ProjectActionResponseResultProjectActionResultBlocked {
		t.Fatalf("expected blocked destroy result, got %#v", result.Items[0])
	}
	if !slices.ContainsFunc(result.Items[0].GuardResults, func(guard GuardResult) bool {
		return guard.Code == "compose_down_completed"
	}) {
		t.Fatalf("expected compose-down guard after partial destroy, got %#v", result.Items[0].GuardResults)
	}
	if !slices.ContainsFunc(result.Items[0].GuardResults, func(guard GuardResult) bool {
		return guard.Code == "working_directory_delete_failed" && guard.Detail != nil && *guard.Detail == "filesystem_error"
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
		aggregate: projectstore.ProjectAggregate{
			Project: projectstore.Project{
				ID:                   1,
				CanonicalProjectName: "demo",
				HostScope:            projectcontract.HostScopeLocal.String(),
				WorkingDirectory:     workingDirectory,
				OwnershipMode:        projectcontract.OwnershipModeExternal.String(),
				LastRefreshStatus:    projectcontract.RefreshStatusSuccess.String(),
			},
		},
		unregisterErr: projectstore.ErrProjectConflict,
	}
	service, err := NewService(repo)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	confirmName := "demo"

	result, err := service.BatchAction(authenticatedProjectActionContext(), BatchActionRequest{
		Action:                      generated.ProjectBatchActionRequestActionDestroy,
		ProjectIDs:                  []uint64{1},
		AutoUnregister:              true,
		ConfirmCanonicalProjectName: &confirmName,
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
	if result.Items[0].Result != generated.ProjectActionResponseResultProjectActionResultBlocked {
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
		aggregate: projectstore.ProjectAggregate{
			Project: projectstore.Project{
				ID:                   1,
				CanonicalProjectName: "demo",
				HostScope:            projectcontract.HostScopeLocal.String(),
				WorkingDirectory:     filepath.Join(t.TempDir(), "missing"),
				LastRefreshStatus:    projectcontract.RefreshStatusSuccess.String(),
			},
		},
	}
	service, err := NewService(repo)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	result, err := service.BatchAction(authenticatedProjectActionContext(), BatchActionRequest{
		Action:     generated.ProjectBatchActionRequestActionRedeploy,
		ProjectIDs: []uint64{1},
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
		aggregate: projectstore.ProjectAggregate{
			Project: projectstore.Project{
				ID:                   1,
				CanonicalProjectName: "demo",
				HostScope:            projectcontract.HostScopeLocal.String(),
				WorkingDirectory:     filepath.Join(t.TempDir(), "missing"),
				LastRefreshStatus:    projectcontract.RefreshStatusSuccess.String(),
			},
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
		result, batchErr := service.BatchAction(authenticatedProjectActionContext(), BatchActionRequest{
			Action:     generated.ProjectBatchActionRequestActionStart,
			ProjectIDs: []uint64{1},
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
	if events[0].Action != projectcontract.ProjectAuditActionBatchStart.String() {
		t.Fatalf("expected batch-start audit action, got %#v", events[0])
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
		aggregate: projectstore.ProjectAggregate{
			Project: projectstore.Project{
				ID:                   42,
				CanonicalProjectName: "demo",
				WorkingDirectory:     tempDir,
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
		aggregate: projectstore.ProjectAggregate{
			Project: projectstore.Project{
				ID:                   7,
				CanonicalProjectName: "demo",
				WorkingDirectory:     tempDir,
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
	}, "", nil, nil)
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
	}, nil, nil)
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

func TestComputeConflictsFlagsIndependentWorkingDirectoryAndCanonicalMatches(t *testing.T) {
	t.Parallel()

	repo := &stubProjectRepository{
		aggregate: projectstore.ProjectAggregate{
			Project: projectstore.Project{
				ID:                   1,
				DisplayName:          "Orders",
				CanonicalProjectName: "orders",
				WorkingDirectory:     "/srv/orders",
			},
		},
	}

	service := &Service{}
	conflicts, err := service.computeConflicts(context.Background(), repo, ImportValidationResult{
		WorkingDirectory:     "/srv/orders",
		CanonicalProjectName: "orders",
	})
	if err != nil {
		t.Fatalf("compute conflicts: %v", err)
	}
	expected := []string{"canonical_project_name", "working_directory"}
	if !slices.Equal(conflicts, expected) {
		t.Fatalf("expected conflicts %#v, got %#v", expected, conflicts)
	}
}

func TestBuildConfigurationDiffFileKeepsProposedContentAsNormalizedText(t *testing.T) {
	t.Parallel()

	file := buildConfigurationDiffFile(
		projectcontract.FileKindCompose.String(),
		"/srv/orders/compose.yaml",
		"services:\n  api:\n    image: nginx:latest\n",
		"services:\n  api:\n    image: caddy:latest\n",
	)
	if !file.Changed {
		t.Fatalf("expected changed diff file, got %#v", file)
	}
	want := "services:\n  api:\n    image: caddy:latest\n"
	if file.ProposedContent != want {
		t.Fatalf("expected proposed content %q, got %q", want, file.ProposedContent)
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
	// #nosec G304 -- composePath is created from t.TempDir() within this test.
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
	// #nosec G304 -- composePath is created from t.TempDir() within this test.
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

	if err := os.WriteFile(composePath, []byte("services:\n  api:\n    image: caddy:latest\n"), 0o600); err != nil {
		t.Fatalf("rewrite compose file: %v", err)
	}

	_, err = service.ImportByInspection(context.Background(), ImportExecuteRequest{InspectionID: inspect.InspectionID})
	if !errors.Is(err, errProjectInspectionStale) {
		t.Fatalf("expected inspection stale error, got %v", err)
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
