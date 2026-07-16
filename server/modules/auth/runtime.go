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
	// ErrTokenSigningKeyRequired 表示 token manager 无法在没有签名密钥时安全启动。
	ErrTokenSigningKeyRequired = errors.New("token signing key is required")
	// ErrSessionIDRequired 表示 access/refresh token 必须绑定服务端 session。
	ErrSessionIDRequired = errors.New("session id is required")
	// ErrTokenIDRequired 表示 refresh token 必须包含用于轮换和吊销的唯一 token 标识。
	ErrTokenIDRequired = errors.New("token id is required")
	// ErrInvalidAccessToken 表示 access token 格式、签名或 claims 校验失败。
	ErrInvalidAccessToken = errors.New("invalid access token")
	// ErrExpiredAccessToken 表示 access token 已过期，调用方应重新认证或刷新会话。
	ErrExpiredAccessToken = errors.New("expired access token")
	// ErrRefreshTokenRequired 表示 refresh 流程未从请求中取得 token。
	ErrRefreshTokenRequired = errors.New("refresh token is required")
	// ErrInvalidRefreshToken 表示 refresh token 格式、签名、claims 或服务端 session 校验失败。
	ErrInvalidRefreshToken = errors.New("invalid refresh token")
	// ErrExpiredRefreshToken 表示 refresh token 已过期，不能继续轮换 session。
	ErrExpiredRefreshToken = errors.New("expired refresh token")
)

// AccessTokenSubject 描述 access token 的用户和服务端 session 绑定关系。
type AccessTokenSubject struct {
	UserID       uint64
	SessionID    string
	TokenVersion int
}

// RefreshTokenSubject 描述 refresh token 的用户、session 和轮换标识；TokenID 用于服务端吊销状态关联。
type RefreshTokenSubject struct {
	UserID    uint64
	SessionID string
	TokenID   string
}

// AccessTokenManager 负责 access token 的签发与解析；Parse 只验证 JWT，不替代服务端 session 状态校验。
type AccessTokenManager struct {
	secret []byte
	ttl    time.Duration
	now    func() time.Time
}

// RefreshTokenManager 负责 refresh token 的签发与解析；有效 token 仍须由 auth service 对照持久化 session 校验。
type RefreshTokenManager struct {
	secret []byte
	ttl    time.Duration
	now    func() time.Time
}

// CookieManager 统一 refresh token 的 HttpOnly cookie 行为，包含路径、Secure、SameSite 和过期清除策略。
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

// NewCookieManager 根据认证配置创建 refresh cookie 管理器；未识别的 SameSite 值回退为 Lax。
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

// WriteRefreshCookie 以 HttpOnly cookie 写入 refresh token，并使用 token 到期时间计算 cookie 生命周期。
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

// ClearRefreshCookie 通过同名、同路径的过期 cookie 清除 refresh token。
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

// ReadRefreshCookie 读取并裁剪 refresh cookie；缺失、空值或 nil 请求上下文统一返回 ErrRefreshTokenRequired。
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
