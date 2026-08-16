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
	mu         sync.Mutex
	deliveries map[string]*memoryOutboxDelivery
	completed  chan ClaimedDelivery
}

type memoryOutboxDelivery struct {
	delivery     ClaimedDelivery
	status       string
	availableAt  time.Time
	leaseExpires time.Time
	failedAt     time.Time
}

func (s *memoryOutboxStore) Append(_ context.Context, event Event, consumerIDs []string) (Receipt, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.deliveries == nil {
		s.deliveries = make(map[string]*memoryOutboxDelivery)
	}
	for _, consumerID := range consumerIDs {
		key := memoryDeliveryKey(event.ID, consumerID)
		if _, exists := s.deliveries[key]; exists {
			continue
		}
		s.deliveries[key] = &memoryOutboxDelivery{
			delivery:    ClaimedDelivery{Event: event, ConsumerID: consumerID},
			status:      deliveryPending,
			availableAt: event.CreatedAt,
		}
	}
	return Receipt{EventID: event.ID, Delivery: DeliveryDurable}, nil
}

func (s *memoryOutboxStore) Claim(_ context.Context, owner string, now time.Time, lease time.Duration, limit int) ([]ClaimedDelivery, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	claimed := make([]ClaimedDelivery, 0, limit)
	for _, item := range s.deliveries {
		if len(claimed) == limit || !memoryDeliveryClaimable(item, now) {
			continue
		}
		item.status = deliveryProcessing
		item.delivery.Attempt++
		item.delivery.claimOwner = owner
		item.leaseExpires = now.Add(lease)
		claimed = append(claimed, item.delivery)
	}
	return claimed, nil
}

func (s *memoryOutboxStore) Renew(_ context.Context, delivery ClaimedDelivery, now time.Time, lease time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	item, ok := s.deliveries[memoryDeliveryKey(delivery.Event.ID, delivery.ConsumerID)]
	if !ok || !memoryDeliveryClaimMatches(item, delivery) {
		return ErrClaimLost
	}
	item.leaseExpires = now.Add(lease)
	return nil
}

func (s *memoryOutboxStore) Complete(_ context.Context, delivery ClaimedDelivery) error {
	s.mu.Lock()
	item, ok := s.deliveries[memoryDeliveryKey(delivery.Event.ID, delivery.ConsumerID)]
	if !ok || !memoryDeliveryClaimMatches(item, delivery) {
		s.mu.Unlock()
		return ErrClaimLost
	}
	item.status = deliveryDelivered
	item.delivery.claimOwner = ""
	item.leaseExpires = time.Time{}
	s.mu.Unlock()
	if s.completed != nil {
		select {
		case s.completed <- delivery:
		default:
		}
	}
	return nil
}

func (s *memoryOutboxStore) Retry(_ context.Context, delivery ClaimedDelivery, availableAt time.Time, _ error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	item, ok := s.deliveries[memoryDeliveryKey(delivery.Event.ID, delivery.ConsumerID)]
	if !ok || !memoryDeliveryClaimMatches(item, delivery) {
		return ErrClaimLost
	}
	item.status = deliveryPending
	item.delivery.claimOwner = ""
	item.availableAt = availableAt
	item.leaseExpires = time.Time{}
	return nil
}

func (s *memoryOutboxStore) Fail(_ context.Context, delivery ClaimedDelivery, _ error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	item, ok := s.deliveries[memoryDeliveryKey(delivery.Event.ID, delivery.ConsumerID)]
	if !ok || !memoryDeliveryClaimMatches(item, delivery) {
		return ErrClaimLost
	}
	item.status = deliveryFailed
	item.delivery.claimOwner = ""
	item.leaseExpires = time.Time{}
	item.failedAt = time.Now().UTC()
	return nil
}

func memoryDeliveryKey(eventID, consumerID string) string {
	return eventID + "\x00" + consumerID
}

func memoryDeliveryClaimable(item *memoryOutboxDelivery, now time.Time) bool {
	return (item.status == deliveryPending && !item.availableAt.After(now)) ||
		(item.status == deliveryProcessing && !item.leaseExpires.After(now))
}

