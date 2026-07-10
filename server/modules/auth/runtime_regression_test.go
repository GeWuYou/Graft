package auth

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"graft/server/internal/config"
	"graft/server/internal/moduleapi"
	authstore "graft/server/modules/auth/store"
)

const (
	runtimeTestPassword    = "CorrectHorse9"
	runtimeTestNewPassword = "NewCorrectHorse9"
)

func TestAuthRuntimeLoginRefreshAndAccessSessionValidation(t *testing.T) {
	service, stores, user := newRuntimeRegressionService(t, false)

	login, err := service.LoginWithRefresh(context.Background(), user.Username, runtimeTestPassword)
	if err != nil {
		t.Fatalf("login with refresh: %v", err)
	}
	oldClaims, err := service.ParseAccessToken(context.Background(), login.AccessToken)
	if err != nil {
		t.Fatalf("validate access token before rotation: %v", err)
	}

	rotated, err := service.RefreshWithRotation(context.Background(), login.RefreshToken)
	if err != nil {
		t.Fatalf("refresh with rotation: %v", err)
	}
	if rotated.RefreshToken == login.RefreshToken {
		t.Fatal("refresh rotation must issue a new token")
	}
	if _, err := service.ParseAccessToken(context.Background(), login.AccessToken); !errors.Is(err, moduleapi.ErrInvalidAccessToken) {
		t.Fatalf("old access token must be invalid after rotation, got %v", err)
	}
	if _, err := service.ParseAccessToken(context.Background(), rotated.AccessToken); err != nil {
		t.Fatalf("validate rotated access token: %v", err)
	}
	if _, err := service.RefreshWithRotation(context.Background(), login.RefreshToken); !errors.Is(err, errInvalidRefreshToken) {
		t.Fatalf("rotated refresh token must be rejected, got %v", err)
	}

	old := stores.sessions[oldClaims.SessionID]
	if old.RevokedAt == nil || old.ReplacedByTokenID == nil {
		t.Fatalf("rotated session must be revoked and linked: %#v", old)
	}
}

func TestAuthRuntimeLogoutAndTargetedSessionRevokeInvalidateAccessTokens(t *testing.T) {
	service, _, user := newRuntimeRegressionService(t, false)
	first, err := service.LoginWithRefresh(context.Background(), user.Username, runtimeTestPassword)
	if err != nil {
		t.Fatalf("first login: %v", err)
	}
	second, err := service.LoginWithRefresh(context.Background(), user.Username, runtimeTestPassword)
	if err != nil {
		t.Fatalf("second login: %v", err)
	}
	firstClaims, err := service.tokens.Parse(first.AccessToken)
	if err != nil {
		t.Fatalf("parse first access token: %v", err)
	}
	secondClaims, err := service.tokens.Parse(second.AccessToken)
	if err != nil {
		t.Fatalf("parse second access token: %v", err)
	}

	if err := service.LogoutCurrentSession(context.Background(), first.RefreshToken); err != nil {
		t.Fatalf("logout current session: %v", err)
	}
	if _, err := service.ParseAccessToken(context.Background(), first.AccessToken); !errors.Is(err, moduleapi.ErrInvalidAccessToken) {
		t.Fatalf("logged-out access token must be invalid, got %v", err)
	}
	if _, err := service.ParseAccessToken(context.Background(), second.AccessToken); err != nil {
		t.Fatalf("other active access token unexpectedly invalid: %v", err)
	}

	result, err := service.RevokeSessionByUserID(context.Background(), user.ID, secondClaims.SessionID)
	if err != nil || !result.Revoked {
		t.Fatalf("revoke targeted session = %#v, %v", result, err)
	}
	if _, err := service.ParseAccessToken(context.Background(), second.AccessToken); !errors.Is(err, moduleapi.ErrInvalidAccessToken) {
		t.Fatalf("target-revoked access token must be invalid, got %v", err)
	}
	if firstClaims.SessionID == secondClaims.SessionID {
		t.Fatal("independent logins must create independent sessions")
	}
}

