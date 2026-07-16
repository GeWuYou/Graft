package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"graft/server/internal/moduleapi"
	authstore "graft/server/modules/auth/store"
)

var errInvalidLoginCredentials = errors.New("invalid login credentials")

const invalidLoginPlaceholderHash = "$2a$10$7EqJtq98hPqEX7fNZaFWoO.H8F6dPtkn6rJm5b1Pb9l.eD0P4Qh7K"

type loginResult struct {
	User               moduleapi.CurrentUser
	MustChangePassword bool
}

// Login 校验最小用户名/密码并返回当前主体摘要。
//
// 该流程只负责认证，不提前签发未绑定 refresh session 的 access token；
// 需要建立会话的调用方应继续走 LoginWithRefresh，由它在持久化 session 后
// 签发与服务端状态一致的 access token。
func (s authService) Login(ctx context.Context, username string, password string) (loginResult, error) {
	user, credential, err := s.authenticateUser(ctx, username, password)
	if err != nil {
		return loginResult{}, err
	}

	return loginResult{
		User:               user,
		MustChangePassword: credential.MustChangePassword,
	}, nil
}

// ProvisionPasswordCredential 为已存在的 user profile 创建 auth credential；profile identity 仍由 user 模块拥有，密码策略和散列由 auth 拥有。
func (s authService) ProvisionPasswordCredential(ctx context.Context, userID uint64, password string, mustChangePassword bool) error {
	if s.credentials == nil {
		return errors.New("auth repository is unavailable")
	}
	if err := s.policy.ValidateNewPassword(password); err != nil {
		return err
	}
	hash, err := s.passwords.Hash(password)
	if err != nil {
		return fmt.Errorf("hash initial password: %w", err)
	}
	changedAt := s.nowUTC()
	return s.credentials.SetPasswordHash(ctx, authstore.SetPasswordHashInput{UserID: userID, PasswordHash: hash, MustChangePassword: mustChangePassword, ChangedAt: &changedAt})
}

// ResetPassword 按管理员重置策略更新密码，并原子吊销该用户的全部 refresh session，使旧 access token 失效。
func (s authService) ResetPassword(ctx context.Context, userID uint64, password string) error {
	if s.credentials == nil {
		return errors.New("auth repository is unavailable")
	}
	if err := s.policy.ValidateNewPassword(password); err != nil {
		return err
	}
	hash, err := s.passwords.Hash(password)
	if err != nil {
		return fmt.Errorf("hash reset password: %w", err)
	}
	return s.credentials.ResetPasswordAndRevokeRefreshSessions(ctx, authstore.ResetPasswordAndRevokeSessionsInput{UserID: userID, PasswordHash: hash, MustChangePassword: true, ChangedAt: s.nowUTC()})
}

// RevokeSessions 在 user profile 生命周期结束或身份状态变化时吊销全部 refresh session。
func (s authService) RevokeSessions(ctx context.Context, userID uint64) error {
	if s.sessions == nil {
		return errors.New("auth repository is unavailable")
	}
	return s.sessions.RevokeRefreshSessionsByUserID(ctx, authstore.RevokeRefreshSessionsByUserIDInput{UserID: userID, RevokedAt: s.nowUTC()})
}

func (s authService) authenticateUser(ctx context.Context, username string, password string) (moduleapi.CurrentUser, authstore.UserCredential, error) {
	if s.credentials == nil {
		return moduleapi.CurrentUser{}, authstore.UserCredential{}, errors.New("auth repository is unavailable")
	}
	if s.identity == nil {
		return moduleapi.CurrentUser{}, authstore.UserCredential{}, errors.New("user repository is unavailable")
	}

	credential, err := s.credentials.GetUserCredentialByUsername(ctx, strings.TrimSpace(username))
	if err != nil {
		if errors.Is(err, authstore.ErrCredentialNotFound) {
			// 用户不存在时仍执行一次固定成本的 bcrypt 校验，尽量收敛用户名枚举的时序差异。
			_ = s.passwords.Compare(invalidLoginPlaceholderHash, password)
			return moduleapi.CurrentUser{}, authstore.UserCredential{}, errInvalidLoginCredentials
		}
		return moduleapi.CurrentUser{}, authstore.UserCredential{}, fmt.Errorf("get user credential by username: %w", err)
	}

	if credential.PasswordHash == nil || *credential.PasswordHash == "" {
		// 空散列同样走一次占位校验，避免与真实用户分支出现明显时延差异。
		_ = s.passwords.Compare(invalidLoginPlaceholderHash, password)
		return moduleapi.CurrentUser{}, authstore.UserCredential{}, errInvalidLoginCredentials
	}

	if err := s.passwords.Compare(*credential.PasswordHash, password); err != nil {
		return moduleapi.CurrentUser{}, authstore.UserCredential{}, errInvalidLoginCredentials
	}

	record, err := s.identity.GetCurrentUserByID(ctx, credential.UserID)
	if err != nil {
		if errors.Is(err, moduleapi.ErrUserNotFound) {
			return moduleapi.CurrentUser{}, authstore.UserCredential{}, errInvalidLoginCredentials
		}
		return moduleapi.CurrentUser{}, authstore.UserCredential{}, fmt.Errorf("get user profile by id: %w", err)
	}

	return moduleapi.CurrentUser{
		ID:          record.ID,
		Username:    record.Username,
		DisplayName: record.DisplayName,
	}, credential, nil
}
