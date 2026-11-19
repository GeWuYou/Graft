package rbac

import (
	"context"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	messagecontract "graft/server/internal/contract/message"
	generated "graft/server/internal/contract/openapi/generated"
	rbacopenapi "graft/server/internal/contract/openapi/rbac"
	"graft/server/internal/httpx"
	"graft/server/internal/module"
	rbacstore "graft/server/modules/rbac/store"
)

// handleReplaceRolePermissionsRoute 处理为单个角色替换权限的请求。
// 它将请求体中的权限 ID 绑定到生成的操作，并写入角色权限替换结果。
func handleReplaceRolePermissionsRoute(ginCtx *gin.Context, ctx *module.Context, moduleName string, writer writeManagementService) {
	handleStableIDsRoute(
		ginCtx, ctx, moduleName,
		generatedStableIDsRouteConfig[rbacopenapi.PostRolePermissionsReplaceJSONRequestBody]{
			invalidField: "permission_ids",
			read:         readGeneratedRolePermissionReplaceRequest,
			write: func(ctx context.Context, targetID uint64, ids []uint64) error {
				return writer.ReplacePermissionsForRole(ctx, rbacstore.ReplacePermissionsForRoleInput{RoleID: targetID, PermissionIDs: ids})
			},
			allowEmptyIDs: true,
		},
	)
}

// handleAddRolePermissionsRoute 处理为角色添加权限的路由请求。
func handleAddRolePermissionsRoute(ginCtx *gin.Context, ctx *module.Context, moduleName string, writer writeManagementService) {
	handleStableIDsRoute(
		ginCtx, ctx, moduleName,
		generatedStableIDsRouteConfig[rbacopenapi.PostRolePermissionsAddJSONRequestBody]{
			invalidField: "permission_ids",
			read:         readGeneratedRolePermissionAddRequest,
			write: func(ctx context.Context, targetID uint64, ids []uint64) error {
				return writer.AddPermissionsToRole(ctx, rbacstore.AddPermissionsToRoleInput{RoleID: targetID, PermissionIDs: ids})
			},
			allowEmptyIDs: true,
		},
	)
}

// handleRemoveRolePermissionsRoute 处理为角色移除权限的路由请求。
func handleRemoveRolePermissionsRoute(ginCtx *gin.Context, ctx *module.Context, moduleName string, writer writeManagementService) {
	handleStableIDsRoute(
		ginCtx, ctx, moduleName,
		generatedStableIDsRouteConfig[rbacopenapi.DeleteRolePermissionsJSONRequestBody]{
			invalidField: "permission_ids",
			read:         readGeneratedRolePermissionRemoveRequest,
			write: func(ctx context.Context, targetID uint64, ids []uint64) error {
				return writer.RemovePermissionsFromRole(ctx, rbacstore.RemovePermissionsFromRoleInput{RoleID: targetID, PermissionIDs: ids})
			},
			resultStatus: stableIDBatchResultStatusRemoved,
		},
	)
}

// handleReplaceUserRolesRoute 处理将用户角色替换为请求中指定的角色列表的 RBAC 路由。
func handleReplaceUserRolesRoute(ginCtx *gin.Context, ctx *module.Context, moduleName string, writer writeManagementService) {
	handleStableIDsRoute(
		ginCtx, ctx, moduleName,
		generatedStableIDsRouteConfig[rbacopenapi.PostUserRolesReplaceJSONRequestBody]{
			invalidField: "role_ids",
			read:         readGeneratedUserRoleReplaceRequest,
			write: func(ctx context.Context, targetID uint64, ids []uint64) error {
				return writer.ReplaceRolesForUser(ctx, rbacstore.ReplaceRolesForUserInput{UserID: targetID, RoleIDs: ids})
			},
			allowEmptyIDs: true,
		},
	)
}

// handleAddUserRolesRoute 处理为单个用户添加角色的写请求。
func handleAddUserRolesRoute(ginCtx *gin.Context, ctx *module.Context, moduleName string, writer writeManagementService) {
	handleStableIDsRoute(
		ginCtx, ctx, moduleName,
		generatedStableIDsRouteConfig[rbacopenapi.PostUserRolesAddJSONRequestBody]{
			invalidField: "role_ids",
			read:         readGeneratedUserRoleAddRequest,
			write: func(ctx context.Context, targetID uint64, ids []uint64) error {
				return writer.AddRolesToUser(ctx, rbacstore.AddRolesToUserInput{UserID: targetID, RoleIDs: ids})
			},
			allowEmptyIDs: true,
		},
	)
}

