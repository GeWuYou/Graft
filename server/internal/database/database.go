package database

import (
	"database/sql"
	"errors"
	"fmt"

	entsql "entgo.io/ent/dialect/sql"
	// 注册 pgx 的 database/sql 驱动，供 sql.Open("pgx", ...) 显式使用。
	_ "github.com/jackc/pgx/v5/stdlib"

	"graft/server/internal/config"
	"graft/server/internal/ent"
)

// Resources 持有数据库层对外暴露的运行时资源句柄。
//
// SQL 连接池与 Ent 客户端共享同一底层数据库连接来源，因此资源关闭必须统一
// 通过 Close 执行，而不是交由各个调用方分别关闭。
type Resources struct {
	SQL    *sql.DB
	Client *ent.Client
}

// Open 创建服务端运行时需要的 PostgreSQL 资源集合。
//
// 返回值中的资源所有权转移给调用方；调用方在完成启动或失败回滚时都必须调用 Close。
// 该函数只负责构造连接池与 Ent 客户端，真实连通性由后续首次使用或上层探活确认。
func Open(cfg config.DatabaseConfig) (*Resources, error) {
	if cfg.Driver != "postgres" {
		return nil, fmt.Errorf("unsupported database driver %q: only postgres is supported", cfg.Driver)
	}

	sqlDB, err := sql.Open("pgx", cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("open postgres database pool: %w", err)
	}

	resources := &Resources{
		SQL: sqlDB,
	}
	success := false
	defer func() {
		// 这里为后续可能新增的装配步骤保留统一回滚，避免在返回前的失败路径泄漏底层连接池。
		if !success {
			_ = Close(resources)
		}
	}()

	driver := entsql.OpenDB("postgres", sqlDB)
	resources.Client = ent.NewClient(ent.Driver(driver))

	success = true
	return resources, nil
}

// Close 按 Resources 的统一所有权释放数据库资源。
//
// 如果上层在失败路径中仅拿到部分资源，Close 仍允许按 nil 安全调用；关闭过程中会继续尝试
// 释放剩余资源，并把所有失败聚合后返回，避免前一个错误掩盖后续清理结果。
func Close(resources *Resources) error {
	if resources == nil {
		return nil
	}

	var err error

	if resources.Client != nil {
		if closeErr := resources.Client.Close(); closeErr != nil {
			err = errors.Join(err, fmt.Errorf("close ent client: %w", closeErr))
		}
	}

	if resources.SQL != nil {
		if closeErr := resources.SQL.Close(); closeErr != nil {
			err = errors.Join(err, fmt.Errorf("close sql pool: %w", closeErr))
		}
	}

	if err != nil {
		return fmt.Errorf("close database resources: %w", err)
	}

	return nil
}
