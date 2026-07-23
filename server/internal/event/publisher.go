package event

import (
	"context"
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
