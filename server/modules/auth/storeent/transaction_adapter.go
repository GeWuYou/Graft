package storeent

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	entsql "entgo.io/ent/dialect/sql"

	"graft/server/internal/moduleapi"
	authent "graft/server/modules/auth/ent"
	authstore "graft/server/modules/auth/store"
)

// PasswordCredentialPreparer 保留 auth 的密码策略、散列和时间语义；持久化 adapter 只接收其
// 产物，从而不会把认证生命周期规则泄漏给 user transaction owner。
type PasswordCredentialPreparer func(context.Context, moduleapi.AuthCredentialProvisionInput) (string, time.Time, error)

// TransactionAdapterFactory 将调用方拥有的 SQL transaction 绑定为 auth 写参与者。
type TransactionAdapterFactory struct {
	prepare PasswordCredentialPreparer
}

func NewTransactionAdapterFactory(prepare PasswordCredentialPreparer) (*TransactionAdapterFactory, error) {
	if prepare == nil {
		return nil, errors.New("auth transaction adapter requires a password credential preparer")
	}
	return &TransactionAdapterFactory{prepare: prepare}, nil
}

func (f *TransactionAdapterFactory) BindAuthTransaction(tx *sql.Tx) (moduleapi.AuthTransactionAdapter, error) {
	if tx == nil {
		return nil, errors.New("auth transaction adapter requires a caller-owned sql transaction")
	}
	client := authent.NewClient(authent.Driver(entsql.NewDriver("postgres", entsql.Conn{ExecQuerier: tx})))
	return &transactionAdapter{
		credentials: &credentialStore{client: client},
		sessions:    &sessionStore{client: client},
		prepare:     f.prepare,
	}, nil
}

type transactionAdapter struct {
	credentials *credentialStore
	sessions    *sessionStore
	prepare     PasswordCredentialPreparer
}

func (a *transactionAdapter) ProvisionPasswordCredential(ctx context.Context, input moduleapi.AuthCredentialProvisionInput) error {
	if input.UserID == 0 {
		return errors.New("auth credential user id is required")
	}
	hash, changedAt, err := a.prepare(ctx, input)
	if err != nil {
		return err
	}
	// user 已在同一个 caller-owned transaction 中创建；独立 identity provider 无法观察未提交资料。
	if err := a.credentials.savePasswordHash(ctx, input.UserID, hash, input.MustChangePassword, &changedAt); err != nil {
		return fmt.Errorf("provision transaction-bound auth credential: %w", err)
	}
	return nil
}

func (a *transactionAdapter) RevokeSessions(ctx context.Context, userID uint64) error {
	if userID == 0 {
		return errors.New("auth session user id is required")
	}
	return a.sessions.RevokeRefreshSessionsByUserID(ctx, authstore.RevokeRefreshSessionsByUserIDInput{
		UserID:    userID,
		RevokedAt: time.Now().UTC(),
	})
}

var _ moduleapi.AuthTransactionAdapterFactory = (*TransactionAdapterFactory)(nil)
var _ moduleapi.AuthTransactionAdapter = (*transactionAdapter)(nil)
