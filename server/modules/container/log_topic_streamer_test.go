package container

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"go.uber.org/zap"

	"graft/server/internal/realtime"
	containercontract "graft/server/modules/container/contract"
	"graft/server/modules/container/terminal"
)

func TestLogTopicStreamerPublishesIncrementalLogEventsForActiveTopic(t *testing.T) {
	t.Parallel()

	hub := realtime.NewHub()
	streamRuntime := &streamingRuntime{
		detail: Detail{Summary: Summary{ID: "canonical-web", Name: "web", Runtime: runtimeNameDocker}},
		stream: []LogChunk{
			{Line: "line-1", Stream: "stdout", Timestamp: time.Date(2026, 6, 26, 10, 0, 0, 0, time.UTC)},
			{Line: "line-2", Stream: "stderr", Timestamp: time.Date(2026, 6, 26, 10, 0, 1, 0, time.UTC)},
		},
	}
	streamer, err := newLogTopicStreamer(hub, zap.NewNop(), func() (Runtime, error) {
		return streamRuntime, nil
	})
	if err != nil {
		t.Fatalf("new log topic streamer: %v", err)
	}

	topic := containercontract.ContainerLogsTopicPrefix + "web"
	if err := streamer.EnsureTopic(context.Background(), topic, Ref{Value: "web"}, LogQuery{Tail: 200, Stdout: true, Stderr: true}); err != nil {
		t.Fatalf("ensure topic: %v", err)
	}

	events, unsubscribe := hub.Subscribe(topic)
	defer unsubscribe()

	for i := 0; i < 2; i++ {
		select {
		case event := <-events:
			payload, ok := event.Data.(containerLogPublished)
			if !ok {
				t.Fatalf("unexpected payload %T", event.Data)
			}
			if payload.Topic != topic || payload.ID != "canonical-web" {
				t.Fatalf("unexpected payload %#v", payload)
			}
			if payload.Entry.Line == "" || payload.Entry.Stream == "" || payload.Entry.OccurredAt.IsZero() {
				t.Fatalf("expected structured entry payload, got %#v", payload)
			}
		case <-time.After(2 * time.Second):
			t.Fatalf("timed out waiting for streamed log event %d", i+1)
		}
	}

	if err := streamer.Close(context.Background()); err != nil {
		t.Fatalf("close streamer: %v", err)
	}
}

func TestContainerLogPublishedUsesStableLogEntryJSONShape(t *testing.T) {
	t.Parallel()

	payload := containerLogPublished{
		Topic: "container.logs:web",
		ID:    "canonical-web",
		Entry: LogEntry{
			Line:       "line-1",
			Stream:     "stdout",
			OccurredAt: time.Date(2026, 6, 26, 10, 0, 0, 0, time.UTC),
		},
	}

	encoded, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}

	entry, ok := decoded["entry"].(map[string]any)
	if !ok {
		t.Fatalf("expected entry object, got %#v", decoded["entry"])
	}
	if _, exists := entry["Line"]; exists {
		t.Fatalf("expected stable json field names, got %#v", entry)
	}
	if entry["line"] != "line-1" || entry["stream"] != "stdout" {
		t.Fatalf("expected stable line/stream fields, got %#v", entry)
	}
	if _, exists := entry["occurred_at"]; !exists {
		t.Fatalf("expected occurred_at field, got %#v", entry)
	}
}

type streamingRuntime struct {
	detail    Detail
	detailErr error
	stream    []LogChunk
}

