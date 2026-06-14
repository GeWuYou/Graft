// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package container

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"

	"go.uber.org/zap"

	"graft/server/internal/config"
	"graft/server/internal/configregistry"
	"graft/server/internal/eventbus"
	"graft/server/internal/httpx"
	"graft/server/internal/module"
	"graft/server/internal/moduleapi"
	containercontract "graft/server/modules/container/contract"
)

func TestParseRefRejectsUnsafeValues(t *testing.T) {
	t.Parallel()

	cases := []string{"", "%2Fvar", "name%2Fchild", "bad%00id", "%zz"}
	for _, raw := range cases {
		if _, err := parseRef(raw); !errors.Is(err, errInvalidRef) {
			t.Fatalf("expected invalid ref for %q, got %v", raw, err)
		}
	}
	ref, err := parseRef("web%2D1")
	if err != nil {
		t.Fatalf("parse valid ref: %v", err)
	}
	if ref.Value != "web-1" {
		t.Fatalf("unexpected ref %q", ref.Value)
	}
}

func TestServiceNormalizesLogQuery(t *testing.T) {
	t.Parallel()

	service, err := newService(containerServiceOptions{
		runtime:     fakeRuntime{},
		enabled:     true,
		defaultTail: defaultContainerLogsDefaultTail,
		maxTail:     defaultContainerLogsMaxTail,
	})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	logs, err := service.Logs(context.Background(), Ref{Value: "web"}, LogQuery{})
	if err != nil {
		t.Fatalf("logs: %v", err)
	}
	if logs.Tail != defaultContainerLogsDefaultTail || !logs.Stdout || !logs.Stderr {
		t.Fatalf("unexpected normalized logs: %#v", logs)
	}
	_, err = service.Logs(context.Background(), Ref{Value: "web"}, LogQuery{Tail: defaultContainerLogsMaxTail + 1})
	if !errors.Is(err, errLogsTooLarge) {
		t.Fatalf("expected logs too large, got %v", err)
	}
	_, err = service.Logs(context.Background(), Ref{Value: "web"}, LogQuery{Since: "not-a-time"})
	if !errors.Is(err, errInvalidLogQuery) {
		t.Fatalf("expected invalid log query, got %v", err)
	}
}

func TestDangerousActionsDisabledPublishesFailureAudit(t *testing.T) {
	t.Parallel()

	bus := eventbus.New(zap.NewNop())
	events := make([]moduleapi.AuditEvent, 0, 1)
	if err := bus.Subscribe(string(moduleapi.AuditRecordEventName), func(_ context.Context, event eventbus.Event) error {
		payload, ok := event.Payload.(moduleapi.AuditEvent)
		if !ok {
			t.Fatalf("unexpected payload %T", event.Payload)
		}
		events = append(events, payload)
		return nil
	}); err != nil {
		t.Fatalf("subscribe audit: %v", err)
	}
	service, err := newService(containerServiceOptions{
		runtime:                 fakeRuntime{},
		auditBus:                bus,
		moduleName:              moduleID,
		enabled:                 true,
		dangerousActionsEnabled: false,
		defaultTail:             defaultContainerLogsDefaultTail,
		maxTail:                 defaultContainerLogsMaxTail,
	})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	requestCtx := httpx.WithRequestAuditContext(context.Background(), httpx.RequestAuditContext{
		RequestID: "req-1",
		TraceID:   "trace-1",
		Route:     "/ops/containers/:id/start",
		Method:    "POST",
	})
	_, err = service.Start(requestCtx, Ref{Value: "web"})
	if !errors.Is(err, errDangerousActionsDisabled) {
		t.Fatalf("expected dangerous action guard, got %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected one audit event, got %#v", events)
	}
	event := events[0]
	if event.Action != "ops.container.start" || event.Success {
		t.Fatalf("unexpected audit event %#v", event)
	}
	if event.MessageKey != "ops.container.error.dangerousActionsDisabled" {
		t.Fatalf("unexpected message key %q", event.MessageKey)
	}
	if event.Metadata["requestId"] != "req-1" {
		t.Fatalf("expected request id metadata, got %#v", event.Metadata)
	}
}

