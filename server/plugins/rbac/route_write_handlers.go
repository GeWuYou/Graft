package rbac

import (
	"context"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	messagecontract "graft/server/internal/contract/message"
	openapicontract "graft/server/internal/contract/openapi"
	"graft/server/internal/httpx"
	"graft/server/internal/plugin"
	rbaccontract "graft/server/plugins/rbac/contract"
	rbacstore "graft/server/plugins/rbac/store"
)

func registerRoleWriteRoutes(
	group *gin.RouterGroup,
	ctx *plugin.Context,
	pluginName string,
	writer writeManagementService,
	guards managementGuards,
) {
	group.POST(rbaccontract.RoleCollection, guards.roleCreate, func(ginCtx *gin.Context) {
		handleCreateRoleRoute(ginCtx, ctx, pluginName, writer)
	})

	group.POST(rbaccontract.RoleUpdateRoute, guards.roleUpdate, func(ginCtx *gin.Context) {
		handleUpdateRoleRoute(ginCtx, ctx, pluginName, writer)
	})

	group.POST(rbaccontract.RolePermissionAssignRoute, guards.rolePermissionAssign, func(ginCtx *gin.Context) {
		handleReplaceStableIDsRoute(ginCtx, ctx, pluginName, replaceStableIDsHandlerConfig{
			invalidField: "permission_ids",
			readIDs:      readRolePermissionIDs,
			write: func(ctx context.Context, targetID uint64, ids []uint64) error {
				return writer.ReplacePermissionsForRole(ctx, rbacstore.ReplacePermissionsForRoleInput{
					RoleID:        targetID,
					PermissionIDs: ids,
				})
			},
		})
	})
}

func handleCreateRoleRoute(
	ginCtx *gin.Context,
	ctx *plugin.Context,
	pluginName string,
	writer writeManagementService,
) {
	var request openapicontract.PostRolesJSONRequestBody
	if err := ginCtx.ShouldBindJSON(&request); err != nil {
		writeLocalizedContractError(ginCtx, ctx.I18n, http.StatusBadRequest, messagecontract.CommonInvalidArgument, map[string]any{
			"field": "body",
		})
		return
	}

	roleInput, ok := normalizeCreateRoleInput(request)
	if !ok {
		writeLocalizedContractError(ginCtx, ctx.I18n, http.StatusBadRequest, messagecontract.CommonInvalidArgument, map[string]any{
			"field": "name",
		})
		return
	}
	if strings.TrimSpace(roleInput.Display) == "" {
		writeLocalizedContractError(ginCtx, ctx.I18n, http.StatusBadRequest, messagecontract.CommonInvalidArgument, map[string]any{
			"field": "display",
		})
		return
	}

	role, err := writer.CreateRole(ginCtx.Request.Context(), roleInput)
	if err != nil {
		writeRBACManagementError(ginCtx, ctx.I18n, ctx.Logger, pluginName, err, "id")
		return
	}

	httpx.WriteSuccess(ginCtx, http.StatusOK, toRoleListItem(role))
}

func handleUpdateRoleRoute(
	ginCtx *gin.Context,
	ctx *plugin.Context,
	pluginName string,
	writer writeManagementService,
) {
	roleID, err := parseManagementID(ginCtx.Param("id"))
	if err != nil {
		writeLocalizedContractError(ginCtx, ctx.I18n, http.StatusBadRequest, messagecontract.CommonInvalidArgument, map[string]any{
			"field": "id",
		})
		return
	}

	var request openapicontract.PostRoleUpdateJSONRequestBody
	if err := ginCtx.ShouldBindJSON(&request); err != nil {
		writeLocalizedContractError(ginCtx, ctx.I18n, http.StatusBadRequest, messagecontract.CommonInvalidArgument, map[string]any{
			"field": "body",
		})
		return
	}

	roleInput, ok := normalizeUpdateRoleInput(roleID, request)
	if !ok {
		writeLocalizedContractError(ginCtx, ctx.I18n, http.StatusBadRequest, messagecontract.CommonInvalidArgument, map[string]any{
			"field": "name",
		})
		return
	}
	if strings.TrimSpace(roleInput.Display) == "" {
		writeLocalizedContractError(ginCtx, ctx.I18n, http.StatusBadRequest, messagecontract.CommonInvalidArgument, map[string]any{
			"field": "display",
		})
		return
	}

	role, err := writer.UpdateRole(ginCtx.Request.Context(), roleInput)
	if err != nil {
		writeRBACManagementError(ginCtx, ctx.I18n, ctx.Logger, pluginName, err, "id")
		return
	}

	httpx.WriteSuccess(ginCtx, http.StatusOK, toRoleListItem(role))
}

func handleReplaceStableIDsRoute(
	ginCtx *gin.Context,
	ctx *plugin.Context,
	pluginName string,
	config replaceStableIDsHandlerConfig,
) {
	targetID, err := parseManagementID(ginCtx.Param("id"))
	if err != nil {
		writeLocalizedContractError(ginCtx, ctx.I18n, http.StatusBadRequest, messagecontract.CommonInvalidArgument, map[string]any{
			"field": "id",
		})
		return
	}

	ids, err := config.readIDs(ginCtx)
	if err != nil {
		writeLocalizedContractError(ginCtx, ctx.I18n, http.StatusBadRequest, messagecontract.CommonInvalidArgument, map[string]any{
			"field": "body",
		})
		return
	}
	if ids == nil || hasInvalidStableIDs(ids) {
		writeLocalizedContractError(ginCtx, ctx.I18n, http.StatusBadRequest, messagecontract.CommonInvalidArgument, map[string]any{
			"field": config.invalidField,
		})
		return
	}

	if err := config.write(ginCtx.Request.Context(), targetID, ids); err != nil {
		writeRBACManagementError(ginCtx, ctx.I18n, ctx.Logger, pluginName, err, config.invalidField)
		return
	}

	httpx.WriteSuccess[any](ginCtx, http.StatusOK, nil)
}

func handleAssignUserRolesRoute(
	ginCtx *gin.Context,
	ctx *plugin.Context,
	pluginName string,
	writer writeManagementService,
) {
	targetID, err := parseManagementID(ginCtx.Param("id"))
	if err != nil {
		writeLocalizedContractError(ginCtx, ctx.I18n, http.StatusBadRequest, messagecontract.CommonInvalidArgument, map[string]any{
			"field": "id",
		})
		return
	}

	body, ids, err := readGeneratedUserRoleAssignRequest(ginCtx)
	if err != nil {
		writeLocalizedContractError(ginCtx, ctx.I18n, http.StatusBadRequest, messagecontract.CommonInvalidArgument, map[string]any{
			"field": "body",
		})
		return
	}
	if ids == nil || hasInvalidStableIDs(ids) {
		writeLocalizedContractError(ginCtx, ctx.I18n, http.StatusBadRequest, messagecontract.CommonInvalidArgument, map[string]any{
			"field": "role_ids",
		})
		return
	}

	rbacUserRoleGeneratedHandler{}.PostUserRolesAssign(targetID, bindGeneratedUserRoleAssignParams(ginCtx), body)

	if err := writer.ReplaceRolesForUser(ginCtx.Request.Context(), rbacstore.ReplaceRolesForUserInput{
		UserID:  targetID,
		RoleIDs: ids,
	}); err != nil {
		writeRBACManagementError(ginCtx, ctx.I18n, ctx.Logger, pluginName, err, "role_ids")
		return
	}

	httpx.WriteSuccess[any](ginCtx, http.StatusOK, nil)
}
