package user

import (
	"context"
	"errors"
	"fmt"

	"graft/server/internal/pluginapi"
	"graft/server/internal/store"
)

type changePasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

// ChangeCurrentUserPassword 在当前登录态下完成一次自助改密，并保留当前会话。
func (s authService) ChangeCurrentUserPassword(ctx context.Context, currentPassword string, newPassword string) error {
	if s.auth == nil {
		return errors.New("auth repository is unavailable")
	}

	requestAuth, err := currentRequestAuth(ctx)
	if err != nil {
		return err
	}
	credential, err := s.currentUserCredential(ctx, requestAuth.User.Username)
	if err != nil {
		return err
	}
	if err := s.validateCurrentPassword(credential, currentPassword); err != nil {
		return err
	}
	if credential.UserID != requestAuth.Claims.UserID {
		return pluginapi.ErrUnauthenticated
	}
	if err := s.policy.ValidateNewPassword(currentPassword, newPassword); err != nil {
		return err
	}

	hash, err := s.passwords.Hash(newPassword)
	if err != nil {
		return fmt.Errorf("hash new password: %w", err)
	}
	changedAt := s.nowUTC()
	if s.passwordChanges == nil {
		return errors.New("auth repository does not support atomic password change")
	}
	if err := s.passwordChanges.ChangePasswordAndRevokeOtherRefreshSessions(ctx, store.ChangePasswordAndRevokeOtherRefreshSessionsInput{
		UserID:             credential.UserID,
		PasswordHash:       hash,
		MustChangePassword: false,
		ChangedAt:          changedAt,
		CurrentTokenID:     requestAuth.Claims.SessionID,
	}); err != nil {
		return fmt.Errorf("change current user password: %w", err)
	}

	return nil
}

func currentRequestAuth(ctx context.Context) (pluginapi.RequestAuthContext, error) {
	requestAuth, ok := pluginapi.RequestAuthContextFromContext(ctx)
	if !ok || requestAuth.User == nil || requestAuth.Claims == nil {
		return pluginapi.RequestAuthContext{}, pluginapi.ErrUnauthenticated
	}

	return requestAuth, nil
}

func (s authService) currentUserCredential(ctx context.Context, username string) (store.UserCredential, error) {
	credential, err := s.auth.GetUserCredentialByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, store.ErrUserNotFound) {
			return store.UserCredential{}, pluginapi.ErrUnauthenticated
		}
		return store.UserCredential{}, fmt.Errorf("get current user credential: %w", err)
	}

	return credential, nil
}

func (s authService) validateCurrentPassword(credential store.UserCredential, currentPassword string) error {
	if credential.PasswordHash == nil || *credential.PasswordHash == "" {
		return errCurrentPasswordInvalid
	}
	if err := s.passwords.Compare(*credential.PasswordHash, currentPassword); err != nil {
		return errCurrentPasswordInvalid
	}

	return nil
}
