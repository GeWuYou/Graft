package store

import (
	"context"
	"errors"
	"time"
)

var (
	// ErrUserNotFound indicates the requested user does not exist.
	ErrUserNotFound = errors.New("user not found")

	// ErrInvalidID indicates the provided stable identifier is invalid.
	ErrInvalidID = errors.New("invalid id")
)

// User is the stable user DTO visible inside the user plugin.
type User struct {
	ID        uint64
	Username  string
	Display   string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// UserRepository exposes the user plugin's private user read contract.
type UserRepository interface {
	GetByID(ctx context.Context, id uint64) (User, error)
	List(ctx context.Context) ([]User, error)
}
