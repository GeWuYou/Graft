package app

import (
	"context"
	"errors"
	"fmt"

	generated "graft/server/internal/contract/openapi/generated"
	"graft/server/internal/dashboard"
	"graft/server/internal/httpx"
	"graft/server/internal/i18n"
	"graft/server/internal/logger"
	"graft/server/internal/module"
	"graft/server/internal/moduleapi"
	"graft/server/internal/moduleregistry"
	"graft/server/internal/moduleruntime"
)

func (r *Runtime) registerCoreDashboardWidgets() error {
	if r.dashboardRegistry == nil {
		return errors.New("dashboard registry is unavailable")
	}

	for _, register := range []func() error{
		r.registerCoreModuleRuntimeDashboard,
		r.registerCoreAccessLogDashboard,
		r.registerCoreAppLogDashboard,
	} {
		if err := register(); err != nil {
			return err
		}
	}

	return nil
}

func (r *Runtime) registerCoreModuleRuntimeDashboard() error {
	if err := r.dashboardRegistry.Register(dashboard.WidgetDefinition{
		ID:                  "core.module-runtime-health",
		ModuleKey:           "core",
		TitleKey:            moduleRuntimeHealthTitleKey,
		Title:               r.mustLookupCoreDisplay(moduleRuntimeHealthTitleKey),
		DescriptionKey:      moduleRuntimeHealthDescriptionKey,
		Description:         r.mustLookupCoreDisplay(moduleRuntimeHealthDescriptionKey),
		Type:                dashboard.WidgetTypeHealth,
		Size:                dashboard.WidgetSizeMedium,
		Category:            dashboard.WidgetCategorySystem,
		Priority:            dashboard.WidgetPriorityInfo,
		Order:               coreModuleRuntimeHealthWidgetOrder,
		RouteLocation:       moduleruntime.MenuRuntimePath(),
		Action:              r.dashboardDetailsAction(moduleruntime.MenuRuntimePath()),
		RequiredPermissions: []string{moduleruntime.PermissionRead},
		Loader: dashboard.WidgetLoaderFunc(func(context.Context, dashboard.WidgetRequest) (dashboard.WidgetPayload, error) {
			snapshot := moduleruntime.BuildSnapshot(r.config, r.moduleRuntimeSpecs())
			items := make([]dashboard.HealthItem, 0, len(snapshot.Items))
			for _, item := range snapshot.Items {
				if item.Health == generated.ModuleRuntimeItemHealthHealthy {
					continue
				}
				items = append(items, dashboard.HealthItem{
					Key:            item.ModuleKey,
					LabelKey:       "dashboard.widget.moduleRuntimeHealth.item." + item.ModuleKey,
					Label:          item.ModuleKey,
					Status:         dashboard.HealthStatus(string(item.Health)),
					DescriptionKey: moduleRuntimeStatusDescriptionKey(item.RuntimeStatus),
					Description:    string(item.RuntimeStatus),
					RouteLocation:  moduleruntime.MenuRuntimePath(),
				})
			}

			abnormalModules := snapshot.Summary.DegradedModules + snapshot.Summary.UnknownModules
			summaryStatus := dashboard.HealthStatusHealthy
			widgetState := dashboard.WidgetStateNormal
			widgetPriority := dashboard.WidgetPriorityInfo
			if abnormalModules > 0 {
				summaryStatus = dashboard.HealthStatusDegraded
				widgetState = dashboard.WidgetStateWarning
				widgetPriority = dashboard.WidgetPriorityWarning
			}
			if snapshot.Summary.EnabledModules == 0 && snapshot.Summary.TotalModules > 0 {
				summaryStatus = dashboard.HealthStatusDisabled
				widgetState = dashboard.WidgetStateWarning
				widgetPriority = dashboard.WidgetPriorityWarning
			}

			return dashboard.WidgetPayload{
				"summary": dashboard.HealthSummaryItem{
					Status:   summaryStatus,
					LabelKey: moduleRuntimeHealthSummaryKey,
					Label:    r.mustLookupCoreDisplay(moduleRuntimeHealthSummaryKey),
				},
				"items":             items,
				"healthy_modules":   snapshot.Summary.HealthyModules,
				"enabled_modules":   snapshot.Summary.EnabledModules,
				"abnormal_services": abnormalModules,
				"state":             string(widgetState),
				"priority":          string(widgetPriority),
			}, nil
		}),
	}); err != nil {
		return fmt.Errorf("register core dashboard widget: %w", err)
	}

	return nil
}

