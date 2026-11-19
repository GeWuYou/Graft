package container

import (
	"encoding/json"
	containeropenapi "graft/server/internal/contract/openapi/container"
	"graft/server/internal/contract/openapi/strictsupport"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"graft/server/internal/httpx"
	"graft/server/internal/i18n"
	"graft/server/internal/module"
	"graft/server/internal/moduleapi"
)

const (
	containerListSavedViewSurface     = "container.list"
	dockerImageListSavedViewSurface   = "docker-image.list"
	dockerNetworkListSavedViewSurface = "docker-network.list"
	dockerVolumeListSavedViewSurface  = "docker-volume.list"
)

type containerSavedViewDefinition struct {
	surface        string
	queryFields    map[string]httpx.SavedViewQueryValueKind
	visibleColumns map[string]struct{}
	bind           func(*gin.Context, *module.Context) bool
}

// containerSavedViewListResponse 是容器资源列表保存视图的显式 HTTP 响应 DTO。
// 它保持与 OpenAPI saved-view-list-response wrapper 一致，避免 map 形状成为隐式契约。
type containerSavedViewListResponse struct {
	Items []httpx.SavedViewResponse `json:"items"`
}

var containerSavedViewDefinitions = []containerSavedViewDefinition{
	{
		surface: containerListSavedViewSurface,
		queryFields: map[string]httpx.SavedViewQueryValueKind{
			"keyword": httpx.SavedViewQueryString, "state": httpx.SavedViewQueryString,
			"health": httpx.SavedViewQueryString, "deployment_type": httpx.SavedViewQueryString,
			"runtime_target_id": httpx.SavedViewQueryNumber,
		},
		visibleColumns: map[string]struct{}{
			"row-select": {}, "state": {}, "name": {}, "image": {}, "runtime": {}, "ports": {}, "health": {}, "operation": {},
		},
		bind: func(ctx *gin.Context, moduleCtx *module.Context) bool {
			_, ok := bindGetContainersParams(ctx, moduleCtx)
			return ok
		},
	},
	{
		surface: dockerImageListSavedViewSurface,
		queryFields: map[string]httpx.SavedViewQueryValueKind{
			"keyword": httpx.SavedViewQueryString, "unused": httpx.SavedViewQueryBool,
		},
		visibleColumns: map[string]struct{}{
			"row-select": {}, "tags": {}, "size": {}, "containers": {}, "status": {}, "created_at": {}, "actions": {},
		},
		bind: func(ctx *gin.Context, moduleCtx *module.Context) bool {
			_, ok := bindGetDockerImagesParams(ctx, moduleCtx)
			return ok
		},
	},
	{
		surface: dockerNetworkListSavedViewSurface,
		queryFields: map[string]httpx.SavedViewQueryValueKind{
			"keyword": httpx.SavedViewQueryString, "driver": httpx.SavedViewQueryString,
			"scope": httpx.SavedViewQueryString, "usage": httpx.SavedViewQueryString,
			"source": httpx.SavedViewQueryString, "compose_project": httpx.SavedViewQueryString,
		},
		visibleColumns: map[string]struct{}{
			"row-select": {}, "name": {}, "driver": {}, "scope": {}, "source": {}, "containers": {}, "status": {}, "operation": {},
		},
		bind: func(ctx *gin.Context, moduleCtx *module.Context) bool {
			_, ok := bindGetDockerNetworksParams(ctx, moduleCtx)
			return ok
		},
	},
	{
		surface: dockerVolumeListSavedViewSurface,
		queryFields: map[string]httpx.SavedViewQueryValueKind{
			"keyword": httpx.SavedViewQueryString, "driver": httpx.SavedViewQueryString,
			"scope": httpx.SavedViewQueryString, "usage": httpx.SavedViewQueryString,
			"source": httpx.SavedViewQueryString, "compose_project": httpx.SavedViewQueryString,
			"created_after": httpx.SavedViewQueryString, "created_before": httpx.SavedViewQueryString,
			"size_min_bytes": httpx.SavedViewQueryNumber, "size_max_bytes": httpx.SavedViewQueryNumber,
			"anonymous": httpx.SavedViewQueryBool, "orphaned": httpx.SavedViewQueryBool,
			"sort_by": httpx.SavedViewQueryString, "sort_order": httpx.SavedViewQueryString,
		},
		visibleColumns: map[string]struct{}{
			"row-select": {}, "name": {}, "driver": {}, "scope": {}, "source": {}, "usage": {}, "size_bytes": {}, "containers": {}, "created_at": {}, "operation": {},
		},
		bind: func(ctx *gin.Context, moduleCtx *module.Context) bool {
			_, ok := bindGetDockerVolumesParams(ctx, moduleCtx)
			return ok
		},
	},
}

