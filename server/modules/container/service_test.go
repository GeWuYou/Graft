// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package container

import (
	"context"
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"go.uber.org/zap"

	"graft/server/internal/config"
	"graft/server/internal/configregistry"
	"graft/server/internal/eventbus"
	"graft/server/internal/httpx"
	"graft/server/internal/module"
	"graft/server/internal/moduleapi"
	containercontract "graft/server/modules/container/contract"
	"graft/server/modules/container/terminal"
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

func TestRemoveDangerousActionsDisabledPublishesForceAudit(t *testing.T) {
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

	_, err = service.Remove(context.Background(), Ref{Value: "web"}, RemoveOptions{Force: true})
	if !errors.Is(err, errDangerousActionsDisabled) {
		t.Fatalf("expected dangerous action guard, got %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected one audit event, got %#v", events)
	}
	event := events[0]
	if event.Action != "ops.container.remove" || event.Success {
		t.Fatalf("unexpected audit event %#v", event)
	}
	if event.Metadata["force"] != true {
		t.Fatalf("expected force metadata, got %#v", event.Metadata)
	}
	if event.Metadata["endpoint"] != "unix:///var/run/docker.sock" {
		t.Fatalf("expected endpoint metadata, got %#v", event.Metadata)
	}
}

func TestServiceActionResponseCarriesMessageKey(t *testing.T) {
	t.Parallel()

	service, err := newService(containerServiceOptions{
		runtime:                 fakeRuntime{},
		enabled:                 true,
		dangerousActionsEnabled: true,
		defaultTail:             defaultContainerLogsDefaultTail,
		maxTail:                 defaultContainerLogsMaxTail,
	})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	result, err := service.Restart(context.Background(), Ref{Value: "web"})
	if err != nil {
		t.Fatalf("restart: %v", err)
	}
	if result.MessageKey != containercontract.ContainerActionRestartCompleted.String() {
		t.Fatalf("unexpected action message key %q", result.MessageKey)
	}
	mapped := toContainerAction(result)
	if mapped.MessageKey == nil || *mapped.MessageKey != containercontract.ContainerActionRestartCompleted.String() {
		t.Fatalf("expected mapped message key, got %#v", mapped.MessageKey)
	}
	if mapped.Message == nil || *mapped.Message != containercontract.ContainerActionRestartCompleted.String() {
		t.Fatalf("expected mapped fallback message, got %#v", mapped.Message)
	}
}

func TestServiceRemoveResponseCarriesMessageKey(t *testing.T) {
	t.Parallel()

	service, err := newService(containerServiceOptions{
		runtime:                 fakeRuntime{},
		enabled:                 true,
		dangerousActionsEnabled: true,
		defaultTail:             defaultContainerLogsDefaultTail,
		maxTail:                 defaultContainerLogsMaxTail,
	})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	result, err := service.Remove(context.Background(), Ref{Value: "web"}, RemoveOptions{Force: true})
	if err != nil {
		t.Fatalf("remove: %v", err)
	}
	if result.MessageKey != containercontract.ContainerActionRemoveCompleted.String() {
		t.Fatalf("unexpected action message key %q", result.MessageKey)
	}
}

func TestServiceAppliesEnvironmentDisplayPolicy(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		policy   string
		expected expectedEnvironmentPolicyResult
	}{
		{
			name:   "hidden",
			policy: containercontract.ContainerEnvironmentPolicyHidden.String(),
			expected: expectedEnvironmentPolicyResult{
				firstMasked:   true,
				firstDisplay:  "[HIDDEN]",
				firstHidden:   true,
				secondMasked:  true,
				secondDisplay: "[HIDDEN]",
				secondHidden:  true,
			},
		},
		{
			name:   "masked",
			policy: containercontract.ContainerEnvironmentPolicyMasked.String(),
			expected: expectedEnvironmentPolicyResult{
				firstValue:        "prod",
				firstDisplay:      "prod",
				secondMasked:      true,
				secondDisplay:     maskedEnvironmentPlaceholder,
				secondValueMasked: true,
			},
		},
		{
			name:   "plain",
			policy: containercontract.ContainerEnvironmentPolicyPlain.String(),
			expected: expectedEnvironmentPolicyResult{
				policy:            containercontract.ContainerEnvironmentPolicyMasked.String(),
				firstValue:        "prod",
				firstDisplay:      "prod",
				secondMasked:      true,
				secondDisplay:     maskedEnvironmentPlaceholder,
				secondValueMasked: true,
			},
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			service := newEnvironmentPolicyTestService(t, tc.policy)

			detail, err := service.Detail(context.Background(), Ref{Value: "web"})
			if err != nil {
				t.Fatalf("detail: %v", err)
			}
			assertEnvironmentPolicyResult(t, detail, tc.policy, tc.expected)
		})
	}
}