func TestRuntimeAccessDisabledUsesResolverAndDoesNotTouchRuntime(t *testing.T) {
	t.Parallel()

	runtime := &countingRuntime{}
	service, err := newService(containerServiceOptions{
		runtime: runtime,
		systemConfig: serviceTestSystemConfig{values: map[string]bool{
			containercontract.ContainerRuntimeEnabledConfig.String(): false,
		}},
		enabled:     true,
		defaultTail: defaultContainerLogsDefaultTail,
		maxTail:     defaultContainerLogsMaxTail,
	})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	_, _, err = service.List(context.Background())
	if !errors.Is(err, errRuntimeDisabled) {
		t.Fatalf("expected runtime disabled, got %v", err)
	}
	if calls := runtime.calls.Load(); calls != 0 {
		t.Fatalf("expected disabled runtime access to avoid runtime calls, got %d", calls)
	}
}

func TestRuntimeAccessEnabledButRuntimeUnavailableUsesConnectionErrorKey(t *testing.T) {
	t.Parallel()

	service, err := newService(containerServiceOptions{
		runtime: failingRuntime{err: errRuntimeDaemonUnavailable},
		systemConfig: serviceTestSystemConfig{values: map[string]bool{
			containercontract.ContainerRuntimeEnabledConfig.String(): true,
		}},
		defaultTail: defaultContainerLogsDefaultTail,
		maxTail:     defaultContainerLogsMaxTail,
	})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	_, _, err = service.List(context.Background())
	if !errors.Is(err, errRuntimeDaemonUnavailable) {
		t.Fatalf("expected runtime daemon unavailable, got %v", err)
	}
	if got := messageKeyForError(err); got != containercontract.ContainerRuntimeUnavailable {
		t.Fatalf("expected runtime unavailable message key, got %s", got)
	}
}

func TestDangerousActionsResolverControlsWriteActionsOnly(t *testing.T) {
	t.Parallel()

	service, err := newService(containerServiceOptions{
		runtime: fakeRuntime{},
		systemConfig: serviceTestSystemConfig{values: map[string]bool{
			containercontract.ContainerRuntimeEnabledConfig.String():          true,
			containercontract.ContainerDangerousActionsEnabledConfig.String(): false,
		}},
		enabled:                 false,
		dangerousActionsEnabled: true,
		defaultTail:             defaultContainerLogsDefaultTail,
		maxTail:                 defaultContainerLogsMaxTail,
	})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	if _, _, err := service.List(context.Background()); err != nil {
		t.Fatalf("expected read path to stay available, got %v", err)
	}
	_, err = service.Start(context.Background(), Ref{Value: "web"})
	if !errors.Is(err, errDangerousActionsDisabled) {
		t.Fatalf("expected dangerous actions guard, got %v", err)
	}
}

func TestContainerOptionsFromConfigUsesRegisteredDefaults(t *testing.T) {
	t.Parallel()

	registry := newContainerConfigRegistry(t)
	options := containerOptionsFromConfig(&module.Context{ConfigRegistry: registry})
	if !options.enabled {
		t.Fatalf("expected runtime access enabled from config defaults")
	}
	if options.runtime != defaultContainerRuntime {
		t.Fatalf("expected runtime %q, got %q", defaultContainerRuntime, options.runtime)
	}
	if options.endpoint != "unix:///tmp/docker.sock" {
		t.Fatalf("expected configured endpoint, got %q", options.endpoint)
	}
	if options.defaultTail != 50 || options.maxTail != 500 {
		t.Fatalf("expected configured tail limits, got default=%d max=%d", options.defaultTail, options.maxTail)
	}
	if !options.dangerousActionsEnabled {
		t.Fatalf("expected dangerous actions default from config")
	}
}

