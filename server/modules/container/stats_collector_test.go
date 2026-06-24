package container

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

func TestStatsCollectorStopIsSafeAcrossRestart(t *testing.T) {
	t.Parallel()

	var calls atomic.Int64
	collector := newStatsCollector(func(ctx context.Context) ([]StatsSnapshot, error) {
		calls.Add(1)
		<-ctx.Done()
		return nil, nil
	}, nil, nil, moduleID)
	collector.interval = time.Hour

	if err := collector.Start(context.Background()); err != nil {
		t.Fatalf("start collector first run: %v", err)
	}
	if err := collector.Stop(context.Background()); err != nil {
		t.Fatalf("stop collector first run: %v", err)
	}
	if err := collector.Start(context.Background()); err != nil {
		t.Fatalf("start collector second run: %v", err)
	}
	if err := collector.Stop(context.Background()); err != nil {
		t.Fatalf("stop collector second run: %v", err)
	}
	if err := collector.Stop(context.Background()); err != nil {
		t.Fatalf("stop collector should be idempotent: %v", err)
	}
	if calls.Load() < 2 {
		t.Fatalf("expected collector to run twice, got %d", calls.Load())
	}
}
