// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package user

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"graft/server/internal/config"
	"graft/server/internal/moduleapi"
	authruntime "graft/server/modules/auth"
)

var (
	errTokenSigningKeyRequired = authruntime.ErrTokenSigningKeyRequired
	errSessionIDRequired       = authruntime.ErrSessionIDRequired
	errInvalidAccessToken      = authruntime.ErrInvalidAccessToken
	errExpiredAccessToken      = authruntime.ErrExpiredAccessToken
)

type accessTokenSubject = authruntime.AccessTokenSubject

// accessTokenManager keeps the legacy user-module call surface stable during
// Phase 2 while canonical ownership moves to server/modules/auth.
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

func (m *accessTokenManager) Issue(subject accessTokenSubject) (string, moduleapi.AccessTokenClaims, error) {
	if subject.UserID == 0 {
		return "", moduleapi.AccessTokenClaims{}, fmt.Errorf("user id is required")
	}
	if strings.TrimSpace(subject.SessionID) == "" {
		return "", moduleapi.AccessTokenClaims{}, errSessionIDRequired
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

func (m *accessTokenManager) Parse(token string) (*moduleapi.AccessTokenClaims, error) {
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

	return &moduleapi.AccessTokenClaims{
		UserID:       userID,
		SessionID:    claims.SessionID,
		TokenVersion: claims.TokenVersion,
		IssuedAt:     claims.IssuedAt.UTC(),
		ExpiresAt:    claims.ExpiresAt.UTC(),
	}, nil
}
