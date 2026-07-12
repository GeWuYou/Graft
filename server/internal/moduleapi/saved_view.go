// Package moduleapi exposes stable cross-module saved-view capabilities.
package moduleapi

import (
	"context"
	"encoding/json"
	"errors"
	"time"
)

var (
	// ErrSavedViewNotFound means the requested live view is not owned by the caller.
	ErrSavedViewNotFound = errors.New("saved view not found")
	// ErrSavedViewConflict means a live view name already exists for an owner and surface.
	ErrSavedViewConflict = errors.New("saved view conflict")
	// ErrSavedViewInvalidInput means generic saved-view state is structurally invalid.
	ErrSavedViewInvalidInput = errors.New("saved view invalid input")
)

// SavedView is a consumer-neutral persisted list-page view. QueryState remains opaque to this service.
type SavedView struct {
	ID             uint64
	OwnerUserID    uint64
	SurfaceKey     string
	Name           string
	QueryState     json.RawMessage
	PageSize       int
	VisibleColumns []string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// SavedViewCreateInput creates a view after the consumer validates its surface and query state.
type SavedViewCreateInput struct {
	OwnerUserID    uint64
	SurfaceKey     string
	Name           string
	QueryState     json.RawMessage
	PageSize       int
	VisibleColumns []string
}

// SavedViewUpdateInput updates an existing view owned by OwnerUserID.
type SavedViewUpdateInput struct {
	ID             uint64
	OwnerUserID    uint64
	SurfaceKey     string
	Name           string
	QueryState     json.RawMessage
	PageSize       int
	VisibleColumns []string
}

// SavedViewService is the generic persistence boundary. Consumers own authorization and payload semantics.
type SavedViewService interface {
	List(ctx context.Context, ownerUserID uint64, surfaceKey string) ([]SavedView, error)
	Create(ctx context.Context, input SavedViewCreateInput) (SavedView, error)
	Update(ctx context.Context, input SavedViewUpdateInput) (SavedView, error)
	Delete(ctx context.Context, ownerUserID uint64, surfaceKey string, id uint64) error
}
