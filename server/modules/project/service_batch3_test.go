package project

import (
	"context"
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

func (s *stubProjectRepository) ImportProject(context.Context, projectstore.ImportProjectInput) (projectstore.ProjectAggregate, error) {
	return projectstore.ProjectAggregate{}, errors.New("not implemented")
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
