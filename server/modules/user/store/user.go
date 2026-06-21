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

	// ErrUsernameConflict indicates the requested username already exists.
	ErrUsernameConflict = errors.New("username already exists")
)

// User is the stable user DTO visible inside the user module.
type User struct {
	ID        uint64
	Username  string
	Display   string
	Status    string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// CreateUserInput describes the minimal user creation input.
type CreateUserInput struct {
	Username           string
	Display            string
	Status             string
	PasswordHash       string
	MustChangePassword bool
	ActorID            uint64
}

// UpdateUserInput describes the minimal user profile update input.
type UpdateUserInput struct {
	ID       uint64
	Username string
	Display  string
	ActorID  uint64
}

// SetUserStatusInput describes the minimal status-change input.
type SetUserStatusInput struct {
	ID      uint64
	Status  string
	ActorID uint64
}

// DeleteUserInput describes the minimal soft-delete input.
type DeleteUserInput struct {
	ID        uint64
	DeletedAt time.Time
	ActorID   uint64
}

// UserRepository exposes the user module's private user read contract.
type UserRepository interface {
	GetByID(ctx context.Context, id uint64) (User, error)
	List(ctx context.Context) ([]User, error)
	Count(ctx context.Context) (int, error)
	Create(ctx context.Context, input CreateUserInput) (User, error)
	Update(ctx context.Context, input UpdateUserInput) (User, error)
	SetStatus(ctx context.Context, input SetUserStatusInput) (User, error)
	Delete(ctx context.Context, input DeleteUserInput) error
}
