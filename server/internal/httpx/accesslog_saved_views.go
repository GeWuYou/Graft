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

func listAccessLogSavedViews(ctx context.Context, service moduleapi.SavedViewService, ownerID uint64) ([]moduleapi.SavedView, error) {
	if service == nil || ownerID == 0 {
		return nil, moduleapi.ErrSavedViewInvalidInput
	}
	return service.List(ctx, ownerID, accessLogSavedViewSurface)
}

func createAccessLogSavedView(ctx context.Context, service moduleapi.SavedViewService, ownerID uint64, request SavedViewRequest) (moduleapi.SavedView, error) {
	if service == nil || ownerID == 0 {
		return moduleapi.SavedView{}, moduleapi.ErrSavedViewInvalidInput
	}
	if err := validateAccessLogSavedView(request); err != nil {
		return moduleapi.SavedView{}, err
	}
	return service.Create(ctx, moduleapi.SavedViewCreateInput{OwnerUserID: ownerID, SurfaceKey: accessLogSavedViewSurface, Name: request.Name, QueryState: request.QueryState, PageSize: request.PageSize, VisibleColumns: request.VisibleColumns})
}

func updateAccessLogSavedView(ctx context.Context, service moduleapi.SavedViewService, ownerID, id uint64, request SavedViewRequest) (moduleapi.SavedView, error) {
	if service == nil || ownerID == 0 || id == 0 {
		return moduleapi.SavedView{}, moduleapi.ErrSavedViewInvalidInput
	}
	if err := validateAccessLogSavedView(request); err != nil {
		return moduleapi.SavedView{}, err
	}
	return service.Update(ctx, moduleapi.SavedViewUpdateInput{ID: id, OwnerUserID: ownerID, SurfaceKey: accessLogSavedViewSurface, Name: request.Name, QueryState: request.QueryState, PageSize: request.PageSize, VisibleColumns: request.VisibleColumns})
}

func registerAccessLogSavedViewRoutes(group *gin.RouterGroup, localizer *i18n.Service, guard gin.HandlerFunc, service moduleapi.SavedViewService) {
	group.GET("/saved-views", guard, handleListAccessLogSavedViews(localizer, service))
	group.POST("/saved-views", guard, handleCreateAccessLogSavedView(localizer, service))
	group.PUT("/saved-views/:viewId", guard, handleUpdateAccessLogSavedView(localizer, service))
	group.DELETE("/saved-views/:viewId", guard, handleDeleteAccessLogSavedView(localizer, service))
}

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

func writeSavedView(ctx *gin.Context, localizer *i18n.Service, status int, view moduleapi.SavedView) {
	mapped, err := ToSavedViewResponse(view)
	if err != nil {
		WriteSavedViewError(ctx, localizer, err)
		return
	}
	WriteSuccess(ctx, status, mapped)
}
