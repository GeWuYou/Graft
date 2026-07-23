package event

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
)

const (
	defaultBufferSize     = 1024
	defaultWorkerCount    = 2
	defaultMaxAttempts    = 3
	defaultRetryDelay     = 100 * time.Millisecond
	defaultHandlerTimeout = 5 * time.Second
)

// Options 控制本地内存 dispatcher 的资源上限和失败处理。
type Options struct {
	BufferSize     int
	WorkerCount    int
	MaxAttempts    int
	RetryDelay     time.Duration
	HandlerTimeout time.Duration
}

func (o Options) normalized() Options {
	if o.BufferSize <= 0 {
		o.BufferSize = defaultBufferSize
	}
	if o.WorkerCount <= 0 {
		o.WorkerCount = defaultWorkerCount
	}
	if o.MaxAttempts <= 0 {
		o.MaxAttempts = defaultMaxAttempts
	}
	if o.RetryDelay <= 0 {
		o.RetryDelay = defaultRetryDelay
	}
	if o.HandlerTimeout <= 0 {
		o.HandlerTimeout = defaultHandlerTimeout
	}
	return o
}

// Dispatcher 是 Runtime 持有的有界本地异步事件分发器。
type Dispatcher struct {
	logger  *zap.Logger
	options Options
	queue   chan Event

	mu         sync.RWMutex
	handlers   map[Type][]Handler
	handlerIDs map[string]struct{}
	accepting  bool
	started    bool
	cancel     context.CancelFunc
	work       sync.WaitGroup
	workers    sync.WaitGroup
}

// NewDispatcher 创建尚未启动的 dispatcher；Handler 必须在 Start 前注册。
func NewDispatcher(logger *zap.Logger, options Options) *Dispatcher {
	if logger == nil {
		logger = zap.NewNop()
	}
	options = options.normalized()
	return &Dispatcher{
		logger:     logger,
		options:    options,
		queue:      make(chan Event, options.BufferSize),
		handlers:   make(map[Type][]Handler),
		handlerIDs: make(map[string]struct{}),
	}
}

// Register 声明一个事件消费者。运行后修改注册表会被拒绝，确保消费者集合稳定。
func (d *Dispatcher) Register(handler Handler) error {
	if d == nil || handler == nil {
		return errors.New("event handler is required")
	}
	id := strings.TrimSpace(handler.ID())
	if id == "" {
		return errors.New("event handler id is required")
	}
	types := handler.Types()
	if len(types) == 0 {
		return errors.New("event handler types are required")
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.started {
		return errors.New("event handler registration is closed")
	}
	if _, exists := d.handlerIDs[id]; exists {
		return fmt.Errorf("event handler %q is already registered", id)
	}
	for _, eventType := range types {
		eventType = Type(strings.TrimSpace(string(eventType)))
		if eventType == "" {
			return errors.New("event handler type is required")
		}
		d.handlers[eventType] = append(d.handlers[eventType], handler)
	}
	d.handlerIDs[id] = struct{}{}
	return nil
}

// Start 启动本地 worker pool。重复调用不会额外创建 worker。
func (d *Dispatcher) Start(ctx context.Context) error {
	if d == nil {
		return errors.New("event dispatcher is required")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.started {
		return nil
	}
	runCtx, cancel := context.WithCancel(ctx)
	d.cancel = cancel
	d.accepting = true
	d.started = true
	for index := 0; index < d.options.WorkerCount; index++ {
		d.workers.Add(1)
		go d.runWorker(runCtx)
	}
	return nil
}

// Publish 等待队列接收事件或调用方 context 结束。
func (d *Dispatcher) Publish(ctx context.Context, event Event, options PublishOptions) (Receipt, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	return d.publish(ctx, event, options, false)
}

// PublishAsync 只尝试立即入队；它不会创建未受 Runtime 管理的 goroutine。
func (d *Dispatcher) PublishAsync(event Event, options PublishOptions) (Receipt, error) {
	return d.publish(context.Background(), event, options, true)
}

// PublishBatch 逐项发布并保留所有被拒绝事件的错误，避免假设 channel 批量入队具备原子性。
func (d *Dispatcher) PublishBatch(ctx context.Context, events []Event, options PublishOptions) BatchReceipt {
	result := BatchReceipt{Accepted: make([]Receipt, 0, len(events)), Rejected: make(map[string]error)}
	for _, event := range events {
		receipt, err := d.Publish(ctx, event, options)
		if err != nil {
			result.Rejected[event.ID] = err
			continue
		}
		result.Accepted = append(result.Accepted, receipt)
	}
	return result
}

func (d *Dispatcher) publish(ctx context.Context, event Event, options PublishOptions, nonBlocking bool) (Receipt, error) {
	if d == nil {
		return Receipt{}, ErrDispatcherStopped
	}
	if options.Delivery == "" {
		options.Delivery = DeliveryBestEffort
	}
	if options.Delivery != DeliveryBestEffort {
		return Receipt{}, ErrDurableUnavailable
	}
	normalized, err := event.normalized(time.Now().UTC())
	if err != nil {
		return Receipt{}, err
	}
	d.mu.RLock()
	if !d.accepting || !d.started {
		d.mu.RUnlock()
		return Receipt{}, ErrDispatcherStopped
	}
	if len(d.handlers[normalized.Type]) == 0 {
		d.mu.RUnlock()
		return Receipt{}, fmt.Errorf("%w: %s", ErrNoHandlers, normalized.Type)
	}
	d.work.Add(1)
	d.mu.RUnlock()
	if nonBlocking {
		select {
		case d.queue <- normalized:
			return Receipt{EventID: normalized.ID, Delivery: options.Delivery}, nil
		default:
			d.work.Done()
			return Receipt{}, ErrBackpressure
		}
	}
	select {
	case d.queue <- normalized:
		return Receipt{EventID: normalized.ID, Delivery: options.Delivery}, nil
	case <-ctx.Done():
		d.work.Done()
		return Receipt{}, ctx.Err()
	}
}

// Shutdown 停止接收新事件，并在 deadline 内等待已接收的事件完成。
func (d *Dispatcher) Shutdown(ctx context.Context) error {
	if d == nil {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	d.mu.Lock()
	if !d.started {
		d.mu.Unlock()
		return nil
	}
	d.accepting = false
	cancel := d.cancel
	d.mu.Unlock()

	drained := make(chan struct{})
	go func() {
		d.work.Wait()
		close(drained)
	}()
	select {
	case <-drained:
		cancel()
		d.workers.Wait()
		return nil
	case <-ctx.Done():
		cancel()
		d.workers.Wait()
		return ctx.Err()
	}
}

func (d *Dispatcher) handlersFor(eventType Type) []Handler {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return append([]Handler(nil), d.handlers[eventType]...)
}
