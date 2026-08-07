// Package capability owns the server-side capability observation authority.
package capability

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	"graft/server/internal/moduleapi"
)

// Descriptor 是能力静态描述的 coordinator 别名。
type Descriptor = moduleapi.CapabilityDescriptor

// Status 是能力状态的 coordinator 别名。
type Status = moduleapi.CapabilityStatus

// Observation 是能力观测结果的 coordinator 别名。
type Observation = moduleapi.CapabilityObservation

// Provider 是能力 provider 的 coordinator 别名。
type Provider = moduleapi.CapabilityProvider

// Entry 将静态能力描述与其唯一 provider 绑定。
type Entry struct {
	Descriptor moduleapi.CapabilityDescriptor
	Provider   moduleapi.CapabilityProvider
}

// Registry 是静态且确定性的能力声明注册表。
type Registry struct {
	entries map[string]Entry
}

// NewRegistry 校验并构造能力注册表。
func NewRegistry(entries []Entry) (*Registry, error) {
	r := &Registry{entries: make(map[string]Entry, len(entries))}
	for _, entry := range entries {
		if entry.Descriptor.Key == "" {
			return nil, fmt.Errorf("capability key is required")
		}
		if !validCategory(entry.Descriptor.Category) {
			return nil, fmt.Errorf("invalid capability category %q for %q", entry.Descriptor.Category, entry.Descriptor.Key)
		}
		if !validImpact(entry.Descriptor.Impact) {
			return nil, fmt.Errorf("invalid capability impact %q for %q", entry.Descriptor.Impact, entry.Descriptor.Key)
		}
		if entry.Descriptor.StaleAfter < 0 {
			return nil, fmt.Errorf("capability stale duration must not be negative for %q", entry.Descriptor.Key)
		}
		if _, exists := r.entries[entry.Descriptor.Key]; exists {
			return nil, fmt.Errorf("duplicate capability %q", entry.Descriptor.Key)
		}
		if entry.Provider == nil {
			return nil, fmt.Errorf("provider is required for capability %q", entry.Descriptor.Key)
		}
		r.entries[entry.Descriptor.Key] = entry
	}
	return r, nil
}

