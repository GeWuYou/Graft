// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package rbac

import (
	"github.com/gin-gonic/gin"

	rbacopenapi "graft/server/internal/contract/openapi/rbac"
	"graft/server/internal/httpx"
	"graft/server/internal/menu"
	"graft/server/internal/module"
	"graft/server/internal/permission"
	rbaccontract "graft/server/modules/rbac/contract"
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

const (
	accessControlMenuOrderRoot        = 0
	accessControlMenuOrderOverview    = 1
	accessControlMenuOrderUsers       = 2
	accessControlMenuOrderRoles       = 3
	accessControlMenuOrderPermissions = 4
)

func registerRBACPermissions(registry *permission.Registry, moduleName string) {
	for _, item := range rbacPermissionItems(moduleName) {
		registry.Register(item)
	}
}

func registerRBACMenu(registry *menu.Registry, moduleName string) {
	registry.Register(menu.Item{
		Code:       "access-control.root",
		Title:      "访问控制",
		TitleKey:   rbaccontract.AccessControlMenuTitle.String(),
		Path:       "/access-control",
		Icon:       "secured",
		Order:      accessControlMenuOrderRoot,
		Permission: "",
		Module:     moduleName,
	})
	registry.Register(menu.Item{
		Code:       "access-control.overview",
		Title:      "概览",
		TitleKey:   rbaccontract.AccessControlOverviewMenuTitle.String(),
		Path:       "/access-control/overview",
		Icon:       "dashboard",
		Order:      accessControlMenuOrderOverview,
		Permission: "",
		Module:     moduleName,
	})
	registry.Register(menu.Item{
		Code:       "role.list",
		Title:      "角色管理",
		TitleKey:   rbaccontract.RoleListMenuTitle.String(),
		Path:       "/access-control/roles",
		Icon:       "secured",
		Order:      accessControlMenuOrderRoles,
		Permission: rbaccontract.RoleReadPermission.String(),
		Module:     moduleName,
	})
	registry.Register(menu.Item{
		Code:       "permission.list",
		Title:      "权限管理",
		TitleKey:   rbaccontract.PermissionListMenuTitle.String(),
		Path:       "/access-control/permissions",
		Icon:       "lock-on",
		Order:      accessControlMenuOrderPermissions,
		Permission: rbaccontract.PermissionReadPermission.String(),
		Module:     moduleName,
	})
}

func rbacPermissionItems(moduleName string) []permission.Item {
	return []permission.Item{
		{
			Code:           rbaccontract.RoleReadPermission.String(),
			Name:           "Read Roles",
			DisplayKey:     "rbac.permissionCatalog.roleRead.display",
			Description:    "Allows reading role management data.",
			DescriptionKey: "rbac.permissionCatalog.roleRead.description",
			Category:       "api",
			Module:         moduleName,
		},
		{
			Code:           rbaccontract.RoleCreatePermission.String(),
			Name:           "Create Roles",
			DisplayKey:     "rbac.permissionCatalog.roleCreate.display",
			Description:    "Allows creating role-management data.",
			DescriptionKey: "rbac.permissionCatalog.roleCreate.description",
			Category:       "api",
			Module:         moduleName,
		},
		{
			Code:           rbaccontract.RoleUpdatePermission.String(),
			Name:           "Update Roles",
			DisplayKey:     "rbac.permissionCatalog.roleUpdate.display",
			Description:    "Allows updating role-management data.",
			DescriptionKey: "rbac.permissionCatalog.roleUpdate.description",
			Category:       "api",
			Module:         moduleName,
		},
		{
			Code:           rbaccontract.RoleStatusUpdatePermission.String(),
			Name:           "Update Role Status",
			DisplayKey:     "rbac.permissionCatalog.roleStatusUpdate.display",
			Description:    "Allows changing role lifecycle status.",
			DescriptionKey: "rbac.permissionCatalog.roleStatusUpdate.description",
			Category:       "api",
			Module:         moduleName,
		},
		{
			Code:           rbaccontract.RoleDeletePermission.String(),
			Name:           "Delete Roles",
			DisplayKey:     "rbac.permissionCatalog.roleDelete.display",
			Description:    "Allows deleting disabled roles without bindings.",
			DescriptionKey: "rbac.permissionCatalog.roleDelete.description",
			Category:       "api",
			Module:         moduleName,
		},
		{
			Code:           rbaccontract.RolePermissionAssignPermission.String(),
			Name:           "Assign Role Permissions",
			DisplayKey:     "rbac.permissionCatalog.rolePermissionAssign.display",
			Description:    "Allows updating role-permission bindings.",
			DescriptionKey: "rbac.permissionCatalog.rolePermissionAssign.description",
			Category:       "api",
			Module:         moduleName,
		},
		{
			Code:           rbaccontract.PermissionReadPermission.String(),
			Name:           "Read Permissions",
			DisplayKey:     "rbac.permissionCatalog.permissionRead.display",
			Description:    "Allows reading permission management data.",
			DescriptionKey: "rbac.permissionCatalog.permissionRead.description",
			Category:       "api",
			Module:         moduleName,
		},
		{
			Code:           rbaccontract.UserRoleReadPermission.String(),
			Name:           "Read User Roles",
			DisplayKey:     "rbac.permissionCatalog.userRoleRead.display",
			Description:    "Allows reading user-role binding snapshots.",
			DescriptionKey: "rbac.permissionCatalog.userRoleRead.description",
			Category:       "api",
			Module:         moduleName,
		},
		{
			Code:           rbaccontract.UserRoleAssignPermission.String(),
			Name:           "Assign User Roles",
			DisplayKey:     "rbac.permissionCatalog.userRoleAssign.display",
			Description:    "Allows updating user-role bindings.",
			DescriptionKey: "rbac.permissionCatalog.userRoleAssign.description",
			Category:       "api",
			Module:         moduleName,
		},
	}
}

func registerManagementRoutes(
	ctx *module.Context,
	moduleName string,
	reader readManagementService,
	writer writeManagementService,
	guards managementGuards,
) {
	registerRoleRoutes(ctx, moduleName, reader, writer, guards)
	registerPermissionRoutes(ctx, moduleName, reader, guards.permissionRead)
	registerUserRoleRoutes(ctx, moduleName, reader, writer, guards)
}

func registerRoleRoutes(
	ctx *module.Context,
	moduleName string,
	reader readManagementService,
	writer writeManagementService,
	guards managementGuards,
) {
	group := ctx.Router.Group(rbaccontract.RolesGroup)
	group.Use(httpx.RequestIDMiddleware())
	group.GET(rbaccontract.RoleCollection, guards.roleRead, handleListRoles(ctx, moduleName, reader))
	group.GET(rbaccontract.RoleDetailRoute, guards.roleRead, handleGetRole(ctx, moduleName, reader))
	group.GET(rbaccontract.RolePermissionBindingRoute, guards.permissionRead, handleListRolePermissionBindings(ctx, moduleName, reader))
	registerRoleWriteRoutes(group, ctx, moduleName, writer, guards)
}

func registerPermissionRoutes(
	ctx *module.Context,
	moduleName string,
	reader readManagementService,
	authenticated gin.HandlerFunc,
) {
	group := ctx.Router.Group(rbaccontract.PermissionsGroup)
	group.Use(httpx.RequestIDMiddleware())
	group.GET(rbaccontract.PermissionCollection, authenticated, handleListPermissions(ctx, moduleName, reader))
	group.GET(rbaccontract.PermissionDetailRoute, authenticated, handleGetPermission(ctx, moduleName, reader))
}

func registerUserRoleRoutes(
	ctx *module.Context,
	moduleName string,
	reader readManagementService,
	writer writeManagementService,
	guards managementGuards,
) {
	group := ctx.Router.Group(rbaccontract.UsersGroup)
	group.Use(httpx.RequestIDMiddleware())
	group.GET(rbaccontract.UserRoleBindingRoute, guards.userRoleRead, handleListUserRoleBindings(ctx, moduleName, reader))
	group.POST(rbaccontract.UserRoleReplaceRoute, guards.userRoleAssign, func(ginCtx *gin.Context) { handleReplaceUserRolesRoute(ginCtx, ctx, moduleName, writer) })
	group.POST(rbaccontract.UserRoleAddRoute, guards.userRoleAssign, func(ginCtx *gin.Context) { handleAddUserRolesRoute(ginCtx, ctx, moduleName, writer) })
	group.POST(rbaccontract.UserRoleRemoveRoute, guards.userRoleAssign, func(ginCtx *gin.Context) { handleRemoveUserRolesRoute(ginCtx, ctx, moduleName, writer) })
	group.POST(rbaccontract.BatchUserRoleReplaceRoute, guards.userRoleAssign, func(ginCtx *gin.Context) { handleBatchReplaceUserRolesRoute(ginCtx, ctx, moduleName, writer) })
	group.POST(rbaccontract.BatchUserRoleAddRoute, guards.userRoleAssign, func(ginCtx *gin.Context) { handleBatchAddUserRolesRoute(ginCtx, ctx, moduleName, writer) })
	group.POST(rbaccontract.BatchUserRoleRemoveRoute, guards.userRoleAssign, func(ginCtx *gin.Context) { handleBatchRemoveUserRolesRoute(ginCtx, ctx, moduleName, writer) })
}

var _ rbacopenapi.ReadServerInterface = rbacReadGeneratedHandler{}
var _ rbacopenapi.UserRoleServerInterface = rbacUserRoleGeneratedHandler{}
var _ rbacopenapi.WriteServerInterface = rbacWriteGeneratedHandler{}
