package storeent

import (
	"database/sql"
	"fmt"

	entsql "entgo.io/ent/dialect/sql"
	authent "graft/server/modules/auth/ent"
)

// It returns an error if sqlDB is nil.
func NewClient(sqlDB *sql.DB) (*authent.Client, error) {
	if sqlDB == nil {
		return nil, fmt.Errorf("auth storeent requires a non-nil sql db")
	}
	return authent.NewClient(authent.Driver(entsql.OpenDB("postgres", sqlDB))), nil
}
