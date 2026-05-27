package rbac

import (
	"github.com/gin-gonic/gin"

	rbacopenapi "graft/server/internal/contract/openapi/rbac"
	"graft/server/internal/httpx"
	"graft/server/internal/menu"
	"graft/server/internal/permission"
	"graft/server/internal/plugin"
	rbaccontract "graft/server/plugins/rbac/contract"
)

type managementGuards struct {
	roleRead             gin.HandlerFunc
	permissionRead       gin.HandlerFunc
	roleCreate           gin.HandlerFunc
	roleUpdate           gin.HandlerFunc
	roleStatus           gin.HandlerFunc
	roleDelete           gin.HandlerFunc
	rolePermissionAssign gin.HandlerFunc
	userRoleRead         gin.HandlerFunc
	userRoleAssign       gin.HandlerFunc
}

func registerRBACPermissions(registry *permission.Registry, pluginName string) {
	for _, item := range rbacPermissionItems(pluginName) {
		registry.Register(item)
	}
}

func registerRBACMenu(registry *menu.Registry, pluginName string) {
	registry.Register(menu.Item{
		Code:       "access-control.root",
		Title:      "访问控制",
		TitleKey:   rbaccontract.AccessControlMenuTitle.String(),
		Path:       "/access-control",
		Icon:       "secured",
		Permission: "",
		Plugin:     pluginName,
	})
	registry.Register(menu.Item{
		Code:       "access-control.overview",
		Title:      "概览",
		TitleKey:   rbaccontract.AccessControlOverviewMenuTitle.String(),
		Path:       "/access-control/overview",
		Icon:       "dashboard",
		Permission: "",
		Plugin:     pluginName,
	})
	registry.Register(menu.Item{
		Code:       "role.list",
		Title:      "角色管理",
		TitleKey:   rbaccontract.RoleListMenuTitle.String(),
		Path:       "/access-control/roles",
		Icon:       "secured",
		Permission: rbaccontract.RoleReadPermission.String(),
		Plugin:     pluginName,
	})
	registry.Register(menu.Item{
		Code:       "permission.list",
		Title:      "权限管理",
		TitleKey:   rbaccontract.PermissionListMenuTitle.String(),
		Path:       "/access-control/permissions",
		Icon:       "lock-on",
		Permission: rbaccontract.PermissionReadPermission.String(),
		Plugin:     pluginName,
	})
}

func rbacPermissionItems(pluginName string) []permission.Item {
	return []permission.Item{
		{
			Code:        rbaccontract.RoleReadPermission.String(),
			Name:        "Read Roles",
			Description: "Allows reading role management data.",
			Category:    "api",
			Plugin:      pluginName,
		},
		{
			Code:        rbaccontract.RoleCreatePermission.String(),
			Name:        "Create Roles",
			Description: "Allows creating role-management data.",
			Category:    "api",
			Plugin:      pluginName,
		},
		{
			Code:        rbaccontract.RoleUpdatePermission.String(),
			Name:        "Update Roles",
			Description: "Allows updating role-management data.",
			Category:    "api",
			Plugin:      pluginName,
		},
		{
			Code:        rbaccontract.RoleStatusUpdatePermission.String(),
			Name:        "Update Role Status",
			Description: "Allows changing role lifecycle status.",
			Category:    "api",
			Plugin:      pluginName,
		},
		{
			Code:        rbaccontract.RoleDeletePermission.String(),
			Name:        "Delete Roles",
			Description: "Allows deleting disabled roles without bindings.",
			Category:    "api",
			Plugin:      pluginName,
		},
		{
			Code:        rbaccontract.RolePermissionAssignPermission.String(),
			Name:        "Assign Role Permissions",
			Description: "Allows updating role-permission bindings.",
			Category:    "api",
			Plugin:      pluginName,
		},
		{
			Code:        rbaccontract.PermissionReadPermission.String(),
			Name:        "Read Permissions",
			Description: "Allows reading permission management data.",
			Category:    "api",
			Plugin:      pluginName,
		},
		{
			Code:        rbaccontract.UserRoleReadPermission.String(),
			Name:        "Read User Roles",
			Description: "Allows reading user-role binding snapshots.",
			Category:    "api",
			Plugin:      pluginName,
		},
		{
			Code:        rbaccontract.UserRoleAssignPermission.String(),
			Name:        "Assign User Roles",
			Description: "Allows updating user-role bindings.",
			Category:    "api",
			Plugin:      pluginName,
		},
	}
}

