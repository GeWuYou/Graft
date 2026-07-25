package auth

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"

	"graft/server/internal/moduleapi"
	authstore "graft/server/modules/auth/store"
)

var (
	errRefreshTokenRequired       = ErrRefreshTokenRequired
	errInvalidRefreshToken        = ErrInvalidRefreshToken
	errExpiredRefreshToken        = ErrExpiredRefreshToken
	errRefreshSessionFailed       = errors.New("refresh session is unavailable")
	errAccessSessionFailed        = errors.New("access session is unavailable")
	errSessionNotFound            = errors.New("session not found")
	errPasswordPolicyViolation    = moduleapi.ErrPasswordPolicyViolation
	errPasswordReuseForbidden     = moduleapi.ErrPasswordReuseForbidden
	errCurrentPasswordRequired    = errors.New("current password is required")
	errCurrentPasswordInvalid     = errors.New("current password is invalid")
	errRequiredPasswordChangeOnly = errors.New("required password change only")
)

type refreshTokenSubject = RefreshTokenSubject
type accessTokenSubject = AccessTokenSubject

type loginUserResponse struct {
	ID          uint64 `json:"id"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
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
	Session       authstore.RefreshSession
	Token         string
	TokenExpiryAt time.Time
}

type sessionListOptions struct {
	Limit int
}

type sessionSummary struct {
	SessionID string    `json:"session_id"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
	Current   bool      `json:"current"`
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

// RefreshWithRotation 校验 refresh token 与服务端 session，并原子轮换 session；旧 token 和其 access token 随后不可继续使用。
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

// LogoutCurrentSession 吊销 refresh token 对应的服务端 session；调用方随后应清除浏览器 cookie。
func (s authService) LogoutCurrentSession(ctx context.Context, refreshToken string) error {
	if err := s.ensureLogoutDependencies(); err != nil {
		return err
	}
	claims, err := s.parseRefreshClaims(refreshToken)
	if err != nil {
		return err
	}
	session, err := s.sessions.GetRefreshSessionByTokenID(ctx, claims.TokenID)
	if err != nil {
		if errors.Is(err, authstore.ErrRefreshSessionNotFound) {
			return errInvalidRefreshToken
		}
		return err
	}

	now := s.refreshTokens.now().UTC()
	if session.RevokedAt != nil || !session.ExpiresAt.After(now) {
		return errInvalidRefreshToken
	}

	if err := s.sessions.RevokeRefreshSession(ctx, authstore.RevokeRefreshSessionInput{
		TokenID:   claims.TokenID,
		RevokedAt: now,
	}); err != nil {
		if errors.Is(err, authstore.ErrRefreshSessionNotFound) {
			return errInvalidRefreshToken
		}
		return err
	}

	return nil
}

func (s authService) ensureRefreshDependencies() error {
	switch {
	case s.credentials == nil:
		return errors.New("credential store is unavailable")
	case s.sessions == nil:
		return errors.New("session store is unavailable")
	case s.identity == nil:
		return errors.New("user identity provider is unavailable")
	case s.tokens == nil:
		return errors.New("access token manager is unavailable")
	case s.refreshTokens == nil:
		return errors.New("refresh token manager is unavailable")
	default:
		return nil
	}
}

func (s authService) ensureLogoutDependencies() error {
	if s.sessions == nil {
		return errors.New("session store is unavailable")
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
) (moduleapi.CurrentUser, authstore.UserCredential, error) {
	record, err := s.identity.GetCurrentUserByID(ctx, userID)
	if err != nil {
		if errors.Is(err, moduleapi.ErrUserNotFound) {
			return moduleapi.CurrentUser{}, authstore.UserCredential{}, errInvalidRefreshToken
		}
		return moduleapi.CurrentUser{}, authstore.UserCredential{}, err
	}
	credential, err := s.credentials.GetUserCredentialByUsername(ctx, record.Username)
	if err != nil {
		if errors.Is(err, authstore.ErrCredentialNotFound) {
			return moduleapi.CurrentUser{}, authstore.UserCredential{}, errInvalidRefreshToken
		}
		return moduleapi.CurrentUser{}, authstore.UserCredential{}, err
	}

	return record, credential, nil
}

func (s authService) validateActiveRefreshSession(
	ctx context.Context,
	claims *refreshTokenSubject,
	now time.Time,
) error {
	session, err := s.sessions.GetRefreshSessionByTokenID(ctx, claims.TokenID)
	if err != nil {
		return mapRefreshSessionRepositoryError(err)
	}
	if session.UserID != claims.UserID || session.RevokedAt != nil || !session.ExpiresAt.After(now) {
		return errInvalidRefreshToken
	}

	return nil
}

// validateRefreshRotationAllowed 根据用户凭据状态判断 refresh token 是否允许轮换；要求先改密时返回受限会话错误。
func validateRefreshRotationAllowed(credential authstore.UserCredential) error {
	if credential.MustChangePassword {
		return errRequiredPasswordChangeOnly
	}

	return nil
}

func (s authService) rotateRefreshSession(
	ctx context.Context,
	currentTokenID string,
	now time.Time,
) (authstore.RefreshSession, error) {
	if s.transactions == nil {
		return authstore.RefreshSession{}, errors.New("auth transaction runner is unavailable")
	}
	var nextSession authstore.RefreshSession
	err := s.transactions.RunInTransaction(ctx, func(txCtx context.Context, _ authstore.CredentialStore, sessions authstore.SessionStore) error {
		var rotateErr error
		nextSession, rotateErr = sessions.RotateRefreshSession(txCtx, authstore.RotateRefreshSessionInput{
			CurrentTokenID: currentTokenID,
			NewTokenID:     uuid.NewString(),
			Now:            now,
			RevokedAt:      now,
			NewExpiresAt:   now.Add(s.refreshTokens.ttl),
		})
		return rotateErr
	})
	if err != nil {
		return authstore.RefreshSession{}, mapRefreshSessionRepositoryError(err)
	}

	return nextSession, nil
}

func (s authService) issueRefreshRotationResult(
	record moduleapi.CurrentUser,
	credential authstore.UserCredential,
	nextSession authstore.RefreshSession,
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
			DisplayName: record.DisplayName,
		},
	}, nil
}

