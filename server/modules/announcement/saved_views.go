package announcement

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"graft/server/internal/httpx"
	"graft/server/internal/i18n"
	"graft/server/internal/module"
	"graft/server/internal/moduleapi"
	announcementcontract "graft/server/modules/announcement/contract"
)

const announcementManagementSavedViewSurface = "announcement.management"

var announcementManagementSavedViewColumns = map[string]struct{}{
	"title": {}, "status": {}, "visibility": {}, "level": {}, "published_at": {}, "publish_at": {},
	"expire_at": {}, "delivery_mode": {}, "pinned": {}, "published_by": {}, "created_at": {},
	"updated_at": {}, "archived_at": {}, "operation": {},
}

var announcementManagementSavedViewQueryFields = map[string]httpx.SavedViewQueryValueKind{
	"keyword": httpx.SavedViewQueryString,
	"status":  httpx.SavedViewQueryString,
	"level":   httpx.SavedViewQueryString,
	"pinned":  httpx.SavedViewQueryBool,
	"sort":    httpx.SavedViewQueryString,
}

// validateAnnouncementManagementSavedView 校验公告管理列表保存视图的查询状态、分页大小和可见列。
func validateAnnouncementManagementSavedView(request httpx.SavedViewRequest, moduleCtx *module.Context) error {
	if strings.TrimSpace(request.Name) == "" || !announcementManagementSavedViewPageSize(request.PageSize) || !json.Valid(request.QueryState) {
		return moduleapi.ErrSavedViewInvalidInput
	}
	if !validAnnouncementManagementSavedViewQuery(request.QueryState, moduleCtx) || !validAnnouncementManagementSavedViewColumns(request.VisibleColumns) {
		return moduleapi.ErrSavedViewInvalidInput
	}
	return nil
}

func validAnnouncementManagementSavedViewQuery(raw json.RawMessage, moduleCtx *module.Context) bool {
	if err := httpx.ValidateSavedViewQueryState(raw, announcementManagementSavedViewQueryFields, func(ctx *gin.Context) string {
		_, ok := bindAdminListParams(ctx, moduleCtx)
		if !ok {
			return "invalid announcement query"
		}
		return ""
	}); err != nil {
		return false
	}
	var state struct {
		Status *string `json:"status"`
		Level  *string `json:"level"`
		Sort   *string `json:"sort"`
	}
	if err := json.Unmarshal(raw, &state); err != nil ||
		(state.Status != nil && !announcementStatusValue(*state.Status)) ||
		(state.Level != nil && !announcementLevelValue(*state.Level)) ||
		(state.Sort != nil && !announcementSortValue(*state.Sort)) {
		return false
	}
	return true
}

func validAnnouncementManagementSavedViewColumns(columns []string) bool {
	for _, column := range columns {
		if _, ok := announcementManagementSavedViewColumns[strings.TrimSpace(column)]; !ok {
			return false
		}
	}
	return true
}

func announcementStatusValue(value string) bool {
	return announcementcontract.ValidAnnouncementStatus(announcementcontract.AnnouncementStatus(strings.TrimSpace(value)))
}

func announcementLevelValue(value string) bool {
	return announcementcontract.ValidAnnouncementLevel(announcementcontract.AnnouncementLevel(strings.TrimSpace(value)))
}

func announcementSortValue(value string) bool {
	switch strings.TrimSpace(value) {
	case "updated_desc", "publish_desc", "pinned_publish_desc":
		return true
	default:
		return false
	}
}

func announcementManagementSavedViewPageSize(size int) bool {
	return size == 10 || size == 20 || size == 50 || size == 100
}

func registerAnnouncementSavedViewRoutes(group *gin.RouterGroup, moduleCtx *module.Context, service moduleapi.SavedViewService, guard gin.HandlerFunc) {
	if group == nil || moduleCtx == nil || service == nil {
		return
	}
	group.GET(announcementcontract.AnnouncementSavedViewsRoute, guard, handleAnnouncementSavedViewList(moduleCtx.I18n, service))
	group.POST(announcementcontract.AnnouncementSavedViewsRoute, guard, handleAnnouncementSavedViewCreate(moduleCtx.I18n, moduleCtx, service))
	group.PUT(announcementcontract.AnnouncementSavedViewRoute, guard, handleAnnouncementSavedViewUpdate(moduleCtx.I18n, moduleCtx, service))
	group.DELETE(announcementcontract.AnnouncementSavedViewRoute, guard, handleAnnouncementSavedViewDelete(moduleCtx.I18n, service))
}

