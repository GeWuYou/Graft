package realtime

import (
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const defaultSubscriberBuffer = 8

var errTopicObserverRequired = errors.New("realtime topic observer requires at least one callback")

type memoryHub struct {
	mu        sync.RWMutex
	topics    map[string]map[uint64]*subscriber
	states    map[string]topicLifecycleState
	observers map[string]map[uint64]topicObserver
	nextID    uint64
}

type subscriber struct {
	ch           chan Event
	unsubscribed atomic.Bool
}

type topicLifecycleState struct {
	generation uint64
	active     bool
}

type topicObserver struct {
	onActive   func(topic string)
	onInactive func(topic string)
}

// NewHub 创建一个基于内存的实时话题 Hub，并初始化订阅与观察者映射。
func NewHub() Hub {
	return &memoryHub{
		topics:    make(map[string]map[uint64]*subscriber),
		states:    make(map[string]topicLifecycleState),
		observers: make(map[string]map[uint64]topicObserver),
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
	h.nextID++
	id := h.nextID
	wasInactive := len(h.topics[normalized]) == 0
	state := h.states[normalized]
	if h.topics[normalized] == nil {
		h.topics[normalized] = make(map[uint64]*subscriber)
	}
	sub := &subscriber{ch: ch}
	h.topics[normalized][id] = sub
	if wasInactive {
		state.generation++
		state.active = true
		h.states[normalized] = state
	}
	observers := copyTopicObservers(h.observers[normalized])
	h.mu.Unlock()

	if wasInactive {
		notifyTopicObservers(normalized, observers, true)
	}

	return ch, func() {
		notifyInactive := false
		notifyGeneration := uint64(0)
		observers := []topicObserver(nil)

		h.mu.Lock()
		subscribers := h.topics[normalized]
		if subscribers == nil {
			h.mu.Unlock()
			return
		}
		if existing, ok := subscribers[id]; ok {
			delete(subscribers, id)
			existing.unsubscribed.Store(true)
		}
		if len(subscribers) == 0 {
			delete(h.topics, normalized)
			state := h.states[normalized]
			state.active = false
			h.states[normalized] = state
			notifyInactive = true
			notifyGeneration = state.generation
			observers = copyTopicObservers(h.observers[normalized])
		}
		h.mu.Unlock()

		if notifyInactive {
			h.notifyInactiveIfCurrent(normalized, notifyGeneration, observers)
		}
	}
}

func (h *memoryHub) RegisterTopicObserver(
	topic string,
	onActive func(topic string),
	onInactive func(topic string),
) (func(), error) {
	normalized := strings.TrimSpace(topic)
	if h == nil || normalized == "" {
		return func() {}, nil
	}
	if onActive == nil && onInactive == nil {
		return nil, errTopicObserverRequired
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	h.nextID++
	id := h.nextID
	if h.observers[normalized] == nil {
		h.observers[normalized] = make(map[uint64]topicObserver)
	}
	h.observers[normalized][id] = topicObserver{
		onActive:   onActive,
		onInactive: onInactive,
	}

	return func() {
		h.mu.Lock()
		defer h.mu.Unlock()

		observers := h.observers[normalized]
		if observers == nil {
			return
		}
		delete(observers, id)
		if len(observers) == 0 {
			delete(h.observers, normalized)
		}
	}, nil
}

// copyTopicObservers 将主题观察者映射复制为切片。
//
// 返回包含所有观察者值的切片；当输入为空时返回 nil。
func copyTopicObservers(observers map[uint64]topicObserver) []topicObserver {
	if len(observers) == 0 {
		return nil
	}
	copied := make([]topicObserver, 0, len(observers))
	for _, observer := range observers {
		copied = append(copied, observer)
	}
	return copied
}

// notifyTopicObservers 按主题状态变更调用观察者回调。
// active 为 true 时调用各观察者的 onActive；否则调用 onInactive。
func notifyTopicObservers(topic string, observers []topicObserver, active bool) {
	for _, observer := range observers {
		if active {
			if observer.onActive != nil {
				observer.onActive(topic)
			}
			continue
		}
		if observer.onInactive != nil {
			observer.onInactive(topic)
		}
	}
}

func (h *memoryHub) notifyInactiveIfCurrent(topic string, generation uint64, observers []topicObserver) {
	h.mu.RLock()
	state := h.states[topic]
	h.mu.RUnlock()

	if state.generation != generation || state.active {
		return
	}

	notifyTopicObservers(topic, observers, false)
}
