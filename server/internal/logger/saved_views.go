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
	"component": httpx.SavedViewQueryString, "operation": httpx.SavedViewQueryString, "request_id": httpx.SavedViewQueryString,
	"trace_id": httpx.SavedViewQueryString, "keyword": httpx.SavedViewQueryString, "message": httpx.SavedViewQueryString,
	"error": httpx.SavedViewQueryString, "sort": httpx.SavedViewQueryStringSlice,
}

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

func registerAppLogSavedViewRoutes(group *gin.RouterGroup, localizer *i18n.Service, guard gin.HandlerFunc, service moduleapi.SavedViewService) {
	group.GET("/saved-views", guard, handleListAppLogSavedViews(localizer, service))
	group.POST("/saved-views", guard, handleCreateAppLogSavedView(localizer, service))
	group.PUT("/saved-views/:viewId", guard, handleUpdateAppLogSavedView(localizer, service))
	group.DELETE("/saved-views/:viewId", guard, handleDeleteAppLogSavedView(localizer, service))
}

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
		view, err := service.Create(ctx.Request.Context(), moduleapi.SavedViewCreateInput{OwnerUserID: owner, SurfaceKey: appLogSavedViewSurface, Name: request.Name, QueryState: request.QueryState, PageSize: request.PageSize, VisibleColumns: request.VisibleColumns})
		if err != nil {
			httpx.WriteSavedViewError(ctx, localizer, err)
			return
		}
		writeAppLogSavedView(ctx, localizer, http.StatusCreated, view)
	}
}

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
		view, err := service.Update(ctx.Request.Context(), moduleapi.SavedViewUpdateInput{ID: id, OwnerUserID: owner, SurfaceKey: appLogSavedViewSurface, Name: request.Name, QueryState: request.QueryState, PageSize: request.PageSize, VisibleColumns: request.VisibleColumns})
		if err != nil {
			httpx.WriteSavedViewError(ctx, localizer, err)
			return
		}
		writeAppLogSavedView(ctx, localizer, http.StatusOK, view)
	}
}

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

func writeAppLogSavedView(ctx *gin.Context, localizer *i18n.Service, status int, view moduleapi.SavedView) {
	mapped, err := httpx.ToSavedViewResponse(view)
	if err != nil {
		httpx.WriteSavedViewError(ctx, localizer, err)
		return
	}
	httpx.WriteSuccess(ctx, status, mapped)
}
