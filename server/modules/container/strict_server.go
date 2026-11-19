package container

import (
	"context"
	"github.com/gin-gonic/gin"
	openapigen "graft/server/internal/contract/openapi/generated"

	containeropenapi "graft/server/internal/contract/openapi/container"
	"graft/server/internal/contract/openapi/strictsupport"
)

// containerStrictServer 将现有 HTTP 响应写入接入生成的请求契约。
type containerStrictServer struct {
	routes routeRuntime
	saved  containerSavedViewHandlers
}

var _ containeropenapi.StrictServerInterface = containerStrictServer{}

func (s containerStrictServer) GetContainers(ctx context.Context, _ containeropenapi.GetContainersRequestObject) (containeropenapi.GetContainersResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.routes.handleList)
}

func (s containerStrictServer) GetContainerDashboardSummary(ctx context.Context, _ containeropenapi.GetContainerDashboardSummaryRequestObject) (containeropenapi.GetContainerDashboardSummaryResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.routes.handleDashboardSummary)
}

func (s containerStrictServer) GetContainer(ctx context.Context, _ containeropenapi.GetContainerRequestObject) (containeropenapi.GetContainerResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.routes.handleDetail)
}

func (s containerStrictServer) GetContainerEvents(ctx context.Context, _ containeropenapi.GetContainerEventsRequestObject) (containeropenapi.GetContainerEventsResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.routes.handleEvents)
}

func (s containerStrictServer) GetContainerLogs(ctx context.Context, _ containeropenapi.GetContainerLogsRequestObject) (containeropenapi.GetContainerLogsResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.routes.handleLogs)
}

func (s containerStrictServer) PostContainerShellSession(ctx context.Context, req containeropenapi.PostContainerShellSessionRequestObject) (containeropenapi.PostContainerShellSessionResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, func(c *gin.Context) { s.routes.handleShellSessionCreate(c, req.Body) })
}

func (s containerStrictServer) GetContainerMountUsage(ctx context.Context, _ containeropenapi.GetContainerMountUsageRequestObject) (containeropenapi.GetContainerMountUsageResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.routes.handleMountUsageList)
}

func (s containerStrictServer) PostContainerMountUsageRefresh(ctx context.Context, _ containeropenapi.PostContainerMountUsageRefreshRequestObject) (containeropenapi.PostContainerMountUsageRefreshResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.routes.handleMountUsageRefresh)
}

func (s containerStrictServer) PostContainerStart(ctx context.Context, _ containeropenapi.PostContainerStartRequestObject) (containeropenapi.PostContainerStartResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.routes.handleStart)
}

func (s containerStrictServer) PostContainerStop(ctx context.Context, _ containeropenapi.PostContainerStopRequestObject) (containeropenapi.PostContainerStopResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.routes.handleStop)
}

func (s containerStrictServer) PostContainerRestart(ctx context.Context, _ containeropenapi.PostContainerRestartRequestObject) (containeropenapi.PostContainerRestartResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.routes.handleRestart)
}

func (s containerStrictServer) PostContainerRemove(ctx context.Context, req containeropenapi.PostContainerRemoveRequestObject) (containeropenapi.PostContainerRemoveResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, func(c *gin.Context) { s.routes.handleRemove(c, req.Body) })
}

func (s containerStrictServer) PostContainerBatchActions(ctx context.Context, req containeropenapi.PostContainerBatchActionsRequestObject) (containeropenapi.PostContainerBatchActionsResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, func(c *gin.Context) { s.routes.handleBatchAction(c, req.Body) })
}

func (s containerStrictServer) GetDockerImages(ctx context.Context, _ containeropenapi.GetDockerImagesRequestObject) (containeropenapi.GetDockerImagesResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.routes.handleDockerImages)
}

func (s containerStrictServer) PostDockerImagePull(ctx context.Context, req containeropenapi.PostDockerImagePullRequestObject) (containeropenapi.PostDockerImagePullResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, func(c *gin.Context) { s.routes.handleDockerImagePull(c, req.Body) })
}