// Entries 按 key 排序返回注册项，避免调用方依赖 map 顺序。
func (r *Registry) Entries() []Entry {
	keys := make([]string, 0, len(r.entries))
	for key := range r.entries {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	result := make([]Entry, 0, len(keys))
	for _, key := range keys {
		result = append(result, r.entries[key])
	}
	return result
}

// Descriptor 返回指定能力的静态描述。
func (r *Registry) Descriptor(key string) (moduleapi.CapabilityDescriptor, bool) {
	entry, ok := r.entries[key]
	if !ok {
		return moduleapi.CapabilityDescriptor{}, false
	}
	return entry.Descriptor, true
}

// Coordinator 是当前能力观测的唯一事实源；Snapshot 不暴露内部可变状态。
type Coordinator struct {
	registry *Registry
	now      func() time.Time
	mu       sync.RWMutex
	current  map[string]moduleapi.CapabilityObservation
}

// NewCoordinator 创建绑定静态注册表的 coordinator。
func NewCoordinator(registry *Registry) *Coordinator {
	return &Coordinator{registry: registry, now: time.Now, current: make(map[string]moduleapi.CapabilityObservation)}
}

// Snapshot 返回所有已观测能力，并在读取时统一应用 TTL 新鲜度规则。
func (c *Coordinator) Snapshot() map[string]moduleapi.CapabilityObservation {
	c.mu.Lock()
	defer c.mu.Unlock()
	result := make(map[string]moduleapi.CapabilityObservation, len(c.registry.entries))
	for key := range c.registry.entries {
		observation, ok := c.current[key]
		if !ok {
			observation.Status = moduleapi.CapabilityStatusUnknown
		}
		result[key] = c.withFreshness(key, observation)
	}
	return result
}

// RegistryEntries 返回静态能力声明，供 HTTP 投影按稳定顺序构造响应。
func (c *Coordinator) RegistryEntries() []Entry {
	if c == nil || c.registry == nil {
		return nil
	}
	return c.registry.Entries()
}

// Get 返回单项能力观测；未知或尚未观测的 key 返回 false。
func (c *Coordinator) Get(key string) (moduleapi.CapabilityObservation, bool) {
	c.mu.RLock()
	observation, ok := c.current[key]
	c.mu.RUnlock()
	if !ok {
		return moduleapi.CapabilityObservation{}, false
	}
	return c.withFreshness(key, observation), true
}

// Observe 顺序执行静态注册的 provider，并把 provider 错误归一化为 unavailable。
func (c *Coordinator) Observe(ctx context.Context) (map[string]moduleapi.CapabilityObservation, error) {
	entries := c.registry.Entries()
	observations := make(map[string]moduleapi.CapabilityObservation, len(entries))
	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		observation, err := entry.Provider.Observe(ctx)
		if contextErr := ctx.Err(); contextErr != nil {
			return nil, contextErr
		}
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return nil, err
			}
			observation = moduleapi.CapabilityObservation{Status: moduleapi.CapabilityStatusUnavailable, Summary: "Capability observation failed"}
		}
		if !validStatus(observation.Status) {
			observation = moduleapi.CapabilityObservation{Status: moduleapi.CapabilityStatusUnavailable, Summary: "provider returned invalid capability status"}
		}
		if observation.ObservedAt.IsZero() {
			observation.ObservedAt = c.now()
		}
		if entry.Descriptor.StaleAfter > 0 {
			observation.ExpiresAt = observation.ObservedAt.Add(entry.Descriptor.StaleAfter)
		}
		observations[entry.Descriptor.Key] = observation
	}
	c.mu.Lock()
	c.current = observations
	c.mu.Unlock()
	return c.Snapshot(), nil
}

func (c *Coordinator) withFreshness(key string, observation moduleapi.CapabilityObservation) moduleapi.CapabilityObservation {
	entry, ok := c.registry.entries[key]
	if !ok {
		return observation
	}
	if observation.ExpiresAt.IsZero() && entry.Descriptor.StaleAfter > 0 {
		observation.ExpiresAt = observation.ObservedAt.Add(entry.Descriptor.StaleAfter)
	}
	if !observation.ExpiresAt.IsZero() && !c.now().Before(observation.ExpiresAt) {
		observation.Stale = true
		observation.Status = moduleapi.CapabilityStatusUnknown
	}
	return observation
}

func validCategory(category moduleapi.CapabilityCategory) bool {
	switch category {
	case moduleapi.CapabilityCategoryInfrastructure, moduleapi.CapabilityCategoryRuntime,
		moduleapi.CapabilityCategoryStorage, moduleapi.CapabilityCategoryIntegration,
		moduleapi.CapabilityCategorySecurity, moduleapi.CapabilityCategoryObservability,
		moduleapi.CapabilityCategoryPlatform, moduleapi.CapabilityCategoryAI,
		moduleapi.CapabilityCategoryExtension:
		return true
	default:
		return false
	}
}

func validImpact(impact moduleapi.CapabilityImpact) bool {
	switch impact {
	case moduleapi.CapabilityImpactPlatform, moduleapi.CapabilityImpactFeature, moduleapi.CapabilityImpactAdvisory:
		return true
	default:
		return false
	}
}

func validStatus(status moduleapi.CapabilityStatus) bool {
	switch status {
	case moduleapi.CapabilityStatusUnknown, moduleapi.CapabilityStatusChecking,
		moduleapi.CapabilityStatusHealthy, moduleapi.CapabilityStatusDegraded,
		moduleapi.CapabilityStatusUnavailable, moduleapi.CapabilityStatusDisabled,
		moduleapi.CapabilityStatusUnsupported:
		return true
	default:
		return false
	}
}
