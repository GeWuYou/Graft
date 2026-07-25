package event

import (
	"context"
	"database/sql"
	"errors"
)

var (
	// ErrBackpressure 表示本地有界队列无法立即接收事件。
	ErrBackpressure = errors.New("event dispatcher is at capacity")
	// ErrDispatcherStopped 表示 dispatcher 已停止接受新的发布请求。
	ErrDispatcherStopped = errors.New("event dispatcher is not accepting events")
	// ErrDurableUnavailable 表示当前本地实现不能承诺跨重启可靠投递。
	ErrDurableUnavailable = errors.New("durable event delivery is unavailable")
	// ErrNoHandlers 表示没有注册事件类型的消费者。
	ErrNoHandlers = errors.New("event has no registered handlers")
	// ErrClaimLost 表示 delivery 的租约已被新的 worker 接管，当前执行结果不得写回。
	ErrClaimLost = errors.New("event delivery claim is no longer owned")
)

// PublishOptions 只描述投递机制，不承载领域业务字段。
type PublishOptions struct {
	Delivery DeliveryMode
}

// Receipt 表示平台已接收事件，不表示任何 Handler 已经完成。
type Receipt struct {
	EventID  string
	Delivery DeliveryMode
}

// BatchReceipt 给出批量发布中每个事件的接收结果。
type BatchReceipt struct {
	Accepted []Receipt
	Rejected map[string]error
}

// Publisher 是业务模块依赖的稳定事件发布边界。
type Publisher interface {
	Publish(context.Context, Event, PublishOptions) (Receipt, error)
	PublishAsync(Event, PublishOptions) (Receipt, error)
	PublishBatch(context.Context, []Event, PublishOptions) BatchReceipt
}

// TransactionalPublisher 为需要与业务事实同事务提交的 durable event 提供可选边界。
//
// 它不替代 Publisher；业务模块只有在自己已经持有 SQL transaction 时才应使用此接口，
// Runtime 负责从冻结的 handler 注册表解析 consumer delivery。
type TransactionalPublisher interface {
	PublishTx(context.Context, *sql.Tx, Event, PublishOptions) (Receipt, error)
}
