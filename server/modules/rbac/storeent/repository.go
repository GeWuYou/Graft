package storeent

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	rbacstore "graft/server/modules/rbac/store"
)

type repository struct {
	db *sql.DB
}

type transactionContextKey struct{}

type sqlExecutor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

const permissionSearchFields = 3

// NewRepository 基于共享连接池构建 RBAC 模块的 SQL repository。
// 当 db 为空时返回错误。
func NewRepository(db *sql.DB) (rbacstore.Repository, error) {
	if db == nil {
		return nil, errors.New("rbac repository requires a non-nil sql db")
	}

	return &repository{db: db}, nil
}

// RunInTransaction 让同一 repository 的写入复用一个受控 SQL transaction。
// callback 失败会回滚；只有 callback 与 durable event 均成功时才提交。
func (r *repository) RunInTransaction(ctx context.Context, callback func(context.Context, *sql.Tx) error) error {
	if callback == nil {
		return errors.New("rbac transaction callback is required")
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin rbac transaction: %w", err)
	}
	committed := false
	defer rollbackUncommitted(tx, &committed)

	if err := callback(context.WithValue(ctx, transactionContextKey{}, tx), tx); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit rbac transaction: %w", err)
	}
	committed = true
	return nil
}

func (r *repository) executor(ctx context.Context) sqlExecutor {
	if tx, ok := ctx.Value(transactionContextKey{}).(*sql.Tx); ok && tx != nil {
		return tx
	}
	return r.db
}

func transactionFromContext(ctx context.Context) (*sql.Tx, bool) {
	tx, ok := ctx.Value(transactionContextKey{}).(*sql.Tx)
	return tx, ok && tx != nil
}

func (r *repository) execQuerier(ctx context.Context) execQuerier {
	if tx, ok := transactionFromContext(ctx); ok {
		return execQuerier{tx: tx}
	}
	return execQuerier{db: r.db}
}
