package cachex

import "time"

// Event 描述一次机械缓存操作及其命中、共享和耗时结果。
type Event struct {
	Cache     string
	Backend   string
	Operation string
	Hit       bool
	Shared    bool
	Duration  time.Duration
	Err       error
}

// Metrics 接收缓存操作事件；实现可将其接入指标或诊断系统。
type Metrics interface {
	Observe(Event)
}

type nopMetrics struct{}

// Observe 丢弃缓存事件；nopMetrics 用它提供无副作用的默认实现。
func (nopMetrics) Observe(Event) {}

// NopMetrics 返回丢弃所有事件的指标接收器。
func NopMetrics() Metrics {
	return nopMetrics{}
}
