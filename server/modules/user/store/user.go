package store

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

var (
	// ErrUserNotFound 表示目标用户不存在或已不再满足仓储的可见条件。
	ErrUserNotFound = errors.New("user not found")

	// ErrInvalidID 表示调用方提供了无效的稳定用户标识。
	ErrInvalidID = errors.New("invalid id")

	// ErrUsernameConflict 表示用户名已被其他用户占用，调用方不得将其当作更新成功处理。
	ErrUsernameConflict = errors.New("username already exists")
)

const protectedDefaultAdminUsername = "graft"

// User 是 user 模块内部稳定传递的用户摘要；ProtectedDefaultAdmin 标记不可被普通管理操作破坏的默认管理员。
type User struct {
	ID                    uint64
	Username              string
	Display               string
	Status                string
	ProtectedDefaultAdmin bool
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

// CreateUserInput 描述创建用户所需的最小输入，并保留执行操作的主体用于审计归属。
type CreateUserInput struct {
	Username string
	Display  string
	Status   string
	ActorID  uint64
}

// UpdateUserInput 描述用户资料更新输入；ActorID 标识执行更新的主体而非被修改的目标用户。
type UpdateUserInput struct {
	ID       uint64
	Username string
	Display  string
	ActorID  uint64
}

// SetUserStatusInput 描述用户状态变更输入；状态变更的审计主体由 ActorID 指定。
type SetUserStatusInput struct {
	ID      uint64
	Status  string
	ActorID uint64
}

// DeleteUserInput 描述软删除输入；删除时间和 ActorID 共同保留删除事实及其责任主体。
type DeleteUserInput struct {
	ID        uint64
	DeletedAt time.Time
	ActorID   uint64
}

// UserRepository 暴露 user 模块私有的用户持久化契约；调用方应通过模块服务执行授权与业务规则。
type UserRepository interface {
	GetByID(ctx context.Context, id uint64) (User, error)
	GetByUsername(ctx context.Context, username string) (User, error)
	List(ctx context.Context) ([]User, error)
	ListSecuritySummaries(ctx context.Context, afterID uint64, limit int) ([]User, error)
	Count(ctx context.Context) (int, error)
	Create(ctx context.Context, input CreateUserInput) (User, error)
	Update(ctx context.Context, input UpdateUserInput) (User, error)
	SetStatus(ctx context.Context, input SetUserStatusInput) (User, error)
	Delete(ctx context.Context, input DeleteUserInput) error
}

// TransactionRunner 是 user 模块资料生命周期的本地事务边界。服务定义 callback 内的 profile 写入范围；
// 实现负责一次 Begin、失败回滚和成功提交，并只向 callback 提供绑定同一事务的仓储。
// callback 不得自行提交、回滚或在返回后继续使用该仓储。
type TransactionRunner interface {
	RunInTransaction(ctx context.Context, callback func(context.Context, UserRepository) error) error
}

// CompositeTransactionRunner 是 user/auth 复合生命周期的唯一事务边界。实现创建原始 SQL
// transaction 并把同一 transaction 绑定到 profile repository；callback 的 auth 参与者由 user
// service 另外绑定，但不得提交或回滚该 transaction。
type CompositeTransactionRunner interface {
	RunInCompositeTransaction(context.Context, func(context.Context, UserRepository, *sql.Tx) error) error
}

// IsProtectedDefaultAdminUsername 判断用户名是否属于内置的受保护默认管理员账号。
// 如果用户名等于内置默认管理员用户名，则返回 `true`，否则返回 `false`。
func IsProtectedDefaultAdminUsername(username string) bool {
	return username == protectedDefaultAdminUsername
}