func (s containerStrictServer) GetDockerImage(ctx context.Context, _ containeropenapi.GetDockerImageRequestObject) (containeropenapi.GetDockerImageResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.routes.handleDockerImage)
}

func (s containerStrictServer) PostDockerImageTag(ctx context.Context, req containeropenapi.PostDockerImageTagRequestObject) (containeropenapi.PostDockerImageTagResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, func(c *gin.Context) { s.routes.handleDockerImageTag(c, req.Body) })
}

func (s containerStrictServer) PostDockerImageUntag(ctx context.Context, req containeropenapi.PostDockerImageUntagRequestObject) (containeropenapi.PostDockerImageUntagResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, func(c *gin.Context) { s.routes.handleDockerImageUntag(c, req.Body) })
}

func (s containerStrictServer) PostDockerImageRemove(ctx context.Context, req containeropenapi.PostDockerImageRemoveRequestObject) (containeropenapi.PostDockerImageRemoveResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, func(c *gin.Context) { s.routes.handleDockerImageRemove(c, req.Body) })
}

func (s containerStrictServer) PostDockerImageBatchRemove(ctx context.Context, req containeropenapi.PostDockerImageBatchRemoveRequestObject) (containeropenapi.PostDockerImageBatchRemoveResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, func(c *gin.Context) { s.routes.handleDockerImageBatchRemove(c, req.Body) })
}

func (s containerStrictServer) GetDockerNetworks(ctx context.Context, _ containeropenapi.GetDockerNetworksRequestObject) (containeropenapi.GetDockerNetworksResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.routes.handleDockerNetworks)
}

func (s containerStrictServer) PostDockerNetwork(ctx context.Context, req containeropenapi.PostDockerNetworkRequestObject) (containeropenapi.PostDockerNetworkResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, func(c *gin.Context) { s.routes.handleDockerNetworkCreate(c, req.Body) })
}

func (s containerStrictServer) GetDockerNetwork(ctx context.Context, _ containeropenapi.GetDockerNetworkRequestObject) (containeropenapi.GetDockerNetworkResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.routes.handleDockerNetwork)
}

func (s containerStrictServer) DeleteDockerNetwork(ctx context.Context, req containeropenapi.DeleteDockerNetworkRequestObject) (containeropenapi.DeleteDockerNetworkResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, func(c *gin.Context) {
		s.routes.handleDockerNetworkRemove(c, (*openapigen.DeleteDockerNetworkJSONRequestBody)(req.Body))
	})
}

func (s containerStrictServer) GetDockerVolumes(ctx context.Context, _ containeropenapi.GetDockerVolumesRequestObject) (containeropenapi.GetDockerVolumesResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.routes.handleDockerVolumes)
}

func (s containerStrictServer) GetDockerVolume(ctx context.Context, _ containeropenapi.GetDockerVolumeRequestObject) (containeropenapi.GetDockerVolumeResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.routes.handleDockerVolume)
}

func (s containerStrictServer) PostDockerVolumeRemove(ctx context.Context, req containeropenapi.PostDockerVolumeRemoveRequestObject) (containeropenapi.PostDockerVolumeRemoveResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, func(c *gin.Context) { s.routes.handleDockerVolumeRemove(c, req.Body) })
}

func (s containerStrictServer) PostDockerVolumeBatchRemove(ctx context.Context, req containeropenapi.PostDockerVolumeBatchRemoveRequestObject) (containeropenapi.PostDockerVolumeBatchRemoveResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, func(c *gin.Context) { s.routes.handleDockerVolumeBatchRemove(c, req.Body) })
}

func (s containerStrictServer) GetDockerSystem(ctx context.Context, _ containeropenapi.GetDockerSystemRequestObject) (containeropenapi.GetDockerSystemResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.routes.handleDockerSystem)
}

