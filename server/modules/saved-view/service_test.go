package savedview

import (
	"context"
	"testing"

	"graft/server/internal/moduleapi"
)

type serviceTestRepository struct {
	updateCalls int
	deleteCalls int
}

func (*serviceTestRepository) List(context.Context, uint64, string) ([]moduleapi.SavedView, error) {
	return nil, nil
}

func (*serviceTestRepository) Create(context.Context, moduleapi.SavedViewCreateInput) (moduleapi.SavedView, error) {
	return moduleapi.SavedView{}, nil
}

func (r *serviceTestRepository) Update(_ context.Context, input moduleapi.SavedViewUpdateInput) (moduleapi.SavedView, error) {
	r.updateCalls++
	return moduleapi.SavedView{ID: input.ID}, nil
}

func (r *serviceTestRepository) Delete(context.Context, uint64, string, uint64) error {
	r.deleteCalls++
	return nil
}

func TestServiceRejectsInvalidUpdateAndDeleteBeforeDelegation(t *testing.T) {
	t.Parallel()
	repository := &serviceTestRepository{}
	service := NewService(repository)
	ctx := context.Background()

	if _, err := service.Update(ctx, moduleapi.SavedViewUpdateInput{OwnerUserID: 1, SurfaceKey: "project.list"}); err != moduleapi.ErrSavedViewInvalidInput {
		t.Fatalf("expected invalid update input, got %v", err)
	}
	if err := service.Delete(ctx, 1, "project.list", 0); err != moduleapi.ErrSavedViewInvalidInput {
		t.Fatalf("expected invalid delete input, got %v", err)
	}
	if repository.updateCalls != 0 || repository.deleteCalls != 0 {
		t.Fatalf("invalid input delegated to repository: updates=%d deletes=%d", repository.updateCalls, repository.deleteCalls)
	}
	if _, err := service.Update(ctx, moduleapi.SavedViewUpdateInput{ID: 1, OwnerUserID: 1, SurfaceKey: "project.list"}); err != nil {
		t.Fatalf("valid update: %v", err)
	}
	if err := service.Delete(ctx, 1, "project.list", 1); err != nil {
		t.Fatalf("valid delete: %v", err)
	}
	if repository.updateCalls != 1 || repository.deleteCalls != 1 {
		t.Fatalf("valid input not delegated: updates=%d deletes=%d", repository.updateCalls, repository.deleteCalls)
	}
}
