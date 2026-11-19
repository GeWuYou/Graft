package project

import (
	"context"
	"github.com/gin-gonic/gin"
	"graft/server/internal/contract/openapi/generated"

	projectopenapi "graft/server/internal/contract/openapi/project"
	"graft/server/internal/contract/openapi/strictsupport"
)

// projectStrictServer 将现有 HTTP 响应写入接入生成的请求契约。
type projectStrictServer struct {
	routes routeRuntime
}

var _ projectopenapi.StrictServerInterface = projectStrictServer{}

func (s projectStrictServer) GetApplications(ctx context.Context, _ projectopenapi.GetApplicationsRequestObject) (projectopenapi.GetApplicationsResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.routes.handleList)
}

func (s projectStrictServer) PostApplicationComposeContextReferences(ctx context.Context, _ projectopenapi.PostApplicationComposeContextReferencesRequestObject) (projectopenapi.PostApplicationComposeContextReferencesResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.routes.handleComposeContextReferences)
}

func (s projectStrictServer) GetApplicationSavedViews(ctx context.Context, _ projectopenapi.GetApplicationSavedViewsRequestObject) (projectopenapi.GetApplicationSavedViewsResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.routes.handleSavedViewList)
}

func (s projectStrictServer) PostApplicationSavedView(ctx context.Context, _ projectopenapi.PostApplicationSavedViewRequestObject) (projectopenapi.PostApplicationSavedViewResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.routes.handleSavedViewCreate)
}

func (s projectStrictServer) PutApplicationSavedView(ctx context.Context, _ projectopenapi.PutApplicationSavedViewRequestObject) (projectopenapi.PutApplicationSavedViewResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.routes.handleSavedViewUpdate)
}

func (s projectStrictServer) DeleteApplicationSavedView(ctx context.Context, _ projectopenapi.DeleteApplicationSavedViewRequestObject) (projectopenapi.DeleteApplicationSavedViewResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.routes.handleSavedViewDelete)
}

func (s projectStrictServer) PostApplicationImportValidate(ctx context.Context, req projectopenapi.PostApplicationImportValidateRequestObject) (projectopenapi.PostApplicationImportValidateResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, func(c *gin.Context) {
		s.routes.handleImportValidate(c, (*generated.PostApplicationImportValidateJSONRequestBody)(req.Body))
	})
}

func (s projectStrictServer) GetApplicationImportRuntimeCandidates(ctx context.Context, _ projectopenapi.GetApplicationImportRuntimeCandidatesRequestObject) (projectopenapi.GetApplicationImportRuntimeCandidatesResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.routes.handleImportRuntimeCandidates)
}

func (s projectStrictServer) PostApplicationImportRuntimeInspect(ctx context.Context, req projectopenapi.PostApplicationImportRuntimeInspectRequestObject) (projectopenapi.PostApplicationImportRuntimeInspectResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, func(c *gin.Context) {
		s.routes.handleImportRuntimeInspect(c, (*generated.PostApplicationImportRuntimeInspectJSONRequestBody)(req.Body))
	})
}

func (s projectStrictServer) PostApplicationImportInspect(ctx context.Context, _ projectopenapi.PostApplicationImportInspectRequestObject) (projectopenapi.PostApplicationImportInspectResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.routes.handleImportInspect)
}

func (s projectStrictServer) PostApplicationImport(ctx context.Context, _ projectopenapi.PostApplicationImportRequestObject) (projectopenapi.PostApplicationImportResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.routes.handleImport)
}

func (s projectStrictServer) GetApplicationImportDirectorySources(ctx context.Context, _ projectopenapi.GetApplicationImportDirectorySourcesRequestObject) (projectopenapi.GetApplicationImportDirectorySourcesResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.routes.handleImportDirectorySources)
}

func (s projectStrictServer) GetApplicationImportDirectories(ctx context.Context, _ projectopenapi.GetApplicationImportDirectoriesRequestObject) (projectopenapi.GetApplicationImportDirectoriesResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.routes.handleImportDirectories)
}

