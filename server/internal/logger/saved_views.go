package logger

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"graft/server/internal/httpx"
	"graft/server/internal/i18n"
	"graft/server/internal/moduleapi"
)

const appLogSavedViewSurface = "app-log.list"

var appLogSavedViewColumns = map[string]struct{}{
	"occurred_at": {}, "severity": {}, "component": {}, "operation": {}, "message": {}, "correlation": {}, "request_id": {}, "fields": {},
}

var appLogSavedViewQueryFields = map[string]httpx.SavedViewQueryValueKind{
	"occurred_from": httpx.SavedViewQueryString, "occurred_to": httpx.SavedViewQueryString, "severity": httpx.SavedViewQueryString,
	"category":  httpx.SavedViewQueryString,
	"component": httpx.SavedViewQueryString, "operation": httpx.SavedViewQueryString, "request_id": httpx.SavedViewQueryString,
	"trace_id": httpx.SavedViewQueryString, "keyword": httpx.SavedViewQueryString, "message": httpx.SavedViewQueryString,
	"error": httpx.SavedViewQueryString, "sort": httpx.SavedViewQueryStringSlice,
}

// JSON 且符合应用日志查询字段定义，所有可见列必须属于允许的列集合。
func validateAppLogSavedView(request httpx.SavedViewRequest) error {
	if strings.TrimSpace(request.Name) == "" || request.PageSize < 1 || request.PageSize > appLogMaxPageSize || !json.Valid(request.QueryState) {
		return moduleapi.ErrSavedViewInvalidInput
	}
	if err := httpx.ValidateSavedViewQueryState(request.QueryState, appLogSavedViewQueryFields, func(ctx *gin.Context) string {
		_, invalid := bindAppLogListQuery(ctx)
		return invalid
	}); err != nil {
		return moduleapi.ErrSavedViewInvalidInput
	}
	for _, column := range request.VisibleColumns {
		if _, ok := appLogSavedViewColumns[strings.TrimSpace(column)]; !ok {
			return moduleapi.ErrSavedViewInvalidInput
		}
	}
	return nil
}

// registerAppLogSavedViewRoutes 为应用日志已保存视图注册列表、创建、更新和删除接口。
func registerAppLogSavedViewRoutes(group *gin.RouterGroup, localizer *i18n.Service, guard gin.HandlerFunc, service moduleapi.SavedViewService) {
	group.GET("/saved-views", guard, handleListAppLogSavedViews(localizer, service))
	group.POST("/saved-views", guard, handleCreateAppLogSavedView(localizer, service))
	group.PUT("/saved-views/:viewId", guard, handleUpdateAppLogSavedView(localizer, service))
	group.DELETE("/saved-views/:viewId", guard, handleDeleteAppLogSavedView(localizer, service))
}

// handleListAppLogSavedViews 返回处理应用日志已保存视图列表请求的 Gin 处理函数。
// 处理函数会按当前用户和应用日志视图 surface 查询已保存视图，并写入列表响应。
func handleListAppLogSavedViews(localizer *i18n.Service, service moduleapi.SavedViewService) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		owner, ok := httpx.SavedViewOwnerID(ctx)
		if !ok || service == nil {
			httpx.WriteSavedViewError(ctx, localizer, moduleapi.ErrSavedViewInvalidInput)
			return
		}
		views, err := service.List(ctx.Request.Context(), owner, appLogSavedViewSurface)
		if err != nil {
			httpx.WriteSavedViewError(ctx, localizer, err)
			return
		}
		writeAppLogSavedViewList(ctx, localizer, views)
	}
}

