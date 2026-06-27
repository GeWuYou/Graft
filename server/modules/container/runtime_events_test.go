package container

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	containercontract "graft/server/modules/container/contract"
	"graft/server/modules/container/terminal"
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

func TestRuntimeEventManagerDedupesEquivalentEventsWithoutAdvancingSeq(t *testing.T) {
	t.Parallel()

	manager := newRuntimeEventManager(nil, nil, nil, RuntimeEventStreamContext{Runtime: runtimeNameDocker})
	manager.historyLimit = 4
	manager.historyTTL = 0

	candidate := RuntimeEventCandidate{
		ResourceID: "container-dup",
		EventType:  containercontract.RuntimeEventTypeContainerStarted,
		OccurredAt: time.Date(2026, time.June, 27, 6, 0, 0, 0, time.UTC),
		Attributes: map[string]string{"name": "web"},
	}
	if err := manager.Append(candidate); err != nil {
		t.Fatalf("append first candidate: %v", err)
	}
	if err := manager.Append(candidate); err != nil {
		t.Fatalf("append duplicate candidate: %v", err)
	}

	history := manager.History("container-dup")
	if len(history.Items) != 1 {
		t.Fatalf("expected duplicate event to be suppressed, got %d items", len(history.Items))
	}
	if history.Items[0].Seq != 1 {
		t.Fatalf("expected seq to remain 1 after duplicate append, got %d", history.Items[0].Seq)
	}
}

func TestRuntimeEventManagerHistoryEvictsExpiredResourcesOnRead(t *testing.T) {
	t.Parallel()

	manager := newRuntimeEventManager(nil, nil, nil, RuntimeEventStreamContext{Runtime: runtimeNameDocker})
	manager.historyLimit = 4
	manager.historyTTL = 5 * time.Millisecond

	if err := manager.Append(RuntimeEventCandidate{
		ResourceID: "container-ttl",
		EventType:  containercontract.RuntimeEventTypeContainerStarted,
		OccurredAt: time.Date(2026, time.June, 27, 6, 1, 0, 0, time.UTC),
	}); err != nil {
		t.Fatalf("append candidate: %v", err)
	}

	time.Sleep(20 * time.Millisecond)
	history := manager.History("container-ttl")
	if len(history.Items) != 0 {
		t.Fatalf("expected expired history to be evicted on read, got %d items", len(history.Items))
	}
	if history.Context.Runtime != runtimeNameDocker {
		t.Fatalf("expected fallback stream context runtime %q, got %q", runtimeNameDocker, history.Context.Runtime)
	}
}

func TestRuntimeEventManagerStartLoadsRegisteredSourcesOnceAndTracksDiagnostics(t *testing.T) {
	t.Parallel()

	manager, loadCalls, streamCalls := newRuntimeEventManagerWithFailingSource()

	startAndStopRuntimeEventManager(t, manager)
	assertRuntimeEventSourceExecutionCounts(t, loadCalls, streamCalls)
	assertRuntimeEventSourceHistory(t, manager)
	assertRuntimeEventSourceDiagnostics(t, manager.Diagnostics())
}

func newRuntimeEventManagerWithFailingSource() (*runtimeEventManager, *atomic.Int64, *atomic.Int64) {
	loadCalls := &atomic.Int64{}
	streamCalls := &atomic.Int64{}
	source := runtimeEventSourceStub{
		stream: func(_ context.Context, emit func(RuntimeEventCandidate) error) error {
			streamCalls.Add(1)
			if err := emit(startedRuntimeEventCandidate("container-source")); err != nil {
				return err
			}
			if err := emit(startedRuntimeEventCandidate("container-source")); err != nil {
				return err
			}
			return errors.New("source boom")
		},
	}
	manager := newRuntimeEventManager(nil, nil, []runtimeEventSourceRegistration{
		{
			name:          "docker-primary",
			streamContext: RuntimeEventStreamContext{Runtime: runtimeNameDocker},
			load: func() (RuntimeEventSource, error) {
				loadCalls.Add(1)
				return source, nil
			},
		},
	}, RuntimeEventStreamContext{Runtime: runtimeNameDocker})
	manager.historyTTL = 0
	return manager, loadCalls, streamCalls
}

func startedRuntimeEventCandidate(resourceID string) RuntimeEventCandidate {
	return RuntimeEventCandidate{
		ResourceID: resourceID,
		EventType:  containercontract.RuntimeEventTypeContainerStarted,
		OccurredAt: time.Date(2026, time.June, 27, 6, 2, 0, 0, time.UTC),
	}
}

