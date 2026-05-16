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
	errRefreshTokenRequired       = errors.New("refresh token is required")
	errInvalidRefreshToken        = errors.New("invalid refresh token")
	errExpiredRefreshToken        = errors.New("expired refresh token")
	errRefreshSessionFailed       = errors.New("refresh session is unavailable")
	errAccessSessionFailed        = errors.New("access session is unavailable")
	errSessionNotFound            = errors.New("session not found")
	errPasswordPolicyViolation    = errors.New("password policy violation")
	errPasswordReuseForbidden     = errors.New("password reuse forbidden")
	errCurrentPasswordRequired    = errors.New("current password is required")
	errCurrentPasswordInvalid     = errors.New("current password is invalid")
	errRequiredPasswordChangeOnly = errors.New("required password change only")
)

type refreshTokenSubject struct {
	UserID    uint64
	SessionID string
	TokenID   string
}

type refreshResult struct {
	AccessToken        string
	AccessExpiry       time.Time
	RefreshToken       string
	RefreshExpiry      time.Time
	MustChangePassword bool
	User               loginUserResponse
}

type refreshSessionGrant struct {
	Session       store.RefreshSession
	Token         string
	TokenExpiryAt time.Time
}

type sessionSummary struct {
	SessionID string    `json:"session_id"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
	Current   bool      `json:"current"`
}

type sessionListOptions struct {
	Limit int
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

	refreshGrant, err := s.createRefreshSession(ctx, login.User.ID)
	if err != nil {
		return refreshResult{}, err
	}

	accessToken, accessClaims, err := s.tokens.Issue(accessTokenSubject{
		UserID:       login.User.ID,
		SessionID:    refreshGrant.Session.TokenID,
		TokenVersion: 1,
	})
	if err != nil {
		return refreshResult{}, err
	}

	return refreshResult{
		AccessToken:        accessToken,
		AccessExpiry:       accessClaims.ExpiresAt,
		RefreshToken:       refreshGrant.Token,
		RefreshExpiry:      refreshGrant.TokenExpiryAt,
		MustChangePassword: login.MustChangePassword,
		User: loginUserResponse{
			ID:          login.User.ID,
			Username:    login.User.Username,
			DisplayName: login.User.DisplayName,
		},
	}, nil
}

// RefreshWithRotation 校验 refresh token 并完成一次会话轮换。
func (s authService) RefreshWithRotation(ctx context.Context, refreshToken string) (refreshResult, error) {
	if err := s.ensureRefreshDependencies(); err != nil {
		return refreshResult{}, err
	}

	claims, err := s.parseRefreshClaims(refreshToken)
	if err != nil {
		return refreshResult{}, err
	}

	record, credential, err := s.loadRefreshActor(ctx, claims.UserID)
	if err != nil {
		return refreshResult{}, err
	}

	now := s.refreshTokens.now().UTC()
	if err := s.validateActiveRefreshSession(ctx, claims, now); err != nil {
		return refreshResult{}, err
	}

	if err := validateRefreshRotationAllowed(credential); err != nil {
		return refreshResult{}, err
	}

	nextSession, err := s.rotateRefreshSession(ctx, claims.TokenID, now)
	if err != nil {
		return refreshResult{}, err
	}

	return s.issueRefreshRotationResult(record, credential, nextSession)
}

// LogoutCurrentSession 读取当前 refresh token 对应的会话并吊销它。
//
// 该流程只处理“当前 cookie 携带的单个 refresh session”；当前用户的全量撤销
// 由独立自助入口负责，管理员按用户批量撤销由专用管理路由负责。
func (s authService) LogoutCurrentSession(ctx context.Context, refreshToken string) error {
	if err := s.ensureLogoutDependencies(); err != nil {
		return err
	}
	claims, err := s.parseRefreshClaims(refreshToken)
	if err != nil {
		return err
	}
	session, err := s.auth.GetRefreshSessionByTokenID(ctx, claims.TokenID)
	if err != nil {
		if errors.Is(err, store.ErrRefreshSessionNotFound) {
			return errInvalidRefreshToken
		}
		return err
	}

	now := s.refreshTokens.now().UTC()
	if session.RevokedAt != nil || !session.ExpiresAt.After(now) {
		return errInvalidRefreshToken
	}

	if err := s.auth.RevokeRefreshSession(ctx, store.RevokeRefreshSessionInput{
		TokenID:   claims.TokenID,
		RevokedAt: now,
	}); err != nil {
		if errors.Is(err, store.ErrRefreshSessionNotFound) {
			return errInvalidRefreshToken
		}
		return err
	}

	return nil
}

func (s authService) ensureRefreshDependencies() error {
	switch {
	case s.auth == nil:
		return errors.New("auth repository is unavailable")
	case s.users == nil:
		return errors.New("user repository is unavailable")
	case s.tokens == nil:
		return errors.New("access token manager is unavailable")
	case s.refreshTokens == nil:
		return errors.New("refresh token manager is unavailable")
	default:
		return nil
	}
}

func (s authService) ensureLogoutDependencies() error {
	if s.auth == nil {
		return errors.New("auth repository is unavailable")
	}
	if s.refreshTokens == nil {
		return errors.New("refresh token manager is unavailable")
	}

	return nil
}

func (s authService) parseRefreshClaims(refreshToken string) (*refreshTokenSubject, error) {
	claims, err := s.refreshTokens.Parse(refreshToken)
	if err == nil {
		return claims, nil
	}

	switch {
	case errors.Is(err, errRefreshTokenRequired):
		return nil, errRefreshTokenRequired
	case errors.Is(err, errExpiredRefreshToken):
		return nil, errExpiredRefreshToken
	case errors.Is(err, errInvalidRefreshToken):
		return nil, errInvalidRefreshToken
	default:
		return nil, err
	}
}

func (s authService) loadRefreshActor(
	ctx context.Context,
	userID uint64,
) (store.User, store.UserCredential, error) {
	record, err := s.users.GetByID(ctx, userID)
	if err != nil {
		if errors.Is(err, store.ErrUserNotFound) {
			return store.User{}, store.UserCredential{}, errInvalidRefreshToken
		}
		return store.User{}, store.UserCredential{}, err
	}
	credential, err := s.auth.GetUserCredentialByUsername(ctx, record.Username)
	if err != nil {
		if errors.Is(err, store.ErrUserNotFound) {
			return store.User{}, store.UserCredential{}, errInvalidRefreshToken
		}
		return store.User{}, store.UserCredential{}, err
	}

	return record, credential, nil
}

func (s authService) validateActiveRefreshSession(
	ctx context.Context,
	claims *refreshTokenSubject,
	now time.Time,
) error {
	session, err := s.auth.GetRefreshSessionByTokenID(ctx, claims.TokenID)
	if err != nil {
		return mapRefreshSessionRepositoryError(err)
	}
	if session.UserID != claims.UserID || session.RevokedAt != nil || !session.ExpiresAt.After(now) {
		return errInvalidRefreshToken
	}

	return nil
}

func validateRefreshRotationAllowed(credential store.UserCredential) error {
	if credential.MustChangePassword {
		return errRequiredPasswordChangeOnly
	}

	return nil
}

func (s authService) rotateRefreshSession(
	ctx context.Context,
	currentTokenID string,
	now time.Time,
) (store.RefreshSession, error) {
	nextSession, err := s.auth.RotateRefreshSession(ctx, store.RotateRefreshSessionInput{
		CurrentTokenID: currentTokenID,
		NewTokenID:     uuid.NewString(),
		Now:            now,
		RevokedAt:      now,
		NewExpiresAt:   now.Add(s.refreshTokens.ttl),
	})
	if err != nil {
		return store.RefreshSession{}, mapRefreshSessionRepositoryError(err)
	}

	return nextSession, nil
}

func (s authService) issueRefreshRotationResult(
	record store.User,
	credential store.UserCredential,
	nextSession store.RefreshSession,
) (refreshResult, error) {
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
		AccessToken:        accessToken,
		AccessExpiry:       accessClaims.ExpiresAt,
		RefreshToken:       nextRefreshToken,
		RefreshExpiry:      nextRefreshExpiry,
		MustChangePassword: credential.MustChangePassword,
		User: loginUserResponse{
			ID:          record.ID,
			Username:    record.Username,
			DisplayName: record.Display,
		},
	}, nil
}

func mapRefreshSessionRepositoryError(err error) error {
	if errors.Is(err, store.ErrRefreshSessionNotFound) {
		return errInvalidRefreshToken
	}

	return err
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

// RevokeOtherCurrentUserSessions 吊销当前登录主体除当前请求外的其它有效 refresh sessions。
//
// 该能力服务于“保留当前登录态并清退其它设备”的最小治理场景，继续把批量遍历与
// 定向吊销逻辑留在 user 插件内，而不提前扩展仓储或跨插件公共契约。
func (s authService) RevokeOtherCurrentUserSessions(ctx context.Context) error {
	requestAuth, ok := pluginapi.RequestAuthContextFromContext(ctx)
	if !ok || requestAuth.Claims == nil {
		return pluginapi.ErrUnauthenticated
	}

	sessions, err := s.ListUserSessions(ctx, requestAuth.Claims.UserID, sessionListOptions{})
	if err != nil {
		return err
	}

	for _, session := range sessions {
		if session.SessionID == requestAuth.Claims.SessionID {
			continue
		}
		if err := s.RevokeUserSession(ctx, requestAuth.Claims.UserID, session.SessionID); err != nil {
			// 其它端会话在枚举后到定向吊销前可能已自然过期或被并发清退；这里把
			// “已不存在可吊销会话”视为幂等成功，避免中断剩余 session 的清退。
			if errors.Is(err, errSessionNotFound) {
				continue
			}
			return err
		}
	}

	return nil
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

// RevokeCurrentUserSession 吊销当前登录主体名下的单个有效 refresh session。
//
// 该能力只允许当前主体在自身会话集合内做定向吊销，避免把跨用户治理语义混入
// 自助入口。
func (s authService) RevokeCurrentUserSession(ctx context.Context, sessionID string) error {
	requestAuth, ok := pluginapi.RequestAuthContextFromContext(ctx)
	if !ok || requestAuth.Claims == nil {
		return pluginapi.ErrUnauthenticated
	}

	return s.RevokeUserSession(ctx, requestAuth.Claims.UserID, sessionID)
}

// RevokeUserSession 吊销指定用户名下的单个有效 refresh session。
//
// 该能力保持在 user 插件内，通过用户 ID 与 session ID 的显式组合约束定向
// 吊销范围，不把底层查询或权限细节扩散到 core。
func (s authService) RevokeUserSession(ctx context.Context, userID uint64, sessionID string) error {
	if s.auth == nil {
		return errors.New("auth repository is unavailable")
	}

	if strings.TrimSpace(sessionID) == "" {
		return errSessionNotFound
	}

	if err := s.auth.RevokeRefreshSessionByUserID(ctx, store.RevokeRefreshSessionByUserIDInput{
		UserID:    userID,
		TokenID:   strings.TrimSpace(sessionID),
		RevokedAt: s.nowUTC(),
	}); err != nil {
		if errors.Is(err, store.ErrRefreshSessionNotFound) {
			return errSessionNotFound
		}
		return err
	}

	return nil
}

// ListCurrentUserSessions 返回当前登录主体可见的有效 refresh session 摘要。
//
// 该能力只读取 request-auth 已建立的稳定主体上下文，不引入额外跨插件会话契约。
func (s authService) ListCurrentUserSessions(ctx context.Context, options sessionListOptions) ([]sessionSummary, error) {
	requestAuth, ok := pluginapi.RequestAuthContextFromContext(ctx)
	if !ok || requestAuth.Claims == nil {
		return nil, pluginapi.ErrUnauthenticated
	}

	return s.ListUserSessions(ctx, requestAuth.Claims.UserID, options)
}

// ListUserSessions 返回指定用户当前有效 refresh session 摘要。
//
// 该能力仍停留在 user 插件内，把持久化层 session 记录映射为最小治理视图，
// 不把历史轮换细节或底层 ORM 结构直接暴露给调用方。
func (s authService) ListUserSessions(ctx context.Context, userID uint64, options sessionListOptions) ([]sessionSummary, error) {
	if s.auth == nil {
		return nil, errors.New("auth repository is unavailable")
	}

	requestAuth, _ := pluginapi.RequestAuthContextFromContext(ctx)
	sessions, err := s.auth.ListActiveRefreshSessionsByUserID(ctx, store.ListActiveRefreshSessionsByUserIDInput{
		UserID: userID,
		Now:    s.nowUTC(),
	})
	if err != nil {
		return nil, err
	}

	summaries := make([]sessionSummary, 0, len(sessions))
	for _, session := range sessions {
		summaries = append(summaries, sessionSummary{
			SessionID: session.TokenID,
			CreatedAt: session.CreatedAt,
			ExpiresAt: session.ExpiresAt,
			Current:   requestAuth.Claims != nil && requestAuth.Claims.SessionID == session.TokenID,
		})
	}

	if options.Limit > 0 && len(summaries) > options.Limit {
		summaries = summaries[:options.Limit]
	}

	return summaries, nil
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

func (s authService) createRefreshSession(ctx context.Context, userID uint64) (refreshSessionGrant, error) {
	if s.auth == nil {
		return refreshSessionGrant{}, errors.New("auth repository is unavailable")
	}
	if s.refreshTokens == nil {
		return refreshSessionGrant{}, errors.New("refresh token manager is unavailable")
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
		return refreshSessionGrant{}, err
	}

	token, tokenExpiresAt, err := s.refreshTokens.Issue(refreshTokenSubject{
		UserID:    userID,
		SessionID: session.TokenID,
		TokenID:   session.TokenID,
	})
	if err != nil {
		return refreshSessionGrant{}, err
	}

	return refreshSessionGrant{
		Session:       session,
		Token:         token,
		TokenExpiryAt: tokenExpiresAt,
	}, nil
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
