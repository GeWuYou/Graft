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
	entries             *outboundRegistrationRegistry
	connectivityEntries *connectivityRegistrationRegistry
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
	if r == nil || r.entries == nil {
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
	return &DiagnosticRegistry{entries: newOutboundRegistrationRegistry(), connectivityEntries: newConnectivityRegistrationRegistry()}
}

// RegisterOutboundDiagnosticTarget 注册唯一的固定诊断目标。
func (r *DiagnosticRegistry) RegisterOutboundDiagnosticTarget(target moduleapi.OutboundDiagnosticTarget) error {
	if r == nil || r.entries == nil {
		return errors.New("outbound diagnostic registry is unavailable")
	}
	if target == nil {
		return errors.New("outbound diagnostic target is invalid")
	}
	var connectivityTarget moduleapi.ConnectivityTarget
	if candidate, ok := target.(moduleapi.ConnectivityTarget); ok {
		descriptor := connectivityTargetDescriptorSnapshot(candidate.ConnectivityTargetDescriptor())
		if !validConnectivityTargetDescriptor(descriptor) {
			return errors.New("connectivity target descriptor is invalid")
		}
		if r.connectivityEntries == nil || r.connectivityEntries.has(descriptor.ID) {
			return errors.New("connectivity target is already registered")
		}
		connectivityTarget = candidate
	}
	if err := r.entries.register(target.Name(), target.DisplayName(), target, "outbound diagnostic target"); err != nil {
		return err
	}
	if connectivityTarget != nil {
		if err := r.RegisterConnectivityTarget(connectivityTarget); err != nil {
			return err
		}
	}
	return nil
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

// RegisterConnectivityTarget 注册 canonical 连通性目标。它与 legacy HTTP 诊断并行存在，直到后续 Phase 替换旧存储和 API。
func (r *DiagnosticRegistry) RegisterConnectivityTarget(target moduleapi.ConnectivityTarget) error {
	if r == nil || r.connectivityEntries == nil {
		return errors.New("connectivity target registry is unavailable")
	}
	if target == nil {
		return errors.New("connectivity target is invalid")
	}
	descriptor := target.ConnectivityTargetDescriptor()
	return r.connectivityEntries.register(descriptor, target)
}

// ConnectivityTarget 按稳定 target ID 返回一个 canonical 连通性目标。
func (r *DiagnosticRegistry) ConnectivityTarget(id moduleapi.ConnectivityTargetID) (moduleapi.ConnectivityTarget, bool) {
	if r == nil || r.connectivityEntries == nil {
		return nil, false
	}
	return r.connectivityEntries.get(id)
}

// ConnectivityTargets 返回按稳定 target ID 排序的 canonical 连通性目标快照。
func (r *DiagnosticRegistry) ConnectivityTargets() []moduleapi.ConnectivityTarget {
	if r == nil || r.connectivityEntries == nil {
		return nil
	}
	return r.connectivityEntries.items()
}

// ConnectivityTargetDescriptors 返回按稳定 target ID 排序且不共享 capability slice 的描述快照。
func (r *DiagnosticRegistry) ConnectivityTargetDescriptors() []moduleapi.ConnectivityTargetDescriptor {
	if r == nil || r.connectivityEntries == nil {
		return nil
	}
	return r.connectivityEntries.descriptors()
}

type outboundRegistrationRegistry struct {
	mu      sync.RWMutex
	entries map[string]any
}

type connectivityRegistrationRegistry struct {
	mu      sync.RWMutex
	entries map[moduleapi.ConnectivityTargetID]connectivityRegistration
}

type connectivityRegistration struct {
	target     moduleapi.ConnectivityTarget
	descriptor moduleapi.ConnectivityTargetDescriptor
}

func newConnectivityRegistrationRegistry() *connectivityRegistrationRegistry {
	return &connectivityRegistrationRegistry{entries: make(map[moduleapi.ConnectivityTargetID]connectivityRegistration)}
}

func (r *connectivityRegistrationRegistry) register(descriptor moduleapi.ConnectivityTargetDescriptor, target moduleapi.ConnectivityTarget) error {
	descriptor = connectivityTargetDescriptorSnapshot(descriptor)
	if !validConnectivityTargetDescriptor(descriptor) {
		return errors.New("connectivity target descriptor is invalid")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.entries[descriptor.ID]; exists {
		return errors.New("connectivity target is already registered")
	}
	r.entries[descriptor.ID] = connectivityRegistration{target: target, descriptor: descriptor}
	return nil
}

func validConnectivityTargetDescriptor(descriptor moduleapi.ConnectivityTargetDescriptor) bool {
	return descriptor.ID != "" && descriptor.ModuleID != "" && descriptor.Category != "" && descriptor.TitleKey != ""
}

func (r *connectivityRegistrationRegistry) get(id moduleapi.ConnectivityTargetID) (moduleapi.ConnectivityTarget, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	entry, ok := r.entries[moduleapi.ConnectivityTargetID(strings.TrimSpace(string(id)))]
	return entry.target, ok
}

func (r *connectivityRegistrationRegistry) has(id moduleapi.ConnectivityTargetID) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.entries[id]
	return ok
}

func (r *connectivityRegistrationRegistry) items() []moduleapi.ConnectivityTarget {
	r.mu.RLock()
	ids := make([]string, 0, len(r.entries))
	for id := range r.entries {
		ids = append(ids, string(id))
	}
	sort.Strings(ids)
	items := make([]moduleapi.ConnectivityTarget, 0, len(ids))
	for _, id := range ids {
		items = append(items, r.entries[moduleapi.ConnectivityTargetID(id)].target)
	}
	r.mu.RUnlock()
	return items
}

func (r *connectivityRegistrationRegistry) descriptors() []moduleapi.ConnectivityTargetDescriptor {
	r.mu.RLock()
	ids := make([]string, 0, len(r.entries))
	for id := range r.entries {
		ids = append(ids, string(id))
	}
	sort.Strings(ids)
	items := make([]moduleapi.ConnectivityTargetDescriptor, 0, len(ids))
	for _, id := range ids {
		items = append(items, r.entries[moduleapi.ConnectivityTargetID(id)].descriptor.Snapshot())
	}
	r.mu.RUnlock()
	return items
}

func connectivityTargetDescriptorSnapshot(value moduleapi.ConnectivityTargetDescriptor) moduleapi.ConnectivityTargetDescriptor {
	value.ID = moduleapi.ConnectivityTargetID(strings.TrimSpace(string(value.ID)))
	value.ModuleID = strings.TrimSpace(value.ModuleID)
	value.Category = strings.TrimSpace(value.Category)
	value.TitleKey = strings.TrimSpace(value.TitleKey)
	return value.Snapshot()
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