func startAndStopRuntimeEventManager(t *testing.T, manager *runtimeEventManager) {
	t.Helper()

	if err := manager.Start(context.Background()); err != nil {
		t.Fatalf("start manager: %v", err)
	}
	if err := manager.Start(context.Background()); err != nil {
		t.Fatalf("start manager second time: %v", err)
	}
	if err := manager.Stop(context.Background()); err != nil {
		t.Fatalf("stop manager: %v", err)
	}
}

func assertRuntimeEventSourceExecutionCounts(t *testing.T, loadCalls *atomic.Int64, streamCalls *atomic.Int64) {
	t.Helper()

	if loadCalls.Load() != 1 {
		t.Fatalf("expected one source load call, got %d", loadCalls.Load())
	}
	if streamCalls.Load() != 1 {
		t.Fatalf("expected one source stream call, got %d", streamCalls.Load())
	}
}

func assertRuntimeEventSourceHistory(t *testing.T, manager *runtimeEventManager) {
	t.Helper()

	history := manager.History("container-source")
	if len(history.Items) != 1 {
		t.Fatalf("expected duplicate source event to be suppressed, got %d items", len(history.Items))
	}
	if history.Context.Runtime != runtimeNameDocker {
		t.Fatalf("expected source stream context runtime %q, got %q", runtimeNameDocker, history.Context.Runtime)
	}
}

func assertRuntimeEventSourceDiagnostics(t *testing.T, diagnostics runtimeEventManagerDiagnostics) {
	t.Helper()

	sourceDiagnostics, ok := diagnostics.sources["docker-primary"]
	if !ok {
		t.Fatalf("expected diagnostics for docker-primary, got %#v", diagnostics.sources)
	}
	if sourceDiagnostics.streamStarts != 1 {
		t.Fatalf("expected one stream start, got %d", sourceDiagnostics.streamStarts)
	}
	if sourceDiagnostics.streamErrors != 1 {
		t.Fatalf("expected one stream error, got %d", sourceDiagnostics.streamErrors)
	}
	if sourceDiagnostics.duplicateDrops != 1 {
		t.Fatalf("expected one duplicate drop, got %d", sourceDiagnostics.duplicateDrops)
	}
	if sourceDiagnostics.lastError != "source boom" {
		t.Fatalf("expected last error to be recorded, got %q", sourceDiagnostics.lastError)
	}
	if sourceDiagnostics.lastEventAt.IsZero() {
		t.Fatalf("expected last event timestamp to be recorded")
	}
}

type runtimeEventSourceStub struct {
	stream func(context.Context, func(RuntimeEventCandidate) error) error
}

func (runtimeEventSourceStub) Info(context.Context) (RuntimeInfo, error)          { return RuntimeInfo{}, nil }
func (runtimeEventSourceStub) List(context.Context, ListQuery) ([]Summary, error) { return nil, nil }
func (runtimeEventSourceStub) Detail(context.Context, Ref) (Detail, error)        { return Detail{}, nil }
func (runtimeEventSourceStub) Mounts(context.Context, Ref) ([]Mount, error)       { return nil, nil }
func (runtimeEventSourceStub) MountUsage(context.Context, Ref, string) (MountUsage, error) {
	return MountUsage{}, nil
}
func (runtimeEventSourceStub) Logs(context.Context, Ref, LogQuery) (Logs, error) { return Logs{}, nil }
func (runtimeEventSourceStub) StreamLogs(context.Context, Ref, LogQuery, func(LogChunk) error) error {
	return nil
}
func (s runtimeEventSourceStub) StreamRuntimeEvents(
	ctx context.Context,
	emit func(RuntimeEventCandidate) error,
) error {
	if s.stream == nil {
		return nil
	}
	return s.stream(ctx, emit)
}
func (runtimeEventSourceStub) Shell(context.Context, Ref, string) (terminal.Session, error) {
	return nil, nil
}
func (runtimeEventSourceStub) Start(context.Context, Ref) (ActionResult, error) {
	return ActionResult{}, nil
}
func (runtimeEventSourceStub) Stop(context.Context, Ref) (ActionResult, error) {
	return ActionResult{}, nil
}
func (runtimeEventSourceStub) Restart(context.Context, Ref) (ActionResult, error) {
	return ActionResult{}, nil
}
func (runtimeEventSourceStub) Remove(context.Context, Ref, RemoveOptions) (ActionResult, error) {
	return ActionResult{}, nil
}
func (runtimeEventSourceStub) Close() error { return nil }
