package storeent

import (
	"database/sql"

	sharedentstore "graft/server/internal/store/entstore"
	authent "graft/server/modules/auth/ent"
)

func newCallerTransactionClient(tx *sql.Tx) *authent.Client {
	return authent.NewClient(authent.Driver(sharedentstore.NewCallerTransactionDriver(tx)))
}
