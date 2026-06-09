// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package user

import (
	"errors"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"graft/server/internal/config"
)

func TestAccessTokenManagerIssuesAndParsesHS256Token(t *testing.T) {
	manager, err := newAccessTokenManager(config.AuthConfig{
		AccessTokenTTL: 15 * time.Minute,
		SigningKey:     "test-signing-key",
	})
	if err != nil {
		t.Fatalf("new access token manager: %v", err)
	}
	fixedNow := time.Date(2026, 5, 14, 10, 0, 0, 0, time.UTC)
	manager.now = func() time.Time { return fixedNow }

	token, claims, err := manager.Issue(accessTokenSubject{
		UserID:       7,
		SessionID:    "session-1",
		TokenVersion: 2,
	})
	if err != nil {
		t.Fatalf("issue access token: %v", err)
	}

	parsed, err := manager.Parse(token)
	if err != nil {
		t.Fatalf("parse access token: %v", err)
	}
	if *parsed != claims {
		t.Fatalf("expected parsed claims %#v, got %#v", claims, parsed)
	}
}

func TestAccessTokenManagerRejectsExpiredToken(t *testing.T) {
	manager, err := newAccessTokenManager(config.AuthConfig{
		AccessTokenTTL: time.Minute,
		SigningKey:     "test-signing-key",
	})
	if err != nil {
		t.Fatalf("new access token manager: %v", err)
	}
	expiredClaims := accessTokenJWTClaims{
		SessionID: "session-1",
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "7",
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Minute)),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Minute)),
		},
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, expiredClaims).SignedString([]byte("test-signing-key"))
	if err != nil {
		t.Fatalf("sign expired token: %v", err)
	}

	_, err = manager.Parse(token)
	if !errors.Is(err, errExpiredAccessToken) {
		t.Fatalf("expected expired token error, got %v", err)
	}
}

func TestAccessTokenManagerRejectsUnexpectedSigningMethod(t *testing.T) {
	manager, err := newAccessTokenManager(config.AuthConfig{
		AccessTokenTTL: time.Minute,
		SigningKey:     "test-signing-key",
	})
	if err != nil {
		t.Fatalf("new access token manager: %v", err)
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS384, accessTokenJWTClaims{
		SessionID: "session-1",
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "7",
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Minute)),
		},
	}).SignedString([]byte("test-signing-key"))
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}

	_, err = manager.Parse(token)
	if !errors.Is(err, errInvalidAccessToken) {
		t.Fatalf("expected invalid token error, got %v", err)
	}
}

func TestAccessTokenManagerRequiresSessionID(t *testing.T) {
	manager, err := newAccessTokenManager(config.AuthConfig{
		AccessTokenTTL: time.Minute,
		SigningKey:     "test-signing-key",
	})
	if err != nil {
		t.Fatalf("new access token manager: %v", err)
	}

	_, _, err = manager.Issue(accessTokenSubject{UserID: 7})
	if !errors.Is(err, errSessionIDRequired) {
		t.Fatalf("expected missing session id error, got %v", err)
	}
}
