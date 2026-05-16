package user

import (
	"context"
	"errors"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"

	"graft/server/internal/pluginapi"
	"graft/server/internal/store"
)

type passwordChangeAtomicAuthRepository struct {
	*pluginTestAuthRepository
	changePasswordAndRevokeOtherRefreshSessions func(
		ctx context.Context,
		input store.ChangePasswordAndRevokeOtherRefreshSessionsInput,
	) error
}

func (r *passwordChangeAtomicAuthRepository) ChangePasswordAndRevokeOtherRefreshSessions(
	ctx context.Context,
	input store.ChangePasswordAndRevokeOtherRefreshSessionsInput,
) error {
	if r.changePasswordAndRevokeOtherRefreshSessions == nil {
		return errors.New("changePasswordAndRevokeOtherRefreshSessions callback is nil")
	}

	return r.changePasswordAndRevokeOtherRefreshSessions(ctx, input)
}

// TestChangeCurrentUserPasswordUsesAtomicRepositoryOperation 验证改密路径改为显式依赖原子仓储写操作。
func TestChangeCurrentUserPasswordUsesAtomicRepositoryOperation(t *testing.T) {
	currentHashBytes, err := bcrypt.GenerateFromPassword([]byte("current-password"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("generate current password hash: %v", err)
	}
	currentHash := string(currentHashBytes)

	var called bool
	var received store.ChangePasswordAndRevokeOtherRefreshSessionsInput
	repo := &passwordChangeAtomicAuthRepository{
		pluginTestAuthRepository: &pluginTestAuthRepository{
			getUserCredentialByUsername: func(context.Context, string) (store.UserCredential, error) {
				return store.UserCredential{
					UserID:       7,
					Username:     "alice",
					PasswordHash: &currentHash,
				}, nil
			},
		},
		changePasswordAndRevokeOtherRefreshSessions: func(_ context.Context, input store.ChangePasswordAndRevokeOtherRefreshSessionsInput) error {
			called = true
			received = input
			return nil
		},
	}

	fixedNow := time.Date(2026, 5, 16, 8, 30, 0, 0, time.UTC)
	service := authService{
		auth:            repo,
		passwordChanges: repo,
		passwords:       newPasswordHasher(),
		policy:          newPasswordPolicy(),
		refreshTokens:   &refreshTokenManager{now: func() time.Time { return fixedNow }},
	}

	ctx := pluginapi.WithRequestAuthContext(context.Background(), pluginapi.RequestAuthContext{
		User: &pluginapi.CurrentUser{
			ID:          7,
			Username:    "alice",
			DisplayName: "Alice",
		},
		Claims: &pluginapi.AccessTokenClaims{
			UserID:    7,
			SessionID: "keep-current-session",
		},
	})

	if err := service.ChangeCurrentUserPassword(ctx, "current-password", "next-password-123"); err != nil {
		t.Fatalf("change current user password: %v", err)
	}
	if !called {
		t.Fatal("expected atomic password change repository operation to be called")
	}
	if received.UserID != 7 {
		t.Fatalf("expected user id 7, got %d", received.UserID)
	}
	if received.CurrentTokenID != "keep-current-session" {
		t.Fatalf("expected current token id to be preserved, got %q", received.CurrentTokenID)
	}
	if received.ChangedAt != fixedNow {
		t.Fatalf("expected fixed changed time, got %v", received.ChangedAt)
	}
	if received.MustChangePassword {
		t.Fatal("expected password change flow to clear must-change flag")
	}
	if received.PasswordHash == "" {
		t.Fatal("expected new password hash to be populated")
	}
	if received.PasswordHash == currentHash {
		t.Fatal("expected new password hash to differ from current hash")
	}
}

// TestChangeCurrentUserPasswordRequiresAtomicRepositoryOperation 验证缺失原子仓储能力时显式失败。
func TestChangeCurrentUserPasswordRequiresAtomicRepositoryOperation(t *testing.T) {
	currentHashBytes, err := bcrypt.GenerateFromPassword([]byte("current-password"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("generate current password hash: %v", err)
	}
	currentHash := string(currentHashBytes)

	service := authService{
		auth: &pluginTestAuthRepository{
			getUserCredentialByUsername: func(context.Context, string) (store.UserCredential, error) {
				return store.UserCredential{
					UserID:       7,
					Username:     "alice",
					PasswordHash: &currentHash,
				}, nil
			},
		},
		passwords:     newPasswordHasher(),
		policy:        newPasswordPolicy(),
		refreshTokens: &refreshTokenManager{now: func() time.Time { return time.Date(2026, 5, 16, 8, 30, 0, 0, time.UTC) }},
	}

	ctx := pluginapi.WithRequestAuthContext(context.Background(), pluginapi.RequestAuthContext{
		User: &pluginapi.CurrentUser{
			ID:          7,
			Username:    "alice",
			DisplayName: "Alice",
		},
		Claims: &pluginapi.AccessTokenClaims{
			UserID:    7,
			SessionID: "keep-current-session",
		},
	})

	err = service.ChangeCurrentUserPassword(ctx, "current-password", "next-password-123")
	if err == nil || err.Error() != "auth repository does not support atomic password change" {
		t.Fatalf("expected explicit atomic capability error, got %v", err)
	}
}

// TestChangeCurrentUserPasswordRejectsMismatchedRequestPrincipal 验证改密路径不会混用
// 不一致的 request user 与 access token claims。
func TestChangeCurrentUserPasswordRejectsMismatchedRequestPrincipal(t *testing.T) {
	currentHashBytes, err := bcrypt.GenerateFromPassword([]byte("current-password"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("generate current password hash: %v", err)
	}
	currentHash := string(currentHashBytes)

	repo := &passwordChangeAtomicAuthRepository{
		pluginTestAuthRepository: &pluginTestAuthRepository{
			getUserCredentialByUsername: func(context.Context, string) (store.UserCredential, error) {
				return store.UserCredential{
					UserID:       7,
					Username:     "alice",
					PasswordHash: &currentHash,
				}, nil
			},
		},
		changePasswordAndRevokeOtherRefreshSessions: func(context.Context, store.ChangePasswordAndRevokeOtherRefreshSessionsInput) error {
			t.Fatal("expected atomic password change operation not to be called")
			return nil
		},
	}

	service := authService{
		auth:            repo,
		passwordChanges: repo,
		passwords:       newPasswordHasher(),
		policy:          newPasswordPolicy(),
		refreshTokens:   &refreshTokenManager{now: func() time.Time { return time.Date(2026, 5, 16, 8, 30, 0, 0, time.UTC) }},
	}

	ctx := pluginapi.WithRequestAuthContext(context.Background(), pluginapi.RequestAuthContext{
		User: &pluginapi.CurrentUser{
			ID:          7,
			Username:    "alice",
			DisplayName: "Alice",
		},
		Claims: &pluginapi.AccessTokenClaims{
			UserID:    8,
			SessionID: "keep-current-session",
		},
	})

	err = service.ChangeCurrentUserPassword(ctx, "current-password", "next-password-123")
	if !errors.Is(err, pluginapi.ErrUnauthenticated) {
		t.Fatalf("expected unauthenticated error for mismatched principal, got %v", err)
	}
}