func memoryDeliveryClaimMatches(item *memoryOutboxDelivery, delivery ClaimedDelivery) bool {
	return item.status == deliveryProcessing &&
		item.delivery.Attempt == delivery.Attempt &&
		item.delivery.claimOwner == delivery.claimOwner
}

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

func TestMemoryOutboxStoreCompleteDoesNotBlockWithoutCompletionObserver(t *testing.T) {
	for _, testCase := range []struct {
		name      string
		completed chan ClaimedDelivery
	}{
		{name: "no receiver", completed: make(chan ClaimedDelivery)},
		{name: "full buffer", completed: func() chan ClaimedDelivery {
			channel := make(chan ClaimedDelivery, 1)
			channel <- ClaimedDelivery{}
			return channel
		}()},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			store := &memoryOutboxStore{completed: testCase.completed}
			event := newTestEvent(t, "complete-without-observer")
			event.CreatedAt = time.Now().UTC()
			if _, err := store.Append(context.Background(), event, []string{"stable-consumer"}); err != nil {
				t.Fatalf("append durable delivery: %v", err)
			}
			claimed, err := store.Claim(context.Background(), "worker-a", event.CreatedAt, time.Second, 1)
			if err != nil || len(claimed) != 1 {
				t.Fatalf("claim durable delivery: %#v, %v", claimed, err)
			}

			completed := make(chan error, 1)
			go func() { completed <- store.Complete(context.Background(), claimed[0]) }()
			select {
			case err := <-completed:
				if err != nil {
					t.Fatalf("complete durable delivery: %v", err)
				}
			case <-time.After(time.Second):
				t.Fatal("complete blocked without a completion observer")
			}

			store.mu.Lock()
			status := store.deliveries[memoryDeliveryKey(event.ID, "stable-consumer")].status
			store.mu.Unlock()
			if status != deliveryDelivered {
				t.Fatalf("delivery status = %q, want %q", status, deliveryDelivered)
			}
		})
	}
}

func TestMemoryOutboxStoreReclaimsExpiredLease(t *testing.T) {
	store := &memoryOutboxStore{}
	now := time.Now().UTC()
	event := newTestEvent(t, "lease-recovery")
	event.CreatedAt = now
	if _, err := store.Append(context.Background(), event, []string{"stable-consumer"}); err != nil {
		t.Fatalf("append event: %v", err)
	}
	first, err := store.Claim(context.Background(), "worker-a", now, time.Second, 1)
	if err != nil || len(first) != 1 || first[0].Attempt != 1 {
		t.Fatalf("unexpected initial claim: %#v, %v", first, err)
	}
	beforeExpiry, err := store.Claim(context.Background(), "worker-b", now.Add(500*time.Millisecond), time.Second, 1)
	if err != nil || len(beforeExpiry) != 0 {
		t.Fatalf("lease was claimed before expiry: %#v, %v", beforeExpiry, err)
	}
	recovered, err := store.Claim(context.Background(), "worker-b", now.Add(time.Second), time.Second, 1)
	if err != nil || len(recovered) != 1 || recovered[0].Attempt != 2 || recovered[0].claimOwner != "worker-b" {
		t.Fatalf("unexpected recovered claim: %#v, %v", recovered, err)
	}
}

