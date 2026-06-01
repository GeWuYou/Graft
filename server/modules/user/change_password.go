package user

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/bcrypt"

	"graft/server/internal/pluginapi"
	userstore "graft/server/modules/user/store"
)

type changePasswordActor struct {
	requestAuth pluginapi.RequestAuthContext
	credential  userstore.UserCredential
}

// ChangeCurrentUserPassword 在当前登录态下完成一次自助改密，并保留当前会话。
func (s authService) ChangeCurrentUserPassword(ctx context.Context, currentPassword string, newPassword string) error {
	if s.auth == nil {
		return errors.New("auth repository is unavailable")
	}

	actor, err := s.loadChangePasswordActor(ctx)
	if err != nil {
		return err
	}

	if err := s.validateCurrentPassword(actor.credential, currentPassword); err != nil {
		return err
	}
	if err := s.policy.ValidateNewPassword(newPassword); err != nil {
		return err
	}
	if newPassword == currentPassword {
		return errPasswordReuseForbidden
	}

	hash, err := s.passwords.Hash(newPassword)
	if err != nil {
		return fmt.Errorf("hash new password: %w", err)
	}
	changedAt := s.nowUTC()
	if s.passwordChanges == nil {
		return errors.New("auth repository does not support atomic password change")
	}
	if err := s.passwordChanges.ChangePasswordAndRevokeOtherRefreshSessions(ctx, userstore.ChangePasswordAndRevokeOtherRefreshSessionsInput{
		UserID:             actor.credential.UserID,
		PasswordHash:       hash,
		MustChangePassword: false,
		ChangedAt:          changedAt,
		CurrentTokenID:     actor.requestAuth.Claims.SessionID,
	}); err != nil {
		return fmt.Errorf("change current user password: %w", err)
	}

	return nil
}

// CompleteRequiredPasswordChange 在受限首次改密态下完成一次强制改密。
func (s authService) CompleteRequiredPasswordChange(ctx context.Context, newPassword string) error {
	if s.auth == nil {
		return errors.New("auth repository is unavailable")
	}

	actor, err := s.loadRequiredPasswordChangeActor(ctx)
	if err != nil {
		return err
	}

	if err := s.validateNewPasswordAgainstCurrentHash(actor.credential, newPassword); err != nil {
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
	if err := s.passwordChanges.ChangePasswordAndRevokeOtherRefreshSessions(ctx, userstore.ChangePasswordAndRevokeOtherRefreshSessionsInput{
		UserID:             actor.credential.UserID,
		PasswordHash:       hash,
		MustChangePassword: false,
		ChangedAt:          changedAt,
		CurrentTokenID:     actor.requestAuth.Claims.SessionID,
	}); err != nil {
		return fmt.Errorf("complete required password change: %w", err)
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

func (s authService) currentUserCredential(ctx context.Context, username string) (userstore.UserCredential, error) {
	credential, err := s.auth.GetUserCredentialByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, userstore.ErrUserNotFound) {
			return userstore.UserCredential{}, pluginapi.ErrUnauthenticated
		}
		return userstore.UserCredential{}, fmt.Errorf("get current user credential: %w", err)
	}

	return credential, nil
}

func (s authService) loadChangePasswordActor(ctx context.Context) (changePasswordActor, error) {
	requestAuth, err := currentRequestAuth(ctx)
	if err != nil {
		return changePasswordActor{}, err
	}

	credential, err := s.currentUserCredential(ctx, requestAuth.User.Username)
	if err != nil {
		return changePasswordActor{}, err
	}
	if credential.UserID != requestAuth.Claims.UserID {
		return changePasswordActor{}, pluginapi.ErrUnauthenticated
	}

	return changePasswordActor{
		requestAuth: requestAuth,
		credential:  credential,
	}, nil
}

func (s authService) loadRequiredPasswordChangeActor(ctx context.Context) (changePasswordActor, error) {
	actor, err := s.loadChangePasswordActor(ctx)
	if err != nil {
		return changePasswordActor{}, err
	}
	if !actor.credential.MustChangePassword {
		return changePasswordActor{}, errRequiredPasswordChangeOnly
	}
	if actor.credential.PasswordHash == nil || *actor.credential.PasswordHash == "" {
		return changePasswordActor{}, errors.New("current password hash is unavailable")
	}

	return actor, nil
}

func (s authService) isRestrictedPasswordChangeSession(ctx context.Context) (bool, error) {
	actor, err := s.loadChangePasswordActor(ctx)
	if err != nil {
		return false, err
	}
	return actor.credential.MustChangePassword, nil
}

func (s authService) validateNewPasswordAgainstCurrentHash(credential userstore.UserCredential, newPassword string) error {
	if err := s.policy.ValidateNewPassword(newPassword); err != nil {
		return err
	}
	if credential.PasswordHash == nil || *credential.PasswordHash == "" {
		return errors.New("current password hash is unavailable")
	}
	if err := s.passwords.Compare(*credential.PasswordHash, newPassword); err != nil {
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) || errors.Is(err, errPasswordRequired) {
			return nil
		}
		return fmt.Errorf("compare current password hash: %w", err)
	}

	return errPasswordReuseForbidden
}

func (s authService) validateCurrentPassword(credential userstore.UserCredential, currentPassword string) error {
	if strings.TrimSpace(currentPassword) == "" {
		return errCurrentPasswordRequired
	}
	if credential.PasswordHash == nil || *credential.PasswordHash == "" {
		return errCurrentPasswordInvalid
	}
	if err := s.passwords.Compare(*credential.PasswordHash, currentPassword); err != nil {
		return errCurrentPasswordInvalid
	}

	return nil
}