type savedViewHandlers struct {
	list   gin.HandlerFunc
	create gin.HandlerFunc
	update gin.HandlerFunc
	remove gin.HandlerFunc
}

type containerSavedViewHandlers struct {
	containers savedViewHandlers
	images     savedViewHandlers
	networks   savedViewHandlers
	volumes    savedViewHandlers
}

func newContainerSavedViewHandlers(ctx *module.Context, service moduleapi.SavedViewService) containerSavedViewHandlers {
	if service == nil {
		return containerSavedViewHandlers{}
	}
	bySurface := make(map[string]containerSavedViewDefinition, len(containerSavedViewDefinitions))
	for _, definition := range containerSavedViewDefinitions {
		bySurface[definition.surface] = definition
	}
	handlers := func(surface string) savedViewHandlers {
		definition := bySurface[surface]
		return savedViewHandlers{
			list:   handleContainerSavedViewList(ctx.I18n, service, definition),
			create: handleContainerSavedViewCreate(ctx.I18n, ctx, service, definition),
			update: handleContainerSavedViewUpdate(ctx.I18n, ctx, service, definition),
			remove: handleContainerSavedViewDelete(ctx.I18n, service, definition),
		}
	}
	return containerSavedViewHandlers{
		containers: handlers(containerListSavedViewSurface),
		images:     handlers(dockerImageListSavedViewSurface),
		networks:   handlers(dockerNetworkListSavedViewSurface),
		volumes:    handlers(dockerVolumeListSavedViewSurface),
	}
}

func registerContainerSavedViewRoutes(ctx *module.Context, handlers containerSavedViewHandlers, wrapper containeropenapi.ServerInterfaceWrapper, guard gin.HandlerFunc) {
	if ctx == nil || ctx.Router == nil || handlers.containers.list == nil {
		return
	}
	containers := ctx.Router.Group("/ops/containers")
	containers.GET("/saved-views", guard, strictsupport.Route(wrapper.GetContainerSavedViews, nil))
	containers.POST("/saved-views", guard, strictsupport.CacheRequestBody(), strictsupport.Route(wrapper.PostContainerSavedView, handlers.containers.create))
	containers.PUT("/saved-views/:viewId", guard, strictsupport.CacheRequestBody(), strictsupport.Route(wrapper.PutContainerSavedView, handlers.containers.update))
	containers.DELETE("/saved-views/:viewId", guard, strictsupport.Route(wrapper.DeleteContainerSavedView, handlers.containers.remove))
	images := ctx.Router.Group("/docker/images")
	images.GET("/saved-views", guard, strictsupport.Route(wrapper.GetDockerImageSavedViews, nil))
	images.POST("/saved-views", guard, strictsupport.CacheRequestBody(), strictsupport.Route(wrapper.PostDockerImageSavedView, handlers.images.create))
	images.PUT("/saved-views/:viewId", guard, strictsupport.CacheRequestBody(), strictsupport.Route(wrapper.PutDockerImageSavedView, handlers.images.update))
	images.DELETE("/saved-views/:viewId", guard, strictsupport.Route(wrapper.DeleteDockerImageSavedView, handlers.images.remove))
	networks := ctx.Router.Group("/ops/docker/networks")
	networks.GET("/saved-views", guard, strictsupport.Route(wrapper.GetDockerNetworkSavedViews, nil))
	networks.POST("/saved-views", guard, strictsupport.CacheRequestBody(), strictsupport.Route(wrapper.PostDockerNetworkSavedView, handlers.networks.create))
	networks.PUT("/saved-views/:viewId", guard, strictsupport.CacheRequestBody(), strictsupport.Route(wrapper.PutDockerNetworkSavedView, handlers.networks.update))
	networks.DELETE("/saved-views/:viewId", guard, strictsupport.Route(wrapper.DeleteDockerNetworkSavedView, handlers.networks.remove))
	volumes := ctx.Router.Group("/ops/docker/volumes")
	volumes.GET("/saved-views", guard, strictsupport.Route(wrapper.GetDockerVolumeSavedViews, nil))
	volumes.POST("/saved-views", guard, strictsupport.CacheRequestBody(), strictsupport.Route(wrapper.PostDockerVolumeSavedView, handlers.volumes.create))
	volumes.PUT("/saved-views/:viewId", guard, strictsupport.CacheRequestBody(), strictsupport.Route(wrapper.PutDockerVolumeSavedView, handlers.volumes.update))
	volumes.DELETE("/saved-views/:viewId", guard, strictsupport.Route(wrapper.DeleteDockerVolumeSavedView, handlers.volumes.remove))
}