func TestServiceAppliesPlainEnvironmentPolicyWithPermissionContext(t *testing.T) {
	t.Parallel()

	service := newEnvironmentPolicyTestService(t, containercontract.ContainerEnvironmentPolicyPlain.String())
	detail, err := service.Detail(withEnvironmentPlainAccess(context.Background()), Ref{Value: "web"})
	if err != nil {
		t.Fatalf("detail: %v", err)
	}
	assertEnvironmentPolicyResult(t, detail, containercontract.ContainerEnvironmentPolicyPlain.String(), expectedEnvironmentPolicyResult{
		policy:        containercontract.ContainerEnvironmentPolicyPlain.String(),
		firstValue:    "prod",
		firstDisplay:  "prod",
		secondValue:   "secret",
		secondDisplay: "secret",
	})
}

func TestServiceExposesMaskedEnvironmentCopyValueOnlyWhenConfigured(t *testing.T) {
	t.Parallel()

	service := newEnvironmentPolicyTestService(t, containercontract.ContainerEnvironmentPolicyMasked.String())
	detail, err := service.Detail(withEnvironmentPlainAccess(context.Background()), Ref{Value: "web"})
	if err != nil {
		t.Fatalf("detail: %v", err)
	}
	if detail.Environment[1].Value != "" || detail.Environment[1].CopyValue != "" || detail.Environment[1].DisplayValue != maskedEnvironmentPlaceholder || !detail.Environment[1].ValueMasked {
		t.Fatalf("expected masked display state by default, got %#v", detail.Environment[1])
	}

	enabledService := newEnvironmentPolicyTestServiceWithValues(
		t,
		containercontract.ContainerEnvironmentPolicyMasked.String(),
		map[string]bool{containercontract.ContainerEnvironmentMaskedCopyEnabledConfig.String(): true},
	)
	detail, err = enabledService.Detail(withEnvironmentPlainAccess(context.Background()), Ref{Value: "web"})
	if err != nil {
		t.Fatalf("detail with copy enabled: %v", err)
	}
	if detail.Environment[1].Value != "" || detail.Environment[1].CopyValue != "secret" || !detail.Environment[1].Masked || detail.Environment[1].DisplayValue != maskedEnvironmentPlaceholder {
		t.Fatalf("expected masked display value to remain stable when copy is enabled, got %#v", detail.Environment[1])
	}
}

func TestServiceHiddenEnvironmentPolicyNeverExposesCopyValue(t *testing.T) {
	t.Parallel()

	service := newEnvironmentPolicyTestServiceWithValues(
		t,
		containercontract.ContainerEnvironmentPolicyHidden.String(),
		map[string]bool{containercontract.ContainerEnvironmentMaskedCopyEnabledConfig.String(): true},
	)
	detail, err := service.Detail(withEnvironmentPlainAccess(context.Background()), Ref{Value: "web"})
	if err != nil {
		t.Fatalf("detail: %v", err)
	}
	if detail.Environment[1].CopyValue != "" || detail.Environment[1].Value != "" || !detail.Environment[1].ValueHidden {
		t.Fatalf("expected hidden policy to block real-value copy, got %#v", detail.Environment[1])
	}
}

type expectedEnvironmentPolicyResult struct {
	policy            string
	firstValue        string
	firstDisplay      string
	firstMasked       bool
	firstHidden       bool
	secondValue       string
	secondDisplay     string
	secondMasked      bool
	secondValueMasked bool
	secondHidden      bool
}

func newEnvironmentPolicyTestService(t *testing.T, policy string) *service {
	return newEnvironmentPolicyTestServiceWithValues(t, policy, nil)
}

func newEnvironmentPolicyTestServiceWithValues(t *testing.T, policy string, values map[string]bool) *service {
	t.Helper()

	configValues := map[string]bool{
		containercontract.ContainerRuntimeEnabledConfig.String(): true,
	}
	for key, value := range values {
		configValues[key] = value
	}

	service, err := newService(containerServiceOptions{
		runtime: fakeRuntime{},
		systemConfig: serviceTestPolicyConfig{
			serviceTestSystemConfig: serviceTestSystemConfig{values: configValues},
			policy:                  policy,
		},
		enabled:     true,
		defaultTail: defaultContainerLogsDefaultTail,
		maxTail:     defaultContainerLogsMaxTail,
	})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	return service
}

