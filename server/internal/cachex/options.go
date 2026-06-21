// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package cachex

import (
	"time"

	"graft/server/internal/cachex/backend"
)

// ManagerOptions configures manager-wide mechanical dependencies.
type ManagerOptions struct {
	Backend   backend.Backend
	Metrics   Metrics
	Group     *Group
	Namespace string
}

// CacheOptions configures one logical cache instance.
type CacheOptions struct {
	TTL     time.Duration
	Metrics Metrics
	Group   *Group
}

// Option mutates one cache option set.
type Option func(*CacheOptions)

// WithTTL sets the default TTL for items that do not specify one.
func WithTTL(ttl time.Duration) Option {
	return func(options *CacheOptions) {
		options.TTL = ttl
	}
}

// WithMetrics overrides the metrics sink for one cache.
func WithMetrics(metrics Metrics) Option {
	return func(options *CacheOptions) {
		options.Metrics = metrics
	}
}

// WithSingleflight overrides the miss-collapse group for one cache.
func WithSingleflight(group *Group) Option {
	return func(options *CacheOptions) {
		options.Group = group
	}
}

// defaultCacheOptions 返回缓存实例的默认配置。
func defaultCacheOptions() CacheOptions {
	return CacheOptions{}
}
