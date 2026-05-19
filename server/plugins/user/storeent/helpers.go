package storeent

import (
	"math"

	userstore "graft/server/plugins/user/store"
)

// toEntID converts the stable uint64 identifier used by the user plugin into
// the current Ent int primary key.
func toEntID(id uint64) (int, error) {
	if id == 0 || id > math.MaxInt {
		return 0, userstore.ErrInvalidID
	}

	return int(id), nil
}

func toStoreID(id int) uint64 {
	//nolint:gosec // Ent IDs come from the controlled schema and remain positive.
	return uint64(id)
}
