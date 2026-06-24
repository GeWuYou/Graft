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
		stats: richDockerStatsFixture(),
	}
	cache := newResourceStatsCache(2*time.Second, 5*time.Second)
	cache.now = func() time.Time { return now }
	runtime := &DockerRuntime{
		client:        client,
		endpoint:      "unix:///var/run/docker.sock",
		resourceStats: cache,
	}

	first, err := runtime.CollectStatsSnapshots(context.Background())
	if err != nil {
		t.Fatalf("collect initial stats snapshots: %v", err)
	}
	if len(first) != 1 || !first[0].Resource.Available || !first[0].Resource.StatsAvailable {
		t.Fatalf("expected initial snapshot to seed cache, got %#v", first)
	}

	now = now.Add(3 * time.Second)
	client.statsErr = timeoutError{}

	stale, err := runtime.CollectStatsSnapshots(context.Background())
	if err != nil {
		t.Fatalf("collect stale snapshots: %v", err)
	}
	if len(stale) != 1 {
		t.Fatalf("expected one stale snapshot, got %#v", stale)
	}
	if !stale[0].Resource.Available || !stale[0].Resource.StatsAvailable {
		t.Fatalf("expected stale cache snapshot to stay publishable, got %#v", stale[0].Resource)
	}
	assertRichDockerResourceStats(t, stale[0].Resource)

	waitForAtomicInt64(t, &client.statsCalls, 2)

	current := runtime.currentResourceSummary("1234567890abcdef")
	if !current.Available || !current.StatsAvailable {
		t.Fatalf("expected refresh failure to preserve last successful snapshot, got %#v", current)
	}
	assertRichDockerResourceStats(t, current)
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
