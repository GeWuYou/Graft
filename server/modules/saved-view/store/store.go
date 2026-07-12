// Package store persists generic saved-view state without interpreting consumer filters.
package store

import (
	"context"

	"graft/server/internal/moduleapi"
)

// Repository persists generic saved views.
type Repository interface {
	List(context.Context, uint64, string) ([]moduleapi.SavedView, error)
	Create(context.Context, moduleapi.SavedViewCreateInput) (moduleapi.SavedView, error)
	Update(context.Context, moduleapi.SavedViewUpdateInput) (moduleapi.SavedView, error)
	Delete(context.Context, uint64, string, uint64) error
}