func (s projectStrictServer) GetApplicationCreationMethods(ctx context.Context, _ projectopenapi.GetApplicationCreationMethodsRequestObject) (projectopenapi.GetApplicationCreationMethodsResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.routes.handleCreationMethods)
}

func (s projectStrictServer) GetApplicationComposeRuntimeTargets(ctx context.Context, _ projectopenapi.GetApplicationComposeRuntimeTargetsRequestObject) (projectopenapi.GetApplicationComposeRuntimeTargetsResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.routes.handleComposeRuntimeTargets)
}

func (s projectStrictServer) GetApplicationDiscoveryCandidates(ctx context.Context, _ projectopenapi.GetApplicationDiscoveryCandidatesRequestObject) (projectopenapi.GetApplicationDiscoveryCandidatesResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.routes.handleDiscoveryCandidates)
}

func (s projectStrictServer) GetApplicationManagedRoot(ctx context.Context, _ projectopenapi.GetApplicationManagedRootRequestObject) (projectopenapi.GetApplicationManagedRootResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.routes.handleManagedRoot)
}

func (s projectStrictServer) PostApplicationCreateValidate(ctx context.Context, _ projectopenapi.PostApplicationCreateValidateRequestObject) (projectopenapi.PostApplicationCreateValidateResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.routes.handleCreateValidate)
}

func (s projectStrictServer) PostApplicationNameAvailability(ctx context.Context, req projectopenapi.PostApplicationNameAvailabilityRequestObject) (projectopenapi.PostApplicationNameAvailabilityResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, func(c *gin.Context) {
		s.routes.handleApplicationNameAvailability(c, (*generated.PostApplicationNameAvailabilityJSONRequestBody)(req.Body))
	})
}

func (s projectStrictServer) PostApplicationCreate(ctx context.Context, _ projectopenapi.PostApplicationCreateRequestObject) (projectopenapi.PostApplicationCreateResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.routes.handleCreate)
}

func (s projectStrictServer) GetApplicationTemplates(ctx context.Context, _ projectopenapi.GetApplicationTemplatesRequestObject) (projectopenapi.GetApplicationTemplatesResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.routes.handlePublishedTemplates)
}

func (s projectStrictServer) GetPublishedApplicationTemplate(ctx context.Context, _ projectopenapi.GetPublishedApplicationTemplateRequestObject) (projectopenapi.GetPublishedApplicationTemplateResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.routes.handlePublishedTemplateDetail)
}

func (s projectStrictServer) GetPublishedApplicationTemplateVersion(ctx context.Context, _ projectopenapi.GetPublishedApplicationTemplateVersionRequestObject) (projectopenapi.GetPublishedApplicationTemplateVersionResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.routes.handlePublishedTemplateVersion)
}

func (s projectStrictServer) GetApplicationManagedTemplates(ctx context.Context, _ projectopenapi.GetApplicationManagedTemplatesRequestObject) (projectopenapi.GetApplicationManagedTemplatesResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.routes.handleManagedTemplates)
}

func (s projectStrictServer) GetApplicationTemplateSavedViews(ctx context.Context, _ projectopenapi.GetApplicationTemplateSavedViewsRequestObject) (projectopenapi.GetApplicationTemplateSavedViewsResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.routes.handleTemplateSavedViewList)
}

func (s projectStrictServer) PostApplicationTemplateSavedView(ctx context.Context, _ projectopenapi.PostApplicationTemplateSavedViewRequestObject) (projectopenapi.PostApplicationTemplateSavedViewResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.routes.handleTemplateSavedViewCreate)
}

func (s projectStrictServer) PutApplicationTemplateSavedView(ctx context.Context, _ projectopenapi.PutApplicationTemplateSavedViewRequestObject) (projectopenapi.PutApplicationTemplateSavedViewResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.routes.handleTemplateSavedViewUpdate)
}

