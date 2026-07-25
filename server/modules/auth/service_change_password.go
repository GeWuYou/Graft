package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"graft/server/internal/moduleapi"
	authstore "graft/server/modules/auth/store"
)

type changePasswordActor struct {
	requestAuth moduleapi.RequestAuthContext
	credential  authstore.UserCredential
}

// ChangeCurrentUserPassword 在当前登录态下完成一次自助改密，并保留当前会话。
func (s authService) ChangeCurrentUserPassword(ctx context.Context, currentPassword string, newPassword string) error {
	if s.credentials == nil {
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
	if err := s.updatePasswordAndRevokeSessions(ctx, actor.credential.UserID, hash, false, changedAt, actor.requestAuth.Claims.SessionID); err != nil {
		return fmt.Errorf("change current user password: %w", err)
	}

	return nil
}

// CompleteRequiredPasswordChange 在受限首次改密态下完成一次强制改密。
func (s authService) CompleteRequiredPasswordChange(ctx context.Context, newPassword string) error {
	if s.credentials == nil {
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
	if err := s.updatePasswordAndRevokeSessions(ctx, actor.credential.UserID, hash, false, changedAt, actor.requestAuth.Claims.SessionID); err != nil {
		return fmt.Errorf("complete required password change: %w", err)
	}

	return nil
}

// updatePasswordAndRevokeSessions 在 auth 服务拥有的单个事务范围内更新凭据并吊销会话。
func (s authService) updatePasswordAndRevokeSessions(ctx context.Context, userID uint64, hash string, mustChange bool, changedAt time.Time, currentTokenID string) error {
	if s.transactions == nil {
		return errors.New("auth transaction runner is unavailable")
	}
	return s.transactions.RunInTransaction(ctx, func(txCtx context.Context, credentials authstore.CredentialStore, sessions authstore.SessionStore) error {
		if err := credentials.SetPasswordHash(txCtx, authstore.SetPasswordHashInput{UserID: userID, PasswordHash: hash, MustChangePassword: mustChange, ChangedAt: &changedAt}); err != nil {
			return fmt.Errorf("update password hash: %w", err)
		}
		if currentTokenID == "" {
			if err := sessions.RevokeRefreshSessionsByUserID(txCtx, authstore.RevokeRefreshSessionsByUserIDInput{UserID: userID, RevokedAt: changedAt}); err != nil {
				return fmt.Errorf("revoke refresh sessions: %w", err)
			}
			return nil
		}
		if err := sessions.RevokeOtherRefreshSessionsByUserID(txCtx, authstore.RevokeOtherRefreshSessionsInput{UserID: userID, CurrentTokenID: currentTokenID, RevokedAt: changedAt}); err != nil {
			return fmt.Errorf("revoke other refresh sessions: %w", err)
		}
		return nil
	})
}

// requireRequestAuth 从 ctx 读取已认证请求上下文；上下文、用户或 claims 缺失时返回未认证错误。
func requireRequestAuth(ctx context.Context) (moduleapi.RequestAuthContext, error) {
	requestAuth, ok := moduleapi.RequestAuthContextFromContext(ctx)
	if !ok || requestAuth.User == nil || requestAuth.Claims == nil {
		return moduleapi.RequestAuthContext{}, moduleapi.ErrUnauthenticated
	}

	return requestAuth, nil
}

func (s authService) currentUserCredential(ctx context.Context, username string) (authstore.UserCredential, error) {
	credential, err := s.credentials.GetUserCredentialByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, authstore.ErrCredentialNotFound) {
			return authstore.UserCredential{}, moduleapi.ErrUnauthenticated
		}
		return authstore.UserCredential{}, fmt.Errorf("get current user credential: %w", err)
	}

	return credential, nil
}

func (s authService) loadChangePasswordActor(ctx context.Context) (changePasswordActor, error) {
	requestAuth, err := requireRequestAuth(ctx)
	if err != nil {
		return changePasswordActor{}, err
	}

	credential, err := s.currentUserCredential(ctx, requestAuth.User.Username)
	if err != nil {
		return changePasswordActor{}, err
	}
	if credential.UserID != requestAuth.Claims.UserID {
		return changePasswordActor{}, moduleapi.ErrUnauthenticated
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

func (s authService) validateNewPasswordAgainstCurrentHash(credential authstore.UserCredential, newPassword string) error {
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

func (s authService) validateCurrentPassword(credential authstore.UserCredential, currentPassword string) error {
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
