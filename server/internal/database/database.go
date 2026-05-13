package database

import (
	"database/sql"
	"fmt"

	entsql "entgo.io/ent/dialect/sql"
	_ "github.com/jackc/pgx/v5/stdlib"

	"graft/server/internal/config"
	"graft/server/internal/ent"
)

// Resources 持有数据库层对外暴露的运行时资源句柄。
//
// SQL 连接池与 Ent 客户端共享同一底层数据库连接来源，因此必须按约定通过 Close
// 统一释放，而不是交由各个调用方分别关闭。
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

	driver := entsql.OpenDB("postgres", sqlDB)

	return &Resources{
		SQL:    sqlDB,
		Client: ent.NewClient(ent.Driver(driver)),
	}, nil
}

// Close 按资源归属顺序释放 Ent 客户端及其底层 SQL 连接池。
//
// 如果上层在失败路径中仅拿到部分资源，Close 仍允许按 nil 安全调用；一旦某一步关闭失败，
// 会立即返回该错误，调用方应把它并入整体清理错误中。
func Close(resources *Resources) error {
	if resources == nil {
		return nil
	}

	if resources.Client != nil {
		if err := resources.Client.Close(); err != nil {
			return fmt.Errorf("close ent client: %w", err)
		}
	}

	if resources.SQL != nil {
		if err := resources.SQL.Close(); err != nil {
			return fmt.Errorf("close sql pool: %w", err)
		}
	}

	return nil
}
