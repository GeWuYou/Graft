package audit

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"graft/server/internal/httpx"
	"graft/server/internal/i18n"
	"graft/server/internal/module"
	"graft/server/internal/moduleapi"
)

const auditLogSavedViewSurface = "audit-log.list"

var auditLogSavedViewColumns = map[string]struct{}{
	"action": {}, "actor": {}, "resource": {}, "correlation": {}, "session_id": {}, "ip": {}, "result": {}, "risk": {}, "created_at": {},
}

var auditLogSavedViewQueryFields = map[string]httpx.SavedViewQueryValueKind{
	"actor_user_id": httpx.SavedViewQueryNumber, "keyword": httpx.SavedViewQueryString, "actor": httpx.SavedViewQueryString,
	"action": httpx.SavedViewQueryString, "preset": httpx.SavedViewQueryString, "visibility_scope": httpx.SavedViewQueryString,
	"business_category": httpx.SavedViewQueryString, "action_prefix": httpx.SavedViewQueryString, "action_prefixes": httpx.SavedViewQueryStringSlice,
	"action_keywords": httpx.SavedViewQueryStringSlice, "source": httpx.SavedViewQueryString, "resource_type": httpx.SavedViewQueryString,
	"resource_types": httpx.SavedViewQueryStringSlice, "resource_id": httpx.SavedViewQueryString, "resource_name": httpx.SavedViewQueryString,
	"request_path_prefixes": httpx.SavedViewQueryStringSlice, "request_id": httpx.SavedViewQueryString, "session_id": httpx.SavedViewQueryString,
	"result": httpx.SavedViewQueryString, "results": httpx.SavedViewQueryStringSlice, "risk_level": httpx.SavedViewQueryString,
	"risk_levels": httpx.SavedViewQueryStringSlice, "success": httpx.SavedViewQueryBool, "created_from": httpx.SavedViewQueryString,
	"created_to": httpx.SavedViewQueryString, "sort": httpx.SavedViewQueryStringSlice,
}

// validateAuditLogSavedView 校验审计日志保存视图的名称、分页大小、查询状态和可见列。
// 任一字段无效时返回 moduleapi.ErrSavedViewInvalidInput。
func validateAuditLogSavedView(request httpx.SavedViewRequest) error {
	if strings.TrimSpace(request.Name) == "" || request.PageSize < 1 || request.PageSize > maxPageSize || !json.Valid(request.QueryState) {
		return moduleapi.ErrSavedViewInvalidInput
	}
	if err := httpx.ValidateSavedViewQueryState(request.QueryState, auditLogSavedViewQueryFields, func(ctx *gin.Context) string {
		_, _, invalid := bindGeneratedAuditListParams(ctx)
		return invalid
	}); err != nil {
		return moduleapi.ErrSavedViewInvalidInput
	}
	for _, column := range request.VisibleColumns {
		if _, ok := auditLogSavedViewColumns[strings.TrimSpace(column)]; !ok {
			return moduleapi.ErrSavedViewInvalidInput
		}
	}
	return nil
}

// registerAuditSavedViewRoutes 在指定路由组上注册审计日志保存视图的 CRUD 路由。
func registerAuditSavedViewRoutes(group *gin.RouterGroup, ctx *module.Context, guard gin.HandlerFunc, service moduleapi.SavedViewService) {
	group.GET("/logs/saved-views", guard, handleListAuditSavedViews(ctx.I18n, service))
	group.POST("/logs/saved-views", guard, handleCreateAuditSavedView(ctx.I18n, service))
	group.PUT("/logs/saved-views/:viewId", guard, handleUpdateAuditSavedView(ctx.I18n, service))
	group.DELETE("/logs/saved-views/:viewId", guard, handleDeleteAuditSavedView(ctx.I18n, service))
}

// handleListAuditSavedViews 创建按当前所有者查询审计日志保存视图的处理器。
// 所有者或服务不可用时写入输入错误；服务错误直接向上返回。
func handleListAuditSavedViews(localizer *i18n.Service, service moduleapi.SavedViewService) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		owner, ok := httpx.SavedViewOwnerID(ctx)
		if !ok || service == nil {
			httpx.WriteSavedViewError(ctx, localizer, moduleapi.ErrSavedViewInvalidInput)
			return
		}
		views, err := service.List(ctx.Request.Context(), owner, auditLogSavedViewSurface)
		if err != nil {
			httpx.WriteSavedViewError(ctx, localizer, err)
			return
		}
		writeAuditSavedViewList(ctx, localizer, views)
	}
}

