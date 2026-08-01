package httpx

import (
	"context"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"graft/server/internal/i18n"
	"graft/server/internal/moduleapi"
)

const accessLogSavedViewSurface = "access-log.list"

var accessLogSavedViewColumns = map[string]struct{}{
	"started_at": {}, "method": {}, "path": {}, "status_code": {}, "duration_ms": {}, "user": {}, "request_id": {}, "client_ip": {}, "user_agent": {}, "occurred_at": {},
}

var accessLogSavedViewQueryFields = map[string]SavedViewQueryValueKind{
	"request_id": SavedViewQueryString, "trace_id": SavedViewQueryString, "keyword": SavedViewQueryString, "user_id": SavedViewQueryNumber,
	"username": SavedViewQueryString, "method": SavedViewQueryString, "path": SavedViewQueryString, "path_match": SavedViewQueryString,
	"route": SavedViewQueryString, "status_code": SavedViewQueryNumber, "status_group": SavedViewQueryStringSlice,
	"duration_min_ms": SavedViewQueryNumber, "duration_max_ms": SavedViewQueryNumber, "started_from": SavedViewQueryString,
	"started_to": SavedViewQueryString, "occurred_from": SavedViewQueryString, "occurred_to": SavedViewQueryString, "sort": SavedViewQueryStringSlice,
}

// validateAccessLogSavedView validates an access log saved view request and returns
// moduleapi.ErrSavedViewInvalidInput when any field is invalid.
func validateAccessLogSavedView(request SavedViewRequest) error {
	if strings.TrimSpace(request.Name) == "" || request.PageSize < 1 || request.PageSize > accessLogMaxPageSize {
		return moduleapi.ErrSavedViewInvalidInput
	}
	if err := ValidateSavedViewQueryState(request.QueryState, accessLogSavedViewQueryFields, func(ctx *gin.Context) string {
		_, invalid := bindAccessLogListQuery(ctx)
		return invalid
	}); err != nil {
		return moduleapi.ErrSavedViewInvalidInput
	}
	for _, column := range request.VisibleColumns {
		if _, ok := accessLogSavedViewColumns[strings.TrimSpace(column)]; !ok {
			return moduleapi.ErrSavedViewInvalidInput
		}
	}
	return nil
}

// listAccessLogSavedViews retrieves the access log saved views owned by the specified user.
// It returns moduleapi.ErrSavedViewInvalidInput when the service is nil or ownerID is zero.
func listAccessLogSavedViews(ctx context.Context, service moduleapi.SavedViewService, ownerID uint64) ([]moduleapi.SavedView, error) {
	if service == nil || ownerID == 0 {
		return nil, moduleapi.ErrSavedViewInvalidInput
	}
	return service.List(ctx, ownerID, accessLogSavedViewSurface)
}

// createAccessLogSavedView 创建并返回访问日志的已保存视图。
// 如果服务、所有者标识或请求无效，则返回相应错误。
//
// 参数 ownerID 指定已保存视图的所有者。
//
// 返回创建的已保存视图及错误。
func createAccessLogSavedView(ctx context.Context, service moduleapi.SavedViewService, ownerID uint64, request SavedViewRequest) (moduleapi.SavedView, error) {
	if service == nil || ownerID == 0 {
		return moduleapi.SavedView{}, moduleapi.ErrSavedViewInvalidInput
	}
	if err := validateAccessLogSavedView(request); err != nil {
		return moduleapi.SavedView{}, err
	}
	return service.Create(ctx, moduleapi.SavedViewCreateInput{OwnerUserID: ownerID, SurfaceKey: accessLogSavedViewSurface, Name: request.Name, QueryState: request.QueryState, PageSize: request.PageSize, VisibleColumns: request.VisibleColumns, IsDefault: request.IsDefault})
}

// updateAccessLogSavedView validates and updates an access log saved view.
// It returns an invalid-input error when the service, owner ID, or view ID is
// invalid, or when the request fails validation.
func updateAccessLogSavedView(ctx context.Context, service moduleapi.SavedViewService, ownerID, id uint64, request SavedViewRequest) (moduleapi.SavedView, error) {
	if service == nil || ownerID == 0 || id == 0 {
		return moduleapi.SavedView{}, moduleapi.ErrSavedViewInvalidInput
	}
	if err := validateAccessLogSavedView(request); err != nil {
		return moduleapi.SavedView{}, err
	}
	return service.Update(ctx, moduleapi.SavedViewUpdateInput{ID: id, OwnerUserID: ownerID, SurfaceKey: accessLogSavedViewSurface, Name: request.Name, QueryState: request.QueryState, PageSize: request.PageSize, VisibleColumns: request.VisibleColumns, IsDefault: request.IsDefault})
}

