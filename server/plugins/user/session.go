package user

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"graft/server/internal/config"
	"graft/server/internal/pluginapi"
	"graft/server/internal/store"
)

var (
	errRefreshTokenRequired = errors.New("refresh token is required")
	errInvalidRefreshToken  = errors.New("invalid refresh token")
	errExpiredRefreshToken  = errors.New("expired refresh token")
	errRefreshSessionFailed = errors.New("refresh session is unavailable")
	errAccessSessionFailed  = errors.New("access session is unavailable")
)

type refreshTokenSubject struct {
	UserID    uint64
	SessionID string
	TokenID   string
}

type refreshResult struct {
	AccessToken   string
	AccessExpiry  time.Time
	RefreshToken  string
	RefreshExpiry time.Time
	User          loginUserResponse
}

type refreshTokenManager struct {
	secret []byte
	ttl    time.Duration
	now    func() time.Time
}

type authCookieManager struct {
	name     string
	path     string
	secure   bool
	sameSite http.SameSite
}

func newRefreshTokenManager(auth config.AuthConfig) (*refreshTokenManager, error) {
	secret := strings.TrimSpace(auth.SigningKey)
	if secret == "" {
		secret = strings.TrimSpace(auth.JWTSecret)
	}
	if secret == "" {
		return nil, errTokenSigningKeyRequired
	}
	if auth.RefreshTokenTTL <= 0 {
		return nil, errors.New("refresh token ttl must be positive")
	}

	return &refreshTokenManager{
		secret: []byte(secret),
		ttl:    auth.RefreshTokenTTL,
		now:    time.Now,
	}, nil
}

func newAuthCookieManager(auth config.AuthConfig) authCookieManager {
	return authCookieManager{
		name:     auth.RefreshCookieName,
		path:     auth.RefreshCookiePath,
		secure:   auth.RefreshCookieSecure,
		sameSite: parseSameSite(strings.TrimSpace(auth.RefreshCookieSameSite)),
	}
}

// writeRefreshCookie 把 refresh token 写入当前响应 cookie。
func (m authCookieManager) writeRefreshCookie(ctx *gin.Context, token string, expiresAt time.Time) {
	if ctx == nil {
		return
	}

	ctx.SetSameSite(m.sameSite)
	ctx.SetCookie(
		m.name,
		token,
		int(time.Until(expiresAt).Seconds()),
		m.path,
		"",
		m.secure,
		true,
	)
}

// clearRefreshCookie 主动让客户端删除当前 refresh token cookie。
func (m authCookieManager) clearRefreshCookie(ctx *gin.Context) {
	if ctx == nil {
		return
	}

	ctx.SetSameSite(m.sameSite)
	ctx.SetCookie(
		m.name,
		"",
		-1,
		m.path,
		"",
		m.secure,
		true,
	)
}

// readRefreshCookie 从请求中读取 refresh token cookie。
func (m authCookieManager) readRefreshCookie(ctx *gin.Context) (string, error) {
	if ctx == nil {
		return "", errRefreshTokenRequired
	}

	value, err := ctx.Cookie(m.name)
	if err != nil {
		return "", errRefreshTokenRequired
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return "", errRefreshTokenRequired
	}

	return value, nil
}

// LoginWithRefresh 在登录成功后同步创建 refresh session、写入 refresh cookie，并返回新 access token。
func (s authService) LoginWithRefresh(ctx context.Context, username string, password string) (refreshResult, error) {
	login, err := s.Login(ctx, username, password)
	if err != nil {
		return refreshResult{}, err
	}

	session, refreshToken, refreshExpiry, err := s.createRefreshSession(ctx, login.User.ID)
	if err != nil {
		return refreshResult{}, err
	}

	accessToken, accessClaims, err := s.tokens.Issue(accessTokenSubject{
		UserID:       login.User.ID,
		SessionID:    session.TokenID,
		TokenVersion: 1,
	})
	if err != nil {
		return refreshResult{}, err
	}

	return refreshResult{
		AccessToken:   accessToken,
		AccessExpiry:  accessClaims.ExpiresAt,
		RefreshToken:  refreshToken,
		RefreshExpiry: refreshExpiry,
		User: loginUserResponse{
			ID:          login.User.ID,
			Username:    login.User.Username,
			DisplayName: login.User.DisplayName,
		},
	}, nil
}

