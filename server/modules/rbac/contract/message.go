package contract

// MenuMessageKey identifies a stable rbac module menu title message key.
type MenuMessageKey string

// String returns the canonical menu message key value.
func (k MenuMessageKey) String() string {
	return string(k)
}

const (
	// AccessControlMenuTitle identifies the localized title for the access-control root menu.
	AccessControlMenuTitle MenuMessageKey = "menu.access_control.title"
	// RoleListMenuTitle identifies the localized title for the role list menu.
	RoleListMenuTitle MenuMessageKey = "menu.access_control.roles.title"
	// PermissionListMenuTitle identifies the localized title for the permission list menu.
	PermissionListMenuTitle MenuMessageKey = "menu.access_control.permissions.title"
	// AccessControlOverviewMenuTitle identifies the localized title for the access-control overview menu.
	AccessControlOverviewMenuTitle MenuMessageKey = "menu.access_control.overview.title"
	// AccessSummaryDashboardTitle identifies the localized title for the RBAC dashboard summary widget.
	AccessSummaryDashboardTitle MenuMessageKey = "rbac.dashboard.accessSummary.title"
	// AccessSummaryDashboardDescription identifies the localized description for the RBAC dashboard summary widget.
	AccessSummaryDashboardDescription MenuMessageKey = "rbac.dashboard.accessSummary.description"
	// AccessSummaryUsersStat identifies the localized users stat label for the RBAC dashboard summary widget.
	AccessSummaryUsersStat MenuMessageKey = "rbac.dashboard.accessSummary.users"
	// AccessSummaryRolesStat identifies the localized roles stat label for the RBAC dashboard summary widget.
	AccessSummaryRolesStat MenuMessageKey = "rbac.dashboard.accessSummary.roles"
	// AccessSummaryPermissionsStat identifies the localized permissions stat label for the RBAC dashboard summary widget.
	AccessSummaryPermissionsStat MenuMessageKey = "rbac.dashboard.accessSummary.permissions"
)
