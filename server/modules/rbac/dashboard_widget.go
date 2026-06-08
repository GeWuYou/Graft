package rbac

import (
	"fmt"

	"graft/server/internal/dashboard"
	"graft/server/internal/module"
	rbaccontract "graft/server/modules/rbac/contract"
	usercontract "graft/server/modules/user/contract"
)

const (
	accessControlOverviewQuickLinkID       = "rbac.access-control.overview"
	accessControlUsersQuickLinkID          = "rbac.access-control.users"
	accessControlRolesQuickLinkID          = "rbac.access-control.roles"
	accessControlPermissionsQuickLinkID    = "rbac.access-control.permissions"
	accessControlOverviewQuickLinkOrder    = 20
	accessControlUsersQuickLinkOrder       = 30
	accessControlRolesQuickLinkOrder       = 40
	accessControlPermissionsQuickLinkOrder = 50
)

func registerDashboardWidgets(ctx *module.Context, _ managementReader) error {
	if ctx.DashboardRegistry == nil {
		return nil
	}

	for _, link := range accessControlQuickLinks() {
		if err := ctx.DashboardRegistry.RegisterQuickLink(link); err != nil {
			return fmt.Errorf("register rbac dashboard quick link: %w", err)
		}
	}

	return nil
}

func accessControlQuickLinks() []dashboard.QuickLinkDefinition {
	return []dashboard.QuickLinkDefinition{
		{
			ID:            accessControlOverviewQuickLinkID,
			ModuleKey:     moduleID,
			TitleKey:      rbaccontract.AccessControlOverviewMenuTitle.String(),
			Title:         "Access Control Overview",
			Icon:          "dashboard",
			RouteLocation: "/access-control/overview",
			Order:         accessControlOverviewQuickLinkOrder,
		},
		{
			ID:                  accessControlUsersQuickLinkID,
			ModuleKey:           moduleID,
			TitleKey:            usercontract.UserListMenuTitle.String(),
			Title:               "User Management",
			Icon:                "user",
			RouteLocation:       "/access-control/users",
			RequiredPermissions: []string{usercontract.UserReadPermission.String()},
			Order:               accessControlUsersQuickLinkOrder,
		},
		{
			ID:                  accessControlRolesQuickLinkID,
			ModuleKey:           moduleID,
			TitleKey:            rbaccontract.RoleListMenuTitle.String(),
			Title:               "Role Management",
			Icon:                "secured",
			RouteLocation:       "/access-control/roles",
			RequiredPermissions: []string{rbaccontract.RoleReadPermission.String()},
			Order:               accessControlRolesQuickLinkOrder,
		},
		{
			ID:                  accessControlPermissionsQuickLinkID,
			ModuleKey:           moduleID,
			TitleKey:            rbaccontract.PermissionListMenuTitle.String(),
			Title:               "Permission Management",
			Icon:                "lock-on",
			RouteLocation:       "/access-control/permissions",
			RequiredPermissions: []string{rbaccontract.PermissionReadPermission.String()},
			Order:               accessControlPermissionsQuickLinkOrder,
		},
	}
}
