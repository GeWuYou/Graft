package rbac

import (
	"context"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	messagecontract "graft/server/internal/contract/message"
	rbacopenapi "graft/server/internal/contract/openapi/rbac"
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
		handleAssignRolePermissionsRoute(ginCtx, ctx, pluginName, writer)
	})
}

func handleCreateRoleRoute(
	ginCtx *gin.Context,
	ctx *plugin.Context,
	pluginName string,
	writer writeManagementService,
) {
	var request rbacopenapi.PostRolesJSONRequestBody
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

	rbacWriteGeneratedHandler{}.PostRoles(bindGeneratedRoleCreateParams(ginCtx), request)

	role, err := writer.CreateRole(ginCtx.Request.Context(), roleInput)
	if err != nil {
		writeRBACManagementError(ginCtx, ctx.I18n, ctx.Logger, pluginName, err, "id")
		return
	}

	payload, mapErr := toRoleListItem(role)
	if mapErr != nil {
		writeRBACManagementError(ginCtx, ctx.I18n, ctx.Logger, pluginName, mapErr, "id")
		return
	}

	httpx.WriteSuccess(ginCtx, http.StatusOK, payload)
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

	var request rbacopenapi.PostRoleUpdateJSONRequestBody
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

	rbacWriteGeneratedHandler{}.PostRoleUpdate(roleID, bindGeneratedRoleUpdateParams(ginCtx), request)

	role, err := writer.UpdateRole(ginCtx.Request.Context(), roleInput)
	if err != nil {
		writeRBACManagementError(ginCtx, ctx.I18n, ctx.Logger, pluginName, err, "id")
		return
	}

	payload, mapErr := toRoleListItem(role)
	if mapErr != nil {
		writeRBACManagementError(ginCtx, ctx.I18n, ctx.Logger, pluginName, mapErr, "id")
		return
	}

	httpx.WriteSuccess(ginCtx, http.StatusOK, payload)
}

type rbacWriteGeneratedHandler struct {
}

func (h rbacWriteGeneratedHandler) PostRoles(
	params rbacopenapi.PostRolesParams,
	body rbacopenapi.PostRolesJSONRequestBody,
) {
	_ = h
	_ = params
	_ = body
}

func (h rbacWriteGeneratedHandler) PostRoleUpdate(
	id uint64,
	params rbacopenapi.PostRoleUpdateParams,
	body rbacopenapi.PostRoleUpdateJSONRequestBody,
) {
	_ = h
	_ = id
	_ = params
	_ = body
}

func (h rbacWriteGeneratedHandler) PostRolePermissionAssign(
	id uint64,
	params rbacopenapi.PostRolePermissionAssignParams,
	body rbacopenapi.PostRolePermissionAssignJSONRequestBody,
) {
	_ = h
	_ = id
	_ = params
	_ = body
}

func (h rbacWriteGeneratedHandler) PostUserRolesAssign(
	id uint64,
	params rbacopenapi.PostUserRolesAssignParams,
	body rbacopenapi.PostUserRolesAssignJSONRequestBody,
) {
	_ = h
	_ = id
	_ = params
	_ = body
}

func bindGeneratedRoleCreateParams(ginCtx *gin.Context) rbacopenapi.PostRolesParams {
	locale, requestID := bindGeneratedReadHeaders(ginCtx)
	return rbacopenapi.PostRolesParams{
		XGraftLocale: locale,
		XRequestId:   requestID,
	}
}

func bindGeneratedRoleUpdateParams(ginCtx *gin.Context) rbacopenapi.PostRoleUpdateParams {
	locale, requestID := bindGeneratedReadHeaders(ginCtx)
	return rbacopenapi.PostRoleUpdateParams{
		XGraftLocale: locale,
		XRequestId:   requestID,
	}
}

func bindGeneratedRolePermissionAssignParams(ginCtx *gin.Context) rbacopenapi.PostRolePermissionAssignParams {
	locale, requestID := bindGeneratedReadHeaders(ginCtx)
	return rbacopenapi.PostRolePermissionAssignParams{
		XGraftLocale: locale,
		XRequestId:   requestID,
	}
}

//nolint:dupl // Generated-operation wrappers intentionally stay parallel while request handling is shared below.
func handleAssignRolePermissionsRoute(
	ginCtx *gin.Context,
	ctx *plugin.Context,
	pluginName string,
	writer writeManagementService,
) {
	handleReplaceStableIDsRoute(ginCtx, ctx, pluginName, replaceStableIDsHandlerConfig{
		invalidField: "permission_ids",
		readAndBindGenerated: func(ginCtx *gin.Context, targetID uint64) ([]uint64, error) {
			body, ids, err := readGeneratedRolePermissionAssignRequest(ginCtx)
			if err != nil {
				return nil, err
			}
			rbacWriteGeneratedHandler{}.PostRolePermissionAssign(targetID, bindGeneratedRolePermissionAssignParams(ginCtx), body)
			return ids, nil
		},
		write: func(ctx context.Context, targetID uint64, ids []uint64) error {
			return writer.ReplacePermissionsForRole(ctx, rbacstore.ReplacePermissionsForRoleInput{
				RoleID:        targetID,
				PermissionIDs: ids,
			})
		},
	})
}

//nolint:dupl // Generated-operation wrappers intentionally stay parallel while request handling is shared below.
func handleAssignUserRolesRoute(
	ginCtx *gin.Context,
	ctx *plugin.Context,
	pluginName string,
	writer writeManagementService,
) {
	handleReplaceStableIDsRoute(ginCtx, ctx, pluginName, replaceStableIDsHandlerConfig{
		invalidField: "role_ids",
		readAndBindGenerated: func(ginCtx *gin.Context, targetID uint64) ([]uint64, error) {
			body, ids, err := readGeneratedUserRoleAssignRequest(ginCtx)
			if err != nil {
				return nil, err
			}
			rbacWriteGeneratedHandler{}.PostUserRolesAssign(targetID, bindGeneratedUserRoleAssignParams(ginCtx), body)
			return ids, nil
		},
		write: func(ctx context.Context, targetID uint64, ids []uint64) error {
			return writer.ReplaceRolesForUser(ctx, rbacstore.ReplaceRolesForUserInput{
				UserID:  targetID,
				RoleIDs: ids,
			})
		},
	})
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

	ids, err := config.readAndBindGenerated(ginCtx, targetID)
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
