package storeent

import (
	"database/sql"
	"fmt"

	entsql "entgo.io/ent/dialect/sql"
	authent "graft/server/modules/auth/ent"
)

// NewClient 根据 sqlDB 创建 auth Ent 客户端；sqlDB 为空时返回错误。
func NewClient(sqlDB *sql.DB) (*authent.Client, error) {
	if sqlDB == nil {
		return nil, fmt.Errorf("auth storeent requires a non-nil sql db")
	}
	return authent.NewClient(authent.Driver(entsql.OpenDB("postgres", sqlDB))), nil
}
