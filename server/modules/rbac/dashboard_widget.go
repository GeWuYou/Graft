package rbac

import (
	"context"
	"fmt"
	"strconv"

	"graft/server/internal/dashboard"
	"graft/server/internal/module"
	"graft/server/internal/moduleapi"
	rbaccontract "graft/server/modules/rbac/contract"
	rbacstore "graft/server/modules/rbac/store"
	usercontract "graft/server/modules/user/contract"
)

const (
	accessSummaryWidgetID    = "rbac.access-summary"
	accessSummaryWidgetOrder = 20
)

func registerDashboardWidgets(
	ctx *module.Context,
	reader managementReader,
) error {
	if ctx.DashboardRegistry == nil {
		return nil
	}

	if err := ctx.DashboardRegistry.Register(dashboard.WidgetDefinition{
		ID:             accessSummaryWidgetID,
		ModuleKey:      moduleID,
		TitleKey:       rbaccontract.AccessSummaryDashboardTitle.String(),
		Title:          "Access Control Summary",
		DescriptionKey: rbaccontract.AccessSummaryDashboardDescription.String(),
		Description:    "Review managed users, roles, and permissions.",
		Type:           dashboard.WidgetTypeStatGroup,
		Size:           dashboard.WidgetSizeMedium,
		Order:          accessSummaryWidgetOrder,
		RouteLocation:  "/access-control/overview",
		RequiredPermissions: []string{
			usercontract.UserReadPermission.String(),
			rbaccontract.RoleReadPermission.String(),
			rbaccontract.PermissionReadPermission.String(),
		},
		Loader: dashboard.WidgetLoaderFunc(func(loadCtx context.Context, _ dashboard.WidgetRequest) (dashboard.WidgetPayload, error) {
			return loadAccessSummaryWidget(loadCtx, reader)
		}),
	}); err != nil {
		return fmt.Errorf("register rbac access summary dashboard widget: %w", err)
	}

	return nil
}

func loadAccessSummaryWidget(ctx context.Context, reader managementReader) (dashboard.WidgetPayload, error) {
	userCount, err := countUsers(ctx, reader.users)
	if err != nil {
		return nil, fmt.Errorf("count users: %w", err)
	}
	roleCount, err := countRoles(ctx, reader)
	if err != nil {
		return nil, err
	}
	permissionCount, err := countPermissions(ctx, reader)
	if err != nil {
		return nil, err
	}

	return dashboard.WidgetPayload{
		"items": []map[string]any{
			{
				"key":            "users",
				"label_key":      rbaccontract.AccessSummaryUsersStat.String(),
				"label":          "Users",
				"value":          strconv.Itoa(userCount),
				"route_location": "/access-control/users",
				"tone":           "normal",
			},
			{
				"key":            "roles",
				"label_key":      rbaccontract.AccessSummaryRolesStat.String(),
				"label":          "Roles",
				"value":          strconv.Itoa(roleCount),
				"route_location": "/access-control/roles",
				"tone":           "normal",
			},
			{
				"key":            "permissions",
				"label_key":      rbaccontract.AccessSummaryPermissionsStat.String(),
				"label":          "Permissions",
				"value":          strconv.Itoa(permissionCount),
				"route_location": "/access-control/permissions",
				"tone":           "normal",
			},
		},
	}, nil
}

func countUsers(ctx context.Context, users moduleapi.UserService) (int, error) {
	if users == nil {
		return 0, fmt.Errorf("user service is unavailable")
	}
	return users.CountUsers(ctx)
}

func countRoles(ctx context.Context, reader managementReader) (int, error) {
	roles, err := reader.ListRoles(ctx, rbacstore.RoleFilter{})
	if err != nil {
		return 0, fmt.Errorf("list roles: %w", err)
	}
	return len(roles), nil
}

func countPermissions(ctx context.Context, reader managementReader) (int, error) {
	permissions, err := reader.ListPermissions(ctx, rbacstore.PermissionFilter{})
	if err != nil {
		return 0, fmt.Errorf("list permissions: %w", err)
	}
	return len(permissions), nil
}