func (s projectStrictServer) DeleteApplicationTemplateSavedView(ctx context.Context, _ projectopenapi.DeleteApplicationTemplateSavedViewRequestObject) (projectopenapi.DeleteApplicationTemplateSavedViewResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.routes.handleTemplateSavedViewDelete)
}

func (s projectStrictServer) PostApplicationTemplate(ctx context.Context, _ projectopenapi.PostApplicationTemplateRequestObject) (projectopenapi.PostApplicationTemplateResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.routes.handleCreateTemplateDraft)
}

func (s projectStrictServer) GetApplicationTemplate(ctx context.Context, _ projectopenapi.GetApplicationTemplateRequestObject) (projectopenapi.GetApplicationTemplateResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.routes.handleTemplateDetail)
}

func (s projectStrictServer) PutApplicationTemplate(ctx context.Context, _ projectopenapi.PutApplicationTemplateRequestObject) (projectopenapi.PutApplicationTemplateResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.routes.handleUpdateTemplateDraft)
}

func (s projectStrictServer) DeleteApplicationTemplate(ctx context.Context, _ projectopenapi.DeleteApplicationTemplateRequestObject) (projectopenapi.DeleteApplicationTemplateResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.routes.handleDeleteTemplate)
}

func (s projectStrictServer) PostApplicationTemplateClone(ctx context.Context, _ projectopenapi.PostApplicationTemplateCloneRequestObject) (projectopenapi.PostApplicationTemplateCloneResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.routes.handleCloneTemplate)
}

func (s projectStrictServer) PostApplicationTemplatePublish(ctx context.Context, _ projectopenapi.PostApplicationTemplatePublishRequestObject) (projectopenapi.PostApplicationTemplatePublishResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.routes.handlePublishTemplateDraft)
}

func (s projectStrictServer) PostApplicationTemplateWithdraw(ctx context.Context, _ projectopenapi.PostApplicationTemplateWithdrawRequestObject) (projectopenapi.PostApplicationTemplateWithdrawResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.routes.handleWithdrawTemplate)
}

func (s projectStrictServer) PostApplicationTemplateArchive(ctx context.Context, _ projectopenapi.PostApplicationTemplateArchiveRequestObject) (projectopenapi.PostApplicationTemplateArchiveResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.routes.handleArchiveTemplate)
}

func (s projectStrictServer) GetApplication(ctx context.Context, _ projectopenapi.GetApplicationRequestObject) (projectopenapi.GetApplicationResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.routes.handleDetail)
}

func (s projectStrictServer) GetApplicationOverview(ctx context.Context, _ projectopenapi.GetApplicationOverviewRequestObject) (projectopenapi.GetApplicationOverviewResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.routes.handleOverview)
}

func (s projectStrictServer) GetApplicationServices(ctx context.Context, _ projectopenapi.GetApplicationServicesRequestObject) (projectopenapi.GetApplicationServicesResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.routes.handleServices)
}

func (s projectStrictServer) GetApplicationLogs(ctx context.Context, _ projectopenapi.GetApplicationLogsRequestObject) (projectopenapi.GetApplicationLogsResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.routes.handleLogs)
}

func (s projectStrictServer) GetApplicationConfiguration(ctx context.Context, _ projectopenapi.GetApplicationConfigurationRequestObject) (projectopenapi.GetApplicationConfigurationResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.routes.handleConfiguration)
}

func (s projectStrictServer) GetApplicationConfigurationPreview(ctx context.Context, _ projectopenapi.GetApplicationConfigurationPreviewRequestObject) (projectopenapi.GetApplicationConfigurationPreviewResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.routes.handleConfigurationPreview)
}

func (s projectStrictServer) GetApplicationFiles(ctx context.Context, _ projectopenapi.GetApplicationFilesRequestObject) (projectopenapi.GetApplicationFilesResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.routes.handleProjectWorkspaceFiles)
}

func (s projectStrictServer) GetApplicationFileContent(ctx context.Context, _ projectopenapi.GetApplicationFileContentRequestObject) (projectopenapi.GetApplicationFileContentResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.routes.handleProjectWorkspaceFileContent)
}

