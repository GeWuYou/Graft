package network

import (
	"errors"
	"sort"
	"strings"
	"sync"

	"graft/server/internal/moduleapi"
)

// DiagnosticRegistry 保存固定注册的诊断目标，并以复制快照避免向调用方暴露内部映射。
type DiagnosticRegistry struct {
	mu      sync.RWMutex
	targets map[string]moduleapi.OutboundDiagnosticTarget
}

// NewDiagnosticRegistry 创建空的出站网络诊断目标注册表。
func NewDiagnosticRegistry() *DiagnosticRegistry {
	return &DiagnosticRegistry{targets: make(map[string]moduleapi.OutboundDiagnosticTarget)}
}

// RegisterOutboundDiagnosticTarget 注册唯一的固定诊断目标。
func (r *DiagnosticRegistry) RegisterOutboundDiagnosticTarget(target moduleapi.OutboundDiagnosticTarget) error {
	if r == nil {
		return errors.New("outbound diagnostic registry is unavailable")
	}
	if target == nil || strings.TrimSpace(target.Name()) == "" || strings.TrimSpace(target.DisplayName()) == "" {
		return errors.New("outbound diagnostic target is invalid")
	}
	name := strings.TrimSpace(target.Name())
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.targets[name]; exists {
		return errors.New("outbound diagnostic target is already registered")
	}
	r.targets[name] = target
	return nil
}

// OutboundDiagnosticTarget 按稳定名称读取一个固定诊断目标。
func (r *DiagnosticRegistry) OutboundDiagnosticTarget(name string) (moduleapi.OutboundDiagnosticTarget, bool) {
	if r == nil {
		return nil, false
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	target, ok := r.targets[strings.TrimSpace(name)]
	return target, ok
}

// OutboundDiagnosticTargets 返回按目标名称排序的注册目标快照。
func (r *DiagnosticRegistry) OutboundDiagnosticTargets() []moduleapi.OutboundDiagnosticTarget {
	if r == nil {
		return nil
	}
	r.mu.RLock()
	targets := make([]moduleapi.OutboundDiagnosticTarget, 0, len(r.targets))
	for _, target := range r.targets {
		targets = append(targets, target)
	}
	r.mu.RUnlock()
	sort.Slice(targets, func(left, right int) bool { return targets[left].Name() < targets[right].Name() })
	return targets
}