func TestAuthRuntimeRestrictedPasswordChangeAndBootstrapState(t *testing.T) {
	service, stores, user := newRuntimeRegressionService(t, true)
	login, err := service.LoginWithRefresh(context.Background(), user.Username, runtimeTestPassword)
	if err != nil {
		t.Fatalf("restricted login: %v", err)
	}
	if !login.MustChangePassword {
		t.Fatal("restricted credential must be reported by login")
	}
	if _, err := service.RefreshWithRotation(context.Background(), login.RefreshToken); !errors.Is(err, errRequiredPasswordChangeOnly) {
		t.Fatalf("restricted session refresh must be rejected, got %v", err)
	}
	claims, err := service.tokens.Parse(login.AccessToken)
	if err != nil {
		t.Fatalf("parse access token: %v", err)
	}
	requestContext := moduleapi.WithRequestAuthContext(context.Background(), moduleapi.RequestAuthContext{User: &user, Claims: claims})
	flow := authFlowBridge{
		auth:      service,
		bootstrap: runtimeBootstrapProvider{payload: moduleapi.AuthBootstrapPayload{User: user, Permissions: []string{"user:read"}}},
	}

	restricted, err := flow.IsRestrictedPasswordChangeSession(requestContext)
	if err != nil || !restricted {
		t.Fatalf("restricted password state = %t, %v", restricted, err)
	}
	payload, err := flow.ReadBootstrapPayload(requestContext, httptest.NewRequest("GET", "/api/auth/bootstrap", nil))
	if err != nil || !payload.MustChangePassword {
		t.Fatalf("bootstrap must include auth-owned restricted state: %#v, %v", payload, err)
	}
	if err := flow.CompleteRequiredPasswordChange(requestContext, runtimeTestNewPassword); err != nil {
		t.Fatalf("complete required password change: %v", err)
	}
	restricted, err = flow.IsRestrictedPasswordChangeSession(requestContext)
	if err != nil || restricted {
		t.Fatalf("password change must clear restricted state = %t, %v", restricted, err)
	}
	if stores.credentials[user.Username].MustChangePassword {
		t.Fatal("credential store must persist cleared restricted state")
	}
}

func TestAuthRuntimeRejectsMissingIdentityDuringLoginAndRefresh(t *testing.T) {
	service, _, user := newRuntimeRegressionService(t, false)
	login, err := service.LoginWithRefresh(context.Background(), user.Username, runtimeTestPassword)
	if err != nil {
		t.Fatalf("login with refresh: %v", err)
	}

	service.identity = runtimeIdentityProvider{users: map[uint64]moduleapi.CurrentUser{}}
	if _, err := service.Login(context.Background(), user.Username, runtimeTestPassword); !errors.Is(err, errInvalidLoginCredentials) {
		t.Fatalf("login with missing identity = %v, want invalid credentials", err)
	}
	if _, err := service.RefreshWithRotation(context.Background(), login.RefreshToken); !errors.Is(err, errInvalidRefreshToken) {
		t.Fatalf("refresh with missing identity = %v, want invalid refresh token", err)
	}
}

func newRuntimeRegressionService(t *testing.T, mustChangePassword bool) (*authService, *runtimeAuthStores, moduleapi.CurrentUser) {
	t.Helper()
	user := moduleapi.CurrentUser{ID: 42, Username: "alice", DisplayName: "Alice"}
	hash, err := newPasswordHasher().Hash(runtimeTestPassword)
	if err != nil {
		t.Fatalf("hash runtime password: %v", err)
	}
	stores := &runtimeAuthStores{
		credentials: map[string]authstore.UserCredential{user.Username: {UserID: user.ID, Username: user.Username, PasswordHash: &hash, MustChangePassword: mustChangePassword}},
		sessions:    map[string]authstore.RefreshSession{},
	}
	service, err := newAuthService(config.AuthConfig{SigningKey: "runtime-regression-signing-key", AccessTokenTTL: time.Hour, RefreshTokenTTL: 2 * time.Hour}, stores, stores, runtimeIdentityProvider{users: map[uint64]moduleapi.CurrentUser{user.ID: user}})
	if err != nil {
		t.Fatalf("new auth service: %v", err)
	}
	now := time.Date(2026, time.July, 10, 12, 0, 0, 0, time.UTC)
	service.tokens.now = func() time.Time { return now }
	service.refreshTokens.now = func() time.Time { return now }
	return service, stores, user
}

