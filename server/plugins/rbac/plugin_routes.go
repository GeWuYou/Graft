package rbac

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	messagecontract "graft/server/internal/contract/message"
	"graft/server/internal/httpx"
	"graft/server/internal/menu"
	"graft/server/internal/permission"
	"graft/server/internal/plugin"
	rbaccontract "graft/server/plugins/rbac/contract"
)

type roleListResponse struct {
	Items []roleListItem `json:"items"`
}

type roleListItem struct {
	ID          uint64  `json:"id"`
	Name        string  `json:"name"`
	Display     string  `json:"display"`
	Description *string `json:"description,omitempty"`
	Builtin     bool    `json:"builtin"`
}

type permissionListResponse struct {
	Items []permissionListItem `json:"items"`
}

type permissionListItem struct {
	ID          uint64  `json:"id"`
	Code        string  `json:"code"`
	Display     string  `json:"display"`
	Description *string `json:"description,omitempty"`
	Category    string  `json:"category"`
}

type managementGuards struct {
	roleRead             gin.HandlerFunc
	permissionRead       gin.HandlerFunc
	roleCreate           gin.HandlerFunc
	roleUpdate           gin.HandlerFunc
	rolePermissionAssign gin.HandlerFunc
	userRoleAssign       gin.HandlerFunc
}

func registerRBACPermissions(registry *permission.Registry, pluginName string) {
	for _, item := range rbacPermissionItems(pluginName) {
		registry.Register(item)
	}
}

func registerRBACMenu(registry *menu.Registry, pluginName string) {
	registry.Register(menu.Item{
		Code:       "role.list",
		Title:      "角色管理",
		Path:       rbaccontract.RolesGroup,
		Icon:       "secured",
		Permission: rbaccontract.RoleReadPermission.String(),
		Plugin:     pluginName,
	})
}

func rbacPermissionItems(pluginName string) []permission.Item {
	return []permission.Item{
		{
			Code:        rbaccontract.RoleReadPermission.String(),
			Name:        "Read Roles",
			Description: "Allows reading role management data.",
			Plugin:      pluginName,
		},
		{
			Code:        rbaccontract.RoleCreatePermission.String(),
			Name:        "Create Roles",
			Description: "Allows creating role-management data.",
			Plugin:      pluginName,
		},
		{
			Code:        rbaccontract.RoleUpdatePermission.String(),
			Name:        "Update Roles",
			Description: "Allows updating role-management data.",
			Plugin:      pluginName,
		},
		{
			Code:        rbaccontract.RolePermissionAssignPermission.String(),
			Name:        "Assign Role Permissions",
			Description: "Allows updating role-permission bindings.",
			Plugin:      pluginName,
		},
		{
			Code:        rbaccontract.PermissionReadPermission.String(),
			Name:        "Read Permissions",
			Description: "Allows reading permission management data.",
			Plugin:      pluginName,
		},
		{
			Code:        rbaccontract.UserRoleAssignPermission.String(),
			Name:        "Assign User Roles",
			Description: "Allows updating user-role bindings.",
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
	registerUserRoleRoutes(ctx, pluginName, writer, guards.userRoleAssign)
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
	group.GET(rbaccontract.RoleCollection, guards.roleRead, func(ginCtx *gin.Context) {
		roles, err := reader.ListRoles(ginCtx.Request.Context())
		if err != nil {
			ctx.Logger.Error("list roles failed",
				zap.String("plugin", pluginName),
				zap.Error(err),
			)
			httpx.AbortLocalizedError(ginCtx, ctx.I18n, http.StatusInternalServerError, messagecontract.CommonInternalError.String(), nil)
			return
		}

		items := make([]roleListItem, 0, len(roles))
		for _, role := range roles {
			items = append(items, roleListItem{
				ID:          role.ID,
				Name:        role.Name,
				Display:     role.Display,
				Description: role.Description,
				Builtin:     role.Builtin,
			})
		}

		httpx.WriteSuccess(ginCtx, http.StatusOK, roleListResponse{Items: items})
	})
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
	group.GET(rbaccontract.PermissionCollection, authenticated, func(ginCtx *gin.Context) {
		permissions, err := reader.ListPermissions(ginCtx.Request.Context())
		if err != nil {
			ctx.Logger.Error("list permissions failed",
				zap.String("plugin", pluginName),
				zap.Error(err),
			)
			httpx.AbortLocalizedError(ginCtx, ctx.I18n, http.StatusInternalServerError, messagecontract.CommonInternalError.String(), nil)
			return
		}

		items := make([]permissionListItem, 0, len(permissions))
		for _, item := range permissions {
			items = append(items, permissionListItem{
				ID:          item.ID,
				Code:        item.Code,
				Display:     item.Display,
				Description: item.Description,
				Category:    item.Category,
			})
		}

		httpx.WriteSuccess(ginCtx, http.StatusOK, permissionListResponse{Items: items})
	})
}
