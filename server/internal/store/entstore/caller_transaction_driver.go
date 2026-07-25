package entstore

import (
	"context"
	"database/sql"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
)

// NewCallerTransactionDriver 将调用方拥有的 SQL transaction 适配为 Ent driver。
// 该 driver 将 Ent 单节点写入的内部 transaction 变为 no-op，避免提前结束外层 transaction。
func NewCallerTransactionDriver(tx *sql.Tx) dialect.Driver {
	return &callerTransactionDriver{Driver: entsql.NewDriver("postgres", entsql.Conn{ExecQuerier: tx})}
}

type callerTransactionDriver struct {
	*entsql.Driver
}

func (d *callerTransactionDriver) Tx(context.Context) (dialect.Tx, error) {
	return dialect.NopTx(d), nil
}