type runtimeIdentityProvider struct {
	users map[uint64]moduleapi.CurrentUser
}

func (p runtimeIdentityProvider) LookupUserByUsername(_ context.Context, username string) (moduleapi.CurrentUser, error) {
	for _, user := range p.users {
		if user.Username == username {
			return user, nil
		}
	}
	return moduleapi.CurrentUser{}, moduleapi.ErrUserNotFound
}

func (p runtimeIdentityProvider) GetCurrentUserByID(_ context.Context, userID uint64) (moduleapi.CurrentUser, error) {
	user, ok := p.users[userID]
	if !ok {
		return moduleapi.CurrentUser{}, moduleapi.ErrUserNotFound
	}
	return user, nil
}

func (runtimeIdentityProvider) EnsureDefaultAdminProfile(context.Context) (moduleapi.CurrentUser, error) {
	return moduleapi.CurrentUser{}, moduleapi.ErrUserNotFound
}

type runtimeBootstrapProvider struct {
	payload moduleapi.AuthBootstrapPayload
}

func (p runtimeBootstrapProvider) ReadBootstrap(context.Context, *http.Request) (moduleapi.AuthBootstrapPayload, error) {
	return p.payload, nil
}

type runtimeAuthStores struct {
	credentials map[string]authstore.UserCredential
	sessions    map[string]authstore.RefreshSession
}

func (s *runtimeAuthStores) GetUserCredentialByUsername(_ context.Context, username string) (authstore.UserCredential, error) {
	credential, ok := s.credentials[username]
	if !ok {
		return authstore.UserCredential{}, authstore.ErrCredentialNotFound
	}
	return credential, nil
}

func (s *runtimeAuthStores) SetPasswordHash(_ context.Context, input authstore.SetPasswordHashInput) error {
	for username, credential := range s.credentials {
		if credential.UserID == input.UserID {
			credential.PasswordHash = &input.PasswordHash
			credential.MustChangePassword = input.MustChangePassword
			credential.PasswordChangedAt = input.ChangedAt
			s.credentials[username] = credential
			return nil
		}
	}
	return authstore.ErrCredentialNotFound
}

func (s *runtimeAuthStores) EnsureUserCredential(_ context.Context, input authstore.EnsureUserCredentialInput) (authstore.UserCredential, error) {
	if credential, ok := s.credentials[input.Username]; ok {
		return credential, nil
	}
	credential := authstore.UserCredential{Username: input.Username, PasswordHash: &input.PasswordHash, MustChangePassword: input.MustChangePassword}
	s.credentials[input.Username] = credential
	return credential, nil
}

func (s *runtimeAuthStores) ResetPasswordAndRevokeRefreshSessions(ctx context.Context, input authstore.ResetPasswordAndRevokeSessionsInput) error {
	if err := s.SetPasswordHash(ctx, authstore.SetPasswordHashInput{UserID: input.UserID, PasswordHash: input.PasswordHash, MustChangePassword: input.MustChangePassword, ChangedAt: &input.ChangedAt}); err != nil {
		return err
	}
	return s.RevokeRefreshSessionsByUserID(ctx, authstore.RevokeRefreshSessionsByUserIDInput{UserID: input.UserID, RevokedAt: input.ChangedAt})
}

func (s *runtimeAuthStores) ChangePasswordAndRevokeOtherRefreshSessions(ctx context.Context, input authstore.ChangePasswordAndRevokeOtherRefreshSessionsInput) error {
	if err := s.SetPasswordHash(ctx, authstore.SetPasswordHashInput{UserID: input.UserID, PasswordHash: input.PasswordHash, MustChangePassword: input.MustChangePassword, ChangedAt: &input.ChangedAt}); err != nil {
		return err
	}
	return s.RevokeOtherRefreshSessionsByUserID(ctx, authstore.RevokeOtherRefreshSessionsInput{UserID: input.UserID, CurrentTokenID: input.CurrentTokenID, RevokedAt: input.ChangedAt})
}

