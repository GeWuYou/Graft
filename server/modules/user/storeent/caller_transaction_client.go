package storeent

import (
	"context"
	"database/sql"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"

	ent "graft/server/modules/user/ent"
)

// callerTransactionDriver 让 Ent 在调用方拥有的原始 SQL transaction 上执行语句。
// Ent 的单节点写入会申请并完成内部 transaction；这里将其变为 no-op，避免提前结束外层生命周期。
type callerTransactionDriver struct {
	*entsql.Driver
}

func (d *callerTransactionDriver) Tx(context.Context) (dialect.Tx, error) {
	return dialect.NopTx(d), nil
}

func newCallerTransactionClient(tx *sql.Tx) *ent.Client {
	driver := entsql.NewDriver("postgres", entsql.Conn{ExecQuerier: tx})
	return ent.NewClient(ent.Driver(&callerTransactionDriver{Driver: driver}))
}
