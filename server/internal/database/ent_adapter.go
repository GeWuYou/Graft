package database

import (
	"database/sql"
	"fmt"
	"unsafe"

	entsql "entgo.io/ent/dialect/sql"

	"graft/server/internal/ent"
)

// SQLDBFromEntClient 在运行时仍同时持有 Ent client 与共享 SQL 连接池期间，
// 从仓库生成的 Ent client 中提取底层共享连接池。
//
//nolint:gosec // 过渡期需要访问生成 client 的私有 driver，才能把 reset helper 保持在当前 owned scope 内。
func SQLDBFromEntClient(client *ent.Client) (*sql.DB, error) {
	if client == nil {
		return nil, fmt.Errorf("ent client is nil")
	}

	driver := entClientDriver(client)
	if driver == nil {
		return nil, fmt.Errorf("ent client driver is unavailable")
	}

	sqlDriver, ok := driver.(*entsql.Driver)
	if !ok {
		return nil, fmt.Errorf("ent client requires SQL driver, got %T", driver)
	}

	sqlDB := sqlDriver.DB()
	if sqlDB == nil {
		return nil, fmt.Errorf("ent client sql db is unavailable")
	}

	return sqlDB, nil
}

func entClientDriver(client *ent.Client) any {
	type clientConfig struct {
		driver any
	}
	type clientView struct {
		Config clientConfig
	}

	view := (*clientView)(unsafe.Pointer(client)) //nolint:gosec // 见 SQLDBFromEntClient 的过渡期说明。
	return view.Config.driver
}
