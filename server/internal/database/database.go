// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package database

import (
	"database/sql"
	"fmt"
	// 注册 pgx 的 database/sql 驱动，供 sql.Open("pgx", ...) 显式使用。
	_ "github.com/jackc/pgx/v5/stdlib"

	"graft/server/internal/config"
)

// Resources 持有数据库层对外暴露的运行时资源句柄。
//
// 当前 core runtime 只在这里持有共享的 SQL 连接池；更高层或模块私有 ORM
// 句柄应由各自边界显式构造，而不是重新注册回 core。
type Resources struct {
	SQL *sql.DB
}

// Open 创建服务端运行时需要的 PostgreSQL 资源集合。
//
// 返回值中的资源所有权转移给调用方；调用方在完成启动或失败回滚时都必须调用 Close。
// 该函数只负责构造共享连接池，真实连通性由后续首次使用或上层探活确认。
func Open(cfg config.DatabaseConfig) (*Resources, error) {
	if cfg.Driver != "postgres" {
		return nil, fmt.Errorf("unsupported database driver %q: only postgres is supported", cfg.Driver)
	}

	sqlDB, err := sql.Open("pgx", cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("open postgres database pool: %w", err)
	}
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	sqlDB.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)

	return &Resources{SQL: sqlDB}, nil
}

// Close 按 Resources 的统一所有权释放数据库资源。
//
// 如果上层在失败路径中仅拿到部分资源，Close 仍允许按 nil 安全调用。
func Close(resources *Resources) error {
	if resources == nil {
		return nil
	}

	if resources.SQL != nil {
		if closeErr := resources.SQL.Close(); closeErr != nil {
			return fmt.Errorf("close database resources: close sql pool: %w", closeErr)
		}
	}

	return nil
}
