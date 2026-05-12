package database

import (
	"database/sql"
	"fmt"

	entsql "entgo.io/ent/dialect/sql"
	_ "github.com/jackc/pgx/v5/stdlib"

	"graft/server/internal/config"
	"graft/server/internal/ent"
)

// Resources owns the SQL pool and Ent client required by the server runtime.
type Resources struct {
	SQL    *sql.DB
	Client *ent.Client
}

// Open creates the PostgreSQL resources required by the server runtime.
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

// Close releases the Ent client and underlying SQL pool.
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