func assertEnvironmentPolicyResult(
	t *testing.T,
	detail Detail,
	policy string,
	expected expectedEnvironmentPolicyResult,
) {
	t.Helper()

	expectedPolicy := firstNonEmpty(expected.policy, policy)
	if detail.EnvironmentPolicy != expectedPolicy {
		t.Fatalf("expected policy %q, got %#v", expectedPolicy, detail)
	}
	if len(detail.Environment) != 2 {
		t.Fatalf("expected two environment entries, got %#v", detail.Environment)
	}
	if detail.Environment[0].Value != expected.firstValue ||
		detail.Environment[0].DisplayValue != firstNonEmpty(expected.firstDisplay, expected.firstValue) ||
		detail.Environment[0].Masked != expected.firstMasked ||
		detail.Environment[0].ValueHidden != expected.firstHidden ||
		detail.Environment[0].Sensitive {
		t.Fatalf("unexpected first environment entry %#v", detail.Environment[0])
	}
	if detail.Environment[1].Value != expected.secondValue ||
		detail.Environment[1].DisplayValue != firstNonEmpty(expected.secondDisplay, expected.secondValue) ||
		detail.Environment[1].Masked != expected.secondMasked ||
		detail.Environment[1].ValueMasked != expected.secondValueMasked ||
		detail.Environment[1].ValueHidden != expected.secondHidden ||
		!detail.Environment[1].Sensitive {
		t.Fatalf("unexpected second environment entry %#v", detail.Environment[1])
	}
}

func TestServiceActionFailurePublishesAuditWithRuntimeContext(t *testing.T) {
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
		runtime:                 failingRuntime{err: errInvalidContainerState},
		auditBus:                bus,
		moduleName:              moduleID,
		enabled:                 true,
		dangerousActionsEnabled: true,
		defaultTail:             defaultContainerLogsDefaultTail,
		maxTail:                 defaultContainerLogsMaxTail,
	})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	_, err = service.Stop(context.Background(), Ref{Value: "web"})
	if !errors.Is(err, errInvalidContainerState) {
		t.Fatalf("expected invalid state, got %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected one audit event, got %#v", events)
	}
	event := events[0]
	if event.Action != "ops.container.stop" || event.Success {
		t.Fatalf("unexpected audit event %#v", event)
	}
	if event.MessageKey != containercontract.ContainerInvalidState.String() {
		t.Fatalf("unexpected message key %q", event.MessageKey)
	}
	if event.Metadata["runtime"] != runtimeNameDocker {
		t.Fatalf("expected runtime metadata, got %#v", event.Metadata)
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

	_, err = service.List(context.Background(), ListQuery{})
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

	_, err = service.List(context.Background(), ListQuery{})
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

	if _, err := service.List(context.Background(), ListQuery{}); err != nil {
		t.Fatalf("expected read path to stay available, got %v", err)
	}
	_, err = service.Start(context.Background(), Ref{Value: "web"})
	if !errors.Is(err, errDangerousActionsDisabled) {
		t.Fatalf("expected dangerous actions guard, got %v", err)
	}
}

func TestServiceListAppliesPaginationFiltersAndActionAvailability(t *testing.T) {
	t.Parallel()

	service := newListTestService(t, false)
	result, err := service.List(context.Background(), ListQuery{
		Limit:   1,
		Offset:  1,
		Keyword: "graft",
	})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if result.Total != 2 || result.Limit != 1 || result.Offset != 1 {
		t.Fatalf("unexpected page metadata %#v", result)
	}
	if result.Summary.Total != 2 || result.Summary.Running != 1 || result.Summary.Stopped != 1 {
		t.Fatalf("unexpected summary %#v", result.Summary)
	}
	if len(result.Items) != 1 || result.Items[0].Name != "graft-worker" {
		t.Fatalf("unexpected paged items %#v", result.Items)
	}
	if result.Items[0].CanStart || result.Items[0].CanStop || result.Items[0].CanRestart || result.Items[0].CanRemove {
		t.Fatalf("expected dangerous action gate to disable row actions, got %#v", result.Items[0])
	}
}

func TestServiceBatchActionAllowsPartialSuccess(t *testing.T) {
	t.Parallel()

	bus := eventbus.New(zap.NewNop())
	events := make([]moduleapi.AuditEvent, 0, 2)
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
		runtime:                 selectiveRemoveRuntime{failID: "bad"},
		auditBus:                bus,
		enabled:                 true,
		dangerousActionsEnabled: true,
		defaultTail:             defaultContainerLogsDefaultTail,
		maxTail:                 defaultContainerLogsMaxTail,
	})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	ctx := httpx.WithRequestAuditContext(context.Background(), httpx.RequestAuditContext{RequestID: "req-batch"})

	result, err := service.BatchAction(ctx, BatchActionCommand{
		Action: containerActionRemove,
		IDs:    []string{"ok", "bad"},
		Force:  true,
	})
	if err != nil {
		t.Fatalf("batch action should aggregate partial failures, got %v", err)
	}
	if result.SuccessCount != 1 || result.FailedCount != 1 || result.MessageKey != containercontract.ContainerBatchActionPartial.String() {
		t.Fatalf("unexpected batch result %#v", result)
	}
	if len(result.Items) != 2 || !result.Items[0].Success || result.Items[1].Success {
		t.Fatalf("unexpected batch items %#v", result.Items)
	}
	if len(events) != 2 {
		t.Fatalf("expected one audit per batch item, got %#v", events)
	}
	for _, event := range events {
		if event.Metadata["force"] != true || event.Metadata["requestId"] != "req-batch" {
			t.Fatalf("expected force/request audit metadata, got %#v", event.Metadata)
		}
	}
}

