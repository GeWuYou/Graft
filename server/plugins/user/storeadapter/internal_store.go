// Package storeadapter keeps Phase 2 compatibility adapters between shared store
// seams and user-plugin-owned persistence contracts.
package storeadapter

import (
	"context"
	"errors"

	internalstore "graft/server/internal/store"
	userstore "graft/server/plugins/user/store"
)

// NewUserRepositoryAdapter adapts the shared user repository to the user plugin contract.
func NewUserRepositoryAdapter(repo internalstore.UserRepository) userstore.UserRepository {
	return userRepositoryAdapter{delegate: repo}
}

// NewAuthRepositoryAdapter adapts the shared auth repository to the user plugin contract.
func NewAuthRepositoryAdapter(repo internalstore.AuthRepository) userstore.AuthRepository {
	return authRepositoryAdapter{delegate: repo}
}

// NewPasswordChangeRepositoryAdapter adapts the shared optional password-change capability.
func NewPasswordChangeRepositoryAdapter(repo internalstore.PasswordChangeRepository) userstore.PasswordChangeRepository {
	return passwordChangeRepositoryAdapter{delegate: repo}
}

type userRepositoryAdapter struct {
	delegate internalstore.UserRepository
}

func (a userRepositoryAdapter) GetByID(ctx context.Context, id uint64) (userstore.User, error) {
	record, err := a.delegate.GetByID(ctx, id)
	return toUser(record), mapError(err)
}

func (a userRepositoryAdapter) List(ctx context.Context) ([]userstore.User, error) {
	records, err := a.delegate.List(ctx)
	return toUsers(records), mapError(err)
}

type authRepositoryAdapter struct {
	delegate internalstore.AuthRepository
}

func (a authRepositoryAdapter) GetUserCredentialByUsername(ctx context.Context, username string) (userstore.UserCredential, error) {
	record, err := a.delegate.GetUserCredentialByUsername(ctx, username)
	return toUserCredential(record), mapError(err)
}

func (a authRepositoryAdapter) SetPasswordHash(ctx context.Context, input userstore.SetPasswordHashInput) error {
	return mapError(a.delegate.SetPasswordHash(ctx, internalstore.SetPasswordHashInput{
		UserID:             input.UserID,
		PasswordHash:       input.PasswordHash,
		MustChangePassword: input.MustChangePassword,
		ChangedAt:          input.ChangedAt,
	}))
}

func (a authRepositoryAdapter) EnsureUserCredential(ctx context.Context, input userstore.EnsureUserCredentialInput) (userstore.UserCredential, error) {
	record, err := a.delegate.EnsureUserCredential(ctx, internalstore.EnsureUserCredentialInput{
		Username:           input.Username,
		Display:            input.Display,
		PasswordHash:       input.PasswordHash,
		MustChangePassword: input.MustChangePassword,
	})
	return toUserCredential(record), mapError(err)
}

func (a authRepositoryAdapter) CreateRefreshSession(ctx context.Context, input userstore.CreateRefreshSessionInput) (userstore.RefreshSession, error) {
	record, err := a.delegate.CreateRefreshSession(ctx, internalstore.CreateRefreshSessionInput{
		UserID:    input.UserID,
		TokenID:   input.TokenID,
		ExpiresAt: input.ExpiresAt,
	})
	return toRefreshSession(record), mapError(err)
}

func (a authRepositoryAdapter) GetRefreshSessionByTokenID(ctx context.Context, tokenID string) (userstore.RefreshSession, error) {
	record, err := a.delegate.GetRefreshSessionByTokenID(ctx, tokenID)
	return toRefreshSession(record), mapError(err)
}

func (a authRepositoryAdapter) RevokeRefreshSession(ctx context.Context, input userstore.RevokeRefreshSessionInput) error {
	return mapError(a.delegate.RevokeRefreshSession(ctx, internalstore.RevokeRefreshSessionInput{
		TokenID:           input.TokenID,
		RevokedAt:         input.RevokedAt,
		ReplacedByTokenID: input.ReplacedByTokenID,
	}))
}

func (a authRepositoryAdapter) RevokeRefreshSessionsByUserID(ctx context.Context, input userstore.RevokeRefreshSessionsByUserIDInput) error {
	return mapError(a.delegate.RevokeRefreshSessionsByUserID(ctx, internalstore.RevokeRefreshSessionsByUserIDInput{
		UserID:    input.UserID,
		RevokedAt: input.RevokedAt,
	}))
}

func (a authRepositoryAdapter) RevokeOtherRefreshSessionsByUserID(ctx context.Context, input userstore.RevokeOtherRefreshSessionsInput) error {
	return mapError(a.delegate.RevokeOtherRefreshSessionsByUserID(ctx, internalstore.RevokeOtherRefreshSessionsInput{
		UserID:         input.UserID,
		CurrentTokenID: input.CurrentTokenID,
		RevokedAt:      input.RevokedAt,
	}))
}

