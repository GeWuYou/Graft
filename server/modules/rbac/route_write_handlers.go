// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package rbac

import (
	"context"
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

func (h rbacWriteGeneratedHandler) PostRoleDelete(
	id uint64,
	params rbacopenapi.PostRoleDeleteParams,
) {
	_ = h
	_ = id
	_ = params
}

func (h rbacWriteGeneratedHandler) PostRolePermissionsAdd(
	id uint64,
	params rbacopenapi.PostRolePermissionsAddParams,
	body rbacopenapi.PostRolePermissionsAddJSONRequestBody,
) {
	_ = h
	_ = id
	_ = params
	_ = body
}

func (h rbacWriteGeneratedHandler) PostRolePermissionsRemove(
	id uint64,
	params rbacopenapi.PostRolePermissionsRemoveParams,
	body rbacopenapi.PostRolePermissionsRemoveJSONRequestBody,
) {
	_ = h
	_ = id
	_ = params
	_ = body
}

func (h rbacWriteGeneratedHandler) PostRolePermissionsReplace(
	id uint64,
	params rbacopenapi.PostRolePermissionsReplaceParams,
	body rbacopenapi.PostRolePermissionsReplaceJSONRequestBody,
) {
	_ = h
	_ = id
	_ = params
	_ = body
}

func (h rbacWriteGeneratedHandler) PostRoleStatus(
	id uint64,
	params rbacopenapi.PostRoleStatusParams,
	body rbacopenapi.PostRoleStatusJSONRequestBody,
) {
	_ = h
	_ = id
	_ = params
	_ = body
}

func (h rbacWriteGeneratedHandler) PostUserRolesAdd(
	id uint64,
	params rbacopenapi.PostUserRolesAddParams,
	body rbacopenapi.PostUserRolesAddJSONRequestBody,
) {
	_ = h
	_ = id
	_ = params
	_ = body
}

func (h rbacWriteGeneratedHandler) PostUserRolesRemove(
	id uint64,
	params rbacopenapi.PostUserRolesRemoveParams,
	body rbacopenapi.PostUserRolesRemoveJSONRequestBody,
) {
	_ = h
	_ = id
	_ = params
	_ = body
}

func (h rbacWriteGeneratedHandler) PostUserRolesReplace(
	id uint64,
	params rbacopenapi.PostUserRolesReplaceParams,
	body rbacopenapi.PostUserRolesReplaceJSONRequestBody,
) {
	_ = h
	_ = id
	_ = params
	_ = body
}

func (h rbacWriteGeneratedHandler) PostUsersRolesAdd(
	params rbacopenapi.PostUsersRolesAddParams,
	body rbacopenapi.PostUsersRolesAddJSONRequestBody,
) {
	_ = h
	_ = params
	_ = body
}

func (h rbacWriteGeneratedHandler) PostUsersRolesRemove(
	params rbacopenapi.PostUsersRolesRemoveParams,
	body rbacopenapi.PostUsersRolesRemoveJSONRequestBody,
) {
	_ = h
	_ = params
	_ = body
}

func (h rbacWriteGeneratedHandler) PostUsersRolesReplace(
	params rbacopenapi.PostUsersRolesReplaceParams,
	body rbacopenapi.PostUsersRolesReplaceJSONRequestBody,
) {
	_ = h
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

func bindGeneratedRoleDeleteParams(ginCtx *gin.Context) rbacopenapi.PostRoleDeleteParams {
	locale, requestID := bindGeneratedReadHeaders(ginCtx)
	return rbacopenapi.PostRoleDeleteParams{
		XGraftLocale: locale,
		XRequestId:   requestID,
	}
}

func bindGeneratedRolePermissionReplaceParams(ginCtx *gin.Context) rbacopenapi.PostRolePermissionsReplaceParams {
	locale, requestID := bindGeneratedReadHeaders(ginCtx)
	return rbacopenapi.PostRolePermissionsReplaceParams{XGraftLocale: locale, XRequestId: requestID}
}

func bindGeneratedRolePermissionAddParams(ginCtx *gin.Context) rbacopenapi.PostRolePermissionsAddParams {
	locale, requestID := bindGeneratedReadHeaders(ginCtx)
	return rbacopenapi.PostRolePermissionsAddParams{XGraftLocale: locale, XRequestId: requestID}
}

func bindGeneratedRolePermissionRemoveParams(ginCtx *gin.Context) rbacopenapi.PostRolePermissionsRemoveParams {
	locale, requestID := bindGeneratedReadHeaders(ginCtx)
	return rbacopenapi.PostRolePermissionsRemoveParams{XGraftLocale: locale, XRequestId: requestID}
}

func bindGeneratedRoleStatusParams(ginCtx *gin.Context) rbacopenapi.PostRoleStatusParams {
	locale, requestID := bindGeneratedReadHeaders(ginCtx)
	return rbacopenapi.PostRoleStatusParams{XGraftLocale: locale, XRequestId: requestID}
}

func bindGeneratedUserRoleReplaceParams(ginCtx *gin.Context) rbacopenapi.PostUserRolesReplaceParams {
	locale, requestID := bindGeneratedReadHeaders(ginCtx)
	return rbacopenapi.PostUserRolesReplaceParams{XGraftLocale: locale, XRequestId: requestID}
}

func bindGeneratedUserRoleAddParams(ginCtx *gin.Context) rbacopenapi.PostUserRolesAddParams {
	locale, requestID := bindGeneratedReadHeaders(ginCtx)
	return rbacopenapi.PostUserRolesAddParams{XGraftLocale: locale, XRequestId: requestID}
}

func bindGeneratedUserRoleRemoveParams(ginCtx *gin.Context) rbacopenapi.PostUserRolesRemoveParams {
	locale, requestID := bindGeneratedReadHeaders(ginCtx)
	return rbacopenapi.PostUserRolesRemoveParams{XGraftLocale: locale, XRequestId: requestID}
}

func bindGeneratedUsersRoleReplaceParams(ginCtx *gin.Context) rbacopenapi.PostUsersRolesReplaceParams {
	locale, requestID := bindGeneratedReadHeaders(ginCtx)
	return rbacopenapi.PostUsersRolesReplaceParams{XGraftLocale: locale, XRequestId: requestID}
}

func bindGeneratedUsersRoleAddParams(ginCtx *gin.Context) rbacopenapi.PostUsersRolesAddParams {
	locale, requestID := bindGeneratedReadHeaders(ginCtx)
	return rbacopenapi.PostUsersRolesAddParams{XGraftLocale: locale, XRequestId: requestID}
}

func bindGeneratedUsersRoleRemoveParams(ginCtx *gin.Context) rbacopenapi.PostUsersRolesRemoveParams {
	locale, requestID := bindGeneratedReadHeaders(ginCtx)
	return rbacopenapi.PostUsersRolesRemoveParams{XGraftLocale: locale, XRequestId: requestID}
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

func handleUserScopedStableIDsRoute(
	ginCtx *gin.Context,
	ctx *module.Context,
	moduleName string,
	invalidField string,
	readAndBindGenerated func(ginCtx *gin.Context, targetID uint64) ([]uint64, error),
	write func(ctx context.Context, targetID uint64, ids []uint64) error,
) {
	handleReplaceStableIDsRoute(ginCtx, ctx, moduleName, replaceStableIDsHandlerConfig{
		invalidField:         invalidField,
		readAndBindGenerated: readAndBindGenerated,
		write:                write,
	})
}

func handleBatchUserRoleRoute(
	ginCtx *gin.Context,
	ctx *module.Context,
	moduleName string,
	readAndBindGenerated func(ginCtx *gin.Context) (batchStableIDSet, error),
	write func(ctx context.Context, userIDs []uint64, roleIDs []uint64) error,
) {
	handleBatchStableIDsRoute(ginCtx, ctx, moduleName, batchStableIDsHandlerConfig{
		invalidField:         "role_ids",
		readAndBindGenerated: readAndBindGenerated,
		write:                write,
	})
}

//nolint:dupl // The generated request binders and write-service calls must stay explicit per operation.
func handleReplaceRolePermissionsRoute(ginCtx *gin.Context, ctx *module.Context, moduleName string, writer writeManagementService) {
	handleUserScopedStableIDsRoute(ginCtx, ctx, moduleName, "permission_ids",
		func(ginCtx *gin.Context, targetID uint64) ([]uint64, error) {
			body, ids, err := readGeneratedRolePermissionReplaceRequest(ginCtx)
			if err != nil {
				return nil, err
			}
			rbacWriteGeneratedHandler{}.PostRolePermissionsReplace(targetID, bindGeneratedRolePermissionReplaceParams(ginCtx), body)
			return ids, nil
		},
		func(ctx context.Context, targetID uint64, ids []uint64) error {
			return writer.ReplacePermissionsForRole(ctx, rbacstore.ReplacePermissionsForRoleInput{RoleID: targetID, PermissionIDs: ids})
		},
	)
}

//nolint:dupl // The generated request binders and write-service calls must stay explicit per operation.
func handleAddRolePermissionsRoute(ginCtx *gin.Context, ctx *module.Context, moduleName string, writer writeManagementService) {
	handleUserScopedStableIDsRoute(ginCtx, ctx, moduleName, "permission_ids",
		func(ginCtx *gin.Context, targetID uint64) ([]uint64, error) {
			body, ids, err := readGeneratedRolePermissionAddRequest(ginCtx)
			if err != nil {
				return nil, err
			}
			rbacWriteGeneratedHandler{}.PostRolePermissionsAdd(targetID, bindGeneratedRolePermissionAddParams(ginCtx), body)
			return ids, nil
		},
		func(ctx context.Context, targetID uint64, ids []uint64) error {
			return writer.AddPermissionsToRole(ctx, rbacstore.AddPermissionsToRoleInput{RoleID: targetID, PermissionIDs: ids})
		},
	)
}

//nolint:dupl // The generated request binders and write-service calls must stay explicit per operation.
func handleRemoveRolePermissionsRoute(ginCtx *gin.Context, ctx *module.Context, moduleName string, writer writeManagementService) {
	handleUserScopedStableIDsRoute(ginCtx, ctx, moduleName, "permission_ids",
		func(ginCtx *gin.Context, targetID uint64) ([]uint64, error) {
			body, ids, err := readGeneratedRolePermissionRemoveRequest(ginCtx)
			if err != nil {
				return nil, err
			}
			rbacWriteGeneratedHandler{}.PostRolePermissionsRemove(targetID, bindGeneratedRolePermissionRemoveParams(ginCtx), body)
			return ids, nil
		},
		func(ctx context.Context, targetID uint64, ids []uint64) error {
			return writer.RemovePermissionsFromRole(ctx, rbacstore.RemovePermissionsFromRoleInput{RoleID: targetID, PermissionIDs: ids})
		},
	)
}

//nolint:dupl // The generated request binders and write-service calls must stay explicit per operation.
func handleReplaceUserRolesRoute(ginCtx *gin.Context, ctx *module.Context, moduleName string, writer writeManagementService) {
	handleUserScopedStableIDsRoute(ginCtx, ctx, moduleName, "role_ids",
		func(ginCtx *gin.Context, targetID uint64) ([]uint64, error) {
			body, ids, err := readGeneratedUserRoleReplaceRequest(ginCtx)
			if err != nil {
				return nil, err
			}
			rbacWriteGeneratedHandler{}.PostUserRolesReplace(targetID, bindGeneratedUserRoleReplaceParams(ginCtx), body)
			return ids, nil
		},
		func(ctx context.Context, targetID uint64, ids []uint64) error {
			return writer.ReplaceRolesForUser(ctx, rbacstore.ReplaceRolesForUserInput{UserID: targetID, RoleIDs: ids})
		},
	)
}

//nolint:dupl // The generated request binders and write-service calls must stay explicit per operation.
func handleAddUserRolesRoute(ginCtx *gin.Context, ctx *module.Context, moduleName string, writer writeManagementService) {
	handleUserScopedStableIDsRoute(ginCtx, ctx, moduleName, "role_ids",
		func(ginCtx *gin.Context, targetID uint64) ([]uint64, error) {
			body, ids, err := readGeneratedUserRoleAddRequest(ginCtx)
			if err != nil {
				return nil, err
			}
			rbacWriteGeneratedHandler{}.PostUserRolesAdd(targetID, bindGeneratedUserRoleAddParams(ginCtx), body)
			return ids, nil
		},
		func(ctx context.Context, targetID uint64, ids []uint64) error {
			return writer.AddRolesToUser(ctx, rbacstore.AddRolesToUserInput{UserID: targetID, RoleIDs: ids})
		},
	)
}

//nolint:dupl // The generated request binders and write-service calls must stay explicit per operation.
func handleRemoveUserRolesRoute(ginCtx *gin.Context, ctx *module.Context, moduleName string, writer writeManagementService) {
	handleUserScopedStableIDsRoute(ginCtx, ctx, moduleName, "role_ids",
		func(ginCtx *gin.Context, targetID uint64) ([]uint64, error) {
			body, ids, err := readGeneratedUserRoleRemoveRequest(ginCtx)
			if err != nil {
				return nil, err
			}
			rbacWriteGeneratedHandler{}.PostUserRolesRemove(targetID, bindGeneratedUserRoleRemoveParams(ginCtx), body)
			return ids, nil
		},
		func(ctx context.Context, targetID uint64, ids []uint64) error {
			return writer.RemoveRolesFromUser(ctx, rbacstore.RemoveRolesFromUserInput{UserID: targetID, RoleIDs: ids})
		},
	)
}

//nolint:dupl // Batch generated request binders intentionally stay parallel to preserve operation ownership.
func handleBatchReplaceUserRolesRoute(ginCtx *gin.Context, ctx *module.Context, moduleName string, writer writeManagementService) {
	handleBatchUserRoleRoute(ginCtx, ctx, moduleName,
		func(ginCtx *gin.Context) (batchStableIDSet, error) {
			body, request, err := readGeneratedBatchUserRoleReplaceRequest(ginCtx)
			if err != nil {
				return batchStableIDSet{}, err
			}
			rbacWriteGeneratedHandler{}.PostUsersRolesReplace(bindGeneratedUsersRoleReplaceParams(ginCtx), body)
			return request, nil
		},
		func(ctx context.Context, userIDs []uint64, roleIDs []uint64) error {
			return writer.ReplaceRolesForUsers(ctx, rbacstore.BatchUserRoleMutationInput{UserIDs: userIDs, RoleIDs: roleIDs})
		},
	)
}

//nolint:dupl // Batch generated request binders intentionally stay parallel to preserve operation ownership.
func handleBatchAddUserRolesRoute(ginCtx *gin.Context, ctx *module.Context, moduleName string, writer writeManagementService) {
	handleBatchUserRoleRoute(ginCtx, ctx, moduleName,
		func(ginCtx *gin.Context) (batchStableIDSet, error) {
			body, request, err := readGeneratedBatchUserRoleAddRequest(ginCtx)
			if err != nil {
				return batchStableIDSet{}, err
			}
			rbacWriteGeneratedHandler{}.PostUsersRolesAdd(bindGeneratedUsersRoleAddParams(ginCtx), body)
			return request, nil
		},
		func(ctx context.Context, userIDs []uint64, roleIDs []uint64) error {
			return writer.AddRolesToUsers(ctx, rbacstore.BatchUserRoleMutationInput{UserIDs: userIDs, RoleIDs: roleIDs})
		},
	)
}

//nolint:dupl // Batch generated request binders intentionally stay parallel to preserve operation ownership.
func handleBatchRemoveUserRolesRoute(ginCtx *gin.Context, ctx *module.Context, moduleName string, writer writeManagementService) {
	handleBatchUserRoleRoute(ginCtx, ctx, moduleName,
		func(ginCtx *gin.Context) (batchStableIDSet, error) {
			body, request, err := readGeneratedBatchUserRoleRemoveRequest(ginCtx)
			if err != nil {
				return batchStableIDSet{}, err
			}
			rbacWriteGeneratedHandler{}.PostUsersRolesRemove(bindGeneratedUsersRoleRemoveParams(ginCtx), body)
			return request, nil
		},
		func(ctx context.Context, userIDs []uint64, roleIDs []uint64) error {
			return writer.RemoveRolesFromUsers(ctx, rbacstore.BatchUserRoleMutationInput{UserIDs: userIDs, RoleIDs: roleIDs})
		},
	)
}

func handleReplaceStableIDsRoute(
	ginCtx *gin.Context,
	ctx *module.Context,
	moduleName string,
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

	requestCtx := ginCtx.Request.Context()
	if err := config.write(requestCtx, targetID, ids); err != nil {
		writeRBACManagementError(ginCtx, ctx.I18n, ctx.Logger, moduleName, err, config.invalidField)
		return
	}

	httpx.WriteSuccess[any](ginCtx, http.StatusOK, nil)
}

func handleBatchStableIDsRoute(
	ginCtx *gin.Context,
	ctx *module.Context,
	moduleName string,
	config batchStableIDsHandlerConfig,
) {
	request, err := config.readAndBindGenerated(ginCtx)
	if err != nil {
		writeLocalizedContractError(ginCtx, ctx.I18n, http.StatusBadRequest, messagecontract.CommonInvalidArgument, map[string]any{"field": "body"})
		return
	}
	if request.userIDs == nil || request.roleIDs == nil || hasInvalidStableIDs(request.userIDs) || hasInvalidStableIDs(request.roleIDs) {
		writeLocalizedContractError(ginCtx, ctx.I18n, http.StatusBadRequest, messagecontract.CommonInvalidArgument, map[string]any{"field": config.invalidField})
		return
	}
	requestCtx := ginCtx.Request.Context()
	if err := config.write(requestCtx, request.userIDs, request.roleIDs); err != nil {
		writeRBACManagementError(ginCtx, ctx.I18n, ctx.Logger, moduleName, err, config.invalidField)
		return
	}
	httpx.WriteSuccess[any](ginCtx, http.StatusOK, nil)
}
