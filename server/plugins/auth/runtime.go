package auth

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"graft/server/internal/config"
	"graft/server/internal/pluginapi"
)

var (
	ErrTokenSigningKeyRequired = errors.New("token signing key is required")
	ErrSessionIDRequired       = errors.New("session id is required")
	ErrTokenIDRequired         = errors.New("token id is required")
	ErrInvalidAccessToken      = errors.New("invalid access token")
	ErrExpiredAccessToken      = errors.New("expired access token")
	ErrRefreshTokenRequired    = errors.New("refresh token is required")
	ErrInvalidRefreshToken     = errors.New("invalid refresh token")
	ErrExpiredRefreshToken     = errors.New("expired refresh token")
)

type AccessTokenSubject struct {
	UserID       uint64
	SessionID    string
	TokenVersion int
}

type RefreshTokenSubject struct {
	UserID    uint64
	SessionID string
	TokenID   string
}

type AccessTokenManager struct {
	secret []byte
	ttl    time.Duration
	now    func() time.Time
}

type RefreshTokenManager struct {
	secret []byte
	ttl    time.Duration
	now    func() time.Time
}

type CookieManager struct {
	name     string
	path     string
	secure   bool
	sameSite http.SameSite
}

type accessTokenJWTClaims struct {
	SessionID    string `json:"session_id"`
	TokenVersion int    `json:"token_version,omitempty"`
	jwt.RegisteredClaims
}

type refreshTokenJWTClaims struct {
	SessionID string `json:"session_id"`
	TokenID   string `json:"token_id"`
	jwt.RegisteredClaims
}

func NewAccessTokenManager(auth config.AuthConfig) (*AccessTokenManager, error) {
	secret := strings.TrimSpace(auth.SigningKey)
	if secret == "" {
		secret = strings.TrimSpace(auth.JWTSecret)
	}
	if secret == "" {
		return nil, ErrTokenSigningKeyRequired
	}
	if auth.AccessTokenTTL <= 0 {
		return nil, fmt.Errorf("access token ttl must be positive")
	}

	return &AccessTokenManager{
		secret: []byte(secret),
		ttl:    auth.AccessTokenTTL,
		now:    time.Now,
	}, nil
}

func NewRefreshTokenManager(auth config.AuthConfig) (*RefreshTokenManager, error) {
	secret := strings.TrimSpace(auth.SigningKey)
	if secret == "" {
		secret = strings.TrimSpace(auth.JWTSecret)
	}
	if secret == "" {
		return nil, ErrTokenSigningKeyRequired
	}
	if auth.RefreshTokenTTL <= 0 {
		return nil, errors.New("refresh token ttl must be positive")
	}

	return &RefreshTokenManager{
		secret: []byte(secret),
		ttl:    auth.RefreshTokenTTL,
		now:    time.Now,
	}, nil
}

func NewCookieManager(auth config.AuthConfig) CookieManager {
	return CookieManager{
		name:     auth.RefreshCookieName,
		path:     auth.RefreshCookiePath,
		secure:   auth.RefreshCookieSecure,
		sameSite: parseSameSite(strings.TrimSpace(auth.RefreshCookieSameSite)),
	}
}

func (m *AccessTokenManager) Issue(subject AccessTokenSubject) (string, pluginapi.AccessTokenClaims, error) {
	if subject.UserID == 0 {
		return "", pluginapi.AccessTokenClaims{}, fmt.Errorf("user id is required")
	}
	if strings.TrimSpace(subject.SessionID) == "" {
		return "", pluginapi.AccessTokenClaims{}, ErrSessionIDRequired
	}

	issuedAt := m.now().UTC()
	expiresAt := issuedAt.Add(m.ttl)
	tokenClaims := accessTokenJWTClaims{
		SessionID:    subject.SessionID,
		TokenVersion: subject.TokenVersion,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   strconv.FormatUint(subject.UserID, 10),
			IssuedAt:  jwt.NewNumericDate(issuedAt),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	}

	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, tokenClaims).SignedString(m.secret)
	if err != nil {
		return "", pluginapi.AccessTokenClaims{}, fmt.Errorf("sign access token: %w", err)
	}

	return signed, pluginapi.AccessTokenClaims{
		UserID:       subject.UserID,
		SessionID:    subject.SessionID,
		TokenVersion: subject.TokenVersion,
		ExpiresAt:    expiresAt,
		IssuedAt:     issuedAt,
	}, nil
}

