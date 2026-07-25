// Package storeent 提供 auth 所拥有的 Ent 持久化实现。
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

// authRepository 组合凭据和会话存储，为仍依赖聚合仓储契约的调用方提供兼容入口。
type authRepository struct {
	*credentialStore
	*sessionStore
}

// NewAuthRepository 根据指定 Ent 客户端和用户身份提供方创建 auth 聚合仓储；凭据或会话存储初始化失败时返回错误。
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

// NewCredentialStore 创建 auth 所拥有的凭据存储；用户身份仍由 user 模块提供，客户端或身份提供方为空时返回错误。
func NewCredentialStore(client *authent.Client, identity moduleapi.UserIdentityProvider) (store.CredentialStore, error) {
	return newCredentialStore(client, identity)
}

type credentialStore struct {
	client   *authent.Client
	identity moduleapi.UserIdentityProvider
}

// newCredentialStore 创建凭据存储，并验证其所需的 Ent 客户端和用户身份提供者。
// 返回初始化的凭据存储，或在依赖无效时返回错误。
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

// EnsureUserCredential 只为已经创建的用户资料写入凭据；资料创建由 UserIdentityProvider 负责并应先于该存储调用完成。
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
	// 凭据写入集中在 helper，事务边界由 auth service 注入的 TransactionRunner 统一拥有。
	return r.savePasswordHash(ctx, userID, hash, mustChange, changedAt)
}

// toStoreUserCredential 将 Ent 凭据记录转换为存储层用户凭据。
func toStoreUserCredential(record *authent.AuthCredential) store.UserCredential {
	return store.UserCredential{
		UserID:             record.UserID,
		PasswordHash:       record.PasswordHash,
		MustChangePassword: record.MustChangePassword,
		PasswordChangedAt:  record.PasswordChangedAt,
	}
}

var _ store.AuthRepository = (*authRepository)(nil)
var _ store.CredentialStore = (*credentialStore)(nil)
var _ store.TransactionRunner = (*credentialStore)(nil)
