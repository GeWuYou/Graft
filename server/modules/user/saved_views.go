package user

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"graft/server/internal/httpx"
	"graft/server/internal/i18n"
	"graft/server/internal/module"
	"graft/server/internal/moduleapi"
	usercontract "graft/server/modules/user/contract"
)

const userListSavedViewSurface = "user.list"

var userSavedViewColumns = map[string]struct{}{
	"user": {}, "status": {}, "roles": {}, "last_login_at": {}, "created_at": {}, "updated_at": {}, "operation": {},
}

type userSavedViewListResponse struct {
	Items []httpx.SavedViewResponse `json:"items"`
}

func registerUserSavedViews(group *gin.RouterGroup, moduleCtx *module.Context, service moduleapi.SavedViewService, guard gin.HandlerFunc) {
	if group == nil || moduleCtx == nil || service == nil {
		return
	}
	group.GET(usercontract.UserSavedViewsRoute, guard, handleUserSavedViewList(moduleCtx.I18n, service))
	group.POST(usercontract.UserSavedViewsRoute, guard, handleUserSavedViewCreate(moduleCtx.I18n, service))
	group.PUT(usercontract.UserSavedViewRoute, guard, handleUserSavedViewUpdate(moduleCtx.I18n, service))
	group.DELETE(usercontract.UserSavedViewRoute, guard, handleUserSavedViewDelete(moduleCtx.I18n, service))
}

func validateUserSavedView(request httpx.SavedViewRequest) error {
	if strings.TrimSpace(request.Name) == "" || !userSavedViewPageSize(request.PageSize) || !json.Valid(request.QueryState) {
		return moduleapi.ErrSavedViewInvalidInput
	}
	if err := httpx.ValidateSavedViewQueryState(request.QueryState, map[string]httpx.SavedViewQueryValueKind{
		"keyword": httpx.SavedViewQueryString, "status": httpx.SavedViewQueryString, "roleId": httpx.SavedViewQueryNumber,
	}, func(ctx *gin.Context) string {
		status := ctx.Query("status")
		if status != "" && status != "enabled" && status != "disabled" {
			return "invalid status"
		}
		return ""
	}); err != nil {
		return moduleapi.ErrSavedViewInvalidInput
	}
	for _, column := range request.VisibleColumns {
		if _, ok := userSavedViewColumns[strings.TrimSpace(column)]; !ok {
			return moduleapi.ErrSavedViewInvalidInput
		}
	}
	return nil
}

func userSavedViewPageSize(size int) bool {
	return size == 10 || size == 20 || size == 50 || size == 100
}

func handleUserSavedViewList(localizer *i18n.Service, service moduleapi.SavedViewService) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		owner, ok := httpx.SavedViewOwnerID(ctx)
		if !ok {
			httpx.WriteSavedViewError(ctx, localizer, moduleapi.ErrSavedViewInvalidInput)
			return
		}
		views, err := service.List(ctx.Request.Context(), owner, userListSavedViewSurface)
		if err != nil {
			httpx.WriteSavedViewError(ctx, localizer, err)
			return
		}
		items := make([]httpx.SavedViewResponse, 0, len(views))
		for _, view := range views {
			item, err := httpx.ToSavedViewResponse(view)
			if err != nil {
				httpx.WriteSavedViewError(ctx, localizer, err)
				return
			}
			items = append(items, item)
		}
		httpx.WriteSuccess(ctx, http.StatusOK, userSavedViewListResponse{Items: items})
	}
}

func handleUserSavedViewCreate(localizer *i18n.Service, service moduleapi.SavedViewService) gin.HandlerFunc {
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
		if err := validateUserSavedView(request); err != nil {
			httpx.WriteSavedViewError(ctx, localizer, err)
			return
		}
		view, err := service.Create(ctx.Request.Context(), moduleapi.SavedViewCreateInput{OwnerUserID: owner, SurfaceKey: userListSavedViewSurface, Name: request.Name, QueryState: request.QueryState, PageSize: request.PageSize, VisibleColumns: request.VisibleColumns, IsDefault: request.IsDefault})
		if err != nil {
			httpx.WriteSavedViewError(ctx, localizer, err)
			return
		}
		writeUserSavedView(ctx, localizer, http.StatusCreated, view)
	}
}

func handleUserSavedViewUpdate(localizer *i18n.Service, service moduleapi.SavedViewService) gin.HandlerFunc {
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
		if err := validateUserSavedView(request); err != nil {
			httpx.WriteSavedViewError(ctx, localizer, err)
			return
		}
		view, err := service.Update(ctx.Request.Context(), moduleapi.SavedViewUpdateInput{ID: id, OwnerUserID: owner, SurfaceKey: userListSavedViewSurface, Name: request.Name, QueryState: request.QueryState, PageSize: request.PageSize, VisibleColumns: request.VisibleColumns, IsDefault: request.IsDefault})
		if err != nil {
			httpx.WriteSavedViewError(ctx, localizer, err)
			return
		}
		writeUserSavedView(ctx, localizer, http.StatusOK, view)
	}
}

func handleUserSavedViewDelete(localizer *i18n.Service, service moduleapi.SavedViewService) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		owner, ownerOK := httpx.SavedViewOwnerID(ctx)
		id, idOK := httpx.SavedViewID(ctx)
		if !ownerOK || !idOK {
			httpx.WriteSavedViewError(ctx, localizer, moduleapi.ErrSavedViewInvalidInput)
			return
		}
		if err := service.Delete(ctx.Request.Context(), owner, userListSavedViewSurface, id); err != nil {
			httpx.WriteSavedViewError(ctx, localizer, err)
			return
		}
		ctx.Status(http.StatusNoContent)
	}
}

func writeUserSavedView(ctx *gin.Context, localizer *i18n.Service, status int, view moduleapi.SavedView) {
	item, err := httpx.ToSavedViewResponse(view)
	if err != nil {
		httpx.WriteSavedViewError(ctx, localizer, err)
		return
	}
	httpx.WriteSuccess(ctx, status, item)
}
