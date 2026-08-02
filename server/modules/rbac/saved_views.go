package rbac

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"graft/server/internal/httpx"
	"graft/server/internal/i18n"
	"graft/server/internal/module"
	"graft/server/internal/moduleapi"
	rbaccontract "graft/server/modules/rbac/contract"
)

type rbacSavedViewDefinition struct {
	surface         string
	fields          map[string]httpx.SavedViewQueryValueKind
	columns         map[string]struct{}
	collectionRoute string
	itemRoute       string
}

var (
	roleSavedViewDefinition = rbacSavedViewDefinition{
		surface: "role.list", collectionRoute: rbaccontract.RoleSavedViewsRoute, itemRoute: rbaccontract.RoleSavedViewRoute,
		fields:  map[string]httpx.SavedViewQueryValueKind{"keyword": httpx.SavedViewQueryString, "type": httpx.SavedViewQueryString},
		columns: map[string]struct{}{"role": {}, "builtin": {}, "permission_count": {}, "user_count": {}, "remark": {}, "updated_at": {}, "operation": {}},
	}
	permissionSavedViewDefinition = rbacSavedViewDefinition{
		surface: "permission.list", collectionRoute: rbaccontract.PermissionSavedViewsRoute, itemRoute: rbaccontract.PermissionSavedViewRoute,
		fields:  map[string]httpx.SavedViewQueryValueKind{"keyword": httpx.SavedViewQueryString, "module": httpx.SavedViewQueryString},
		columns: map[string]struct{}{"permission": {}, "module": {}, "code": {}, "description": {}, "role_count": {}, "created_at": {}, "updated_at": {}, "operation": {}},
	}
)

type rbacSavedViewListResponse struct {
	Items []httpx.SavedViewResponse `json:"items"`
}

func registerRBACSavedViewRoutes(group *gin.RouterGroup, moduleCtx *module.Context, service moduleapi.SavedViewService, guard gin.HandlerFunc, definition rbacSavedViewDefinition) {
	if group == nil || moduleCtx == nil || service == nil {
		return
	}
	group.GET(definition.collectionRoute, guard, handleRBACSavedViewList(moduleCtx.I18n, service, definition))
	group.POST(definition.collectionRoute, guard, handleRBACSavedViewCreate(moduleCtx.I18n, service, definition))
	group.PUT(definition.itemRoute, guard, handleRBACSavedViewUpdate(moduleCtx.I18n, service, definition))
	group.DELETE(definition.itemRoute, guard, handleRBACSavedViewDelete(moduleCtx.I18n, service, definition))
}

//nolint:cyclop // 角色与权限保存视图的字段、类型和列白名单必须在同一 HTTP 边界逐项校验。
func validateRBACSavedView(request httpx.SavedViewRequest, definition rbacSavedViewDefinition) error {
	if strings.TrimSpace(request.Name) == "" || !rbacSavedViewPageSize(request.PageSize) || !json.Valid(request.QueryState) {
		return moduleapi.ErrSavedViewInvalidInput
	}
	if err := httpx.ValidateSavedViewQueryState(request.QueryState, definition.fields, func(ctx *gin.Context) string {
		if definition.surface == roleSavedViewDefinition.surface {
			roleType := ctx.Query("type")
			if roleType != "" && roleType != "builtin" && roleType != "custom" {
				return "invalid role type"
			}
		}
		return ""
	}); err != nil {
		return moduleapi.ErrSavedViewInvalidInput
	}
	for _, column := range request.VisibleColumns {
		if _, ok := definition.columns[strings.TrimSpace(column)]; !ok {
			return moduleapi.ErrSavedViewInvalidInput
		}
	}
	return nil
}

func rbacSavedViewPageSize(size int) bool {
	return size == 10 || size == 20 || size == 50 || size == 100
}

func handleRBACSavedViewList(localizer *i18n.Service, service moduleapi.SavedViewService, definition rbacSavedViewDefinition) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		owner, ok := httpx.SavedViewOwnerID(ctx)
		if !ok {
			httpx.WriteSavedViewError(ctx, localizer, moduleapi.ErrSavedViewInvalidInput)
			return
		}
		views, err := service.List(ctx.Request.Context(), owner, definition.surface)
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
		httpx.WriteSuccess(ctx, http.StatusOK, rbacSavedViewListResponse{Items: items})
	}
}

func handleRBACSavedViewCreate(localizer *i18n.Service, service moduleapi.SavedViewService, definition rbacSavedViewDefinition) gin.HandlerFunc {
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
		if err := validateRBACSavedView(request, definition); err != nil {
			httpx.WriteSavedViewError(ctx, localizer, err)
			return
		}
		view, err := service.Create(ctx.Request.Context(), moduleapi.SavedViewCreateInput{OwnerUserID: owner, SurfaceKey: definition.surface, Name: request.Name, QueryState: request.QueryState, PageSize: request.PageSize, VisibleColumns: request.VisibleColumns, IsDefault: request.IsDefault})
		if err != nil {
			httpx.WriteSavedViewError(ctx, localizer, err)
			return
		}
		writeRBACSavedView(ctx, localizer, http.StatusCreated, view)
	}
}

func handleRBACSavedViewUpdate(localizer *i18n.Service, service moduleapi.SavedViewService, definition rbacSavedViewDefinition) gin.HandlerFunc {
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
		if err := validateRBACSavedView(request, definition); err != nil {
			httpx.WriteSavedViewError(ctx, localizer, err)
			return
		}
		view, err := service.Update(ctx.Request.Context(), moduleapi.SavedViewUpdateInput{ID: id, OwnerUserID: owner, SurfaceKey: definition.surface, Name: request.Name, QueryState: request.QueryState, PageSize: request.PageSize, VisibleColumns: request.VisibleColumns, IsDefault: request.IsDefault})
		if err != nil {
			httpx.WriteSavedViewError(ctx, localizer, err)
			return
		}
		writeRBACSavedView(ctx, localizer, http.StatusOK, view)
	}
}

func handleRBACSavedViewDelete(localizer *i18n.Service, service moduleapi.SavedViewService, definition rbacSavedViewDefinition) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		owner, ownerOK := httpx.SavedViewOwnerID(ctx)
		id, idOK := httpx.SavedViewID(ctx)
		if !ownerOK || !idOK {
			httpx.WriteSavedViewError(ctx, localizer, moduleapi.ErrSavedViewInvalidInput)
			return
		}
		if err := service.Delete(ctx.Request.Context(), owner, definition.surface, id); err != nil {
			httpx.WriteSavedViewError(ctx, localizer, err)
			return
		}
		ctx.Status(http.StatusNoContent)
	}
}

func writeRBACSavedView(ctx *gin.Context, localizer *i18n.Service, status int, view moduleapi.SavedView) {
	item, err := httpx.ToSavedViewResponse(view)
	if err != nil {
		httpx.WriteSavedViewError(ctx, localizer, err)
		return
	}
	httpx.WriteSuccess(ctx, status, item)
}