func validateContainerSavedView(request httpx.SavedViewRequest, ctx *module.Context, definition containerSavedViewDefinition) error {
	if strings.TrimSpace(request.Name) == "" || request.PageSize < 1 || request.PageSize > 200 || !json.Valid(request.QueryState) {
		return moduleapi.ErrSavedViewInvalidInput
	}
	if err := httpx.ValidateSavedViewQueryState(request.QueryState, definition.queryFields, definition.bindWithContext(ctx)); err != nil {
		return moduleapi.ErrSavedViewInvalidInput
	}
	for _, column := range request.VisibleColumns {
		if _, ok := definition.visibleColumns[strings.TrimSpace(column)]; !ok {
			return moduleapi.ErrSavedViewInvalidInput
		}
	}
	return nil
}

func (definition containerSavedViewDefinition) bindWithContext(ctx *module.Context) func(*gin.Context) string {
	return func(ginCtx *gin.Context) string {
		if definition.bind(ginCtx, ctx) {
			return ""
		}
		return "invalid query"
	}
}

func handleContainerSavedViewList(localizer *i18n.Service, service moduleapi.SavedViewService, definition containerSavedViewDefinition) gin.HandlerFunc {
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
			mapped, mapErr := httpx.ToSavedViewResponse(view)
			if mapErr != nil {
				httpx.WriteSavedViewError(ctx, localizer, mapErr)
				return
			}
			items = append(items, mapped)
		}
		httpx.WriteSuccess(ctx, http.StatusOK, containerSavedViewListResponse{Items: items})
	}
}

func handleContainerSavedViewCreate(localizer *i18n.Service, moduleCtx *module.Context, service moduleapi.SavedViewService, definition containerSavedViewDefinition) gin.HandlerFunc {
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
		if err := validateContainerSavedView(request, moduleCtx, definition); err != nil {
			httpx.WriteSavedViewError(ctx, localizer, err)
			return
		}
		view, err := service.Create(ctx.Request.Context(), moduleapi.SavedViewCreateInput{OwnerUserID: owner, SurfaceKey: definition.surface, Name: request.Name, QueryState: request.QueryState, PageSize: request.PageSize, VisibleColumns: request.VisibleColumns, IsDefault: request.IsDefault})
		if err != nil {
			httpx.WriteSavedViewError(ctx, localizer, err)
			return
		}
		writeContainerSavedView(ctx, localizer, http.StatusCreated, view)
	}
}

func handleContainerSavedViewUpdate(localizer *i18n.Service, moduleCtx *module.Context, service moduleapi.SavedViewService, definition containerSavedViewDefinition) gin.HandlerFunc {
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
		if err := validateContainerSavedView(request, moduleCtx, definition); err != nil {
			httpx.WriteSavedViewError(ctx, localizer, err)
			return
		}
		view, err := service.Update(ctx.Request.Context(), moduleapi.SavedViewUpdateInput{ID: id, OwnerUserID: owner, SurfaceKey: definition.surface, Name: request.Name, QueryState: request.QueryState, PageSize: request.PageSize, VisibleColumns: request.VisibleColumns, IsDefault: request.IsDefault})
		if err != nil {
			httpx.WriteSavedViewError(ctx, localizer, err)
			return
		}
		writeContainerSavedView(ctx, localizer, http.StatusOK, view)
	}
}

func handleContainerSavedViewDelete(localizer *i18n.Service, service moduleapi.SavedViewService, definition containerSavedViewDefinition) gin.HandlerFunc {
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

func writeContainerSavedView(ctx *gin.Context, localizer *i18n.Service, status int, view moduleapi.SavedView) {
	mapped, err := httpx.ToSavedViewResponse(view)
	if err != nil {
		httpx.WriteSavedViewError(ctx, localizer, err)
		return
	}
	httpx.WriteSuccess(ctx, status, mapped)
}
