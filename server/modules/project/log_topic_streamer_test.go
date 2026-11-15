package project

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"

	"graft/server/internal/realtime"
	projectstore "graft/server/modules/project/store"
)

func TestProjectLogTopicStreamerSanitizesDiagnosticsWithUnwrappedLogger(t *testing.T) {
	t.Parallel()

	core, recorded := observer.New(zapcore.WarnLevel)
	logged := make(chan struct{}, 1)
	output := zap.New(core, zap.Hooks(func(zapcore.Entry) error {
		logged <- struct{}{}
		return nil
	}))
	service, err := NewService(&stubProjectRepository{aggregate: projectstore.ApplicationAggregate{
		Application: projectstore.Application{ApplicationRecordID: 1},
	}})
	if err != nil {
		t.Fatal(err)
	}
	service.SetLifecycleContext(context.Background())
	service.logReader = stubLogReader{err: errors.New("runtime read\r\nfailed\treason")}
	streamer, err := newProjectLogTopicStreamer(realtime.NewHub(), output, service)
	if err != nil {
		t.Fatal(err)
	}
	topic := "application.logs:entry\r\nsegment"
	if err := streamer.EnsureTopic(topic, 1, LogQuery{}); err != nil {
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
	if len(entries) != 1 || entries[0].Message != "project log stream stopped with error" {
		t.Fatalf("unexpected diagnostic entries: %#v", entries)
	}
	fields := entries[0].ContextMap()
	if fields["topic"] != "application.logs:entry segment" || fields["error"] != "runtime read failed reason" {
		t.Fatalf("diagnostic fields must be single-line: %#v", fields)
	}
}
