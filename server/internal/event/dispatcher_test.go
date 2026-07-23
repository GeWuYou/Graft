package event

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"go.uber.org/zap"
)

const testEventType Type = "test.event"

type testHandler struct {
	id     string
	types  []Type
	handle func(context.Context, Event) error
}

type memoryOutboxStore struct {
	mu        sync.Mutex
	pending   []ClaimedDelivery
	completed chan ClaimedDelivery
}

func (s *memoryOutboxStore) Append(_ context.Context, event Event, consumerIDs []string) (Receipt, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, consumerID := range consumerIDs {
		s.pending = append(s.pending, ClaimedDelivery{Event: event, ConsumerID: consumerID})
	}
	return Receipt{EventID: event.ID, Delivery: DeliveryDurable}, nil
}

func (s *memoryOutboxStore) Claim(_ context.Context, _ string, _ time.Time, _ time.Duration, limit int) ([]ClaimedDelivery, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if limit > len(s.pending) {
		limit = len(s.pending)
	}
	claimed := append([]ClaimedDelivery(nil), s.pending[:limit]...)
	s.pending = s.pending[limit:]
	for index := range claimed {
		claimed[index].Attempt++
	}
	return claimed, nil
}

func (s *memoryOutboxStore) Complete(_ context.Context, delivery ClaimedDelivery) error {
	s.completed <- delivery
	return nil
}

func (*memoryOutboxStore) Retry(context.Context, ClaimedDelivery, time.Time, error) error { return nil }

func (h testHandler) ID() string { return h.id }

func (h testHandler) Types() []Type { return h.types }

func (h testHandler) Handle(ctx context.Context, event Event) error {
	return h.handle(ctx, event)
}

func newTestEvent(t *testing.T, id string) Event {
	t.Helper()
	payload, err := json.Marshal(map[string]string{"value": "test"})
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	return Event{ID: id, Type: testEventType, Version: 1, Source: "test", Payload: payload}
}

