package storeent

import (
	"database/sql"

	sharedentstore "graft/server/internal/store/entstore"
	ent "graft/server/modules/user/ent"
)

func newCallerTransactionClient(tx *sql.Tx) *ent.Client {
	return ent.NewClient(ent.Driver(sharedentstore.NewCallerTransactionDriver(tx)))
}
