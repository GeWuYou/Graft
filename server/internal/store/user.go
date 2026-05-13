package store

import (
	"context"
	"errors"
	"time"
)

// ErrUserNotFound reports that the requested user does not exist.
var ErrUserNotFound = errors.New("user not found")

// User is the persistence-facing record returned by the user repository.
type User struct {
	ID        uint64
	Username  string
	Display   string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// UserRepository exposes the minimal user persistence operations needed by the
// current MVP plugin surface.
type UserRepository interface {
	// GetByID returns one user record or ErrUserNotFound.
	GetByID(ctx context.Context, id uint64) (User, error)
}
