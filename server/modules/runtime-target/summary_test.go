package runtimetarget

import "testing"

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
