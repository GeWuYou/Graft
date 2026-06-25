package container

import (
	"context"
	"encoding/json"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	containergen "graft/server/internal/contract/openapi/generated"
	containercontract "graft/server/modules/container/contract"
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

func TestContainerDashboardSummaryPublishedUsesRealtimeSummaryShape(t *testing.T) {
	t.Parallel()

	payload := containerDashboardSummaryPublished{
		Topic:       containercontract.ContainerDashboardSummaryTopic,
		CollectedAt: time.Unix(1_700_000_010, 0).UTC(),
		Data: containerDashboardSummaryResponse{
			CollectedAt: "2023-11-14T22:13:20Z",
			Overview: containerDashboardOverviewResponse{
				RunningContainers:  1,
				AbnormalContainers: 1,
				CPUTotalPercent:    12.5,
			},
			Anomalies: []containerDashboardAnomalyItemResponse{
				{
					ID:          "container-1",
					Name:        "graft-web",
					ShortID:     "container-1",
					Image:       "graft/web:latest",
					State:       "restarting",
					Status:      stringPtr("Restarting"),
					ReasonCode:  stringPtr("state.restarting"),
					ReasonLabel: stringPtr("Restarting"),
				},
			},
		},
	}

	encoded, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal dashboard summary payload: %v", err)
	}

	text := string(encoded)
	if !strings.Contains(text, "\"topic\":\"container.dashboard.summary\"") {
		t.Fatalf("expected dashboard summary topic in payload, got %s", text)
	}
	if !strings.Contains(text, "\"collected_at\":\"2023-11-14T22:13:20Z\"") {
		t.Fatalf("expected summary collected_at field in payload data, got %s", text)
	}
	if !strings.Contains(text, "\"reason_code\":\"state.restarting\"") {
		t.Fatalf("expected anomaly reason_code in payload, got %s", text)
	}
}

func stringPtr(value string) *string {
	return &value
}

func timePtr(value time.Time) *time.Time {
	return &value
}
