// Package store defines system-config module persistence contracts.
package store

import (
	"context"
	"encoding/json"
	"errors"
	"time"
)

// ErrOverrideNotFound indicates that no user override exists for the key.
var ErrOverrideNotFound = errors.New("system config override not found")

// Override stores user-provided JSON for one config key.
type Override struct {
	Key       string
	Value     json.RawMessage
	CreatedAt time.Time
	CreatedBy *uint64
	UpdatedAt time.Time
	UpdatedBy *uint64
}

// Repository persists user overrides only.
type Repository interface {
	GetOverride(ctx context.Context, key string) (Override, error)
	SetOverride(ctx context.Context, key string, value json.RawMessage, userID *uint64) (Override, error)
	DeleteOverride(ctx context.Context, key string) error
}
