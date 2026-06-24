package container

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

func TestResourceStatsCacheReturnsStaleAndRefreshesInBackground(t *testing.T) {
	t.Parallel()

	start := time.Unix(1_700_000_000, 0)
	now := start
	cache := newResourceStatsCache(2*time.Second, 5*time.Second)
	cache.now = func() time.Time { return now }

	var calls atomic.Int64
	refreshStarted := make(chan struct{})
	releaseRefresh := make(chan struct{})
	refreshDone := make(chan struct{})
	loader := func(context.Context) ResourceSummary {
		value := calls.Add(1)
		if value == 2 {
			close(refreshStarted)
			<-releaseRefresh
			defer close(refreshDone)
		}
		return fullResourceSummary(float64(value), float64(value*10))
	}

	first := cache.get(context.Background(), "container-1", loader)
	assertFloatPtr(t, first.CPUPercent, 1, "initial cpu percent")
	assertFloatPtr(t, first.MemoryPercent, 10, "initial memory percent")
	if calls.Load() != 1 {
		t.Fatalf("expected first load to call loader once, got %d", calls.Load())
	}

	now = now.Add(3 * time.Second)
	stale := cache.get(context.Background(), "container-1", loader)
	assertFloatPtr(t, stale.CPUPercent, 1, "stale cpu percent")
	assertFloatPtr(t, stale.MemoryPercent, 10, "stale memory percent")

	<-refreshStarted
	close(releaseRefresh)
	<-refreshDone

	fresh := cache.get(context.Background(), "container-1", loader)
	assertFloatPtr(t, fresh.CPUPercent, 2, "refreshed cpu percent")
	assertFloatPtr(t, fresh.MemoryPercent, 20, "refreshed memory percent")
	if calls.Load() != 2 {
		t.Fatalf("expected refreshed cache hit to avoid extra load, got %d", calls.Load())
	}
}

func TestResourceStatsCacheRefreshFailurePreservesLastSuccessWithinStaleWindow(t *testing.T) {
	t.Parallel()

	start := time.Unix(1_700_000_000, 0)
	now := start
	cache := newResourceStatsCache(2*time.Second, 5*time.Second)
	cache.now = func() time.Time { return now }

	var calls atomic.Int64
	refreshStarted := make(chan struct{})
	releaseRefresh := make(chan struct{})
	refreshDone := make(chan struct{})
	loader := func(context.Context) ResourceSummary {
		switch calls.Add(1) {
		case 1:
			return fullResourceSummary(0.6, 12)
		case 2:
			close(refreshStarted)
			<-releaseRefresh
			defer close(refreshDone)
			return unavailableResourceSummary(containerStatsTimeoutReason)
		default:
			return unavailableResourceSummary(containerStatsTimeoutReason)
		}
	}

	first := cache.get(context.Background(), "container-1", loader)
	assertFloatPtr(t, first.CPUPercent, 0.6, "initial cpu percent")
	assertFloatPtr(t, first.MemoryPercent, 12, "initial memory percent")

	now = now.Add(3 * time.Second)
	stale := cache.get(context.Background(), "container-1", loader)
	assertFloatPtr(t, stale.CPUPercent, 0.6, "stale cpu percent")
	assertFloatPtr(t, stale.MemoryPercent, 12, "stale memory percent")

	<-refreshStarted
	close(releaseRefresh)
	<-refreshDone

	afterFailure := cache.get(context.Background(), "container-1", loader)
	assertFloatPtr(t, afterFailure.CPUPercent, 0.6, "preserved cpu percent")
	assertFloatPtr(t, afterFailure.MemoryPercent, 12, "preserved memory percent")
	if !cache.items["container-1"].updatedAt.Equal(start) {
		t.Fatalf("expected last successful snapshot time to remain anchored at %s, got %s", start, cache.items["container-1"].updatedAt)
	}
}

func TestResourceStatsCachePartialRefreshPromotesUsablePartialSnapshot(t *testing.T) {
	t.Parallel()

	start := time.Unix(1_700_000_000, 0)
	now := start
	cache := newResourceStatsCache(2*time.Second, 5*time.Second)
	cache.now = func() time.Time { return now }

	var calls atomic.Int64
	refreshStarted := make(chan struct{})
	releaseRefresh := make(chan struct{})
	refreshDone := make(chan struct{})
	loader := func(context.Context) ResourceSummary {
		switch calls.Add(1) {
		case 1:
			return fullResourceSummary(0.6, 12)
		case 2:
			close(refreshStarted)
			<-releaseRefresh
			defer close(refreshDone)
			return partialResourceSummary(cpuOnlySummary(0.8))
		default:
			return partialResourceSummary(cpuOnlySummary(0.8))
		}
	}

	first := cache.get(context.Background(), "container-1", loader)
	assertFloatPtr(t, first.CPUPercent, 0.6, "initial cpu percent")
	assertFloatPtr(t, first.MemoryPercent, 12, "initial memory percent")

	now = now.Add(3 * time.Second)
	stale := cache.get(context.Background(), "container-1", loader)
	assertFloatPtr(t, stale.CPUPercent, 0.6, "stale cpu percent")
	assertFloatPtr(t, stale.MemoryPercent, 12, "stale memory percent")

	<-refreshStarted
	close(releaseRefresh)
	<-refreshDone

	afterPartial := cache.get(context.Background(), "container-1", loader)
	assertFloatPtr(t, afterPartial.CPUPercent, 0.8, "refreshed cpu percent")
	if afterPartial.MemoryPercent != nil {
		t.Fatalf("expected refreshed partial snapshot to omit memory percent, got %#v", afterPartial.MemoryPercent)
	}
}

func TestResourceStatsCacheWithoutPriorSnapshotCachesUsablePartialResult(t *testing.T) {
	t.Parallel()

	cache := newResourceStatsCache(2*time.Second, 5*time.Second)
	summary := cache.get(context.Background(), "container-1", func(context.Context) ResourceSummary {
		return partialResourceSummary(memoryOnlySummary(12))
	})

	if !summary.Available || !summary.StatsAvailable {
		t.Fatalf("expected partial result without prior snapshot to stay available, got %#v", summary)
	}
	if summary.CPUPercent != nil {
		t.Fatalf("expected memory-only summary to omit cpu percent, got %#v", summary.CPUPercent)
	}
	assertFloatPtr(t, summary.MemoryPercent, 12, "cached memory-only percent")
	if summary.UnavailableReason != "" || summary.StatsErrorKey != "" || summary.StatsErrorMessage != "" {
		t.Fatalf("expected usable partial result to clear unavailable metadata, got %#v", summary)
	}
	if _, ok := cache.items["container-1"]; !ok {
		t.Fatalf("expected usable partial result to be cached")
	}
}

func fullResourceSummary(cpu float64, memoryPercent float64) ResourceSummary {
	return ResourceSummary{
		Available:      true,
		StatsAvailable: true,
		CPUPercent:     float64Ptr(cpu),
		MemoryPercent:  float64Ptr(memoryPercent),
	}
}

func cpuOnlySummary(cpu float64) ResourceSummary {
	return ResourceSummary{
		Available:      true,
		StatsAvailable: true,
		CPUPercent:     float64Ptr(cpu),
	}
}

func memoryOnlySummary(memoryPercent float64) ResourceSummary {
	return ResourceSummary{
		Available:      true,
		StatsAvailable: true,
		MemoryPercent:  float64Ptr(memoryPercent),
	}
}

func partialResourceSummary(summary ResourceSummary) ResourceSummary {
	summary.StatsErrorKey = "should-not-survive"
	summary.StatsErrorMessage = "should-not-survive"
	return summary
}
