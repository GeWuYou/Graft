package rbac

import (
	"github.com/gin-gonic/gin"

	rbacopenapi "graft/server/internal/contract/openapi/rbac"
	"graft/server/internal/httpx"
	"graft/server/internal/menu"
	"graft/server/internal/module"
	"graft/server/internal/moduleapi"
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
	accessControlMenuOrderRoles       = 4
	accessControlMenuOrderPermissions = 5
)

// registerRBACPermissions 注册 RBAC 模块拥有的权限定义，供路由鉴权和菜单可见性共同消费。
func registerRBACPermissions(registry *permission.Registry, moduleName string) {
	for _, item := range rbacPermissionItems(moduleName) {
		registry.Register(item)
	}
}

// registerRBACMenu 注册角色和权限管理菜单；菜单声明只描述可见入口，实际访问仍由对应权限守卫决定。
func registerRBACMenu(registry *menu.Registry, moduleName string) {
	registry.Register(menu.Item{
		Code:            "role.list",
		ParentCode:      "domain.security",
		Kind:            menu.NodeKindEntry,
		Title:           "",
		TitleKey:        rbaccontract.RoleListMenuTitle.String(),
		SectionKey:      menu.AccessControlSectionKey,
		SectionTitleKey: menu.AccessControlSectionTitleKey,
		Path:            rbaccontract.RoleListMenuPath,
		Icon:            "role-groups",
		Order:           accessControlMenuOrderRoles,
		Permission:      rbaccontract.RoleReadPermission.String(),
		Module:          moduleName,
	})
	registry.Register(menu.Item{
		Code:            "permission.list",
		ParentCode:      "domain.security",
		Kind:            menu.NodeKindEntry,
		Title:           "",
		TitleKey:        rbaccontract.PermissionListMenuTitle.String(),
		SectionKey:      menu.AccessControlSectionKey,
		SectionTitleKey: menu.AccessControlSectionTitleKey,
		Path:            rbaccontract.PermissionListMenuPath,
		Icon:            "access-policy",
		Order:           accessControlMenuOrderPermissions,
		Permission:      rbaccontract.PermissionReadPermission.String(),
		Module:          moduleName,
	})
}

func rbacPermissionItems(moduleName string) []permission.Item {
	items := []permission.Item{
		{
			Code:           rbaccontract.RoleReadPermission.String(),
			Name:           "",
			DisplayKey:     "rbac.permissionCatalog.roleRead.display",
			Description:    "",
			DescriptionKey: "rbac.permissionCatalog.roleRead.description",
			Module:         moduleName,
		},
		{
			Code:           rbaccontract.RoleCreatePermission.String(),
			Name:           "",
			DisplayKey:     "rbac.permissionCatalog.roleCreate.display",
			Description:    "",
			DescriptionKey: "rbac.permissionCatalog.roleCreate.description",
			Module:         moduleName,
		},
		{
			Code:           rbaccontract.RoleUpdatePermission.String(),
			Name:           "",
			DisplayKey:     "rbac.permissionCatalog.roleUpdate.display",
			Description:    "",
			DescriptionKey: "rbac.permissionCatalog.roleUpdate.description",
			Module:         moduleName,
		},
		{
			Code:           rbaccontract.RoleStatusUpdatePermission.String(),
			Name:           "",
			DisplayKey:     "rbac.permissionCatalog.roleStatusUpdate.display",
			Description:    "",
			DescriptionKey: "rbac.permissionCatalog.roleStatusUpdate.description",
			Module:         moduleName,
		},
		{
			Code:           rbaccontract.RoleDeletePermission.String(),
			Name:           "",
			DisplayKey:     "rbac.permissionCatalog.roleDelete.display",
			Description:    "",
			DescriptionKey: "rbac.permissionCatalog.roleDelete.description",
			Module:         moduleName,
		},
		{
			Code:           rbaccontract.RolePermissionAssignPermission.String(),
			Name:           "",
			DisplayKey:     "rbac.permissionCatalog.rolePermissionAssign.display",
			Description:    "",
			DescriptionKey: "rbac.permissionCatalog.rolePermissionAssign.description",
			Module:         moduleName,
		},
		{
			Code:           rbaccontract.PermissionReadPermission.String(),
			Name:           "",
			DisplayKey:     "rbac.permissionCatalog.permissionRead.display",
			Description:    "",
			DescriptionKey: "rbac.permissionCatalog.permissionRead.description",
			Module:         moduleName,
		},
		{
			Code:           rbaccontract.UserRoleReadPermission.String(),
			Name:           "",
			DisplayKey:     "rbac.permissionCatalog.userRoleRead.display",
			Description:    "",
			DescriptionKey: "rbac.permissionCatalog.userRoleRead.description",
			Module:         moduleName,
		},
		{
			Code:           rbaccontract.UserRoleAssignPermission.String(),
			Name:           "",
			DisplayKey:     "rbac.permissionCatalog.userRoleAssign.display",
			Description:    "",
			DescriptionKey: "rbac.permissionCatalog.userRoleAssign.description",
			Module:         moduleName,
		},
	}
	metadata := map[string]struct {
		action   string
		level    string
		category string
	}{
		"role.read":              {"read", permission.RiskLevelLow, permission.RiskCategoryRead},
		"role.create":            {"create", permission.RiskLevelHigh, permission.RiskCategorySecurity},
		"role.update":            {"update", permission.RiskLevelHigh, permission.RiskCategorySecurity},
		"role.status.update":     {"status.update", permission.RiskLevelHigh, permission.RiskCategorySecurity},
		"role.delete":            {"delete", permission.RiskLevelHigh, permission.RiskCategorySecurity},
		"role.permission.assign": {"permission.assign", permission.RiskLevelCritical, permission.RiskCategorySecurity},
		"permission.read":        {"read", permission.RiskLevelLow, permission.RiskCategoryRead},
		"user.role.read":         {"role.read", permission.RiskLevelLow, permission.RiskCategoryRead},
		"user.role.assign":       {"role.assign", permission.RiskLevelCritical, permission.RiskCategorySecurity},
	}
	for index := range items {
		item := &items[index]
		value := metadata[item.Code]
		item.Resource, item.Action, item.RiskLevel, item.RiskCategory = "rbac", value.action, value.level, value.category
	}
	return items
}

