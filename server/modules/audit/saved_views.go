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

func registerAuditSavedViewRoutes(group *gin.RouterGroup, ctx *module.Context, guard gin.HandlerFunc, service moduleapi.SavedViewService) {
	group.GET("/logs/saved-views", guard, handleListAuditSavedViews(ctx.I18n, service))
	group.POST("/logs/saved-views", guard, handleCreateAuditSavedView(ctx.I18n, service))
	group.PUT("/logs/saved-views/:viewId", guard, handleUpdateAuditSavedView(ctx.I18n, service))
	group.DELETE("/logs/saved-views/:viewId", guard, handleDeleteAuditSavedView(ctx.I18n, service))
}

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

func handleCreateAuditSavedView(localizer *i18n.Service, service moduleapi.SavedViewService) gin.HandlerFunc {
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

func handleUpdateAuditSavedView(localizer *i18n.Service, service moduleapi.SavedViewService) gin.HandlerFunc {
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

func writeAuditSavedView(ctx *gin.Context, localizer *i18n.Service, status int, view moduleapi.SavedView) {
	mapped, err := httpx.ToSavedViewResponse(view)
	if err != nil {
		httpx.WriteSavedViewError(ctx, localizer, err)
		return
	}
	httpx.WriteSuccess(ctx, status, mapped)
}
