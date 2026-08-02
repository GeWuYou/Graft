package project

import (
	"errors"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"graft/server/internal/contract/httpheader"
	messagecontract "graft/server/internal/contract/message"
	generated "graft/server/internal/contract/openapi/generated"
	"graft/server/internal/httpx"
	"graft/server/internal/logger/logsafe"
	"graft/server/internal/module"
	"graft/server/internal/moduleapi"
	projectcontract "graft/server/modules/project/contract"
	projectstore "graft/server/modules/project/store"
)

type routeRuntime struct {
	ctx        *module.Context
	service    *Service
	authorizer moduleapi.Authorizer
}

type projectRouteErrorRule struct {
	target error
	status int
	code   projectcontract.ErrorCode
}

var projectInputErrorRules = []projectRouteErrorRule{
	{target: errProjectApplicationNameRequired, status: http.StatusBadRequest, code: projectcontract.ApplicationNameRequired},
	{target: errProjectInvalidApplicationName, status: http.StatusBadRequest, code: projectcontract.ApplicationInvalidApplicationName},
	{target: errProjectApplicationNameOccupied, status: http.StatusConflict, code: projectcontract.ApplicationNameOccupied},
	{target: errProjectManagedRootUnconfigured, status: http.StatusBadRequest, code: projectcontract.ApplicationManagedRootUnconfigured},
	{target: errProjectManagedRootInvalid, status: http.StatusBadRequest, code: projectcontract.ApplicationManagedRootInvalid},
	{target: errProjectInvalidCompose, status: http.StatusBadRequest, code: projectcontract.ApplicationInvalidCompose},
	{target: errProjectWorkspaceUnsafe, status: http.StatusBadRequest, code: projectcontract.ApplicationWorkspaceUnsafe},
	{target: errProjectWorkspaceWriteFailed, status: http.StatusBadRequest, code: projectcontract.ApplicationWorkspaceWriteFailed},
	{target: errProjectImportValidation, status: http.StatusBadRequest, code: projectcontract.ApplicationImportValidationFailed},
}

const minimumProjectListLimit = 1

// registerRoutes 为项目模块注册 HTTP 路由，并统一挂载请求 ID、审计和权限校验中间件。
// 当上下文或路由器为空时跳过注册；项目服务缺失或认证依赖解析失败时返回错误。
//
//nolint:funlen // Application routes deliberately remain visible in one registration boundary.
func registerRoutes(ctx *module.Context, moduleName string, service *Service) error {
	if ctx == nil || ctx.Router == nil {
		return nil
	}
	if service == nil {
		return errors.New("project service is unavailable")
	}
	authService, err := resolveAuthService(ctx)
	if err != nil {
		return fmt.Errorf("resolve auth service: %w", err)
	}
	authorizer, err := resolveAuthorizer(ctx)
	if err != nil {
		return fmt.Errorf("resolve authorizer: %w", err)
	}

	routes := routeRuntime{ctx: ctx, service: service, authorizer: authorizer}
	publisher := httpx.NewSecurityAuditPublisher(ctx.EventBus, ctx.Logger, moduleName)
	group := ctx.Router.Group(projectcontract.ApplicationAPIGroup)
	group.Use(httpx.RequestIDMiddleware())
	group.GET(projectcontract.ApplicationCollectionRoute, httpx.RequirePermission(ctx.I18n, authService, authorizer, projectcontract.ApplicationViewPermission.String(), publisher), routes.handleList)
	group.POST(projectcontract.ApplicationComposeContextReferencesRoute, httpx.RequirePermission(ctx.I18n, authService, authorizer, projectcontract.ApplicationViewPermission.String(), publisher), routes.handleComposeContextReferences)
	group.GET(projectcontract.ApplicationSavedViewsRoute, httpx.RequirePermission(ctx.I18n, authService, authorizer, projectcontract.ApplicationViewPermission.String(), publisher), routes.handleSavedViewList)
	group.POST(projectcontract.ApplicationSavedViewsRoute, httpx.RequirePermission(ctx.I18n, authService, authorizer, projectcontract.ApplicationViewPermission.String(), publisher), routes.handleSavedViewCreate)
	group.PUT(projectcontract.ApplicationSavedViewRoute, httpx.RequirePermission(ctx.I18n, authService, authorizer, projectcontract.ApplicationViewPermission.String(), publisher), routes.handleSavedViewUpdate)
	group.DELETE(projectcontract.ApplicationSavedViewRoute, httpx.RequirePermission(ctx.I18n, authService, authorizer, projectcontract.ApplicationViewPermission.String(), publisher), routes.handleSavedViewDelete)
	group.POST(projectcontract.ApplicationImportValidateRoute, httpx.RequirePermission(ctx.I18n, authService, authorizer, projectcontract.ApplicationImportPermission.String(), publisher), routes.handleImportValidate)
	group.GET(projectcontract.ApplicationImportRuntimeCandidatesRoute, httpx.RequirePermission(ctx.I18n, authService, authorizer, projectcontract.ApplicationImportPermission.String(), publisher), routes.handleImportRuntimeCandidates)
	group.POST(projectcontract.ApplicationImportRuntimeInspectRoute, httpx.RequirePermission(ctx.I18n, authService, authorizer, projectcontract.ApplicationImportPermission.String(), publisher), routes.handleImportRuntimeInspect)
	group.POST(projectcontract.ApplicationImportInspectRoute, httpx.RequirePermission(ctx.I18n, authService, authorizer, projectcontract.ApplicationImportPermission.String(), publisher), routes.handleImportInspect)
	group.POST(projectcontract.ApplicationImportRoute, httpx.RequirePermission(ctx.I18n, authService, authorizer, projectcontract.ApplicationImportPermission.String(), publisher), routes.handleImport)
	group.GET(projectcontract.ApplicationImportDirectorySourcesRoute, httpx.RequirePermission(ctx.I18n, authService, authorizer, projectcontract.ApplicationImportPermission.String(), publisher), routes.handleImportDirectorySources)
	group.GET(projectcontract.ApplicationImportDirectoriesRoute, httpx.RequirePermission(ctx.I18n, authService, authorizer, projectcontract.ApplicationImportPermission.String(), publisher), routes.handleImportDirectories)
	group.GET(projectcontract.ApplicationCreationMethodsRoute, httpx.RequirePermission(ctx.I18n, authService, authorizer, projectcontract.ApplicationCreationMethodViewPermission.String(), publisher), routes.handleCreationMethods)
	group.GET(projectcontract.ApplicationComposeRuntimeTargetsRoute, httpx.RequirePermission(ctx.I18n, authService, authorizer, projectcontract.ApplicationCreatePermission.String(), publisher), routes.handleComposeRuntimeTargets)
	group.GET(projectcontract.ApplicationDiscoveryCandidatesRoute, httpx.RequirePermission(ctx.I18n, authService, authorizer, projectcontract.ApplicationDiscoveryViewPermission.String(), publisher), routes.handleDiscoveryCandidates)
	group.GET(projectcontract.ApplicationManagedRootRoute, httpx.RequirePermission(ctx.I18n, authService, authorizer, projectcontract.ApplicationCreatePermission.String(), publisher), routes.handleManagedRoot)
	group.POST(projectcontract.ApplicationCreateValidateRoute, httpx.RequirePermission(ctx.I18n, authService, authorizer, projectcontract.ApplicationCreatePermission.String(), publisher), routes.handleCreateValidate)
	group.POST(projectcontract.ApplicationNameAvailabilityRoute, httpx.RequirePermission(ctx.I18n, authService, authorizer, projectcontract.ApplicationCreatePermission.String(), publisher), routes.handleApplicationNameAvailability)
	group.POST(projectcontract.ApplicationCreateRoute, httpx.RequirePermission(ctx.I18n, authService, authorizer, projectcontract.ApplicationCreatePermission.String(), publisher), routes.handleCreate)
	// Template static routes precede /:applicationId so the Application detail route cannot capture them.
	group.GET(projectcontract.ApplicationTemplatesRoute, httpx.RequirePermission(ctx.I18n, authService, authorizer, projectcontract.ApplicationCreatePermission.String(), publisher), routes.handlePublishedTemplates)
	group.GET(projectcontract.ApplicationTemplatePublishedRoute, httpx.RequirePermission(ctx.I18n, authService, authorizer, projectcontract.ApplicationCreatePermission.String(), publisher), routes.handlePublishedTemplateDetail)
	group.GET(projectcontract.ApplicationTemplateVersionRoute, httpx.RequirePermission(ctx.I18n, authService, authorizer, projectcontract.ApplicationCreatePermission.String(), publisher), routes.handlePublishedTemplateVersion)
	group.GET(projectcontract.ApplicationTemplateManagementRoute, httpx.RequirePermission(ctx.I18n, authService, authorizer, projectcontract.ApplicationTemplateManagePermission.String(), publisher), routes.handleManagedTemplates)
	group.GET(projectcontract.ApplicationTemplateSavedViewsRoute, httpx.RequirePermission(ctx.I18n, authService, authorizer, projectcontract.ApplicationTemplateManagePermission.String(), publisher), routes.handleTemplateSavedViewList)
	group.POST(projectcontract.ApplicationTemplateSavedViewsRoute, httpx.RequirePermission(ctx.I18n, authService, authorizer, projectcontract.ApplicationTemplateManagePermission.String(), publisher), routes.handleTemplateSavedViewCreate)
	group.PUT(projectcontract.ApplicationTemplateSavedViewRoute, httpx.RequirePermission(ctx.I18n, authService, authorizer, projectcontract.ApplicationTemplateManagePermission.String(), publisher), routes.handleTemplateSavedViewUpdate)
	group.DELETE(projectcontract.ApplicationTemplateSavedViewRoute, httpx.RequirePermission(ctx.I18n, authService, authorizer, projectcontract.ApplicationTemplateManagePermission.String(), publisher), routes.handleTemplateSavedViewDelete)
	group.POST(projectcontract.ApplicationTemplatesRoute, httpx.RequirePermission(ctx.I18n, authService, authorizer, projectcontract.ApplicationTemplateManagePermission.String(), publisher), routes.handleCreateTemplateDraft)
	group.GET(projectcontract.ApplicationTemplateDetailRoute, httpx.RequirePermission(ctx.I18n, authService, authorizer, projectcontract.ApplicationTemplateManagePermission.String(), publisher), routes.handleTemplateDetail)
	group.PUT(projectcontract.ApplicationTemplateDetailRoute, httpx.RequirePermission(ctx.I18n, authService, authorizer, projectcontract.ApplicationTemplateManagePermission.String(), publisher), routes.handleUpdateTemplateDraft)
	group.DELETE(projectcontract.ApplicationTemplateDetailRoute, httpx.RequirePermission(ctx.I18n, authService, authorizer, projectcontract.ApplicationTemplateManagePermission.String(), publisher), routes.handleDeleteTemplate)
	group.POST(projectcontract.ApplicationTemplateCloneRoute, httpx.RequirePermission(ctx.I18n, authService, authorizer, projectcontract.ApplicationTemplateManagePermission.String(), publisher), routes.handleCloneTemplate)
	group.POST(projectcontract.ApplicationTemplatePublishRoute, httpx.RequirePermission(ctx.I18n, authService, authorizer, projectcontract.ApplicationTemplatePublishPermission.String(), publisher), routes.handlePublishTemplateDraft)
	group.POST(projectcontract.ApplicationTemplateWithdrawRoute, httpx.RequirePermission(ctx.I18n, authService, authorizer, projectcontract.ApplicationTemplatePublishPermission.String(), publisher), routes.handleWithdrawTemplate)
	group.POST(projectcontract.ApplicationTemplateArchiveRoute, httpx.RequirePermission(ctx.I18n, authService, authorizer, projectcontract.ApplicationTemplateManagePermission.String(), publisher), routes.handleArchiveTemplate)
	group.GET(projectcontract.ApplicationDetailRoute, httpx.RequirePermission(ctx.I18n, authService, authorizer, projectcontract.ApplicationViewPermission.String(), publisher), routes.handleDetail)
	group.GET(projectcontract.ApplicationOverviewRoute, httpx.RequirePermission(ctx.I18n, authService, authorizer, projectcontract.ApplicationViewPermission.String(), publisher), routes.handleOverview)
	group.GET(projectcontract.ApplicationServicesRoute, httpx.RequirePermission(ctx.I18n, authService, authorizer, projectcontract.ApplicationViewPermission.String(), publisher), routes.handleServices)
	group.GET(projectcontract.ApplicationLogsRoute, httpx.RequirePermission(ctx.I18n, authService, authorizer, projectcontract.ApplicationViewPermission.String(), publisher), routes.handleLogs)
	group.GET(projectcontract.ApplicationConfigurationRoute, httpx.RequirePermission(ctx.I18n, authService, authorizer, projectcontract.ApplicationViewPermission.String(), publisher), routes.handleConfiguration)
	group.GET(projectcontract.ApplicationConfigurationPreviewRoute, httpx.RequirePermission(ctx.I18n, authService, authorizer, projectcontract.ApplicationViewPermission.String(), publisher), routes.handleConfigurationPreview)
	group.GET(projectcontract.ApplicationWorkspaceFilesRoute, httpx.RequirePermission(ctx.I18n, authService, authorizer, projectcontract.ApplicationViewPermission.String(), publisher), routes.handleProjectWorkspaceFiles)
	group.GET(projectcontract.ApplicationWorkspaceFileContentRoute, httpx.RequirePermission(ctx.I18n, authService, authorizer, projectcontract.ApplicationViewPermission.String(), publisher), routes.handleProjectWorkspaceFileContent)
	group.PUT(projectcontract.ApplicationWorkspaceFileContentRoute, httpx.RequirePermission(ctx.I18n, authService, authorizer, projectcontract.ApplicationDeployPermission.String(), publisher), routes.handleSaveProjectWorkspaceFileContent)
	group.PUT(projectcontract.ApplicationWorkspaceFileAnnotationRoute, httpx.RequirePermission(ctx.I18n, authService, authorizer, projectcontract.ApplicationDeployPermission.String(), publisher), routes.handleProjectWorkspaceFileAnnotation)
	group.POST(projectcontract.ApplicationWorkspaceEntryRoute, httpx.RequirePermission(ctx.I18n, authService, authorizer, projectcontract.ApplicationDeployPermission.String(), publisher), routes.handleCreateProjectWorkspaceEntry)
	group.POST(projectcontract.ApplicationWorkspaceRenameRoute, httpx.RequirePermission(ctx.I18n, authService, authorizer, projectcontract.ApplicationDeployPermission.String(), publisher), routes.handleRenameProjectWorkspaceEntry)
	group.DELETE(projectcontract.ApplicationWorkspaceEntryRoute, httpx.RequirePermission(ctx.I18n, authService, authorizer, projectcontract.ApplicationDeployPermission.String(), publisher), routes.handleDeleteApplicationWorkspaceEntry)
	group.POST(projectcontract.ApplicationRefreshRoute, httpx.RequirePermission(ctx.I18n, authService, authorizer, projectcontract.ApplicationRefreshPermission.String(), publisher), routes.handleRefresh)
	group.POST(projectcontract.ApplicationUpRoute, httpx.RequirePermission(ctx.I18n, authService, authorizer, projectcontract.ApplicationLifecyclePermission.String(), publisher), routes.handleUp)
	group.POST(projectcontract.ApplicationStopRoute, httpx.RequirePermission(ctx.I18n, authService, authorizer, projectcontract.ApplicationLifecyclePermission.String(), publisher), routes.handleStop)
	group.POST(projectcontract.ApplicationRestartRoute, httpx.RequirePermission(ctx.I18n, authService, authorizer, projectcontract.ApplicationLifecyclePermission.String(), publisher), routes.handleRestart)
	group.POST(projectcontract.ApplicationRedeployRoute, httpx.RequirePermission(ctx.I18n, authService, authorizer, projectcontract.ApplicationLifecyclePermission.String(), publisher), routes.handleRedeploy)
	group.PUT(projectcontract.ApplicationLifecycleConfigurationRoute, httpx.RequirePermission(ctx.I18n, authService, authorizer, projectcontract.ApplicationLifecyclePermission.String(), publisher), routes.handleLifecycleConfiguration)
	group.POST(projectcontract.ApplicationBatchActionsRoute, httpx.RequirePermission(ctx.I18n, authService, authorizer, "", publisher), routes.handleBatchActions)
	group.POST(projectcontract.ApplicationUnregisterRoute, httpx.RequirePermission(ctx.I18n, authService, authorizer, projectcontract.ApplicationDestroyPermission.String(), publisher), routes.handleUnregister)
	group.POST(projectcontract.ApplicationDestroyRoute, httpx.RequirePermission(ctx.I18n, authService, authorizer, projectcontract.ApplicationDestroyPermission.String(), publisher), routes.handleDestroy)
	return nil
}

