package storeent

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	entsql "entgo.io/ent/dialect/sql"

	ent "graft/server/modules/user/ent"
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

// RunInCompositeTransaction 为 user/auth 生命周期创建唯一的原始 SQL transaction。两个模块
// 必须从 callback 收到的同一 transaction 构造各自 Ent client，只有本方法可以完成 transaction。
func (r *userRepository) RunInCompositeTransaction(ctx context.Context, callback func(context.Context, userstore.UserRepository, *sql.Tx) error) error {
	if callback == nil {
		return errors.New("user composite transaction callback is required")
	}
	if r.db == nil {
		return errors.New("user composite transaction requires a sql db")
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin user composite transaction: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()
	txClient := ent.NewClient(ent.Driver(entsql.NewDriver("postgres", entsql.Conn{ExecQuerier: tx})))
	profiles := &userRepository{client: txClient}
	if err := callback(ctx, profiles, tx); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit user composite transaction: %w", err)
	}
	committed = true
	return nil
}

var _ userstore.CompositeTransactionRunner = (*userRepository)(nil)
