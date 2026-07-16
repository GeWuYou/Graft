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
	"graft/server/internal/moduleapi"
)

var (
	// ErrTokenSigningKeyRequired 表示未配置 token 签名密钥。
	ErrTokenSigningKeyRequired = errors.New("token signing key is required")
	// ErrSessionIDRequired 表示 token 必须包含会话标识。
	ErrSessionIDRequired = errors.New("session id is required")
	// ErrTokenIDRequired 表示 refresh token 必须包含 token 标识。
	ErrTokenIDRequired = errors.New("token id is required")
	// ErrInvalidAccessToken 表示 access token 格式错误或校验无效。
	ErrInvalidAccessToken = errors.New("invalid access token")
	// ErrExpiredAccessToken 表示 access token 已过期。
	ErrExpiredAccessToken = errors.New("expired access token")
	// ErrRefreshTokenRequired 表示请求必须提供 refresh token。
	ErrRefreshTokenRequired = errors.New("refresh token is required")
	// ErrInvalidRefreshToken 表示 refresh token 格式错误或校验无效。
	ErrInvalidRefreshToken = errors.New("invalid refresh token")
	// ErrExpiredRefreshToken 表示 refresh token 已过期。
	ErrExpiredRefreshToken = errors.New("expired refresh token")
)

// AccessTokenSubject 是签发单个 access token 所需的最小主体信息。
type AccessTokenSubject struct {
	UserID       uint64
	SessionID    string
	TokenVersion int
}

// RefreshTokenSubject 是签发单个 refresh token 所需的最小主体信息。
type RefreshTokenSubject struct {
	UserID    uint64
	SessionID string
	TokenID   string
}

// AccessTokenManager 负责 auth 所有 access token 的签发与解析。
type AccessTokenManager struct {
	secret []byte
	ttl    time.Duration
	now    func() time.Time
}

// RefreshTokenManager 负责 auth 所有 refresh token 的签发与解析。
type RefreshTokenManager struct {
	secret []byte
	ttl    time.Duration
	now    func() time.Time
}

// CookieManager 负责 refresh cookie 的读取、写入和清除语义。
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

// NewAccessTokenManager 根据认证配置创建 access token 管理器；签名密钥或有效期缺失时返回错误。
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

// NewRefreshTokenManager 根据认证配置创建 refresh token 管理器；签名密钥或有效期缺失时返回错误。
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

// NewCookieManager 根据认证配置创建 refresh cookie 管理器。
func NewCookieManager(auth config.AuthConfig) CookieManager {
	return CookieManager{
		name:     auth.RefreshCookieName,
		path:     auth.RefreshCookiePath,
		secure:   auth.RefreshCookieSecure,
		sameSite: parseSameSite(strings.TrimSpace(auth.RefreshCookieSameSite)),
	}
}

// Issue 为指定主体签发 access token；主体缺少用户或会话标识时返回错误。
func (m *AccessTokenManager) Issue(subject AccessTokenSubject) (string, moduleapi.AccessTokenClaims, error) {
	if subject.UserID == 0 {
		return "", moduleapi.AccessTokenClaims{}, fmt.Errorf("user id is required")
	}
	if strings.TrimSpace(subject.SessionID) == "" {
		return "", moduleapi.AccessTokenClaims{}, ErrSessionIDRequired
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
		return "", moduleapi.AccessTokenClaims{}, fmt.Errorf("sign access token: %w", err)
	}

	return signed, moduleapi.AccessTokenClaims{
		UserID:       subject.UserID,
		SessionID:    subject.SessionID,
		TokenVersion: subject.TokenVersion,
		ExpiresAt:    expiresAt,
		IssuedAt:     issuedAt,
	}, nil
}

// Parse 校验 access token 并返回稳定 claims；过期和格式无效分别映射为对应认证错误。
func (m *AccessTokenManager) Parse(token string) (*moduleapi.AccessTokenClaims, error) {
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

	return &moduleapi.AccessTokenClaims{
		UserID:       userID,
		SessionID:    claims.SessionID,
		TokenVersion: claims.TokenVersion,
		IssuedAt:     claims.IssuedAt.UTC(),
		ExpiresAt:    claims.ExpiresAt.UTC(),
	}, nil
}

// Issue 为指定主体签发 refresh token；主体缺少用户、会话或 token 标识时返回错误。
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

// Parse 校验 refresh token 并返回稳定主体信息；过期和格式无效分别映射为对应认证错误。
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

// WriteRefreshCookie 写入当前 refresh token cookie。
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

// ClearRefreshCookie 清除当前 refresh token cookie。
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

// ReadRefreshCookie 读取当前 refresh token cookie；cookie 缺失时返回 refresh token 必填错误。
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