func registerManagementRoutes(
	ctx *plugin.Context,
	pluginName string,
	reader readManagementService,
	writer writeManagementService,
	guards managementGuards,
) {
	registerRoleRoutes(ctx, pluginName, reader, writer, guards)
	registerPermissionRoutes(ctx, pluginName, reader, guards.permissionRead)
	registerUserRoleRoutes(ctx, pluginName, reader, writer, guards)
}

func registerRoleRoutes(
	ctx *plugin.Context,
	pluginName string,
	reader readManagementService,
	writer writeManagementService,
	guards managementGuards,
) {
	group := ctx.Router.Group(rbaccontract.RolesGroup)
	group.Use(httpx.RequestIDMiddleware())
	group.GET(rbaccontract.RoleCollection, guards.roleRead, handleListRoles(ctx, pluginName, reader))
	group.GET(rbaccontract.RoleDetailRoute, guards.roleRead, handleGetRole(ctx, pluginName, reader))
	group.GET(rbaccontract.RolePermissionBindingRoute, guards.permissionRead, handleListRolePermissionBindings(ctx, pluginName, reader))
	registerRoleWriteRoutes(group, ctx, pluginName, writer, guards)
}

func registerPermissionRoutes(
	ctx *plugin.Context,
	pluginName string,
	reader readManagementService,
	authenticated gin.HandlerFunc,
) {
	group := ctx.Router.Group(rbaccontract.PermissionsGroup)
	group.Use(httpx.RequestIDMiddleware())
	group.GET(rbaccontract.PermissionCollection, authenticated, handleListPermissions(ctx, pluginName, reader))
	group.GET(rbaccontract.PermissionDetailRoute, authenticated, handleGetPermission(ctx, pluginName, reader))
}

func registerUserRoleRoutes(
	ctx *plugin.Context,
	pluginName string,
	reader readManagementService,
	writer writeManagementService,
	guards managementGuards,
) {
	group := ctx.Router.Group(rbaccontract.UsersGroup)
	group.Use(httpx.RequestIDMiddleware())
	group.GET(rbaccontract.UserRoleBindingRoute, guards.userRoleRead, handleListUserRoleBindings(ctx, pluginName, reader))
	group.POST(rbaccontract.UserRoleReplaceRoute, guards.userRoleAssign, func(ginCtx *gin.Context) { handleReplaceUserRolesRoute(ginCtx, ctx, pluginName, writer) })
	group.POST(rbaccontract.UserRoleAddRoute, guards.userRoleAssign, func(ginCtx *gin.Context) { handleAddUserRolesRoute(ginCtx, ctx, pluginName, writer) })
	group.POST(rbaccontract.UserRoleRemoveRoute, guards.userRoleAssign, func(ginCtx *gin.Context) { handleRemoveUserRolesRoute(ginCtx, ctx, pluginName, writer) })
	group.POST(rbaccontract.BatchUserRoleReplaceRoute, guards.userRoleAssign, func(ginCtx *gin.Context) { handleBatchReplaceUserRolesRoute(ginCtx, ctx, pluginName, writer) })
	group.POST(rbaccontract.BatchUserRoleAddRoute, guards.userRoleAssign, func(ginCtx *gin.Context) { handleBatchAddUserRolesRoute(ginCtx, ctx, pluginName, writer) })
	group.POST(rbaccontract.BatchUserRoleRemoveRoute, guards.userRoleAssign, func(ginCtx *gin.Context) { handleBatchRemoveUserRolesRoute(ginCtx, ctx, pluginName, writer) })
}

var _ rbacopenapi.ReadServerInterface = rbacReadGeneratedHandler{}
var _ rbacopenapi.UserRoleServerInterface = rbacUserRoleGeneratedHandler{}
var _ rbacopenapi.WriteServerInterface = rbacWriteGeneratedHandler{}