func TestContainerOptionsFromConfigPrefersProcessConfig(t *testing.T) {
	t.Parallel()

	options := containerOptionsFromConfig(&module.Context{
		ConfigRegistry: newContainerConfigRegistry(t),
		Config: &config.Config{
			Container: config.ContainerConfig{
				RuntimeEnabled:          true,
				Runtime:                 runtimeNameDocker,
				DockerEndpoint:          "unix:///process/docker.sock",
				LogsDefaultTail:         25,
				LogsMaxTail:             250,
				DangerousActionsEnabled: true,
			},
		},
	})

	if !options.enabled || !options.dangerousActionsEnabled {
		t.Fatalf("expected process config booleans, got %#v", options)
	}
	if options.runtime != runtimeNameDocker || options.endpoint != "unix:///process/docker.sock" {
		t.Fatalf("expected process runtime config, got %#v", options)
	}
	if options.defaultTail != 25 || options.maxTail != 250 {
		t.Fatalf("expected process tail limits, got %#v", options)
	}
}

func newContainerConfigRegistry(t *testing.T) *configregistry.Registry {
	t.Helper()

	registry := configregistry.NewRegistry()
	for _, definition := range configDefinitions() {
		switch definition.Key {
		case containercontract.ContainerRuntimeEnabledConfig.String():
			definition.DefaultValue = mustRawJSON(true)
		case containercontract.ContainerRuntimeConfig.String():
			definition.DefaultValue = mustRawJSON(defaultContainerRuntime)
		case containercontract.ContainerDockerEndpointConfig.String():
			definition.DefaultValue = mustRawJSON("unix:///tmp/docker.sock")
		case containercontract.ContainerLogsDefaultTailConfig.String():
			definition.DefaultValue = mustRawJSON(50)
		case containercontract.ContainerLogsMaxTailConfig.String():
			definition.DefaultValue = mustRawJSON(500)
		case containercontract.ContainerDangerousActionsEnabledConfig.String():
			definition.DefaultValue = mustRawJSON(true)
		}
		if err := registry.Register(definition); err != nil {
			t.Fatalf("register config definition %s: %v", definition.Key, err)
		}
	}
	return registry
}

func TestRuntimeForRequestInitializesOnceUnderConcurrentAccess(t *testing.T) {
	t.Parallel()

	var factoryCalls atomic.Int64
	service, err := newService(containerServiceOptions{
		runtime: disabledRuntime{},
		runtimeOptions: containerRuntimeOptions{
			runtime:  runtimeNameDocker,
			endpoint: "unix:///tmp/docker.sock",
		},
		runtimeFactory: func(options containerRuntimeOptions) (Runtime, error) {
			factoryCalls.Add(1)
			if !options.enabled {
				return nil, errors.New("expected lazy runtime init to enable runtime")
			}
			if options.endpoint != "unix:///tmp/docker.sock" {
				return nil, errRuntimeDaemonUnavailable
			}
			return fakeRuntime{}, nil
		},
		enabled:     true,
		defaultTail: defaultContainerLogsDefaultTail,
		maxTail:     defaultContainerLogsMaxTail,
	})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	var wg sync.WaitGroup
	errs := make(chan error, 16)
	for range 16 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := service.runtimeForRequest()
			errs <- err
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("runtime for request: %v", err)
		}
	}
	if calls := factoryCalls.Load(); calls != 1 {
		t.Fatalf("expected one runtime factory call, got %d", calls)
	}
}

type serviceTestSystemConfig struct {
	values map[string]bool
}

func (r serviceTestSystemConfig) IsBooleanConfigEnabled(_ context.Context, key string, fallback bool) bool {
	value, ok := r.values[key]
	if !ok {
		return fallback
	}
	return value
}

var _ moduleapi.SystemConfigResolver = serviceTestSystemConfig{}

type countingRuntime struct {
	calls atomic.Int64
}

func (r *countingRuntime) Info(context.Context) (RuntimeInfo, error) {
	r.calls.Add(1)
	return RuntimeInfo{}, nil
}