func registerManagementRoutes(
	ctx *module.Context,
	moduleName string,
	reader readManagementService,
	writer writeManagementService,
	guards managementGuards,
	savedViews moduleapi.SavedViewService,
) {
	registerRoleRoutes(ctx, moduleName, reader, writer, guards, savedViews)
	registerPermissionRoutes(ctx, moduleName, reader, guards.permissionRead, savedViews)
	registerUserRoleRoutes(ctx, moduleName, reader, writer, guards)
}

func registerRoleRoutes(
	ctx *module.Context,
	moduleName string,
	reader readManagementService,
	writer writeManagementService,
	guards managementGuards,
	savedViews moduleapi.SavedViewService,
) {
	group := ctx.Router.Group(rbaccontract.RolesGroup)
	group.Use(httpx.RequestIDMiddleware())
	group.GET(rbaccontract.RoleCollection, guards.roleRead, handleListRoles(ctx, moduleName, reader))
	registerRBACSavedViewRoutes(group, ctx, savedViews, guards.roleRead, roleSavedViewDefinition)
	group.GET(rbaccontract.RoleDetailRoute, guards.roleRead, handleGetRole(ctx, moduleName, reader))
	group.GET(rbaccontract.RolePermissionBindingRoute, guards.permissionRead, handleListRolePermissionBindings(ctx, moduleName, reader))
	registerRoleWriteRoutes(group, ctx, moduleName, writer, guards)
}

func registerPermissionRoutes(
	ctx *module.Context,
	moduleName string,
	reader readManagementService,
	authenticated gin.HandlerFunc,
	savedViews moduleapi.SavedViewService,
) {
	group := ctx.Router.Group(rbaccontract.PermissionsGroup)
	group.Use(httpx.RequestIDMiddleware())
	group.GET(rbaccontract.PermissionCollection, authenticated, handleListPermissions(ctx, moduleName, reader))
	registerRBACSavedViewRoutes(group, ctx, savedViews, authenticated, permissionSavedViewDefinition)
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
	group.DELETE(rbaccontract.UserRoleRemoveRoute, guards.userRoleAssign, func(ginCtx *gin.Context) { handleRemoveUserRolesRoute(ginCtx, ctx, moduleName, writer) })
	group.POST(rbaccontract.BatchUserRoleReplaceRoute, guards.userRoleAssign, func(ginCtx *gin.Context) { handleBatchReplaceUserRolesRoute(ginCtx, ctx, moduleName, writer) })
	group.POST(rbaccontract.BatchUserRoleAddRoute, guards.userRoleAssign, func(ginCtx *gin.Context) { handleBatchAddUserRolesRoute(ginCtx, ctx, moduleName, writer) })
	group.DELETE(rbaccontract.BatchUserRoleRemoveRoute, guards.userRoleAssign, func(ginCtx *gin.Context) { handleBatchRemoveUserRolesRoute(ginCtx, ctx, moduleName, writer) })
}

var _ rbacopenapi.ReadServerInterface = rbacReadGeneratedHandler{}
var _ rbacopenapi.UserRoleServerInterface = rbacUserRoleGeneratedHandler{}
var _ rbacopenapi.WriteServerInterface = rbacWriteGeneratedHandler{}