func (s containerStrictServer) GetContainerSavedViews(ctx context.Context, _ containeropenapi.GetContainerSavedViewsRequestObject) (containeropenapi.GetContainerSavedViewsResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.saved.containers.list)
}

func (s containerStrictServer) PostContainerSavedView(ctx context.Context, _ containeropenapi.PostContainerSavedViewRequestObject) (containeropenapi.PostContainerSavedViewResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.saved.containers.create)
}

func (s containerStrictServer) PutContainerSavedView(ctx context.Context, _ containeropenapi.PutContainerSavedViewRequestObject) (containeropenapi.PutContainerSavedViewResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.saved.containers.update)
}

func (s containerStrictServer) DeleteContainerSavedView(ctx context.Context, _ containeropenapi.DeleteContainerSavedViewRequestObject) (containeropenapi.DeleteContainerSavedViewResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.saved.containers.remove)
}

func (s containerStrictServer) GetDockerImageSavedViews(ctx context.Context, _ containeropenapi.GetDockerImageSavedViewsRequestObject) (containeropenapi.GetDockerImageSavedViewsResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.saved.images.list)
}

func (s containerStrictServer) PostDockerImageSavedView(ctx context.Context, _ containeropenapi.PostDockerImageSavedViewRequestObject) (containeropenapi.PostDockerImageSavedViewResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.saved.images.create)
}

func (s containerStrictServer) PutDockerImageSavedView(ctx context.Context, _ containeropenapi.PutDockerImageSavedViewRequestObject) (containeropenapi.PutDockerImageSavedViewResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.saved.images.update)
}

func (s containerStrictServer) DeleteDockerImageSavedView(ctx context.Context, _ containeropenapi.DeleteDockerImageSavedViewRequestObject) (containeropenapi.DeleteDockerImageSavedViewResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.saved.images.remove)
}

func (s containerStrictServer) GetDockerNetworkSavedViews(ctx context.Context, _ containeropenapi.GetDockerNetworkSavedViewsRequestObject) (containeropenapi.GetDockerNetworkSavedViewsResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.saved.networks.list)
}

func (s containerStrictServer) PostDockerNetworkSavedView(ctx context.Context, _ containeropenapi.PostDockerNetworkSavedViewRequestObject) (containeropenapi.PostDockerNetworkSavedViewResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.saved.networks.create)
}

func (s containerStrictServer) PutDockerNetworkSavedView(ctx context.Context, _ containeropenapi.PutDockerNetworkSavedViewRequestObject) (containeropenapi.PutDockerNetworkSavedViewResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.saved.networks.update)
}

func (s containerStrictServer) DeleteDockerNetworkSavedView(ctx context.Context, _ containeropenapi.DeleteDockerNetworkSavedViewRequestObject) (containeropenapi.DeleteDockerNetworkSavedViewResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.saved.networks.remove)
}

func (s containerStrictServer) GetDockerVolumeSavedViews(ctx context.Context, _ containeropenapi.GetDockerVolumeSavedViewsRequestObject) (containeropenapi.GetDockerVolumeSavedViewsResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.saved.volumes.list)
}

func (s containerStrictServer) PostDockerVolumeSavedView(ctx context.Context, _ containeropenapi.PostDockerVolumeSavedViewRequestObject) (containeropenapi.PostDockerVolumeSavedViewResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.saved.volumes.create)
}

func (s containerStrictServer) PutDockerVolumeSavedView(ctx context.Context, _ containeropenapi.PutDockerVolumeSavedViewRequestObject) (containeropenapi.PutDockerVolumeSavedViewResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.saved.volumes.update)
}

func (s containerStrictServer) DeleteDockerVolumeSavedView(ctx context.Context, _ containeropenapi.DeleteDockerVolumeSavedViewRequestObject) (containeropenapi.DeleteDockerVolumeSavedViewResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.saved.volumes.remove)
}
