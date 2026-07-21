package auth

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"graft/server/internal/moduleapi"
	authstore "graft/server/modules/auth/store"
)

func TestPersonalAccessTokenLifecycleKeepsSecretOneTimeAndTracksUse(t *testing.T) {
	now := time.Date(2026, time.July, 21, 9, 30, 0, 0, time.UTC)
	user := moduleapi.CurrentUser{ID: 42, Username: "alice", DisplayName: "Alice"}
	store := &personalAccessTokenTestStore{records: make(map[string]authstore.PersonalAccessToken)}
	service := authService{
		identity:       runtimeIdentityProvider{users: map[uint64]moduleapi.CurrentUser{user.ID: user}},
		personalTokens: store,
		now:            func() time.Time { return now },
	}
	ctx := moduleapi.WithRequestAuthContext(context.Background(), moduleapi.RequestAuthContext{User: &user})

	issued := issuePersonalAccessToken(ctx, t, service, now)
	assertIssuedPersonalAccessToken(t, issued, store)
	assertPersonalAccessTokenAuthentication(t, service, issued, user, store, now)
	assertPersonalAccessTokenRevocation(ctx, t, service, issued)
}

func issuePersonalAccessToken(ctx context.Context, t *testing.T, service authService, now time.Time) moduleapi.PersonalAccessTokenIssued {
	t.Helper()
	issued, err := service.CreateCurrentUserPersonalAccessToken(ctx, moduleapi.PersonalAccessTokenCreateInput{
		Name:      "mcp development",
		Scopes:    []string{"audit.read"},
		ExpiresAt: now.Add(time.Hour),
	})
	if err != nil {
		t.Fatalf("create personal access token: %v", err)
	}
	return issued
}

func assertIssuedPersonalAccessToken(t *testing.T, issued moduleapi.PersonalAccessTokenIssued, store *personalAccessTokenTestStore) {
	t.Helper()
	if !strings.HasPrefix(issued.Token, personalAccessTokenPrefix) {
		t.Fatalf("expected token prefix %q, got %q", personalAccessTokenPrefix, issued.Token)
	}
	if issued.Summary.TokenPrefix == issued.Token || !strings.HasPrefix(issued.Token, issued.Summary.TokenPrefix) {
		t.Fatalf("summary must retain only a non-secret display prefix: %#v", issued.Summary)
	}
	if store.created.SecretHash != hashPersonalAccessToken(issued.Token) || store.created.SecretHash == issued.Token {
		t.Fatalf("store must retain only the token hash, got %#v", store.created)
	}
}

func assertPersonalAccessTokenAuthentication(
	t *testing.T,
	service authService,
	issued moduleapi.PersonalAccessTokenIssued,
	user moduleapi.CurrentUser,
	store *personalAccessTokenTestStore,
	now time.Time,
) {
	t.Helper()
	caller, err := service.AuthenticatePersonalAccessToken(context.Background(), issued.Token)
	if err != nil {
		t.Fatalf("authenticate issued token: %v", err)
	}
	if caller.TokenID != issued.Summary.ID || caller.User.ID != user.ID || len(caller.Scopes) != 1 || caller.Scopes[0] != "audit.read" {
		t.Fatalf("unexpected caller: %#v", caller)
	}
	if store.lastUsedAt == nil || !store.lastUsedAt.Equal(now) {
		t.Fatalf("expected last-used time %s, got %#v", now, store.lastUsedAt)
	}
}

func assertPersonalAccessTokenRevocation(
	ctx context.Context,
	t *testing.T,
	service authService,
	issued moduleapi.PersonalAccessTokenIssued,
) {
	t.Helper()
	if err := service.RevokeCurrentUserPersonalAccessToken(ctx, issued.Summary.ID); err != nil {
		t.Fatalf("revoke personal access token: %v", err)
	}
	if _, err := service.AuthenticatePersonalAccessToken(context.Background(), issued.Token); !errors.Is(err, moduleapi.ErrInvalidPersonalAccessToken) {
		t.Fatalf("revoked token authentication error = %v, want invalid token", err)
	}
}

func TestPersonalAccessTokenRejectsUnboundedExpiry(t *testing.T) {
	now := time.Date(2026, time.July, 21, 9, 30, 0, 0, time.UTC)
	user := moduleapi.CurrentUser{ID: 42}
	service := authService{
		identity:       runtimeIdentityProvider{users: map[uint64]moduleapi.CurrentUser{user.ID: user}},
		personalTokens: &personalAccessTokenTestStore{records: make(map[string]authstore.PersonalAccessToken)},
		now:            func() time.Time { return now },
	}
	ctx := moduleapi.WithRequestAuthContext(context.Background(), moduleapi.RequestAuthContext{User: &user})

	_, err := service.CreateCurrentUserPersonalAccessToken(ctx, moduleapi.PersonalAccessTokenCreateInput{
		Name:   "missing expiry",
		Scopes: []string{"audit.read"},
	})
	if !errors.Is(err, errInvalidPersonalAccessTokenInput) {
		t.Fatalf("create unbounded personal access token = %v, want invalid input", err)
	}
}

type personalAccessTokenTestStore struct {
	records    map[string]authstore.PersonalAccessToken
	created    authstore.PersonalAccessToken
	lastUsedAt *time.Time
}

func (s *personalAccessTokenTestStore) CreatePersonalAccessToken(_ context.Context, input authstore.CreatePersonalAccessTokenInput) (authstore.PersonalAccessToken, error) {
	record := authstore.PersonalAccessToken{
		ID:          11,
		UserID:      input.UserID,
		Name:        input.Name,
		TokenPrefix: input.TokenPrefix,
		SecretHash:  input.SecretHash,
		Scopes:      append([]string(nil), input.Scopes...),
		ExpiresAt:   input.ExpiresAt,
		CreatedAt:   input.ExpiresAt.Add(-time.Hour),
	}
	s.records[input.SecretHash] = record
	s.created = record
	return record, nil
}

func (s *personalAccessTokenTestStore) GetPersonalAccessTokenBySecretHash(_ context.Context, secretHash string) (authstore.PersonalAccessToken, error) {
	record, ok := s.records[secretHash]
	if !ok {
		return authstore.PersonalAccessToken{}, authstore.ErrPersonalAccessTokenNotFound
	}
	return record, nil
}

func (s *personalAccessTokenTestStore) ListPersonalAccessTokensByUserID(_ context.Context, _ authstore.ListPersonalAccessTokensByUserIDInput) ([]authstore.PersonalAccessToken, error) {
	return nil, nil
}

func (s *personalAccessTokenTestStore) RevokePersonalAccessTokenByUserID(_ context.Context, input authstore.RevokePersonalAccessTokenByUserIDInput) error {
	for hash, record := range s.records {
		if record.UserID == input.UserID && record.ID == input.TokenID {
			revokedAt := input.RevokedAt
			record.RevokedAt = &revokedAt
			s.records[hash] = record
		}
	}
	return nil
}

func (s *personalAccessTokenTestStore) MarkPersonalAccessTokenUsed(_ context.Context, tokenID uint64, usedAt time.Time) error {
	for hash, record := range s.records {
		if record.ID == tokenID {
			lastUsedAt := usedAt
			record.LastUsedAt = &lastUsedAt
			s.records[hash] = record
			s.lastUsedAt = &lastUsedAt
			return nil
		}
	}
	return authstore.ErrPersonalAccessTokenNotFound
}

var _ authstore.PersonalAccessTokenStore = (*personalAccessTokenTestStore)(nil)
