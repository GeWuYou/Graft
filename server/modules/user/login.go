// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package user

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"graft/server/internal/config"
	"graft/server/internal/moduleapi"
	userstore "graft/server/modules/user/store"
)

var errInvalidLoginCredentials = errors.New("invalid login credentials")

const invalidLoginPlaceholderHash = "$2a$10$7EqJtq98hPqEX7fNZaFWoO.H8F6dPtkn6rJm5b1Pb9l.eD0P4Qh7K"

type loginResult struct {
	User               moduleapi.CurrentUser
	MustChangePassword bool
}

func newAuthService(authConfig config.AuthConfig, authRepo userstore.AuthRepository, usersRepo userstore.UserRepository) (*authService, error) {
	tokens, err := newAccessTokenManager(authConfig)
	if err != nil {
		return nil, err
	}
	refreshTokens, err := newRefreshTokenManager(authConfig)
	if err != nil {
		return nil, err
	}

	var passwordChanges userstore.PasswordChangeRepository
	if candidate, ok := authRepo.(userstore.PasswordChangeRepository); ok {
		passwordChanges = candidate
	}

	return &authService{
		auth:            authRepo,
		passwordChanges: passwordChanges,
		users:           usersRepo,
		passwords:       newPasswordHasher(),
		policy:          newPasswordPolicy(),
		tokens:          tokens,
		refreshTokens:   refreshTokens,
		cookies:         newAuthCookieManager(authConfig),
	}, nil
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

func (s authService) authenticateUser(ctx context.Context, username string, password string) (moduleapi.CurrentUser, userstore.UserCredential, error) {
	if s.auth == nil {
		return moduleapi.CurrentUser{}, userstore.UserCredential{}, errors.New("auth repository is unavailable")
	}
	if s.users == nil {
		return moduleapi.CurrentUser{}, userstore.UserCredential{}, errors.New("user repository is unavailable")
	}

	credential, err := s.auth.GetUserCredentialByUsername(ctx, strings.TrimSpace(username))
	if err != nil {
		if errors.Is(err, userstore.ErrUserNotFound) {
			// 用户不存在时仍执行一次固定成本的 bcrypt 校验，尽量收敛用户名枚举的时序差异。
			_ = s.passwords.Compare(invalidLoginPlaceholderHash, password)
			return moduleapi.CurrentUser{}, userstore.UserCredential{}, errInvalidLoginCredentials
		}
		return moduleapi.CurrentUser{}, userstore.UserCredential{}, fmt.Errorf("get user credential by username: %w", err)
	}

	if credential.PasswordHash == nil || *credential.PasswordHash == "" {
		// 空散列同样走一次占位校验，避免与真实用户分支出现明显时延差异。
		_ = s.passwords.Compare(invalidLoginPlaceholderHash, password)
		return moduleapi.CurrentUser{}, userstore.UserCredential{}, errInvalidLoginCredentials
	}

	if err := s.passwords.Compare(*credential.PasswordHash, password); err != nil {
		return moduleapi.CurrentUser{}, userstore.UserCredential{}, errInvalidLoginCredentials
	}

	record, err := s.users.GetByID(ctx, credential.UserID)
	if err != nil {
		if errors.Is(err, userstore.ErrUserNotFound) {
			return moduleapi.CurrentUser{}, userstore.UserCredential{}, errInvalidLoginCredentials
		}
		return moduleapi.CurrentUser{}, userstore.UserCredential{}, fmt.Errorf("get user profile by id: %w", err)
	}

	return moduleapi.CurrentUser{
		ID:          record.ID,
		Username:    record.Username,
		DisplayName: record.Display,
	}, credential, nil
}
