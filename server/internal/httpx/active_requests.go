package httpx

import (
	"context"
	"sync"
	"sync/atomic"

	"graft/server/internal/moduleapi"
)

type activeRequestContextKey struct{}

// activeRequestTracker 持有当前进程普通 HTTP 请求的瞬时计数，不承担历史趋势或持久化职责。
type activeRequestTracker struct {
	count atomic.Int64
}

func newActiveRequestTracker() *activeRequestTracker {
	return &activeRequestTracker{}
}

func (t *activeRequestTracker) begin(ctx context.Context) (context.Context, func()) {
	if t == nil {
		return ctx, func() {}
	}
	t.count.Add(1)
	trackedContext := context.WithValue(ctx, activeRequestContextKey{}, t)
	var once sync.Once
	return trackedContext, func() {
		once.Do(t.decrement)
	}
}

func (t *activeRequestTracker) decrement() {
	for {
		current := t.count.Load()
		if current <= 0 || t.count.CompareAndSwap(current, current-1) {
			return
		}
	}
}

// ReadActiveRequests 返回调用时的活动请求数；若 ctx 属于当前 tracker 管理的请求，则从结果中排除该请求自身。
func (t *activeRequestTracker) ReadActiveRequests(ctx context.Context) int64 {
	if t == nil {
		return 0
	}
	current := t.count.Load()
	if current > 0 && ctx != nil && ctx.Value(activeRequestContextKey{}) == t {
		return current - 1
	}
	return current
}

var _ moduleapi.ActiveRequestReader = (*activeRequestTracker)(nil)
