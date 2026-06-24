package container

import (
	"context"
	"strings"
	"sync"
	"time"
)

const (
	containerResourceStatsCacheTTL         = 2 * time.Second
	containerResourceStatsCacheStaleWindow = 8 * time.Second
)

type resourceStatsCache struct {
	mu          sync.Mutex
	ttl         time.Duration
	staleWindow time.Duration
	now         func() time.Time
	items       map[string]resourceStatsCacheEntry
	inflight    map[string]*resourceStatsLoad
}

type resourceStatsCacheEntry struct {
	summary    ResourceSummary
	updatedAt  time.Time
	freshUntil time.Time
	staleUntil time.Time
}

type resourceStatsLoader func(context.Context) ResourceSummary

type resourceStatsLoad struct {
	done    chan struct{}
	summary ResourceSummary
}

// 当 ttl 或 staleWindow 小于等于 0 时，会分别使用默认值。
func newResourceStatsCache(ttl time.Duration, staleWindow time.Duration) *resourceStatsCache {
	if ttl <= 0 {
		ttl = containerResourceStatsCacheTTL
	}
	if staleWindow <= 0 {
		staleWindow = containerResourceStatsCacheStaleWindow
	}
	return &resourceStatsCache{
		ttl:         ttl,
		staleWindow: staleWindow,
		now:         time.Now,
		items:       make(map[string]resourceStatsCacheEntry),
		inflight:    make(map[string]*resourceStatsLoad),
	}
}

// get serves fresh snapshots immediately, returns stale snapshots while one background refresh is running,
// and preserves the last successful full snapshot when a refresh degrades.
func (c *resourceStatsCache) get(ctx context.Context, key string, loader resourceStatsLoader) ResourceSummary {
	key = strings.TrimSpace(key)
	if c == nil || loader == nil || key == "" {
		return unavailableResourceSummary(containerStatsIncompleteReason)
	}

	now := c.now()

	c.mu.Lock()
	entry, hasEntry := c.items[key]
	if summary, ok := resourceStatsFreshHit(entry, hasEntry, now); ok {
		c.mu.Unlock()
		return summary
	}
	if load, ok := c.inflight[key]; ok {
		return c.handleInflight(ctx, entry, hasEntry, now, load)
	}
	if summary, ok := c.serveStaleWhileRefresh(ctx, key, entry, hasEntry, now, loader); ok {
		return summary
	}

	load := &resourceStatsLoad{done: make(chan struct{})}
	c.inflight[key] = load
	c.mu.Unlock()

	return c.completeLoad(ctx, key, load, loader)
}

func (c *resourceStatsCache) current(key string) ResourceSummary {
	key = strings.TrimSpace(key)
	if c == nil || key == "" {
		return unavailableResourceSummary(containerStatsNotCollectedReason)
	}
	now := c.now()
	c.mu.Lock()
	defer c.mu.Unlock()
	entry, ok := c.items[key]
	if !ok {
		return unavailableResourceSummary(containerStatsNotCollectedReason)
	}
	if now.Before(entry.staleUntil) {
		return entry.summary
	}
	return unavailableResourceSummary(containerStatsNotCollectedReason)
}

func (c *resourceStatsCache) invalidate(keys ...string) {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, key := range keys {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		delete(c.items, key)
	}
}

// resourceStatsFreshHit 判断缓存条目是否仍处于新鲜期并可直接返回。
// 当条目存在且当前时间早于 freshUntil 时，返回缓存的 ResourceSummary。
func resourceStatsFreshHit(entry resourceStatsCacheEntry, ok bool, now time.Time) (ResourceSummary, bool) {
	if ok && now.Before(entry.freshUntil) {
		return entry.summary, true
	}
	return ResourceSummary{}, false
}

