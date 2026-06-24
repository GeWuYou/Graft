package realtime

import (
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const defaultSubscriberBuffer = 8

type memoryHub struct {
	mu     sync.RWMutex
	topics map[string]map[uint64]*subscriber
	nextID uint64
}

type subscriber struct {
	ch           chan Event
	unsubscribed atomic.Bool
}

// NewHub 创建一个基于内存的实时话题 Hub，并初始化订阅映射。
func NewHub() Hub {
	return &memoryHub{
		topics: make(map[string]map[uint64]*subscriber),
	}
}

func (h *memoryHub) Publish(topic string, payload any) {
	normalized := strings.TrimSpace(topic)
	if h == nil || normalized == "" {
		return
	}

	event := Event{
		Topic:      normalized,
		Data:       payload,
		OccurredAt: time.Now().UTC(),
	}

	h.mu.RLock()
	subscribers := h.topics[normalized]
	targets := make([]*subscriber, 0, len(subscribers))
	for _, current := range subscribers {
		targets = append(targets, current)
	}
	h.mu.RUnlock()

	for _, current := range targets {
		if current == nil || current.unsubscribed.Load() {
			continue
		}
		publishLatestEvent(current.ch, event)
	}
}

// publishLatestEvent 尝试将事件发送到通道，并在通道已满时丢弃一个旧事件后重试一次。
func publishLatestEvent(ch chan Event, event Event) {
	select {
	case ch <- event:
	default:
		drainStaleEvent(ch)
		select {
		case ch <- event:
		default:
		}
	}
}

// drainStaleEvent 从通道中非阻塞地移除一个待处理事件。
func drainStaleEvent(ch chan Event) {
	select {
	case <-ch:
	default:
	}
}

func (h *memoryHub) Subscribe(topic string) (<-chan Event, func()) {
	normalized := strings.TrimSpace(topic)
	ch := make(chan Event, defaultSubscriberBuffer)
	if h == nil || normalized == "" {
		close(ch)
		return ch, func() {}
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	h.nextID++
	id := h.nextID
	if h.topics[normalized] == nil {
		h.topics[normalized] = make(map[uint64]*subscriber)
	}
	sub := &subscriber{ch: ch}
	h.topics[normalized][id] = sub

	return ch, func() {
		h.mu.Lock()
		defer h.mu.Unlock()

		subscribers := h.topics[normalized]
		if subscribers == nil {
			return
		}
		if existing, ok := subscribers[id]; ok {
			delete(subscribers, id)
			existing.unsubscribed.Store(true)
		}
		if len(subscribers) == 0 {
			delete(h.topics, normalized)
		}
	}
}