func (a authRepositoryAdapter) RevokeRefreshSessionByUserID(ctx context.Context, input userstore.RevokeRefreshSessionByUserIDInput) error {
	return mapError(a.delegate.RevokeRefreshSessionByUserID(ctx, internalstore.RevokeRefreshSessionByUserIDInput{
		UserID:    input.UserID,
		TokenID:   input.TokenID,
		RevokedAt: input.RevokedAt,
	}))
}

func (a authRepositoryAdapter) ListActiveRefreshSessionsByUserID(ctx context.Context, input userstore.ListActiveRefreshSessionsByUserIDInput) ([]userstore.RefreshSession, error) {
	records, err := a.delegate.ListActiveRefreshSessionsByUserID(ctx, internalstore.ListActiveRefreshSessionsByUserIDInput{
		UserID: input.UserID,
		Now:    input.Now,
	})
	return toRefreshSessions(records), mapError(err)
}

func (a authRepositoryAdapter) RotateRefreshSession(ctx context.Context, input userstore.RotateRefreshSessionInput) (userstore.RefreshSession, error) {
	record, err := a.delegate.RotateRefreshSession(ctx, internalstore.RotateRefreshSessionInput{
		CurrentTokenID: input.CurrentTokenID,
		NewTokenID:     input.NewTokenID,
		Now:            input.Now,
		RevokedAt:      input.RevokedAt,
		NewExpiresAt:   input.NewExpiresAt,
	})
	return toRefreshSession(record), mapError(err)
}

func (a authRepositoryAdapter) ChangePasswordAndRevokeOtherRefreshSessions(
	ctx context.Context,
	input userstore.ChangePasswordAndRevokeOtherRefreshSessionsInput,
) error {
	delegate, ok := a.delegate.(internalstore.PasswordChangeRepository)
	if !ok {
		return errors.New("auth repository does not support atomic password change")
	}

	return mapError(delegate.ChangePasswordAndRevokeOtherRefreshSessions(ctx, internalstore.ChangePasswordAndRevokeOtherRefreshSessionsInput{
		UserID:             input.UserID,
		PasswordHash:       input.PasswordHash,
		MustChangePassword: input.MustChangePassword,
		ChangedAt:          input.ChangedAt,
		CurrentTokenID:     input.CurrentTokenID,
	}))
}

type passwordChangeRepositoryAdapter struct {
	delegate internalstore.PasswordChangeRepository
}

func (a passwordChangeRepositoryAdapter) ChangePasswordAndRevokeOtherRefreshSessions(
	ctx context.Context,
	input userstore.ChangePasswordAndRevokeOtherRefreshSessionsInput,
) error {
	return mapError(a.delegate.ChangePasswordAndRevokeOtherRefreshSessions(ctx, internalstore.ChangePasswordAndRevokeOtherRefreshSessionsInput{
		UserID:             input.UserID,
		PasswordHash:       input.PasswordHash,
		MustChangePassword: input.MustChangePassword,
		ChangedAt:          input.ChangedAt,
		CurrentTokenID:     input.CurrentTokenID,
	}))
}

func toUser(record internalstore.User) userstore.User {
	return userstore.User{
		ID:        record.ID,
		Username:  record.Username,
		Display:   record.Display,
		CreatedAt: record.CreatedAt,
		UpdatedAt: record.UpdatedAt,
	}
}

func toUsers(records []internalstore.User) []userstore.User {
	items := make([]userstore.User, 0, len(records))
	for _, record := range records {
		items = append(items, toUser(record))
	}
	return items
}

func toUserCredential(record internalstore.UserCredential) userstore.UserCredential {
	return userstore.UserCredential{
		UserID:             record.UserID,
		Username:           record.Username,
		PasswordHash:       record.PasswordHash,
		MustChangePassword: record.MustChangePassword,
		PasswordChangedAt:  record.PasswordChangedAt,
	}
}

func toRefreshSession(record internalstore.RefreshSession) userstore.RefreshSession {
	return userstore.RefreshSession{
		ID:                record.ID,
		UserID:            record.UserID,
		TokenID:           record.TokenID,
		ExpiresAt:         record.ExpiresAt,
		RevokedAt:         record.RevokedAt,
		ReplacedByTokenID: record.ReplacedByTokenID,
		CreatedAt:         record.CreatedAt,
		UpdatedAt:         record.UpdatedAt,
	}
}

func toRefreshSessions(records []internalstore.RefreshSession) []userstore.RefreshSession {
	items := make([]userstore.RefreshSession, 0, len(records))
	for _, record := range records {
		items = append(items, toRefreshSession(record))
	}
	return items
}

func mapError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, internalstore.ErrUserNotFound):
		return userstore.ErrUserNotFound
	case errors.Is(err, internalstore.ErrInvalidID):
		return userstore.ErrInvalidID
	case errors.Is(err, internalstore.ErrRefreshSessionNotFound):
		return userstore.ErrRefreshSessionNotFound
	default:
		return err
	}
}