func (s *runtimeAuthStores) CreateRefreshSession(_ context.Context, input authstore.CreateRefreshSessionInput) (authstore.RefreshSession, error) {
	if _, exists := s.sessions[input.TokenID]; exists {
		return authstore.RefreshSession{}, errors.New("duplicate refresh session")
	}
	session := authstore.RefreshSession{UserID: input.UserID, TokenID: input.TokenID, ExpiresAt: input.ExpiresAt, CreatedAt: input.ExpiresAt.Add(-time.Hour), UpdatedAt: input.ExpiresAt.Add(-time.Hour)}
	s.sessions[input.TokenID] = session
	return session, nil
}

func (s *runtimeAuthStores) GetRefreshSessionByTokenID(_ context.Context, tokenID string) (authstore.RefreshSession, error) {
	session, ok := s.sessions[tokenID]
	if !ok {
		return authstore.RefreshSession{}, authstore.ErrRefreshSessionNotFound
	}
	return session, nil
}

func (s *runtimeAuthStores) RevokeRefreshSession(_ context.Context, input authstore.RevokeRefreshSessionInput) error {
	session, ok := s.sessions[input.TokenID]
	if !ok {
		return authstore.ErrRefreshSessionNotFound
	}
	session.RevokedAt = &input.RevokedAt
	session.ReplacedByTokenID = input.ReplacedByTokenID
	s.sessions[input.TokenID] = session
	return nil
}

func (s *runtimeAuthStores) RevokeRefreshSessionsByUserID(_ context.Context, input authstore.RevokeRefreshSessionsByUserIDInput) error {
	for tokenID, session := range s.sessions {
		if session.UserID == input.UserID && session.RevokedAt == nil {
			session.RevokedAt = &input.RevokedAt
			s.sessions[tokenID] = session
		}
	}
	return nil
}

func (s *runtimeAuthStores) RevokeOtherRefreshSessionsByUserID(_ context.Context, input authstore.RevokeOtherRefreshSessionsInput) error {
	for tokenID, session := range s.sessions {
		if session.UserID == input.UserID && tokenID != input.CurrentTokenID && session.RevokedAt == nil {
			session.RevokedAt = &input.RevokedAt
			s.sessions[tokenID] = session
		}
	}
	return nil
}

func (s *runtimeAuthStores) RevokeRefreshSessionByUserID(_ context.Context, input authstore.RevokeRefreshSessionByUserIDInput) error {
	session, ok := s.sessions[input.TokenID]
	if !ok || session.UserID != input.UserID || session.RevokedAt != nil || !session.ExpiresAt.After(input.RevokedAt) {
		return authstore.ErrRefreshSessionNotFound
	}
	session.RevokedAt = &input.RevokedAt
	s.sessions[input.TokenID] = session
	return nil
}

func (s *runtimeAuthStores) ListActiveRefreshSessionsByUserID(_ context.Context, input authstore.ListActiveRefreshSessionsByUserIDInput) ([]authstore.RefreshSession, error) {
	result := make([]authstore.RefreshSession, 0)
	for _, session := range s.sessions {
		if session.UserID == input.UserID && session.RevokedAt == nil && session.ExpiresAt.After(input.Now) {
			result = append(result, session)
		}
	}
	return result, nil
}

func (s *runtimeAuthStores) RotateRefreshSession(ctx context.Context, input authstore.RotateRefreshSessionInput) (authstore.RefreshSession, error) {
	current, err := s.GetRefreshSessionByTokenID(ctx, input.CurrentTokenID)
	if err != nil || current.RevokedAt != nil || !current.ExpiresAt.After(input.Now) {
		return authstore.RefreshSession{}, authstore.ErrRefreshSessionNotFound
	}
	if err := s.RevokeRefreshSession(ctx, authstore.RevokeRefreshSessionInput{TokenID: input.CurrentTokenID, RevokedAt: input.RevokedAt, ReplacedByTokenID: &input.NewTokenID}); err != nil {
		return authstore.RefreshSession{}, err
	}
	return s.CreateRefreshSession(ctx, authstore.CreateRefreshSessionInput{UserID: current.UserID, TokenID: input.NewTokenID, ExpiresAt: input.NewExpiresAt})
}

var _ authstore.CredentialStore = (*runtimeAuthStores)(nil)
var _ authstore.SessionStore = (*runtimeAuthStores)(nil)
var _ authstore.PasswordChangeRepository = (*runtimeAuthStores)(nil)