//nolint:dupl // 项目列表与运行时候选处理器有意共享生成绑定、查询和响应写入骨架。
func (r routeRuntime) handleList(ginCtx *gin.Context) {
	params, ok := bindListParams(ginCtx, r.ctx)
	if !ok {
		return
	}
	applicationGeneratedHandler{}.GetApplications(params)
	result, err := r.service.List(ginCtx.Request.Context(), ListQuery{
		Limit:                 intPtrValue(params.Limit),
		Offset:                intPtrValue(params.Offset),
		Keyword:               stringPtrValue(params.Keyword),
		Sort:                  projectListSortParamValue(params.Sort),
		DeploymentAdapterKind: stringPtrValue(params.DeploymentAdapterKind),
		RuntimeTargetID:       params.RuntimeTargetId,
		Provider:              stringPtrValue(params.Provider),
		SourceType:            stringPtrValue(params.SourceType),
		RuntimeStatus:         stringPtrValue(params.RuntimeStatus),
		DriftStatus:           stringPtrValue(params.DriftStatus),
	})
	if err != nil {
		r.writeRouteError(ginCtx, err)
		return
	}
	httpx.WriteSuccess(ginCtx, http.StatusOK, toProjectListResponse(result))
}

func (r routeRuntime) handleComposeContextReferences(ginCtx *gin.Context) {
	var request generated.PostApplicationComposeContextReferencesJSONRequestBody
	if !bindJSON(ginCtx, r.ctx, &request) {
		return
	}
	applicationGeneratedHandler{}.PostApplicationComposeContextReferences(
		bindPostApplicationComposeContextReferencesParams(ginCtx),
		request,
	)
	contexts := make([]ComposeContextReferenceRequest, 0, len(request.Contexts))
	for _, item := range request.Contexts {
		if item.RuntimeTargetId < 1 {
			r.writeRouteError(ginCtx, errProjectInvalidArgument)
			return
		}
		contexts = append(contexts, ComposeContextReferenceRequest{
			RuntimeTargetID:    item.RuntimeTargetId,
			ComposeProjectName: item.ComposeProjectName,
		})
	}
	result, err := r.service.ResolveComposeContextReferences(ginCtx.Request.Context(), contexts)
	if err != nil {
		r.writeRouteError(ginCtx, err)
		return
	}
	httpx.WriteSuccess(ginCtx, http.StatusOK, toComposeContextReferenceResponse(result))
}

func (r routeRuntime) handleComposeRuntimeTargets(ginCtx *gin.Context) {
	targets, err := r.service.ComposeRuntimeTargets(ginCtx.Request.Context())
	if err != nil {
		r.writeRouteError(ginCtx, err)
		return
	}
	items := make([]generated.ApplicationComposeRuntimeTarget, 0, len(targets))
	for _, target := range targets {
		readiness := generated.ApplicationComposeRuntimeTargetReadinessRuntimeUnavailable
		if target.Available {
			readiness = generated.ApplicationComposeRuntimeTargetReadinessReady
		}
		items = append(items, generated.ApplicationComposeRuntimeTarget{RuntimeTargetId: target.ID, DisplayName: target.DisplayName, Provider: target.Provider, Availability: target.Available, Readiness: readiness, Capabilities: target.Capabilities})
	}
	httpx.WriteSuccess(ginCtx, http.StatusOK, generated.ApplicationComposeRuntimeTargetCatalogResponse{DeploymentType: generated.ApplicationComposeRuntimeTargetCatalogResponseDeploymentTypeCompose, Items: items})
}

func (r routeRuntime) handleSavedViewList(ginCtx *gin.Context) {
	ownerID, ok := currentUserID(ginCtx)
	if !ok {
		r.writeInvalidArgumentError(ginCtx)
		return
	}
	applicationGeneratedHandler{}.GetApplicationSavedViews(generated.GetApplicationSavedViewsParams{})
	items, err := r.service.listSavedViews(ginCtx.Request.Context(), ownerID)
	if err != nil {
		r.writeRouteError(ginCtx, err)
		return
	}
	mapped := make([]generated.ApplicationSavedView, 0, len(items))
	for _, item := range items {
		view, mapErr := toGeneratedProjectSavedView(item)
		if mapErr != nil {
			r.writeSavedViewError(ginCtx, mapErr)
			return
		}
		mapped = append(mapped, view)
	}
	httpx.WriteSuccess(ginCtx, http.StatusOK, generated.ApplicationSavedViewListResponse{Items: mapped})
}

func (r routeRuntime) handleSavedViewCreate(ginCtx *gin.Context) {
	ownerID, ok := currentUserID(ginCtx)
	if !ok {
		r.writeInvalidArgumentError(ginCtx)
		return
	}
	var body generated.PostApplicationSavedViewJSONRequestBody
	if !bindJSON(ginCtx, r.ctx, &body) {
		return
	}
	request, err := projectSavedViewRequestFromGenerated(body)
	if err != nil {
		r.writeSavedViewError(ginCtx, err)
		return
	}
	applicationGeneratedHandler{}.PostApplicationSavedView(generated.PostApplicationSavedViewParams{}, body)
	item, err := r.service.createSavedView(ginCtx.Request.Context(), ownerID, request)
	if err != nil {
		r.writeSavedViewError(ginCtx, err)
		return
	}
	mapped, err := toGeneratedProjectSavedView(item)
	if err != nil {
		r.writeSavedViewError(ginCtx, err)
		return
	}
	httpx.WriteSuccess(ginCtx, http.StatusCreated, mapped)
}

func (r routeRuntime) handleSavedViewUpdate(ginCtx *gin.Context) {
	ownerID, ok := currentUserID(ginCtx)
	if !ok {
		r.writeInvalidArgumentError(ginCtx)
		return
	}
	id, ok := bindSavedViewID(ginCtx)
	if !ok {
		r.writeInvalidArgumentError(ginCtx)
		return
	}
	var body generated.PutApplicationSavedViewJSONRequestBody
	if !bindJSON(ginCtx, r.ctx, &body) {
		return
	}
	request, err := projectSavedViewRequestFromGenerated(body)
	if err != nil {
		r.writeSavedViewError(ginCtx, err)
		return
	}
	generatedID, ok := generatedSavedViewID(id)
	if !ok {
		r.writeInvalidArgumentError(ginCtx)
		return
	}
	applicationGeneratedHandler{}.PutApplicationSavedView(generatedID, generated.PutApplicationSavedViewParams{}, body)
	item, err := r.service.updateSavedView(ginCtx.Request.Context(), ownerID, id, request)
	if err != nil {
		r.writeSavedViewError(ginCtx, err)
		return
	}
	mapped, err := toGeneratedProjectSavedView(item)
	if err != nil {
		r.writeSavedViewError(ginCtx, err)
		return
	}
	httpx.WriteSuccess(ginCtx, http.StatusOK, mapped)
}

func (r routeRuntime) handleSavedViewDelete(ginCtx *gin.Context) {
	ownerID, ok := currentUserID(ginCtx)
	if !ok {
		r.writeInvalidArgumentError(ginCtx)
		return
	}
	id, ok := bindSavedViewID(ginCtx)
	if !ok {
		r.writeInvalidArgumentError(ginCtx)
		return
	}
	generatedID, ok := generatedSavedViewID(id)
	if !ok {
		r.writeInvalidArgumentError(ginCtx)
		return
	}
	applicationGeneratedHandler{}.DeleteApplicationSavedView(generatedID, generated.DeleteApplicationSavedViewParams{})
	if err := r.service.deleteSavedView(ginCtx.Request.Context(), ownerID, id); err != nil {
		r.writeSavedViewError(ginCtx, err)
		return
	}
	ginCtx.Status(http.StatusNoContent)
}