func TestDurableDispatcherRecoversAfterRestartWithStableHandlerID(t *testing.T) {
	store := &memoryOutboxStore{completed: make(chan ClaimedDelivery, 1)}
	firstAttempt := make(chan struct{}, 1)
	first, err := NewDurableDispatcher(zap.NewNop(), Options{WorkerCount: 1, OutboxPoll: time.Millisecond, RetryDelay: time.Hour}, store)
	if err != nil {
		t.Fatalf("new first durable dispatcher: %v", err)
	}
	if err := first.Register(testHandler{id: "stable-consumer", types: []Type{testEventType}, handle: func(context.Context, Event) error {
		firstAttempt <- struct{}{}
		return errors.New("temporary failure")
	}}); err != nil {
		t.Fatalf("register first handler: %v", err)
	}
	if err := first.Start(context.Background()); err != nil {
		t.Fatalf("start first dispatcher: %v", err)
	}
	if _, err := first.Publish(context.Background(), newTestEvent(t, "restart-recovery"), PublishOptions{Delivery: DeliveryDurable}); err != nil {
		t.Fatalf("publish durable event: %v", err)
	}
	select {
	case <-firstAttempt:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for failed first delivery")
	}
	shutdownDispatcher(t, first)
	store.mu.Lock()
	store.deliveries[memoryDeliveryKey("restart-recovery", "stable-consumer")].availableAt = time.Now().UTC()
	store.mu.Unlock()

	second, err := NewDurableDispatcher(zap.NewNop(), Options{WorkerCount: 1, OutboxPoll: time.Millisecond}, store)
	if err != nil {
		t.Fatalf("new second durable dispatcher: %v", err)
	}
	if err := second.Register(testHandler{id: "stable-consumer", types: []Type{testEventType}, handle: func(context.Context, Event) error {
		return nil
	}}); err != nil {
		t.Fatalf("register second handler: %v", err)
	}
	if err := second.Start(context.Background()); err != nil {
		t.Fatalf("start second dispatcher: %v", err)
	}
	select {
	case completed := <-store.completed:
		if completed.ConsumerID != "stable-consumer" || completed.Attempt != 2 {
			t.Fatalf("unexpected recovered delivery: %+v", completed)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for recovered durable delivery")
	}
	shutdownDispatcher(t, second)
}

func TestDurableDispatcherTerminalizesAfterMaxAttempts(t *testing.T) {
	store := &memoryOutboxStore{}
	dispatcher, err := NewDurableDispatcher(zap.NewNop(), Options{
		WorkerCount: 1, OutboxPoll: time.Millisecond, MaxAttempts: 2, RetryDelay: time.Millisecond,
	}, store)
	if err != nil {
		t.Fatalf("new durable dispatcher: %v", err)
	}
	if err := dispatcher.Register(testHandler{id: "always-fails", types: []Type{testEventType}, handle: func(context.Context, Event) error {
		return errors.New("permanent failure")
	}}); err != nil {
		t.Fatalf("register handler: %v", err)
	}
	if err := dispatcher.Start(context.Background()); err != nil {
		t.Fatalf("start dispatcher: %v", err)
	}
	if _, err := dispatcher.Publish(context.Background(), newTestEvent(t, "terminal-failure"), PublishOptions{Delivery: DeliveryDurable}); err != nil {
		t.Fatalf("publish durable event: %v", err)
	}
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		store.mu.Lock()
		item := store.deliveries[memoryDeliveryKey("terminal-failure", "always-fails")]
		terminal := item != nil && item.status == deliveryFailed && item.delivery.Attempt == 2 && !item.failedAt.IsZero()
		store.mu.Unlock()
		if terminal {
			claimed, err := store.Claim(context.Background(), "next-worker", time.Now().UTC(), time.Second, 1)
			if err != nil || len(claimed) != 0 {
				shutdownDispatcher(t, dispatcher)
				t.Fatalf("failed delivery was claimable again: %#v, %v", claimed, err)
			}
			shutdownDispatcher(t, dispatcher)
			return
		}
		time.Sleep(time.Millisecond)
	}
	shutdownDispatcher(t, dispatcher)
	t.Fatal("durable delivery did not enter failed terminal state")
}

