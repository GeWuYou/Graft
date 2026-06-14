// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package container

import (
	"context"
	"errors"
	"testing"

	"go.uber.org/zap"

	"graft/server/internal/eventbus"
	"graft/server/internal/httpx"
	"graft/server/internal/moduleapi"
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
