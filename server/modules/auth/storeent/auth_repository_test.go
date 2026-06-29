package storeent

import (
	"context"
	"errors"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"

	authstore "graft/server/modules/auth/store"
	"graft/server/modules/user/ent/enttest"
	userstore "graft/server/modules/user/store"
)

func TestAuthRepositorySetPasswordHashRejectsSoftDeletedUser(t *testing.T) {
	t.Parallel()

	client := enttest.Open(t, "sqlite3", "file:auth-storeent-set-password?mode=memory&cache=shared&_fk=1")
	repo, err := newAuthRepository(client)
	if err != nil {
		t.Fatalf("new auth repository: %v", err)
	}

	ctx := context.Background()
	changedAt := time.Now().UTC().Truncate(time.Second)
	record, err := client.User.Create().
		SetUsername("deleted-set-password").
		SetDisplay("Deleted Set Password").
		SetPasswordHash("original-hash").
		SetMustChangePassword(false).
		SetPasswordChangedAt(changedAt.Add(-time.Hour)).
		SetDeletedAt(changedAt.Unix()).
		Save(ctx)
	if err != nil {
		t.Fatalf("seed deleted user: %v", err)
	}

	err = repo.SetPasswordHash(ctx, authstore.SetPasswordHashInput{
		UserID:             toStoreID(record.ID),
		PasswordHash:       "updated-hash",
		MustChangePassword: true,
		ChangedAt:          &changedAt,
	})
	if !errors.Is(err, userstore.ErrUserNotFound) {
		t.Fatalf("expected ErrUserNotFound, got %v", err)
	}

	refreshed, err := client.User.Get(ctx, record.ID)
	if err != nil {
		t.Fatalf("reload user: %v", err)
	}
	if refreshed.PasswordHash == nil || *refreshed.PasswordHash != "original-hash" {
		t.Fatalf("expected password hash to remain unchanged, got %#v", refreshed.PasswordHash)
	}
	if refreshed.MustChangePassword {
		t.Fatalf("expected must_change_password to remain false")
	}
	if refreshed.PasswordChangedAt == nil || !refreshed.PasswordChangedAt.Equal(changedAt.Add(-time.Hour)) {
		t.Fatalf("expected password_changed_at to remain unchanged, got %#v", refreshed.PasswordChangedAt)
	}
}

func TestAuthRepositoryChangePasswordRejectsSoftDeletedUser(t *testing.T) {
	t.Parallel()

	client := enttest.Open(t, "sqlite3", "file:auth-storeent-change-password?mode=memory&cache=shared&_fk=1")
	repo, err := newAuthRepository(client)
	if err != nil {
		t.Fatalf("new auth repository: %v", err)
	}

	ctx := context.Background()
	changedAt := time.Now().UTC().Truncate(time.Second)
	record, err := client.User.Create().
		SetUsername("deleted-change-password").
		SetDisplay("Deleted Change Password").
		SetPasswordHash("original-hash").
		SetMustChangePassword(false).
		SetPasswordChangedAt(changedAt.Add(-2 * time.Hour)).
		SetDeletedAt(changedAt.Unix()).
		Save(ctx)
	if err != nil {
		t.Fatalf("seed deleted user: %v", err)
	}
	if _, err := client.RefreshSession.Create().
		SetUserID(record.ID).
		SetTokenID("current-token").
		SetExpiresAt(changedAt.Add(24 * time.Hour)).
		Save(ctx); err != nil {
		t.Fatalf("seed current refresh session: %v", err)
	}
	otherSession, err := client.RefreshSession.Create().
		SetUserID(record.ID).
		SetTokenID("other-token").
		SetExpiresAt(changedAt.Add(24 * time.Hour)).
		Save(ctx)
	if err != nil {
		t.Fatalf("seed other refresh session: %v", err)
	}

	err = repo.ChangePasswordAndRevokeOtherRefreshSessions(ctx, authstore.ChangePasswordAndRevokeOtherRefreshSessionsInput{
		UserID:             toStoreID(record.ID),
		PasswordHash:       "updated-hash",
		MustChangePassword: true,
		ChangedAt:          changedAt,
		CurrentTokenID:     "current-token",
	})
	if !errors.Is(err, userstore.ErrUserNotFound) {
		t.Fatalf("expected ErrUserNotFound, got %v", err)
	}

	refreshedUser, err := client.User.Get(ctx, record.ID)
	if err != nil {
		t.Fatalf("reload user: %v", err)
	}
	if refreshedUser.PasswordHash == nil || *refreshedUser.PasswordHash != "original-hash" {
		t.Fatalf("expected password hash to remain unchanged, got %#v", refreshedUser.PasswordHash)
	}
	if refreshedUser.MustChangePassword {
		t.Fatalf("expected must_change_password to remain false")
	}
	if refreshedUser.PasswordChangedAt == nil || !refreshedUser.PasswordChangedAt.Equal(changedAt.Add(-2*time.Hour)) {
		t.Fatalf("expected password_changed_at to remain unchanged, got %#v", refreshedUser.PasswordChangedAt)
	}

	refreshedSession, err := client.RefreshSession.Get(ctx, otherSession.ID)
	if err != nil {
		t.Fatalf("reload other refresh session: %v", err)
	}
	if refreshedSession.RevokedAt != nil {
		t.Fatalf("expected other refresh session to remain active, got revoked_at=%v", refreshedSession.RevokedAt)
	}
}