func TestDurableDispatcherDoesNotHandleRecoveredAttemptBeyondMaximum(t *testing.T) {
	store := &memoryOutboxStore{}
	dispatcher, err := NewDurableDispatcher(zap.NewNop(), Options{MaxAttempts: 2}, store)
	if err != nil {
		t.Fatalf("new durable dispatcher: %v", err)
	}
	var handled atomic.Int32
	if err := dispatcher.Register(testHandler{id: "final-attempt", types: []Type{testEventType}, handle: func(context.Context, Event) error {
		handled.Add(1)
		return nil
	}}); err != nil {
		t.Fatalf("register handler: %v", err)
	}

	now := time.Now().UTC()
	event := newTestEvent(t, "recovered-final-attempt")
	event.CreatedAt = now
	if _, err := store.Append(context.Background(), event, []string{"final-attempt"}); err != nil {
		t.Fatalf("append durable delivery: %v", err)
	}
	if _, err := store.Claim(context.Background(), "worker-a", now, time.Second, 1); err != nil {
		t.Fatalf("claim initial attempt: %v", err)
	}
	if _, err := store.Claim(context.Background(), "worker-b", now.Add(time.Second), time.Second, 1); err != nil {
		t.Fatalf("claim final permitted attempt: %v", err)
	}
	recovered, err := store.Claim(context.Background(), "worker-c", now.Add(2*time.Second), time.Second, 1)
	if err != nil || len(recovered) != 1 || recovered[0].Attempt != 3 {
		t.Fatalf("unexpected final recovered attempt: %#v, %v", recovered, err)
	}

	dispatcher.work.Add(1)
	dispatcher.deliverDurable(context.Background(), recovered[0])
	if got := handled.Load(); got != 0 {
		t.Fatalf("expected exhausted delivery to skip handler, got %d calls", got)
	}
	store.mu.Lock()
	item := store.deliveries[memoryDeliveryKey(event.ID, "final-attempt")]
	terminal := item != nil && item.status == deliveryFailed && !item.failedAt.IsZero()
	store.mu.Unlock()
	if !terminal {
		t.Fatal("expected exhausted recovered delivery to be terminally failed")
	}
}

func TestDurableDispatcherRenewsLeaseWhileHandlerRuns(t *testing.T) {
	store := &memoryOutboxStore{}
	dispatcher, err := NewDurableDispatcher(zap.NewNop(), Options{HandlerTimeout: 100 * time.Millisecond, LeaseDuration: 10 * time.Millisecond}, store)
	if err != nil {
		t.Fatalf("new durable dispatcher: %v", err)
	}
	started := make(chan struct{})
	finish := make(chan struct{})
	if err := dispatcher.Register(testHandler{id: "slow-handler", types: []Type{testEventType}, handle: func(context.Context, Event) error {
		close(started)
		<-finish
		return nil
	}}); err != nil {
		t.Fatalf("register handler: %v", err)
	}

	now := time.Now().UTC()
	event := newTestEvent(t, "renewed-lease")
	event.CreatedAt = now
	if _, err := store.Append(context.Background(), event, []string{"slow-handler"}); err != nil {
		t.Fatalf("append durable delivery: %v", err)
	}
	claimed, err := store.Claim(context.Background(), "worker-a", now, 10*time.Millisecond, 1)
	if err != nil || len(claimed) != 1 {
		t.Fatalf("claim durable delivery: %#v, %v", claimed, err)
	}

	dispatcher.work.Add(1)
	delivered := make(chan struct{})
	go func() {
		dispatcher.deliverDurable(context.Background(), claimed[0])
		close(delivered)
	}()
	<-started
	time.Sleep(15 * time.Millisecond)
	reclaimed, err := store.Claim(context.Background(), "worker-b", time.Now().UTC(), 10*time.Millisecond, 1)
	if err != nil || len(reclaimed) != 0 {
		close(finish)
		<-delivered
		t.Fatalf("renewed delivery was reclaimed: %#v, %v", reclaimed, err)
	}
	close(finish)
	<-delivered

	store.mu.Lock()
	status := store.deliveries[memoryDeliveryKey(event.ID, "slow-handler")].status
	store.mu.Unlock()
	if status != deliveryDelivered {
		t.Fatalf("delivery status = %q, want %q", status, deliveryDelivered)
	}
}

