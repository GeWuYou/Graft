package event

import "context"

// Handler 消费一个或多个稳定事件类型。
//
// ID 必须在运行时内唯一，并会在 Outbox delivery 表中作为 consumer identity。
// 处理器必须以 Event.ID 或业务幂等键实现幂等：best-effort 重试会再次调用同一事件的全部 handler，
// durable Outbox 则会在当前 consumer 尚未确认完成时至少一次重投。
type Handler interface {
	ID() string
	Types() []Type
	Handle(context.Context, Event) error
}

// Registry 仅向模块开放处理器声明能力；worker 生命周期仍由 Runtime 持有。
type Registry interface {
	Register(Handler) error
}
