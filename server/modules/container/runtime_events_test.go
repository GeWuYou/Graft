package container

import (
	"testing"
	"time"

	containercontract "graft/server/modules/container/contract"
)

func TestNewRuntimeEventAppliesCanonicalSeverity(t *testing.T) {
	t.Parallel()

	occurredAt := time.Date(2026, time.June, 27, 5, 0, 0, 0, time.UTC)
	event, err := newRuntimeEvent(RuntimeEventCandidate{
		ResourceID: "container-1",
		EventType:  containercontract.RuntimeEventTypeContainerOOMKilled,
		OccurredAt: occurredAt,
		Attributes: map[string]string{
			"exit_code": "137",
		},
	})
	if err != nil {
		t.Fatalf("new runtime event: %v", err)
	}

	if event.ResourceType != runtimeEventResourceTypeContainer {
		t.Fatalf("expected resource type %q, got %q", runtimeEventResourceTypeContainer, event.ResourceType)
	}
	if event.ResourceID != "container-1" {
		t.Fatalf("expected resource id container-1, got %q", event.ResourceID)
	}
	if event.EventType != containercontract.RuntimeEventTypeContainerOOMKilled.String() {
		t.Fatalf("expected event type %q, got %q", containercontract.RuntimeEventTypeContainerOOMKilled, event.EventType)
	}
	if event.Severity != containercontract.RuntimeEventSeverityError.String() {
		t.Fatalf("expected canonical severity %q, got %q", containercontract.RuntimeEventSeverityError, event.Severity)
	}
	if event.OccurredAt != occurredAt {
		t.Fatalf("expected occurred_at %s, got %s", occurredAt, event.OccurredAt)
	}
	if event.Attributes["exit_code"] != "137" {
		t.Fatalf("expected exit_code attribute to survive normalization, got %#v", event.Attributes)
	}
	if event.ID == "" {
		t.Fatal("expected opaque runtime event id")
	}
}

func TestRuntimeEventManagerHistoryUsesSeqAndBoundedReplay(t *testing.T) {
	t.Parallel()

	manager := newRuntimeEventManager(nil, nil, nil, RuntimeEventStreamContext{Runtime: runtimeNameDocker})
	manager.historyLimit = 2
	manager.historyTTL = 0

	candidates := []RuntimeEventCandidate{
		{ResourceID: "container-1", EventType: containercontract.RuntimeEventTypeContainerStarted, OccurredAt: time.Date(2026, time.June, 27, 5, 1, 0, 0, time.UTC)},
		{ResourceID: "container-1", EventType: containercontract.RuntimeEventTypeContainerRestarted, OccurredAt: time.Date(2026, time.June, 27, 5, 2, 0, 0, time.UTC)},
		{ResourceID: "container-1", EventType: containercontract.RuntimeEventTypeContainerStopped, OccurredAt: time.Date(2026, time.June, 27, 5, 3, 0, 0, time.UTC)},
	}
	for _, candidate := range candidates {
		if err := manager.Append(candidate); err != nil {
			t.Fatalf("append candidate: %v", err)
		}
	}

	history := manager.History("container-1")
	if history.ResourceID != "container-1" {
		t.Fatalf("expected resource id container-1, got %q", history.ResourceID)
	}
	if history.Context.Runtime != runtimeNameDocker {
		t.Fatalf("expected runtime context %q, got %q", runtimeNameDocker, history.Context.Runtime)
	}
	if len(history.Items) != 2 {
		t.Fatalf("expected bounded history size 2, got %d", len(history.Items))
	}
	if history.Items[0].Seq != 2 || history.Items[1].Seq != 3 {
		t.Fatalf("expected trailing seq values [2 3], got [%d %d]", history.Items[0].Seq, history.Items[1].Seq)
	}
	if history.Items[0].Event.EventType != containercontract.RuntimeEventTypeContainerRestarted.String() {
		t.Fatalf("expected second event to be restart, got %q", history.Items[0].Event.EventType)
	}
	if history.Items[1].Event.EventType != containercontract.RuntimeEventTypeContainerStopped.String() {
		t.Fatalf("expected third event to be stop, got %q", history.Items[1].Event.EventType)
	}
}
