package storeent

import (
	"context"
	"errors"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"graft/server/internal/moduleapi"
	"graft/server/modules/auth/ent/enttest"
	"graft/server/modules/auth/store"
)

func TestAuthTransactionRunnerRollsBackFailedRefreshRotation(t *testing.T) {
	t.Parallel()

	client := enttest.Open(t, "sqlite3", "file:auth-transaction-rotation?mode=memory&cache=shared&_fk=1")
	identity := identityProvider{users: map[string]moduleapi.CurrentUser{
		"alice": {ID: 7, Username: "alice"},
	}}
	credentials, err := newCredentialStore(client, identity)
	if err != nil {
		t.Fatalf("new credential store: %v", err)
	}
	sessions, err := newSessionStore(client)
	if err != nil {
		t.Fatalf("new session store: %v", err)
	}
	now := time.Now().UTC().Truncate(time.Second)
	if _, err := sessions.CreateRefreshSession(context.Background(), store.CreateRefreshSessionInput{UserID: 7, TokenID: "current", ExpiresAt: now.Add(time.Hour)}); err != nil {
		t.Fatalf("create current session: %v", err)
	}
	if _, err := sessions.CreateRefreshSession(context.Background(), store.CreateRefreshSessionInput{UserID: 8, TokenID: "duplicate", ExpiresAt: now.Add(time.Hour)}); err != nil {
		t.Fatalf("create duplicate token session: %v", err)
	}

	err = credentials.RunInTransaction(context.Background(), func(ctx context.Context, _ store.CredentialStore, txSessions store.SessionStore) error {
		_, rotateErr := txSessions.RotateRefreshSession(ctx, store.RotateRefreshSessionInput{
			CurrentTokenID: "current",
			NewTokenID:     "duplicate",
			Now:            now,
			RevokedAt:      now,
			NewExpiresAt:   now.Add(2 * time.Hour),
		})
		return rotateErr
	})
	if err == nil {
		t.Fatal("rotation with duplicate token must fail")
	}

	current, err := sessions.GetRefreshSessionByTokenID(context.Background(), "current")
	if err != nil {
		t.Fatalf("get current session after rollback: %v", err)
	}
	if current.RevokedAt != nil || current.ReplacedByTokenID != nil {
		t.Fatalf("failed rotation must not persist current-session mutation: %#v", current)
	}
}

func TestAuthTransactionRunnerRollsBackPasswordAndSessionWrites(t *testing.T) {
	t.Parallel()

	client := enttest.Open(t, "sqlite3", "file:auth-transaction-password?mode=memory&cache=shared&_fk=1")
	identity := identityProvider{users: map[string]moduleapi.CurrentUser{
		"alice": {ID: 7, Username: "alice"},
	}}
	credentials, err := newCredentialStore(client, identity)
	if err != nil {
		t.Fatalf("new credential store: %v", err)
	}
	sessions, err := newSessionStore(client)
	if err != nil {
		t.Fatalf("new session store: %v", err)
	}
	now := time.Now().UTC().Truncate(time.Second)
	if err := credentials.SetPasswordHash(context.Background(), store.SetPasswordHashInput{UserID: 7, PasswordHash: "before", ChangedAt: &now}); err != nil {
		t.Fatalf("create credential: %v", err)
	}
	if _, err := sessions.CreateRefreshSession(context.Background(), store.CreateRefreshSessionInput{UserID: 7, TokenID: "current", ExpiresAt: now.Add(time.Hour)}); err != nil {
		t.Fatalf("create current session: %v", err)
	}

	err = credentials.RunInTransaction(context.Background(), func(ctx context.Context, txCredentials store.CredentialStore, txSessions store.SessionStore) error {
		if err := txCredentials.SetPasswordHash(ctx, store.SetPasswordHashInput{UserID: 7, PasswordHash: "after", MustChangePassword: true, ChangedAt: &now}); err != nil {
			return err
		}
		if err := txSessions.RevokeRefreshSessionsByUserID(ctx, store.RevokeRefreshSessionsByUserIDInput{UserID: 7, RevokedAt: now}); err != nil {
			return err
		}
		return errors.New("simulate auth workflow failure")
	})
	if err == nil {
		t.Fatal("failing transaction callback must return an error")
	}

	credential, err := credentials.GetUserCredentialByUsername(context.Background(), "alice")
	if err != nil {
		t.Fatalf("get credential after rollback: %v", err)
	}
	if credential.PasswordHash == nil || *credential.PasswordHash != "before" || credential.MustChangePassword {
		t.Fatalf("failed transaction must not persist password mutation: %#v", credential)
	}
	current, err := sessions.GetRefreshSessionByTokenID(context.Background(), "current")
	if err != nil {
		t.Fatalf("get session after rollback: %v", err)
	}
	if current.RevokedAt != nil {
		t.Fatalf("failed transaction must not persist session revocation: %#v", current)
	}
}