func TestDurableDispatcherTerminalizesMissingHandler(t *testing.T) {
	store := &memoryOutboxStore{}
	event := newTestEvent(t, "missing-handler")
	event.CreatedAt = time.Now().UTC()
	if _, err := store.Append(context.Background(), event, []string{"removed-consumer"}); err != nil {
		t.Fatalf("append durable delivery: %v", err)
	}
	dispatcher, err := NewDurableDispatcher(zap.NewNop(), Options{WorkerCount: 1, OutboxPoll: time.Millisecond}, store)
	if err != nil {
		t.Fatalf("new durable dispatcher: %v", err)
	}
	if err := dispatcher.Start(context.Background()); err != nil {
		t.Fatalf("start dispatcher: %v", err)
	}
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		store.mu.Lock()
		item := store.deliveries[memoryDeliveryKey("missing-handler", "removed-consumer")]
		terminal := item != nil && item.status == deliveryFailed && item.delivery.Attempt == 1 && !item.failedAt.IsZero()
		store.mu.Unlock()
		if terminal {
			shutdownDispatcher(t, dispatcher)
			return
		}
		time.Sleep(time.Millisecond)
	}
	shutdownDispatcher(t, dispatcher)
	t.Fatal("missing durable handler did not enter failed terminal state")
}

func TestDispatcherCapsRetryDelay(t *testing.T) {
	dispatcher := NewDispatcher(zap.NewNop(), Options{RetryDelay: 100 * time.Millisecond, MaxRetryDelay: 250 * time.Millisecond})
	for attempt, want := range map[int]time.Duration{1: 100 * time.Millisecond, 2: 200 * time.Millisecond, 3: 250 * time.Millisecond, 4: 250 * time.Millisecond} {
		if got := dispatcher.retryDelay(attempt); got != want {
			t.Fatalf("retry delay for attempt %d: got %s want %s", attempt, got, want)
		}
	}
}

func TestDispatcherShutdownReturnsAtDeadlineWhenHandlerBlocks(t *testing.T) {
	dispatcher := NewDispatcher(zap.NewNop(), Options{WorkerCount: 1, HandlerTimeout: time.Second})
	started := make(chan struct{}, 1)
	blocked := make(chan struct{})
	if err := dispatcher.Register(testHandler{id: "blocking", types: []Type{testEventType}, handle: func(context.Context, Event) error {
		started <- struct{}{}
		<-blocked
		return nil
	}}); err != nil {
		t.Fatalf("register handler: %v", err)
	}
	if err := dispatcher.Start(context.Background()); err != nil {
		t.Fatalf("start dispatcher: %v", err)
	}
	if _, err := dispatcher.Publish(context.Background(), newTestEvent(t, "shutdown-deadline"), PublishOptions{}); err != nil {
		t.Fatalf("publish event: %v", err)
	}
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("blocking handler did not start")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	if err := dispatcher.Shutdown(ctx); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("shutdown error: got %v want deadline exceeded", err)
	}
	dispatcher.mu.RLock()
	stopped, stateStarted, accepting, forcedStop := dispatcher.stopped, dispatcher.started, dispatcher.accepting, dispatcher.forcedStop
	dispatcher.mu.RUnlock()
	if !stopped || stateStarted || accepting || !forcedStop {
		t.Fatalf("dispatcher state after forced shutdown: stopped=%v started=%v accepting=%v forcedStop=%v", stopped, stateStarted, accepting, forcedStop)
	}
	if err := dispatcher.Start(context.Background()); !errors.Is(err, ErrDispatcherStopped) {
		t.Fatalf("expected terminal dispatcher to reject restart, got %v", err)
	}
	if err := dispatcher.Shutdown(context.Background()); err != nil {
		t.Fatalf("expected repeated shutdown to be idempotent, got %v", err)
	}
	close(blocked)
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

func TestDispatcherRejectsHandlerIDWithSurroundingWhitespace(t *testing.T) {
	dispatcher := NewDispatcher(zap.NewNop(), Options{})
	err := dispatcher.Register(testHandler{id: " stable-consumer ", types: []Type{testEventType}, handle: func(context.Context, Event) error { return nil }})
	if err == nil {
		t.Fatal("expected handler ID whitespace to be rejected")
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
