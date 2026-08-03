package network

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"

	"graft/server/internal/moduleapi"
)

// DiagnosticRegistry 保存固定注册的诊断目标，并以复制快照避免向调用方暴露内部映射。
type DiagnosticRegistry struct {
	entries *outboundRegistrationRegistry
}

// ConsumerRegistry 保存显式注册的平台策略消费者。
type ConsumerRegistry struct {
	entries *outboundRegistrationRegistry
}

// NewConsumerRegistry 创建空的出站网络消费者注册表。
func NewConsumerRegistry() *ConsumerRegistry {
	return &ConsumerRegistry{entries: newOutboundRegistrationRegistry()}
}

// RegisterOutboundNetworkConsumer 注册唯一的模块消费者。
func (r *ConsumerRegistry) RegisterOutboundNetworkConsumer(consumer moduleapi.OutboundNetworkConsumer) error {
	if r == nil || r.entries == nil {
		return errors.New("outbound network consumer registry is unavailable")
	}
	if consumer == nil {
		return errors.New("outbound network consumer is invalid")
	}
	return r.entries.register(consumer.Name(), consumer.DisplayName(), consumer, "outbound network consumer")
}

// OutboundNetworkConsumers 返回按稳定名称排序的消费者快照。
func (r *ConsumerRegistry) OutboundNetworkConsumers() []moduleapi.OutboundNetworkConsumer {
	if r == nil {
		return nil
	}
	entries := r.entries.items()
	consumers := make([]moduleapi.OutboundNetworkConsumer, 0, len(entries))
	for _, entry := range entries {
		consumers = append(consumers, entry.(moduleapi.OutboundNetworkConsumer))
	}
	return consumers
}

// NewDiagnosticRegistry 创建空的出站网络诊断目标注册表。
func NewDiagnosticRegistry() *DiagnosticRegistry {
	return &DiagnosticRegistry{entries: newOutboundRegistrationRegistry()}
}

// RegisterOutboundDiagnosticTarget 注册唯一的固定诊断目标。
func (r *DiagnosticRegistry) RegisterOutboundDiagnosticTarget(target moduleapi.OutboundDiagnosticTarget) error {
	if r == nil || r.entries == nil {
		return errors.New("outbound diagnostic registry is unavailable")
	}
	if target == nil {
		return errors.New("outbound diagnostic target is invalid")
	}
	return r.entries.register(target.Name(), target.DisplayName(), target, "outbound diagnostic target")
}

// OutboundDiagnosticTarget 按稳定名称读取一个固定诊断目标。
func (r *DiagnosticRegistry) OutboundDiagnosticTarget(name string) (moduleapi.OutboundDiagnosticTarget, bool) {
	if r == nil || r.entries == nil {
		return nil, false
	}
	entry, found := r.entries.get(name)
	if !found {
		return nil, false
	}
	target, ok := entry.(moduleapi.OutboundDiagnosticTarget)
	return target, ok
}

// OutboundDiagnosticTargets 返回按目标名称排序的注册目标快照。
func (r *DiagnosticRegistry) OutboundDiagnosticTargets() []moduleapi.OutboundDiagnosticTarget {
	if r == nil || r.entries == nil {
		return nil
	}
	entries := r.entries.items()
	targets := make([]moduleapi.OutboundDiagnosticTarget, 0, len(entries))
	for _, entry := range entries {
		targets = append(targets, entry.(moduleapi.OutboundDiagnosticTarget))
	}
	return targets
}

type outboundRegistrationRegistry struct {
	mu      sync.RWMutex
	entries map[string]any
}

func newOutboundRegistrationRegistry() *outboundRegistrationRegistry {
	return &outboundRegistrationRegistry{entries: make(map[string]any)}
}

func (r *outboundRegistrationRegistry) register(name string, displayName string, entry any, resource string) error {
	name = strings.TrimSpace(name)
	if name == "" || strings.TrimSpace(displayName) == "" {
		return fmt.Errorf("%s is invalid", resource)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.entries[name]; exists {
		return fmt.Errorf("%s is already registered", resource)
	}
	r.entries[name] = entry
	return nil
}

func (r *outboundRegistrationRegistry) get(name string) (any, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	entry, ok := r.entries[strings.TrimSpace(name)]
	return entry, ok
}

func (r *outboundRegistrationRegistry) items() []any {
	r.mu.RLock()
	names := make([]string, 0, len(r.entries))
	for name := range r.entries {
		names = append(names, name)
	}
	sort.Strings(names)
	items := make([]any, 0, len(names))
	for _, name := range names {
		items = append(items, r.entries[name])
	}
	r.mu.RUnlock()
	return items
}
