package storeent

import (
	"context"
	"errors"
	"fmt"

	userstore "graft/server/modules/user/store"
)

// RunInTransaction 为 user service 的 profile 写入创建唯一的 Ent transaction。
// callback 收到的 repository 共享该 transaction；只有 callback 成功后才提交。
func (r *userRepository) RunInTransaction(ctx context.Context, callback func(context.Context, userstore.UserRepository) error) error {
	if callback == nil {
		return errors.New("user transaction callback is required")
	}
	tx, err := r.client.Tx(ctx)
	if err != nil {
		return fmt.Errorf("begin user transaction: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	profiles := &userRepository{client: tx.Client()}
	if err := callback(ctx, profiles); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit user transaction: %w", err)
	}
	committed = true
	return nil
}

var _ userstore.TransactionRunner = (*userRepository)(nil)