func (s projectStrictServer) PutApplicationFileContent(ctx context.Context, req projectopenapi.PutApplicationFileContentRequestObject) (projectopenapi.PutApplicationFileContentResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, func(c *gin.Context) {
		s.routes.handleSaveProjectWorkspaceFileContent(c, (*generated.PutApplicationFileContentJSONRequestBody)(req.Body))
	})
}

func (s projectStrictServer) PutApplicationFileAnnotation(ctx context.Context, req projectopenapi.PutApplicationFileAnnotationRequestObject) (projectopenapi.PutApplicationFileAnnotationResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, func(c *gin.Context) {
		s.routes.handleProjectWorkspaceFileAnnotation(c, (*generated.PutApplicationFileAnnotationJSONRequestBody)(req.Body))
	})
}

func (s projectStrictServer) PostApplicationWorkspaceEntry(ctx context.Context, _ projectopenapi.PostApplicationWorkspaceEntryRequestObject) (projectopenapi.PostApplicationWorkspaceEntryResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.routes.handleCreateProjectWorkspaceEntry)
}

func (s projectStrictServer) PostApplicationWorkspaceEntryRename(ctx context.Context, _ projectopenapi.PostApplicationWorkspaceEntryRenameRequestObject) (projectopenapi.PostApplicationWorkspaceEntryRenameResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.routes.handleRenameProjectWorkspaceEntry)
}

func (s projectStrictServer) DeleteApplicationWorkspaceEntry(ctx context.Context, _ projectopenapi.DeleteApplicationWorkspaceEntryRequestObject) (projectopenapi.DeleteApplicationWorkspaceEntryResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.routes.handleDeleteApplicationWorkspaceEntry)
}

func (s projectStrictServer) PostApplicationRefresh(ctx context.Context, _ projectopenapi.PostApplicationRefreshRequestObject) (projectopenapi.PostApplicationRefreshResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.routes.handleRefresh)
}

func (s projectStrictServer) PostApplicationUp(ctx context.Context, _ projectopenapi.PostApplicationUpRequestObject) (projectopenapi.PostApplicationUpResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.routes.handleUp)
}

func (s projectStrictServer) PostApplicationStop(ctx context.Context, _ projectopenapi.PostApplicationStopRequestObject) (projectopenapi.PostApplicationStopResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.routes.handleStop)
}

func (s projectStrictServer) PostApplicationRestart(ctx context.Context, _ projectopenapi.PostApplicationRestartRequestObject) (projectopenapi.PostApplicationRestartResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.routes.handleRestart)
}

func (s projectStrictServer) PostApplicationRedeploy(ctx context.Context, _ projectopenapi.PostApplicationRedeployRequestObject) (projectopenapi.PostApplicationRedeployResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.routes.handleRedeploy)
}

func (s projectStrictServer) PutApplicationLifecycleConfiguration(ctx context.Context, _ projectopenapi.PutApplicationLifecycleConfigurationRequestObject) (projectopenapi.PutApplicationLifecycleConfigurationResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.routes.handleLifecycleConfiguration)
}

func (s projectStrictServer) PostApplicationBatchActions(ctx context.Context, _ projectopenapi.PostApplicationBatchActionsRequestObject) (projectopenapi.PostApplicationBatchActionsResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.routes.handleBatchActions)
}

func (s projectStrictServer) PostApplicationUnregister(ctx context.Context, _ projectopenapi.PostApplicationUnregisterRequestObject) (projectopenapi.PostApplicationUnregisterResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.routes.handleUnregister)
}

func (s projectStrictServer) PostApplicationDestroy(ctx context.Context, req projectopenapi.PostApplicationDestroyRequestObject) (projectopenapi.PostApplicationDestroyResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, func(c *gin.Context) {
		s.routes.handleDestroy(c, (*generated.PostApplicationDestroyJSONRequestBody)(req.Body))
	})
}