// mapRefreshSessionRepositoryError 将 refresh session 不存在映射为无效 refresh token 错误。
func mapRefreshSessionRepositoryError(err error) error {
	if errors.Is(err, authstore.ErrRefreshSessionNotFound) {
		return errInvalidRefreshToken
	}

	return err
}

func (s authService) RevokeAllCurrentUserSessions(ctx context.Context) error {
	requestAuth, ok := moduleapi.RequestAuthContextFromContext(ctx)
	if !ok || requestAuth.Claims == nil {
		return moduleapi.ErrUnauthenticated
	}

	return s.RevokeAllUserSessions(ctx, requestAuth.Claims.UserID)
}

func (s authService) RevokeOtherCurrentUserSessions(ctx context.Context) error {
	requestAuth, ok := moduleapi.RequestAuthContextFromContext(ctx)
	if !ok || requestAuth.Claims == nil {
		return moduleapi.ErrUnauthenticated
	}

	_, err := s.revokeOtherSessions(ctx, requestAuth.Claims.UserID, requestAuth.Claims.SessionID)
	return err
}

func (s authService) RevokeAllUserSessions(ctx context.Context, userID uint64) error {
	if s.sessions == nil {
		return errors.New("session store is unavailable")
	}

	return s.sessions.RevokeRefreshSessionsByUserID(ctx, authstore.RevokeRefreshSessionsByUserIDInput{
		UserID:    userID,
		RevokedAt: s.nowUTC(),
	})
}

func (s authService) RevokeCurrentUserSession(ctx context.Context, sessionID string) error {
	requestAuth, ok := moduleapi.RequestAuthContextFromContext(ctx)
	if !ok || requestAuth.Claims == nil {
		return moduleapi.ErrUnauthenticated
	}

	return s.RevokeUserSession(ctx, requestAuth.Claims.UserID, sessionID)
}

func (s authService) RevokeUserSession(ctx context.Context, userID uint64, sessionID string) error {
	if s.sessions == nil {
		return errors.New("session store is unavailable")
	}

	if strings.TrimSpace(sessionID) == "" {
		return errSessionNotFound
	}

	if err := s.sessions.RevokeRefreshSessionByUserID(ctx, authstore.RevokeRefreshSessionByUserIDInput{
		UserID:    userID,
		TokenID:   strings.TrimSpace(sessionID),
		RevokedAt: s.nowUTC(),
	}); err != nil {
		if errors.Is(err, authstore.ErrRefreshSessionNotFound) {
			return errSessionNotFound
		}
		return err
	}

	return nil
}

func (s authService) ListCurrentUserSessions(ctx context.Context, options sessionListOptions) ([]sessionSummary, error) {
	requestAuth, ok := moduleapi.RequestAuthContextFromContext(ctx)
	if !ok || requestAuth.Claims == nil {
		return nil, moduleapi.ErrUnauthenticated
	}

	return s.ListUserSessions(ctx, requestAuth.Claims.UserID, options)
}

func (s authService) ListUserSessions(ctx context.Context, userID uint64, options sessionListOptions) ([]sessionSummary, error) {
	if s.sessions == nil {
		return nil, errors.New("session store is unavailable")
	}

	requestAuth, _ := moduleapi.RequestAuthContextFromContext(ctx)
	sessions, err := s.sessions.ListActiveRefreshSessionsByUserID(ctx, authstore.ListActiveRefreshSessionsByUserIDInput{
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

func (s authService) validateAccessSession(ctx context.Context, claims *moduleapi.AccessTokenClaims) error {
	if s.sessions == nil {
		return errors.New("session store is unavailable")
	}
	if claims == nil || strings.TrimSpace(claims.SessionID) == "" {
		return errAccessSessionFailed
	}

	session, err := s.sessions.GetRefreshSessionByTokenID(ctx, claims.SessionID)
	if err != nil {
		if errors.Is(err, authstore.ErrRefreshSessionNotFound) {
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
	if s.sessions == nil {
		return refreshSessionGrant{}, errors.New("session store is unavailable")
	}
	if s.refreshTokens == nil {
		return refreshSessionGrant{}, errors.New("refresh token manager is unavailable")
	}

	tokenID := uuid.NewString()
	issuedAt := s.refreshTokens.now().UTC()
	expiresAt := issuedAt.Add(s.refreshTokens.ttl)
	session, err := s.sessions.CreateRefreshSession(ctx, authstore.CreateRefreshSessionInput{
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
	case s.tokens != nil:
		return time.Now().UTC()
	default:
		return time.Now().UTC()
	}
}