func TestServiceListFiltersHealth(t *testing.T) {
	t.Parallel()

	service := newListTestService(t, true)
	healthResult, err := service.List(context.Background(), ListQuery{Health: containerHealthUnavailable})
	if err != nil {
		t.Fatalf("list by health: %v", err)
	}
	if healthResult.Total != 1 || healthResult.Items[0].Name != "cache" {
		t.Fatalf("unexpected health-filtered result %#v", healthResult)
	}
}

func TestServiceMountUsageListDoesNotScanAndUsesCache(t *testing.T) {
	t.Parallel()

	mount := Mount{Type: "bind", Source: "/host/data", Destination: "/data"}
	cached := MountUsage{
		MountID:     stableMountID(mount),
		ContainerID: "docker-inspect-id",
		Type:        "bind",
		Source:      "/host/data",
		Destination: "/data",
		SizeBytes:   1024,
		SizeDisplay: "1 KiB",
		Status:      containerMountUsageStatusMeasured,
		MeasuredAt:  "2026-06-17T00:00:00Z",
	}
	cache := newMountUsageCache(time.Minute)
	cache.set(mountUsageCacheKey(Ref{Value: "web"}, stableMountID(mount)), cached)
	runtime := &countingRuntime{mounts: []Mount{mount}}
	service, err := newService(containerServiceOptions{
		runtime:         runtime,
		mountUsageCache: cache,
		enabled:         true,
		defaultTail:     defaultContainerLogsDefaultTail,
		maxTail:         defaultContainerLogsMaxTail,
	})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	items, err := service.MountUsageList(context.Background(), Ref{Value: "web"})
	if err != nil {
		t.Fatalf("mount usage list: %v", err)
	}
	if len(items) != 1 || !items[0].Cached || items[0].SizeDisplay != "1 KiB" {
		t.Fatalf("expected cached mount usage, got %#v", items)
	}
	if items[0].ContainerID != "web" {
		t.Fatalf("expected list cache result to use request ref container id, got %#v", items[0])
	}
	if runtime.mountUsageCalls.Load() != 0 {
		t.Fatalf("expected cache-only list to avoid scan, got %d scans", runtime.mountUsageCalls.Load())
	}
}

func TestServiceDetailAttachesCachedMountUsageWithoutScanning(t *testing.T) {
	t.Parallel()

	mount := Mount{Type: "bind", Source: "/host/data", Destination: "/data"}
	mount.ID = stableMountID(mount)
	cached := MountUsage{
		MountID:     mount.ID,
		ContainerID: "web",
		Type:        "bind",
		Source:      "/host/data",
		Destination: "/data",
		SizeBytes:   2048,
		SizeDisplay: "2 KiB",
		Status:      containerMountUsageStatusMeasured,
		MeasuredAt:  "2026-06-17T00:00:00Z",
	}
	cache := newMountUsageCache(time.Minute)
	cache.set(mountUsageCacheKey(Ref{Value: "web"}, mount.ID), cached)
	runtime := &countingRuntime{
		detail: Detail{
			Summary: Summary{
				ID:        "web",
				ShortID:   "web",
				Name:      "web",
				Image:     "nginx:latest",
				Runtime:   runtimeNameDocker,
				CreatedAt: "2026-06-14T00:00:00Z",
				State:     "running",
				Status:    "running",
			},
			Mounts: []Mount{mount},
		},
	}
	service, err := newService(containerServiceOptions{
		runtime:         runtime,
		mountUsageCache: cache,
		enabled:         true,
		defaultTail:     defaultContainerLogsDefaultTail,
		maxTail:         defaultContainerLogsMaxTail,
	})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	detail, err := service.Detail(context.Background(), Ref{Value: "web"})
	if err != nil {
		t.Fatalf("detail: %v", err)
	}
	if len(detail.Mounts) != 1 || detail.Mounts[0].Usage == nil || detail.Mounts[0].Usage.SizeDisplay != "2 KiB" {
		t.Fatalf("expected cached usage attached to detail, got %#v", detail.Mounts)
	}
	if runtime.mountUsageCalls.Load() != 0 {
		t.Fatalf("expected detail cache attachment to avoid scan, got %d scans", runtime.mountUsageCalls.Load())
	}
}

