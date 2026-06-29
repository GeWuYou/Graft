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

// handleUserScopedStableIDsRoute 处理单个目标的稳定 ID 替换路由。
//
// 它将指定的读取、绑定和写入逻辑交给通用替换处理器执行。
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

// handleBatchUserRoleRoute 处理批量用户角色稳定 ID 写入路由。
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

// handleReplaceRolePermissionsRoute 处理为单个角色替换权限的请求。
// 它将请求体中的权限 ID 绑定到生成的操作，并写入角色权限替换结果。
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

// handleAddRolePermissionsRoute 处理为角色添加权限的路由请求。
//
/**
 * This is invalid in Go comment format. Need just line comments.
 */
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

// handleRemoveRolePermissionsRoute 处理为角色移除权限的路由请求。
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

// handleReplaceUserRolesRoute 处理将用户角色替换为请求中指定的角色列表的 RBAC 路由。
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

// handleAddUserRolesRoute 处理为单个用户添加角色的写请求。
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

// handleRemoveUserRolesRoute 处理为单个用户移除角色的路由请求。
//
// 该处理器读取请求中的角色 ID 并将其写入 RBAC 管理服务。
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

// handleBatchReplaceUserRolesRoute 处理批量替换用户角色的写入请求。
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

// handleBatchAddUserRolesRoute 处理批量为用户添加角色的 RBAC 写入请求。
//
// 它解析请求体和参数，记录生成的操作，并将批量用户 ID 与角色 ID 写入持久层。
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

// handleBatchRemoveUserRolesRoute 处理批量移除用户角色的路由请求。
// 它读取并绑定请求中的用户 ID 和角色 ID，记录生成的操作，并将变更写入权限管理服务。
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

// handleReplaceStableIDsRoute 处理单个目标的稳定 ID 替换写入请求。
// 它会解析路径中的 `id`，读取并绑定请求体中的稳定 ID 列表，完成校验后执行写入；成功时返回 HTTP 200。
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

// handleBatchStableIDsRoute 处理批量稳定 ID 写入请求，完成请求解析、ID 校验和持久化。
// 成功时返回 HTTP 200。
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

// handleStableIDsRoute 处理单实体稳定 ID 写入路由，并记录生成的操作。
// 它先读取请求体并绑定参数，再将解析结果交给用户作用域的稳定 ID 写入流程。
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

// handleBatchStableIDsOperation 处理批量稳定 ID 写入操作，并记录对应的生成请求。
// 它读取请求体与绑定参数，记录操作后，委托给批量稳定 ID 路由完成校验和写入。
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
