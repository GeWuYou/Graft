package rbac

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	messagecontract "graft/server/internal/contract/message"
	rbacopenapi "graft/server/internal/contract/openapi/rbac"
	"graft/server/internal/httpx"
	"graft/server/internal/module"
	rbacstore "graft/server/modules/rbac/store"
)

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
	handleStableIDsRoute(
		ginCtx, ctx, moduleName,
		generatedStableIDsRouteConfig[rbacopenapi.PostRolePermissionsReplaceJSONRequestBody, rbacopenapi.PostRolePermissionsReplaceParams]{
			invalidField: "permission_ids",
			read:         readGeneratedRolePermissionReplaceRequest,
			bindParams:   bindGeneratedRolePermissionReplaceParams,
			record:       rbacWriteGeneratedHandler{}.PostRolePermissionsReplace,
			write: func(ctx context.Context, targetID uint64, ids []uint64) error {
				return writer.ReplacePermissionsForRole(ctx, rbacstore.ReplacePermissionsForRoleInput{RoleID: targetID, PermissionIDs: ids})
			},
		},
	)
}

//nolint:dupl // The generated request binders and write-service calls must stay explicit per operation.
func handleAddRolePermissionsRoute(ginCtx *gin.Context, ctx *module.Context, moduleName string, writer writeManagementService) {
	handleStableIDsRoute(
		ginCtx, ctx, moduleName,
		generatedStableIDsRouteConfig[rbacopenapi.PostRolePermissionsAddJSONRequestBody, rbacopenapi.PostRolePermissionsAddParams]{
			invalidField: "permission_ids",
			read:         readGeneratedRolePermissionAddRequest,
			bindParams:   bindGeneratedRolePermissionAddParams,
			record:       rbacWriteGeneratedHandler{}.PostRolePermissionsAdd,
			write: func(ctx context.Context, targetID uint64, ids []uint64) error {
				return writer.AddPermissionsToRole(ctx, rbacstore.AddPermissionsToRoleInput{RoleID: targetID, PermissionIDs: ids})
			},
		},
	)
}

//nolint:dupl // The generated request binders and write-service calls must stay explicit per operation.
func handleRemoveRolePermissionsRoute(ginCtx *gin.Context, ctx *module.Context, moduleName string, writer writeManagementService) {
	handleStableIDsRoute(
		ginCtx, ctx, moduleName,
		generatedStableIDsRouteConfig[rbacopenapi.PostRolePermissionsRemoveJSONRequestBody, rbacopenapi.PostRolePermissionsRemoveParams]{
			invalidField: "permission_ids",
			read:         readGeneratedRolePermissionRemoveRequest,
			bindParams:   bindGeneratedRolePermissionRemoveParams,
			record:       rbacWriteGeneratedHandler{}.PostRolePermissionsRemove,
			write: func(ctx context.Context, targetID uint64, ids []uint64) error {
				return writer.RemovePermissionsFromRole(ctx, rbacstore.RemovePermissionsFromRoleInput{RoleID: targetID, PermissionIDs: ids})
			},
		},
	)
}

//nolint:dupl // The generated request binders and write-service calls must stay explicit per operation.
func handleReplaceUserRolesRoute(ginCtx *gin.Context, ctx *module.Context, moduleName string, writer writeManagementService) {
	handleStableIDsRoute(
		ginCtx, ctx, moduleName,
		generatedStableIDsRouteConfig[rbacopenapi.PostUserRolesReplaceJSONRequestBody, rbacopenapi.PostUserRolesReplaceParams]{
			invalidField: "role_ids",
			read:         readGeneratedUserRoleReplaceRequest,
			bindParams:   bindGeneratedUserRoleReplaceParams,
			record:       rbacWriteGeneratedHandler{}.PostUserRolesReplace,
			write: func(ctx context.Context, targetID uint64, ids []uint64) error {
				return writer.ReplaceRolesForUser(ctx, rbacstore.ReplaceRolesForUserInput{UserID: targetID, RoleIDs: ids})
			},
		},
	)
}

//nolint:dupl // The generated request binders and write-service calls must stay explicit per operation.
func handleAddUserRolesRoute(ginCtx *gin.Context, ctx *module.Context, moduleName string, writer writeManagementService) {
	handleStableIDsRoute(
		ginCtx, ctx, moduleName,
		generatedStableIDsRouteConfig[rbacopenapi.PostUserRolesAddJSONRequestBody, rbacopenapi.PostUserRolesAddParams]{
			invalidField: "role_ids",
			read:         readGeneratedUserRoleAddRequest,
			bindParams:   bindGeneratedUserRoleAddParams,
			record:       rbacWriteGeneratedHandler{}.PostUserRolesAdd,
			write: func(ctx context.Context, targetID uint64, ids []uint64) error {
				return writer.AddRolesToUser(ctx, rbacstore.AddRolesToUserInput{UserID: targetID, RoleIDs: ids})
			},
		},
	)
}

