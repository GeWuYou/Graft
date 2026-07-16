package storeent

import (
	"math"

	userstore "graft/server/modules/user/store"
)

// toEntID 将 user 模块使用的稳定 uint64 标识转换为当前 Ent 的 int 主键。
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
