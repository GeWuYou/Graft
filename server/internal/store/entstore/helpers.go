package entstore

import (
	"math"

	"graft/server/internal/store"
)

// toEntID 把上层稳定的 uint64 标识转换为 Ent 当前使用的 int 主键。
//
// 非法或超范围的标识返回 store.ErrInvalidID，由具体仓储方法决定是否需要转换为领域错误。
func toEntID(id uint64) (int, error) {
	if id == 0 || id > math.MaxInt {
		return 0, store.ErrInvalidID
	}

	return int(id), nil
}

func toStoreID(id int) uint64 {
	//nolint:gosec // Ent 主键来自当前受控 schema，经过仓储边界约束后只会写入正整数标识。
	return uint64(id)
}