func (r routeRuntime) handleTemplateSavedViewList(c *gin.Context) {
	owner, ok := currentUserID(c)
	if !ok {
		r.writeInvalidArgumentError(c)
		return
	}
	items, err := r.service.listTemplateSavedViews(c.Request.Context(), owner)
	if err != nil {
		r.writeSavedViewError(c, err)
		return
	}
	mapped := make([]generated.SavedView, 0, len(items))
	for _, item := range items {
		view, mapErr := toGeneratedSavedView(item)
		if mapErr != nil {
			r.writeSavedViewError(c, mapErr)
			return
		}
		mapped = append(mapped, view)
	}
	httpx.WriteSuccess(c, http.StatusOK, generated.SavedViewListResponse{Items: mapped})
}
func (r routeRuntime) handleTemplateSavedViewCreate(c *gin.Context) {
	owner, ok := currentUserID(c)
	if !ok {
		r.writeInvalidArgumentError(c)
		return
	}
	var body generated.PostApplicationTemplateSavedViewJSONRequestBody
	if !bindJSON(c, r.ctx, &body) {
		return
	}
	request, err := genericSavedViewRequestFromGenerated(body)
	if err != nil {
		r.writeSavedViewError(c, err)
		return
	}
	view, err := r.service.createTemplateSavedView(c.Request.Context(), owner, request)
	if err != nil {
		r.writeSavedViewError(c, err)
		return
	}
	mapped, err := toGeneratedSavedView(view)
	if err != nil {
		r.writeSavedViewError(c, err)
		return
	}
	httpx.WriteSuccess(c, http.StatusCreated, mapped)
}
func (r routeRuntime) handleTemplateSavedViewUpdate(c *gin.Context) {
	owner, ok := currentUserID(c)
	if !ok {
		r.writeInvalidArgumentError(c)
		return
	}
	id, ok := bindSavedViewID(c)
	if !ok {
		r.writeInvalidArgumentError(c)
		return
	}
	var body generated.PutApplicationTemplateSavedViewJSONRequestBody
	if !bindJSON(c, r.ctx, &body) {
		return
	}
	request, err := genericSavedViewRequestFromGenerated(body)
	if err != nil {
		r.writeSavedViewError(c, err)
		return
	}
	view, err := r.service.updateTemplateSavedView(c.Request.Context(), owner, id, request)
	if err != nil {
		r.writeSavedViewError(c, err)
		return
	}
	mapped, err := toGeneratedSavedView(view)
	if err != nil {
		r.writeSavedViewError(c, err)
		return
	}
	httpx.WriteSuccess(c, http.StatusOK, mapped)
}
func (r routeRuntime) handleTemplateSavedViewDelete(c *gin.Context) {
	owner, ok := currentUserID(c)
	if !ok {
		r.writeInvalidArgumentError(c)
		return
	}
	id, ok := bindSavedViewID(c)
	if !ok {
		r.writeInvalidArgumentError(c)
		return
	}
	if err := r.service.deleteTemplateSavedView(c.Request.Context(), owner, id); err != nil {
		r.writeSavedViewError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (r routeRuntime) handleImportValidate(ginCtx *gin.Context) {
	var request generated.PostApplicationImportValidateJSONRequestBody
	if !bindJSON(ginCtx, r.ctx, &request) {
		return
	}
	applicationGeneratedHandler{}.PostApplicationImportValidate(bindImportValidateParams(ginCtx), request)
	result, err := r.service.ValidateImport(ginCtx.Request.Context(), toImportRequest(ginCtx, request))
	if err != nil {
		r.writeRouteError(ginCtx, err)
		return
	}
	httpx.WriteSuccess(ginCtx, http.StatusOK, toImportValidateResponse(result))
}

//nolint:dupl // 项目列表与运行时候选处理器有意共享生成绑定、查询和响应写入骨架。
func (r routeRuntime) handleImportRuntimeCandidates(ginCtx *gin.Context) {
	params, ok := bindGetApplicationImportRuntimeCandidatesParams(ginCtx, r.ctx)
	if !ok {
		return
	}
	applicationGeneratedHandler{}.GetApplicationImportRuntimeCandidates(params)
	result, err := r.service.ListRuntimeImportCandidates(ginCtx.Request.Context(), RuntimeImportCandidateListQuery{
		Availability: runtimeCandidateAvailabilityFromGenerated(params.Availability),
		Keyword:      stringPtrValue(params.Keyword),
		Limit:        intPtrValue(params.Limit),
		Offset:       intPtrValue(params.Offset),
	})
	if err != nil {
		r.writeRouteError(ginCtx, err)
		return
	}
	httpx.WriteSuccess(ginCtx, http.StatusOK, toRuntimeImportCandidatesResponse(result))
}

func (r routeRuntime) handleImportRuntimeInspect(ginCtx *gin.Context) {
	var request generated.PostApplicationImportRuntimeInspectJSONRequestBody
	if !bindJSON(ginCtx, r.ctx, &request) {
		return
	}
	applicationGeneratedHandler{}.PostApplicationImportRuntimeInspect(bindPostApplicationImportRuntimeInspectParams(ginCtx), request)
	result, err := r.service.InspectRuntimeCandidate(ginCtx.Request.Context(), RuntimeImportInspectRequest{
		CandidateKey:               request.CandidateKey,
		DisplayName:                request.DisplayName,
		ComposeProjectNameOverride: request.ComposeProjectNameOverride,
	})
	if err != nil {
		r.writeRouteError(ginCtx, err)
		return
	}
	httpx.WriteSuccess(ginCtx, http.StatusOK, toRuntimeImportInspectResponse(result))
}

func (r routeRuntime) handleImport(ginCtx *gin.Context) {
	var request generated.PostApplicationImportJSONRequestBody
	if !bindJSON(ginCtx, r.ctx, &request) {
		return
	}
	applicationGeneratedHandler{}.PostApplicationImport(bindPostApplicationImportParams(ginCtx), request)
	lifecycleConfig, err := lifecycleStandardConfigFromGenerated(request.LifecycleConfiguration)
	if err != nil {
		r.writeRouteError(ginCtx, err)
		return
	}
	result, err := r.service.ImportByInspection(ginCtx.Request.Context(), ImportExecuteRequest{
		InspectionID:               request.InspectionId,
		DisplayName:                request.DisplayName,
		ComposeProjectNameOverride: request.ComposeProjectNameOverride,
		LifecycleConfiguration:     &lifecycleConfig,
		ActorID:                    currentUserIDPointer(ginCtx),
	})
	if err != nil {
		r.writeRouteError(ginCtx, err)
		return
	}
	httpx.WriteSuccess(ginCtx, http.StatusOK, result)
}

func (r routeRuntime) handleImportInspect(ginCtx *gin.Context) {
	var request generated.PostApplicationImportInspectJSONRequestBody
	if !bindJSON(ginCtx, r.ctx, &request) {
		return
	}
	applicationGeneratedHandler{}.PostApplicationImportInspect(bindPostApplicationImportInspectParams(ginCtx), request)
	result, err := r.service.InspectImportDirectory(ginCtx.Request.Context(), ImportInspectRequest{
		DirectoryRef: ImportDirectoryReference{
			Provider: request.DirectoryRef.Provider,
			RootID:   request.DirectoryRef.RootId,
			Path:     request.DirectoryRef.Path,
		},
		DisplayName:                request.DisplayName,
		ComposeProjectNameOverride: request.ComposeProjectNameOverride,
	})
	if err != nil {
		r.writeRouteError(ginCtx, err)
		return
	}
	httpx.WriteSuccess(ginCtx, http.StatusOK, result)
}

func (r routeRuntime) handleImportDirectorySources(ginCtx *gin.Context) {
	result, err := r.service.ImportDirectorySources(ginCtx.Request.Context())
	if err != nil {
		r.writeRouteError(ginCtx, err)
		return
	}
	httpx.WriteSuccess(ginCtx, http.StatusOK, result)
}

func (r routeRuntime) handleImportDirectories(ginCtx *gin.Context) {
	query, ok := bindImportDirectoryBrowseQuery(ginCtx, r.ctx)
	if !ok {
		return
	}
	result, err := r.service.BrowseImportDirectories(ginCtx.Request.Context(), query)
	if err != nil {
		r.writeRouteError(ginCtx, err)
		return
	}
	httpx.WriteSuccess(ginCtx, http.StatusOK, result)
}

func (r routeRuntime) handleCreationMethods(ginCtx *gin.Context) {
	applicationGeneratedHandler{}.GetApplicationCreationMethods(bindGetApplicationCreationMethodsParams(ginCtx))
	result, err := r.service.CreationMethodCatalog(ginCtx.Request.Context())
	if err != nil {
		r.writeRouteError(ginCtx, err)
		return
	}
	httpx.WriteSuccess(ginCtx, http.StatusOK, toCreationMethodCatalogResponse(result))
}

func (r routeRuntime) handleDiscoveryCandidates(ginCtx *gin.Context) {
	applicationGeneratedHandler{}.GetApplicationDiscoveryCandidates(bindGetApplicationDiscoveryCandidatesParams(ginCtx))
	result, err := r.service.DiscoveryCandidates(ginCtx.Request.Context())
	if err != nil {
		r.writeRouteError(ginCtx, err)
		return
	}
	httpx.WriteSuccess(ginCtx, http.StatusOK, toDiscoveryCandidatesResponse(result))
}

func (r routeRuntime) handleManagedRoot(ginCtx *gin.Context) {
	applicationGeneratedHandler{}.GetApplicationManagedRoot(bindGetApplicationManagedRootParams(ginCtx))
	result, err := r.service.ManagedRoot(ginCtx.Request.Context())
	if err != nil {
		r.writeRouteError(ginCtx, err)
		return
	}
	httpx.WriteSuccess(ginCtx, http.StatusOK, toManagedRootResponse(result))
}

func (r routeRuntime) handleCreateValidate(ginCtx *gin.Context) {
	var request generated.PostApplicationCreateValidateJSONRequestBody
	if !bindJSON(ginCtx, r.ctx, &request) {
		return
	}
	applicationGeneratedHandler{}.PostApplicationCreateValidate(bindPostApplicationCreateValidateParams(ginCtx), request)
	managedRequest, err := toManagedCreateRequest(request)
	if err != nil {
		r.logManagedCreateFailure(ginCtx, "request_mapping", err)
		r.writeRouteError(ginCtx, err)
		return
	}
	result, err := r.service.ValidateManagedCreate(ginCtx.Request.Context(), managedRequest)
	if err != nil {
		r.logManagedCreateFailure(ginCtx, "validation", err)
		r.writeRouteError(ginCtx, err)
		return
	}
	httpx.WriteSuccess(ginCtx, http.StatusOK, toManagedCreateValidateResponse(result))
}

func (r routeRuntime) handleApplicationNameAvailability(ginCtx *gin.Context) {
	var request generated.PostApplicationNameAvailabilityJSONRequestBody
	if !bindJSON(ginCtx, r.ctx, &request) {
		return
	}
	applicationGeneratedHandler{}.PostApplicationNameAvailability(bindPostApplicationNameAvailabilityParams(ginCtx), request)
	result, err := r.service.CheckApplicationNameAvailability(ginCtx.Request.Context(), ApplicationNameAvailabilityRequest{ApplicationName: request.ApplicationName})
	if err != nil {
		r.writeRouteError(ginCtx, err)
		return
	}
	httpx.WriteSuccess(ginCtx, http.StatusOK, toApplicationNameAvailabilityResponse(result))
}

func (r routeRuntime) handleCreate(ginCtx *gin.Context) {
	var request generated.PostApplicationCreateJSONRequestBody
	if !bindJSON(ginCtx, r.ctx, &request) {
		return
	}
	applicationGeneratedHandler{}.PostApplicationCreate(bindPostApplicationCreateParams(ginCtx), request)
	managedRequest, err := toManagedCreateExecuteRequest(request)
	if err != nil {
		r.logManagedCreateFailure(ginCtx, "request_mapping", err)
		r.writeRouteError(ginCtx, err)
		return
	}
	result, err := r.service.CreateManagedApplication(ginCtx.Request.Context(), managedRequest, currentUserIDPointer(ginCtx))
	if err != nil {
		r.logManagedCreateFailure(ginCtx, "create", err)
		r.writeRouteError(ginCtx, err)
		return
	}
	httpx.WriteSuccess(ginCtx, http.StatusCreated, toManagedCreateResponse(result))
}

func (r routeRuntime) handleDetail(ginCtx *gin.Context) {
	projectID, generatedID, ok := r.bindApplicationRecordID(ginCtx)
	if !ok {
		return
	}
	applicationGeneratedHandler{}.GetApplication(generatedID, bindGetApplicationParams(ginCtx))
	result, err := r.service.Get(ginCtx.Request.Context(), projectID)
	if err != nil {
		r.writeRouteError(ginCtx, err)
		return
	}
	httpx.WriteSuccess(ginCtx, http.StatusOK, result)
}

func (r routeRuntime) handleOverview(ginCtx *gin.Context) {
	projectID, generatedID, ok := r.bindApplicationRecordID(ginCtx)
	if !ok {
		return
	}
	applicationGeneratedHandler{}.GetApplicationOverview(generatedID, bindGetApplicationOverviewParams(ginCtx))
	result, err := r.service.Overview(ginCtx.Request.Context(), projectID)
	if err != nil {
		r.writeRouteError(ginCtx, err)
		return
	}
	httpx.WriteSuccess(ginCtx, http.StatusOK, result)
}

func (r routeRuntime) handleServices(ginCtx *gin.Context) {
	projectID, generatedID, ok := r.bindApplicationRecordID(ginCtx)
	if !ok {
		return
	}
	applicationGeneratedHandler{}.GetApplicationServices(generatedID, bindGetApplicationServicesParams(ginCtx))
	result, err := r.service.Services(ginCtx.Request.Context(), projectID)
	if err != nil {
		r.writeRouteError(ginCtx, err)
		return
	}
	httpx.WriteSuccess(ginCtx, http.StatusOK, result)
}

func (r routeRuntime) handleLogs(ginCtx *gin.Context) {
	projectID, generatedID, ok := r.bindApplicationRecordID(ginCtx)
	if !ok {
		return
	}
	params, ok := bindGetApplicationLogsParams(ginCtx, r.ctx)
	if !ok {
		return
	}
	applicationGeneratedHandler{}.GetApplicationLogs(generatedID, params)
	result, err := r.service.Logs(ginCtx.Request.Context(), projectID, LogQuery{
		Tail:       intPtrValue(params.Tail),
		Since:      stringValue(params.Since),
		Timestamps: boolValue(params.Timestamps),
		Stdout:     boolValue(params.Stdout),
		Stderr:     boolValue(params.Stderr),
	})
	if err != nil {
		r.writeRouteError(ginCtx, err)
		return
	}
	httpx.WriteSuccess(ginCtx, http.StatusOK, result)
}

func (r routeRuntime) handleConfiguration(ginCtx *gin.Context) {
	projectID, generatedID, ok := r.bindApplicationRecordID(ginCtx)
	if !ok {
		return
	}
	applicationGeneratedHandler{}.GetApplicationConfiguration(generatedID, bindGetApplicationConfigurationParams(ginCtx))
	result, err := r.service.ConfigurationMetadata(ginCtx.Request.Context(), projectID)
	if err != nil {
		r.writeRouteError(ginCtx, err)
		return
	}
	httpx.WriteSuccess(ginCtx, http.StatusOK, toConfigurationMetadataResponse(result))
}

func (r routeRuntime) handleConfigurationPreview(ginCtx *gin.Context) {
	projectID, generatedID, ok := r.bindApplicationRecordID(ginCtx)
	if !ok {
		return
	}
	applicationGeneratedHandler{}.GetApplicationConfigurationPreview(generatedID, bindGetApplicationConfigurationPreviewParams(ginCtx))
	result, err := r.service.ConfigurationPreview(ginCtx.Request.Context(), projectID)
	if err != nil {
		r.writeRouteError(ginCtx, err)
		return
	}
	httpx.WriteSuccess(ginCtx, http.StatusOK, toConfigurationPreviewResponse(result))
}

func (r routeRuntime) handleProjectWorkspaceFiles(ginCtx *gin.Context) {
	projectID, generatedID, ok := r.bindApplicationRecordID(ginCtx)
	if !ok {
		return
	}
	query, ok := bindProjectWorkspaceFilesQuery(ginCtx, r.ctx)
	if !ok {
		return
	}
	applicationGeneratedHandler{}.GetApplicationFiles(generatedID, bindGetApplicationFilesParams(ginCtx, query))
	result, err := r.service.browseProjectFiles(ginCtx.Request.Context(), projectID, query)
	if err != nil {
		r.writeRouteError(ginCtx, err)
		return
	}
	httpx.WriteSuccess(ginCtx, http.StatusOK, toProjectWorkspaceFilesResponse(result))
}

func (r routeRuntime) handleProjectWorkspaceFileContent(ginCtx *gin.Context) {
	projectID, generatedID, ok := r.bindApplicationRecordID(ginCtx)
	if !ok {
		return
	}
	path, ok := bindProjectWorkspaceFilePath(ginCtx, r.ctx)
	if !ok {
		return
	}
	applicationGeneratedHandler{}.GetApplicationFileContent(generatedID, bindGetApplicationFileContentParams(ginCtx, path))
	result, err := r.service.projectFileContent(ginCtx.Request.Context(), projectID, path)
	if err != nil {
		r.writeRouteError(ginCtx, err)
		return
	}
	httpx.WriteSuccess(ginCtx, http.StatusOK, toProjectWorkspaceFileContentResponse(result))
}

func (r routeRuntime) handleSaveProjectWorkspaceFileContent(ginCtx *gin.Context) {
	projectID, generatedID, ok := r.bindApplicationRecordID(ginCtx)
	if !ok {
		return
	}
	path, ok := bindProjectWorkspaceFilePath(ginCtx, r.ctx)
	if !ok {
		return
	}
	var request generated.PutApplicationFileContentJSONRequestBody
	if !bindJSON(ginCtx, r.ctx, &request) {
		return
	}
	applicationGeneratedHandler{}.PutApplicationFileContent(generatedID, bindPutApplicationFileContentParams(ginCtx, path), request)
	result, err := r.service.saveProjectFileContent(ginCtx.Request.Context(), projectID, path, workspaceFileSaveRequest{
		Content: request.Content,
	})
	if err != nil {
		r.writeRouteError(ginCtx, err)
		return
	}
	httpx.WriteSuccess(ginCtx, http.StatusOK, toProjectWorkspaceFileSaveResponse(result))
}

func (r routeRuntime) handleProjectWorkspaceFileAnnotation(ginCtx *gin.Context) {
	projectID, generatedID, ok := r.bindApplicationRecordID(ginCtx)
	if !ok {
		return
	}
	path, ok := bindProjectWorkspaceFilePath(ginCtx, r.ctx)
	if !ok {
		return
	}
	var request generated.PutApplicationFileAnnotationJSONRequestBody
	if !bindJSON(ginCtx, r.ctx, &request) {
		return
	}
	applicationGeneratedHandler{}.PutApplicationFileAnnotation(generatedID, bindPutApplicationFileAnnotationParams(ginCtx, path), request)
	result, err := r.service.updateProjectWorkspaceAnnotation(
		ginCtx.Request.Context(),
		projectID,
		path,
		request.Annotation,
		currentUserIDPointer(ginCtx),
	)
	if err != nil {
		r.writeRouteError(ginCtx, err)
		return
	}
	httpx.WriteSuccess(ginCtx, http.StatusOK, generated.ApplicationFileTreeItem{
		Name:            result.Name,
		RelativePath:    result.RelativePath,
		NodeType:        generated.ApplicationFileTreeNodeType(result.NodeType),
		FileKind:        generated.ApplicationWorkspaceFileKind(result.FileKind),
		Editable:        result.Editable,
		LanguageHint:    result.LanguageHint,
		SizeBytes:       result.SizeBytes,
		HiddenByDefault: result.HiddenByDefault,
		HasChildren:     result.HasChildren,
		ApplicationNote: optionalString(result.ApplicationNote),
		Tooltip:         optionalString(result.Tooltip),
		TooltipSource:   optionalTooltipSource(result.TooltipSource),
	})
}

type workspaceEntryMutationHTTP struct {
	Path     string  `json:"path"`
	NodeType string  `json:"node_type"`
	Content  *string `json:"content"`
}
type workspaceEntryRenameHTTP struct {
	Path    string `json:"path"`
	NewPath string `json:"new_path"`
}

func (r routeRuntime) handleCreateProjectWorkspaceEntry(ginCtx *gin.Context) {
	projectID, _, ok := r.bindApplicationRecordID(ginCtx)
	if !ok {
		return
	}
	var request workspaceEntryMutationHTTP
	if !bindJSON(ginCtx, r.ctx, &request) {
		return
	}
	if err := r.service.createProjectWorkspaceEntry(ginCtx.Request.Context(), projectID, workspaceEntryCreateRequest(request)); err != nil {
		r.writeRouteError(ginCtx, err)
		return
	}
	ginCtx.Status(http.StatusCreated)
}

func (r routeRuntime) handleRenameProjectWorkspaceEntry(ginCtx *gin.Context) {
	projectID, _, ok := r.bindApplicationRecordID(ginCtx)
	if !ok {
		return
	}
	var request workspaceEntryRenameHTTP
	if !bindJSON(ginCtx, r.ctx, &request) {
		return
	}
	if err := r.service.renameProjectWorkspaceEntry(ginCtx.Request.Context(), projectID, workspaceEntryRenameRequest(request)); err != nil {
		r.writeRouteError(ginCtx, err)
		return
	}
	ginCtx.Status(http.StatusNoContent)
}

func (r routeRuntime) handleDeleteApplicationWorkspaceEntry(ginCtx *gin.Context) {
	projectID, _, ok := r.bindApplicationRecordID(ginCtx)
	if !ok {
		return
	}
	path := strings.TrimSpace(ginCtx.Query("path"))
	recursive, err := strconv.ParseBool(ginCtx.DefaultQuery("recursive", "false"))
	if path == "" || err != nil {
		r.writeInvalidArgumentError(ginCtx)
		return
	}
	if err := r.service.deleteProjectWorkspaceEntry(ginCtx.Request.Context(), projectID, path, recursive); err != nil {
		r.writeRouteError(ginCtx, err)
		return
	}
	ginCtx.Status(http.StatusNoContent)
}

func (r routeRuntime) handleRefresh(ginCtx *gin.Context) {
	projectID, generatedID, ok := r.bindApplicationRecordID(ginCtx)
	if !ok {
		return
	}
	applicationGeneratedHandler{}.PostApplicationRefresh(generatedID, bindPostApplicationRefreshParams(ginCtx))
	result, err := r.service.Refresh(ginCtx.Request.Context(), projectID, currentUserIDPointer(ginCtx))
	result.ApplicationID = generatedID
	if err != nil {
		r.writeRouteError(ginCtx, err)
		return
	}
	httpx.WriteSuccess(ginCtx, http.StatusOK, toActionResponse(result))
}

func (r routeRuntime) handleUp(ginCtx *gin.Context) {
	projectID, generatedID, ok := r.bindApplicationRecordID(ginCtx)
	if !ok {
		return
	}
	applicationGeneratedHandler{}.PostApplicationUp(generatedID, bindPostApplicationUpParams(ginCtx))
	result, err := r.service.Up(ginCtx.Request.Context(), projectID, currentUserIDPointer(ginCtx))
	result.ApplicationID = generatedID
	if err != nil {
		r.writeRouteErrorWithAction(ginCtx, err, result)
		return
	}
	httpx.WriteSuccess(ginCtx, http.StatusAccepted, toTaskReceiptResponse(result))
}

func (r routeRuntime) handleStop(ginCtx *gin.Context) {
	projectID, generatedID, ok := r.bindApplicationRecordID(ginCtx)
	if !ok {
		return
	}
	applicationGeneratedHandler{}.PostApplicationStop(generatedID, bindPostApplicationStopParams(ginCtx))
	result, err := r.service.Stop(ginCtx.Request.Context(), projectID, currentUserIDPointer(ginCtx))
	result.ApplicationID = generatedID
	if err != nil {
		r.writeRouteErrorWithAction(ginCtx, err, result)
		return
	}
	httpx.WriteSuccess(ginCtx, http.StatusAccepted, toTaskReceiptResponse(result))
}

func (r routeRuntime) handleRestart(ginCtx *gin.Context) {
	projectID, generatedID, ok := r.bindApplicationRecordID(ginCtx)
	if !ok {
		return
	}
	applicationGeneratedHandler{}.PostApplicationRestart(generatedID, bindPostApplicationRestartParams(ginCtx))
	result, err := r.service.Restart(ginCtx.Request.Context(), projectID, currentUserIDPointer(ginCtx))
	result.ApplicationID = generatedID
	if err != nil {
		r.writeRouteErrorWithAction(ginCtx, err, result)
		return
	}
	httpx.WriteSuccess(ginCtx, http.StatusAccepted, toTaskReceiptResponse(result))
}

func (r routeRuntime) handleRedeploy(ginCtx *gin.Context) {
	projectID, generatedID, ok := r.bindApplicationRecordID(ginCtx)
	if !ok {
		return
	}
	applicationGeneratedHandler{}.PostApplicationRedeploy(generatedID, bindPostApplicationRedeployParams(ginCtx))
	result, err := r.service.Redeploy(ginCtx.Request.Context(), projectID, currentUserIDPointer(ginCtx))
	result.ApplicationID = generatedID
	if err != nil {
		r.writeRouteErrorWithAction(ginCtx, err, result)
		return
	}
	httpx.WriteSuccess(ginCtx, http.StatusAccepted, toTaskReceiptResponse(result))
}

func (r routeRuntime) handleLifecycleConfiguration(ginCtx *gin.Context) {
	projectID, generatedID, ok := r.bindApplicationRecordID(ginCtx)
	if !ok {
		return
	}
	var request generated.ApplicationLifecycleConfigurationRequest
	if !bindJSON(ginCtx, r.ctx, &request) {
		return
	}
	applicationGeneratedHandler{}.PutApplicationLifecycleConfiguration(
		generatedID,
		bindPutApplicationLifecycleConfigurationParams(ginCtx),
		request,
	)
	result, err := r.service.UpdateLifecycleConfiguration(
		ginCtx.Request.Context(),
		projectID,
		toLifecycleConfigurationRequest(request),
		currentUserIDPointer(ginCtx),
	)
	if err != nil {
		r.writeRouteError(ginCtx, err)
		return
	}
	httpx.WriteSuccess(ginCtx, http.StatusOK, toProjectLifecycleConfigurationResponse(result))
}

func (r routeRuntime) handleBatchActions(ginCtx *gin.Context) {
	var request generated.ApplicationBatchActionRequest
	if !bindJSON(ginCtx, r.ctx, &request) {
		return
	}
	if !r.authorizeBatchAction(ginCtx, request.Action) {
		return
	}
	applicationGeneratedHandler{}.PostApplicationBatchActions(bindPostApplicationBatchActionsParams(ginCtx), request)
	projectIDs, err := r.resolveBatchApplicationRecordIDs(ginCtx, request.ApplicationIds)
	if err != nil {
		r.writeRouteError(ginCtx, err)
		return
	}
	result, err := r.service.BatchAction(ginCtx.Request.Context(), BatchActionRequest{
		Action:                    request.Action,
		ApplicationRecordIDs:      projectIDs,
		RemoveNamedVolumes:        boolValue(request.RemoveNamedVolumes),
		AutoUnregister:            boolValue(request.AutoUnregister),
		ImagePrune:                boolValue(request.ImagePrune),
		DeleteWorkspacePath:       boolValue(request.DeleteWorkspacePath),
		ConfirmComposeProjectName: request.ConfirmComposeProjectName,
		ActorID:                   currentUserIDPointer(ginCtx),
	})
	if err != nil {
		r.writeRouteError(ginCtx, err)
		return
	}
	for index := range result.Items {
		if index < len(request.ApplicationIds) {
			result.Items[index].ApplicationID = request.ApplicationIds[index]
		}
	}
	httpx.WriteSuccess(ginCtx, http.StatusOK, toBatchActionResponse(result))
}

func (r routeRuntime) resolveBatchApplicationRecordIDs(ginCtx *gin.Context, applicationIDs []string) ([]uint64, error) {
	repository, err := r.service.repositoryOrErr()
	if err != nil {
		return nil, err
	}
	lookup, ok := repository.(projectstore.ApplicationIDBatchLookupRepository)
	if !ok {
		return nil, errProjectServiceUnavailable
	}
	resolved, err := lookup.GetRecordIDsByApplicationIDs(ginCtx.Request.Context(), applicationIDs)
	if err != nil {
		return nil, err
	}
	projectIDs := make([]uint64, 0, len(applicationIDs))
	for _, applicationID := range applicationIDs {
		if !isApplicationID(applicationID) {
			return nil, errProjectInvalidArgument
		}
		projectID, ok := resolved[applicationID]
		if !ok {
			return nil, errProjectNotFound
		}
		projectIDs = append(projectIDs, projectID)
	}
	return projectIDs, nil
}

func (r routeRuntime) authorizeBatchAction(ginCtx *gin.Context, action generated.ApplicationBatchActionRequestAction) bool {
	permission, ok := batchActionPermission(action)
	if !ok {
		r.writeRouteError(ginCtx, errProjectInvalidArgument)
		return false
	}
	if strings.TrimSpace(permission) == "" {
		return true
	}
	requestAuth, ok := moduleapi.RequestAuthContextFromContext(ginCtx.Request.Context())
	if !ok {
		httpx.AbortLocalizedError(ginCtx, r.ctx.I18n, http.StatusUnauthorized, messagecontract.AuthTokenMissing.String(), nil)
		return false
	}
	if r.authorizer == nil {
		httpx.AbortAppError(ginCtx, r.ctx.I18n, r.ctx.Logger, errors.New("project authorizer is unavailable"))
		return false
	}
	if err := r.authorizer.Authorize(ginCtx.Request.Context(), requestAuth, permission); err != nil {
		switch {
		case errors.Is(err, moduleapi.ErrPermissionDenied):
			httpx.AbortLocalizedError(ginCtx, r.ctx.I18n, http.StatusForbidden, messagecontract.AuthForbidden.String(), nil)
		case errors.Is(err, moduleapi.ErrInvalidAccessToken):
			httpx.AbortLocalizedError(ginCtx, r.ctx.I18n, http.StatusUnauthorized, messagecontract.AuthTokenInvalid.String(), nil)
		case errors.Is(err, moduleapi.ErrUnauthenticated):
			httpx.AbortLocalizedError(ginCtx, r.ctx.I18n, http.StatusUnauthorized, messagecontract.AuthTokenMissing.String(), nil)
		default:
			httpx.AbortAppError(ginCtx, r.ctx.I18n, r.ctx.Logger, err)
		}
		return false
	}
	return true
}

// batchActionPermission 返回批量操作所需的权限及该操作是否受支持。
func batchActionPermission(action generated.ApplicationBatchActionRequestAction) (string, bool) {
	switch action {
	case generated.ApplicationBatchActionRequestActionStart,
		generated.ApplicationBatchActionRequestActionStop,
		generated.ApplicationBatchActionRequestActionRestart,
		generated.ApplicationBatchActionRequestActionRedeploy:
		return projectcontract.ApplicationLifecyclePermission.String(), true
	case generated.ApplicationBatchActionRequestActionUnregister,
		generated.ApplicationBatchActionRequestActionDestroy:
		return projectcontract.ApplicationDestroyPermission.String(), true
	default:
		return "", false
	}
}

func (r routeRuntime) handleUnregister(ginCtx *gin.Context) {
	projectID, generatedID, ok := r.bindApplicationRecordID(ginCtx)
	if !ok {
		return
	}
	applicationGeneratedHandler{}.PostApplicationUnregister(generatedID, bindPostApplicationUnregisterParams(ginCtx))
	result, err := r.service.Unregister(ginCtx.Request.Context(), projectID, currentUserIDPointer(ginCtx))
	result.ApplicationID = generatedID
	if err != nil {
		r.writeRouteErrorWithAction(ginCtx, err, result)
		return
	}
	httpx.WriteSuccess(ginCtx, http.StatusOK, toActionResponse(result))
}

func (r routeRuntime) handleDestroy(ginCtx *gin.Context) {
	projectID, generatedID, ok := r.bindApplicationRecordID(ginCtx)
	if !ok {
		return
	}
	var request generated.PostApplicationDestroyJSONRequestBody
	if !bindJSON(ginCtx, r.ctx, &request) {
		return
	}
	applicationGeneratedHandler{}.PostApplicationDestroy(generatedID, bindPostApplicationDestroyParams(ginCtx), request)
	result, err := r.service.Destroy(ginCtx.Request.Context(), projectID, DestroyRequest{
		RemoveNamedVolumes:        request.RemoveNamedVolumes,
		AutoUnregister:            boolValue(request.AutoUnregister),
		ImagePrune:                boolValue(request.ImagePrune),
		DeleteWorkspacePath:       request.DeleteWorkspace,
		ConfirmComposeProjectName: request.ConfirmApplicationId,
		ActorID:                   currentUserIDPointer(ginCtx),
	})
	result.ApplicationID = generatedID
	if err != nil {
		r.writeRouteErrorWithAction(ginCtx, err, result)
		return
	}
	httpx.WriteSuccess(ginCtx, http.StatusOK, toActionResponse(result))
}

func (r routeRuntime) writeRouteError(ginCtx *gin.Context, err error) {
	r.writeRouteErrorWithAction(ginCtx, err, ActionResult{})
}

func (r routeRuntime) writeRouteErrorWithAction(ginCtx *gin.Context, err error, action ActionResult) {
	if !r.writeHandledRouteError(ginCtx, err, action) {
		httpx.AbortAppError(ginCtx, r.ctx.I18n, r.ctx.Logger, err)
		return
	}
	ginCtx.Abort()
}

//nolint:cyclop // Explicit error classes preserve localized HTTP semantics at the route boundary.
func (r routeRuntime) writeHandledRouteError(ginCtx *gin.Context, err error, action ActionResult) bool {
	if r.writeProjectInputError(ginCtx, err) {
		return true
	}
	switch {
	case errors.Is(err, moduleapi.ErrTaskOwnerBusy):
		r.writeLocalizedActionError(ginCtx, http.StatusConflict, projectcontract.ApplicationConflict.String(), map[string]any{
			"code":         projectcontract.ApplicationConflict.String(),
			"actionResult": toActionResponse(action),
		})
	case errors.Is(err, errProjectFileNotFound):
		r.writeFileNotFoundError(ginCtx)
	case errors.Is(err, errProjectInvalidArgument):
		r.writeInvalidArgumentError(ginCtx)
	case errors.Is(err, errProjectInvalidCanonicalName):
		r.writeLocalizedActionError(ginCtx, http.StatusBadRequest, projectcontract.ApplicationInvalidComposeProjectName.String(), map[string]any{
			"code": projectcontract.ApplicationInvalidComposeProjectName.String(),
		})
	case r.writeProjectConflictError(ginCtx, err):
	case errors.Is(err, errProjectDestroyBlocked):
		r.writeLocalizedActionError(ginCtx, http.StatusConflict, projectcontract.ApplicationConflict.String(), map[string]any{
			"code":         projectcontract.ApplicationConflict.String(),
			"actionResult": toActionResponse(action),
		})
	case errors.Is(err, errProjectUnsupportedLifecycle), errors.Is(err, errProjectManagedFlow), errors.Is(err, errProjectRuntimeUnavailable):
		messageKey := projectcontract.ApplicationUnsupportedLifecycle
		if errors.Is(err, errProjectRuntimeUnavailable) {
			messageKey = projectcontract.ApplicationRuntimeUnavailable
		}
		r.writeLocalizedActionError(ginCtx, http.StatusConflict, messageKey.String(), map[string]any{
			"code":         mapLifecycleErrorCode(err),
			"actionResult": toActionResponse(action),
		})
	case errors.Is(err, projectstore.ErrTemplateNameOccupied):
		r.writeLocalizedProjectError(ginCtx, http.StatusConflict, projectcontract.ApplicationTemplateNameOccupied.String())
	case errors.Is(err, errProjectTemplateArchived), errors.Is(err, errProjectTemplateUnpublished), errors.Is(err, projectstore.ErrTemplateConflict), errors.Is(err, projectstore.ErrTemplateDraftNotFound), errors.Is(err, projectstore.ErrTemplatePublishedState):
		r.writeLocalizedProjectError(ginCtx, http.StatusConflict, projectcontract.ApplicationConflict.String())
	case errors.Is(err, projectstore.ErrTemplateNotFound):
		r.writeLocalizedProjectError(ginCtx, http.StatusNotFound, projectcontract.ApplicationTemplateNotFound.String())
	default:
		return false
	}
	return true
}

func (r routeRuntime) writeProjectInputError(ginCtx *gin.Context, err error) bool {
	for _, rule := range projectInputErrorRules {
		if !errors.Is(err, rule.target) {
			continue
		}
		r.writeLocalizedProjectError(ginCtx, rule.status, rule.code.String())
		return true
	}
	return false
}

func (r routeRuntime) logManagedCreateFailure(ginCtx *gin.Context, stage string, err error) {
	if r.ctx == nil {
		return
	}
	logsafe.Warn(r.ctx.Logger, "managed project create failed",
		zap.String("stage", stage),
		zap.String("requestId", httpx.EnsureRequestID(ginCtx)),
		zap.String("traceId", httpx.EnsureTraceID(ginCtx)),
		zap.Error(err),
	)
}

func (r routeRuntime) writeProjectConflictError(ginCtx *gin.Context, err error) bool {
	switch {
	case errors.Is(err, errProjectNotFound):
		r.writeLocalizedProjectError(ginCtx, http.StatusNotFound, projectcontract.ApplicationNotFound.String())
	case errors.Is(err, errProjectComposeNameOccupied):
		r.writeLocalizedProjectError(ginCtx, http.StatusConflict, projectcontract.ApplicationComposeProjectNameOccupied.String())
	case errors.Is(err, errProjectConflict):
		r.writeLocalizedProjectError(ginCtx, http.StatusConflict, projectcontract.ApplicationConflict.String())
	case errors.Is(err, errProjectDirectoryForbidden):
		r.writeLocalizedProjectError(ginCtx, http.StatusForbidden, projectcontract.ApplicationDirectoryBrowseForbidden.String())
	case errors.Is(err, errProjectInspectionExpired):
		r.writeLocalizedProjectError(ginCtx, http.StatusConflict, projectcontract.ApplicationInspectionExpired.String())
	case errors.Is(err, errProjectInspectionStale):
		r.writeLocalizedProjectError(ginCtx, http.StatusConflict, projectcontract.ApplicationInspectionStale.String())
	default:
		return false
	}
	return true
}

func (r routeRuntime) writeInvalidArgumentError(ginCtx *gin.Context) {
	r.writeLocalizedActionError(ginCtx, http.StatusBadRequest, projectcontract.ApplicationInvalidArgument.String(), map[string]any{"code": projectcontract.ApplicationInvalidArgument.String()})
}

func (r routeRuntime) writeFileNotFoundError(ginCtx *gin.Context) {
	r.writeLocalizedActionError(ginCtx, http.StatusNotFound, projectcontract.ApplicationInvalidFileID.String(), map[string]any{"code": projectcontract.ApplicationInvalidFileID.String()})
}

func (r routeRuntime) writeLocalizedProjectError(ginCtx *gin.Context, status int, code string) {
	r.writeLocalizedActionError(ginCtx, status, code, map[string]any{"code": code})
}

func (r routeRuntime) writeLocalizedActionError(ginCtx *gin.Context, status int, code string, details map[string]any) {
	httpx.WriteLocalizedErrorCode(ginCtx, r.ctx.I18n, status, code, projectErrorMessageKey(code), details)
}

type applicationGeneratedHandler struct{}

func (applicationGeneratedHandler) GetApplications(generated.GetApplicationsParams) {}
func (applicationGeneratedHandler) GetApplicationSavedViews(generated.GetApplicationSavedViewsParams) {
}
func (applicationGeneratedHandler) PostApplicationSavedView(generated.PostApplicationSavedViewParams, generated.PostApplicationSavedViewJSONRequestBody) {
}
func (applicationGeneratedHandler) PutApplicationSavedView(int64, generated.PutApplicationSavedViewParams, generated.PutApplicationSavedViewJSONRequestBody) {
}
func (applicationGeneratedHandler) DeleteApplicationSavedView(int64, generated.DeleteApplicationSavedViewParams) {
}
func (applicationGeneratedHandler) GetApplicationCreationMethods(generated.GetApplicationCreationMethodsParams) {
}
func (applicationGeneratedHandler) GetApplicationDiscoveryCandidates(generated.GetApplicationDiscoveryCandidatesParams) {
}
func (applicationGeneratedHandler) GetApplicationImportRuntimeCandidates(generated.GetApplicationImportRuntimeCandidatesParams) {
}
func (applicationGeneratedHandler) PostApplicationImportValidate(generated.PostApplicationImportValidateParams, generated.PostApplicationImportValidateJSONRequestBody) {
}
func (applicationGeneratedHandler) PostApplicationImportRuntimeInspect(generated.PostApplicationImportRuntimeInspectParams, generated.PostApplicationImportRuntimeInspectJSONRequestBody) {
}
func (applicationGeneratedHandler) PostApplicationImport(generated.PostApplicationImportParams, generated.PostApplicationImportJSONRequestBody) {
}
func (applicationGeneratedHandler) PostApplicationImportInspect(generated.PostApplicationImportInspectParams, generated.PostApplicationImportInspectJSONRequestBody) {
}
func (applicationGeneratedHandler) GetApplicationManagedRoot(generated.GetApplicationManagedRootParams) {
}
func (applicationGeneratedHandler) PostApplicationCreateValidate(generated.PostApplicationCreateValidateParams, generated.PostApplicationCreateValidateJSONRequestBody) {
}
func (applicationGeneratedHandler) PostApplicationNameAvailability(generated.PostApplicationNameAvailabilityParams, generated.PostApplicationNameAvailabilityJSONRequestBody) {
}
func (applicationGeneratedHandler) PostApplicationComposeContextReferences(generated.PostApplicationComposeContextReferencesParams, generated.PostApplicationComposeContextReferencesJSONRequestBody) {
}
func (applicationGeneratedHandler) PostApplicationCreate(generated.PostApplicationCreateParams, generated.PostApplicationCreateJSONRequestBody) {
}
func (applicationGeneratedHandler) GetApplication(string, generated.GetApplicationParams) {}
func (applicationGeneratedHandler) GetApplicationOverview(string, generated.GetApplicationOverviewParams) {
}
func (applicationGeneratedHandler) GetApplicationServices(string, generated.GetApplicationServicesParams) {
}
func (applicationGeneratedHandler) GetApplicationLogs(string, generated.GetApplicationLogsParams) {}
func (applicationGeneratedHandler) GetApplicationConfiguration(string, generated.GetApplicationConfigurationParams) {
}
func (applicationGeneratedHandler) GetApplicationConfigurationPreview(string, generated.GetApplicationConfigurationPreviewParams) {
}
func (applicationGeneratedHandler) GetApplicationFiles(string, generated.GetApplicationFilesParams) {}
func (applicationGeneratedHandler) GetApplicationFileContent(string, generated.GetApplicationFileContentParams) {
}
func (applicationGeneratedHandler) PutApplicationFileContent(string, generated.PutApplicationFileContentParams, generated.PutApplicationFileContentJSONRequestBody) {
}
func (applicationGeneratedHandler) PutApplicationFileAnnotation(string, generated.PutApplicationFileAnnotationParams, generated.PutApplicationFileAnnotationJSONRequestBody) {
}
func (applicationGeneratedHandler) PostApplicationRefresh(string, generated.PostApplicationRefreshParams) {
}
func (applicationGeneratedHandler) PostApplicationUp(string, generated.PostApplicationUpParams)     {}
func (applicationGeneratedHandler) PostApplicationStop(string, generated.PostApplicationStopParams) {}
func (applicationGeneratedHandler) PostApplicationRestart(string, generated.PostApplicationRestartParams) {
}
func (applicationGeneratedHandler) PostApplicationRedeploy(string, generated.PostApplicationRedeployParams) {
}
func (applicationGeneratedHandler) PutApplicationLifecycleConfiguration(string, generated.PutApplicationLifecycleConfigurationParams, generated.ApplicationLifecycleConfigurationRequest) {
}
func (applicationGeneratedHandler) PostApplicationBatchActions(generated.PostApplicationBatchActionsParams, generated.ApplicationBatchActionRequest) {
}
func (applicationGeneratedHandler) PostApplicationUnregister(string, generated.PostApplicationUnregisterParams) {
}
func (applicationGeneratedHandler) PostApplicationDestroy(string, generated.PostApplicationDestroyParams, generated.PostApplicationDestroyJSONRequestBody) {
}

// bindListParams 绑定项目列表查询参数和公共请求头。
// 它解析 source_type、drift_status、limit 和 offset，并在分页参数无效时中止请求。
//
// bindListParams 解析并校验项目列表查询参数。
// bindListParams 解析项目列表查询参数；参数无效时中止请求并返回 false，否则返回解析后的参数和 true。
func bindListParams(ginCtx *gin.Context, ctx *module.Context) (generated.GetApplicationsParams, bool) {
	locale, requestID := commonHeaders(ginCtx)
	params := generated.GetApplicationsParams{
		XGraftLocale: locale,
		XRequestId:   requestID,
	}
	filters, ok := bindListFilterParams(ginCtx, ctx)
	if !ok {
		return generated.GetApplicationsParams{}, false
	}
	params.SourceType = filters.SourceType
	params.DriftStatus = filters.DriftStatus
	params.DeploymentAdapterKind = filters.DeploymentAdapterKind
	params.Provider = filters.Provider
	params.RuntimeStatus = filters.RuntimeStatus
	query := ginCtx.Request.URL.Query()
	if params.Sort, ok = bindProjectListSort(query); !ok {
		abortInvalidQuery(ginCtx, ctx)
		return generated.GetApplicationsParams{}, false
	}
	if keyword := strings.TrimSpace(query.Get("keyword")); keyword != "" {
		params.Keyword = &keyword
	}
	if value := strings.TrimSpace(query.Get("runtime_target_id")); value != "" {
		parsed, err := strconv.ParseInt(value, 10, 64)
		if err != nil || parsed < 1 {
			abortInvalidQuery(ginCtx, ctx)
			return generated.GetApplicationsParams{}, false
		}
		params.RuntimeTargetId = &parsed
	}
	if params.Limit, ok = optionalIntQuery[generated.ApplicationListLimit](query.Get("limit"), minimumProjectListLimit, maxProjectListLimit); !ok {
		abortInvalidQuery(ginCtx, ctx)
		return generated.GetApplicationsParams{}, false
	}
	if params.Offset, ok = optionalIntQuery[generated.ApplicationListOffset](query.Get("offset"), 0, 0); !ok {
		abortInvalidQuery(ginCtx, ctx)
		return generated.GetApplicationsParams{}, false
	}
	return params, true
}

// projectListSortValues 返回通过规范查询键或方括号查询键提供的排序值。
func projectListSortValues(query url.Values) []string {
	values := append([]string(nil), query["sort"]...)
	values = append(values, query["sort[]"]...)
	return values
}

// 查询中未提供排序值时返回 nil；排序值无效或存在多个值时返回 false。
func bindProjectListSort(query url.Values) (*generated.ApplicationListSort, bool) {
	rawSorts := projectListSortValues(query)
	if len(rawSorts) == 0 {
		return nil, true
	}
	if len(rawSorts) > 1 {
		return nil, false
	}
	value := generated.GetApplicationsParamsSort(strings.TrimSpace(rawSorts[0]))
	if !value.Valid() {
		return nil, false
	}
	sorts := generated.ApplicationListSort{string(value)}
	return &sorts, true
}

// projectListSortParamValue 返回第一个排序值；未提供排序值时返回空字符串。
func projectListSortParamValue(values *generated.ApplicationListSort) string {
	if values == nil || len(*values) == 0 {
		return ""
	}
	return string((*values)[0])
}

// 返回包含有效筛选条件的参数；任一参数无效时中止请求并返回 false。
func bindListFilterParams(ginCtx *gin.Context, ctx *module.Context) (generated.GetApplicationsParams, bool) {
	query := ginCtx.Request.URL.Query()
	params := generated.GetApplicationsParams{}
	sourceKind, ok := optionalValidatedEnumQuery(query.Get("source_type"), generated.ApplicationSourceType.Valid)
	if !ok {
		abortInvalidQuery(ginCtx, ctx)
		return generated.GetApplicationsParams{}, false
	}
	driftStatus, ok := optionalValidatedEnumQuery(query.Get("drift_status"), generated.ApplicationDriftStatus.Valid)
	if !ok {
		abortInvalidQuery(ginCtx, ctx)
		return generated.GetApplicationsParams{}, false
	}
	params.SourceType = sourceKind
	params.DriftStatus = driftStatus
	deploymentAdapterKind, ok := optionalValidatedEnumQuery(query.Get("deployment_adapter_kind"), generated.ApplicationListApplicationType.Valid)
	if !ok {
		abortInvalidQuery(ginCtx, ctx)
		return generated.GetApplicationsParams{}, false
	}
	params.DeploymentAdapterKind = deploymentAdapterKind
	provider, ok := optionalValidatedEnumQuery(query.Get("provider"), generated.GetApplicationsParamsProvider.Valid)
	if !ok {
		abortInvalidQuery(ginCtx, ctx)
		return generated.GetApplicationsParams{}, false
	}
	params.Provider = provider
	runtimeStatus, ok := optionalValidatedEnumQuery(query.Get("runtime_status"), generated.ApplicationRuntimeStatus.Valid)
	if !ok {
		abortInvalidQuery(ginCtx, ctx)
		return generated.GetApplicationsParams{}, false
	}
	params.RuntimeStatus = runtimeStatus
	return params, true
}

func bindGetApplicationImportRuntimeCandidatesParams(
	ginCtx *gin.Context,
	ctx *module.Context,
) (generated.GetApplicationImportRuntimeCandidatesParams, bool) {
	locale, requestID := commonHeaders(ginCtx)
	query := ginCtx.Request.URL.Query()
	params := generated.GetApplicationImportRuntimeCandidatesParams{
		XGraftLocale: locale,
		XRequestId:   requestID,
	}
	if strings.TrimSpace(query.Get("keyword")) != "" {
		keyword := strings.TrimSpace(query.Get("keyword"))
		params.Keyword = &keyword
	}
	availability, ok := optionalValidatedEnumQuery(
		query.Get("availability"),
		generated.ApplicationImportRuntimeCandidateAvailability.Valid,
	)
	if !ok {
		abortInvalidQuery(ginCtx, ctx)
		return generated.GetApplicationImportRuntimeCandidatesParams{}, false
	}
	params.Availability = availability
	if params.Limit, ok = optionalIntQuery[generated.ApplicationImportRuntimeCandidateListLimit](
		query.Get("limit"),
		minimumProjectListLimit,
		maxProjectListLimit,
	); !ok {
		abortInvalidQuery(ginCtx, ctx)
		return generated.GetApplicationImportRuntimeCandidatesParams{}, false
	}
	if params.Offset, ok = optionalIntQuery[generated.ApplicationImportRuntimeCandidateListOffset](query.Get("offset"), 0, 0); !ok {
		abortInvalidQuery(ginCtx, ctx)
		return generated.GetApplicationImportRuntimeCandidatesParams{}, false
	}
	return params, true
}

func runtimeCandidateAvailabilityFromGenerated(
	value *generated.ApplicationImportRuntimeCandidateAvailability,
) *RuntimeImportCandidateAvailability {
	if value == nil {
		return nil
	}
	availability := RuntimeImportCandidateAvailability(*value)
	return &availability
}

func bindPostApplicationImportRuntimeInspectParams(ginCtx *gin.Context) generated.PostApplicationImportRuntimeInspectParams {
	locale, requestID := commonHeaders(ginCtx)
	return generated.PostApplicationImportRuntimeInspectParams{
		XGraftLocale: locale,
		XRequestId:   requestID,
	}
}

// bindJSON 绑定请求体中的 JSON 到目标对象。
//
// 绑定失败时中止请求，并返回字段为 body 的本地化无效参数错误。
func bindJSON[T any](ginCtx *gin.Context, ctx *module.Context, target *T) bool {
	if err := ginCtx.ShouldBindJSON(target); err != nil {
		httpx.AbortLocalizedError(ginCtx, ctx.I18n, http.StatusBadRequest, messagecontract.CommonInvalidArgument.String(), map[string]any{"field": "body"})
		return false
	}
	return true
}

// bindApplicationRecordID 将应用路由中的公开 Application ID 解析为应用存储和任务使用的私有数值键。
func (r routeRuntime) bindApplicationRecordID(ginCtx *gin.Context) (uint64, string, bool) {
	raw := strings.TrimSpace(ginCtx.Param("applicationId"))
	if !isApplicationID(raw) {
		httpx.WriteLocalizedErrorCode(ginCtx, r.ctx.I18n, http.StatusBadRequest, projectcontract.ApplicationInvalidID.String(), messagecontract.CommonInvalidArgument.String(), map[string]any{"field": "applicationId", "code": projectcontract.ApplicationInvalidID.String()})
		ginCtx.Abort()
		return 0, "", false
	}
	projectID, err := r.service.ResolveApplicationID(ginCtx.Request.Context(), raw)
	if err != nil {
		r.writeRouteError(ginCtx, err)
		return 0, "", false
	}
	if projectID == 0 {
		httpx.WriteLocalizedErrorCode(ginCtx, r.ctx.I18n, http.StatusBadRequest, projectcontract.ApplicationInvalidID.String(), messagecontract.CommonInvalidArgument.String(), map[string]any{"field": "applicationId", "code": projectcontract.ApplicationInvalidID.String()})
		ginCtx.Abort()
		return 0, "", false
	}
	return projectID, raw, true
}

// bindImportValidateParams 组装导入校验接口的请求参数，包含请求语言和请求 ID。
func bindImportValidateParams(ginCtx *gin.Context) generated.PostApplicationImportValidateParams {
	locale, requestID := commonHeaders(ginCtx)
	return generated.PostApplicationImportValidateParams{XGraftLocale: locale, XRequestId: requestID}
}

// bindPostApplicationImportParams 组装应用导入接口的请求参数。
// 它从请求头中提取 `XGraftLocale` 和 `XRequestId` 并填充到返回值中。
func bindPostApplicationImportParams(ginCtx *gin.Context) generated.PostApplicationImportParams {
	locale, requestID := commonHeaders(ginCtx)
	return generated.PostApplicationImportParams{XGraftLocale: locale, XRequestId: requestID}
}

// bindPostApplicationImportInspectParams 构造应用导入检查请求参数，并填充本地化语言和请求 ID。
func bindPostApplicationImportInspectParams(ginCtx *gin.Context) generated.PostApplicationImportInspectParams {
	locale, requestID := commonHeaders(ginCtx)
	return generated.PostApplicationImportInspectParams{XGraftLocale: locale, XRequestId: requestID}
}

// bindGetApplicationCreationMethodsParams 为创建方式目录组装公共请求头。
func bindGetApplicationCreationMethodsParams(ginCtx *gin.Context) generated.GetApplicationCreationMethodsParams {
	locale, requestID := commonHeaders(ginCtx)
	return generated.GetApplicationCreationMethodsParams{XGraftLocale: locale, XRequestId: requestID}
}

// bindGetApplicationDiscoveryCandidatesParams 构造应用发现候选列表请求参数。
//
// 它从请求头中提取语言和请求 ID，并填充到生成的参数中。
func bindGetApplicationDiscoveryCandidatesParams(ginCtx *gin.Context) generated.GetApplicationDiscoveryCandidatesParams {
	locale, requestID := commonHeaders(ginCtx)
	return generated.GetApplicationDiscoveryCandidatesParams{XGraftLocale: locale, XRequestId: requestID}
}

// 当分页参数无效时会中止当前请求并返回 false。
func bindImportDirectoryBrowseQuery(ginCtx *gin.Context, ctx *module.Context) (ImportDirectoryBrowseQuery, bool) {
	query := ginCtx.Request.URL.Query()
	limit, ok := optionalIntQuery[int](query.Get("limit"), 1, importDirectoryBrowseMaxLimit)
	if !ok {
		abortInvalidQuery(ginCtx, ctx)
		return ImportDirectoryBrowseQuery{}, false
	}
	offset, ok := optionalIntQuery[int](query.Get("offset"), 0, 0)
	if !ok {
		abortInvalidQuery(ginCtx, ctx)
		return ImportDirectoryBrowseQuery{}, false
	}
	return ImportDirectoryBrowseQuery{
		Provider: strings.TrimSpace(query.Get("provider")),
		RootID:   strings.TrimSpace(query.Get("root_id")),
		Path:     strings.TrimSpace(query.Get("path")),
		Limit:    intPtrValue(limit),
		Offset:   intPtrValue(offset),
		SortBy:   strings.TrimSpace(query.Get("sort")),
		Order:    strings.TrimSpace(query.Get("order")),
	}, true
}

// bindGetApplicationParams 生成获取应用接口的请求参数，包含语言和请求 ID。
func bindGetApplicationParams(ginCtx *gin.Context) generated.GetApplicationParams {
	locale, requestID := commonHeaders(ginCtx)
	return generated.GetApplicationParams{XGraftLocale: locale, XRequestId: requestID}
}

// bindGetApplicationServicesParams 构造获取应用服务列表请求的公共参数。
func bindGetApplicationServicesParams(ginCtx *gin.Context) generated.GetApplicationServicesParams {
	locale, requestID := commonHeaders(ginCtx)
	return generated.GetApplicationServicesParams{XGraftLocale: locale, XRequestId: requestID}
}

// bindGetApplicationLogsParams 绑定应用日志查询参数，并在参数无效时中止请求。
// 返回解析后的参数和参数是否有效。
func bindGetApplicationLogsParams(ginCtx *gin.Context, ctx *module.Context) (generated.GetApplicationLogsParams, bool) {
	locale, requestID := commonHeaders(ginCtx)
	params := generated.GetApplicationLogsParams{XGraftLocale: locale, XRequestId: requestID}
	if value, ok := optionalIntQuery[generated.ApplicationLogsTail](ginCtx.Query("tail"), 1, maxProjectLogsTail); ok {
		params.Tail = value
	} else {
		abortInvalidQuery(ginCtx, ctx)
		return generated.GetApplicationLogsParams{}, false
	}
	if value := strings.TrimSpace(ginCtx.Query("since")); value != "" {
		params.Since = &value
	}
	if value, ok := optionalBoolQuery(ginCtx.Query("timestamps")); ok {
		params.Timestamps = value
	} else {
		abortInvalidQuery(ginCtx, ctx)
		return generated.GetApplicationLogsParams{}, false
	}
	if value, ok := optionalBoolQuery(ginCtx.Query("stdout")); ok {
		params.Stdout = value
	} else {
		abortInvalidQuery(ginCtx, ctx)
		return generated.GetApplicationLogsParams{}, false
	}
	if value, ok := optionalBoolQuery(ginCtx.Query("stderr")); ok {
		params.Stderr = value
	} else {
		abortInvalidQuery(ginCtx, ctx)
		return generated.GetApplicationLogsParams{}, false
	}
	return params, true
}

func bindGetApplicationOverviewParams(ginCtx *gin.Context) generated.GetApplicationOverviewParams {
	locale, requestID := commonHeaders(ginCtx)
	return generated.GetApplicationOverviewParams{XGraftLocale: locale, XRequestId: requestID}
}

// bindGetApplicationConfigurationParams 组装获取应用配置接口的请求参数。
//
// 它从请求头提取 locale 和 request ID，并填充到对应的生成参数中。
func bindGetApplicationConfigurationParams(ginCtx *gin.Context) generated.GetApplicationConfigurationParams {
	locale, requestID := commonHeaders(ginCtx)
	return generated.GetApplicationConfigurationParams{XGraftLocale: locale, XRequestId: requestID}
}

// bindGetApplicationConfigurationPreviewParams 构造应用配置预览接口的公共请求参数。
// 它包含从请求头提取的语言区域和请求 ID。
func bindGetApplicationConfigurationPreviewParams(ginCtx *gin.Context) generated.GetApplicationConfigurationPreviewParams {
	locale, requestID := commonHeaders(ginCtx)
	return generated.GetApplicationConfigurationPreviewParams{XGraftLocale: locale, XRequestId: requestID}
}

func bindProjectWorkspaceFilesQuery(ginCtx *gin.Context, ctx *module.Context) (workspaceFileBrowseQuery, bool) {
	query := ginCtx.Request.URL.Query()
	showHidden, ok := optionalBoolQuery(query.Get("show_hidden"))
	if !ok {
		abortInvalidQuery(ginCtx, ctx)
		return workspaceFileBrowseQuery{}, false
	}
	return workspaceFileBrowseQuery{
		Path:       strings.TrimSpace(query.Get("path")),
		ShowHidden: boolValue(showHidden),
	}, true
}

func bindProjectWorkspaceFilePath(ginCtx *gin.Context, ctx *module.Context) (string, bool) {
	path := strings.TrimSpace(ginCtx.Query("path"))
	if path == "" {
		abortInvalidQuery(ginCtx, ctx)
		return "", false
	}
	return path, true
}

func bindGetApplicationFilesParams(ginCtx *gin.Context, query workspaceFileBrowseQuery) generated.GetApplicationFilesParams {
	locale, requestID := commonHeaders(ginCtx)
	params := generated.GetApplicationFilesParams{XGraftLocale: locale, XRequestId: requestID}
	if trimmed := strings.TrimSpace(query.Path); trimmed != "" {
		params.Path = &trimmed
	}
	if query.ShowHidden {
		showHidden := true
		params.ShowHidden = &showHidden
	}
	return params
}

func bindGetApplicationFileContentParams(ginCtx *gin.Context, path string) generated.GetApplicationFileContentParams {
	locale, requestID := commonHeaders(ginCtx)
	queryPath := generated.ApplicationWorkspacePathQuery(path)
	return generated.GetApplicationFileContentParams{XGraftLocale: locale, XRequestId: requestID, Path: &queryPath}
}

func bindPutApplicationFileContentParams(ginCtx *gin.Context, path string) generated.PutApplicationFileContentParams {
	locale, requestID := commonHeaders(ginCtx)
	queryPath := generated.ApplicationWorkspacePathQuery(path)
	return generated.PutApplicationFileContentParams{XGraftLocale: locale, XRequestId: requestID, Path: &queryPath}
}

// bindPutApplicationFileAnnotationParams 构造更新应用文件注释所需的请求参数。
//
// Path 会被编码为工作区路径查询参数。
func bindPutApplicationFileAnnotationParams(ginCtx *gin.Context, path string) generated.PutApplicationFileAnnotationParams {
	locale, requestID := commonHeaders(ginCtx)
	queryPath := generated.ApplicationWorkspacePathQuery(path)
	return generated.PutApplicationFileAnnotationParams{XGraftLocale: locale, XRequestId: requestID, Path: &queryPath}
}

// bindGetApplicationManagedRootParams 构造获取托管根信息请求的公共参数。
//
// 它包含请求的语言环境和请求 ID。
func bindGetApplicationManagedRootParams(ginCtx *gin.Context) generated.GetApplicationManagedRootParams {
	locale, requestID := commonHeaders(ginCtx)
	return generated.GetApplicationManagedRootParams{XGraftLocale: locale, XRequestId: requestID}
}

// bindPostApplicationCreateValidateParams 构造应用创建校验接口的公共请求参数。
// 它包含语言信息和请求 ID。
func bindPostApplicationCreateValidateParams(ginCtx *gin.Context) generated.PostApplicationCreateValidateParams {
	locale, requestID := commonHeaders(ginCtx)
	return generated.PostApplicationCreateValidateParams{XGraftLocale: locale, XRequestId: requestID}
}

func bindPostApplicationNameAvailabilityParams(ginCtx *gin.Context) generated.PostApplicationNameAvailabilityParams {
	locale, requestID := commonHeaders(ginCtx)
	return generated.PostApplicationNameAvailabilityParams{XGraftLocale: locale, XRequestId: requestID}
}

func bindPostApplicationComposeContextReferencesParams(ginCtx *gin.Context) generated.PostApplicationComposeContextReferencesParams {
	locale, requestID := commonHeaders(ginCtx)
	return generated.PostApplicationComposeContextReferencesParams{XGraftLocale: locale, XRequestId: requestID}
}

// bindPostApplicationCreateParams 构造应用创建请求的公共请求头参数。
func bindPostApplicationCreateParams(ginCtx *gin.Context) generated.PostApplicationCreateParams {
	locale, requestID := commonHeaders(ginCtx)
	return generated.PostApplicationCreateParams{XGraftLocale: locale, XRequestId: requestID}
}

// bindPostApplicationRefreshParams 构造应用刷新接口的请求头参数。
func bindPostApplicationRefreshParams(ginCtx *gin.Context) generated.PostApplicationRefreshParams {
	locale, requestID := commonHeaders(ginCtx)
	return generated.PostApplicationRefreshParams{XGraftLocale: locale, XRequestId: requestID}
}

// bindPostApplicationUpParams 组装应用启动接口的请求参数，包含语言环境和请求 ID。
func bindPostApplicationUpParams(ginCtx *gin.Context) generated.PostApplicationUpParams {
	locale, requestID := commonHeaders(ginCtx)
	return generated.PostApplicationUpParams{XGraftLocale: locale, XRequestId: requestID}
}

// bindPostApplicationStopParams 组装应用停止接口的公共请求参数。
// 它包含请求的语言标识和请求 ID。
func bindPostApplicationStopParams(ginCtx *gin.Context) generated.PostApplicationStopParams {
	locale, requestID := commonHeaders(ginCtx)
	return generated.PostApplicationStopParams{XGraftLocale: locale, XRequestId: requestID}
}

// bindPostApplicationRestartParams 构造应用重启接口的公共请求参数。
// 其中包含从请求头提取的语言环境和请求 ID。
func bindPostApplicationRestartParams(ginCtx *gin.Context) generated.PostApplicationRestartParams {
	locale, requestID := commonHeaders(ginCtx)
	return generated.PostApplicationRestartParams{XGraftLocale: locale, XRequestId: requestID}
}

func bindPostApplicationRedeployParams(ginCtx *gin.Context) generated.PostApplicationRedeployParams {
	locale, requestID := commonHeaders(ginCtx)
	return generated.PostApplicationRedeployParams{XGraftLocale: locale, XRequestId: requestID}
}

func bindPutApplicationLifecycleConfigurationParams(ginCtx *gin.Context) generated.PutApplicationLifecycleConfigurationParams {
	locale, requestID := commonHeaders(ginCtx)
	return generated.PutApplicationLifecycleConfigurationParams{XGraftLocale: locale, XRequestId: requestID}
}

func bindPostApplicationBatchActionsParams(ginCtx *gin.Context) generated.PostApplicationBatchActionsParams {
	locale, requestID := commonHeaders(ginCtx)
	return generated.PostApplicationBatchActionsParams{XGraftLocale: locale, XRequestId: requestID}
}

// bindPostApplicationUnregisterParams 构造应用注销接口的请求参数。
// 它包含请求语言和请求 ID。
func bindPostApplicationUnregisterParams(ginCtx *gin.Context) generated.PostApplicationUnregisterParams {
	locale, requestID := commonHeaders(ginCtx)
	return generated.PostApplicationUnregisterParams{XGraftLocale: locale, XRequestId: requestID}
}

// bindPostApplicationDestroyParams 构造应用销毁接口的公共请求参数。
func bindPostApplicationDestroyParams(ginCtx *gin.Context) generated.PostApplicationDestroyParams {
	locale, requestID := commonHeaders(ginCtx)
	return generated.PostApplicationDestroyParams{XGraftLocale: locale, XRequestId: requestID}
}

// 返回语言环境头与请求 ID 的指针；请求 ID 会在缺失时生成并写回请求上下文。
func commonHeaders(ginCtx *gin.Context) (*string, *string) {
	locale := ginCtx.GetHeader(string(httpheader.Locale))
	requestID := httpx.EnsureRequestID(ginCtx)
	return &locale, &requestID
}

// optionalTypedQuery 将查询字符串转换为指定字符串类型的指针。
// 空白字符串返回 nil。
func optionalTypedQuery[T ~string](raw string) *T {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil
	}
	value := T(trimmed)
	return &value
}

func optionalValidatedEnumQuery[T ~string](raw string, validate func(T) bool) (*T, bool) {
	value := optionalTypedQuery[T](raw)
	if value == nil {
		return nil, true
	}
	if validate == nil || !validate(*value) {
		return nil, false
	}
	return value, true
}

// optionalIntQuery 将原始字符串解析为整数类型的可选查询值，并校验其取值范围。
// 为空字符串时返回 nil 和 true；解析失败、低于最小值或高于最大值时返回 false。
func optionalIntQuery[T ~int](raw string, min int, max int) (*T, bool) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, true
	}
	value, err := strconv.Atoi(trimmed)
	if err != nil {
		return nil, false
	}
	if value < min {
		return nil, false
	}
	if max > 0 && value > max {
		return nil, false
	}
	typed := T(value)
	return &typed, true
}

