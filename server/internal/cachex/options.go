package cachex

import (
	"time"

	"graft/server/internal/cachex/backend"
)

// ManagerOptions 配置 Manager 共享的后端、指标、singleflight 分组和命名空间。
type ManagerOptions struct {
	Backend   backend.Backend
	Metrics   Metrics
	Group     *Group
	Namespace string
}

// CacheOptions 配置一个逻辑缓存实例的默认 TTL 及可覆盖的共享依赖。
type CacheOptions struct {
	TTL     time.Duration
	Metrics Metrics
	Group   *Group
}

// Option 修改一个逻辑缓存实例的配置。
type Option func(*CacheOptions)

// WithTTL 设置未指定过期时间的缓存项所使用的默认 TTL。
func WithTTL(ttl time.Duration) Option {
	return func(options *CacheOptions) {
		options.TTL = ttl
	}
}

// WithMetrics 为单个缓存覆盖指标接收器。
func WithMetrics(metrics Metrics) Option {
	return func(options *CacheOptions) {
		options.Metrics = metrics
	}
}

// WithSingleflight 为单个缓存覆盖未命中合并分组。
func WithSingleflight(group *Group) Option {
	return func(options *CacheOptions) {
		options.Group = group
	}
}

// defaultCacheOptions 返回缓存实例的默认配置。
func defaultCacheOptions() CacheOptions {
	return CacheOptions{}
}
