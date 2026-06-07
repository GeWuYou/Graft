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

func mustRegisterWidget(t *testing.T, registry *Registry, definition WidgetDefinition) {
	t.Helper()
	if err := registry.Register(definition); err != nil {
		t.Fatalf("register widget: %v", err)
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
