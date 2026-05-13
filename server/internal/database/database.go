package database

import (
	"database/sql"
	"fmt"

	entsql "entgo.io/ent/dialect/sql"
	_ "github.com/jackc/pgx/v5/stdlib"

	"graft/server/internal/config"
	"graft/server/internal/ent"
)

// Resources 持有服务端运行时所需的 SQL 连接池与 Ent 客户端。
type Resources struct {
	SQL    *sql.DB
	Client *ent.Client
}

// Open 创建服务端运行时所需的 PostgreSQL 相关资源。
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