// handleCreateAppLogSavedView 创建应用日志的已保存视图，并返回创建后的视图。
// 请求参数或保存操作失败时写入相应的错误响应。
func handleCreateAppLogSavedView(localizer *i18n.Service, service moduleapi.SavedViewService) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		owner, ownerOK := httpx.SavedViewOwnerID(ctx)
		request, requestOK := httpx.BindSavedViewRequest(ctx, localizer)
		if !ownerOK || !requestOK || service == nil {
			if ownerOK && !requestOK {
				return
			}
			httpx.WriteSavedViewError(ctx, localizer, moduleapi.ErrSavedViewInvalidInput)
			return
		}
		if err := validateAppLogSavedView(request); err != nil {
			httpx.WriteSavedViewError(ctx, localizer, err)
			return
		}
		view, err := service.Create(ctx.Request.Context(), moduleapi.SavedViewCreateInput{OwnerUserID: owner, SurfaceKey: appLogSavedViewSurface, Name: request.Name, QueryState: request.QueryState, PageSize: request.PageSize, VisibleColumns: request.VisibleColumns, IsDefault: request.IsDefault})
		if err != nil {
			httpx.WriteSavedViewError(ctx, localizer, err)
			return
		}
		writeAppLogSavedView(ctx, localizer, http.StatusCreated, view)
	}
}

// handleUpdateAppLogSavedView 创建用于更新应用日志已保存视图的 HTTP 处理器。
// 处理器校验请求参数并更新指定视图，成功时返回更新后的视图；参数、请求或服务操作失败时返回相应错误。
func handleUpdateAppLogSavedView(localizer *i18n.Service, service moduleapi.SavedViewService) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		owner, ownerOK := httpx.SavedViewOwnerID(ctx)
		id, idOK := httpx.SavedViewID(ctx)
		request, requestOK := httpx.BindSavedViewRequest(ctx, localizer)
		if !ownerOK || !idOK || !requestOK || service == nil {
			if ownerOK && idOK && !requestOK {
				return
			}
			httpx.WriteSavedViewError(ctx, localizer, moduleapi.ErrSavedViewInvalidInput)
			return
		}
		if err := validateAppLogSavedView(request); err != nil {
			httpx.WriteSavedViewError(ctx, localizer, err)
			return
		}
		view, err := service.Update(ctx.Request.Context(), moduleapi.SavedViewUpdateInput{ID: id, OwnerUserID: owner, SurfaceKey: appLogSavedViewSurface, Name: request.Name, QueryState: request.QueryState, PageSize: request.PageSize, VisibleColumns: request.VisibleColumns, IsDefault: request.IsDefault})
		if err != nil {
			httpx.WriteSavedViewError(ctx, localizer, err)
			return
		}
		writeAppLogSavedView(ctx, localizer, http.StatusOK, view)
	}
}

// handleDeleteAppLogSavedView handles requests to delete an app-log saved view.
func handleDeleteAppLogSavedView(localizer *i18n.Service, service moduleapi.SavedViewService) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		owner, ownerOK := httpx.SavedViewOwnerID(ctx)
		id, idOK := httpx.SavedViewID(ctx)
		if !ownerOK || !idOK || service == nil {
			httpx.WriteSavedViewError(ctx, localizer, moduleapi.ErrSavedViewInvalidInput)
			return
		}
		if err := service.Delete(ctx.Request.Context(), owner, appLogSavedViewSurface, id); err != nil {
			httpx.WriteSavedViewError(ctx, localizer, err)
			return
		}
		ctx.Status(http.StatusNoContent)
	}
}

// writeAppLogSavedViewList 将保存视图列表转换为响应数据并写入成功响应；转换失败时写入错误响应。
func writeAppLogSavedViewList(ctx *gin.Context, localizer *i18n.Service, views []moduleapi.SavedView) {
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

// writeAppLogSavedView converts a saved view to its HTTP response representation and writes it with the specified status code.
func writeAppLogSavedView(ctx *gin.Context, localizer *i18n.Service, status int, view moduleapi.SavedView) {
	mapped, err := httpx.ToSavedViewResponse(view)
	if err != nil {
		httpx.WriteSavedViewError(ctx, localizer, err)
		return
	}
	httpx.WriteSuccess(ctx, status, mapped)
}
