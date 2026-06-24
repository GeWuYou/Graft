package container

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/moby/moby/api/types/container"
)

func TestDockerRuntimeCollectStatsSnapshotsReturnsStaleCacheDuringRefreshFailure(t *testing.T) {
	t.Parallel()

	start := time.Unix(1_700_000_000, 0)
	now := start
	client := &countingDockerClient{
		list: []container.Summary{
			{
				ID:      "1234567890abcdef",
				Names:   []string{"/graft-web"},
				Image:   "graft/web:latest",
				State:   container.StateRunning,
				Status:  "Up 10 minutes",
				Created: 1781409600,
			},
		},
		statsSequence: []container.StatsResponse{
			baselineDockerStatsFixture(),
			richDockerStatsFixture(),
		},
	}
	cache := newResourceStatsCache(2*time.Second, 5*time.Second)
	cache.now = func() time.Time { return now }
	runtime := &DockerRuntime{
		client:        client,
		endpoint:      "unix:///var/run/docker.sock",
		resourceStats: cache,
		cpuBaselines:  make(map[string]dockerCPUStatsBaseline),
	}

	if _, err := runtime.CollectStatsSnapshots(context.Background()); err != nil {
		t.Fatalf("collect baseline stats snapshots: %v", err)
	}

	first, err := runtime.CollectStatsSnapshots(context.Background())
	if err != nil {
		t.Fatalf("collect initial stats snapshots: %v", err)
	}
	assertSingleFreshSnapshot(t, first, start.UTC(), "initial")

	now = now.Add(3 * time.Second)
	client.statsErr = timeoutError{}

	stale, err := runtime.CollectStatsSnapshots(context.Background())
	if err != nil {
		t.Fatalf("collect stale snapshots: %v", err)
	}
	if len(stale) != 1 {
		t.Fatalf("expected one stale snapshot, got %#v", stale)
	}
	assertSingleFreshSnapshot(t, stale, start.UTC(), "stale")
	assertInt64Ptr(t, stale[0].Resource.MemoryUsageBytes, 256, "stale memory usage bytes")
	assertInt64Ptr(t, stale[0].Resource.MemoryLimitBytes, 1024, "stale memory limit bytes")
	assertFloatPtr(t, stale[0].Resource.MemoryPercent, 25, "stale memory percent")
	if stale[0].Resource.CPUPercent != nil {
		t.Fatalf("expected stale snapshot to preserve first cached one-shot cpu=nil, got %#v", stale[0].Resource.CPUPercent)
	}

	waitForAtomicInt64(t, &client.statsCalls, 2)

	current := runtime.currentResourceSummary("1234567890abcdef")
	if !current.Available || !current.StatsAvailable {
		t.Fatalf("expected refresh failure to preserve last successful snapshot, got %#v", current)
	}
	if current.CollectedAt != start.UTC().Format(time.RFC3339) {
		t.Fatalf("expected current resource collected_at to preserve last usable snapshot, got %q", current.CollectedAt)
	}
	assertInt64Ptr(t, current.MemoryUsageBytes, 256, "current stale memory usage bytes")
	assertInt64Ptr(t, current.MemoryLimitBytes, 1024, "current stale memory limit bytes")
	assertFloatPtr(t, current.MemoryPercent, 25, "current stale memory percent")
	if current.CPUPercent != nil {
		t.Fatalf("expected current stale snapshot to preserve first cached one-shot cpu=nil, got %#v", current.CPUPercent)
	}
}

func assertSingleFreshSnapshot(t *testing.T, snapshots []StatsSnapshot, expectedCollectedAt time.Time, label string) {
	t.Helper()

	if len(snapshots) != 1 || !snapshots[0].Resource.Available || !snapshots[0].Resource.StatsAvailable {
		t.Fatalf("expected %s snapshot to carry one usable resource summary, got %#v", label, snapshots)
	}
	expectedRFC3339 := expectedCollectedAt.Format(time.RFC3339)
	if snapshots[0].Resource.CollectedAt != expectedRFC3339 {
		t.Fatalf("expected %s resource collected_at to anchor at %s, got %q", label, expectedRFC3339, snapshots[0].Resource.CollectedAt)
	}
	if !snapshots[0].CollectedAt.Equal(expectedCollectedAt) {
		t.Fatalf("expected %s snapshot collected_at to reflect resource freshness %s, got %s", label, expectedCollectedAt, snapshots[0].CollectedAt)
	}
}

func waitForAtomicInt64(t *testing.T, value *atomic.Int64, expected int64) {
	t.Helper()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if value.Load() >= expected {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("expected atomic value >= %d, got %d", expected, value.Load())
}