// RefreshWithRotation 校验 refresh token 并完成一次会话轮换。
func (s authService) RefreshWithRotation(ctx context.Context, refreshToken string) (refreshResult, error) {
	if s.auth == nil {
		return refreshResult{}, errors.New("auth repository is unavailable")
	}
	if s.users == nil {
		return refreshResult{}, errors.New("user repository is unavailable")
	}
	if s.tokens == nil {
		return refreshResult{}, errors.New("access token manager is unavailable")
	}
	if s.refreshTokens == nil {
		return refreshResult{}, errors.New("refresh token manager is unavailable")
	}

	claims, err := s.refreshTokens.Parse(refreshToken)
	if err != nil {
		switch {
		case errors.Is(err, errExpiredRefreshToken), errors.Is(err, errInvalidRefreshToken), errors.Is(err, errRefreshTokenRequired):
			return refreshResult{}, errRefreshSessionFailed
		default:
			return refreshResult{}, err
		}
	}

	record, err := s.users.GetByID(ctx, claims.UserID)
	if err != nil {
		if errors.Is(err, store.ErrUserNotFound) {
			return refreshResult{}, errRefreshSessionFailed
		}
		return refreshResult{}, err
	}

	now := s.refreshTokens.now().UTC()
	newTokenID := uuid.NewString()
	nextSession, err := s.auth.RotateRefreshSession(ctx, store.RotateRefreshSessionInput{
		CurrentTokenID: claims.TokenID,
		NewTokenID:     newTokenID,
		Now:            now,
		RevokedAt:      now,
		NewExpiresAt:   now.Add(s.refreshTokens.ttl),
	})
	if err != nil {
		if errors.Is(err, store.ErrRefreshSessionNotFound) {
			return refreshResult{}, errRefreshSessionFailed
		}
		return refreshResult{}, err
	}

	nextRefreshToken, nextRefreshExpiry, err := s.refreshTokens.Issue(refreshTokenSubject{
		UserID:    record.ID,
		SessionID: nextSession.TokenID,
		TokenID:   nextSession.TokenID,
	})
	if err != nil {
		return refreshResult{}, err
	}

	accessToken, accessClaims, err := s.tokens.Issue(accessTokenSubject{
		UserID:       record.ID,
		SessionID:    nextSession.TokenID,
		TokenVersion: 1,
	})
	if err != nil {
		return refreshResult{}, err
	}

	return refreshResult{
		AccessToken:   accessToken,
		AccessExpiry:  accessClaims.ExpiresAt,
		RefreshToken:  nextRefreshToken,
		RefreshExpiry: nextRefreshExpiry,
		User: loginUserResponse{
			ID:          record.ID,
			Username:    record.Username,
			DisplayName: record.Display,
		},
	}, nil
}

// LogoutCurrentSession 读取当前 refresh token 对应的会话并吊销它。
//
// 该流程只处理“当前 cookie 携带的单个 refresh session”；当前用户的全量撤销
// 由独立自助入口负责，管理员按用户批量撤销由专用管理路由负责。
func (s authService) LogoutCurrentSession(ctx context.Context, refreshToken string) error {
	if s.auth == nil {
		return errors.New("auth repository is unavailable")
	}
	if s.refreshTokens == nil {
		return errors.New("refresh token manager is unavailable")
	}

	claims, err := s.refreshTokens.Parse(refreshToken)
	if err != nil {
		switch {
		case errors.Is(err, errExpiredRefreshToken), errors.Is(err, errInvalidRefreshToken), errors.Is(err, errRefreshTokenRequired):
			return errRefreshSessionFailed
		default:
			return err
		}
	}

	session, err := s.auth.GetRefreshSessionByTokenID(ctx, claims.TokenID)
	if err != nil {
		if errors.Is(err, store.ErrRefreshSessionNotFound) {
			return errRefreshSessionFailed
		}
		return err
	}

	now := s.refreshTokens.now().UTC()
	if session.RevokedAt != nil || !session.ExpiresAt.After(now) {
		return errRefreshSessionFailed
	}

	if err := s.auth.RevokeRefreshSession(ctx, store.RevokeRefreshSessionInput{
		TokenID:   claims.TokenID,
		RevokedAt: now,
	}); err != nil {
		if errors.Is(err, store.ErrRefreshSessionNotFound) {
			return errRefreshSessionFailed
		}
		return err
	}

	return nil
}