// handleRemoveUserRolesRoute 处理为单个用户移除角色的路由请求。
//
// 该处理器读取请求中的角色 ID 并将其写入 RBAC 管理服务。
func handleRemoveUserRolesRoute(ginCtx *gin.Context, ctx *module.Context, moduleName string, writer writeManagementService) {
	handleStableIDsRoute(
		ginCtx, ctx, moduleName,
		generatedStableIDsRouteConfig[rbacopenapi.DeleteUserRolesJSONRequestBody]{
			invalidField: "role_ids",
			read:         readGeneratedUserRoleRemoveRequest,
			write: func(ctx context.Context, targetID uint64, ids []uint64) error {
				return writer.RemoveRolesFromUser(ctx, rbacstore.RemoveRolesFromUserInput{UserID: targetID, RoleIDs: ids})
			},
			resultStatus: stableIDBatchResultStatusRemoved,
		},
	)
}

// handleBatchReplaceUserRolesRoute 处理批量替换用户角色的写入请求。
func handleBatchReplaceUserRolesRoute(ginCtx *gin.Context, ctx *module.Context, moduleName string, writer writeManagementService) {
	handleBatchStableIDsOperation(
		ginCtx, ctx, moduleName,
		batchGeneratedStableIDsRouteConfig[rbacopenapi.PostUsersRolesReplaceJSONRequestBody]{
			read: readGeneratedBatchUserRoleReplaceRequest,
			write: func(ctx context.Context, userIDs []uint64, roleIDs []uint64) error {
				return writer.ReplaceRolesForUsers(ctx, rbacstore.BatchUserRoleMutationInput{UserIDs: userIDs, RoleIDs: roleIDs})
			},
			allowEmptyRoleIDs: true,
			resultStatus:      stableIDBatchResultStatusAccepted,
		},
	)
}

// handleBatchAddUserRolesRoute 处理批量为用户添加角色的 RBAC 写入请求。
//
// 它解析请求体和参数，记录生成的操作，并将批量用户 ID 与角色 ID 写入持久层。
func handleBatchAddUserRolesRoute(ginCtx *gin.Context, ctx *module.Context, moduleName string, writer writeManagementService) {
	handleBatchStableIDsOperation(
		ginCtx, ctx, moduleName,
		batchGeneratedStableIDsRouteConfig[rbacopenapi.PostUsersRolesAddJSONRequestBody]{
			read: readGeneratedBatchUserRoleAddRequest,
			write: func(ctx context.Context, userIDs []uint64, roleIDs []uint64) error {
				return writer.AddRolesToUsers(ctx, rbacstore.BatchUserRoleMutationInput{UserIDs: userIDs, RoleIDs: roleIDs})
			},
			allowEmptyRoleIDs: true,
			resultStatus:      stableIDBatchResultStatusAccepted,
		},
	)
}

