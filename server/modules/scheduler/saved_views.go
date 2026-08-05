package scheduler

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"graft/server/internal/httpx"
	"graft/server/internal/i18n"
	"graft/server/internal/module"
	"graft/server/internal/moduleapi"
	schedulercontract "graft/server/modules/scheduler/contract"
)

const schedulerListSavedViewSurface = "scheduled-task.list"

// schedulerSavedViewListResponse 是定时任务保存视图列表的显式 HTTP 响应 DTO。
// 它与 OpenAPI 的 saved-view-list-response wrapper 保持一致，避免匿名 map 成为隐式契约。
type schedulerSavedViewListResponse struct {
	Items []httpx.SavedViewResponse `json:"items"`
}

var schedulerListSavedViewColumns = map[string]struct{}{
	"task": {}, "job_key": {}, "schedule": {}, "status": {}, "last_run": {}, "success_rate": {},
}

var schedulerListSavedViewQueryFields = map[string]httpx.SavedViewQueryValueKind{
	"keyword": httpx.SavedViewQueryString,
	"jobKey":  httpx.SavedViewQueryString,
	"status":  httpx.SavedViewQueryString,
}

// registerSchedulerSavedViewRoutes 注册定时任务列表的用户私有保存视图 CRUD 路由。
func registerSchedulerSavedViewRoutes(group *gin.RouterGroup, ctx *module.Context, guard gin.HandlerFunc, service moduleapi.SavedViewService) {
	if group == nil || ctx == nil || service == nil {
		return
	}
	group.GET(schedulercontract.ScheduledTaskSavedViewsRoute, guard, handleSchedulerSavedViewList(ctx.I18n, service))
	group.POST(schedulercontract.ScheduledTaskSavedViewsRoute, guard, handleSchedulerSavedViewCreate(ctx.I18n, service))
	group.PUT(schedulercontract.ScheduledTaskSavedViewRoute, guard, handleSchedulerSavedViewUpdate(ctx.I18n, service))
	group.DELETE(schedulercontract.ScheduledTaskSavedViewRoute, guard, handleSchedulerSavedViewDelete(ctx.I18n, service))
}

func validateSchedulerSavedView(request httpx.SavedViewRequest) error {
	if strings.TrimSpace(request.Name) == "" || !schedulerSavedViewPageSize(request.PageSize) || !json.Valid(request.QueryState) {
		return moduleapi.ErrSavedViewInvalidInput
	}
	if !validSchedulerSavedViewQuery(request.QueryState) || !validSchedulerSavedViewColumns(request.VisibleColumns) {
		return moduleapi.ErrSavedViewInvalidInput
	}
	return nil
}

func validSchedulerSavedViewQuery(raw json.RawMessage) bool {
	return httpx.ValidateSavedViewQueryState(raw, schedulerListSavedViewQueryFields, func(ctx *gin.Context) string {
		status := strings.TrimSpace(ctx.Query("status"))
		if status != "" && status != "all" && status != "idle" && status != "running" && status != "success" && status != "failed" && status != "unknown" {
			return "invalid status"
		}
		return ""
	}) == nil
}

func validSchedulerSavedViewColumns(columns []string) bool {
	for _, column := range columns {
		if _, ok := schedulerListSavedViewColumns[strings.TrimSpace(column)]; !ok {
			return false
		}
	}
	return true
}

func schedulerSavedViewPageSize(size int) bool {
	return size == 10 || size == 20 || size == 50 || size == 100
}

func handleSchedulerSavedViewList(localizer *i18n.Service, service moduleapi.SavedViewService) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		owner, ok := httpx.SavedViewOwnerID(ctx)
		if !ok || service == nil {
			httpx.WriteSavedViewError(ctx, localizer, moduleapi.ErrSavedViewInvalidInput)
			return
		}
		views, err := service.List(ctx.Request.Context(), owner, schedulerListSavedViewSurface)
		if err != nil {
			httpx.WriteSavedViewError(ctx, localizer, err)
			return
		}
		writeSchedulerSavedViewList(ctx, localizer, views)
	}
}

