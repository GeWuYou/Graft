package runtimetarget

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	generated "graft/server/internal/contract/openapi/generated"
	store "graft/server/modules/runtime-target/store"
)

func TestFilesystemUsageMetricTreatsZeroTotalBlocksAsUnavailable(t *testing.T) {
	metric := filesystemUsageMetric(0, 0, 4096)
	if metric.Available || metric.UnavailableReason == "" {
		t.Fatalf("filesystem metric = %#v", metric)
	}
}

func TestFilesystemUsageMetricTreatsZeroFreeBlocksAsFull(t *testing.T) {
	metric := filesystemUsageMetric(8, 0, 4096)
	if !metric.Available || metric.TotalBytes != 32768 || metric.UsedBytes != 32768 || metric.UsagePercent != 100 {
		t.Fatalf("filesystem metric = %#v", metric)
	}
}

func TestSummaryHealthDoesNotDependOnOptionalMetrics(t *testing.T) {
	cache := newSummaryCache()
	checkedAt := time.Now().UTC()
	cache.entries[1] = summaryCacheEntry{
		summary: targetRuntimeSummary{
			Healthy:         true,
			CheckedAt:       checkedAt,
			OperatingSystem: "Ubuntu 24.04.2 LTS",
			HostName:        "docker-host",
			Workloads:       targetCountMetric{Available: true, Total: 4, Active: 2},
			CPU:             targetUsageMetric{UnavailableReason: "CPU probe failed"},
			Memory:          targetUsageMetric{Available: true, UsedBytes: 2, TotalBytes: 4, UsagePercent: 50},
			Disk:            targetUsageMetric{UnavailableReason: "Storage probe failed"},
		},
		expiresAt: time.Now().Add(time.Minute),
	}
	module := &Module{summaries: cache}
	target := store.Target{ID: 1, DisplayName: "Local Docker", EndpointLabel: "unix:///var/run/docker.sock"}
	response := module.toHTTPSummary(context.Background(), target)
	if response.Health.Status != generated.RuntimeTargetSummaryHealthStatusHealthy {
		t.Fatalf("health status = %q", response.Health.Status)
	}
	if response.Resources.Cpu.Available || response.Resources.Cpu.UnavailableReason != "CPU probe failed" {
		t.Fatalf("CPU metric = %#v", response.Resources.Cpu)
	}
	if !response.Resources.Workloads.Available || response.Resources.Workloads.Active != 2 {
		t.Fatalf("workload metric = %#v", response.Resources.Workloads)
	}
	if response.Runtime.OperatingSystem != "Ubuntu 24.04.2 LTS" || response.Runtime.HostName != "docker-host" {
		t.Fatalf("runtime identity = %#v", response.Runtime)
	}
	detail := module.toHTTP(context.Background(), target)
	if detail.Runtime.OperatingSystem != "Ubuntu 24.04.2 LTS" || detail.Runtime.HostName != "docker-host" {
		t.Fatalf("runtime detail identity = %#v", detail.Runtime)
	}
}

func TestMapRuntimeTargetSummariesPreservesOrderAndBoundsConcurrency(t *testing.T) {
	items := []store.Target{{ID: 1}, {ID: 2}, {ID: 3}, {ID: 4}, {ID: 5}}
	var active int32
	var maxActive int32
	results := mapRuntimeTargetSummaries(context.Background(), items, 2, func(_ context.Context, item store.Target) generated.RuntimeTargetSummary {
		current := atomic.AddInt32(&active, 1)
		for {
			observed := atomic.LoadInt32(&maxActive)
			if current <= observed || atomic.CompareAndSwapInt32(&maxActive, observed, current) {
				break
			}
		}
		time.Sleep(time.Millisecond)
		atomic.AddInt32(&active, -1)
		return generated.RuntimeTargetSummary{Id: map[uint64]int64{1: 1, 2: 2, 3: 3, 4: 4, 5: 5}[item.ID]}
	})
	if got := atomic.LoadInt32(&maxActive); got > 2 {
		t.Fatalf("maximum concurrency = %d, want at most 2", got)
	}
	for index, result := range results {
		if result.Id != int64(index+1) {
			t.Fatalf("result[%d].id = %d, want %d", index, result.Id, index+1)
		}
	}
}

func TestMapRuntimeTargetSummariesUsesBoundedChildContext(t *testing.T) {
	items := []store.Target{{ID: 1}}
	deadlineOK := false
	results := mapRuntimeTargetSummaries(context.Background(), items, 1, func(ctx context.Context, _ store.Target) generated.RuntimeTargetSummary {
		deadline, ok := ctx.Deadline()
		deadlineOK = ok && time.Until(deadline) > 0 && time.Until(deadline) <= runtimeTargetSummaryTimeout
		return generated.RuntimeTargetSummary{Id: 1}
	})
	if !deadlineOK {
		t.Fatalf("child context did not have a bounded deadline")
	}
	if results[0].Id != 1 {
		t.Fatalf("result id = %d, want 1", results[0].Id)
	}
}
