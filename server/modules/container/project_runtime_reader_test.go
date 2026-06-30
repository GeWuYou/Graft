package container

import (
	"context"
	"testing"

	"graft/server/modules/container/terminal"
)

type stubProjectReaderRuntime struct {
	items []Summary
}

func (s stubProjectReaderRuntime) Info(context.Context) (RuntimeInfo, error) { return RuntimeInfo{}, nil }
func (s stubProjectReaderRuntime) List(context.Context, ListQuery) ([]Summary, error) {
	return s.items, nil
}
func (s stubProjectReaderRuntime) Detail(context.Context, Ref) (Detail, error) { return Detail{}, nil }
func (s stubProjectReaderRuntime) Mounts(context.Context, Ref) ([]Mount, error) { return nil, nil }
func (s stubProjectReaderRuntime) MountUsage(context.Context, Ref, string) (MountUsage, error) {
	return MountUsage{}, nil
}
func (s stubProjectReaderRuntime) Logs(context.Context, Ref, LogQuery) (Logs, error) { return Logs{}, nil }
func (s stubProjectReaderRuntime) StreamLogs(context.Context, Ref, LogQuery, func(LogChunk) error) error {
	return nil
}
func (s stubProjectReaderRuntime) Shell(context.Context, Ref, string) (terminal.Session, error) { return nil, nil }
func (s stubProjectReaderRuntime) Start(context.Context, Ref) (ActionResult, error) { return ActionResult{}, nil }
func (s stubProjectReaderRuntime) Stop(context.Context, Ref) (ActionResult, error) { return ActionResult{}, nil }
func (s stubProjectReaderRuntime) Restart(context.Context, Ref) (ActionResult, error) { return ActionResult{}, nil }
func (s stubProjectReaderRuntime) Remove(context.Context, Ref, RemoveOptions) (ActionResult, error) {
	return ActionResult{}, nil
}
func (s stubProjectReaderRuntime) Close() error { return nil }

func TestContainerProjectRuntimeReaderMapsComposeMembers(t *testing.T) {
	t.Parallel()

	reader := containerProjectRuntimeReader{
		service: &service{
			runtime: stubProjectReaderRuntime{
				items: []Summary{
					{ID: "1", Name: "demo-web-1", State: "running", ComposeProject: "demo", ComposeService: "web"},
					{ID: "2", Name: "demo-worker-1", State: "exited", ComposeProject: "demo", ComposeService: "worker"},
					{ID: "3", Name: "other-api-1", State: "running", ComposeProject: "other", ComposeService: "api"},
				},
			},
			enabled: true,
		},
	}

	summary, err := reader.ListProjectMembers(context.Background(), "local", "demo")
	if err != nil {
		t.Fatalf("list project members: %v", err)
	}
	if summary.RunningCount != 1 || summary.StoppedCount != 1 {
		t.Fatalf("unexpected counts: %#v", summary)
	}
	if len(summary.Members) != 2 {
		t.Fatalf("expected 2 members, got %d", len(summary.Members))
	}
	if summary.Members[0].ServiceName != "web" && summary.Members[1].ServiceName != "web" {
		t.Fatalf("expected web member in %#v", summary.Members)
	}
}
