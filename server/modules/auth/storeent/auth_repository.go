// Package storeent provides auth-owned Ent persistence implementations.
package storeent

import (
	"context"
	"errors"
	"fmt"
	"time"

	"graft/server/internal/moduleapi"
	authent "graft/server/modules/auth/ent"
	authcredentialent "graft/server/modules/auth/ent/authcredential"
	"graft/server/modules/auth/store"
)

// authRepository composes the independently owned credential and session stores
// for callers that still use the historical aggregate repository contract.
type authRepository struct {
	*credentialStore
	*sessionStore
}

// NewAuthRepository builds the aggregate compatibility repository from auth-owned storage.
// New runtime wiring should resolve CredentialStore and SessionStore separately.
func NewAuthRepository(client *authent.Client, identity moduleapi.UserIdentityProvider) (store.AuthRepository, error) {
	credentials, err := newCredentialStore(client, identity)
	if err != nil {
		return nil, err
	}
	sessions, err := newSessionStore(client)
	if err != nil {
		return nil, err
	}
	return &authRepository{credentialStore: credentials, sessionStore: sessions}, nil
}

// NewCredentialStore builds the auth-owned credential store. Identity remains
// a stable capability supplied by user rather than an Ent dependency.
func NewCredentialStore(client *authent.Client, identity moduleapi.UserIdentityProvider) (store.CredentialStore, error) {
	return newCredentialStore(client, identity)
}

type credentialStore struct {
	client   *authent.Client
	identity moduleapi.UserIdentityProvider
}

func newCredentialStore(client *authent.Client, identity moduleapi.UserIdentityProvider) (*credentialStore, error) {
	if client == nil {
		return nil, errors.New("auth credential store requires a non-nil ent client")
	}
	if identity == nil {
		return nil, errors.New("auth credential store requires a user identity provider")
	}
	return &credentialStore{client: client, identity: identity}, nil
}

func (r *credentialStore) GetUserCredentialByUsername(ctx context.Context, username string) (store.UserCredential, error) {
	user, err := r.identity.LookupUserByUsername(ctx, username)
	if err != nil {
		return store.UserCredential{}, err
	}
	credential, err := r.queryByUserID(ctx, user.ID)
	if err != nil {
		return store.UserCredential{}, err
	}
	credential.Username = user.Username
	return credential, nil
}

func (r *credentialStore) SetPasswordHash(ctx context.Context, input store.SetPasswordHashInput) error {
	if _, err := r.identity.GetCurrentUserByID(ctx, input.UserID); err != nil {
		return err
	}
	return r.upsertPasswordHash(ctx, input.UserID, input.PasswordHash, input.MustChangePassword, input.ChangedAt)
}

// EnsureUserCredential only writes credentials for an already-provisioned profile.
// Profile provisioning stays with UserIdentityProvider and is completed before this store is used.
func (r *credentialStore) EnsureUserCredential(ctx context.Context, input store.EnsureUserCredentialInput) (store.UserCredential, error) {
	user, err := r.identity.LookupUserByUsername(ctx, input.Username)
	if err != nil {
		return store.UserCredential{}, err
	}
	if err := r.upsertPasswordHash(ctx, user.ID, input.PasswordHash, input.MustChangePassword, nil); err != nil {
		return store.UserCredential{}, err
	}
	credential, err := r.queryByUserID(ctx, user.ID)
	if err != nil {
		return store.UserCredential{}, err
	}
	credential.Username = user.Username
	return credential, nil
}

func (r *credentialStore) ResetPasswordAndRevokeRefreshSessions(ctx context.Context, input store.ResetPasswordAndRevokeSessionsInput) error {
	if _, err := r.identity.GetCurrentUserByID(ctx, input.UserID); err != nil {
		return err
	}
	return r.updatePasswordAndRevokeSessions(ctx, input.UserID, input.PasswordHash, input.MustChangePassword, input.ChangedAt, "")
}

func (r *credentialStore) ChangePasswordAndRevokeOtherRefreshSessions(ctx context.Context, input store.ChangePasswordAndRevokeOtherRefreshSessionsInput) error {
	if _, err := r.identity.GetCurrentUserByID(ctx, input.UserID); err != nil {
		return err
	}
	return r.updatePasswordAndRevokeSessions(ctx, input.UserID, input.PasswordHash, input.MustChangePassword, input.ChangedAt, input.CurrentTokenID)
}

func (r *credentialStore) queryByUserID(ctx context.Context, userID uint64) (store.UserCredential, error) {
	record, err := r.client.AuthCredential.Query().Where(authcredentialent.UserIDEQ(userID)).Only(ctx)
	if err != nil {
		if authent.IsNotFound(err) {
			return store.UserCredential{}, store.ErrCredentialNotFound
		}
		return store.UserCredential{}, fmt.Errorf("query auth credential by user id: %w", err)
	}
	return toStoreUserCredential(record), nil
}

func (r *credentialStore) upsertPasswordHash(ctx context.Context, userID uint64, hash string, mustChange bool, changedAt *time.Time) error {
	// Implemented in auth_repository_helpers.go to keep transaction operations together.
	return r.savePasswordHash(ctx, userID, hash, mustChange, changedAt)
}

func toStoreUserCredential(record *authent.AuthCredential) store.UserCredential {
	return store.UserCredential{
		UserID:             record.UserID,
		PasswordHash:       record.PasswordHash,
		MustChangePassword: record.MustChangePassword,
		PasswordChangedAt:  record.PasswordChangedAt,
	}
}

var _ store.AuthRepository = (*authRepository)(nil)
var _ store.PasswordChangeRepository = (*credentialStore)(nil)
var _ store.CredentialStore = (*credentialStore)(nil)
