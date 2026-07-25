package auth

import (
	"context"
	"errors"
	"strings"
	"time"

	"graft/server/internal/config"
	"graft/server/internal/moduleapi"
	authstore "graft/server/modules/auth/store"
)

// authService 承担认证运行时职责：user 模块提供 profile identity，auth 通过窄化 store 接口拥有 credential 与 refresh session 状态。
type authService struct {
	credentials    authstore.CredentialStore
	sessions       authstore.SessionStore
	transactions   authstore.TransactionRunner
	identity       moduleapi.UserIdentityProvider
	passwords      passwordHasher
	policy         passwordPolicy
	tokens         *AccessTokenManager
	refreshTokens  *RefreshTokenManager
	personalTokens authstore.PersonalAccessTokenStore
	now            func() time.Time
}

// newAuthService 创建认证服务及其必需的 store、身份提供方、密码组件和 token manager。
// 当必需依赖缺失、token manager 初始化失败，或 credential store 不提供 auth 本地事务边界时返回错误。
func newAuthService(
	authConfig config.AuthConfig,
	credentials authstore.CredentialStore,
	sessions authstore.SessionStore,
	identity moduleapi.UserIdentityProvider,
	personalTokens ...authstore.PersonalAccessTokenStore,
) (*authService, error) {
	if credentials == nil || sessions == nil || identity == nil {
		return nil, errors.New("auth runtime dependencies are unavailable")
	}
	tokens, err := NewAccessTokenManager(authConfig)
	if err != nil {
		return nil, err
	}
	refreshTokens, err := NewRefreshTokenManager(authConfig)
	if err != nil {
		return nil, err
	}
	transactions, ok := credentials.(authstore.TransactionRunner)
	if !ok {
		return nil, errors.New("credential store does not support auth transactions")
	}
	var personalTokenStore authstore.PersonalAccessTokenStore
	if len(personalTokens) > 0 {
		personalTokenStore = personalTokens[0]
	}
	return &authService{
		credentials:    credentials,
		sessions:       sessions,
		transactions:   transactions,
		identity:       identity,
		passwords:      newPasswordHasher(),
		policy:         newPasswordPolicy(),
		tokens:         tokens,
		refreshTokens:  refreshTokens,
		personalTokens: personalTokenStore,
		now:            time.Now,
	}, nil
}

func (s authService) CurrentUser(ctx context.Context) (*moduleapi.CurrentUser, error) {
	requestAuth, ok := moduleapi.RequestAuthContextFromContext(ctx)
	if !ok || requestAuth.Claims == nil {
		return nil, moduleapi.ErrUnauthenticated
	}
	user, err := s.identity.GetCurrentUserByID(ctx, requestAuth.Claims.UserID)
	if err != nil {
		return nil, moduleapi.ErrUnauthenticated
	}
	return &user, nil
}

func (s authService) ParseAccessToken(ctx context.Context, token string) (*moduleapi.AccessTokenClaims, error) {
	claims, err := s.tokens.Parse(strings.TrimSpace(token))
	if err != nil {
		switch {
		case errors.Is(err, ErrExpiredAccessToken):
			return nil, moduleapi.ErrExpiredAccessToken
		case errors.Is(err, ErrInvalidAccessToken):
			return nil, moduleapi.ErrInvalidAccessToken
		default:
			return nil, err
		}
	}
	if err := s.validateAccessSession(ctx, claims); err != nil {
		if errors.Is(err, errAccessSessionFailed) {
			return nil, moduleapi.ErrInvalidAccessToken
		}
		return nil, err
	}
	return claims, nil
}

func (s authService) ListSessionsByUserID(ctx context.Context, userID uint64) ([]moduleapi.AuthSessionSummary, error) {
	sessions, err := s.ListUserSessions(ctx, userID, sessionListOptions{})
	if err != nil {
		return nil, err
	}
	result := make([]moduleapi.AuthSessionSummary, 0, len(sessions))
	for _, session := range sessions {
		result = append(result, moduleapi.AuthSessionSummary{SessionID: session.SessionID, UserID: userID, CreatedAt: session.CreatedAt, ExpiresAt: session.ExpiresAt, Current: session.Current})
	}
	return result, nil
}

func (s authService) RevokeSessionByUserID(ctx context.Context, userID uint64, sessionID string) (moduleapi.AuthSessionRevokeResult, error) {
	if err := s.RevokeUserSession(ctx, userID, sessionID); err != nil {
		return moduleapi.AuthSessionRevokeResult{}, err
	}
	return moduleapi.AuthSessionRevokeResult{Revoked: true}, nil
}
func (s authService) RevokeSessionsByUserID(ctx context.Context, userID uint64) (moduleapi.AuthSessionRevokeResult, error) {
	if err := s.RevokeAllUserSessions(ctx, userID); err != nil {
		return moduleapi.AuthSessionRevokeResult{}, err
	}
	return moduleapi.AuthSessionRevokeResult{Revoked: true}, nil
}
func (s authService) RevokeOtherSessionsByUserID(ctx context.Context, userID uint64, currentSessionID string) (moduleapi.AuthSessionRevokeResult, error) {
	sessions, err := s.ListUserSessions(ctx, userID, sessionListOptions{})
	if err != nil {
		return moduleapi.AuthSessionRevokeResult{}, err
	}
	revoked := false
	for _, session := range sessions {
		if session.SessionID != currentSessionID {
			if err := s.RevokeUserSession(ctx, userID, session.SessionID); err != nil {
				return moduleapi.AuthSessionRevokeResult{}, err
			}
			revoked = true
		}
	}
	return moduleapi.AuthSessionRevokeResult{Revoked: revoked}, nil
}

var _ moduleapi.AuthService = authService{}
var _ moduleapi.AuthSessionService = authService{}
var _ moduleapi.AuthCredentialManagementService = authService{}
var _ moduleapi.PersonalAccessTokenService = authService{}