func (r *streamingRuntime) Info(context.Context) (RuntimeInfo, error) { return RuntimeInfo{}, nil }
func (r *streamingRuntime) List(context.Context, ListQuery) ([]Summary, error) {
	return nil, nil
}
func (r *streamingRuntime) Detail(context.Context, Ref) (Detail, error) {
	return r.detail, r.detailErr
}
func (r *streamingRuntime) Mounts(context.Context, Ref) ([]Mount, error) { return nil, nil }
func (r *streamingRuntime) MountUsage(context.Context, Ref, string) (MountUsage, error) {
	return MountUsage{}, nil
}
func (r *streamingRuntime) Logs(context.Context, Ref, LogQuery) (Logs, error) { return Logs{}, nil }
func (r *streamingRuntime) StreamLogs(_ context.Context, _ Ref, _ LogQuery, emit func(LogChunk) error) error {
	for _, chunk := range r.stream {
		if err := emit(chunk); err != nil {
			return err
		}
	}
	return nil
}
func (r *streamingRuntime) Shell(context.Context, Ref, string) (terminal.Session, error) {
	return nil, nil
}
func (r *streamingRuntime) Start(context.Context, Ref) (ActionResult, error) {
	return ActionResult{}, nil
}
func (r *streamingRuntime) Stop(context.Context, Ref) (ActionResult, error) {
	return ActionResult{}, nil
}
func (r *streamingRuntime) Restart(context.Context, Ref) (ActionResult, error) {
	return ActionResult{}, nil
}
func (r *streamingRuntime) Remove(context.Context, Ref, RemoveOptions) (ActionResult, error) {
	return ActionResult{}, nil
}
func (r *streamingRuntime) Close() error { return nil }

var _ Runtime = (*streamingRuntime)(nil)

func TestLogTopicStreamerFallsBackToRefValueWhenDetailFails(t *testing.T) {
	t.Parallel()

	assertLogTopicStreamerFallbackCanonicalID(
		t,
		Detail{},
		errors.New("detail unavailable"),
	)
}

func TestLogTopicStreamerFallsBackToRefValueWhenDetailIDIsEmpty(t *testing.T) {
	t.Parallel()

	assertLogTopicStreamerFallbackCanonicalID(
		t,
		Detail{Summary: Summary{Name: "web", Runtime: runtimeNameDocker}},
		nil,
	)
}

func assertLogTopicStreamerFallbackCanonicalID(t *testing.T, detail Detail, detailErr error) {
	t.Helper()

	hub := realtime.NewHub()
	streamRuntime := &streamingRuntime{
		detail:    detail,
		detailErr: detailErr,
		stream: []LogChunk{
			{Line: "line-1", Stream: "stdout", Timestamp: time.Date(2026, 6, 26, 10, 0, 0, 0, time.UTC)},
		},
	}
	streamer, err := newLogTopicStreamer(hub, zap.NewNop(), func() (Runtime, error) {
		return streamRuntime, nil
	})
	if err != nil {
		t.Fatalf("new log topic streamer: %v", err)
	}

	topic := containercontract.ContainerLogsTopicPrefix + "web"
	ref := Ref{Value: "web"}
	if err := streamer.EnsureTopic(context.Background(), topic, ref, LogQuery{Tail: 200, Stdout: true, Stderr: true}); err != nil {
		t.Fatalf("ensure topic: %v", err)
	}

	events, unsubscribe := hub.Subscribe(topic)
	defer unsubscribe()

	select {
	case event := <-events:
		payload, ok := event.Data.(containerLogPublished)
		if !ok {
			t.Fatalf("unexpected payload %T", event.Data)
		}
		if payload.ID != ref.Value {
			t.Fatalf("expected fallback canonical id %q, got %#v", ref.Value, payload)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for streamed log event")
	}

	if err := streamer.Close(context.Background()); err != nil {
		t.Fatalf("close streamer: %v", err)
	}
}

func TestNewLogTopicStreamerRequiresObservableHub(t *testing.T) {
	t.Parallel()

	_, err := newLogTopicStreamer(nonObservableHub{}, zap.NewNop(), func() (Runtime, error) {
		return nil, errors.New("unreachable")
	})
	if err == nil {
		t.Fatal("expected observable hub requirement")
	}
}

type nonObservableHub struct{}

func (nonObservableHub) Publish(string, any) {}

func (nonObservableHub) Subscribe(string) (<-chan realtime.Event, func()) {
	ch := make(chan realtime.Event)
	close(ch)
	return ch, func() {}
}
