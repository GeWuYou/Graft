package audit

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	messagecontract "graft/server/internal/contract/message"
	auditopenapi "graft/server/internal/contract/openapi/audit"
	"graft/server/internal/httpx"
	"graft/server/internal/module"
	"graft/server/internal/moduleapi"
	auditstore "graft/server/modules/audit/store"
)

// handleReadAuditVisibilityPolicy 读取指定模块的审计可见性策略。
func handleReadAuditVisibilityPolicy(
	ctx *module.Context,
	moduleName string,
	reader auditReader,
) gin.HandlerFunc {
	logger := auditRouteLogger(ctx)

	return func(ginCtx *gin.Context) {
		result, err := reader.VisibilityPolicy(withAuditRequestLocale(ginCtx, ctx))
		if err != nil {
			logger.Error("read audit visibility policy failed", zap.String("module", moduleName), zap.Error(err))
			httpx.AbortLocalizedError(ginCtx, ctx.I18n, http.StatusInternalServerError, messagecontract.CommonInternalError.String(), nil)
			return
		}

		payload, mapErr := toAuditVisibilityPolicyResponse(result)
		if mapErr != nil {
			logger.Error("map audit visibility policy failed", zap.String("module", moduleName), zap.Error(mapErr))
			httpx.AbortLocalizedError(ginCtx, ctx.I18n, http.StatusInternalServerError, messagecontract.CommonInternalError.String(), nil)
			return
		}

		httpx.WriteSuccess(ginCtx, http.StatusOK, payload)
	}
}

// handleUpdateAuditVisibilityDefault 处理审计可见性默认策略的更新请求。
func handleUpdateAuditVisibilityDefault(
	ctx *module.Context,
	moduleName string,
	reader auditReader,
) gin.HandlerFunc {
	logger := auditRouteLogger(ctx)

	return func(ginCtx *gin.Context) {
		var request auditopenapi.PutAuditVisibilityPolicyJSONRequestBody
		if err := ginCtx.ShouldBindJSON(&request); err != nil {
			httpx.AbortLocalizedError(ginCtx, ctx.I18n, http.StatusBadRequest, messagecontract.CommonInvalidArgument.String(), map[string]any{
				"field": "body",
			})
			return
		}

		userID, username := currentAuditActor(ginCtx)
		item, err := reader.UpdateVisibilityDefault(
			withAuditRequestLocale(ginCtx, ctx),
			auditstore.AuditVisibilityStrategy(strings.TrimSpace(string(request.Strategy))),
			userID,
			username,
		)
		if err != nil {
			handleAuditVisibilityWriteError(ginCtx, ctx, logger, moduleName, err)
			return
		}

		payload, mapErr := toAuditVisibilityDefaultResponse(item)
		if mapErr != nil {
			logger.Error("map audit visibility default failed", zap.String("module", moduleName), zap.Error(mapErr))
			httpx.AbortLocalizedError(ginCtx, ctx.I18n, http.StatusInternalServerError, messagecontract.CommonInternalError.String(), nil)
			return
		}

		httpx.WriteSuccess(ginCtx, http.StatusOK, payload)
	}
}

// handleUpsertAuditVisibilityOverride 处理审计可见性覆盖的新增或更新请求。
// 它会绑定请求体，提取当前审计操作者，调用读写器保存覆盖配置，并在成功时返回映射后的结果。
func handleUpsertAuditVisibilityOverride(
	ctx *module.Context,
	moduleName string,
	reader auditReader,
) gin.HandlerFunc {
	logger := auditRouteLogger(ctx)

	return func(ginCtx *gin.Context) {
		var request auditopenapi.PutAuditVisibilityOverrideJSONRequestBody
		if err := ginCtx.ShouldBindJSON(&request); err != nil {
			httpx.AbortLocalizedError(ginCtx, ctx.I18n, http.StatusBadRequest, messagecontract.CommonInvalidArgument.String(), map[string]any{
				"field": "body",
			})
			return
		}

		userID, username := currentAuditActor(ginCtx)
		description := ""
		if request.Description != nil {
			description = *request.Description
		}
		item, err := reader.UpdateVisibilityOverride(
			withAuditRequestLocale(ginCtx, ctx),
			auditstore.UpsertAuditVisibilityOverrideInput{
				Source:      auditstore.AuditSource(strings.TrimSpace(string(request.Source))),
				ActionKey:   request.ActionKey,
				Strategy:    auditstore.AuditVisibilityStrategy(strings.TrimSpace(string(request.Strategy))),
				Description: description,
				Actor: auditstore.AuditVisibilityActor{
					UserID:   userID,
					Username: username,
				},
			},
		)
		if err != nil {
			handleAuditVisibilityWriteError(ginCtx, ctx, logger, moduleName, err)
			return
		}

		payload, mapErr := toAuditVisibilityOverrideResponse(item)
		if mapErr != nil {
			logger.Error("map audit visibility override failed", zap.String("module", moduleName), zap.Error(mapErr))
			httpx.AbortLocalizedError(ginCtx, ctx.I18n, http.StatusInternalServerError, messagecontract.CommonInternalError.String(), nil)
			return
		}

		httpx.WriteSuccess(ginCtx, http.StatusOK, payload)
	}
}

