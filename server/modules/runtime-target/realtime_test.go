package runtimetarget

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	generated "graft/server/internal/contract/openapi/generated"
	"graft/server/internal/realtime"
	contract "graft/server/modules/runtime-target/contract"
)

type runtimeTargetRealtimeHubStub struct {
	published  chan realtime.Event
	onActive   func(string)
	onInactive func(string)
}

func (h *runtimeTargetRealtimeHubStub) Publish(topic string, payload any) {
	h.published <- realtime.Event{Topic: topic, Data: payload}
}

func (h *runtimeTargetRealtimeHubStub) Subscribe(string) (<-chan realtime.Event, func()) {
	stream := make(chan realtime.Event)
	return stream, func() { close(stream) }
}

func (h *runtimeTargetRealtimeHubStub) RegisterTopicObserver(_ string, onActive func(string), onInactive func(string)) (func(), error) {
	h.onActive = onActive
	h.onInactive = onInactive
	return func() { h.onActive, h.onInactive = nil, nil }, nil
}

func TestRuntimeTargetSummaryCollectorPublishesOnlyAfterTopicActivation(t *testing.T) {
	hub := &runtimeTargetRealtimeHubStub{published: make(chan realtime.Event, 1)}
	var calls atomic.Int32
	collector := newRuntimeTargetSummaryCollector(hub, func(context.Context) []generated.RuntimeTargetSummary {
		calls.Add(1)
		return []generated.RuntimeTargetSummary{{Id: 7}}
	})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := collector.Start(ctx); err != nil {
		t.Fatalf("start collector: %v", err)
	}
	time.Sleep(50 * time.Millisecond)
	if calls.Load() != 0 {
		t.Fatalf("collector ran without a subscriber: %d", calls.Load())
	}
	hub.onActive(contract.SummaryTopic)
	select {
	case event := <-hub.published:
		if event.Topic != contract.SummaryTopic {
			t.Fatalf("published topic = %q", event.Topic)
		}
		payload, ok := event.Data.(runtimeTargetSummaryPublished)
		if !ok || len(payload.Items) != 1 || payload.Items[0].Id != 7 {
			t.Fatalf("unexpected payload: %#v", event.Data)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("collector did not publish an active snapshot")
	}
	if err := collector.Stop(context.Background()); err != nil {
		t.Fatalf("stop collector: %v", err)
	}
}

func TestCollectHostTargetUsageReportsIndependentMetrics(t *testing.T) {
	cpuUsage, memoryUsage := collectHostUsage(
		context.Background(),
		func(context.Context) targetUsageMetric {
			return targetUsageMetric{UnavailableReason: "CPU probe failed"}
		},
		func(context.Context) targetUsageMetric {
			return targetUsageMetric{Available: true, UsedBytes: 3, TotalBytes: 8, UsagePercent: 37.5}
		},
	)
	if cpuUsage.Available || cpuUsage.UnavailableReason != "CPU probe failed" {
		t.Fatalf("CPU metric = %#v", cpuUsage)
	}
	if !memoryUsage.Available || memoryUsage.UsedBytes != 3 || memoryUsage.TotalBytes != 8 || memoryUsage.UsagePercent != 37.5 {
		t.Fatalf("memory metric = %#v", memoryUsage)
	}
}