func handleAnnouncementSavedViewList(localizer *i18n.Service, service moduleapi.SavedViewService) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		owner, ok := httpx.SavedViewOwnerID(ctx)
		if !ok {
			httpx.WriteSavedViewError(ctx, localizer, moduleapi.ErrSavedViewInvalidInput)
			return
		}
		views, err := service.List(ctx.Request.Context(), owner, announcementManagementSavedViewSurface)
		if err != nil {
			httpx.WriteSavedViewError(ctx, localizer, err)
			return
		}
		items := make([]httpx.SavedViewResponse, 0, len(views))
		for _, view := range views {
			item, mapErr := httpx.ToSavedViewResponse(view)
			if mapErr != nil {
				httpx.WriteSavedViewError(ctx, localizer, mapErr)
				return
			}
			items = append(items, item)
		}
		httpx.WriteSuccess(ctx, http.StatusOK, map[string]any{"items": items})
	}
}

func handleAnnouncementSavedViewCreate(localizer *i18n.Service, moduleCtx *module.Context, service moduleapi.SavedViewService) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		owner, ok := httpx.SavedViewOwnerID(ctx)
		if !ok {
			httpx.WriteSavedViewError(ctx, localizer, moduleapi.ErrSavedViewInvalidInput)
			return
		}
		request, ok := httpx.BindSavedViewRequest(ctx, localizer)
		if !ok {
			return
		}
		if err := validateAnnouncementManagementSavedView(request, moduleCtx); err != nil {
			httpx.WriteSavedViewError(ctx, localizer, err)
			return
		}
		view, err := service.Create(ctx.Request.Context(), moduleapi.SavedViewCreateInput{OwnerUserID: owner, SurfaceKey: announcementManagementSavedViewSurface, Name: request.Name, QueryState: request.QueryState, PageSize: request.PageSize, VisibleColumns: request.VisibleColumns, IsDefault: request.IsDefault})
		if err != nil {
			httpx.WriteSavedViewError(ctx, localizer, err)
			return
		}
		writeAnnouncementSavedView(ctx, localizer, http.StatusCreated, view)
	}
}

func handleAnnouncementSavedViewUpdate(localizer *i18n.Service, moduleCtx *module.Context, service moduleapi.SavedViewService) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		owner, ownerOK := httpx.SavedViewOwnerID(ctx)
		id, idOK := httpx.SavedViewID(ctx)
		if !ownerOK || !idOK {
			httpx.WriteSavedViewError(ctx, localizer, moduleapi.ErrSavedViewInvalidInput)
			return
		}
		request, ok := httpx.BindSavedViewRequest(ctx, localizer)
		if !ok {
			return
		}
		if err := validateAnnouncementManagementSavedView(request, moduleCtx); err != nil {
			httpx.WriteSavedViewError(ctx, localizer, err)
			return
		}
		view, err := service.Update(ctx.Request.Context(), moduleapi.SavedViewUpdateInput{ID: id, OwnerUserID: owner, SurfaceKey: announcementManagementSavedViewSurface, Name: request.Name, QueryState: request.QueryState, PageSize: request.PageSize, VisibleColumns: request.VisibleColumns, IsDefault: request.IsDefault})
		if err != nil {
			httpx.WriteSavedViewError(ctx, localizer, err)
			return
		}
		writeAnnouncementSavedView(ctx, localizer, http.StatusOK, view)
	}
}

func handleAnnouncementSavedViewDelete(localizer *i18n.Service, service moduleapi.SavedViewService) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		owner, ownerOK := httpx.SavedViewOwnerID(ctx)
		id, idOK := httpx.SavedViewID(ctx)
		if !ownerOK || !idOK {
			httpx.WriteSavedViewError(ctx, localizer, moduleapi.ErrSavedViewInvalidInput)
			return
		}
		if err := service.Delete(ctx.Request.Context(), owner, announcementManagementSavedViewSurface, id); err != nil {
			httpx.WriteSavedViewError(ctx, localizer, err)
			return
		}
		ctx.Status(http.StatusNoContent)
	}
}

func writeAnnouncementSavedView(ctx *gin.Context, localizer *i18n.Service, status int, view moduleapi.SavedView) {
	item, err := httpx.ToSavedViewResponse(view)
	if err != nil {
		httpx.WriteSavedViewError(ctx, localizer, err)
		return
	}
	httpx.WriteSuccess(ctx, status, item)
}
