package rbac

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	messagecontract "graft/server/internal/contract/message"
	"graft/server/internal/httpx"
	"graft/server/internal/plugin"
	"graft/server/internal/pluginapi"
	rbacstore "graft/server/plugins/rbac/store"
)

func handleListRoles(
	ctx *plugin.Context,
	pluginName string,
	reader readManagementService,
) gin.HandlerFunc {
	return newManagementListHandler(
		ctx,
		pluginName,
		"list roles failed",
		func(ginCtx *gin.Context) (roleListResponse, error) {
			roles, err := reader.ListRoles(ginCtx.Request.Context())
			if err != nil {
				return roleListResponse{}, err
			}

			return toRoleListResponse(roles), nil
		},
	)
}

func handleListRolePermissionBindings(
	ctx *plugin.Context,
	pluginName string,
	reader readManagementService,
) gin.HandlerFunc {
	return handleStableIDResponse(
		ctx,
		pluginName,
		"list role permission bindings failed",
		func(requestCtx context.Context, targetID uint64) (rolePermissionBindingResponse, error) {
			bindings, err := reader.ListRolePermissionBindings(requestCtx, targetID)
			if err != nil {
				return rolePermissionBindingResponse{}, err
			}

			return toRolePermissionBindingResponse(bindings), nil
		},
		func(err error) bool { return errors.Is(err, rbacstore.ErrRoleNotFound) },
		messagecontract.RoleNotFound,
	)
}

func handleListPermissions(
	ctx *plugin.Context,
	pluginName string,
	reader readManagementService,
) gin.HandlerFunc {
	return newManagementListHandler(
		ctx,
		pluginName,
		"list permissions failed",
		func(ginCtx *gin.Context) (permissionListResponse, error) {
			permissions, err := reader.ListPermissions(ginCtx.Request.Context())
			if err != nil {
				return permissionListResponse{}, err
			}

			return toPermissionListResponse(permissions), nil
		},
	)
}

func handleListUserRoleBindings(
	ctx *plugin.Context,
	pluginName string,
	reader readManagementService,
) gin.HandlerFunc {
	return handleStableIDResponse(
		ctx,
		pluginName,
		"list user-role bindings failed",
		func(requestCtx context.Context, targetID uint64) (userRoleBindingResponse, error) {
			roleIDs, err := reader.ListRoleIDsByUserID(requestCtx, targetID)
			if err != nil {
				return userRoleBindingResponse{}, err
			}

			return toUserRoleBindingResponse(roleIDs), nil
		},
		func(err error) bool { return errors.Is(err, pluginapi.ErrUserNotFound) },
		messagecontract.UserNotFound,
	)
}

func handleStableIDResponse[T any](
	ctx *plugin.Context,
	pluginName string,
	logMessage string,
	read func(requestCtx context.Context, targetID uint64) (T, error),
	isNotFound func(error) bool,
	notFoundKey messagecontract.Key,
) gin.HandlerFunc {
	return func(ginCtx *gin.Context) {
		targetID, err := parseManagementID(ginCtx.Param("id"))
		if err != nil {
			writeLocalizedContractError(ginCtx, ctx.I18n, http.StatusBadRequest, messagecontract.CommonInvalidArgument, map[string]any{
				"field": "id",
			})
			return
		}

		payload, err := read(ginCtx.Request.Context(), targetID)
		if err != nil {
			if isNotFound(err) {
				writeLocalizedContractError(ginCtx, ctx.I18n, http.StatusNotFound, notFoundKey, nil)
				return
			}

			ctx.Logger.Error(logMessage,
				zap.String("plugin", pluginName),
				zap.Uint64("targetId", targetID),
				zap.Error(err),
			)
			httpx.AbortLocalizedError(ginCtx, ctx.I18n, http.StatusInternalServerError, messagecontract.CommonInternalError.String(), nil)
			return
		}

		httpx.WriteSuccess(ginCtx, http.StatusOK, payload)
	}
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