// registerAccessLogSavedViewRoutes registers the HTTP routes for managing access log saved views.
func registerAccessLogSavedViewRoutes(group *gin.RouterGroup, localizer *i18n.Service, guard gin.HandlerFunc, service moduleapi.SavedViewService) {
	group.GET("/saved-views", guard, handleListAccessLogSavedViews(localizer, service))
	group.POST("/saved-views", guard, handleCreateAccessLogSavedView(localizer, service))
	group.PUT("/saved-views/:viewId", guard, handleUpdateAccessLogSavedView(localizer, service))
	group.DELETE("/saved-views/:viewId", guard, handleDeleteAccessLogSavedView(localizer, service))
}

// handleListAccessLogSavedViews creates a handler that lists saved views for the
// authenticated owner and writes the result or an error response.
func handleListAccessLogSavedViews(localizer *i18n.Service, service moduleapi.SavedViewService) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		ownerID, ok := SavedViewOwnerID(ctx)
		if !ok {
			WriteSavedViewError(ctx, localizer, moduleapi.ErrSavedViewInvalidInput)
			return
		}
		views, err := listAccessLogSavedViews(ctx.Request.Context(), service, ownerID)
		if err != nil {
			WriteSavedViewError(ctx, localizer, err)
			return
		}
		writeSavedViewList(ctx, localizer, views)
	}
}

// handleCreateAccessLogSavedView creates a Gin handler that validates the owner,
// binds the saved view request, creates the view, and writes the created view
// response.
func handleCreateAccessLogSavedView(localizer *i18n.Service, service moduleapi.SavedViewService) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		ownerID, ok := SavedViewOwnerID(ctx)
		if !ok {
			WriteSavedViewError(ctx, localizer, moduleapi.ErrSavedViewInvalidInput)
			return
		}
		request, ok := BindSavedViewRequest(ctx, localizer)
		if !ok {
			return
		}
		view, err := createAccessLogSavedView(ctx.Request.Context(), service, ownerID, request)
		if err != nil {
			WriteSavedViewError(ctx, localizer, err)
			return
		}
		writeSavedView(ctx, localizer, http.StatusCreated, view)
	}
}

// handleUpdateAccessLogSavedView 创建更新访问日志已保存视图的 HTTP 处理器。
// 处理器从请求上下文获取所有者和视图标识，绑定并更新视图请求，并写入更新结果。
func handleUpdateAccessLogSavedView(localizer *i18n.Service, service moduleapi.SavedViewService) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		ownerID, ownerOK := SavedViewOwnerID(ctx)
		id, idOK := SavedViewID(ctx)
		if !ownerOK || !idOK {
			WriteSavedViewError(ctx, localizer, moduleapi.ErrSavedViewInvalidInput)
			return
		}
		request, ok := BindSavedViewRequest(ctx, localizer)
		if !ok {
			return
		}
		view, err := updateAccessLogSavedView(ctx.Request.Context(), service, ownerID, id, request)
		if err != nil {
			WriteSavedViewError(ctx, localizer, err)
			return
		}
		writeSavedView(ctx, localizer, http.StatusOK, view)
	}
}

// handleDeleteAccessLogSavedView 创建删除访问日志已保存视图的 Gin 处理器。
// 处理成功时返回 204 状态；输入无效或删除失败时写入相应错误响应。
func handleDeleteAccessLogSavedView(localizer *i18n.Service, service moduleapi.SavedViewService) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		ownerID, ownerOK := SavedViewOwnerID(ctx)
		id, idOK := SavedViewID(ctx)
		if !ownerOK || !idOK {
			WriteSavedViewError(ctx, localizer, moduleapi.ErrSavedViewInvalidInput)
			return
		}
		if service == nil {
			WriteSavedViewError(ctx, localizer, moduleapi.ErrSavedViewInvalidInput)
			return
		}
		if err := service.Delete(ctx.Request.Context(), ownerID, accessLogSavedViewSurface, id); err != nil {
			WriteSavedViewError(ctx, localizer, err)
			return
		}
		ctx.Status(http.StatusNoContent)
	}
}

// writeSavedViewList 将已保存视图转换为响应并写入成功响应；转换失败时写入错误响应。
func writeSavedViewList(ctx *gin.Context, localizer *i18n.Service, views []moduleapi.SavedView) {
	items := make([]SavedViewResponse, 0, len(views))
	for _, view := range views {
		mapped, err := ToSavedViewResponse(view)
		if err != nil {
			WriteSavedViewError(ctx, localizer, err)
			return
		}
		items = append(items, mapped)
	}
	WriteSuccess(ctx, http.StatusOK, map[string]any{"items": items})
}

// writeSavedView 将已保存视图转换为响应对象并写入 HTTP 响应；转换失败时写入相应错误。
func writeSavedView(ctx *gin.Context, localizer *i18n.Service, status int, view moduleapi.SavedView) {
	mapped, err := ToSavedViewResponse(view)
	if err != nil {
		WriteSavedViewError(ctx, localizer, err)
		return
	}
	WriteSuccess(ctx, status, mapped)
}