//nolint:dupl // The generated request binders and write-service calls must stay explicit per operation.
func handleRemoveUserRolesRoute(ginCtx *gin.Context, ctx *module.Context, moduleName string, writer writeManagementService) {
	handleStableIDsRoute(
		ginCtx, ctx, moduleName,
		generatedStableIDsRouteConfig[rbacopenapi.PostUserRolesRemoveJSONRequestBody, rbacopenapi.PostUserRolesRemoveParams]{
			invalidField: "role_ids",
			read:         readGeneratedUserRoleRemoveRequest,
			bindParams:   bindGeneratedUserRoleRemoveParams,
			record:       rbacWriteGeneratedHandler{}.PostUserRolesRemove,
			write: func(ctx context.Context, targetID uint64, ids []uint64) error {
				return writer.RemoveRolesFromUser(ctx, rbacstore.RemoveRolesFromUserInput{UserID: targetID, RoleIDs: ids})
			},
		},
	)
}

//nolint:dupl // Batch generated request binders intentionally stay parallel to preserve operation ownership.
func handleBatchReplaceUserRolesRoute(ginCtx *gin.Context, ctx *module.Context, moduleName string, writer writeManagementService) {
	handleBatchStableIDsOperation(
		ginCtx, ctx, moduleName,
		batchGeneratedStableIDsRouteConfig[rbacopenapi.PostUsersRolesReplaceJSONRequestBody, rbacopenapi.PostUsersRolesReplaceParams]{
			read:       readGeneratedBatchUserRoleReplaceRequest,
			bindParams: bindGeneratedUsersRoleReplaceParams,
			record:     rbacWriteGeneratedHandler{}.PostUsersRolesReplace,
			write: func(ctx context.Context, userIDs []uint64, roleIDs []uint64) error {
				return writer.ReplaceRolesForUsers(ctx, rbacstore.BatchUserRoleMutationInput{UserIDs: userIDs, RoleIDs: roleIDs})
			},
		},
	)
}

//nolint:dupl // Batch generated request binders intentionally stay parallel to preserve operation ownership.
func handleBatchAddUserRolesRoute(ginCtx *gin.Context, ctx *module.Context, moduleName string, writer writeManagementService) {
	handleBatchStableIDsOperation(
		ginCtx, ctx, moduleName,
		batchGeneratedStableIDsRouteConfig[rbacopenapi.PostUsersRolesAddJSONRequestBody, rbacopenapi.PostUsersRolesAddParams]{
			read:       readGeneratedBatchUserRoleAddRequest,
			bindParams: bindGeneratedUsersRoleAddParams,
			record:     rbacWriteGeneratedHandler{}.PostUsersRolesAdd,
			write: func(ctx context.Context, userIDs []uint64, roleIDs []uint64) error {
				return writer.AddRolesToUsers(ctx, rbacstore.BatchUserRoleMutationInput{UserIDs: userIDs, RoleIDs: roleIDs})
			},
		},
	)
}

//nolint:dupl // Batch generated request binders intentionally stay parallel to preserve operation ownership.
func handleBatchRemoveUserRolesRoute(ginCtx *gin.Context, ctx *module.Context, moduleName string, writer writeManagementService) {
	handleBatchStableIDsOperation(
		ginCtx, ctx, moduleName,
		batchGeneratedStableIDsRouteConfig[rbacopenapi.PostUsersRolesRemoveJSONRequestBody, rbacopenapi.PostUsersRolesRemoveParams]{
			read:       readGeneratedBatchUserRoleRemoveRequest,
			bindParams: bindGeneratedUsersRoleRemoveParams,
			record:     rbacWriteGeneratedHandler{}.PostUsersRolesRemove,
			write: func(ctx context.Context, userIDs []uint64, roleIDs []uint64) error {
				return writer.RemoveRolesFromUsers(ctx, rbacstore.BatchUserRoleMutationInput{UserIDs: userIDs, RoleIDs: roleIDs})
			},
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

func handleStableIDsRoute[Body any, Params any](
	ginCtx *gin.Context,
	ctx *module.Context,
	moduleName string,
	config generatedStableIDsRouteConfig[Body, Params],
) {
	handleUserScopedStableIDsRoute(ginCtx, ctx, moduleName, config.invalidField,
		func(ginCtx *gin.Context, targetID uint64) ([]uint64, error) {
			body, ids, err := config.read(ginCtx)
			if err != nil {
				return nil, err
			}
			config.record(targetID, config.bindParams(ginCtx), body)
			return ids, nil
		},
		config.write,
	)
}

func handleBatchStableIDsOperation[Body any, Params any](
	ginCtx *gin.Context,
	ctx *module.Context,
	moduleName string,
	config batchGeneratedStableIDsRouteConfig[Body, Params],
) {
	handleBatchUserRoleRoute(ginCtx, ctx, moduleName,
		func(ginCtx *gin.Context) (batchStableIDSet, error) {
			body, request, err := config.read(ginCtx)
			if err != nil {
				return batchStableIDSet{}, err
			}
			config.record(config.bindParams(ginCtx), body)
			return request, nil
		},
		config.write,
	)
}