func (r *Runtime) registerCoreAccessLogDashboard() error {
	if repo := r.server.AccessLogRepository(); repo != nil {
		if err := r.dashboardRegistry.Register(dashboard.WidgetDefinition{
			ID:                  httpx.AccessLogDashboardWidgetID,
			ModuleKey:           httpx.AccessLogDashboardModuleKey(),
			TitleKey:            "dashboard.widget.accessLogRequestAttention.title",
			Title:               r.mustLookupCoreDisplay("dashboard.widget.accessLogRequestAttention.title"),
			DescriptionKey:      "dashboard.widget.accessLogRequestAttention.description",
			Description:         r.mustLookupCoreDisplay("dashboard.widget.accessLogRequestAttention.description"),
			Type:                dashboard.WidgetTypeAlertList,
			Size:                dashboard.WidgetSizeMedium,
			Category:            dashboard.WidgetCategoryOperation,
			Priority:            dashboard.WidgetPriorityWarning,
			Order:               httpx.AccessLogDashboardWidgetOrder,
			RouteLocation:       httpx.AccessLogDashboardRouteLocation(),
			Action:              r.dashboardDetailsAction(httpx.AccessLogDashboardRouteLocation()),
			RequiredPermissions: []string{httpx.AccessLogReadPermission},
			Loader: dashboard.WidgetLoaderFunc(func(ctx context.Context, _ dashboard.WidgetRequest) (dashboard.WidgetPayload, error) {
				return httpx.LoadAccessLogRequestAttentionPayload(ctx, repo)
			}),
		}); err != nil {
			return fmt.Errorf("register access-log dashboard widget: %w", err)
		}
	}
	return nil
}

func (r *Runtime) registerCoreAppLogDashboard() error {
	return nil
}

func (r *Runtime) dashboardDetailsAction(route string) dashboard.WidgetAction {
	return dashboard.WidgetAction{
		LabelKey: "dashboard.actions.details",
		Label:    r.mustLookupCoreDisplay("dashboard.actions.details"),
		Route:    route,
	}
}

func (r *Runtime) mustLookupCoreDisplay(key string) string {
	if r == nil || r.i18n == nil {
		panic("core display localization requires i18n service")
	}
	if len(r.i18n.RegisteredMessageResources(i18n.LocaleTag(r.i18n.DefaultLocale()), i18n.MessageKey(key))) == 0 {
		panic("core display localization key missing: " + key)
	}

	return r.i18n.Lookup(i18n.LookupRequest{
		Locale: i18n.LocaleTag(r.i18n.DefaultLocale()),
		Key:    i18n.MessageKey(key),
	})
}

// moduleRuntimeStatusDescriptionKey 返回模块运行状态对应的本地化描述键。
//
// @returns 运行状态对应的本地化键；未知状态返回 `dashboard.widget.moduleRuntimeHealth.runtimeStatus.unknown`。
func moduleRuntimeStatusDescriptionKey(status generated.ModuleRuntimeItemRuntimeStatus) string {
	switch status {
	case generated.ModuleRuntimeItemRuntimeStatusRegistered:
		return "dashboard.widget.moduleRuntimeHealth.runtimeStatus.registered"
	case generated.ModuleRuntimeItemRuntimeStatusDisabled:
		return "dashboard.widget.moduleRuntimeHealth.runtimeStatus.disabled"
	case generated.ModuleRuntimeItemRuntimeStatusDegraded:
		return "dashboard.widget.moduleRuntimeHealth.runtimeStatus.degraded"
	default:
		return "dashboard.widget.moduleRuntimeHealth.runtimeStatus.unknown"
	}
}

func (r *Runtime) registerDashboardWithAuth(
	authService moduleapi.AuthService,
	authorizer moduleapi.Authorizer,
) error {
	if err := dashboard.Register(
		dashboard.Registration{
			I18n:     r.i18n,
			Config:   r.config,
			Registry: r.dashboardRegistry,
			Logger:   r.injectedAppLogger(),
			ModuleRuntimeSummary: func() generated.ModuleRuntimeSummary {
				return moduleruntime.BuildSnapshot(r.config, r.moduleRuntimeSpecs()).Summary
			},
		},
		r.server.Engine().Group("/api"),
		authService,
		authorizer,
	); err != nil {
		return fmt.Errorf("register dashboard routes: %w", err)
	}

	return nil
}

func (r *Runtime) moduleRuntimeSpecs() []module.Spec {
	ordered, err := moduleregistry.OrderedModuleSpecs()
	if err != nil {
		r.appLogger().Warn(context.Background(), "module runtime spec ordering failed",
			logger.StringField(logger.FieldOperation, "module_runtime_specs"),
			logger.ErrorField(err),
		)
		return moduleregistry.ModuleSpecs()
	}

	return ordered
}