func TestServiceRefreshMountUsageCachesMeasuredResult(t *testing.T) {
	t.Parallel()

	mount := Mount{Type: "bind", Source: "/host/data", Destination: "/data"}
	mount.ID = stableMountID(mount)
	runtime := &countingRuntime{
		mountUsage: MountUsage{
			MountID:     mount.ID,
			ContainerID: "web",
			Type:        "bind",
			Source:      "/host/data",
			Destination: "/data",
			SizeBytes:   1536,
			SizeDisplay: "1.5 KiB",
			Status:      containerMountUsageStatusMeasured,
			MeasuredAt:  "2026-06-17T00:00:00Z",
		},
	}
	service, err := newService(containerServiceOptions{
		runtime:     runtime,
		enabled:     true,
		defaultTail: defaultContainerLogsDefaultTail,
		maxTail:     defaultContainerLogsMaxTail,
	})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	usage, err := service.RefreshMountUsage(context.Background(), Ref{Value: "web"}, mount.ID)
	if err != nil {
		t.Fatalf("refresh mount usage: %v", err)
	}
	if usage.SizeDisplay != "1.5 KiB" || runtime.mountUsageCalls.Load() != 1 {
		t.Fatalf("unexpected refreshed usage %#v calls=%d", usage, runtime.mountUsageCalls.Load())
	}
	runtime.mountUsage.SizeBytes = 2048
	runtime.mountUsage.SizeDisplay = "2 KiB"
	usage, err = service.RefreshMountUsage(context.Background(), Ref{Value: "web"}, mount.ID)
	if err != nil {
		t.Fatalf("second refresh mount usage: %v", err)
	}
	if usage.Cached || usage.SizeDisplay != "2 KiB" || runtime.mountUsageCalls.Load() != 2 {
		t.Fatalf("expected refresh to bypass cache, got usage=%#v calls=%d", usage, runtime.mountUsageCalls.Load())
	}
	items, err := service.MountUsageList(context.Background(), Ref{Value: "web"})
	if err != nil {
		t.Fatalf("mount usage list: %v", err)
	}
	if len(items) != 1 || !items[0].Cached || items[0].SizeBytes != 2048 {
		t.Fatalf("expected cached refreshed usage, got %#v", items)
	}
}

func TestServiceRefreshMountUsageRejectsArbitraryPath(t *testing.T) {
	t.Parallel()

	runtime := &countingRuntime{}
	service, err := newService(containerServiceOptions{
		runtime:     runtime,
		enabled:     true,
		defaultTail: defaultContainerLogsDefaultTail,
		maxTail:     defaultContainerLogsMaxTail,
	})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	_, err = service.RefreshMountUsage(context.Background(), Ref{Value: "web"}, "/host/data")
	if !errors.Is(err, errInvalidRef) {
		t.Fatalf("expected invalid mount id, got %v", err)
	}
	if runtime.mountUsageCalls.Load() != 0 {
		t.Fatalf("expected invalid mount id to avoid runtime scan, got %d", runtime.mountUsageCalls.Load())
	}
}

func TestFormatIECBytesUsesExpectedUnits(t *testing.T) {
	t.Parallel()

	cases := map[int64]string{
		0:                      "0 B",
		512:                    "512 B",
		1024:                   "1 KiB",
		1536:                   "1.5 KiB",
		2 * 1024 * 1024:        "2 MiB",
		3 * 1024 * 1024 * 1024: "3 GiB",
	}
	for size, expected := range cases {
		if actual := formatIECBytes(size); actual != expected {
			t.Fatalf("expected %d as %q, got %q", size, expected, actual)
		}
	}
}

func TestSummarizeContainersAccountsForKnownRuntimeStates(t *testing.T) {
	t.Parallel()

	summary := summarizeContainers([]Summary{
		{State: "running", Health: containerHealthHealthy},
		{State: "created", Health: containerHealthUnavailable},
		{State: "exited", Health: containerHealthUnavailable},
		{State: "paused", Health: containerHealthUnavailable},
		{State: "restarting", Health: containerHealthUnavailable},
		{State: "dead", Health: containerHealthUnhealthy},
		{State: "unknown", Health: containerHealthUnavailable},
		{State: "removing", Health: containerHealthUnavailable},
	})

	if summary.Total != 8 || summary.Running != 1 || summary.Stopped != 4 || summary.Error != 3 {
		t.Fatalf("unexpected state summary %#v", summary)
	}
	if summary.Healthy != 1 || summary.Unhealthy != 1 || summary.HealthUnavailable != 6 {
		t.Fatalf("unexpected health summary %#v", summary)
	}
}

func newListTestService(t *testing.T, dangerousActionsEnabled bool) *service {
	t.Helper()

	service, err := newService(containerServiceOptions{
		runtime:                 listRuntime{},
		enabled:                 true,
		dangerousActionsEnabled: dangerousActionsEnabled,
		defaultTail:             defaultContainerLogsDefaultTail,
		maxTail:                 defaultContainerLogsMaxTail,
	})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	return service
}

func TestServiceListRejectsInvalidQuery(t *testing.T) {
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

	_, err = service.List(context.Background(), ListQuery{Limit: maxContainerListLimit + 1})
	if !errors.Is(err, errInvalidListQuery) {
		t.Fatalf("expected invalid list query, got %v", err)
	}
}

