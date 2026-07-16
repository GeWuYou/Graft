package realtime

import "time"

// Event 是发送给主题订阅者的稳定实时事件信封。
type Event struct {
	Topic      string      `json:"topic"`
	Data       any         `json:"data"`
	OccurredAt time.Time   `json:"occurred_at"`
}

// Publisher 向一个 canonical topic 发布一个 payload。
type Publisher interface {
	Publish(topic string, payload any)
}

// Subscriber 订阅一个 canonical topic，并返回事件流及取消订阅函数；调用方负责执行取消函数。
type Subscriber interface {
	Subscribe(topic string) (<-chan Event, func())
}

// Hub 合并实时事件的发布与订阅边界。
type Hub interface {
	Publisher
	Subscriber
}

// TopicSubscriptionMonitor 暴露精确主题的订阅者生命周期观测，供仅应在存在 WebSocket 订阅者时运行的模块后台生产者使用。
type TopicSubscriptionMonitor interface {
	RegisterTopicObserver(
		topic string,
		onActive func(topic string),
		onInactive func(topic string),
	) (unsubscribe func(), err error)
}
