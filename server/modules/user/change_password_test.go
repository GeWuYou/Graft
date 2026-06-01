package user

import (
	"context"
	"errors"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"

	"graft/server/internal/pluginapi"
	userstore "graft/server/modules/user/store"
)

type passwordChangeAtomicAuthRepository struct {
	*pluginTestAuthRepository
	changePasswordAndRevokeOtherRefreshSessions func(
		ctx context.Context,
		input userstore.ChangePasswordAndRevokeOtherRefreshSessionsInput,
	) error
}

func (r *passwordChangeAtomicAuthRepository) ChangePasswordAndRevokeOtherRefreshSessions(
	ctx context.Context,
	input userstore.ChangePasswordAndRevokeOtherRefreshSessionsInput,
) error {
	if r.changePasswordAndRevokeOtherRefreshSessions == nil {
		return errors.New("changePasswordAndRevokeOtherRefreshSessions callback is nil")
	}

	return r.changePasswordAndRevokeOtherRefreshSessions(ctx, input)
}

func TestChangeCurrentUserPasswordUsesAtomicRepositoryOperation(t *testing.T) {
	currentHashBytes, err := bcrypt.GenerateFromPassword([]byte("current-password"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("generate current password hash: %v", err)
	}
	currentHash := string(currentHashBytes)

	var called bool
	var received userstore.ChangePasswordAndRevokeOtherRefreshSessionsInput
	repo := &passwordChangeAtomicAuthRepository{
		pluginTestAuthRepository: &pluginTestAuthRepository{
			getUserCredentialByUsername: func(context.Context, string) (userstore.UserCredential, error) {
				return userstore.UserCredential{
					UserID:       7,
					Username:     "alice",
					PasswordHash: &currentHash,
				}, nil
			},
		},
		changePasswordAndRevokeOtherRefreshSessions: func(_ context.Context, input userstore.ChangePasswordAndRevokeOtherRefreshSessionsInput) error {
			called = true
			received = input
			return nil
		},
	}

	fixedNow := time.Date(2026, 5, 16, 8, 30, 0, 0, time.UTC)
	pair := adaptTestAuthRepository(repo)
	service := authService{
		auth:            pair.auth,
		passwordChanges: pair.passwordChanges,
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

func TestChangeCurrentUserPasswordRejectsMissingCurrentPassword(t *testing.T) {
	currentHashBytes, err := bcrypt.GenerateFromPassword([]byte("current-password"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("generate current password hash: %v", err)
	}
	currentHash := string(currentHashBytes)

	repo := &passwordChangeAtomicAuthRepository{
		pluginTestAuthRepository: &pluginTestAuthRepository{
			getUserCredentialByUsername: func(context.Context, string) (userstore.UserCredential, error) {
				return userstore.UserCredential{
					UserID:             7,
					Username:           "alice",
					PasswordHash:       &currentHash,
					MustChangePassword: true,
				}, nil
			},
		},
		changePasswordAndRevokeOtherRefreshSessions: func(context.Context, userstore.ChangePasswordAndRevokeOtherRefreshSessionsInput) error {
			t.Fatal("expected atomic password change operation not to be called")
			return nil
		},
	}

	pair := adaptTestAuthRepository(repo)
	service := authService{
		auth:            pair.auth,
		passwordChanges: pair.passwordChanges,
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
			UserID:    7,
			SessionID: "keep-current-session",
		},
	})

	err = service.ChangeCurrentUserPassword(ctx, "", "next-password-123")
	if !errors.Is(err, errCurrentPasswordRequired) {
		t.Fatalf("expected current password required error, got %v", err)
	}
}

func TestCompleteRequiredPasswordChangeAllowsRestrictedSessionWithoutCurrentPassword(t *testing.T) {
	currentHash, err := newPasswordHasher().Hash(defaultAdminPassword)
	if err != nil {
		t.Fatalf("hash default admin password: %v", err)
	}

	var called bool
	var received userstore.ChangePasswordAndRevokeOtherRefreshSessionsInput
	repo := &passwordChangeAtomicAuthRepository{
		pluginTestAuthRepository: &pluginTestAuthRepository{
			getUserCredentialByUsername: func(context.Context, string) (userstore.UserCredential, error) {
				return userstore.UserCredential{
					UserID:             9,
					Username:           defaultAdminUsername,
					PasswordHash:       &currentHash,
					MustChangePassword: true,
				}, nil
			},
		},
		changePasswordAndRevokeOtherRefreshSessions: func(_ context.Context, input userstore.ChangePasswordAndRevokeOtherRefreshSessionsInput) error {
			called = true
			received = input
			return nil
		},
	}

	fixedNow := time.Date(2026, 5, 16, 8, 30, 0, 0, time.UTC)
	pair := adaptTestAuthRepository(repo)
	service := authService{
		auth:            pair.auth,
		passwordChanges: pair.passwordChanges,
		passwords:       newPasswordHasher(),
		policy:          newPasswordPolicy(),
		refreshTokens:   &refreshTokenManager{now: func() time.Time { return fixedNow }},
	}

	ctx := pluginapi.WithRequestAuthContext(context.Background(), pluginapi.RequestAuthContext{
		User: &pluginapi.CurrentUser{
			ID:          9,
			Username:    defaultAdminUsername,
			DisplayName: defaultAdminDisplay,
		},
		Claims: &pluginapi.AccessTokenClaims{
			UserID:    9,
			SessionID: "keep-current-session",
		},
	})

	if err := service.CompleteRequiredPasswordChange(ctx, "next-password-123"); err != nil {
		t.Fatalf("complete required password change: %v", err)
	}
	if !called {
		t.Fatal("expected atomic password change repository operation to be called")
	}
	if received.CurrentTokenID != "keep-current-session" {
		t.Fatalf("expected current token id to be preserved, got %q", received.CurrentTokenID)
	}
	if received.MustChangePassword {
		t.Fatal("expected password change flow to clear must-change flag")
	}
}

func TestCompleteRequiredPasswordChangeRejectsNonRestrictedSession(t *testing.T) {
	currentHash, err := newPasswordHasher().Hash("current-password")
	if err != nil {
		t.Fatalf("hash current password: %v", err)
	}

	repo := &passwordChangeAtomicAuthRepository{
		pluginTestAuthRepository: &pluginTestAuthRepository{
			getUserCredentialByUsername: func(context.Context, string) (userstore.UserCredential, error) {
				return userstore.UserCredential{
					UserID:       7,
					Username:     "alice",
					PasswordHash: &currentHash,
				}, nil
			},
		},
		changePasswordAndRevokeOtherRefreshSessions: func(context.Context, userstore.ChangePasswordAndRevokeOtherRefreshSessionsInput) error {
			t.Fatal("expected atomic password change operation not to be called")
			return nil
		},
	}

	pair := adaptTestAuthRepository(repo)
	service := authService{
		auth:            pair.auth,
		passwordChanges: pair.passwordChanges,
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
			UserID:    7,
			SessionID: "keep-current-session",
		},
	})

	err = service.CompleteRequiredPasswordChange(ctx, "next-password-123")
	if !errors.Is(err, errRequiredPasswordChangeOnly) {
		t.Fatalf("expected required password change only error, got %v", err)
	}
}