// handleBatchRemoveUserRolesRoute 处理批量移除用户角色的路由请求。
// 它读取并绑定请求中的用户 ID 和角色 ID，记录生成的操作，并将变更写入权限管理服务。
func handleBatchRemoveUserRolesRoute(ginCtx *gin.Context, ctx *module.Context, moduleName string, writer writeManagementService) {
	handleBatchStableIDsOperation(
		ginCtx, ctx, moduleName,
		batchGeneratedStableIDsRouteConfig[rbacopenapi.DeleteUsersRolesJSONRequestBody]{
			read: readGeneratedBatchUserRoleRemoveRequest,
			write: func(ctx context.Context, userIDs []uint64, roleIDs []uint64) error {
				return writer.RemoveRolesFromUsers(ctx, rbacstore.BatchUserRoleMutationInput{UserIDs: userIDs, RoleIDs: roleIDs})
			},
			resultStatus: stableIDBatchResultStatusRemoved,
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

	ids, err := config.readAndBindGenerated(ginCtx)
	if err != nil {
		writeLocalizedContractError(ginCtx, ctx.I18n, http.StatusBadRequest, messagecontract.CommonInvalidArgument, map[string]any{
			"field": "body",
		})
		return
	}
	if invalidStableIDBatch(ids, config.allowEmptyIDs) {
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

	if config.resultStatus == "" {
		httpx.WriteSuccess[any](ginCtx, http.StatusOK, nil)
		return
	}
	writeStableIDBatchResult(ginCtx, ids, config.resultStatus)
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
	switch {
	case invalidStableIDBatch(request.userIDs, false):
		writeLocalizedContractError(ginCtx, ctx.I18n, http.StatusBadRequest, messagecontract.CommonInvalidArgument, map[string]any{"field": "user_ids"})
		return
	case invalidStableIDBatch(request.roleIDs, config.allowEmptyRoleIDs):
		writeLocalizedContractError(ginCtx, ctx.I18n, http.StatusBadRequest, messagecontract.CommonInvalidArgument, map[string]any{"field": config.invalidField})
		return
	}
	requestCtx := ginCtx.Request.Context()
	if err := config.write(requestCtx, request.userIDs, request.roleIDs); err != nil {
		writeRBACManagementError(ginCtx, ctx.I18n, ctx.Logger, moduleName, err, config.invalidField)
		return
	}
	writeStableIDBatchResult(ginCtx, request.userIDs, config.resultStatus)
}

// handleStableIDsRoute 将单实体写入的请求体解析交给保留无符号 ID 语义的模块 binder。
func handleStableIDsRoute[Body any](
	ginCtx *gin.Context,
	ctx *module.Context,
	moduleName string,
	config generatedStableIDsRouteConfig[Body],
) {
	handleReplaceStableIDsRoute(ginCtx, ctx, moduleName, replaceStableIDsHandlerConfig{
		invalidField: config.invalidField,
		readAndBindGenerated: func(ginCtx *gin.Context) ([]uint64, error) {
			_, ids, err := config.read(ginCtx)
			if err != nil {
				return nil, err
			}
			return ids, nil
		},
		write:         config.write,
		allowEmptyIDs: config.allowEmptyIDs,
		resultStatus:  config.resultStatus,
	})
}

// handleBatchStableIDsOperation 保持批量写入的请求解析与原子校验顺序。
func handleBatchStableIDsOperation[Body any](
	ginCtx *gin.Context,
	ctx *module.Context,
	moduleName string,
	config batchGeneratedStableIDsRouteConfig[Body],
) {
	handleBatchStableIDsRoute(ginCtx, ctx, moduleName, batchStableIDsHandlerConfig{
		invalidField: "role_ids",
		readAndBindGenerated: func(ginCtx *gin.Context) (batchStableIDSet, error) {
			_, request, err := config.read(ginCtx)
			if err != nil {
				return batchStableIDSet{}, err
			}
			return request, nil
		},
		write:             config.write,
		allowEmptyRoleIDs: config.allowEmptyRoleIDs,
		resultStatus:      config.resultStatus,
	})
}

const maxRBACAtomicBatchItems = 100

// invalidStableIDBatch 校验 RBAC 原子批次的边界与稳定 ID 合法性。
func invalidStableIDBatch(ids []uint64, allowEmpty bool) bool {
	return ids == nil || (!allowEmpty && len(ids) == 0) || len(ids) > maxRBACAtomicBatchItems || hasInvalidStableIDs(ids) || hasDuplicateStableIDs(ids)
}

func hasDuplicateStableIDs(ids []uint64) bool {
	seen := make(map[uint64]struct{}, len(ids))
	for _, id := range ids {
		if _, exists := seen[id]; exists {
			return true
		}
		seen[id] = struct{}{}
	}
	return false
}

// writeStableIDBatchResult 按请求顺序返回已原子提交的共享批次结果。
func writeStableIDBatchResult(
	ginCtx *gin.Context,
	requestedIDs []uint64,
	status stableIDBatchResultStatus,
) {
	results := make([]generated.DestructiveBatchResultItem, 0, len(requestedIDs))
	for _, id := range requestedIDs {
		results = append(results, generated.DestructiveBatchResultItem{
			Id:     strconv.FormatUint(id, 10),
			Status: generated.DestructiveBatchResultItemStatus(status),
		})
	}
	httpx.WriteSuccess(ginCtx, http.StatusOK, generated.DestructiveBatchResult{
		OperationId: httpx.EnsureRequestID(ginCtx),
		Summary: generated.DestructiveBatchResultSummary{
			Requested: len(results),
			Succeeded: len(results),
			Failed:    0,
		},
		Results: results,
	})
}
