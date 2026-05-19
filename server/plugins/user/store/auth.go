package store

import (
	"context"
	"errors"
	"time"
)

var (
	// ErrRefreshSessionNotFound indicates the requested refresh session does not exist.
	ErrRefreshSessionNotFound = errors.New("refresh session not found")
)

// UserCredential is the minimal credential DTO used by the user plugin.
type UserCredential struct {
	UserID             uint64
	Username           string
	PasswordHash       *string
	MustChangePassword bool
	PasswordChangedAt  *time.Time
}

// SetPasswordHashInput describes the minimal password-hash update input.
type SetPasswordHashInput struct {
	UserID             uint64
	PasswordHash       string
	MustChangePassword bool
	ChangedAt          *time.Time
}

// ChangePasswordAndRevokeOtherRefreshSessionsInput describes the minimal
// password-change input that keeps the current session alive.
type ChangePasswordAndRevokeOtherRefreshSessionsInput struct {
	UserID             uint64
	PasswordHash       string
	MustChangePassword bool
	ChangedAt          time.Time
	CurrentTokenID     string
}

// EnsureUserCredentialInput describes the minimal ensured-credential input.
type EnsureUserCredentialInput struct {
	Username           string
	Display            string
	PasswordHash       string
	MustChangePassword bool
}

// RevokeOtherRefreshSessionsInput describes the minimal revoke-others input.
type RevokeOtherRefreshSessionsInput struct {
	UserID         uint64
	CurrentTokenID string
	RevokedAt      time.Time
}

// RefreshSession is the stable refresh-session DTO used by the user plugin.
type RefreshSession struct {
	ID                uint64
	UserID            uint64
	TokenID           string
	ExpiresAt         time.Time
	RevokedAt         *time.Time
	ReplacedByTokenID *string
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// ListActiveRefreshSessionsByUserIDInput describes the minimal active-session query.
type ListActiveRefreshSessionsByUserIDInput struct {
	UserID uint64
	Now    time.Time
}

// CreateRefreshSessionInput describes the minimal refresh-session creation input.
type CreateRefreshSessionInput struct {
	UserID    uint64
	TokenID   string
	ExpiresAt time.Time
}

// RevokeRefreshSessionInput describes the minimal single-session revoke input.
type RevokeRefreshSessionInput struct {
	TokenID           string
	RevokedAt         time.Time
	ReplacedByTokenID *string
}

// RevokeRefreshSessionsByUserIDInput describes the minimal bulk revoke input.
type RevokeRefreshSessionsByUserIDInput struct {
	UserID    uint64
	RevokedAt time.Time
}

// RevokeRefreshSessionByUserIDInput describes the minimal targeted revoke input.
type RevokeRefreshSessionByUserIDInput struct {
	UserID    uint64
	TokenID   string
	RevokedAt time.Time
}

// RotateRefreshSessionInput describes one refresh-session rotation operation.
type RotateRefreshSessionInput struct {
	CurrentTokenID string
	NewTokenID     string
	Now            time.Time
	RevokedAt      time.Time
	NewExpiresAt   time.Time
}

// PasswordChangeRepository exposes the atomic password-change write contract.
type PasswordChangeRepository interface {
	ChangePasswordAndRevokeOtherRefreshSessions(
		ctx context.Context,
		input ChangePasswordAndRevokeOtherRefreshSessionsInput,
	) error
}

// AuthRepository exposes the user plugin's private auth/session persistence contract.
type AuthRepository interface {
	GetUserCredentialByUsername(ctx context.Context, username string) (UserCredential, error)
	SetPasswordHash(ctx context.Context, input SetPasswordHashInput) error
	EnsureUserCredential(ctx context.Context, input EnsureUserCredentialInput) (UserCredential, error)
	CreateRefreshSession(ctx context.Context, input CreateRefreshSessionInput) (RefreshSession, error)
	GetRefreshSessionByTokenID(ctx context.Context, tokenID string) (RefreshSession, error)
	RevokeRefreshSession(ctx context.Context, input RevokeRefreshSessionInput) error
	RevokeRefreshSessionsByUserID(ctx context.Context, input RevokeRefreshSessionsByUserIDInput) error
	RevokeOtherRefreshSessionsByUserID(ctx context.Context, input RevokeOtherRefreshSessionsInput) error
	RevokeRefreshSessionByUserID(ctx context.Context, input RevokeRefreshSessionByUserIDInput) error
	ListActiveRefreshSessionsByUserID(ctx context.Context, input ListActiveRefreshSessionsByUserIDInput) ([]RefreshSession, error)
	RotateRefreshSession(ctx context.Context, input RotateRefreshSessionInput) (RefreshSession, error)
}
