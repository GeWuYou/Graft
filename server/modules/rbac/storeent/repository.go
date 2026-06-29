package storeent

import (
	"database/sql"
	"errors"

	rbacstore "graft/server/modules/rbac/store"
)

type repository struct {
	db *sql.DB
}

const permissionSearchFields = 3

// NewRepository 基于共享连接池构建 RBAC 模块的 SQL repository。
func NewRepository(db *sql.DB) (rbacstore.Repository, error) {
	if db == nil {
		return nil, errors.New("rbac repository requires a non-nil sql db")
	}

	return &repository{db: db}, nil
}
