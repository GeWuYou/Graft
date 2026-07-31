package container

import (
	"context"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"

	containercontract "graft/server/modules/container/contract"
)

type dockerVolumeBatchTestRuntime struct {
	fakeRuntime
	errors map[string]error
}

func (r *dockerVolumeBatchTestRuntime) ListDockerImages(context.Context) (DockerImageListResult, error) {
	return DockerImageListResult{}, nil
}

func (r *dockerVolumeBatchTestRuntime) ReadDockerImage(context.Context, string) (DockerImage, error) {
	return DockerImage{}, nil
}

func (r *dockerVolumeBatchTestRuntime) ListDockerNetworks(context.Context) ([]DockerNetwork, error) {
	return nil, nil
}

func (r *dockerVolumeBatchTestRuntime) ReadDockerNetwork(context.Context, string) (DockerNetwork, error) {
	return DockerNetwork{}, nil
}

func (r *dockerVolumeBatchTestRuntime) ListDockerVolumes(context.Context) ([]DockerVolume, error) {
	return nil, nil
}

func (r *dockerVolumeBatchTestRuntime) ReadDockerVolume(context.Context, string) (DockerVolume, error) {
	return DockerVolume{}, nil
}

func (r *dockerVolumeBatchTestRuntime) RemoveDockerVolume(_ context.Context, name string, _ bool) error {
	return r.errors[name]
}

func TestDockerVolumeBatchRemoveUsesStableErrorCodeAndFallbackMessage(t *testing.T) {
	t.Parallel()

	core, observed := observer.New(zapcore.ErrorLevel)
	runtime := &dockerVolumeBatchTestRuntime{errors: map[string]error{
		"missing": errInvalidContainerState,
	}}
	service, err := newTestService(containerServiceOptions{
		logger:                  zap.New(core),
		runtime:                 newRuntimeLease(runtime),
		enabled:                 true,
		dangerousActionsEnabled: true,
	})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	result, err := service.DockerVolumeBatchRemove(context.Background(), []string{"ok", "missing"}, false)
	if err != nil {
		t.Fatalf("batch remove: %v", err)
	}
	if len(result.Items) != 2 || !result.Items[0].Success || result.Items[1].Success {
		t.Fatalf("unexpected batch result: %#v", result)
	}
	item := result.Items[1]
	wantKey := containercontract.ContainerInvalidState.String()
	if item.ErrorCode != wantKey || item.MessageKey != wantKey || item.Message != fallbackMessageForError(errInvalidContainerState) {
		t.Fatalf("unexpected failure projection: %#v", item)
	}
	entries := observed.All()
	if len(entries) != 1 || entries[0].Message != "docker volume batch removal failed" {
		t.Fatalf("expected one volume removal failure log, got %#v", entries)
	}
	fields := entries[0].ContextMap()
	if fields["volume_name"] != "missing" || fields["force"] != false || fields["error"] != errInvalidContainerState.Error() {
		t.Fatalf("unexpected volume removal failure fields: %#v", fields)
	}
}