func TestNormalizeShellSessionRequestRejectsInvalidSize(t *testing.T) {
	t.Parallel()

	_, err := normalizeShellSessionRequest(ShellSessionRequest{
		Command: "sh",
		Cols:    0,
		Rows:    24,
	})
	if !errors.Is(err, errShellInvalidSize) {
		t.Fatalf("expected invalid shell size, got %v", err)
	}
}

func TestPublishShellSessionClosedDetachesCanceledRequestContext(t *testing.T) {
	t.Parallel()

	bus := &contextStateAuditBus{}
	service, err := newService(containerServiceOptions{
		runtime:     fakeRuntime{},
		auditBus:    bus,
		moduleName:  moduleID,
		enabled:     true,
		defaultTail: defaultContainerLogsDefaultTail,
		maxTail:     defaultContainerLogsMaxTail,
	})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	baseCtx, cancel := context.WithCancel(context.Background())
	ctx := httpx.WithRequestAuditContext(baseCtx, httpx.RequestAuditContext{
		RequestID: "req-shell-close",
		TraceID:   "trace-shell-close",
		Route:     "/api/ops/containers/:id/shell/ws",
		Method:    "GET",
	})
	ctx = moduleapi.WithRequestAuthContext(ctx, moduleapi.RequestAuthContext{
		User: &moduleapi.CurrentUser{ID: 7, Username: "admin"},
	})
	cancel()

	service.publishShellSessionClosed(ctx, ShellHandshake{
		UserID:       7,
		ResourceID:   "abc123",
		ResourceName: "web",
		Command:      "sh",
		SessionID:    "sess-1",
	}, time.Now().UTC().Add(-time.Second), "client_closed", nil)

	if len(bus.events) != 1 {
		t.Fatalf("expected one audit event, got %#v", bus.events)
	}
	if bus.canceled[0] {
		t.Fatalf("expected detached audit publish context")
	}
}

