package savedview

import (
	"context"

	"graft/server/internal/moduleapi"
	"graft/server/modules/saved-view/store"
)

// Service validates generic view structure and delegates opaque consumer state to its store.
type Service struct{ repository store.Repository }

// NewService constructs a generic saved-view service.
func NewService(repository store.Repository) *Service { return &Service{repository: repository} }

// List returns views belonging to one owner and one consumer surface.
func (s *Service) List(ctx context.Context, ownerUserID uint64, surfaceKey string) ([]moduleapi.SavedView, error) {
	if s == nil || s.repository == nil {
		return nil, moduleapi.ErrSavedViewInvalidInput
	}
	return s.repository.List(ctx, ownerUserID, surfaceKey)
}

// Create persists one consumer-validated view.
func (s *Service) Create(ctx context.Context, input moduleapi.SavedViewCreateInput) (moduleapi.SavedView, error) {
	if s == nil || s.repository == nil {
		return moduleapi.SavedView{}, moduleapi.ErrSavedViewInvalidInput
	}
	return s.repository.Create(ctx, input)
}

// Update replaces one owned saved view.
func (s *Service) Update(ctx context.Context, input moduleapi.SavedViewUpdateInput) (moduleapi.SavedView, error) {
	if s == nil || s.repository == nil {
		return moduleapi.SavedView{}, moduleapi.ErrSavedViewInvalidInput
	}
	return s.repository.Update(ctx, input)
}

// Delete soft-deletes one owned saved view.
func (s *Service) Delete(ctx context.Context, ownerUserID uint64, surfaceKey string, id uint64) error {
	if s == nil || s.repository == nil {
		return moduleapi.ErrSavedViewInvalidInput
	}
	return s.repository.Delete(ctx, ownerUserID, surfaceKey, id)
}
