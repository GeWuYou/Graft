package container

import (
	"context"
	"strings"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"

	containercontract "graft/server/modules/container/contract"
)

type dockerVolumeBatchTestRuntime struct {
	fakeRuntime
	errors       map[string]error
	removeForces []bool
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

func (r *dockerVolumeBatchTestRuntime) RemoveDockerVolume(_ context.Context, name string, force bool) error {
	r.removeForces = append(r.removeForces, force)
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
	assertFailedVolumeBatchResult(t, result, 1)
	entries := observed.All()
	if len(entries) != 1 || entries[0].Message != "docker volume batch removal failed" {
		t.Fatalf("expected one volume removal failure log, got %#v", entries)
	}
	assertVolumeFailureLog(t, entries[0], false)

	if len(runtime.removeForces) != 2 || runtime.removeForces[0] || runtime.removeForces[1] {
		t.Fatalf("unexpected runtime force arguments: %#v", runtime.removeForces)
	}
}

func TestDockerVolumeBatchRemovePassesForceAndLogsIt(t *testing.T) {
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

	result, err := service.DockerVolumeBatchRemove(context.Background(), []string{"missing"}, true)
	if err != nil {
		t.Fatalf("forced batch remove: %v", err)
	}
	assertFailedVolumeBatchResult(t, result, 0)
	entries := observed.All()
	if len(entries) != 1 {
		t.Fatalf("expected one volume removal failure log, got %#v", entries)
	}
	assertVolumeFailureLog(t, entries[0], true)
	if len(runtime.removeForces) != 1 || !runtime.removeForces[0] {
		t.Fatalf("unexpected runtime force arguments: %#v", runtime.removeForces)
	}
}

func assertFailedVolumeBatchResult(t *testing.T, result DockerVolumeBatchRemoveResult, failedIndex int) {
	t.Helper()
	if len(result.Items) != failedIndex+1 || result.Items[failedIndex].Success || (failedIndex == 1 && !result.Items[0].Success) {
		t.Fatalf("unexpected batch result: %#v", result)
	}
	item := result.Items[failedIndex]
	wantKey := containercontract.ContainerInvalidState.String()
	if item.ErrorCode != wantKey || item.MessageKey != wantKey || item.Message != fallbackMessageForError(errInvalidContainerState) {
		t.Fatalf("unexpected failure projection: %#v", item)
	}
}

func assertVolumeFailureLog(t *testing.T, entry observer.LoggedEntry, force bool) {
	t.Helper()
	fields := entry.ContextMap()
	if fields["volume_name"] != "missing" || fields["force"] != force || !strings.Contains(fields["error"].(string), errInvalidContainerState.Error()) {
		t.Fatalf("unexpected volume removal failure fields: %#v", fields)
	}
}
