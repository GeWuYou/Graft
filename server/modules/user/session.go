// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package user

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"

	"graft/server/internal/config"
	"graft/server/internal/moduleapi"
	authruntime "graft/server/modules/auth"
	userstore "graft/server/modules/user/store"
)

var (
	errRefreshTokenRequired       = authruntime.ErrRefreshTokenRequired
	errInvalidRefreshToken        = authruntime.ErrInvalidRefreshToken
	errExpiredRefreshToken        = authruntime.ErrExpiredRefreshToken
	errRefreshSessionFailed       = errors.New("refresh session is unavailable")
	errAccessSessionFailed        = errors.New("access session is unavailable")
	errSessionNotFound            = errors.New("session not found")
	errPasswordPolicyViolation    = errors.New("password policy violation")
	errPasswordReuseForbidden     = errors.New("password reuse forbidden")
	errCurrentPasswordRequired    = errors.New("current password is required")
	errCurrentPasswordInvalid     = errors.New("current password is invalid")
	errRequiredPasswordChangeOnly = errors.New("required password change only")
)

type refreshTokenSubject = authruntime.RefreshTokenSubject

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
	Session       userstore.RefreshSession
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

type refreshTokenManager struct {
	inner *authruntime.RefreshTokenManager
	ttl   time.Duration
	now   func() time.Time
}

type authCookieManager struct {
	inner authruntime.CookieManager
}

func newRefreshTokenManager(auth config.AuthConfig) (*refreshTokenManager, error) {
	inner, err := authruntime.NewRefreshTokenManager(auth)
	if err != nil {
		return nil, err
	}

	return &refreshTokenManager{
		inner: inner,
		ttl:   auth.RefreshTokenTTL,
		now:   time.Now,
	}, nil
}

func newAuthCookieManager(auth config.AuthConfig) authCookieManager {
	return authCookieManager{inner: authruntime.NewCookieManager(auth)}
}

func (m *refreshTokenManager) Issue(subject refreshTokenSubject) (string, time.Time, error) {
	return m.inner.Issue(subject)
}

func (m *refreshTokenManager) Parse(token string) (*refreshTokenSubject, error) {
	return m.inner.Parse(token)
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
		if errors.Is(err, userstore.ErrRefreshSessionNotFound) {
			return errInvalidRefreshToken
		}
		return err
	}

	now := s.refreshTokens.now().UTC()
	if session.RevokedAt != nil || !session.ExpiresAt.After(now) {
		return errInvalidRefreshToken
	}

	if err := s.auth.RevokeRefreshSession(ctx, userstore.RevokeRefreshSessionInput{
		TokenID:   claims.TokenID,
		RevokedAt: now,
	}); err != nil {
		if errors.Is(err, userstore.ErrRefreshSessionNotFound) {
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
) (userstore.User, userstore.UserCredential, error) {
	record, err := s.users.GetByID(ctx, userID)
	if err != nil {
		if errors.Is(err, userstore.ErrUserNotFound) {
			return userstore.User{}, userstore.UserCredential{}, errInvalidRefreshToken
		}
		return userstore.User{}, userstore.UserCredential{}, err
	}
	credential, err := s.auth.GetUserCredentialByUsername(ctx, record.Username)
	if err != nil {
		if errors.Is(err, userstore.ErrUserNotFound) {
			return userstore.User{}, userstore.UserCredential{}, errInvalidRefreshToken
		}
		return userstore.User{}, userstore.UserCredential{}, err
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

func validateRefreshRotationAllowed(credential userstore.UserCredential) error {
	if credential.MustChangePassword {
		return errRequiredPasswordChangeOnly
	}

	return nil
}

func (s authService) rotateRefreshSession(
	ctx context.Context,
	currentTokenID string,
	now time.Time,
) (userstore.RefreshSession, error) {
	nextSession, err := s.auth.RotateRefreshSession(ctx, userstore.RotateRefreshSessionInput{
		CurrentTokenID: currentTokenID,
		NewTokenID:     uuid.NewString(),
		Now:            now,
		RevokedAt:      now,
		NewExpiresAt:   now.Add(s.refreshTokens.ttl),
	})
	if err != nil {
		return userstore.RefreshSession{}, mapRefreshSessionRepositoryError(err)
	}

	return nextSession, nil
}

func (s authService) issueRefreshRotationResult(
	record userstore.User,
	credential userstore.UserCredential,
	nextSession userstore.RefreshSession,
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
	if errors.Is(err, userstore.ErrRefreshSessionNotFound) {
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

	sessions, err := s.ListUserSessions(ctx, requestAuth.Claims.UserID, sessionListOptions{})
	if err != nil {
		return err
	}

	for _, session := range sessions {
		if session.SessionID == requestAuth.Claims.SessionID {
			continue
		}
		if err := s.RevokeUserSession(ctx, requestAuth.Claims.UserID, session.SessionID); err != nil {
			if errors.Is(err, errSessionNotFound) {
				continue
			}
			return err
		}
	}

	return nil
}

func (s authService) RevokeAllUserSessions(ctx context.Context, userID uint64) error {
	if s.auth == nil {
		return errors.New("auth repository is unavailable")
	}

	return s.auth.RevokeRefreshSessionsByUserID(ctx, userstore.RevokeRefreshSessionsByUserIDInput{
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
	if s.auth == nil {
		return errors.New("auth repository is unavailable")
	}

	if strings.TrimSpace(sessionID) == "" {
		return errSessionNotFound
	}

	if err := s.auth.RevokeRefreshSessionByUserID(ctx, userstore.RevokeRefreshSessionByUserIDInput{
		UserID:    userID,
		TokenID:   strings.TrimSpace(sessionID),
		RevokedAt: s.nowUTC(),
	}); err != nil {
		if errors.Is(err, userstore.ErrRefreshSessionNotFound) {
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
	if s.auth == nil {
		return nil, errors.New("auth repository is unavailable")
	}

	requestAuth, _ := moduleapi.RequestAuthContextFromContext(ctx)
	sessions, err := s.auth.ListActiveRefreshSessionsByUserID(ctx, userstore.ListActiveRefreshSessionsByUserIDInput{
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
	if s.auth == nil {
		return errors.New("auth repository is unavailable")
	}
	if claims == nil || strings.TrimSpace(claims.SessionID) == "" {
		return errAccessSessionFailed
	}

	session, err := s.auth.GetRefreshSessionByTokenID(ctx, claims.SessionID)
	if err != nil {
		if errors.Is(err, userstore.ErrRefreshSessionNotFound) {
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
	session, err := s.auth.CreateRefreshSession(ctx, userstore.CreateRefreshSessionInput{
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
