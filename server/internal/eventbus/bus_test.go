package eventbus

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

// TestSubscribeRejectsInvalidInput 验证事件名和处理器都必须显式提供，
// 避免生命周期注册阶段留下无法诊断的空订阅。
func TestSubscribeRejectsInvalidInput(t *testing.T) {
	bus := New(zap.NewNop())

	if err := bus.Subscribe("", func(context.Context, Event) error { return nil }); err == nil {
		t.Fatal("expected empty event name to fail")
	}
	if err := bus.Subscribe("user.created", nil); err == nil {
		t.Fatal("expected nil handler to fail")
	}
}

// TestPublishDeliversEventToSubscribers 验证事件会按订阅顺序同步派发，
// 并在缺失 OccurredAt 时补齐发布时间戳。
func TestPublishDeliversEventToSubscribers(t *testing.T) {
	bus := New(zap.NewNop())
	order := make([]string, 0, 2)
	received := make([]Event, 0, 2)

	if err := bus.Subscribe("user.created", func(_ context.Context, event Event) error {
		order = append(order, "first")
		received = append(received, event)
		return nil
	}); err != nil {
		t.Fatalf("subscribe first handler: %v", err)
	}
	if err := bus.Subscribe("user.created", func(_ context.Context, event Event) error {
		order = append(order, "second")
		received = append(received, event)
		return nil
	}); err != nil {
		t.Fatalf("subscribe second handler: %v", err)
	}

	before := time.Now().UTC()
	err := bus.Publish(context.Background(), Event{
		Name:    "user.created",
		Source:  "user",
		Payload: "payload",
	})
	after := time.Now().UTC()
	if err != nil {
		t.Fatalf("publish event: %v", err)
	}

	assertEventOrder(t, order, "first", "second")
	assertReceivedPayloadAndTimestamp(t, received, "payload", before, after)
}

// TestPublishAggregatesHandlerFailures 验证单个处理器失败或 panic 时，
// 总线仍会继续调用其余处理器并返回聚合错误。
func TestPublishAggregatesHandlerFailures(t *testing.T) {
	core, logs := observer.New(zap.ErrorLevel)
	bus := New(zap.New(core))

	expectedErr := errors.New("write audit event failed")
	order := make([]string, 0, 3)

	if err := bus.Subscribe("audit.record", func(_ context.Context, _ Event) error {
		order = append(order, "first")
		return expectedErr
	}); err != nil {
		t.Fatalf("subscribe first handler: %v", err)
	}
	if err := bus.Subscribe("audit.record", func(_ context.Context, _ Event) error {
		order = append(order, "second")
		panic("boom")
	}); err != nil {
		t.Fatalf("subscribe second handler: %v", err)
	}
	if err := bus.Subscribe("audit.record", func(_ context.Context, _ Event) error {
		order = append(order, "third")
		return nil
	}); err != nil {
		t.Fatalf("subscribe third handler: %v", err)
	}

	err := bus.Publish(context.Background(), Event{Name: "audit.record", Source: "audit"})
	if err == nil {
		t.Fatal("expected publish to report aggregated handler failures")
	}
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected aggregated error to include handler failure, got %v", err)
	}
	if !strings.Contains(err.Error(), "panic") {
		t.Fatalf("expected aggregated error to include panic recovery details, got %v", err)
	}
	assertEventOrder(t, order, "first", "second", "third")
	if logs.Len() != 2 {
		t.Fatalf("expected two error logs, got %d", logs.Len())
	}
}

func assertEventOrder(t *testing.T, got []string, want ...string) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("expected handler order %v, got %v", want, got)
	}

	for i, name := range want {
		if got[i] != name {
			t.Fatalf("expected handler order %v, got %v", want, got)
		}
	}
}

func assertReceivedPayloadAndTimestamp(t *testing.T, events []Event, payload string, before time.Time, after time.Time) {
	t.Helper()

	for _, event := range events {
		if event.Payload != payload {
			t.Fatalf("expected payload to be preserved, got %#v", event.Payload)
		}
		if event.OccurredAt.Before(before) || event.OccurredAt.After(after) {
			t.Fatalf("expected occurredAt to be stamped during publish, got %s", event.OccurredAt)
		}
	}
}
