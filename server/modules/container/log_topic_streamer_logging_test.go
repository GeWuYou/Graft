package container

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"

	"graft/server/internal/realtime"
)

func TestLogTopicStreamerSanitizesDiagnosticsWithUnwrappedLogger(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		name        string
		loaderFails bool
		message     string
	}{
		{name: "runtime loading", loaderFails: true, message: "start container log stream failed"},
		{name: "log streaming", message: "container log stream stopped with error"},
	} {
		t.Run(test.name, func(t *testing.T) {
			assertLogTopicStreamerDiagnosticSanitized(t, test.loaderFails, test.message)
		})
	}
}

func assertLogTopicStreamerDiagnosticSanitized(t *testing.T, loaderFails bool, message string) {
	t.Helper()

	core, recorded := observer.New(zapcore.WarnLevel)
	logged := make(chan struct{}, 1)
	output := zap.New(core, zap.Hooks(func(zapcore.Entry) error {
		logged <- struct{}{}
		return nil
	}))
	failure := errors.New("runtime read\r\nfailed\treason")
	runtime := diagnosticFailureRuntime{streamingRuntime: &streamingRuntime{}, failure: failure}
	loads := 0
	streamer, err := newLogTopicStreamer(realtime.NewHub(), output, func() (Runtime, error) {
		loads++
		if loaderFails && loads > 1 {
			return nil, failure
		}
		return runtime, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	topic := "container.logs:entry\r\nsegment"
	if err := streamer.EnsureTopic(context.Background(), topic, Ref{Value: "container"}, LogQuery{}); err != nil {
		t.Fatal(err)
	}
	streamer.start(topic)
	select {
	case <-logged:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for diagnostic")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := streamer.Close(ctx); err != nil {
		t.Fatal(err)
	}
	entries := recorded.All()
	if len(entries) != 1 || entries[0].Message != message {
		t.Fatalf("unexpected diagnostic entries: %#v", entries)
	}
	fields := entries[0].ContextMap()
	if fields["topic"] != "container.logs:entry segment" || fields["error"] != "runtime read failed reason" {
		t.Fatalf("diagnostic fields must be single-line: %#v", fields)
	}
}

type diagnosticFailureRuntime struct {
	*streamingRuntime
	failure error
}

func (r diagnosticFailureRuntime) StreamLogs(context.Context, Ref, LogQuery, func(LogChunk) error) error {
	return r.failure
}
