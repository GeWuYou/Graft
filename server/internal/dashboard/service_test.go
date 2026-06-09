// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package dashboard

import (
	"context"
	"errors"
	"testing"
	"time"

	"graft/server/internal/config"
	generated "graft/server/internal/contract/openapi/generated"
	"graft/server/internal/moduleapi"
)

func TestServiceFiltersWidgetsByRequiredPermissions(t *testing.T) {
	t.Parallel()

	registry := NewRegistry()
	mustRegisterWidget(t, registry, WidgetDefinition{
		ID:                  "core.visible",
		ModuleKey:           "core",
		Type:                WidgetTypeHealth,
		Size:                WidgetSizeSmall,
		RequiredPermissions: []string{"modules.runtime.read"},
		Loader:              noopLoader(),
	})
	mustRegisterWidget(t, registry, WidgetDefinition{
		ID:                  "core.hidden",
		ModuleKey:           "core",
		Type:                WidgetTypeHealth,
		Size:                WidgetSizeSmall,
		RequiredPermissions: []string{"audit.read"},
		Loader:              noopLoader(),
	})

	service := NewService(ServiceOptions{
		Config:   &config.Config{App: config.AppConfig{Env: "test"}},
		Registry: registry,
		Authorizer: testAuthorizer{allow: map[string]bool{
			"modules.runtime.read": true,
		}},
	})

	summary := service.Summary(context.Background(), testRequestAuth())
	if len(summary.Widgets) != 1 || summary.Widgets[0].Id != "core.visible" {
		t.Fatalf("expected only authorized widget, got %#v", summary.Widgets)
	}
	if summary.SystemSummary.VisibleWidgets != 1 {
		t.Fatalf("expected visible widget count 1, got %#v", summary.SystemSummary)
	}
}

func TestServiceFiltersQuickLinksByRequiredPermissions(t *testing.T) {
	t.Parallel()

	registry := NewRegistry()
	mustRegisterQuickLink(t, registry, QuickLinkDefinition{
		ID:                  "core.visible-link",
		ModuleKey:           "core",
		Title:               "Runtime",
		RouteLocation:       "/modules/runtime",
		RequiredPermissions: []string{"modules.runtime.read"},
		Order:               20,
	})
	mustRegisterQuickLink(t, registry, QuickLinkDefinition{
		ID:                  "core.hidden-link",
		ModuleKey:           "core",
		Title:               "Audit",
		RouteLocation:       "/audit/logs",
		RequiredPermissions: []string{"audit.read"},
		Order:               10,
	})
	mustRegisterWidget(t, registry, WidgetDefinition{
		ID:        "core.visible-widget",
		ModuleKey: "core",
		Type:      WidgetTypeHealth,
		Size:      WidgetSizeSmall,
		Loader:    noopLoader(),
	})

	service := NewService(ServiceOptions{
		Registry: registry,
		Authorizer: testAuthorizer{allow: map[string]bool{
			"modules.runtime.read": true,
		}},
	})

	summary := service.Summary(context.Background(), testRequestAuth())
	if len(summary.QuickLinks) != 1 || summary.QuickLinks[0].Id != "core.visible-link" {
		t.Fatalf("expected only authorized quick link, got %#v", summary.QuickLinks)
	}
	if len(summary.Widgets) != 1 || summary.Widgets[0].Id != "core.visible-widget" {
		t.Fatalf("expected widget visibility to remain independent, got %#v", summary.Widgets)
	}
}

func TestServiceReturnsOrderedQuickLinksAndWidgets(t *testing.T) {
	t.Parallel()

	registry := NewRegistry()
	mustRegisterQuickLink(t, registry, QuickLinkDefinition{
		ID:            "b.link",
		ModuleKey:     "core",
		Title:         "B",
		RouteLocation: "/b",
		Order:         20,
	})
	mustRegisterQuickLink(t, registry, QuickLinkDefinition{
		ID:            "a.link",
		ModuleKey:     "core",
		Title:         "A",
		RouteLocation: "/a",
		Order:         10,
	})
	mustRegisterWidget(t, registry, testWidgetDefinitionWithOrder("b.widget", 20))
	mustRegisterWidget(t, registry, testWidgetDefinitionWithOrder("a.widget", 10))

	summary := NewService(ServiceOptions{Registry: registry}).Summary(context.Background(), testRequestAuth())
	if got := []string{summary.QuickLinks[0].Id, summary.QuickLinks[1].Id}; got[0] != "a.link" || got[1] != "b.link" {
		t.Fatalf("unexpected quick link order: %#v", got)
	}
	if got := []string{summary.Widgets[0].Id, summary.Widgets[1].Id}; got[0] != "a.widget" || got[1] != "b.widget" {
		t.Fatalf("unexpected widget order: %#v", got)
	}
}