func TestPublishShellSessionFailedDetachesCanceledRequestContext(t *testing.T) {
	t.Parallel()

	bus := &contextStateAuditBus{}
	service, err := newService(containerServiceOptions{
		runtime:     fakeRuntime{},
		auditBus:    bus,
		moduleName:  moduleID,
		enabled:     true,
		defaultTail: defaultContainerLogsDefaultTail,
		maxTail:     defaultContainerLogsMaxTail,
	})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	baseCtx, cancel := context.WithCancel(context.Background())
	ctx := httpx.WithRequestAuditContext(baseCtx, httpx.RequestAuditContext{
		RequestID: "req-shell-failed",
		TraceID:   "trace-shell-failed",
		Route:     "/api/ops/containers/:id/shell/ws",
		Method:    "GET",
	})
	cancel()

	service.publishShellSessionFailed(ctx, ShellHandshake{
		UserID:       7,
		ResourceID:   "abc123",
		ResourceName: "web",
		Command:      "sh",
		SessionID:    "sess-2",
	}, "bridge_failed", errShellSessionFailed)

	if len(bus.events) != 1 {
		t.Fatalf("expected one audit event, got %#v", bus.events)
	}
	if bus.canceled[0] {
		t.Fatalf("expected detached audit publish context")
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
		case containercontract.ContainerEnvironmentPolicyConfig.String():
			definition.DefaultValue = mustRawJSON(containercontract.ContainerEnvironmentPolicyPlain.String())
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

type serviceTestPolicyConfig struct {
	serviceTestSystemConfig
	policy string
}

func (r serviceTestPolicyConfig) ResolveDefaultConfig(_ context.Context, key string) (string, error) {
	if key != containercontract.ContainerEnvironmentPolicyConfig.String() || strings.TrimSpace(r.policy) == "" {
		return "", errors.New("config unavailable")
	}
	return string(mustRawJSON(r.policy)), nil
}

type countingRuntime struct {
	calls           atomic.Int64
	mountUsageCalls atomic.Int64
	detail          Detail
	mounts          []Mount
	mountUsage      MountUsage
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
	if r.detail.ID != "" {
		return r.detail, nil
	}
	return Detail{}, nil
}

func (r *countingRuntime) Mounts(context.Context, Ref) ([]Mount, error) {
	r.calls.Add(1)
	if len(r.mounts) > 0 {
		return r.mounts, nil
	}
	mount := Mount{Type: "bind", Source: "/host/data", Destination: "/data"}
	mount.ID = stableMountID(mount)
	return []Mount{mount}, nil
}

func (r *countingRuntime) MountUsage(context.Context, Ref, string) (MountUsage, error) {
	r.calls.Add(1)
	r.mountUsageCalls.Add(1)
	return r.mountUsage, nil
}

func (r *countingRuntime) Logs(context.Context, Ref, LogQuery) (Logs, error) {
	r.calls.Add(1)
	return Logs{}, nil
}

func (r *countingRuntime) Shell(context.Context, Ref, string) (terminal.Session, error) {
	r.calls.Add(1)
	return newStubTerminalSession(), nil
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

func (r *countingRuntime) Remove(context.Context, Ref, RemoveOptions) (ActionResult, error) {
	r.calls.Add(1)
	return ActionResult{}, nil
}

func (r *countingRuntime) Close() error { return nil }

type failingRuntime struct {
	err error
}

type listRuntime struct{}

func (listRuntime) Info(context.Context) (RuntimeInfo, error) {
	return RuntimeInfo{Runtime: runtimeNameDocker, Status: "enabled", Endpoint: "unix:///var/run/docker.sock"}, nil
}

func (listRuntime) List(context.Context, ListQuery) ([]Summary, error) {
	return []Summary{
		{
			ID:         "111111111111abcdef",
			ShortID:    "111111111111",
			Name:       "graft-web",
			Names:      []string{"graft-web"},
			Image:      "graft/web:latest",
			Runtime:    runtimeNameDocker,
			CreatedAt:  "2026-06-14T00:00:00Z",
			State:      "running",
			Status:     "Up",
			Health:     containerHealthHealthy,
			Ports:      []Port{{PrivatePort: 80, PublicPort: intPtr(8080), Type: "tcp"}},
			Labels:     map[string]string{composeProjectLabel: "graft", composeServiceLabel: "web"},
			CanStop:    true,
			CanRestart: true,
		},
		{
			ID:         "222222222222abcdef",
			ShortID:    "222222222222",
			Name:       "graft-worker",
			Names:      []string{"graft-worker"},
			Image:      "graft/worker:latest",
			Runtime:    runtimeNameDocker,
			CreatedAt:  "2026-06-14T00:00:00Z",
			State:      "exited",
			Status:     "Exited",
			Health:     containerHealthNone,
			CanStart:   true,
			CanRestart: true,
		},
		{
			ID:         "333333333333abcdef",
			ShortID:    "333333333333",
			Name:       "cache",
			Names:      []string{"cache"},
			Image:      "redis:latest",
			Runtime:    runtimeNameDocker,
			CreatedAt:  "2026-06-14T00:00:00Z",
			State:      "running",
			Status:     "Up",
			CanStop:    true,
			CanRestart: true,
		},
	}, nil
}

func (listRuntime) Detail(context.Context, Ref) (Detail, error)  { return Detail{}, nil }
func (listRuntime) Mounts(context.Context, Ref) ([]Mount, error) { return nil, nil }
func (listRuntime) MountUsage(context.Context, Ref, string) (MountUsage, error) {
	return MountUsage{}, nil
}
func (listRuntime) Logs(context.Context, Ref, LogQuery) (Logs, error) { return Logs{}, nil }
func (listRuntime) Shell(context.Context, Ref, string) (terminal.Session, error) {
	return newStubTerminalSession(), nil
}
func (listRuntime) Start(context.Context, Ref) (ActionResult, error)   { return ActionResult{}, nil }
func (listRuntime) Stop(context.Context, Ref) (ActionResult, error)    { return ActionResult{}, nil }
func (listRuntime) Restart(context.Context, Ref) (ActionResult, error) { return ActionResult{}, nil }
func (listRuntime) Remove(context.Context, Ref, RemoveOptions) (ActionResult, error) {
	return ActionResult{}, nil
}
func (listRuntime) Close() error { return nil }

func (r failingRuntime) Info(context.Context) (RuntimeInfo, error) {
	return RuntimeInfo{}, r.err
}

func (r failingRuntime) List(context.Context, ListQuery) ([]Summary, error) {
	return nil, r.err
}

func (r failingRuntime) Detail(context.Context, Ref) (Detail, error) {
	return Detail{}, r.err
}

func (r failingRuntime) Mounts(context.Context, Ref) ([]Mount, error) {
	return nil, r.err
}

func (r failingRuntime) MountUsage(context.Context, Ref, string) (MountUsage, error) {
	return MountUsage{}, r.err
}

func (r failingRuntime) Logs(context.Context, Ref, LogQuery) (Logs, error) {
	return Logs{}, r.err
}

func (r failingRuntime) Shell(context.Context, Ref, string) (terminal.Session, error) {
	return nil, r.err
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

func (r failingRuntime) Remove(context.Context, Ref, RemoveOptions) (ActionResult, error) {
	return ActionResult{}, r.err
}

func (r failingRuntime) Close() error { return nil }

type selectiveRemoveRuntime struct {
	failID string
}

func (r selectiveRemoveRuntime) Info(context.Context) (RuntimeInfo, error) {
	return fakeRuntime{}.Info(context.Background())
}
func (r selectiveRemoveRuntime) List(context.Context, ListQuery) ([]Summary, error) {
	return fakeRuntime{}.List(context.Background(), ListQuery{})
}
func (r selectiveRemoveRuntime) Detail(context.Context, Ref) (Detail, error) {
	return fakeRuntime{}.Detail(context.Background(), Ref{Value: "web"})
}
func (r selectiveRemoveRuntime) Mounts(context.Context, Ref) ([]Mount, error) {
	return fakeRuntime{}.Mounts(context.Background(), Ref{Value: "web"})
}
func (r selectiveRemoveRuntime) MountUsage(context.Context, Ref, string) (MountUsage, error) {
	return fakeRuntime{}.MountUsage(context.Background(), Ref{Value: "web"}, "")
}
func (r selectiveRemoveRuntime) Logs(context.Context, Ref, LogQuery) (Logs, error) {
	return Logs{}, nil
}
func (r selectiveRemoveRuntime) Shell(context.Context, Ref, string) (terminal.Session, error) {
	return newStubTerminalSession(), nil
}
func (r selectiveRemoveRuntime) Start(context.Context, Ref) (ActionResult, error) {
	return ActionResult{}, nil
}
func (r selectiveRemoveRuntime) Stop(context.Context, Ref) (ActionResult, error) {
	return ActionResult{}, nil
}
func (r selectiveRemoveRuntime) Restart(context.Context, Ref) (ActionResult, error) {
	return ActionResult{}, nil
}
func (r selectiveRemoveRuntime) Remove(_ context.Context, ref Ref, _ RemoveOptions) (ActionResult, error) {
	if ref.Value == r.failID {
		result := fakeAction(containerActionRemove)
		result.ID = ref.Value
		return result, errInvalidContainerState
	}
	result := fakeAction(containerActionRemove)
	result.ID = ref.Value
	result.StatusAfter = actionStatusRemoved
	return result, nil
}
func (r selectiveRemoveRuntime) Close() error { return nil }

type fakeRuntime struct{}

func (fakeRuntime) Info(context.Context) (RuntimeInfo, error) {
	return RuntimeInfo{Runtime: runtimeNameDocker, Status: "enabled", Endpoint: "unix:///var/run/docker.sock"}, nil
}

func (fakeRuntime) List(context.Context, ListQuery) ([]Summary, error) {
	return []Summary{fakeSummary()}, nil
}

func (fakeRuntime) Detail(context.Context, Ref) (Detail, error) {
	return Detail{
		Summary: fakeSummary(),
		Environment: []EnvironmentVariable{
			{Key: "APP_ENV", Value: "prod", Source: dockerEnvironmentSource},
			{Key: "API_TOKEN", Value: "secret", Source: dockerEnvironmentSource},
		},
	}, nil
}

func (fakeRuntime) Mounts(context.Context, Ref) ([]Mount, error) {
	mount := Mount{Type: "bind", Source: "/host/data", Destination: "/data"}
	mount.ID = stableMountID(mount)
	return []Mount{mount}, nil
}

func (fakeRuntime) MountUsage(context.Context, Ref, string) (MountUsage, error) {
	mount := Mount{Type: "bind", Source: "/host/data", Destination: "/data"}
	mount.ID = stableMountID(mount)
	return mountUsageFromMount("abc123", mount, containerMountUsageStatusMeasured, 1024, "2026-06-17T00:00:00Z"), nil
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

func (fakeRuntime) Shell(context.Context, Ref, string) (terminal.Session, error) {
	return newStubTerminalSession(), nil
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

func (fakeRuntime) Remove(context.Context, Ref, RemoveOptions) (ActionResult, error) {
	result := fakeAction(containerActionRemove)
	result.StatusAfter = actionStatusRemoved
	return result, nil
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
		CanRemove: true,
	}
}

func fakeMountID() string {
	mount := Mount{Type: "bind", Source: "/host/data", Destination: "/data"}
	return stableMountID(mount)
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

type stubTerminalSession struct {
	output  chan []byte
	errs    chan error
	once    sync.Once
	started bool
}

func newStubTerminalSession() *stubTerminalSession {
	return &stubTerminalSession{
		output: make(chan []byte, 4),
		errs:   make(chan error, 1),
	}
}

func (s *stubTerminalSession) Start(context.Context, terminal.Size) error {
	if !s.started {
		s.started = true
		s.output <- []byte("/app # ")
	}
	return nil
}

func (s *stubTerminalSession) Write(_ context.Context, data []byte) error {
	if len(data) > 0 {
		s.output <- append([]byte(nil), data...)
	}
	return nil
}

func (s *stubTerminalSession) Resize(context.Context, terminal.Size) error { return nil }
func (s *stubTerminalSession) Output() <-chan []byte                       { return s.output }
func (s *stubTerminalSession) Errors() <-chan error                        { return s.errs }
func (s *stubTerminalSession) Close(context.Context) error {
	s.once.Do(func() {
		close(s.output)
		close(s.errs)
	})
	return nil
}

type contextStateAuditBus struct {
	events   []eventbus.Event
	canceled []bool
}

func (b *contextStateAuditBus) Subscribe(string, eventbus.Handler) error { return nil }

func (b *contextStateAuditBus) Publish(ctx context.Context, event eventbus.Event) error {
	b.events = append(b.events, event)
	b.canceled = append(b.canceled, ctx.Err() != nil)
	return nil
}
