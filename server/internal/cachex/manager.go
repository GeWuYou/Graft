package cachex

import (
	"fmt"
	"strings"

	"graft/server/internal/cachex/backend"
)

// Manager 持有一个机械缓存后端，并在其上创建带命名空间的缓存视图。
type Manager struct {
	backend   backend.Backend
	metrics   Metrics
	group     *Group
	namespace string
}

// NewManager 从选项创建 Manager；命名空间和后端不能为空，指标与 singleflight 分组缺省时使用默认实现。
func NewManager(options ManagerOptions) (*Manager, error) {
	namespace := strings.TrimSpace(options.Namespace)
	if namespace == "" {
		return nil, fmt.Errorf("cache manager namespace is required")
	}
	if options.Backend == nil {
		return nil, fmt.Errorf("cache manager backend is required")
	}

	metrics := options.Metrics
	if metrics == nil {
		metrics = NopMetrics()
	}

	group := options.Group
	if group == nil {
		group = NewGroup()
	}

	return &Manager{
		backend:   options.Backend,
		metrics:   metrics,
		group:     group,
		namespace: namespace,
	}, nil
}

// BackendName 返回当前后端适配器名称；Manager 不可用时返回空字符串。
func (m *Manager) BackendName() string {
	if m == nil || m.backend == nil {
		return ""
	}

	return m.backend.Name()
}

// NewCache 创建一个继承 Manager 后端、指标和 singleflight 依赖的命名缓存视图。
func (m *Manager) NewCache(name string, options ...Option) (*Cache, error) {
	if m == nil {
		return nil, fmt.Errorf("cache manager is unavailable")
	}

	cacheName := strings.TrimSpace(name)
	if cacheName == "" {
		return nil, fmt.Errorf("cache name is required")
	}

	parsed := defaultCacheOptions()
	for _, option := range options {
		if option == nil {
			continue
		}
		option(&parsed)
	}
	if parsed.Metrics == nil {
		parsed.Metrics = m.metrics
	}
	if parsed.Group == nil {
		parsed.Group = m.group
	}

	return &Cache{
		name:    cacheName,
		keyRoot: fmt.Sprintf("%s:%s", m.namespace, cacheName),
		backend: m.backend,
		ttl:     parsed.TTL,
		metrics: parsed.Metrics,
		group:   parsed.Group,
	}, nil
}
