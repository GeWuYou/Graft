package rbac

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	httpheader "graft/server/internal/contract/httpheader"
	messagecontract "graft/server/internal/contract/message"
	rbacopenapi "graft/server/internal/contract/openapi/rbac"
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
	handler := rbacReadGeneratedHandler{}

	return newManagementListHandler(
		ctx,
		pluginName,
		"list roles failed",
		func(ginCtx *gin.Context) (roleListResponse, error) {
			handler.GetRoles(bindGeneratedRoleParams(ginCtx))

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
	handler := rbacReadGeneratedHandler{}

	return func(ginCtx *gin.Context) {
		targetID, err := parseManagementID(ginCtx.Param("id"))
		if err != nil {
			writeLocalizedContractError(ginCtx, ctx.I18n, http.StatusBadRequest, messagecontract.CommonInvalidArgument, map[string]any{
				"field": "id",
			})
			return
		}

		handler.GetRolePermissions(targetID, bindGeneratedRolePermissionParams(ginCtx))

		bindings, err := reader.ListRolePermissionBindings(ginCtx.Request.Context(), targetID)
		if err != nil {
			if errors.Is(err, rbacstore.ErrRoleNotFound) {
				writeLocalizedContractError(ginCtx, ctx.I18n, http.StatusNotFound, messagecontract.RoleNotFound, nil)
				return
			}

			ctx.Logger.Error("list role permission bindings failed",
				zap.String("plugin", pluginName),
				zap.Uint64("targetId", targetID),
				zap.Error(err),
			)
			httpx.AbortLocalizedError(ginCtx, ctx.I18n, http.StatusInternalServerError, messagecontract.CommonInternalError.String(), nil)
			return
		}

		httpx.WriteSuccess(ginCtx, http.StatusOK, toRolePermissionBindingResponse(bindings))
	}
}

func handleListPermissions(
	ctx *plugin.Context,
	pluginName string,
	reader readManagementService,
) gin.HandlerFunc {
	handler := rbacReadGeneratedHandler{}

	return func(ginCtx *gin.Context) {
		params := bindGeneratedPermissionParams(ginCtx)
		handler.GetPermissions(params)

		permissions, err := reader.ListPermissions(ginCtx.Request.Context())
		if err != nil {
			ctx.Logger.Error("list permissions failed",
				zap.String("plugin", pluginName),
				zap.Error(err),
			)
			httpx.AbortLocalizedError(ginCtx, ctx.I18n, http.StatusInternalServerError, messagecontract.CommonInternalError.String(), nil)
			return
		}

		httpx.WriteSuccess(ginCtx, http.StatusOK, toPermissionListResponse(permissions))
	}
}

type rbacReadGeneratedHandler struct {
}

func (h rbacReadGeneratedHandler) GetPermissions(params rbacopenapi.GetPermissionsParams) {
	_ = h
	_ = params
}

func (h rbacReadGeneratedHandler) GetRoles(params rbacopenapi.GetRolesParams) {
	_ = h
	_ = params
}

func (h rbacReadGeneratedHandler) GetRolePermissions(id uint64, params rbacopenapi.GetRolePermissionsParams) {
	_ = h
	_ = id
	_ = params
}

func bindGeneratedPermissionParams(ginCtx *gin.Context) rbacopenapi.GetPermissionsParams {
	locale, requestID := bindGeneratedReadHeaders(ginCtx)
	return rbacopenapi.GetPermissionsParams{
		XGraftLocale: locale,
		XRequestId:   requestID,
	}
}

func bindGeneratedRoleParams(ginCtx *gin.Context) rbacopenapi.GetRolesParams {
	locale, requestID := bindGeneratedReadHeaders(ginCtx)
	return rbacopenapi.GetRolesParams{
		XGraftLocale: locale,
		XRequestId:   requestID,
	}
}

func bindGeneratedRolePermissionParams(ginCtx *gin.Context) rbacopenapi.GetRolePermissionsParams {
	locale, requestID := bindGeneratedReadHeaders(ginCtx)
	return rbacopenapi.GetRolePermissionsParams{
		XGraftLocale: locale,
		XRequestId:   requestID,
	}
}

func bindGeneratedReadHeaders(ginCtx *gin.Context) (locale *string, requestID *string) {
	if raw := strings.TrimSpace(ginCtx.GetHeader(httpx.RequestIDHeader)); raw != "" {
		requestID = &raw
	}

	if raw := strings.TrimSpace(ginCtx.GetHeader(string(httpheader.Locale))); raw != "" {
		locale = &raw
	}
	return locale, requestID
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