func (r *countingRuntime) List(context.Context, ListQuery) ([]Summary, error) {
	r.calls.Add(1)
	return nil, nil
}

func (r *countingRuntime) Detail(context.Context, Ref) (Detail, error) {
	r.calls.Add(1)
	return Detail{}, nil
}

func (r *countingRuntime) Logs(context.Context, Ref, LogQuery) (Logs, error) {
	r.calls.Add(1)
	return Logs{}, nil
}

func (r *countingRuntime) Start(context.Context, Ref) (ActionResult, error) {
	r.calls.Add(1)
	return ActionResult{}, nil
}

func (r *countingRuntime) Stop(context.Context, Ref) (ActionResult, error) {
	r.calls.Add(1)
	return ActionResult{}, nil
}

func (r *countingRuntime) Restart(context.Context, Ref) (ActionResult, error) {
	r.calls.Add(1)
	return ActionResult{}, nil
}

func (r *countingRuntime) Close() error { return nil }

type failingRuntime struct {
	err error
}

func (r failingRuntime) Info(context.Context) (RuntimeInfo, error) {
	return RuntimeInfo{}, r.err
}

func (r failingRuntime) List(context.Context, ListQuery) ([]Summary, error) {
	return nil, r.err
}

func (r failingRuntime) Detail(context.Context, Ref) (Detail, error) {
	return Detail{}, r.err
}

func (r failingRuntime) Logs(context.Context, Ref, LogQuery) (Logs, error) {
	return Logs{}, r.err
}

func (r failingRuntime) Start(context.Context, Ref) (ActionResult, error) {
	return ActionResult{}, r.err
}

func (r failingRuntime) Stop(context.Context, Ref) (ActionResult, error) {
	return ActionResult{}, r.err
}

func (r failingRuntime) Restart(context.Context, Ref) (ActionResult, error) {
	return ActionResult{}, r.err
}

func (r failingRuntime) Close() error { return nil }

type fakeRuntime struct{}

func (fakeRuntime) Info(context.Context) (RuntimeInfo, error) {
	return RuntimeInfo{Runtime: runtimeNameDocker, Status: "enabled", Endpoint: "unix:///var/run/docker.sock"}, nil
}

func (fakeRuntime) List(context.Context, ListQuery) ([]Summary, error) {
	return []Summary{fakeSummary()}, nil
}

func (fakeRuntime) Detail(context.Context, Ref) (Detail, error) {
	return Detail{Summary: fakeSummary()}, nil
}

func (fakeRuntime) Logs(_ context.Context, ref Ref, query LogQuery) (Logs, error) {
	return Logs{
		ID:         ref.Value,
		Runtime:    runtimeNameDocker,
		Lines:      []string{"line"},
		Tail:       query.Tail,
		Stdout:     query.Stdout,
		Stderr:     query.Stderr,
		Timestamps: query.Timestamps,
	}, nil
}

func (fakeRuntime) Start(context.Context, Ref) (ActionResult, error) {
	return fakeAction(containerActionStart), nil
}

func (fakeRuntime) Stop(context.Context, Ref) (ActionResult, error) {
	return fakeAction(containerActionStop), nil
}

func (fakeRuntime) Restart(context.Context, Ref) (ActionResult, error) {
	return fakeAction(containerActionRestart), nil
}

func (fakeRuntime) Close() error { return nil }

func fakeSummary() Summary {
	return Summary{
		ID:        "abc123",
		Names:     []string{"web"},
		Image:     "nginx:latest",
		Runtime:   runtimeNameDocker,
		CreatedAt: "2026-06-14T00:00:00Z",
		State:     "running",
		Status:    "Up",
	}
}

func fakeAction(action string) ActionResult {
	return ActionResult{
		ID:           "abc123",
		Name:         "web",
		Image:        "nginx:latest",
		Action:       action,
		Result:       actionResultCompleted,
		Runtime:      runtimeNameDocker,
		StatusBefore: "exited",
		StatusAfter:  "running",
	}
}
