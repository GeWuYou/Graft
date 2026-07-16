package runtimetarget

import (
	"context"
	"errors"
	"sync"
	"time"

	generated "graft/server/internal/contract/openapi/generated"
	"graft/server/internal/realtime"
	contract "graft/server/modules/runtime-target/contract"
)

const runtimeTargetSummaryCollectInterval = time.Second

type runtimeTargetSummaryPublished struct {
	Topic string                           `json:"topic"`
	Items []generated.RuntimeTargetSummary `json:"items"`
}

// runtimeTargetSummaryCollector 仅在主题存在订阅者时发布最新目标快照。
type runtimeTargetSummaryCollector struct {
	collect func(context.Context) []generated.RuntimeTargetSummary
	hub     realtime.Hub

	mu       sync.Mutex
	cancel   context.CancelFunc
	done     chan struct{}
	active   bool
	observer func()
}

// newRuntimeTargetSummaryCollector 创建运行时目标摘要收集器，并配置其实时发布中心和摘要收集函数。
func newRuntimeTargetSummaryCollector(hub realtime.Hub, collect func(context.Context) []generated.RuntimeTargetSummary) *runtimeTargetSummaryCollector {
	return &runtimeTargetSummaryCollector{hub: hub, collect: collect}
}

// Start 启动摘要采集协程；支持主题观察时仅在存在订阅者期间采集，否则保持兼容性地持续采集。
func (c *runtimeTargetSummaryCollector) Start(ctx context.Context) error {
	if c == nil || c.hub == nil || c.collect == nil {
		return nil
	}
	if ctx == nil {
		return errors.New("runtime target collector context is required")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.done != nil {
		return nil
	}
	observer := func() {}
	if observable, ok := c.hub.(realtime.TopicSubscriptionMonitor); ok {
		var err error
		observer, err = observable.RegisterTopicObserver(contract.SummaryTopic, c.activate, c.deactivate)
		if err != nil {
			return err
		}
	} else {
		// 公开 hub 契约不要求支持主题观察；为兼容其它传输方式，继续保持快照采集活跃。
		c.active = true
	}
	runCtx, cancel := context.WithCancel(ctx)
	c.cancel = cancel
	c.done = make(chan struct{})
	c.observer = observer
	go c.run(runCtx, c.done)
	return nil
}

// Stop 注销主题观察器并等待采集协程退出；ctx 超时只影响等待结果，不会重新启动已停止的采集器。
func (c *runtimeTargetSummaryCollector) Stop(ctx context.Context) error {
	if c == nil {
		return nil
	}
	c.mu.Lock()
	cancel, done, observer := c.cancel, c.done, c.observer
	c.cancel, c.done, c.observer, c.active = nil, nil, nil, false
	c.mu.Unlock()
	if observer != nil {
		observer()
	}
	if cancel != nil {
		cancel()
	}
	if done == nil {
		return nil
	}
	if ctx == nil {
		<-done
		return nil
	}
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (c *runtimeTargetSummaryCollector) activate(string) {
	c.mu.Lock()
	c.active = true
	c.mu.Unlock()
}

func (c *runtimeTargetSummaryCollector) deactivate(string) {
	c.mu.Lock()
	c.active = false
	c.mu.Unlock()
}

func (c *runtimeTargetSummaryCollector) run(ctx context.Context, done chan struct{}) {
	defer close(done)
	ticker := time.NewTicker(runtimeTargetSummaryCollectInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			c.mu.Lock()
			active := c.active
			c.mu.Unlock()
			if !active {
				continue
			}
			items := c.collect(ctx)
			if len(items) > 0 {
				c.hub.Publish(contract.SummaryTopic, runtimeTargetSummaryPublished{Topic: contract.SummaryTopic, Items: items})
			}
		}
	}
}

func (m *Module) collectRealtimeSummaries(ctx context.Context) []generated.RuntimeTargetSummary {
	if m == nil || m.repository == nil {
		return nil
	}
	items, err := m.repository.List(ctx)
	if err != nil {
		return nil
	}
	mapped := make([]generated.RuntimeTargetSummary, 0, len(items))
	for _, item := range items {
		mapped = append(mapped, m.toHTTPSummary(ctx, item))
	}
	return mapped
}

// IssueSubscription 校验权限并为目标摘要更新签发 websocket 票据。
func (m *Module) IssueSubscription(ctx context.Context, request realtime.SubscriptionRequest) (realtime.SubscriptionResponse, error) {
	if m == nil || request.Topic != contract.SummaryTopic || request.RequestAuth.User == nil {
		return realtime.SubscriptionResponse{}, realtime.ErrTopicForbidden
	}
	if m.authorizer == nil || m.realtimeTickets == nil {
		return realtime.SubscriptionResponse{}, realtime.ErrTopicConflict
	}
	if err := m.authorizer.Authorize(ctx, request.RequestAuth, contract.ViewPermission); err != nil {
		return realtime.SubscriptionResponse{}, realtime.ErrTopicForbidden
	}
	issued, err := (realtime.TicketIssuer{Tickets: m.realtimeTickets}).IssueTopicTicket(ctx, request)
	if err != nil {
		return realtime.SubscriptionResponse{}, realtime.ErrTopicConflict
	}
	return realtime.SubscriptionResponse{Topic: request.Topic, Ticket: issued.Ticket, WebSocketURL: realtime.BuildTopicWebSocketURL(request.Topic, issued.Ticket), ExpiresAt: issued.ExpiresAt}, nil
}

var _ realtime.SubscriptionIssuer = (*Module)(nil)
