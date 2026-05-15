package user

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"graft/server/internal/config"
	"graft/server/internal/pluginapi"
)

var (
	errTokenSigningKeyRequired = errors.New("token signing key is required")
	errSessionIDRequired       = errors.New("session id is required")
	errTokenIDRequired         = errors.New("token id is required")
	errInvalidAccessToken      = errors.New("invalid access token")
	errExpiredAccessToken      = errors.New("expired access token")
)

type accessTokenSubject struct {
	UserID       uint64
	SessionID    string
	TokenVersion int
}

type accessTokenManager struct {
	secret []byte
	ttl    time.Duration
	now    func() time.Time
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

func newAccessTokenManager(auth config.AuthConfig) (*accessTokenManager, error) {
	secret := strings.TrimSpace(auth.SigningKey)
	if secret == "" {
		secret = strings.TrimSpace(auth.JWTSecret)
	}
	if secret == "" {
		return nil, errTokenSigningKeyRequired
	}
	if auth.AccessTokenTTL <= 0 {
		return nil, fmt.Errorf("access token ttl must be positive")
	}

	return &accessTokenManager{
		secret: []byte(secret),
		ttl:    auth.AccessTokenTTL,
		now:    time.Now,
	}, nil
}

// Issue 生成 HS256 access token，并返回同步的稳定 claims 视图供调用方复用。
func (m *accessTokenManager) Issue(subject accessTokenSubject) (string, pluginapi.AccessTokenClaims, error) {
	if subject.UserID == 0 {
		return "", pluginapi.AccessTokenClaims{}, fmt.Errorf("user id is required")
	}
	if strings.TrimSpace(subject.SessionID) == "" {
		return "", pluginapi.AccessTokenClaims{}, errSessionIDRequired
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

// Parse 校验 HS256 access token，并收敛为稳定 claims DTO。
func (m *accessTokenManager) Parse(token string) (*pluginapi.AccessTokenClaims, error) {
	claims := &accessTokenJWTClaims{}
	parser := jwt.NewParser(
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithTimeFunc(m.now),
	)
	parsed, err := parser.ParseWithClaims(token, claims, func(current *jwt.Token) (any, error) {
		return m.secret, nil
	})
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, errExpiredAccessToken
		}
		return nil, fmt.Errorf("%w: %v", errInvalidAccessToken, err)
	}
	if !parsed.Valid {
		return nil, errInvalidAccessToken
	}

	userID, err := strconv.ParseUint(claims.Subject, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid subject", errInvalidAccessToken)
	}
	if claims.IssuedAt == nil || claims.ExpiresAt == nil {
		return nil, fmt.Errorf("%w: missing temporal claims", errInvalidAccessToken)
	}
	if strings.TrimSpace(claims.SessionID) == "" {
		return nil, fmt.Errorf("%w: missing session id", errInvalidAccessToken)
	}

	return &pluginapi.AccessTokenClaims{
		UserID:       userID,
		SessionID:    claims.SessionID,
		TokenVersion: claims.TokenVersion,
		IssuedAt:     claims.IssuedAt.Time.UTC(),
		ExpiresAt:    claims.ExpiresAt.Time.UTC(),
	}, nil
}

// Issue 生成 HS256 refresh token，并返回 token 到期时间。
func (m *refreshTokenManager) Issue(subject refreshTokenSubject) (string, time.Time, error) {
	if subject.UserID == 0 {
		return "", time.Time{}, errors.New("user id is required")
	}
	if strings.TrimSpace(subject.SessionID) == "" {
		return "", time.Time{}, errSessionIDRequired
	}
	if strings.TrimSpace(subject.TokenID) == "" {
		return "", time.Time{}, errTokenIDRequired
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

// Parse 校验 refresh token 并返回最小主体信息。
func (m *refreshTokenManager) Parse(token string) (*refreshTokenSubject, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil, errRefreshTokenRequired
	}

	claims := &refreshTokenJWTClaims{}
	parser := jwt.NewParser(
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithTimeFunc(m.now),
	)
	parsed, err := parser.ParseWithClaims(token, claims, func(current *jwt.Token) (any, error) {
		return m.secret, nil
	})
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, errExpiredRefreshToken
		}
		return nil, fmt.Errorf("%w: %v", errInvalidRefreshToken, err)
	}
	if !parsed.Valid {
		return nil, errInvalidRefreshToken
	}

	userID, err := strconv.ParseUint(claims.Subject, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid subject", errInvalidRefreshToken)
	}
	if strings.TrimSpace(claims.SessionID) == "" {
		return nil, fmt.Errorf("%w: missing session id", errInvalidRefreshToken)
	}
	if strings.TrimSpace(claims.TokenID) == "" {
		return nil, fmt.Errorf("%w: missing token id", errInvalidRefreshToken)
	}

	return &refreshTokenSubject{
		UserID:    userID,
		SessionID: claims.SessionID,
		TokenID:   claims.TokenID,
	}, nil
}
