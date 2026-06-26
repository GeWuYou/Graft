package realtime

import "time"

// Event is the stable realtime payload envelope sent to topic subscribers.
type Event struct {
	Topic      string      `json:"topic"`
	Data       any         `json:"data"`
	OccurredAt time.Time   `json:"occurred_at"`
}

// Publisher publishes one payload to one canonical topic.
type Publisher interface {
	Publish(topic string, payload any)
}

// Subscriber subscribes to one canonical topic and returns the event stream plus an unsubscribe function.
type Subscriber interface {
	Subscribe(topic string) (<-chan Event, func())
}

// Hub combines publish and subscribe boundaries.
type Hub interface {
	Publisher
	Subscriber
}

// TopicSubscriptionMonitor exposes exact-topic subscriber lifecycle observation
// for module-owned background producers that should only run while a topic has
// at least one active websocket subscriber.
type TopicSubscriptionMonitor interface {
	RegisterTopicObserver(
		topic string,
		onActive func(topic string),
		onInactive func(topic string),
	) (unsubscribe func(), err error)
}