func TestServiceUsesEmptyRegistryWhenRegistryIsNil(t *testing.T) {
	t.Parallel()

	summary := NewService(ServiceOptions{}).Summary(context.Background(), testRequestAuth())
	if len(summary.QuickLinks) != 0 || len(summary.Widgets) != 0 {
		t.Fatalf("expected empty dashboard contributions, got links=%#v widgets=%#v", summary.QuickLinks, summary.Widgets)
	}
}

func TestServiceReturnsErrorWidgetWhenLoaderFails(t *testing.T) {
	t.Parallel()

	registry := NewRegistry()
	mustRegisterWidget(t, registry, WidgetDefinition{
		ID:        "core.error",
		ModuleKey: "core",
		Type:      WidgetTypeHealth,
		Size:      WidgetSizeSmall,
		Loader: WidgetLoaderFunc(func(context.Context, WidgetRequest) (WidgetPayload, error) {
			return nil, errors.New("load failed")
		}),
	})

	widget := NewService(ServiceOptions{Registry: registry}).Summary(context.Background(), testRequestAuth()).Widgets[0]
	if widget.Status == nil || *widget.Status != generated.DashboardWidgetStatusError {
		t.Fatalf("expected error status, got %#v", widget.Status)
	}
	if widget.Error == nil || widget.Error.Code != errorCodeLoadFailed {
		t.Fatalf("expected load error, got %#v", widget.Error)
	}
}

func TestServiceRecoversLoaderPanic(t *testing.T) {
	t.Parallel()

	registry := NewRegistry()
	mustRegisterWidget(t, registry, WidgetDefinition{
		ID:        "core.panic",
		ModuleKey: "core",
		Type:      WidgetTypeHealth,
		Size:      WidgetSizeSmall,
		Loader: WidgetLoaderFunc(func(context.Context, WidgetRequest) (WidgetPayload, error) {
			panic("boom")
		}),
	})

	widget := NewService(ServiceOptions{Registry: registry}).Summary(context.Background(), testRequestAuth()).Widgets[0]
	if widget.Error == nil || widget.Error.Code != errorCodePanic {
		t.Fatalf("expected panic error, got %#v", widget.Error)
	}
}

func TestServiceTimesOutSlowLoader(t *testing.T) {
	t.Parallel()

	registry := NewRegistry()
	mustRegisterWidget(t, registry, WidgetDefinition{
		ID:            "core.timeout",
		ModuleKey:     "core",
		Type:          WidgetTypeHealth,
		Size:          WidgetSizeSmall,
		LoaderTimeout: time.Millisecond,
		Loader: WidgetLoaderFunc(func(ctx context.Context, _ WidgetRequest) (WidgetPayload, error) {
			<-ctx.Done()
			return nil, ctx.Err()
		}),
	})

	widget := NewService(ServiceOptions{Registry: registry}).Summary(context.Background(), testRequestAuth()).Widgets[0]
	if widget.Error == nil || widget.Error.Code != errorCodeTimeout {
		t.Fatalf("expected timeout error, got %#v", widget.Error)
	}
}

func TestServiceReportsCanceledLoaderContextWithoutTimeout(t *testing.T) {
	t.Parallel()

	registry := NewRegistry()
	mustRegisterWidget(t, registry, WidgetDefinition{
		ID:        "core.canceled",
		ModuleKey: "core",
		Type:      WidgetTypeHealth,
		Size:      WidgetSizeSmall,
		Loader: WidgetLoaderFunc(func(ctx context.Context, _ WidgetRequest) (WidgetPayload, error) {
			<-ctx.Done()
			return nil, ctx.Err()
		}),
	})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	widget := NewService(ServiceOptions{Registry: registry}).Summary(ctx, testRequestAuth()).Widgets[0]
	if widget.Error == nil || widget.Error.Code != errorCodeLoadFailed {
		t.Fatalf("expected canceled context to be reported as load failure, got %#v", widget.Error)
	}
	if widget.Error.Message == nil || *widget.Error.Message != context.Canceled.Error() {
		t.Fatalf("expected canceled context message, got %#v", widget.Error)
	}
}

func TestServiceHidesWidgetsWithHiddenState(t *testing.T) {
	t.Parallel()

	registry := NewRegistry()
	mustRegisterWidget(t, registry, WidgetDefinition{
		ID:        "scheduler.empty",
		ModuleKey: "scheduler",
		Type:      WidgetTypeStatGroup,
		Size:      WidgetSizeMedium,
		Loader: WidgetLoaderFunc(func(context.Context, WidgetRequest) (WidgetPayload, error) {
			return WidgetPayload{"items": []map[string]any{}, "visible": false, "state": string(WidgetStateHidden)}, nil
		}),
	})
	mustRegisterWidget(t, registry, WidgetDefinition{
		ID:        "monitor.health",
		ModuleKey: "monitor",
		Type:      WidgetTypeHealth,
		Size:      WidgetSizeMedium,
		Loader:    noopLoader(),
	})

	summary := NewService(ServiceOptions{Registry: registry}).Summary(context.Background(), testRequestAuth())
	if len(summary.Widgets) != 1 || summary.Widgets[0].Id != "monitor.health" {
		t.Fatalf("expected hidden widget to be omitted, got %#v", summary.Widgets)
	}
	if summary.SystemSummary.VisibleWidgets != 1 {
		t.Fatalf("expected visible widget count 1, got %#v", summary.SystemSummary.VisibleWidgets)
	}
}

