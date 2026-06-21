// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package cachex

import (
	"fmt"
	"strings"

	"graft/server/internal/cachex/backend"
)

// Manager owns one mechanical cache backend and provisions named caches on top of it.
type Manager struct {
	backend   backend.Backend
	metrics   Metrics
	group     *Group
	namespace string
}

// NewManager 从提供的选项创建一个缓存管理器。
// 它验证命名空间（去除空格后）和后端均不为空；若验证失败则返回错误。
// NewManager 从提供的选项创建一个 Manager，验证命名空间非空且后端非空。若指标或分组未提供，则使用默认实现。
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

// BackendName returns the current backend adapter name.
func (m *Manager) BackendName() string {
	if m == nil || m.backend == nil {
		return ""
	}

	return m.backend.Name()
}

// NewCache provisions one named cache view using manager-owned mechanical dependencies.
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
