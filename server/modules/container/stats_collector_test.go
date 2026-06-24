package container

import (
	"context"
	"encoding/json"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	containergen "graft/server/internal/contract/openapi/generated"
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

func TestContainerStatsPublishedUsesOpenAPIResourceJSONShape(t *testing.T) {
	t.Parallel()

	payload := containerStatsPublished{
		Topic:   "container.stats:container-1",
		ID:      "container-1",
		Name:    "graft-web",
		ShortID: "container-1",
		Runtime: "docker",
		Resource: &containergen.ContainerResourceSummary{
			CpuPercent:       float64Ptr(12.5),
			MemoryPercent:    float64Ptr(25),
			MemoryUsageBytes: int64Ptr(256),
			CollectedAt:      timePtr(time.Unix(1_700_000_000, 0).UTC()),
		},
		CollectedAt: time.Unix(1_700_000_001, 0).UTC(),
	}

	encoded, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal stats payload: %v", err)
	}

	text := string(encoded)
	if !strings.Contains(text, "\"cpu_percent\":12.5") {
		t.Fatalf("expected snake_case cpu_percent in realtime payload, got %s", text)
	}
	if strings.Contains(text, "\"CPUPercent\"") {
		t.Fatalf("expected realtime payload to omit PascalCase CPUPercent, got %s", text)
	}
	if !strings.Contains(text, "\"memory_usage_bytes\":256") {
		t.Fatalf("expected snake_case memory_usage_bytes in realtime payload, got %s", text)
	}
}

func timePtr(value time.Time) *time.Time {
	return &value
}