func (c *resourceStatsCache) handleInflight(
	ctx context.Context,
	entry resourceStatsCacheEntry,
	hasEntry bool,
	now time.Time,
	load *resourceStatsLoad,
) ResourceSummary {
	if hasEntry && now.Before(entry.staleUntil) {
		summary := entry.summary
		c.mu.Unlock()
		return summary
	}
	c.mu.Unlock()
	select {
	case <-load.done:
		return load.summary
	case <-ctx.Done():
		return unavailableResourceSummary(resourceStatsErrorReason(ctx.Err()))
	}
}

func (c *resourceStatsCache) serveStaleWhileRefresh(
	ctx context.Context,
	key string,
	entry resourceStatsCacheEntry,
	hasEntry bool,
	now time.Time,
	loader resourceStatsLoader,
) (ResourceSummary, bool) {
	if !hasEntry || !now.Before(entry.staleUntil) {
		return ResourceSummary{}, false
	}
	load := &resourceStatsLoad{done: make(chan struct{})}
	c.inflight[key] = load
	stale := entry.summary
	refreshCtx := context.WithoutCancel(ctx)
	c.mu.Unlock()
	go c.completeLoad(refreshCtx, key, load, loader)
	return stale, true
}

func (c *resourceStatsCache) completeLoad(ctx context.Context, key string, load *resourceStatsLoad, loader resourceStatsLoader) ResourceSummary {
	loaded := loader(ctx)
	summary, cacheable := normalizeResourceStatsSummary(loaded)

	c.mu.Lock()
	defer c.mu.Unlock()
	if cacheable {
		recordedAt := c.now()
		if strings.TrimSpace(summary.CollectedAt) == "" {
			summary.CollectedAt = recordedAt.UTC().Format(time.RFC3339)
		}
		c.items[key] = resourceStatsCacheEntry{
			summary:    summary,
			updatedAt:  recordedAt,
			freshUntil: recordedAt.Add(c.ttl),
			staleUntil: recordedAt.Add(c.ttl + c.staleWindow),
		}
	}
	load.summary = summary
	delete(c.inflight, key)
	close(load.done)
	return summary
}

// normalizeResourceStatsSummary 规范化资源统计摘要，并判断其是否可缓存。
// 当摘要包含可用的统计快照时，清空不可用原因和统计错误信息并返回。
// 否则返回一个不可用摘要。
//
// 返回规范化后的 ResourceSummary；第二个值表示结果是否可缓存。
func normalizeResourceStatsSummary(summary ResourceSummary) (ResourceSummary, bool) {
	if isUsableResourceStatsSnapshot(summary) {
		summary.UnavailableReason = ""
		summary.StatsErrorKey = ""
		summary.StatsErrorMessage = ""
		return summary, true
	}
	return unavailableResourceSummary(resourceStatsUnavailableReason(summary)), false
}

// isUsableResourceStatsSnapshot 判断资源统计快照是否可用。
// 只有在资源可用、统计可用且包含至少一项 CPU 或内存指标时，才视为可用。
func isUsableResourceStatsSnapshot(summary ResourceSummary) bool {
	if !summary.Available || !summary.StatsAvailable {
		return false
	}
	return summary.CPUPercent != nil ||
		summary.MemoryPercent != nil ||
		summary.MemoryUsageBytes != nil ||
		summary.MemoryLimitBytes != nil
}

// resourceStatsUnavailableReason 返回资源统计不可用原因。
//
// 优先使用 summary.UnavailableReason；如果统计数据可用但信息不完整，则返回
// containerStatsIncompleteReason；否则优先使用 summary.StatsErrorKey，最后回退到
// containerStatsIncompleteReason。
func resourceStatsUnavailableReason(summary ResourceSummary) string {
	if reason := strings.TrimSpace(summary.UnavailableReason); reason != "" {
		return reason
	}
	if summary.StatsAvailable {
		return containerStatsIncompleteReason
	}
	if reason := strings.TrimSpace(summary.StatsErrorKey); reason != "" {
		return reason
	}
	return containerStatsIncompleteReason
}