func TestCompleteRequiredPasswordChangeRejectsPasswordReuse(t *testing.T) {
	currentPassword := "CurrentPassword123"
	currentHash, err := newPasswordHasher().Hash(currentPassword)
	if err != nil {
		t.Fatalf("hash current password: %v", err)
	}

	repo := &passwordChangeAtomicAuthRepository{
		pluginTestAuthRepository: &pluginTestAuthRepository{
			getUserCredentialByUsername: func(context.Context, string) (userstore.UserCredential, error) {
				return userstore.UserCredential{
					UserID:             7,
					Username:           "alice",
					PasswordHash:       &currentHash,
					MustChangePassword: true,
				}, nil
			},
		},
		changePasswordAndRevokeOtherRefreshSessions: func(context.Context, userstore.ChangePasswordAndRevokeOtherRefreshSessionsInput) error {
			t.Fatal("expected atomic password change operation not to be called")
			return nil
		},
	}

	pair := adaptTestAuthRepository(repo)
	service := authService{
		auth:            pair.auth,
		passwordChanges: pair.passwordChanges,
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
			UserID:    7,
			SessionID: "keep-current-session",
		},
	})

	err = service.CompleteRequiredPasswordChange(ctx, currentPassword)
	if !errors.Is(err, errPasswordReuseForbidden) {
		t.Fatalf("expected password reuse forbidden error, got %v", err)
	}
}

func TestChangeCurrentUserPasswordRequiresAtomicRepositoryOperation(t *testing.T) {
	currentHashBytes, err := bcrypt.GenerateFromPassword([]byte("current-password"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("generate current password hash: %v", err)
	}
	currentHash := string(currentHashBytes)

	service := authService{
		auth: &pluginTestAuthRepository{
			getUserCredentialByUsername: func(context.Context, string) (userstore.UserCredential, error) {
				return userstore.UserCredential{
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

func TestChangeCurrentUserPasswordRejectsMismatchedRequestPrincipal(t *testing.T) {
	currentHashBytes, err := bcrypt.GenerateFromPassword([]byte("current-password"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("generate current password hash: %v", err)
	}
	currentHash := string(currentHashBytes)

	repo := &passwordChangeAtomicAuthRepository{
		pluginTestAuthRepository: &pluginTestAuthRepository{
			getUserCredentialByUsername: func(context.Context, string) (userstore.UserCredential, error) {
				return userstore.UserCredential{
					UserID:       7,
					Username:     "alice",
					PasswordHash: &currentHash,
				}, nil
			},
		},
		changePasswordAndRevokeOtherRefreshSessions: func(context.Context, userstore.ChangePasswordAndRevokeOtherRefreshSessionsInput) error {
			t.Fatal("expected atomic password change operation not to be called")
			return nil
		},
	}

	pair := adaptTestAuthRepository(repo)
	service := authService{
		auth:            pair.auth,
		passwordChanges: pair.passwordChanges,
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