func (m *AccessTokenManager) Parse(token string) (*pluginapi.AccessTokenClaims, error) {
	claims := &accessTokenJWTClaims{}
	parser := jwt.NewParser(
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithTimeFunc(m.now),
	)
	parsed, err := parser.ParseWithClaims(token, claims, func(_ *jwt.Token) (any, error) {
		return m.secret, nil
	})
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredAccessToken
		}
		return nil, fmt.Errorf("%w: %v", ErrInvalidAccessToken, err)
	}
	if !parsed.Valid {
		return nil, ErrInvalidAccessToken
	}

	userID, err := strconv.ParseUint(claims.Subject, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid subject", ErrInvalidAccessToken)
	}
	if claims.IssuedAt == nil || claims.ExpiresAt == nil {
		return nil, fmt.Errorf("%w: missing temporal claims", ErrInvalidAccessToken)
	}
	if strings.TrimSpace(claims.SessionID) == "" {
		return nil, fmt.Errorf("%w: missing session id", ErrInvalidAccessToken)
	}

	return &pluginapi.AccessTokenClaims{
		UserID:       userID,
		SessionID:    claims.SessionID,
		TokenVersion: claims.TokenVersion,
		IssuedAt:     claims.IssuedAt.UTC(),
		ExpiresAt:    claims.ExpiresAt.UTC(),
	}, nil
}

func (m *RefreshTokenManager) Issue(subject RefreshTokenSubject) (string, time.Time, error) {
	if subject.UserID == 0 {
		return "", time.Time{}, errors.New("user id is required")
	}
	if strings.TrimSpace(subject.SessionID) == "" {
		return "", time.Time{}, ErrSessionIDRequired
	}
	if strings.TrimSpace(subject.TokenID) == "" {
		return "", time.Time{}, ErrTokenIDRequired
	}

	issuedAt := m.now().UTC()
	expiresAt := issuedAt.Add(m.ttl)
	tokenClaims := refreshTokenJWTClaims{
		SessionID: subject.SessionID,
		TokenID:   subject.TokenID,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   strconv.FormatUint(subject.UserID, 10),
			IssuedAt:  jwt.NewNumericDate(issuedAt),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	}

	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, tokenClaims).SignedString(m.secret)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("sign refresh token: %w", err)
	}

	return signed, expiresAt, nil
}

func (m *RefreshTokenManager) Parse(token string) (*RefreshTokenSubject, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil, ErrRefreshTokenRequired
	}

	claims := &refreshTokenJWTClaims{}
	parser := jwt.NewParser(
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithTimeFunc(m.now),
	)
	parsed, err := parser.ParseWithClaims(token, claims, func(_ *jwt.Token) (any, error) {
		return m.secret, nil
	})
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredRefreshToken
		}
		return nil, fmt.Errorf("%w: %v", ErrInvalidRefreshToken, err)
	}
	if !parsed.Valid {
		return nil, ErrInvalidRefreshToken
	}

	userID, err := strconv.ParseUint(claims.Subject, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid subject", ErrInvalidRefreshToken)
	}
	if strings.TrimSpace(claims.SessionID) == "" {
		return nil, fmt.Errorf("%w: missing session id", ErrInvalidRefreshToken)
	}
	if strings.TrimSpace(claims.TokenID) == "" {
		return nil, fmt.Errorf("%w: missing token id", ErrInvalidRefreshToken)
	}

	return &RefreshTokenSubject{
		UserID:    userID,
		SessionID: claims.SessionID,
		TokenID:   claims.TokenID,
	}, nil
}

func (m CookieManager) WriteRefreshCookie(ctx *gin.Context, token string, expiresAt time.Time) {
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

func (m CookieManager) ClearRefreshCookie(ctx *gin.Context) {
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

func (m CookieManager) ReadRefreshCookie(ctx *gin.Context) (string, error) {
	if ctx == nil {
		return "", ErrRefreshTokenRequired
	}

	value, err := ctx.Cookie(m.name)
	if err != nil {
		return "", ErrRefreshTokenRequired
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return "", ErrRefreshTokenRequired
	}

	return value, nil
}

func parseSameSite(raw string) http.SameSite {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "strict":
		return http.SameSiteStrictMode
	case "none":
		return http.SameSiteNoneMode
	default:
		return http.SameSiteLaxMode
	}
}
