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

func TestCredentialStoreUsesIdentityAndAuthOwnedCredential(t *testing.T) {
	t.Parallel()

	client := enttest.Open(t, "sqlite3", "file:auth-owned-credentials?mode=memory&cache=shared&_fk=1")
	identity := identityProvider{users: map[string]moduleapi.CurrentUser{
		"alice": {ID: 7, Username: "alice", DisplayName: "Alice"},
	}}
	repo, err := newCredentialStore(client, identity)
	if err != nil {
		t.Fatalf("new credential store: %v", err)
	}
	changedAt := time.Now().UTC().Truncate(time.Second)
	if err := repo.SetPasswordHash(context.Background(), store.SetPasswordHashInput{UserID: 7, PasswordHash: "hash", MustChangePassword: true, ChangedAt: &changedAt}); err != nil {
		t.Fatalf("set password hash: %v", err)
	}

	credential, err := repo.GetUserCredentialByUsername(context.Background(), "alice")
	if err != nil {
		t.Fatalf("get credential: %v", err)
	}
	if credential.UserID != 7 || credential.Username != "alice" || credential.PasswordHash == nil || *credential.PasswordHash != "hash" || !credential.MustChangePassword {
		t.Fatalf("unexpected credential: %#v", credential)
	}
}

func TestCredentialStoreRequiresExistingProfile(t *testing.T) {
	t.Parallel()

	client := enttest.Open(t, "sqlite3", "file:auth-owned-credentials-missing-profile?mode=memory&cache=shared&_fk=1")
	repo, err := newCredentialStore(client, identityProvider{users: map[string]moduleapi.CurrentUser{}})
	if err != nil {
		t.Fatalf("new credential store: %v", err)
	}
	if err := repo.SetPasswordHash(context.Background(), store.SetPasswordHashInput{UserID: 99, PasswordHash: "hash"}); !errors.Is(err, errIdentityNotFound) {
		t.Fatalf("expected identity lookup failure, got %v", err)
	}
}

func TestSessionStoreRotatesAuthOwnedSession(t *testing.T) {
	t.Parallel()

	client := enttest.Open(t, "sqlite3", "file:auth-owned-sessions?mode=memory&cache=shared&_fk=1")
	repo, err := newSessionStore(client)
	if err != nil {
		t.Fatalf("new session store: %v", err)
	}
	now := time.Now().UTC().Truncate(time.Second)
	if _, err := repo.CreateRefreshSession(context.Background(), store.CreateRefreshSessionInput{UserID: 7, TokenID: "current", ExpiresAt: now.Add(time.Hour)}); err != nil {
		t.Fatalf("create refresh session: %v", err)
	}
	next, err := repo.RotateRefreshSession(context.Background(), store.RotateRefreshSessionInput{CurrentTokenID: "current", NewTokenID: "next", Now: now, RevokedAt: now, NewExpiresAt: now.Add(2 * time.Hour)})
	if err != nil {
		t.Fatalf("rotate refresh session: %v", err)
	}
	if next.UserID != 7 || next.TokenID != "next" {
		t.Fatalf("unexpected rotated session: %#v", next)
	}
	current, err := repo.GetRefreshSessionByTokenID(context.Background(), "current")
	if err != nil {
		t.Fatalf("get current session: %v", err)
	}
	if current.RevokedAt == nil || current.ReplacedByTokenID == nil || *current.ReplacedByTokenID != "next" {
		t.Fatalf("expected revoked current session, got %#v", current)
	}
}

var errIdentityNotFound = errors.New("identity not found")

type identityProvider struct {
	users map[string]moduleapi.CurrentUser
}

func (p identityProvider) LookupUserByUsername(_ context.Context, username string) (moduleapi.CurrentUser, error) {
	user, ok := p.users[username]
	if !ok {
		return moduleapi.CurrentUser{}, errIdentityNotFound
	}
	return user, nil
}

func (p identityProvider) GetCurrentUserByID(_ context.Context, userID uint64) (moduleapi.CurrentUser, error) {
	for _, user := range p.users {
		if user.ID == userID {
			return user, nil
		}
	}
	return moduleapi.CurrentUser{}, errIdentityNotFound
}

func (p identityProvider) EnsureDefaultAdminProfile(context.Context) (moduleapi.CurrentUser, error) {
	return moduleapi.CurrentUser{}, errIdentityNotFound
}
