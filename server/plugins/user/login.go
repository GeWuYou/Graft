package user

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"graft/server/internal/config"
	"graft/server/internal/pluginapi"
	"graft/server/internal/store"
)

var errInvalidLoginCredentials = errors.New("invalid login credentials")

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type loginUserResponse struct {
	ID          uint64 `json:"id"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
}

type loginResponse struct {
	AccessToken string            `json:"access_token"`
	ExpiresAt   time.Time         `json:"expires_at"`
	User        loginUserResponse `json:"user"`
}

type loginResult struct {
	User pluginapi.CurrentUser
}

func newAuthService(authConfig config.AuthConfig, authRepo store.AuthRepository, usersRepo store.UserRepository) (*authService, error) {
	tokens, err := newAccessTokenManager(authConfig)
	if err != nil {
		return nil, err
	}
	refreshTokens, err := newRefreshTokenManager(authConfig)
	if err != nil {
		return nil, err
	}

	return &authService{
		auth:          authRepo,
		users:         usersRepo,
		passwords:     newPasswordHasher(),
		tokens:        tokens,
		refreshTokens: refreshTokens,
		cookies:       newAuthCookieManager(authConfig),
	}, nil
}

// Login 校验最小用户名/密码并返回当前主体摘要。
//
// 该流程只负责认证，不提前签发未绑定 refresh session 的 access token；
// 需要建立会话的调用方应继续走 LoginWithRefresh，由它在持久化 session 后
// 签发与服务端状态一致的 access token。
func (s authService) Login(ctx context.Context, username string, password string) (loginResult, error) {
	user, err := s.authenticateUser(ctx, username, password)
	if err != nil {
		return loginResult{}, err
	}

	return loginResult{
		User: user,
	}, nil
}

func (s authService) authenticateUser(ctx context.Context, username string, password string) (pluginapi.CurrentUser, error) {
	if s.auth == nil {
		return pluginapi.CurrentUser{}, errors.New("auth repository is unavailable")
	}
	if s.users == nil {
		return pluginapi.CurrentUser{}, errors.New("user repository is unavailable")
	}

	credential, err := s.auth.GetUserCredentialByUsername(ctx, strings.TrimSpace(username))
	if err != nil {
		if errors.Is(err, store.ErrUserNotFound) {
			return pluginapi.CurrentUser{}, errInvalidLoginCredentials
		}
		return pluginapi.CurrentUser{}, fmt.Errorf("get user credential by username: %w", err)
	}

	if credential.PasswordHash == nil || *credential.PasswordHash == "" {
		return pluginapi.CurrentUser{}, errInvalidLoginCredentials
	}

	if err := s.passwords.Compare(*credential.PasswordHash, password); err != nil {
		return pluginapi.CurrentUser{}, errInvalidLoginCredentials
	}

	record, err := s.users.GetByID(ctx, credential.UserID)
	if err != nil {
		if errors.Is(err, store.ErrUserNotFound) {
			return pluginapi.CurrentUser{}, errInvalidLoginCredentials
		}
		return pluginapi.CurrentUser{}, fmt.Errorf("get user profile by id: %w", err)
	}

	return pluginapi.CurrentUser{
		ID:          record.ID,
		Username:    record.Username,
		DisplayName: record.Display,
	}, nil
}
