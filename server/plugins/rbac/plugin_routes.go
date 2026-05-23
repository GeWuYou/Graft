package rbac

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	messagecontract "graft/server/internal/contract/message"
	"graft/server/internal/httpx"
	"graft/server/internal/menu"
	"graft/server/internal/permission"
	"graft/server/internal/plugin"
	rbaccontract "graft/server/plugins/rbac/contract"
	rbacstore "graft/server/plugins/rbac/store"
)

type roleListResponse struct {
	Items []roleListItem `json:"items"`
}

type roleListItem struct {
	ID              uint64  `json:"id"`
	Name            string  `json:"name"`
	Display         string  `json:"display"`
	Description     *string `json:"description,omitempty"`
	Builtin         bool    `json:"builtin"`
	UpdatedAt       string  `json:"updated_at"`
	PermissionCount int     `json:"permission_count"`
	UserCount       int     `json:"user_count"`
}

type rolePermissionBindingResponse struct {
	PermissionIDs []uint64 `json:"permission_ids"`
}

type userRoleBindingResponse struct {
	RoleIDs []uint64 `json:"role_ids"`
}

type permissionListResponse struct {
	Items []permissionListItem `json:"items"`
}

type permissionListItem struct {
	ID               uint64  `json:"id"`
	Code             string  `json:"code"`
	Display          string  `json:"display"`
	Description      *string `json:"description,omitempty"`
	Category         string  `json:"category"`
	CreatedAt        string  `json:"created_at"`
	UpdatedAt        string  `json:"updated_at"`
	RoleBindingCount int     `json:"role_binding_count"`
}

type managementGuards struct {
	roleRead             gin.HandlerFunc
	permissionRead       gin.HandlerFunc
	roleCreate           gin.HandlerFunc
	roleUpdate           gin.HandlerFunc
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
		Code:       "access-control.overview",
		Title:      "概览",
		TitleKey:   rbaccontract.AccessControlOverviewMenuTitle.String(),
		Path:       "/access-control/overview",
		Icon:       "secured",
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
		Icon:       "secured",
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
	group.GET(rbaccontract.RoleCollection, guards.roleRead, newManagementListHandler(
		ctx,
		pluginName,
		"list roles failed",
		func(ginCtx *gin.Context) (roleListResponse, error) {
			roles, err := reader.ListRoles(ginCtx.Request.Context())
			if err != nil {
				return roleListResponse{}, err
			}

			items := make([]roleListItem, 0, len(roles))
			for _, role := range roles {
				items = append(items, roleListItem{
					ID:              role.ID,
					Name:            role.Name,
					Display:         role.Display,
					Description:     role.Description,
					Builtin:         role.Builtin,
					UpdatedAt:       role.UpdatedAt.UTC().Format(time.RFC3339),
					PermissionCount: role.PermissionCount,
					UserCount:       role.UserCount,
				})
			}

			return roleListResponse{Items: items}, nil
		},
	))
	group.GET(rbaccontract.RolePermissionBindingRoute, guards.permissionRead, func(ginCtx *gin.Context) {
		roleID, err := parseManagementID(ginCtx.Param("id"))
		if err != nil {
			writeLocalizedContractError(ginCtx, ctx.I18n, http.StatusBadRequest, messagecontract.CommonInvalidArgument, map[string]any{
				"field": "id",
			})
			return
		}

		bindings, err := reader.ListRolePermissionBindings(ginCtx.Request.Context(), roleID)
		if err != nil {
			if errors.Is(err, rbacstore.ErrRoleNotFound) {
				writeLocalizedContractError(ginCtx, ctx.I18n, http.StatusNotFound, messagecontract.RoleNotFound, nil)
				return
			}

			ctx.Logger.Error("list role permission bindings failed",
				zap.String("plugin", pluginName),
				zap.Uint64("roleId", roleID),
				zap.Error(err),
			)
			httpx.AbortLocalizedError(ginCtx, ctx.I18n, http.StatusInternalServerError, messagecontract.CommonInternalError.String(), nil)
			return
		}

		permissionIDs := make([]uint64, 0, len(bindings))
		for _, item := range bindings {
			permissionIDs = append(permissionIDs, item.PermissionID)
		}

		httpx.WriteSuccess(ginCtx, http.StatusOK, rolePermissionBindingResponse{PermissionIDs: permissionIDs})
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
	group.GET(rbaccontract.PermissionCollection, authenticated, newManagementListHandler(
		ctx,
		pluginName,
		"list permissions failed",
		func(ginCtx *gin.Context) (permissionListResponse, error) {
			permissions, err := reader.ListPermissions(ginCtx.Request.Context())
			if err != nil {
				return permissionListResponse{}, err
			}

			items := make([]permissionListItem, 0, len(permissions))
			for _, item := range permissions {
				items = append(items, permissionListItem{
					ID:               item.ID,
					Code:             item.Code,
					Display:          item.Display,
					Description:      item.Description,
					Category:         item.Category,
					CreatedAt:        item.CreatedAt.UTC().Format(time.RFC3339),
					UpdatedAt:        item.UpdatedAt.UTC().Format(time.RFC3339),
					RoleBindingCount: item.RoleBindingCount,
				})
			}

			return permissionListResponse{Items: items}, nil
		},
	))
}

func newManagementListHandler[T any](
	ctx *plugin.Context,
	pluginName string,
	logMessage string,
	read func(ginCtx *gin.Context) (T, error),
) gin.HandlerFunc {
	return func(ginCtx *gin.Context) {
		payload, err := read(ginCtx)
		if err != nil {
			ctx.Logger.Error(logMessage,
				zap.String("plugin", pluginName),
				zap.Error(err),
			)
			httpx.AbortLocalizedError(ginCtx, ctx.I18n, http.StatusInternalServerError, messagecontract.CommonInternalError.String(), nil)
			return
		}

		httpx.WriteSuccess(ginCtx, http.StatusOK, payload)
	}
}
