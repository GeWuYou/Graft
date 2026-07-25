package moduleapi

import (
	"context"
	"database/sql"
)

// AuthCredentialProvisionInput 描述复合 user/auth 写入中由 auth 拥有的凭据创建语义。
// Password 始终以明文传给 auth；哈希策略、凭据存储细节和密码生命周期规则不能泄漏给事务 owner。
type AuthCredentialProvisionInput struct {
	UserID             uint64
	Password           string
	MustChangePassword bool
}

// AuthTransactionAdapter 是绑定到调用方拥有的 SQL transaction 的 auth 写参与者。
// 它只执行 auth 所拥有的凭据和会话写入，绝不开始、提交或回滚 transaction，且不得在 owner
// 的 callback 返回后继续使用。调用方只能在同一数据库连接池创建 transaction 后调用该接口。
type AuthTransactionAdapter interface {
	ProvisionPasswordCredential(context.Context, AuthCredentialProvisionInput) error
	RevokeSessions(context.Context, uint64) error
}

// AuthTransactionAdapterFactory 把调用方已经创建的原始 SQL transaction 绑定为 auth 写参与者。
// user 生命周期 workflow 是当前唯一 consumer 和 transaction owner；该 factory 不构成全局
// UnitOfWork，也不通过 context 传播 transaction。nil 或已完成的 transaction 必须返回错误。
type AuthTransactionAdapterFactory interface {
	BindAuthTransaction(*sql.Tx) (AuthTransactionAdapter, error)
}