func optionalBoolQuery(raw string) (*bool, bool) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, true
	}
	value, err := strconv.ParseBool(trimmed)
	if err != nil {
		return nil, false
	}
	return &value, true
}

// abortInvalidQuery 以“查询参数无效”返回本地化的 400 错误并中止请求。
func abortInvalidQuery(ginCtx *gin.Context, ctx *module.Context) {
	httpx.AbortLocalizedError(ginCtx, ctx.I18n, http.StatusBadRequest, messagecontract.CommonInvalidArgument.String(), map[string]any{"field": "query"})
}

// intPtrValue 将整数指针转换为 int，并在为空时返回 0。
func intPtrValue[T ~int](value *T) int {
	if value == nil {
		return 0
	}
	return int(*value)
}

// stringPtrValue 将字符串指针转换为字符串值。
// 当指针为 nil 时返回空字符串。
func stringPtrValue[T ~string](value *T) string {
	if value == nil {
		return ""
	}
	return string(*value)
}

func boolValue(value *bool) bool {
	return value != nil && *value
}

// toImportRequest 将导入校验请求转换为 ImportRequest。
// 它会复制配置文件与环境文件列表，并附带当前请求中的操作者 ID。
func toImportRequest(ginCtx *gin.Context, request generated.ApplicationImportValidateRequest) ImportRequest {
	return ImportRequest{
		WorkspacePath:              request.WorkspacePath,
		DisplayName:                request.DisplayName,
		ComposeFiles:               slicePtrValue(request.ComposeFiles),
		EnvFiles:                   slicePtrValue(request.EnvFiles),
		ComposeProjectNameOverride: request.ComposeProjectNameOverride,
		ActorID:                    currentUserIDPointer(ginCtx),
	}
}

