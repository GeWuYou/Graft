package rbac

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	messagecontract "graft/server/internal/contract/message"
	rbacopenapi "graft/server/internal/contract/openapi/rbac"
	"graft/server/internal/httpx"
	"graft/server/internal/module"
	rbaccontract "graft/server/modules/rbac/contract"
	rbacstore "graft/server/modules/rbac/store"
)

func registerRoleWriteRoutes(
	group *gin.RouterGroup,
	ctx *module.Context,
	moduleName string,
	writer writeManagementService,
	guards managementGuards,
) {
	group.POST(rbaccontract.RoleCollection, guards.roleCreate, func(ginCtx *gin.Context) {
		handleCreateRoleRoute(ginCtx, ctx, moduleName, writer)
	})

	group.POST(rbaccontract.RoleUpdateRoute, guards.roleUpdate, func(ginCtx *gin.Context) {
		handleUpdateRoleRoute(ginCtx, ctx, moduleName, writer)
	})

	group.POST(rbaccontract.RoleStatusRoute, guards.roleStatus, func(ginCtx *gin.Context) { handleUpdateRoleStatusRoute(ginCtx, ctx, moduleName, writer) })
	group.POST(rbaccontract.RoleDeleteRoute, guards.roleDelete, func(ginCtx *gin.Context) { handleDeleteRoleRoute(ginCtx, ctx, moduleName, writer) })
	group.POST(rbaccontract.RolePermissionReplaceRoute, guards.rolePermissionAssign, func(ginCtx *gin.Context) { handleReplaceRolePermissionsRoute(ginCtx, ctx, moduleName, writer) })
	group.POST(rbaccontract.RolePermissionAddRoute, guards.rolePermissionAssign, func(ginCtx *gin.Context) { handleAddRolePermissionsRoute(ginCtx, ctx, moduleName, writer) })
	group.POST(rbaccontract.RolePermissionRemoveRoute, guards.rolePermissionAssign, func(ginCtx *gin.Context) { handleRemoveRolePermissionsRoute(ginCtx, ctx, moduleName, writer) })
}

func handleCreateRoleRoute(
	ginCtx *gin.Context,
	ctx *module.Context,
	moduleName string,
	writer writeManagementService,
) {
	requestCtx := ginCtx.Request.Context()
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

	role, err := writer.CreateRole(requestCtx, roleInput)
	if err != nil {
		writeRBACManagementError(ginCtx, ctx.I18n, ctx.Logger, moduleName, err, "id")
		return
	}

	payload, mapErr := toRoleListItem(role)
	if mapErr != nil {
		writeRBACManagementError(ginCtx, ctx.I18n, ctx.Logger, moduleName, mapErr, "id")
		return
	}

	httpx.WriteSuccess(ginCtx, http.StatusOK, payload)
}

func handleUpdateRoleRoute(
	ginCtx *gin.Context,
	ctx *module.Context,
	moduleName string,
	writer writeManagementService,
) {
	requestCtx := ginCtx.Request.Context()
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

	role, err := writer.UpdateRole(requestCtx, roleInput)
	if err != nil {
		writeRBACManagementError(ginCtx, ctx.I18n, ctx.Logger, moduleName, err, "id")
		return
	}

	payload, mapErr := toRoleListItem(role)
	if mapErr != nil {
		writeRBACManagementError(ginCtx, ctx.I18n, ctx.Logger, moduleName, mapErr, "id")
		return
	}

	httpx.WriteSuccess(ginCtx, http.StatusOK, payload)
}

func handleUpdateRoleStatusRoute(ginCtx *gin.Context, ctx *module.Context, moduleName string, writer writeManagementService) {
	requestCtx := ginCtx.Request.Context()
	roleID, err := parseManagementID(ginCtx.Param("id"))
	if err != nil {
		writeLocalizedContractError(ginCtx, ctx.I18n, http.StatusBadRequest, messagecontract.CommonInvalidArgument, map[string]any{"field": "id"})
		return
	}

	var request rbacopenapi.PostRoleStatusJSONRequestBody
	if err := ginCtx.ShouldBindJSON(&request); err != nil {
		writeLocalizedContractError(ginCtx, ctx.I18n, http.StatusBadRequest, messagecontract.CommonInvalidArgument, map[string]any{"field": "body"})
		return
	}
	status, ok := normalizeRoleStatusInput(request)
	if !ok {
		writeLocalizedContractError(ginCtx, ctx.I18n, http.StatusBadRequest, messagecontract.CommonInvalidArgument, map[string]any{"field": "status"})
		return
	}

	rbacWriteGeneratedHandler{}.PostRoleStatus(roleID, bindGeneratedRoleStatusParams(ginCtx), request)

	role, err := writer.SetRoleStatus(requestCtx, rbacstore.SetRoleStatusInput{ID: roleID, Status: status})
	if err != nil {
		writeRBACManagementError(ginCtx, ctx.I18n, ctx.Logger, moduleName, err, "status")
		return
	}
	payload, mapErr := toRoleListItem(role)
	if mapErr != nil {
		writeRBACManagementError(ginCtx, ctx.I18n, ctx.Logger, moduleName, mapErr, "status")
		return
	}
	httpx.WriteSuccess(ginCtx, http.StatusOK, payload)
}

func handleDeleteRoleRoute(ginCtx *gin.Context, ctx *module.Context, moduleName string, writer writeManagementService) {
	requestCtx := ginCtx.Request.Context()
	roleID, err := parseManagementID(ginCtx.Param("id"))
	if err != nil {
		writeLocalizedContractError(ginCtx, ctx.I18n, http.StatusBadRequest, messagecontract.CommonInvalidArgument, map[string]any{"field": "id"})
		return
	}
	rbacWriteGeneratedHandler{}.PostRoleDelete(roleID, bindGeneratedRoleDeleteParams(ginCtx))
	if err := writer.SoftDeleteRole(requestCtx, rbacstore.SoftDeleteRoleInput{ID: roleID}); err != nil {
		writeRBACManagementError(ginCtx, ctx.I18n, ctx.Logger, moduleName, err, "id")
		return
	}
	httpx.WriteSuccess[any](ginCtx, http.StatusOK, nil)
}