func TestDispatcherDeliversFrozenEvent(t *testing.T) {
	dispatcher := NewDispatcher(zap.NewNop(), Options{WorkerCount: 1})
	received := make(chan Event, 1)
	if err := dispatcher.Register(testHandler{id: "receiver", types: []Type{testEventType}, handle: func(_ context.Context, event Event) error {
		received <- event
		return nil
	}}); err != nil {
		t.Fatalf("register handler: %v", err)
	}
	if err := dispatcher.Start(context.Background()); err != nil {
		t.Fatalf("start dispatcher: %v", err)
	}
	event := newTestEvent(t, "event-1")
	if _, err := dispatcher.Publish(context.Background(), event, PublishOptions{}); err != nil {
		t.Fatalf("publish event: %v", err)
	}
	event.Payload[0] = 'x'
	select {
	case delivered := <-received:
		if string(delivered.Payload) != `{"value":"test"}` {
			t.Fatalf("event payload was not frozen: %s", delivered.Payload)
		}
		if delivered.OccurredAt.IsZero() || delivered.CreatedAt.IsZero() {
			t.Fatal("expected dispatcher timestamps")
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for delivery")
	}
	shutdownDispatcher(t, dispatcher)
}

func TestDispatcherRetriesHandlerFailure(t *testing.T) {
	dispatcher := NewDispatcher(zap.NewNop(), Options{WorkerCount: 1, MaxAttempts: 3, RetryDelay: time.Millisecond})
	var calls atomic.Int32
	completed := make(chan struct{}, 1)
	if err := dispatcher.Register(testHandler{id: "retry", types: []Type{testEventType}, handle: func(_ context.Context, _ Event) error {
		if calls.Add(1) < 3 {
			return errors.New("temporary failure")
		}
		completed <- struct{}{}
		return nil
	}}); err != nil {
		t.Fatalf("register handler: %v", err)
	}
	if err := dispatcher.Start(context.Background()); err != nil {
		t.Fatalf("start dispatcher: %v", err)
	}
	if _, err := dispatcher.Publish(context.Background(), newTestEvent(t, "event-2"), PublishOptions{}); err != nil {
		t.Fatalf("publish event: %v", err)
	}
	select {
	case <-completed:
		if actual := calls.Load(); actual != 3 {
			t.Fatalf("expected three attempts, got %d", actual)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for retry")
	}
	shutdownDispatcher(t, dispatcher)
}

func TestDispatcherRejectsDurableAndUnknownEvents(t *testing.T) {
	dispatcher := NewDispatcher(zap.NewNop(), Options{})
	if err := dispatcher.Start(context.Background()); err != nil {
		t.Fatalf("start dispatcher: %v", err)
	}
	if _, err := dispatcher.Publish(context.Background(), newTestEvent(t, "event-3"), PublishOptions{}); !errors.Is(err, ErrNoHandlers) {
		t.Fatalf("expected missing handler error, got %v", err)
	}
	if _, err := dispatcher.Publish(context.Background(), newTestEvent(t, "event-4"), PublishOptions{Delivery: DeliveryDurable}); !errors.Is(err, ErrDurableUnavailable) {
		t.Fatalf("expected durable unavailable error, got %v", err)
	}
	shutdownDispatcher(t, dispatcher)
}

func TestDurableDispatcherRecoversPerConsumerDelivery(t *testing.T) {
	store := &memoryOutboxStore{completed: make(chan ClaimedDelivery, 1)}
	dispatcher, err := NewDurableDispatcher(zap.NewNop(), Options{WorkerCount: 1, OutboxPoll: time.Millisecond}, store)
	if err != nil {
		t.Fatalf("new durable dispatcher: %v", err)
	}
	if err := dispatcher.Register(testHandler{id: "durable-receiver", types: []Type{testEventType}, handle: func(context.Context, Event) error {
		return nil
	}}); err != nil {
		t.Fatalf("register handler: %v", err)
	}
	if err := dispatcher.Start(context.Background()); err != nil {
		t.Fatalf("start dispatcher: %v", err)
	}
	if receipt, err := dispatcher.Publish(context.Background(), newTestEvent(t, "durable-1"), PublishOptions{Delivery: DeliveryDurable}); err != nil {
		t.Fatalf("publish durable event: %v", err)
	} else if receipt.Delivery != DeliveryDurable {
		t.Fatalf("expected durable receipt, got %s", receipt.Delivery)
	}
	select {
	case completed := <-store.completed:
		if completed.ConsumerID != "durable-receiver" || completed.Attempt != 1 {
			t.Fatalf("unexpected completed delivery: %+v", completed)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for durable delivery")
	}
	shutdownDispatcher(t, dispatcher)
}

func TestDispatcherShutdownDrainsAcceptedEvents(t *testing.T) {
	dispatcher := NewDispatcher(zap.NewNop(), Options{WorkerCount: 1})
	var delivered atomic.Int32
	if err := dispatcher.Register(testHandler{id: "drain", types: []Type{testEventType}, handle: func(_ context.Context, _ Event) error {
		delivered.Add(1)
		return nil
	}}); err != nil {
		t.Fatalf("register handler: %v", err)
	}
	if err := dispatcher.Start(context.Background()); err != nil {
		t.Fatalf("start dispatcher: %v", err)
	}
	for index := 0; index < 4; index++ {
		if _, err := dispatcher.Publish(context.Background(), newTestEvent(t, "drain-"+string(rune('a'+index))), PublishOptions{}); err != nil {
			t.Fatalf("publish event %d: %v", index, err)
		}
	}
	shutdownDispatcher(t, dispatcher)
	if actual := delivered.Load(); actual != 4 {
		t.Fatalf("expected all accepted events to drain, got %d", actual)
	}
	if _, err := dispatcher.Publish(context.Background(), newTestEvent(t, "after-stop"), PublishOptions{}); !errors.Is(err, ErrDispatcherStopped) {
		t.Fatalf("expected stopped error, got %v", err)
	}
}

func TestDispatcherRejectsConcurrentDuplicateHandler(t *testing.T) {
	dispatcher := NewDispatcher(zap.NewNop(), Options{})
	handler := testHandler{id: "duplicate", types: []Type{testEventType}, handle: func(context.Context, Event) error { return nil }}
	var successful atomic.Int32
	var group sync.WaitGroup
	for index := 0; index < 2; index++ {
		group.Add(1)
		go func() {
			defer group.Done()
			if dispatcher.Register(handler) == nil {
				successful.Add(1)
			}
		}()
	}
	group.Wait()
	if actual := successful.Load(); actual != 1 {
		t.Fatalf("expected one successful registration, got %d", actual)
	}
}

func shutdownDispatcher(t *testing.T, dispatcher *Dispatcher) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := dispatcher.Shutdown(ctx); err != nil {
		t.Fatalf("shutdown dispatcher: %v", err)
	}
}