// handleDeleteAuditVisibilityOverride 删除指定源和操作键的审计可见性覆盖配置。
// 当 source 或 action_key 缺失时返回本地化的 400 InvalidArgument；删除成功后返回空对象。
func handleDeleteAuditVisibilityOverride(
	ctx *module.Context,
	moduleName string,
	reader auditReader,
) gin.HandlerFunc {
	logger := auditRouteLogger(ctx)

	return func(ginCtx *gin.Context) {
		source := strings.TrimSpace(ginCtx.Query("source"))
		actionKey := strings.TrimSpace(ginCtx.Query("action_key"))
		if source == "" || actionKey == "" {
			field := "source"
			if actionKey == "" {
				field = "action_key"
			}
			httpx.AbortLocalizedError(ginCtx, ctx.I18n, http.StatusBadRequest, messagecontract.CommonInvalidArgument.String(), map[string]any{
				"field": field,
			})
			return
		}

		if err := reader.DeleteVisibilityOverride(
			withAuditRequestLocale(ginCtx, ctx),
			auditstore.AuditSource(source),
			actionKey,
		); err != nil {
			handleAuditVisibilityWriteError(ginCtx, ctx, logger, moduleName, err)
			return
		}

		httpx.WriteSuccess(ginCtx, http.StatusOK, map[string]any{})
	}
}

// currentAuditActor 从请求上下文中提取当前审计操作者的 ID 和用户名。
//
// 当 Gin 上下文、请求对象或认证信息缺失时，返回 nil 和空字符串。
// 用户名优先使用 DisplayName；如果为空，则回退到 Username。
func currentAuditActor(ginCtx *gin.Context) (*uint64, string) {
	if ginCtx == nil || ginCtx.Request == nil {
		return nil, ""
	}
	auth, ok := moduleapi.RequestAuthContextFromContext(ginCtx.Request.Context())
	if !ok || auth.User == nil {
		return nil, ""
	}
	userID := auth.User.ID
	username := strings.TrimSpace(auth.User.DisplayName)
	if username == "" {
		username = strings.TrimSpace(auth.User.Username)
	}
	return &userID, username
}

// handleAuditVisibilityWriteError 根据写入错误返回对应的本地化 HTTP 错误响应。
// 当错误属于审计可见性校验错误时，返回 400；否则记录日志并返回 500。
func handleAuditVisibilityWriteError(
	ginCtx *gin.Context,
	ctx *module.Context,
	logger *zap.Logger,
	moduleName string,
	err error,
) {
	if errors.Is(err, auditstore.ErrAuditValidation) {
		httpx.AbortLocalizedError(ginCtx, ctx.I18n, http.StatusBadRequest, messagecontract.CommonInvalidArgument.String(), nil)
		return
	}

	logger.Error("write audit visibility policy failed", zap.String("module", moduleName), zap.Error(err))
	httpx.AbortLocalizedError(ginCtx, ctx.I18n, http.StatusInternalServerError, messagecontract.CommonInternalError.String(), nil)
}

// auditRouteLogger 返回审计路由使用的日志器。
// 当提供了有效的上下文且其中包含日志器时，返回该日志器；否则返回一个空日志器。
func auditRouteLogger(ctx *module.Context) *zap.Logger {
	if ctx != nil && ctx.Logger != nil {
		return ctx.Logger
	}
	return zap.NewNop()
}