func handleSchedulerSavedViewCreate(localizer *i18n.Service, service moduleapi.SavedViewService) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		owner, ok := httpx.SavedViewOwnerID(ctx)
		if !ok || service == nil {
			httpx.WriteSavedViewError(ctx, localizer, moduleapi.ErrSavedViewInvalidInput)
			return
		}
		request, ok := httpx.BindSavedViewRequest(ctx, localizer)
		if !ok {
			return
		}
		if err := validateSchedulerSavedView(request); err != nil {
			httpx.WriteSavedViewError(ctx, localizer, err)
			return
		}
		view, err := service.Create(ctx.Request.Context(), moduleapi.SavedViewCreateInput{OwnerUserID: owner, SurfaceKey: schedulerListSavedViewSurface, Name: request.Name, QueryState: request.QueryState, PageSize: request.PageSize, VisibleColumns: request.VisibleColumns, IsDefault: request.IsDefault})
		if err != nil {
			httpx.WriteSavedViewError(ctx, localizer, err)
			return
		}
		writeSchedulerSavedView(ctx, localizer, http.StatusCreated, view)
	}
}

func handleSchedulerSavedViewUpdate(localizer *i18n.Service, service moduleapi.SavedViewService) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		owner, ownerOK := httpx.SavedViewOwnerID(ctx)
		id, idOK := httpx.SavedViewID(ctx)
		if !ownerOK || !idOK || service == nil {
			httpx.WriteSavedViewError(ctx, localizer, moduleapi.ErrSavedViewInvalidInput)
			return
		}
		request, ok := httpx.BindSavedViewRequest(ctx, localizer)
		if !ok {
			return
		}
		if err := validateSchedulerSavedView(request); err != nil {
			httpx.WriteSavedViewError(ctx, localizer, err)
			return
		}
		view, err := service.Update(ctx.Request.Context(), moduleapi.SavedViewUpdateInput{ID: id, OwnerUserID: owner, SurfaceKey: schedulerListSavedViewSurface, Name: request.Name, QueryState: request.QueryState, PageSize: request.PageSize, VisibleColumns: request.VisibleColumns, IsDefault: request.IsDefault})
		if err != nil {
			httpx.WriteSavedViewError(ctx, localizer, err)
			return
		}
		writeSchedulerSavedView(ctx, localizer, http.StatusOK, view)
	}
}

func handleSchedulerSavedViewDelete(localizer *i18n.Service, service moduleapi.SavedViewService) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		owner, ownerOK := httpx.SavedViewOwnerID(ctx)
		id, idOK := httpx.SavedViewID(ctx)
		if !ownerOK || !idOK || service == nil {
			httpx.WriteSavedViewError(ctx, localizer, moduleapi.ErrSavedViewInvalidInput)
			return
		}
		if err := service.Delete(ctx.Request.Context(), owner, schedulerListSavedViewSurface, id); err != nil {
			httpx.WriteSavedViewError(ctx, localizer, err)
			return
		}
		ctx.Status(http.StatusNoContent)
	}
}

func writeSchedulerSavedViewList(ctx *gin.Context, localizer *i18n.Service, views []moduleapi.SavedView) {
	items := make([]httpx.SavedViewResponse, 0, len(views))
	for _, view := range views {
		item, err := httpx.ToSavedViewResponse(view)
		if err != nil {
			httpx.WriteSavedViewError(ctx, localizer, err)
			return
		}
		items = append(items, item)
	}
	httpx.WriteSuccess(ctx, http.StatusOK, schedulerSavedViewListResponse{Items: items})
}

func writeSchedulerSavedView(ctx *gin.Context, localizer *i18n.Service, status int, view moduleapi.SavedView) {
	item, err := httpx.ToSavedViewResponse(view)
	if err != nil {
		httpx.WriteSavedViewError(ctx, localizer, err)
		return
	}
	httpx.WriteSuccess(ctx, status, item)
}
