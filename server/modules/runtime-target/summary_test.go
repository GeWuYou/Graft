package runtimetarget

import (
	"context"
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
			Healthy:   true,
			CheckedAt: checkedAt,
			Workloads: targetCountMetric{Available: true, Total: 4, Active: 2},
			CPU:       targetUsageMetric{UnavailableReason: "CPU probe failed"},
			Memory:    targetUsageMetric{Available: true, UsedBytes: 2, TotalBytes: 4, UsagePercent: 50},
			Disk:      targetUsageMetric{UnavailableReason: "Storage probe failed"},
		},
		expiresAt: time.Now().Add(time.Minute),
	}
	response := (&Module{summaries: cache}).toHTTPSummary(context.Background(), store.Target{ID: 1, DisplayName: "Local Docker", EndpointLabel: "unix:///var/run/docker.sock"})
	if response.Health.Status != generated.RuntimeTargetSummaryHealthStatusHealthy {
		t.Fatalf("health status = %q", response.Health.Status)
	}
	if response.Resources.Cpu.Available || response.Resources.Cpu.UnavailableReason != "CPU probe failed" {
		t.Fatalf("CPU metric = %#v", response.Resources.Cpu)
	}
	if !response.Resources.Workloads.Available || response.Resources.Workloads.Active != 2 {
		t.Fatalf("workload metric = %#v", response.Resources.Workloads)
	}
}
