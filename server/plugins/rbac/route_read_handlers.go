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
	generated "graft/server/internal/contract/openapi/generated"
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
		func(ginCtx *gin.Context) (generated.RoleListResponse, error) {
			handler.GetRoles(bindGeneratedRoleParams(ginCtx))

			roles, err := reader.ListRoles(ginCtx.Request.Context())
			if err != nil {
				return generated.RoleListResponse{}, err
			}

			return toRoleListResponse(roles)
		},
	)
}

//nolint:dupl // Generated-operation wrappers intentionally stay parallel while read behavior is shared below.
func handleListRolePermissionBindings(
	ctx *plugin.Context,
	pluginName string,
	reader readManagementService,
) gin.HandlerFunc {
	handler := rbacReadGeneratedHandler{}

	return handleStableIDResponse(stableIDResponseHandlerConfig[generated.RolePermissionBindingResponse]{
		ctx:        ctx,
		pluginName: pluginName,
		logMessage: "list role permission bindings failed",
		bindGenerated: func(ginCtx *gin.Context, targetID uint64) {
			handler.GetRolePermissions(targetID, bindGeneratedRolePermissionParams(ginCtx))
		},
		read: func(requestCtx context.Context, targetID uint64) (generated.RolePermissionBindingResponse, error) {
			bindings, err := reader.ListRolePermissionBindings(requestCtx, targetID)
			if err != nil {
				return generated.RolePermissionBindingResponse{}, err
			}
			return toRolePermissionBindingResponse(bindings)
		},
		isNotFound:  func(err error) bool { return errors.Is(err, rbacstore.ErrRoleNotFound) },
		notFoundKey: messagecontract.RoleNotFound,
	})
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

		payload, mapErr := toPermissionListResponse(permissions)
		if mapErr != nil {
			ctx.Logger.Error("map permissions response failed",
				zap.String("plugin", pluginName),
				zap.Error(mapErr),
			)
			httpx.AbortLocalizedError(ginCtx, ctx.I18n, http.StatusInternalServerError, messagecontract.CommonInternalError.String(), nil)
			return
		}

		httpx.WriteSuccess(ginCtx, http.StatusOK, payload)
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

//nolint:dupl // Generated-operation wrappers intentionally stay parallel while read behavior is shared below.
func handleListUserRoleBindings(
	ctx *plugin.Context,
	pluginName string,
	reader readManagementService,
) gin.HandlerFunc {
	handler := rbacUserRoleGeneratedHandler{}

	return handleStableIDResponse(stableIDResponseHandlerConfig[generated.UserRoleBindingResponse]{
		ctx:        ctx,
		pluginName: pluginName,
		logMessage: "list user-role bindings failed",
		bindGenerated: func(ginCtx *gin.Context, targetID uint64) {
			handler.GetUserRoles(targetID, bindGeneratedUserRoleReadParams(ginCtx))
		},
		read: func(requestCtx context.Context, targetID uint64) (generated.UserRoleBindingResponse, error) {
			roleIDs, err := reader.ListRoleIDsByUserID(requestCtx, targetID)
			if err != nil {
				return generated.UserRoleBindingResponse{}, err
			}
			return toUserRoleBindingResponse(roleIDs)
		},
		isNotFound:  func(err error) bool { return errors.Is(err, pluginapi.ErrUserNotFound) },
		notFoundKey: messagecontract.UserNotFound,
	})
}

type rbacUserRoleGeneratedHandler struct {
}

func (h rbacUserRoleGeneratedHandler) GetUserRoles(id uint64, params rbacopenapi.GetUserRolesParams) {
	_ = h
	_ = id
	_ = params
}

func bindGeneratedUserRoleReadParams(ginCtx *gin.Context) rbacopenapi.GetUserRolesParams {
	locale, requestID := bindGeneratedReadHeaders(ginCtx)
	return rbacopenapi.GetUserRolesParams{
		XGraftLocale: locale,
		XRequestId:   requestID,
	}
}

func bindGeneratedUserRoleAssignParams(ginCtx *gin.Context) rbacopenapi.PostUserRolesAssignParams {
	locale, requestID := bindGeneratedReadHeaders(ginCtx)
	return rbacopenapi.PostUserRolesAssignParams{
		XGraftLocale: locale,
		XRequestId:   requestID,
	}
}

type stableIDResponseHandlerConfig[T any] struct {
	ctx           *plugin.Context
	pluginName    string
	logMessage    string
	bindGenerated func(ginCtx *gin.Context, targetID uint64)
	read          func(requestCtx context.Context, targetID uint64) (T, error)
	isNotFound    func(error) bool
	notFoundKey   messagecontract.Key
}

func handleStableIDResponse[T any](config stableIDResponseHandlerConfig[T]) gin.HandlerFunc {
	return func(ginCtx *gin.Context) {
		targetID, ok := readManagementTargetID(ginCtx, config.ctx)
		if !ok {
			return
		}

		config.bindGenerated(ginCtx, targetID)

		payload, err := config.read(ginCtx.Request.Context(), targetID)
		if err != nil {
			if config.isNotFound(err) {
				writeLocalizedContractError(ginCtx, config.ctx.I18n, http.StatusNotFound, config.notFoundKey, nil)
				return
			}

			config.ctx.Logger.Error(config.logMessage,
				zap.String("plugin", config.pluginName),
				zap.Uint64("targetId", targetID),
				zap.Error(err),
			)
			httpx.AbortLocalizedError(ginCtx, config.ctx.I18n, http.StatusInternalServerError, messagecontract.CommonInternalError.String(), nil)
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

func readManagementTargetID(ginCtx *gin.Context, ctx *plugin.Context) (uint64, bool) {
	targetID, err := parseManagementID(ginCtx.Param("id"))
	if err != nil {
		writeLocalizedContractError(ginCtx, ctx.I18n, http.StatusBadRequest, messagecontract.CommonInvalidArgument, map[string]any{
			"field": "id",
		})
		return 0, false
	}

	return targetID, true
}
