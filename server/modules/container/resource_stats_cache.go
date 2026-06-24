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

//nolint:unused // Kept for the tested stale-while-refresh path guarded in resource_stats_cache_test.go.
type resourceStatsLoader func(context.Context) ResourceSummary

//nolint:unused // Kept for the tested stale-while-refresh path guarded in resource_stats_cache_test.go.
type resourceStatsLoad struct {
	done    chan struct{}
	summary ResourceSummary
}

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
//
//nolint:unused // Covered by focused cache tests; runtime currently consumes current/set/invalidate directly.
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

func (c *resourceStatsCache) set(key string, summary ResourceSummary) {
	key = strings.TrimSpace(key)
	if c == nil || key == "" {
		return
	}
	normalized, cacheable := normalizeResourceStatsSummary(summary)
	if !cacheable {
		return
	}
	recordedAt := c.now()
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items[key] = resourceStatsCacheEntry{
		summary:    normalized,
		updatedAt:  recordedAt,
		freshUntil: recordedAt.Add(c.ttl),
		staleUntil: recordedAt.Add(c.ttl + c.staleWindow),
	}
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

//nolint:unused // Kept for the tested stale-while-refresh path guarded in resource_stats_cache_test.go.
func resourceStatsFreshHit(entry resourceStatsCacheEntry, ok bool, now time.Time) (ResourceSummary, bool) {
	if ok && now.Before(entry.freshUntil) {
		return entry.summary, true
	}
	return ResourceSummary{}, false
}

//nolint:unused // Kept with get() to preserve the tested stale-while-refresh behavior.
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

//nolint:unused // Kept with get() to preserve the tested stale-while-refresh behavior.
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

//nolint:unused // Kept with get() to preserve the tested stale-while-refresh behavior.
func (c *resourceStatsCache) completeLoad(ctx context.Context, key string, load *resourceStatsLoad, loader resourceStatsLoader) ResourceSummary {
	loaded := loader(ctx)
	summary, cacheable := normalizeResourceStatsSummary(loaded)

	c.mu.Lock()
	defer c.mu.Unlock()
	load.summary = summary
	if cacheable {
		recordedAt := c.now()
		c.items[key] = resourceStatsCacheEntry{
			summary:    summary,
			updatedAt:  recordedAt,
			freshUntil: recordedAt.Add(c.ttl),
			staleUntil: recordedAt.Add(c.ttl + c.staleWindow),
		}
	}
	delete(c.inflight, key)
	close(load.done)
	return summary
}

func normalizeResourceStatsSummary(summary ResourceSummary) (ResourceSummary, bool) {
	if isUsableResourceStatsSnapshot(summary) {
		summary.UnavailableReason = ""
		summary.StatsErrorKey = ""
		summary.StatsErrorMessage = ""
		return summary, true
	}
	return unavailableResourceSummary(resourceStatsUnavailableReason(summary)), false
}

func isUsableResourceStatsSnapshot(summary ResourceSummary) bool {
	if !summary.Available || !summary.StatsAvailable {
		return false
	}
	return summary.CPUPercent != nil ||
		summary.MemoryPercent != nil ||
		summary.MemoryUsageBytes != nil ||
		summary.MemoryLimitBytes != nil
}

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
