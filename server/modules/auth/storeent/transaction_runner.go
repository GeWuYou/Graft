package storeent

import (
	"context"
	"errors"
	"fmt"

	"graft/server/modules/auth/store"
)

// RunInTransaction 为 auth service 的多写 use case 创建唯一的 Ent transaction。
// callback 收到的 credential 与 session store 共享该 transaction；只有 callback 成功后才提交。
func (r *credentialStore) RunInTransaction(ctx context.Context, callback func(context.Context, store.CredentialStore, store.SessionStore) error) error {
	if callback == nil {
		return errors.New("auth transaction callback is required")
	}
	tx, err := r.client.Tx(ctx)
	if err != nil {
		return fmt.Errorf("begin auth transaction: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	txClient := tx.Client()
	credentials := &credentialStore{client: txClient, identity: r.identity}
	sessions := &sessionStore{client: txClient}
	if err := callback(ctx, credentials, sessions); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit auth transaction: %w", err)
	}
	committed = true
	return nil
}