// RevokeAllCurrentUserSessions 吊销当前已认证用户名下的全部 refresh sessions。
//
// 该流程显式复用 request-auth 中间件已经建立的稳定请求鉴权上下文，只收敛为
// 当前用户自助撤销；管理员代操作则复用同一批量吊销能力走独立管理路由。
func (s authService) RevokeAllCurrentUserSessions(ctx context.Context) error {
	requestAuth, ok := pluginapi.RequestAuthContextFromContext(ctx)
	if !ok || requestAuth.Claims == nil {
		return pluginapi.ErrUnauthenticated
	}

	return s.RevokeAllUserSessions(ctx, requestAuth.Claims.UserID)
}

// RevokeAllUserSessions 吊销指定用户名下的全部 refresh sessions。
//
// 该能力服务于管理员代操作入口，仍然只复用现有仓储批量吊销语义，不额外扩展
// session 模型或把治理细节上推到 core。
func (s authService) RevokeAllUserSessions(ctx context.Context, userID uint64) error {
	if s.auth == nil {
		return errors.New("auth repository is unavailable")
	}

	return s.auth.RevokeRefreshSessionsByUserID(ctx, store.RevokeRefreshSessionsByUserIDInput{
		UserID:    userID,
		RevokedAt: s.nowUTC(),
	})
}

// validateAccessSession 校验 access token 绑定的最小 session 状态。
//
// 当前实现显式复用 refresh session 记录作为 bearer access token 的服务端登录态，
// 这样受保护请求除了验证 JWT 本身，还会拒绝已吊销、已过期或不存在的 session。
func (s authService) validateAccessSession(ctx context.Context, claims *pluginapi.AccessTokenClaims) error {
	if s.auth == nil {
		return errors.New("auth repository is unavailable")
	}
	if claims == nil || strings.TrimSpace(claims.SessionID) == "" {
		return errAccessSessionFailed
	}

	session, err := s.auth.GetRefreshSessionByTokenID(ctx, claims.SessionID)
	if err != nil {
		if errors.Is(err, store.ErrRefreshSessionNotFound) {
			return errAccessSessionFailed
		}
		return err
	}

	now := s.nowUTC()
	if session.UserID != claims.UserID || session.RevokedAt != nil || !session.ExpiresAt.After(now) {
		return errAccessSessionFailed
	}

	return nil
}

func (s authService) createRefreshSession(ctx context.Context, userID uint64) (store.RefreshSession, string, time.Time, error) {
	if s.auth == nil {
		return store.RefreshSession{}, "", time.Time{}, errors.New("auth repository is unavailable")
	}
	if s.refreshTokens == nil {
		return store.RefreshSession{}, "", time.Time{}, errors.New("refresh token manager is unavailable")
	}

	tokenID := uuid.NewString()
	issuedAt := s.refreshTokens.now().UTC()
	expiresAt := issuedAt.Add(s.refreshTokens.ttl)
	session, err := s.auth.CreateRefreshSession(ctx, store.CreateRefreshSessionInput{
		UserID:    userID,
		TokenID:   tokenID,
		ExpiresAt: expiresAt,
	})
	if err != nil {
		return store.RefreshSession{}, "", time.Time{}, err
	}

	token, tokenExpiresAt, err := s.refreshTokens.Issue(refreshTokenSubject{
		UserID:    userID,
		SessionID: session.TokenID,
		TokenID:   session.TokenID,
	})
	if err != nil {
		return store.RefreshSession{}, "", time.Time{}, err
	}

	return session, token, tokenExpiresAt, nil
}

func (s authService) nowUTC() time.Time {
	switch {
	case s.refreshTokens != nil && s.refreshTokens.now != nil:
		return s.refreshTokens.now().UTC()
	case s.tokens != nil && s.tokens.now != nil:
		return s.tokens.now().UTC()
	default:
		return time.Now().UTC()
	}
}

func parseSameSite(value string) http.SameSite {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "strict":
		return http.SameSiteStrictMode
	case "none":
		return http.SameSiteNoneMode
	default:
		return http.SameSiteLaxMode
	}
}