// slicePtrValue 将字符串切片指针转换为字符串切片，并复制底层数据。
// 当输入为 nil 时，返回 nil。
func slicePtrValue(value *[]string) []string {
	if value == nil {
		return nil
	}
	return append([]string(nil), (*value)...)
}

// currentUserIDPointer 从请求上下文中提取当前认证用户的 ID。
// 当请求、认证上下文或用户信息不可用时，返回 nil。
// currentUserIDPointer 获取当前请求认证用户的 ID 指针。
// 当请求上下文、认证信息或用户信息不可用时返回 nil。
func currentUserIDPointer(ginCtx *gin.Context) *uint64 {
	if ginCtx == nil || ginCtx.Request == nil {
		return nil
	}
	auth, ok := moduleapi.RequestAuthContextFromContext(ginCtx.Request.Context())
	if !ok || auth.User == nil {
		return nil
	}
	userID := auth.User.ID
	return &userID
}

// currentUserID 返回已认证用户的非零 ID 及其是否可用。
func currentUserID(ginCtx *gin.Context) (uint64, bool) {
	value := currentUserIDPointer(ginCtx)
	if value == nil || *value == 0 {
		return 0, false
	}
	return *value, true
}

// bindSavedViewID 解析并校验保存视图路由参数，返回有效的非零视图 ID。
func bindSavedViewID(ginCtx *gin.Context) (uint64, bool) {
	value, err := strconv.ParseUint(strings.TrimSpace(ginCtx.Param("viewId")), 10, 64)
	return value, err == nil && value > 0
}