// handleCreateAuditSavedView 为当前所有者创建保存视图，并将创建结果写入响应。
func handleCreateAuditSavedView(localizer *i18n.Service, service moduleapi.SavedViewService) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		owner, ownerOK := httpx.SavedViewOwnerID(ctx)
		if !ownerOK {
			httpx.WriteSavedViewError(ctx, localizer, moduleapi.ErrSavedViewInvalidInput)
			return
		}
		request, requestOK := httpx.BindSavedViewRequest(ctx, localizer)
		if !requestOK {
			return
		}
		if service == nil {
			httpx.WriteSavedViewError(ctx, localizer, moduleapi.ErrSavedViewInvalidInput)
			return
		}
		if err := validateAuditLogSavedView(request); err != nil {
			httpx.WriteSavedViewError(ctx, localizer, err)
			return
		}
		view, err := service.Create(ctx.Request.Context(), moduleapi.SavedViewCreateInput{OwnerUserID: owner, SurfaceKey: auditLogSavedViewSurface, Name: request.Name, QueryState: request.QueryState, PageSize: request.PageSize, VisibleColumns: request.VisibleColumns})
		if err != nil {
			httpx.WriteSavedViewError(ctx, localizer, err)
			return
		}
		writeAuditSavedView(ctx, localizer, http.StatusCreated, view)
	}
}

// handleUpdateAuditSavedView 返回校验并更新当前所有者审计日志保存视图的处理器。
func handleUpdateAuditSavedView(localizer *i18n.Service, service moduleapi.SavedViewService) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		owner, ownerOK := httpx.SavedViewOwnerID(ctx)
		id, idOK := httpx.SavedViewID(ctx)
		if !ownerOK || !idOK {
			httpx.WriteSavedViewError(ctx, localizer, moduleapi.ErrSavedViewInvalidInput)
			return
		}
		request, requestOK := httpx.BindSavedViewRequest(ctx, localizer)
		if !requestOK {
			return
		}
		if service == nil {
			httpx.WriteSavedViewError(ctx, localizer, moduleapi.ErrSavedViewInvalidInput)
			return
		}
		if err := validateAuditLogSavedView(request); err != nil {
			httpx.WriteSavedViewError(ctx, localizer, err)
			return
		}
		view, err := service.Update(ctx.Request.Context(), moduleapi.SavedViewUpdateInput{ID: id, OwnerUserID: owner, SurfaceKey: auditLogSavedViewSurface, Name: request.Name, QueryState: request.QueryState, PageSize: request.PageSize, VisibleColumns: request.VisibleColumns})
		if err != nil {
			httpx.WriteSavedViewError(ctx, localizer, err)
			return
		}
		writeAuditSavedView(ctx, localizer, http.StatusOK, view)
	}
}

// handleDeleteAuditSavedView 返回删除审计日志保存视图的 HTTP 处理器。
// 删除成功时返回 204 No Content；输入无效或服务删除失败时返回相应错误。
func handleDeleteAuditSavedView(localizer *i18n.Service, service moduleapi.SavedViewService) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		owner, ownerOK := httpx.SavedViewOwnerID(ctx)
		id, idOK := httpx.SavedViewID(ctx)
		if !ownerOK || !idOK || service == nil {
			httpx.WriteSavedViewError(ctx, localizer, moduleapi.ErrSavedViewInvalidInput)
			return
		}
		if err := service.Delete(ctx.Request.Context(), owner, auditLogSavedViewSurface, id); err != nil {
			httpx.WriteSavedViewError(ctx, localizer, err)
			return
		}
		ctx.Status(http.StatusNoContent)
	}
}

// writeAuditSavedViewList 将保存的审计日志视图列表写入成功响应。
// 如果视图转换失败，则写入相应的错误响应。
func writeAuditSavedViewList(ctx *gin.Context, localizer *i18n.Service, views []moduleapi.SavedView) {
	items := make([]httpx.SavedViewResponse, 0, len(views))
	for _, view := range views {
		mapped, err := httpx.ToSavedViewResponse(view)
		if err != nil {
			httpx.WriteSavedViewError(ctx, localizer, err)
			return
		}
		items = append(items, mapped)
	}
	httpx.WriteSuccess(ctx, http.StatusOK, map[string]any{"items": items})
}

// writeAuditSavedView 将保存视图转换为响应格式，并写入成功或错误响应。
func writeAuditSavedView(ctx *gin.Context, localizer *i18n.Service, status int, view moduleapi.SavedView) {
	mapped, err := httpx.ToSavedViewResponse(view)
	if err != nil {
		httpx.WriteSavedViewError(ctx, localizer, err)
		return
	}
	httpx.WriteSuccess(ctx, status, mapped)
}
