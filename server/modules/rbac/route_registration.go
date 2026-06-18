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
		Title:      "",
		TitleKey:   rbaccontract.AccessControlMenuTitle.String(),
		Path:       "/access-control",
		Icon:       "secured",
		Order:      accessControlMenuOrderRoot,
		Permission: "",
		Module:     moduleName,
	})
	registry.Register(menu.Item{
		Code:       "access-control.overview",
		Title:      "",
		TitleKey:   rbaccontract.AccessControlOverviewMenuTitle.String(),
		Path:       "/access-control/overview",
		Icon:       "dashboard",
		Order:      accessControlMenuOrderOverview,
		Permission: "",
		Module:     moduleName,
	})
	registry.Register(menu.Item{
		Code:       "role.list",
		Title:      "",
		TitleKey:   rbaccontract.RoleListMenuTitle.String(),
		Path:       "/access-control/roles",
		Icon:       "secured",
		Order:      accessControlMenuOrderRoles,
		Permission: rbaccontract.RoleReadPermission.String(),
		Module:     moduleName,
	})
	registry.Register(menu.Item{
		Code:       "permission.list",
		Title:      "",
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
			Name:           "",
			DisplayKey:     "rbac.permissionCatalog.roleRead.display",
			Description:    "",
			DescriptionKey: "rbac.permissionCatalog.roleRead.description",
			Category:       "api",
			Module:         moduleName,
		},
		{
			Code:           rbaccontract.RoleCreatePermission.String(),
			Name:           "",
			DisplayKey:     "rbac.permissionCatalog.roleCreate.display",
			Description:    "",
			DescriptionKey: "rbac.permissionCatalog.roleCreate.description",
			Category:       "api",
			Module:         moduleName,
		},
		{
			Code:           rbaccontract.RoleUpdatePermission.String(),
			Name:           "",
			DisplayKey:     "rbac.permissionCatalog.roleUpdate.display",
			Description:    "",
			DescriptionKey: "rbac.permissionCatalog.roleUpdate.description",
			Category:       "api",
			Module:         moduleName,
		},
		{
			Code:           rbaccontract.RoleStatusUpdatePermission.String(),
			Name:           "",
			DisplayKey:     "rbac.permissionCatalog.roleStatusUpdate.display",
			Description:    "",
			DescriptionKey: "rbac.permissionCatalog.roleStatusUpdate.description",
			Category:       "api",
			Module:         moduleName,
		},
		{
			Code:           rbaccontract.RoleDeletePermission.String(),
			Name:           "",
			DisplayKey:     "rbac.permissionCatalog.roleDelete.display",
			Description:    "",
			DescriptionKey: "rbac.permissionCatalog.roleDelete.description",
			Category:       "api",
			Module:         moduleName,
		},
		{
			Code:           rbaccontract.RolePermissionAssignPermission.String(),
			Name:           "",
			DisplayKey:     "rbac.permissionCatalog.rolePermissionAssign.display",
			Description:    "",
			DescriptionKey: "rbac.permissionCatalog.rolePermissionAssign.description",
			Category:       "api",
			Module:         moduleName,
		},
		{
			Code:           rbaccontract.PermissionReadPermission.String(),
			Name:           "",
			DisplayKey:     "rbac.permissionCatalog.permissionRead.display",
			Description:    "",
			DescriptionKey: "rbac.permissionCatalog.permissionRead.description",
			Category:       "api",
			Module:         moduleName,
		},
		{
			Code:           rbaccontract.UserRoleReadPermission.String(),
			Name:           "",
			DisplayKey:     "rbac.permissionCatalog.userRoleRead.display",
			Description:    "",
			DescriptionKey: "rbac.permissionCatalog.userRoleRead.description",
			Category:       "api",
			Module:         moduleName,
		},
		{
			Code:           rbaccontract.UserRoleAssignPermission.String(),
			Name:           "",
			DisplayKey:     "rbac.permissionCatalog.userRoleAssign.display",
			Description:    "",
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