// generatedSavedViewID 将有效的无符号保存视图 ID 转换为 int64。
// 返回转换后的 ID 及其有效性。
func generatedSavedViewID(value uint64) (int64, bool) {
	if value == 0 || value > math.MaxInt64 {
		return 0, false
	}
	return int64(value), true
}

func (r routeRuntime) writeSavedViewError(ginCtx *gin.Context, err error) {
	switch {
	case errors.Is(err, errProjectConflict):
		r.writeLocalizedProjectError(ginCtx, http.StatusConflict, projectcontract.ApplicationSavedViewConflict.String())
	case errors.Is(err, errProjectNotFound):
		r.writeLocalizedProjectError(ginCtx, http.StatusNotFound, projectcontract.ApplicationSavedViewNotFound.String())
	default:
		r.writeLocalizedProjectError(ginCtx, http.StatusBadRequest, projectcontract.ApplicationSavedViewInvalid.String())
	}
	ginCtx.Abort()
}

// mapLifecycleErrorCode 将生命周期错误映射为对应的错误码字符串。
// 当错误为 errProjectManagedFlow 时返回 ApplicationManagedFlowUnsupported，
// 否则返回 ApplicationUnsupportedLifecycle。
func mapLifecycleErrorCode(err error) string {
	if errors.Is(err, errProjectManagedFlow) {
		return projectcontract.ApplicationManagedFlowUnsupported.String()
	}
	return projectcontract.ApplicationUnsupportedLifecycle.String()
}

func projectErrorMessageKey(code string) string {
	code = strings.TrimSpace(code)
	if code == "" {
		return messagecontract.CommonInvalidArgument.String()
	}
	return code
}

// resolveAuthService 从服务容器中解析 AuthService。
// 它返回解析到的认证服务实例或错误。
func resolveAuthService(ctx *module.Context) (moduleapi.AuthService, error) {
	return module.ResolveService[moduleapi.AuthService](ctx.Services, (*moduleapi.AuthService)(nil))
}

// resolveAuthorizer 从服务容器中解析鉴权器。
// 它返回注册到 ctx.Services 中的 moduleapi.Authorizer 实现。
//
// @returns 解析到的 moduleapi.Authorizer 实例，或解析失败时的错误。
func resolveAuthorizer(ctx *module.Context) (moduleapi.Authorizer, error) {
	return module.ResolveService[moduleapi.Authorizer](ctx.Services, (*moduleapi.Authorizer)(nil))
}