func TestServiceSortsWidgetsByPriorityBeforeOrder(t *testing.T) {
	t.Parallel()

	registry := NewRegistry()
	mustRegisterWidget(t, registry, WidgetDefinition{
		ID:        "monitor.health",
		ModuleKey: "monitor",
		Type:      WidgetTypeHealth,
		Size:      WidgetSizeMedium,
		Priority:  WidgetPriorityInfo,
		Order:     1,
		Loader:    noopLoader(),
	})
	mustRegisterWidget(t, registry, WidgetDefinition{
		ID:        "audit.risk",
		ModuleKey: "audit",
		Type:      WidgetTypeAlertList,
		Size:      WidgetSizeMedium,
		Priority:  WidgetPriorityNormal,
		Order:     100,
		Loader: WidgetLoaderFunc(func(context.Context, WidgetRequest) (WidgetPayload, error) {
			return WidgetPayload{"items": []map[string]any{}, "state": string(WidgetStateCritical)}, nil
		}),
	})

	widgets := NewService(ServiceOptions{Registry: registry}).Summary(context.Background(), testRequestAuth()).Widgets
	if got := []string{widgets[0].Id, widgets[1].Id}; got[0] != "audit.risk" || got[1] != "monitor.health" {
		t.Fatalf("unexpected priority order: %#v", got)
	}
	if string(widgets[0].Priority) != string(WidgetPriorityCritical) {
		t.Fatalf("expected critical priority override, got %#v", widgets[0].Priority)
	}
}

func TestServiceBuildsFrameworkFieldsAndSummaryMetrics(t *testing.T) {
	t.Parallel()

	registry := NewRegistry()
	mustRegisterWidget(t, registry, WidgetDefinition{
		ID:        "scheduler.attention",
		ModuleKey: "scheduler",
		Type:      WidgetTypeStatGroup,
		Size:      WidgetSizeMedium,
		Category:  WidgetCategoryOperation,
		Priority:  WidgetPriorityWarning,
		Order:     20,
		Action: WidgetAction{
			LabelKey: "dashboard.actions.details",
			Label:    "View details",
			Route:    "/scheduler/tasks",
		},
		Loader: WidgetLoaderFunc(func(context.Context, WidgetRequest) (WidgetPayload, error) {
			return WidgetPayload{
				"items":             []map[string]any{},
				"failed_tasks":      2,
				"high_risk_events":  3,
				"abnormal_services": 1,
			}, nil
		}),
	})

	summary := NewService(ServiceOptions{Registry: registry}).Summary(context.Background(), testRequestAuth())
	widget := summary.Widgets[0]
	if string(widget.Category) != string(WidgetCategoryOperation) || string(widget.Priority) != string(WidgetPriorityWarning) {
		t.Fatalf("unexpected framework fields: %#v", widget)
	}
	if !widget.Visible || string(widget.State) != string(WidgetStateNormal) {
		t.Fatalf("unexpected visibility state: visible=%v state=%q", widget.Visible, widget.State)
	}
	if widget.Action == nil ||
		widget.Action.LabelKey != "dashboard.actions.details" ||
		widget.Action.Label != "View details" ||
		widget.Action.Route != "/scheduler/tasks" {
		t.Fatalf("unexpected action: %#v", widget.Action)
	}
	if summary.SystemSummary.FailedTasks != 2 ||
		summary.SystemSummary.HighRiskEvents != 3 ||
		summary.SystemSummary.AbnormalServices != 1 {
		t.Fatalf("unexpected summary metrics: %#v", summary.SystemSummary)
	}
}

func mustRegisterWidget(t *testing.T, registry *Registry, definition WidgetDefinition) {
	t.Helper()
	if err := registry.Register(definition); err != nil {
		t.Fatalf("register widget: %v", err)
	}
}

func mustRegisterQuickLink(t *testing.T, registry *Registry, definition QuickLinkDefinition) {
	t.Helper()
	if err := registry.RegisterQuickLink(definition); err != nil {
		t.Fatalf("register quick link: %v", err)
	}
}

func testRequestAuth() moduleapi.RequestAuthContext {
	return moduleapi.RequestAuthContext{
		User: &moduleapi.CurrentUser{ID: 7, Username: "alice", DisplayName: "Alice"},
	}
}

type testAuthorizer struct {
	allow map[string]bool
}

func (a testAuthorizer) Authorize(_ context.Context, _ moduleapi.RequestAuthContext, permission string) error {
	if a.allow[permission] {
		return nil
	}
	return moduleapi.ErrPermissionDenied
}
