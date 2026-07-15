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

	"graft/server/internal/contract/httpheader"
	messagecontract "graft/server/internal/contract/message"
	generated "graft/server/internal/contract/openapi/generated"
	"graft/server/internal/httpx"
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

const minimumProjectListLimit = 1

// registerRoutes 为项目模块注册路由并挂载权限校验与请求追踪中间件。
// 当路由器不可用时直接返回；当服务缺失时返回错误。
// registerRoutes 注册 project 模块的 HTTP 路由，并为各路由安装请求 ID、审计和权限校验中间件。
// registerRoutes 注册项目模块的 HTTP 路由及其权限中间件。
// registerRoutes 注册项目模块的 HTTP 路由。
// registerRoutes 注册项目 API 路由及其请求 ID、权限校验中间件。
// registerRoutes 注册项目 API 路由及其权限与请求 ID 中间件。
// 当上下文或路由器为空时跳过注册；项目服务缺失或认证依赖解析失败时返回错误。
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
	group := ctx.Router.Group(projectcontract.ProjectAPIGroup)
	group.Use(httpx.RequestIDMiddleware())
	group.GET(projectcontract.ProjectCollectionRoute, httpx.RequirePermission(ctx.I18n, authService, authorizer, projectcontract.ProjectViewPermission.String(), publisher), routes.handleList)
	group.GET(projectcontract.ProjectSavedViewsRoute, httpx.RequirePermission(ctx.I18n, authService, authorizer, projectcontract.ProjectViewPermission.String(), publisher), routes.handleSavedViewList)
	group.POST(projectcontract.ProjectSavedViewsRoute, httpx.RequirePermission(ctx.I18n, authService, authorizer, projectcontract.ProjectViewPermission.String(), publisher), routes.handleSavedViewCreate)
	group.PUT(projectcontract.ProjectSavedViewRoute, httpx.RequirePermission(ctx.I18n, authService, authorizer, projectcontract.ProjectViewPermission.String(), publisher), routes.handleSavedViewUpdate)
	group.DELETE(projectcontract.ProjectSavedViewRoute, httpx.RequirePermission(ctx.I18n, authService, authorizer, projectcontract.ProjectViewPermission.String(), publisher), routes.handleSavedViewDelete)
	group.POST(projectcontract.ProjectImportValidateRoute, httpx.RequirePermission(ctx.I18n, authService, authorizer, projectcontract.ProjectImportPermission.String(), publisher), routes.handleImportValidate)
	group.GET(projectcontract.ProjectImportRuntimeCandidatesRoute, httpx.RequirePermission(ctx.I18n, authService, authorizer, projectcontract.ProjectImportPermission.String(), publisher), routes.handleImportRuntimeCandidates)
	group.POST(projectcontract.ProjectImportRuntimeInspectRoute, httpx.RequirePermission(ctx.I18n, authService, authorizer, projectcontract.ProjectImportPermission.String(), publisher), routes.handleImportRuntimeInspect)
	group.POST(projectcontract.ProjectImportInspectRoute, httpx.RequirePermission(ctx.I18n, authService, authorizer, projectcontract.ProjectImportPermission.String(), publisher), routes.handleImportInspect)
	group.POST(projectcontract.ProjectImportRoute, httpx.RequirePermission(ctx.I18n, authService, authorizer, projectcontract.ProjectImportPermission.String(), publisher), routes.handleImport)
	group.GET(projectcontract.ProjectImportDirectorySourcesRoute, httpx.RequirePermission(ctx.I18n, authService, authorizer, projectcontract.ProjectImportPermission.String(), publisher), routes.handleImportDirectorySources)
	group.GET(projectcontract.ProjectImportDirectoriesRoute, httpx.RequirePermission(ctx.I18n, authService, authorizer, projectcontract.ProjectImportPermission.String(), publisher), routes.handleImportDirectories)
	group.GET(projectcontract.ProjectCreationMethodsRoute, httpx.RequirePermission(ctx.I18n, authService, authorizer, projectcontract.ProjectCreationMethodViewPermission.String(), publisher), routes.handleCreationMethods)
	group.GET(projectcontract.ProjectComposeRuntimeTargetsRoute, httpx.RequirePermission(ctx.I18n, authService, authorizer, projectcontract.ProjectCreatePermission.String(), publisher), routes.handleComposeRuntimeTargets)
	group.GET(projectcontract.ProjectDiscoveryCandidatesRoute, httpx.RequirePermission(ctx.I18n, authService, authorizer, projectcontract.ProjectDiscoveryViewPermission.String(), publisher), routes.handleDiscoveryCandidates)
	group.GET(projectcontract.ProjectManagedRootRoute, httpx.RequirePermission(ctx.I18n, authService, authorizer, projectcontract.ProjectCreatePermission.String(), publisher), routes.handleManagedRoot)
	group.POST(projectcontract.ProjectCreateValidateRoute, httpx.RequirePermission(ctx.I18n, authService, authorizer, projectcontract.ProjectCreatePermission.String(), publisher), routes.handleCreateValidate)
	group.POST(projectcontract.ProjectCreateRoute, httpx.RequirePermission(ctx.I18n, authService, authorizer, projectcontract.ProjectCreatePermission.String(), publisher), routes.handleCreate)
	group.POST(projectcontract.ProjectCreateTemplateValidateRoute, httpx.RequirePermission(ctx.I18n, authService, authorizer, projectcontract.ProjectCreatePermission.String(), publisher), routes.handleTemplateCreateValidate)
	group.POST(projectcontract.ProjectCreateTemplateRoute, httpx.RequirePermission(ctx.I18n, authService, authorizer, projectcontract.ProjectCreatePermission.String(), publisher), routes.handleTemplateCreate)
	group.GET(projectcontract.ProjectWorkspaceDefaultsRoute, httpx.RequirePermission(ctx.I18n, authService, authorizer, projectcontract.ProjectCreatePermission.String(), publisher), routes.handleWorkspaceDefaults)
	group.GET(projectcontract.ProjectDetailRoute, httpx.RequirePermission(ctx.I18n, authService, authorizer, projectcontract.ProjectViewPermission.String(), publisher), routes.handleDetail)
	group.GET(projectcontract.ProjectOverviewRoute, httpx.RequirePermission(ctx.I18n, authService, authorizer, projectcontract.ProjectViewPermission.String(), publisher), routes.handleOverview)
	group.GET(projectcontract.ProjectServicesRoute, httpx.RequirePermission(ctx.I18n, authService, authorizer, projectcontract.ProjectViewPermission.String(), publisher), routes.handleServices)
	group.GET(projectcontract.ProjectLogsRoute, httpx.RequirePermission(ctx.I18n, authService, authorizer, projectcontract.ProjectViewPermission.String(), publisher), routes.handleLogs)
	group.GET(projectcontract.ProjectConfigurationRoute, httpx.RequirePermission(ctx.I18n, authService, authorizer, projectcontract.ProjectViewPermission.String(), publisher), routes.handleConfiguration)
	group.GET(projectcontract.ProjectConfigurationPreviewRoute, httpx.RequirePermission(ctx.I18n, authService, authorizer, projectcontract.ProjectViewPermission.String(), publisher), routes.handleConfigurationPreview)
	group.GET(projectcontract.ProjectWorkspaceFilesRoute, httpx.RequirePermission(ctx.I18n, authService, authorizer, projectcontract.ProjectViewPermission.String(), publisher), routes.handleProjectWorkspaceFiles)
	group.GET(projectcontract.ProjectWorkspaceFileContentRoute, httpx.RequirePermission(ctx.I18n, authService, authorizer, projectcontract.ProjectViewPermission.String(), publisher), routes.handleProjectWorkspaceFileContent)
	group.PUT(projectcontract.ProjectWorkspaceFileContentRoute, httpx.RequirePermission(ctx.I18n, authService, authorizer, projectcontract.ProjectDeployPermission.String(), publisher), routes.handleSaveProjectWorkspaceFileContent)
	group.PUT(projectcontract.ProjectWorkspaceFileAnnotationRoute, httpx.RequirePermission(ctx.I18n, authService, authorizer, projectcontract.ProjectDeployPermission.String(), publisher), routes.handleProjectWorkspaceFileAnnotation)
	group.POST(projectcontract.ProjectWorkspaceEntryRoute, httpx.RequirePermission(ctx.I18n, authService, authorizer, projectcontract.ProjectDeployPermission.String(), publisher), routes.handleCreateProjectWorkspaceEntry)
	group.POST(projectcontract.ProjectWorkspaceRenameRoute, httpx.RequirePermission(ctx.I18n, authService, authorizer, projectcontract.ProjectDeployPermission.String(), publisher), routes.handleRenameProjectWorkspaceEntry)
	group.DELETE(projectcontract.ProjectWorkspaceEntryRoute, httpx.RequirePermission(ctx.I18n, authService, authorizer, projectcontract.ProjectDeployPermission.String(), publisher), routes.handleDeleteProjectWorkspaceEntry)
	group.POST(projectcontract.ProjectRefreshRoute, httpx.RequirePermission(ctx.I18n, authService, authorizer, projectcontract.ProjectRefreshPermission.String(), publisher), routes.handleRefresh)
	group.POST(projectcontract.ProjectDeployRoute, httpx.RequirePermission(ctx.I18n, authService, authorizer, projectcontract.ProjectDeployPermission.String(), publisher), routes.handleDeploy)
	group.POST(projectcontract.ProjectUpRoute, httpx.RequirePermission(ctx.I18n, authService, authorizer, projectcontract.ProjectLifecyclePermission.String(), publisher), routes.handleUp)
	group.POST(projectcontract.ProjectStopRoute, httpx.RequirePermission(ctx.I18n, authService, authorizer, projectcontract.ProjectLifecyclePermission.String(), publisher), routes.handleStop)
	group.POST(projectcontract.ProjectRestartRoute, httpx.RequirePermission(ctx.I18n, authService, authorizer, projectcontract.ProjectLifecyclePermission.String(), publisher), routes.handleRestart)
	group.POST(projectcontract.ProjectRedeployRoute, httpx.RequirePermission(ctx.I18n, authService, authorizer, projectcontract.ProjectLifecyclePermission.String(), publisher), routes.handleRedeploy)
	group.PUT(projectcontract.ProjectLifecycleConfigurationRoute, httpx.RequirePermission(ctx.I18n, authService, authorizer, projectcontract.ProjectLifecyclePermission.String(), publisher), routes.handleLifecycleConfiguration)
	group.POST(projectcontract.ProjectBatchActionsRoute, httpx.RequirePermission(ctx.I18n, authService, authorizer, "", publisher), routes.handleBatchActions)
	group.POST(projectcontract.ProjectUnregisterRoute, httpx.RequirePermission(ctx.I18n, authService, authorizer, projectcontract.ProjectDestroyPermission.String(), publisher), routes.handleUnregister)
	group.POST(projectcontract.ProjectDestroyRoute, httpx.RequirePermission(ctx.I18n, authService, authorizer, projectcontract.ProjectDestroyPermission.String(), publisher), routes.handleDestroy)
	return nil
}

//nolint:dupl // Project list and runtime-candidate handlers intentionally share the generated-bind/query/write skeleton.
func (r routeRuntime) handleList(ginCtx *gin.Context) {
	params, ok := bindListParams(ginCtx, r.ctx)
	if !ok {
		return
	}
	projectGeneratedHandler{}.GetProjects(params)
	result, err := r.service.List(ginCtx.Request.Context(), ListQuery{
		Limit:           intPtrValue(params.Limit),
		Offset:          intPtrValue(params.Offset),
		Keyword:         stringPtrValue(params.Keyword),
		Sort:            projectListSortParamValue(params.Sort),
		ApplicationType: stringPtrValue(params.ApplicationType),
		RuntimeTargetID: params.RuntimeTargetId,
		Provider:        stringPtrValue(params.Provider),
		SourceKind:      stringPtrValue(params.SourceKind),
		RuntimeStatus:   stringPtrValue(params.RuntimeStatus),
		DriftStatus:     stringPtrValue(params.DriftStatus),
	})
	if err != nil {
		r.writeRouteError(ginCtx, err)
		return
	}
	httpx.WriteSuccess(ginCtx, http.StatusOK, toProjectListResponse(result))
}

func (r routeRuntime) handleComposeRuntimeTargets(ginCtx *gin.Context) {
	targets, err := r.service.ComposeRuntimeTargets(ginCtx.Request.Context())
	if err != nil {
		r.writeRouteError(ginCtx, err)
		return
	}
	items := make([]generated.ProjectComposeRuntimeTarget, 0, len(targets))
	for _, target := range targets {
		readiness := generated.ProjectComposeRuntimeTargetReadinessRuntimeUnavailable
		if target.Available {
			readiness = generated.ProjectComposeRuntimeTargetReadinessReady
		}
		items = append(items, generated.ProjectComposeRuntimeTarget{RuntimeTargetId: target.ID, DisplayName: target.DisplayName, Provider: target.Provider, Availability: target.Available, Readiness: readiness, Capabilities: target.Capabilities})
	}
	httpx.WriteSuccess(ginCtx, http.StatusOK, generated.ProjectComposeRuntimeTargetCatalogResponse{DeploymentType: generated.ProjectComposeRuntimeTargetCatalogResponseDeploymentTypeCompose, Items: items})
}

func (r routeRuntime) handleSavedViewList(ginCtx *gin.Context) {
	ownerID, ok := currentUserID(ginCtx)
	if !ok {
		r.writeInvalidArgumentError(ginCtx)
		return
	}
	projectGeneratedHandler{}.GetProjectSavedViews(generated.GetProjectSavedViewsParams{})
	items, err := r.service.listSavedViews(ginCtx.Request.Context(), ownerID)
	if err != nil {
		r.writeRouteError(ginCtx, err)
		return
	}
	mapped := make([]generated.ProjectSavedView, 0, len(items))
	for _, item := range items {
		view, mapErr := toGeneratedProjectSavedView(item)
		if mapErr != nil {
			r.writeSavedViewError(ginCtx, mapErr)
			return
		}
		mapped = append(mapped, view)
	}
	httpx.WriteSuccess(ginCtx, http.StatusOK, generated.ProjectSavedViewListResponse{Items: mapped})
}

func (r routeRuntime) handleSavedViewCreate(ginCtx *gin.Context) {
	ownerID, ok := currentUserID(ginCtx)
	if !ok {
		r.writeInvalidArgumentError(ginCtx)
		return
	}
	var body generated.PostProjectSavedViewJSONRequestBody
	if !bindJSON(ginCtx, r.ctx, &body) {
		return
	}
	request, err := projectSavedViewRequestFromGenerated(body)
	if err != nil {
		r.writeSavedViewError(ginCtx, err)
		return
	}
	projectGeneratedHandler{}.PostProjectSavedView(generated.PostProjectSavedViewParams{}, body)
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
	var body generated.PutProjectSavedViewJSONRequestBody
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
	projectGeneratedHandler{}.PutProjectSavedView(generatedID, generated.PutProjectSavedViewParams{}, body)
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
	projectGeneratedHandler{}.DeleteProjectSavedView(generatedID, generated.DeleteProjectSavedViewParams{})
	if err := r.service.deleteSavedView(ginCtx.Request.Context(), ownerID, id); err != nil {
		r.writeSavedViewError(ginCtx, err)
		return
	}
	ginCtx.Status(http.StatusNoContent)
}

func (r routeRuntime) handleImportValidate(ginCtx *gin.Context) {
	var request generated.PostProjectImportValidateJSONRequestBody
	if !bindJSON(ginCtx, r.ctx, &request) {
		return
	}
	projectGeneratedHandler{}.PostProjectImportValidate(bindImportValidateParams(ginCtx), request)
	result, err := r.service.ValidateImport(ginCtx.Request.Context(), toImportRequest(ginCtx, request))
	if err != nil {
		r.writeRouteError(ginCtx, err)
		return
	}
	httpx.WriteSuccess(ginCtx, http.StatusOK, toImportValidateResponse(result))
}

//nolint:dupl // Project list and runtime-candidate handlers intentionally share the generated-bind/query/write skeleton.
func (r routeRuntime) handleImportRuntimeCandidates(ginCtx *gin.Context) {
	params, ok := bindGetProjectImportRuntimeCandidatesParams(ginCtx, r.ctx)
	if !ok {
		return
	}
	projectGeneratedHandler{}.GetProjectImportRuntimeCandidates(params)
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
	var request generated.PostProjectImportRuntimeInspectJSONRequestBody
	if !bindJSON(ginCtx, r.ctx, &request) {
		return
	}
	projectGeneratedHandler{}.PostProjectImportRuntimeInspect(bindPostProjectImportRuntimeInspectParams(ginCtx), request)
	result, err := r.service.InspectRuntimeCandidate(ginCtx.Request.Context(), RuntimeImportInspectRequest{
		CandidateKey:                 request.CandidateKey,
		DisplayName:                  request.DisplayName,
		CanonicalProjectNameOverride: request.CanonicalProjectNameOverride,
	})
	if err != nil {
		r.writeRouteError(ginCtx, err)
		return
	}
	httpx.WriteSuccess(ginCtx, http.StatusOK, toRuntimeImportInspectResponse(result))
}

func (r routeRuntime) handleImport(ginCtx *gin.Context) {
	var request generated.PostProjectImportJSONRequestBody
	if !bindJSON(ginCtx, r.ctx, &request) {
		return
	}
	projectGeneratedHandler{}.PostProjectImport(bindPostProjectImportParams(ginCtx), request)
	lifecycleConfig, err := lifecycleStandardConfigFromGenerated(request.LifecycleConfiguration)
	if err != nil {
		r.writeRouteError(ginCtx, err)
		return
	}
	result, err := r.service.ImportByInspection(ginCtx.Request.Context(), ImportExecuteRequest{
		InspectionID:                 request.InspectionId,
		DisplayName:                  request.DisplayName,
		CanonicalProjectNameOverride: request.CanonicalProjectNameOverride,
		LifecycleConfiguration:       &lifecycleConfig,
		ActorID:                      currentUserIDPointer(ginCtx),
	})
	if err != nil {
		r.writeRouteError(ginCtx, err)
		return
	}
	httpx.WriteSuccess(ginCtx, http.StatusOK, result)
}

func (r routeRuntime) handleImportInspect(ginCtx *gin.Context) {
	var request generated.PostProjectImportInspectJSONRequestBody
	if !bindJSON(ginCtx, r.ctx, &request) {
		return
	}
	projectGeneratedHandler{}.PostProjectImportInspect(bindPostProjectImportInspectParams(ginCtx), request)
	result, err := r.service.InspectImportDirectory(ginCtx.Request.Context(), ImportInspectRequest{
		DirectoryRef: ImportDirectoryReference{
			Provider: request.DirectoryRef.Provider,
			RootID:   request.DirectoryRef.RootId,
			Path:     request.DirectoryRef.Path,
		},
		DisplayName:                  request.DisplayName,
		CanonicalProjectNameOverride: request.CanonicalProjectNameOverride,
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
	projectGeneratedHandler{}.GetProjectCreationMethods(bindGetProjectCreationMethodsParams(ginCtx))
	result, err := r.service.CreationMethodCatalog(ginCtx.Request.Context())
	if err != nil {
		r.writeRouteError(ginCtx, err)
		return
	}
	httpx.WriteSuccess(ginCtx, http.StatusOK, toCreationMethodCatalogResponse(result))
}

func (r routeRuntime) handleDiscoveryCandidates(ginCtx *gin.Context) {
	projectGeneratedHandler{}.GetProjectDiscoveryCandidates(bindGetProjectDiscoveryCandidatesParams(ginCtx))
	result, err := r.service.DiscoveryCandidates(ginCtx.Request.Context())
	if err != nil {
		r.writeRouteError(ginCtx, err)
		return
	}
	httpx.WriteSuccess(ginCtx, http.StatusOK, toDiscoveryCandidatesResponse(result))
}

func (r routeRuntime) handleManagedRoot(ginCtx *gin.Context) {
	projectGeneratedHandler{}.GetProjectManagedRoot(bindGetProjectManagedRootParams(ginCtx))
	result, err := r.service.ManagedRoot(ginCtx.Request.Context())
	if err != nil {
		r.writeRouteError(ginCtx, err)
		return
	}
	httpx.WriteSuccess(ginCtx, http.StatusOK, toManagedRootResponse(result))
}

func (r routeRuntime) handleCreateValidate(ginCtx *gin.Context) {
	var request generated.PostProjectCreateValidateJSONRequestBody
	if !bindJSON(ginCtx, r.ctx, &request) {
		return
	}
	projectGeneratedHandler{}.PostProjectCreateValidate(bindPostProjectCreateValidateParams(ginCtx), request)
	managedRequest, err := toManagedCreateRequest(request)
	if err != nil {
		r.writeRouteError(ginCtx, err)
		return
	}
	result, err := r.service.ValidateManagedCreate(ginCtx.Request.Context(), managedRequest)
	if err != nil {
		r.writeRouteError(ginCtx, err)
		return
	}
	httpx.WriteSuccess(ginCtx, http.StatusOK, toManagedCreateValidateResponse(result))
}

func (r routeRuntime) handleCreate(ginCtx *gin.Context) {
	var request generated.PostProjectCreateJSONRequestBody
	if !bindJSON(ginCtx, r.ctx, &request) {
		return
	}
	projectGeneratedHandler{}.PostProjectCreate(bindPostProjectCreateParams(ginCtx), request)
	managedRequest, err := toManagedCreateExecuteRequest(request)
	if err != nil {
		r.writeRouteError(ginCtx, err)
		return
	}
	result, err := r.service.CreateManagedProject(ginCtx.Request.Context(), managedRequest, currentUserIDPointer(ginCtx))
	if err != nil {
		r.writeRouteError(ginCtx, err)
		return
	}
	httpx.WriteSuccess(ginCtx, http.StatusCreated, toManagedCreateResponse(result))
}

type templateProjectCreateHTTP struct {
	DisplayName            string                                          `json:"display_name"`
	RuntimeTargetID        uint64                                          `json:"runtime_target_id"`
	ApplicationName        *string                                         `json:"application_name"`
	TemplateKey            string                                          `json:"template_key"`
	TemplateVersion        string                                          `json:"template_version"`
	TemplateInstanceName   string                                          `json:"template_instance_name"`
	LifecycleConfiguration *generated.ProjectLifecycleConfigurationRequest `json:"lifecycle_configuration"`
}

func (r routeRuntime) handleTemplateCreateValidate(ginCtx *gin.Context) {
	var request templateProjectCreateHTTP
	if !bindJSON(ginCtx, r.ctx, &request) {
		return
	}
	templateRequest, err := toTemplateProjectCreateRequest(request)
	if err != nil {
		r.writeRouteError(ginCtx, err)
		return
	}
	result, err := r.service.ValidateTemplateProject(ginCtx.Request.Context(), templateRequest)
	if err != nil {
		r.writeRouteError(ginCtx, err)
		return
	}
	httpx.WriteSuccess(ginCtx, http.StatusOK, toManagedCreateValidateResponse(result))
}

func (r routeRuntime) handleTemplateCreate(ginCtx *gin.Context) {
	var request templateProjectCreateHTTP
	if !bindJSON(ginCtx, r.ctx, &request) {
		return
	}
	templateRequest, err := toTemplateProjectCreateRequest(request)
	if err != nil {
		r.writeRouteError(ginCtx, err)
		return
	}
	result, err := r.service.CreateTemplateProject(ginCtx.Request.Context(), templateRequest, currentUserIDPointer(ginCtx))
	if err != nil {
		r.writeRouteError(ginCtx, err)
		return
	}
	httpx.WriteSuccess(ginCtx, http.StatusCreated, toManagedCreateResponse(result))
}

func (r routeRuntime) handleWorkspaceDefaults(ginCtx *gin.Context) {
	result, err := r.service.WorkspaceDefaults(ginCtx.Request.Context())
	if err != nil {
		r.writeRouteError(ginCtx, err)
		return
	}
	templates := make([]gin.H, 0, len(result.Templates))
	for _, template := range result.Templates {
		templates = append(templates, gin.H{"key": template.Key, "display_name": template.DisplayName})
	}
	entries := make([]gin.H, 0, len(result.WorkspaceEntries))
	for _, entry := range result.WorkspaceEntries {
		item := gin.H{"path": entry.Path, "node_type": entry.NodeType}
		if entry.Content != nil {
			item["content"] = *entry.Content
		}
		entries = append(entries, item)
	}
	httpx.WriteSuccess(ginCtx, http.StatusOK, gin.H{"templates": templates, "default_template_key": result.DefaultTemplateKey, "workspace_entries": entries, "compose_file_path": result.ComposeFilePath})
}

// toTemplateProjectCreateRequest converts an HTTP template creation request into a domain request.
// toTemplateProjectCreateRequest 将模板项目创建请求转换为领域请求。
// 当生命周期配置无法转换为标准配置时返回错误。
func toTemplateProjectCreateRequest(request templateProjectCreateHTTP) (TemplateProjectCreateRequest, error) {
	result := TemplateProjectCreateRequest{DisplayName: request.DisplayName, RuntimeTargetID: request.RuntimeTargetID, ApplicationName: request.ApplicationName, TemplateKey: request.TemplateKey, TemplateVersion: request.TemplateVersion, TemplateInstanceName: request.TemplateInstanceName}
	if request.LifecycleConfiguration != nil {
		config, err := lifecycleStandardConfigFromGenerated(*request.LifecycleConfiguration)
		if err != nil {
			return TemplateProjectCreateRequest{}, err
		}
		result.LifecycleConfig = &config
	}
	return result, nil
}

func (r routeRuntime) handleDetail(ginCtx *gin.Context) {
	projectID, generatedID, ok := r.bindProjectID(ginCtx)
	if !ok {
		return
	}
	projectGeneratedHandler{}.GetProject(generatedID, bindGetProjectParams(ginCtx))
	result, err := r.service.Get(ginCtx.Request.Context(), projectID)
	if err != nil {
		r.writeRouteError(ginCtx, err)
		return
	}
	httpx.WriteSuccess(ginCtx, http.StatusOK, result)
}

func (r routeRuntime) handleOverview(ginCtx *gin.Context) {
	projectID, generatedID, ok := r.bindProjectID(ginCtx)
	if !ok {
		return
	}
	projectGeneratedHandler{}.GetProjectOverview(generatedID, bindGetProjectOverviewParams(ginCtx))
	result, err := r.service.Overview(ginCtx.Request.Context(), projectID)
	if err != nil {
		r.writeRouteError(ginCtx, err)
		return
	}
	httpx.WriteSuccess(ginCtx, http.StatusOK, result)
}

func (r routeRuntime) handleServices(ginCtx *gin.Context) {
	projectID, generatedID, ok := r.bindProjectID(ginCtx)
	if !ok {
		return
	}
	projectGeneratedHandler{}.GetProjectServices(generatedID, bindGetProjectServicesParams(ginCtx))
	result, err := r.service.Services(ginCtx.Request.Context(), projectID)
	if err != nil {
		r.writeRouteError(ginCtx, err)
		return
	}
	httpx.WriteSuccess(ginCtx, http.StatusOK, result)
}

func (r routeRuntime) handleLogs(ginCtx *gin.Context) {
	projectID, generatedID, ok := r.bindProjectID(ginCtx)
	if !ok {
		return
	}
	params, ok := bindGetProjectLogsParams(ginCtx, r.ctx)
	if !ok {
		return
	}
	projectGeneratedHandler{}.GetProjectLogs(generatedID, params)
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
	projectID, generatedID, ok := r.bindProjectID(ginCtx)
	if !ok {
		return
	}
	projectGeneratedHandler{}.GetProjectConfiguration(generatedID, bindGetProjectConfigurationParams(ginCtx))
	result, err := r.service.ConfigurationMetadata(ginCtx.Request.Context(), projectID)
	if err != nil {
		r.writeRouteError(ginCtx, err)
		return
	}
	httpx.WriteSuccess(ginCtx, http.StatusOK, toConfigurationMetadataResponse(result))
}

func (r routeRuntime) handleConfigurationPreview(ginCtx *gin.Context) {
	projectID, generatedID, ok := r.bindProjectID(ginCtx)
	if !ok {
		return
	}
	projectGeneratedHandler{}.GetProjectConfigurationPreview(generatedID, bindGetProjectConfigurationPreviewParams(ginCtx))
	result, err := r.service.ConfigurationPreview(ginCtx.Request.Context(), projectID)
	if err != nil {
		r.writeRouteError(ginCtx, err)
		return
	}
	httpx.WriteSuccess(ginCtx, http.StatusOK, toConfigurationPreviewResponse(result))
}

func (r routeRuntime) handleProjectWorkspaceFiles(ginCtx *gin.Context) {
	projectID, generatedID, ok := r.bindProjectID(ginCtx)
	if !ok {
		return
	}
	query, ok := bindProjectWorkspaceFilesQuery(ginCtx, r.ctx)
	if !ok {
		return
	}
	projectGeneratedHandler{}.GetProjectFiles(generatedID, bindGetProjectFilesParams(ginCtx, query))
	result, err := r.service.browseProjectFiles(ginCtx.Request.Context(), projectID, query)
	if err != nil {
		r.writeRouteError(ginCtx, err)
		return
	}
	httpx.WriteSuccess(ginCtx, http.StatusOK, toProjectWorkspaceFilesResponse(result))
}

func (r routeRuntime) handleProjectWorkspaceFileContent(ginCtx *gin.Context) {
	projectID, generatedID, ok := r.bindProjectID(ginCtx)
	if !ok {
		return
	}
	path, ok := bindProjectWorkspaceFilePath(ginCtx, r.ctx)
	if !ok {
		return
	}
	projectGeneratedHandler{}.GetProjectFileContent(generatedID, bindGetProjectFileContentParams(ginCtx, path))
	result, err := r.service.projectFileContent(ginCtx.Request.Context(), projectID, path)
	if err != nil {
		r.writeRouteError(ginCtx, err)
		return
	}
	httpx.WriteSuccess(ginCtx, http.StatusOK, toProjectWorkspaceFileContentResponse(result))
}

func (r routeRuntime) handleSaveProjectWorkspaceFileContent(ginCtx *gin.Context) {
	projectID, generatedID, ok := r.bindProjectID(ginCtx)
	if !ok {
		return
	}
	path, ok := bindProjectWorkspaceFilePath(ginCtx, r.ctx)
	if !ok {
		return
	}
	var request generated.PutProjectFileContentJSONRequestBody
	if !bindJSON(ginCtx, r.ctx, &request) {
		return
	}
	projectGeneratedHandler{}.PutProjectFileContent(generatedID, bindPutProjectFileContentParams(ginCtx, path), request)
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
	projectID, generatedID, ok := r.bindProjectID(ginCtx)
	if !ok {
		return
	}
	path, ok := bindProjectWorkspaceFilePath(ginCtx, r.ctx)
	if !ok {
		return
	}
	var request generated.PutProjectFileAnnotationJSONRequestBody
	if !bindJSON(ginCtx, r.ctx, &request) {
		return
	}
	projectGeneratedHandler{}.PutProjectFileAnnotation(generatedID, bindPutProjectFileAnnotationParams(ginCtx, path), request)
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
	httpx.WriteSuccess(ginCtx, http.StatusOK, generated.ProjectFileTreeItem{
		Name:            result.Name,
		RelativePath:    result.RelativePath,
		NodeType:        generated.ProjectFileTreeNodeType(result.NodeType),
		FileKind:        generated.ProjectWorkspaceFileKind(result.FileKind),
		Editable:        result.Editable,
		LanguageHint:    result.LanguageHint,
		SizeBytes:       result.SizeBytes,
		HiddenByDefault: result.HiddenByDefault,
		HasChildren:     result.HasChildren,
		ProjectNote:     optionalString(result.ProjectNote),
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
	projectID, _, ok := r.bindProjectID(ginCtx)
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
	projectID, _, ok := r.bindProjectID(ginCtx)
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

func (r routeRuntime) handleDeleteProjectWorkspaceEntry(ginCtx *gin.Context) {
	projectID, _, ok := r.bindProjectID(ginCtx)
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
	projectID, generatedID, ok := r.bindProjectID(ginCtx)
	if !ok {
		return
	}
	projectGeneratedHandler{}.PostProjectRefresh(generatedID, bindPostProjectRefreshParams(ginCtx))
	result, err := r.service.Refresh(ginCtx.Request.Context(), projectID, currentUserIDPointer(ginCtx))
	if err != nil {
		r.writeRouteError(ginCtx, err)
		return
	}
	httpx.WriteSuccess(ginCtx, http.StatusOK, toActionResponse(result))
}

func (r routeRuntime) handleDeploy(ginCtx *gin.Context) {
	projectID, generatedID, ok := r.bindProjectID(ginCtx)
	if !ok {
		return
	}
	projectGeneratedHandler{}.PostProjectDeploy(generatedID, bindPostProjectDeployParams(ginCtx))
	result, err := r.service.DeployConfiguration(
		ginCtx.Request.Context(),
		projectID,
		currentUserIDPointer(ginCtx),
	)
	if err != nil {
		r.writeRouteError(ginCtx, err)
		return
	}
	httpx.WriteSuccess(ginCtx, http.StatusOK, toDeployResponse(result))
}

func (r routeRuntime) handleUp(ginCtx *gin.Context) {
	projectID, generatedID, ok := r.bindProjectID(ginCtx)
	if !ok {
		return
	}
	projectGeneratedHandler{}.PostProjectUp(generatedID, bindPostProjectUpParams(ginCtx))
	result, err := r.service.Up(ginCtx.Request.Context(), projectID, currentUserIDPointer(ginCtx))
	if err != nil {
		r.writeRouteErrorWithAction(ginCtx, err, result)
		return
	}
	httpx.WriteSuccess(ginCtx, http.StatusAccepted, toTaskReceiptResponse(result))
}

func (r routeRuntime) handleStop(ginCtx *gin.Context) {
	projectID, generatedID, ok := r.bindProjectID(ginCtx)
	if !ok {
		return
	}
	projectGeneratedHandler{}.PostProjectStop(generatedID, bindPostProjectStopParams(ginCtx))
	result, err := r.service.Stop(ginCtx.Request.Context(), projectID, currentUserIDPointer(ginCtx))
	if err != nil {
		r.writeRouteErrorWithAction(ginCtx, err, result)
		return
	}
	httpx.WriteSuccess(ginCtx, http.StatusAccepted, toTaskReceiptResponse(result))
}

func (r routeRuntime) handleRestart(ginCtx *gin.Context) {
	projectID, generatedID, ok := r.bindProjectID(ginCtx)
	if !ok {
		return
	}
	projectGeneratedHandler{}.PostProjectRestart(generatedID, bindPostProjectRestartParams(ginCtx))
	result, err := r.service.Restart(ginCtx.Request.Context(), projectID, currentUserIDPointer(ginCtx))
	if err != nil {
		r.writeRouteErrorWithAction(ginCtx, err, result)
		return
	}
	httpx.WriteSuccess(ginCtx, http.StatusAccepted, toTaskReceiptResponse(result))
}

func (r routeRuntime) handleRedeploy(ginCtx *gin.Context) {
	projectID, generatedID, ok := r.bindProjectID(ginCtx)
	if !ok {
		return
	}
	projectGeneratedHandler{}.PostProjectRedeploy(generatedID, bindPostProjectRedeployParams(ginCtx))
	result, err := r.service.Redeploy(ginCtx.Request.Context(), projectID, currentUserIDPointer(ginCtx))
	if err != nil {
		r.writeRouteErrorWithAction(ginCtx, err, result)
		return
	}
	httpx.WriteSuccess(ginCtx, http.StatusAccepted, toTaskReceiptResponse(result))
}

func (r routeRuntime) handleLifecycleConfiguration(ginCtx *gin.Context) {
	projectID, generatedID, ok := r.bindProjectID(ginCtx)
	if !ok {
		return
	}
	var request generated.ProjectLifecycleConfigurationRequest
	if !bindJSON(ginCtx, r.ctx, &request) {
		return
	}
	projectGeneratedHandler{}.PutProjectLifecycleConfiguration(
		generatedID,
		bindPutProjectLifecycleConfigurationParams(ginCtx),
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
	var request generated.ProjectBatchActionRequest
	if !bindJSON(ginCtx, r.ctx, &request) {
		return
	}
	if !r.authorizeBatchAction(ginCtx, request.Action) {
		return
	}
	projectGeneratedHandler{}.PostProjectBatchActions(bindPostProjectBatchActionsParams(ginCtx), request)
	projectIDs, err := r.resolveBatchProjectIDs(ginCtx, request.ApplicationIds)
	if err != nil {
		r.writeRouteError(ginCtx, err)
		return
	}
	result, err := r.service.BatchAction(ginCtx.Request.Context(), BatchActionRequest{
		Action:                      request.Action,
		ProjectIDs:                  projectIDs,
		RemoveNamedVolumes:          boolValue(request.RemoveNamedVolumes),
		AutoUnregister:              boolValue(request.AutoUnregister),
		ImagePrune:                  boolValue(request.ImagePrune),
		DeleteWorkingDirectory:      boolValue(request.DeleteWorkingDirectory),
		ConfirmCanonicalProjectName: request.ConfirmCanonicalProjectName,
		ActorID:                     currentUserIDPointer(ginCtx),
	})
	if err != nil {
		r.writeRouteError(ginCtx, err)
		return
	}
	httpx.WriteSuccess(ginCtx, http.StatusOK, toBatchActionResponse(result))
}

func (r routeRuntime) resolveBatchProjectIDs(ginCtx *gin.Context, applicationIDs []string) ([]uint64, error) {
	repository, err := r.service.repositoryOrErr()
	if err != nil {
		return nil, err
	}
	lookup, ok := repository.(projectstore.ApplicationIDBatchLookupRepository)
	if !ok {
		return nil, errProjectServiceUnavailable
	}
	resolved, err := lookup.GetIDsByApplicationIDs(ginCtx.Request.Context(), applicationIDs)
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

func (r routeRuntime) authorizeBatchAction(ginCtx *gin.Context, action generated.ProjectBatchActionRequestAction) bool {
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
		httpx.AbortLocalizedError(ginCtx, r.ctx.I18n, http.StatusInternalServerError, messagecontract.CommonInternalError.String(), nil)
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
			httpx.AbortLocalizedError(ginCtx, r.ctx.I18n, http.StatusInternalServerError, messagecontract.CommonInternalError.String(), nil)
		}
		return false
	}
	return true
}

// batchActionPermission returns the required permission for a batch action and whether the action is supported.
func batchActionPermission(action generated.ProjectBatchActionRequestAction) (string, bool) {
	switch action {
	case generated.ProjectBatchActionRequestActionStart,
		generated.ProjectBatchActionRequestActionStop,
		generated.ProjectBatchActionRequestActionRestart,
		generated.ProjectBatchActionRequestActionRedeploy:
		return projectcontract.ProjectLifecyclePermission.String(), true
	case generated.ProjectBatchActionRequestActionUnregister,
		generated.ProjectBatchActionRequestActionDestroy:
		return projectcontract.ProjectDestroyPermission.String(), true
	default:
		return "", false
	}
}

func (r routeRuntime) handleUnregister(ginCtx *gin.Context) {
	projectID, generatedID, ok := r.bindProjectID(ginCtx)
	if !ok {
		return
	}
	projectGeneratedHandler{}.PostProjectUnregister(generatedID, bindPostProjectUnregisterParams(ginCtx))
	result, err := r.service.Unregister(ginCtx.Request.Context(), projectID, currentUserIDPointer(ginCtx))
	if err != nil {
		r.writeRouteErrorWithAction(ginCtx, err, result)
		return
	}
	httpx.WriteSuccess(ginCtx, http.StatusOK, toActionResponse(result))
}

func (r routeRuntime) handleDestroy(ginCtx *gin.Context) {
	projectID, generatedID, ok := r.bindProjectID(ginCtx)
	if !ok {
		return
	}
	var request generated.PostProjectDestroyJSONRequestBody
	if !bindJSON(ginCtx, r.ctx, &request) {
		return
	}
	projectGeneratedHandler{}.PostProjectDestroy(generatedID, bindPostProjectDestroyParams(ginCtx), request)
	result, err := r.service.Destroy(ginCtx.Request.Context(), projectID, DestroyRequest{
		RemoveNamedVolumes:          request.RemoveNamedVolumes,
		AutoUnregister:              boolValue(request.AutoUnregister),
		ImagePrune:                  boolValue(request.ImagePrune),
		DeleteWorkingDirectory:      request.DeleteWorkspace,
		ConfirmCanonicalProjectName: request.ConfirmApplicationId,
		ActorID:                     currentUserIDPointer(ginCtx),
	})
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
		httpx.WriteLocalizedErrorCode(ginCtx, r.ctx.I18n, http.StatusInternalServerError, messagecontract.CommonInternalError.String(), messagecontract.CommonInternalError.String(), nil)
	}
	ginCtx.Abort()
}

func (r routeRuntime) writeHandledRouteError(ginCtx *gin.Context, err error, action ActionResult) bool {
	if r.writeApplicationNameError(ginCtx, err) {
		return true
	}
	switch {
	case errors.Is(err, errProjectFileNotFound):
		r.writeFileNotFoundError(ginCtx)
	case errors.Is(err, errProjectInvalidArgument), errors.Is(err, errProjectImportValidation):
		r.writeInvalidArgumentError(ginCtx)
	case errors.Is(err, errProjectInvalidCanonicalName):
		r.writeLocalizedActionError(ginCtx, http.StatusBadRequest, projectcontract.ProjectInvalidCanonicalProjectName.String(), map[string]any{
			"code": projectcontract.ProjectInvalidCanonicalProjectName.String(),
		})
	case r.writeProjectConflictError(ginCtx, err):
	case errors.Is(err, errProjectDestroyBlocked):
		r.writeLocalizedActionError(ginCtx, http.StatusConflict, projectcontract.ProjectConflict.String(), map[string]any{
			"code":         projectcontract.ProjectConflict.String(),
			"actionResult": toActionResponse(action),
		})
	case errors.Is(err, errProjectUnsupportedLifecycle), errors.Is(err, errProjectManagedFlow), errors.Is(err, errProjectRuntimeUnavailable):
		messageKey := projectcontract.ProjectUnsupportedLifecycle
		if errors.Is(err, errProjectRuntimeUnavailable) {
			messageKey = projectcontract.ProjectRuntimeUnavailable
		}
		r.writeLocalizedActionError(ginCtx, http.StatusConflict, messageKey.String(), map[string]any{
			"code":         mapLifecycleErrorCode(err),
			"actionResult": toActionResponse(action),
		})
	default:
		return false
	}
	return true
}

func (r routeRuntime) writeApplicationNameError(ginCtx *gin.Context, err error) bool {
	switch {
	case errors.Is(err, errProjectApplicationNameRequired):
		r.writeLocalizedProjectError(ginCtx, http.StatusBadRequest, projectcontract.ProjectApplicationNameRequired.String())
	case errors.Is(err, errProjectInvalidApplicationName):
		r.writeLocalizedProjectError(ginCtx, http.StatusBadRequest, projectcontract.ProjectInvalidApplicationName.String())
	case errors.Is(err, errProjectApplicationNameOccupied):
		r.writeLocalizedProjectError(ginCtx, http.StatusConflict, projectcontract.ProjectApplicationNameOccupied.String())
	default:
		return false
	}
	return true
}

func (r routeRuntime) writeProjectConflictError(ginCtx *gin.Context, err error) bool {
	switch {
	case errors.Is(err, errProjectNotFound):
		r.writeLocalizedProjectError(ginCtx, http.StatusNotFound, projectcontract.ProjectNotFound.String())
	case errors.Is(err, errProjectComposeNameOccupied):
		r.writeLocalizedProjectError(ginCtx, http.StatusConflict, projectcontract.ProjectComposeProjectNameOccupied.String())
	case errors.Is(err, errProjectConflict):
		r.writeLocalizedProjectError(ginCtx, http.StatusConflict, projectcontract.ProjectConflict.String())
	case errors.Is(err, errProjectDirectoryForbidden):
		r.writeLocalizedProjectError(ginCtx, http.StatusForbidden, projectcontract.ProjectDirectoryBrowseForbidden.String())
	case errors.Is(err, errProjectInspectionExpired):
		r.writeLocalizedProjectError(ginCtx, http.StatusConflict, projectcontract.ProjectInspectionExpired.String())
	case errors.Is(err, errProjectInspectionStale):
		r.writeLocalizedProjectError(ginCtx, http.StatusConflict, projectcontract.ProjectInspectionStale.String())
	default:
		return false
	}
	return true
}

func (r routeRuntime) writeInvalidArgumentError(ginCtx *gin.Context) {
	r.writeLocalizedActionError(ginCtx, http.StatusBadRequest, projectcontract.ProjectInvalidArgument.String(), map[string]any{"code": projectcontract.ProjectInvalidArgument.String()})
}

func (r routeRuntime) writeFileNotFoundError(ginCtx *gin.Context) {
	r.writeLocalizedActionError(ginCtx, http.StatusNotFound, projectcontract.ProjectInvalidFileID.String(), map[string]any{"code": projectcontract.ProjectInvalidFileID.String()})
}

func (r routeRuntime) writeLocalizedProjectError(ginCtx *gin.Context, status int, code string) {
	r.writeLocalizedActionError(ginCtx, status, code, map[string]any{"code": code})
}

func (r routeRuntime) writeLocalizedActionError(ginCtx *gin.Context, status int, code string, details map[string]any) {
	httpx.WriteLocalizedErrorCode(ginCtx, r.ctx.I18n, status, code, projectErrorMessageKey(code), details)
}

type projectGeneratedHandler struct{}

func (projectGeneratedHandler) GetProjects(generated.GetProjectsParams)                   {}
func (projectGeneratedHandler) GetProjectSavedViews(generated.GetProjectSavedViewsParams) {}
func (projectGeneratedHandler) PostProjectSavedView(generated.PostProjectSavedViewParams, generated.PostProjectSavedViewJSONRequestBody) {
}
func (projectGeneratedHandler) PutProjectSavedView(int64, generated.PutProjectSavedViewParams, generated.PutProjectSavedViewJSONRequestBody) {
}
func (projectGeneratedHandler) DeleteProjectSavedView(int64, generated.DeleteProjectSavedViewParams) {
}
func (projectGeneratedHandler) GetProjectCreationMethods(generated.GetProjectCreationMethodsParams) {}
func (projectGeneratedHandler) GetProjectDiscoveryCandidates(generated.GetProjectDiscoveryCandidatesParams) {
}
func (projectGeneratedHandler) GetProjectImportRuntimeCandidates(generated.GetProjectImportRuntimeCandidatesParams) {
}
func (projectGeneratedHandler) PostProjectImportValidate(generated.PostProjectImportValidateParams, generated.PostProjectImportValidateJSONRequestBody) {
}
func (projectGeneratedHandler) PostProjectImportRuntimeInspect(generated.PostProjectImportRuntimeInspectParams, generated.PostProjectImportRuntimeInspectJSONRequestBody) {
}
func (projectGeneratedHandler) PostProjectImport(generated.PostProjectImportParams, generated.PostProjectImportJSONRequestBody) {
}
func (projectGeneratedHandler) PostProjectImportInspect(generated.PostProjectImportInspectParams, generated.PostProjectImportInspectJSONRequestBody) {
}
func (projectGeneratedHandler) GetProjectManagedRoot(generated.GetProjectManagedRootParams) {}
func (projectGeneratedHandler) PostProjectCreateValidate(generated.PostProjectCreateValidateParams, generated.PostProjectCreateValidateJSONRequestBody) {
}
func (projectGeneratedHandler) PostProjectCreate(generated.PostProjectCreateParams, generated.PostProjectCreateJSONRequestBody) {
}
func (projectGeneratedHandler) GetProject(string, generated.GetProjectParams)                 {}
func (projectGeneratedHandler) GetProjectOverview(string, generated.GetProjectOverviewParams) {}
func (projectGeneratedHandler) GetProjectServices(string, generated.GetProjectServicesParams) {}
func (projectGeneratedHandler) GetProjectLogs(string, generated.GetProjectLogsParams)         {}
func (projectGeneratedHandler) GetProjectConfiguration(string, generated.GetProjectConfigurationParams) {
}
func (projectGeneratedHandler) GetProjectConfigurationPreview(string, generated.GetProjectConfigurationPreviewParams) {
}
func (projectGeneratedHandler) GetProjectFiles(string, generated.GetProjectFilesParams)             {}
func (projectGeneratedHandler) GetProjectFileContent(string, generated.GetProjectFileContentParams) {}
func (projectGeneratedHandler) PutProjectFileContent(string, generated.PutProjectFileContentParams, generated.PutProjectFileContentJSONRequestBody) {
}
func (projectGeneratedHandler) PutProjectFileAnnotation(string, generated.PutProjectFileAnnotationParams, generated.PutProjectFileAnnotationJSONRequestBody) {
}
func (projectGeneratedHandler) PostProjectRefresh(string, generated.PostProjectRefreshParams)   {}
func (projectGeneratedHandler) PostProjectDeploy(string, generated.PostProjectDeployParams)     {}
func (projectGeneratedHandler) PostProjectUp(string, generated.PostProjectUpParams)             {}
func (projectGeneratedHandler) PostProjectStop(string, generated.PostProjectStopParams)         {}
func (projectGeneratedHandler) PostProjectRestart(string, generated.PostProjectRestartParams)   {}
func (projectGeneratedHandler) PostProjectRedeploy(string, generated.PostProjectRedeployParams) {}
func (projectGeneratedHandler) PutProjectLifecycleConfiguration(string, generated.PutProjectLifecycleConfigurationParams, generated.ProjectLifecycleConfigurationRequest) {
}
func (projectGeneratedHandler) PostProjectBatchActions(generated.PostProjectBatchActionsParams, generated.ProjectBatchActionRequest) {
}
func (projectGeneratedHandler) PostProjectUnregister(string, generated.PostProjectUnregisterParams) {
}
func (projectGeneratedHandler) PostProjectDestroy(string, generated.PostProjectDestroyParams, generated.PostProjectDestroyJSONRequestBody) {
}

// bindListParams 绑定项目列表查询参数和公共请求头。
// 它解析 source_kind、drift_status、limit 和 offset，并在分页参数无效时中止请求。
//
// bindListParams 解析并校验项目列表查询参数。
// bindListParams 解析项目列表查询参数；参数无效时中止请求并返回 false，否则返回解析后的参数和 true。
func bindListParams(ginCtx *gin.Context, ctx *module.Context) (generated.GetProjectsParams, bool) {
	locale, requestID := commonHeaders(ginCtx)
	params := generated.GetProjectsParams{
		XGraftLocale: locale,
		XRequestId:   requestID,
	}
	filters, ok := bindListFilterParams(ginCtx, ctx)
	if !ok {
		return generated.GetProjectsParams{}, false
	}
	params.SourceKind = filters.SourceKind
	params.DriftStatus = filters.DriftStatus
	params.ApplicationType = filters.ApplicationType
	params.Provider = filters.Provider
	params.RuntimeStatus = filters.RuntimeStatus
	query := ginCtx.Request.URL.Query()
	if params.Sort, ok = bindProjectListSort(query); !ok {
		abortInvalidQuery(ginCtx, ctx)
		return generated.GetProjectsParams{}, false
	}
	if keyword := strings.TrimSpace(query.Get("keyword")); keyword != "" {
		params.Keyword = &keyword
	}
	if value := strings.TrimSpace(query.Get("runtime_target_id")); value != "" {
		parsed, err := strconv.ParseInt(value, 10, 64)
		if err != nil || parsed < 1 {
			abortInvalidQuery(ginCtx, ctx)
			return generated.GetProjectsParams{}, false
		}
		params.RuntimeTargetId = &parsed
	}
	if params.Limit, ok = optionalIntQuery[generated.ProjectListLimit](query.Get("limit"), minimumProjectListLimit, maxProjectListLimit); !ok {
		abortInvalidQuery(ginCtx, ctx)
		return generated.GetProjectsParams{}, false
	}
	if params.Offset, ok = optionalIntQuery[generated.ProjectListOffset](query.Get("offset"), 0, 0); !ok {
		abortInvalidQuery(ginCtx, ctx)
		return generated.GetProjectsParams{}, false
	}
	return params, true
}

// projectListSortValues returns the sort values provided through canonical or bracketed query keys.
func projectListSortValues(query url.Values) []string {
	values := append([]string(nil), query["sort"]...)
	values = append(values, query["sort[]"]...)
	return values
}

// 查询中未提供排序值时返回 nil；排序值无效或存在多个值时返回 false。
func bindProjectListSort(query url.Values) (*generated.ProjectListSort, bool) {
	rawSorts := projectListSortValues(query)
	if len(rawSorts) == 0 {
		return nil, true
	}
	if len(rawSorts) > 1 {
		return nil, false
	}
	value := generated.GetProjectsParamsSort(strings.TrimSpace(rawSorts[0]))
	if !value.Valid() {
		return nil, false
	}
	sorts := generated.ProjectListSort{string(value)}
	return &sorts, true
}

// projectListSortParamValue returns the first sort value, or an empty string when no sort value is provided.
func projectListSortParamValue(values *generated.ProjectListSort) string {
	if values == nil || len(*values) == 0 {
		return ""
	}
	return string((*values)[0])
}

// 返回包含有效筛选条件的参数；任一参数无效时中止请求并返回 false。
func bindListFilterParams(ginCtx *gin.Context, ctx *module.Context) (generated.GetProjectsParams, bool) {
	query := ginCtx.Request.URL.Query()
	params := generated.GetProjectsParams{}
	sourceKind, ok := optionalValidatedEnumQuery(query.Get("source_kind"), generated.ProjectSourceKind.Valid)
	if !ok {
		abortInvalidQuery(ginCtx, ctx)
		return generated.GetProjectsParams{}, false
	}
	driftStatus, ok := optionalValidatedEnumQuery(query.Get("drift_status"), generated.ProjectDriftStatus.Valid)
	if !ok {
		abortInvalidQuery(ginCtx, ctx)
		return generated.GetProjectsParams{}, false
	}
	params.SourceKind = sourceKind
	params.DriftStatus = driftStatus
	applicationType, ok := optionalValidatedEnumQuery(query.Get("application_type"), generated.GetProjectsParamsApplicationType.Valid)
	if !ok {
		abortInvalidQuery(ginCtx, ctx)
		return generated.GetProjectsParams{}, false
	}
	params.ApplicationType = applicationType
	provider, ok := optionalValidatedEnumQuery(query.Get("provider"), generated.GetProjectsParamsProvider.Valid)
	if !ok {
		abortInvalidQuery(ginCtx, ctx)
		return generated.GetProjectsParams{}, false
	}
	params.Provider = provider
	runtimeStatus, ok := optionalValidatedEnumQuery(query.Get("runtime_status"), generated.ProjectRuntimeStatus.Valid)
	if !ok {
		abortInvalidQuery(ginCtx, ctx)
		return generated.GetProjectsParams{}, false
	}
	params.RuntimeStatus = runtimeStatus
	return params, true
}

func bindGetProjectImportRuntimeCandidatesParams(
	ginCtx *gin.Context,
	ctx *module.Context,
) (generated.GetProjectImportRuntimeCandidatesParams, bool) {
	locale, requestID := commonHeaders(ginCtx)
	query := ginCtx.Request.URL.Query()
	params := generated.GetProjectImportRuntimeCandidatesParams{
		XGraftLocale: locale,
		XRequestId:   requestID,
	}
	if strings.TrimSpace(query.Get("keyword")) != "" {
		keyword := strings.TrimSpace(query.Get("keyword"))
		params.Keyword = &keyword
	}
	availability, ok := optionalValidatedEnumQuery(
		query.Get("availability"),
		generated.ProjectImportRuntimeCandidateAvailability.Valid,
	)
	if !ok {
		abortInvalidQuery(ginCtx, ctx)
		return generated.GetProjectImportRuntimeCandidatesParams{}, false
	}
	params.Availability = availability
	if params.Limit, ok = optionalIntQuery[generated.ProjectImportRuntimeCandidateListLimit](
		query.Get("limit"),
		minimumProjectListLimit,
		maxProjectListLimit,
	); !ok {
		abortInvalidQuery(ginCtx, ctx)
		return generated.GetProjectImportRuntimeCandidatesParams{}, false
	}
	if params.Offset, ok = optionalIntQuery[generated.ProjectImportRuntimeCandidateListOffset](query.Get("offset"), 0, 0); !ok {
		abortInvalidQuery(ginCtx, ctx)
		return generated.GetProjectImportRuntimeCandidatesParams{}, false
	}
	return params, true
}

func runtimeCandidateAvailabilityFromGenerated(
	value *generated.ProjectImportRuntimeCandidateAvailability,
) *RuntimeImportCandidateAvailability {
	if value == nil {
		return nil
	}
	availability := RuntimeImportCandidateAvailability(*value)
	return &availability
}

func bindPostProjectImportRuntimeInspectParams(ginCtx *gin.Context) generated.PostProjectImportRuntimeInspectParams {
	locale, requestID := commonHeaders(ginCtx)
	return generated.PostProjectImportRuntimeInspectParams{
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

// bindProjectID resolves the public Application ID in a project route to the
// private numeric key used by project-owned storage and tasks.
func (r routeRuntime) bindProjectID(ginCtx *gin.Context) (uint64, string, bool) {
	raw := strings.TrimSpace(ginCtx.Param("id"))
	if !isApplicationID(raw) {
		httpx.WriteLocalizedErrorCode(ginCtx, r.ctx.I18n, http.StatusBadRequest, projectcontract.ProjectInvalidID.String(), messagecontract.CommonInvalidArgument.String(), map[string]any{"field": "id", "code": projectcontract.ProjectInvalidID.String()})
		ginCtx.Abort()
		return 0, "", false
	}
	projectID, err := r.service.ResolveApplicationID(ginCtx.Request.Context(), raw)
	if err != nil {
		r.writeRouteError(ginCtx, err)
		return 0, "", false
	}
	if projectID == 0 {
		httpx.WriteLocalizedErrorCode(ginCtx, r.ctx.I18n, http.StatusBadRequest, projectcontract.ProjectInvalidID.String(), messagecontract.CommonInvalidArgument.String(), map[string]any{"field": "id", "code": projectcontract.ProjectInvalidID.String()})
		ginCtx.Abort()
		return 0, "", false
	}
	return projectID, raw, true
}

// bindImportValidateParams 组装导入校验接口的请求参数，包含请求语言和请求 ID。
func bindImportValidateParams(ginCtx *gin.Context) generated.PostProjectImportValidateParams {
	locale, requestID := commonHeaders(ginCtx)
	return generated.PostProjectImportValidateParams{XGraftLocale: locale, XRequestId: requestID}
}

// bindPostProjectImportParams 组装项目导入接口的请求参数。
// 它从请求头中提取 `XGraftLocale` 和 `XRequestId` 并填充到返回值中。
func bindPostProjectImportParams(ginCtx *gin.Context) generated.PostProjectImportParams {
	locale, requestID := commonHeaders(ginCtx)
	return generated.PostProjectImportParams{XGraftLocale: locale, XRequestId: requestID}
}

// bindPostProjectImportInspectParams 组装项目导入检查请求所需的公共头参数。
// bindPostProjectImportInspectParams 构造项目导入检查请求参数，并填充本地化语言和请求 ID。
func bindPostProjectImportInspectParams(ginCtx *gin.Context) generated.PostProjectImportInspectParams {
	locale, requestID := commonHeaders(ginCtx)
	return generated.PostProjectImportInspectParams{XGraftLocale: locale, XRequestId: requestID}
}

// bindGetProjectCreationMethodsParams assembles common headers for the creation-method catalog.
func bindGetProjectCreationMethodsParams(ginCtx *gin.Context) generated.GetProjectCreationMethodsParams {
	locale, requestID := commonHeaders(ginCtx)
	return generated.GetProjectCreationMethodsParams{XGraftLocale: locale, XRequestId: requestID}
}

// bindGetProjectDiscoveryCandidatesParams 构造项目发现候选列表请求参数。
//
// 它从请求头中提取语言和请求 ID，并填充到生成的参数中。
func bindGetProjectDiscoveryCandidatesParams(ginCtx *gin.Context) generated.GetProjectDiscoveryCandidatesParams {
	locale, requestID := commonHeaders(ginCtx)
	return generated.GetProjectDiscoveryCandidatesParams{XGraftLocale: locale, XRequestId: requestID}
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

// bindGetProjectParams 生成获取项目接口的请求参数，包含语言和请求 ID。
func bindGetProjectParams(ginCtx *gin.Context) generated.GetProjectParams {
	locale, requestID := commonHeaders(ginCtx)
	return generated.GetProjectParams{XGraftLocale: locale, XRequestId: requestID}
}

// bindGetProjectServicesParams 构造获取项目服务列表请求的公共参数。
func bindGetProjectServicesParams(ginCtx *gin.Context) generated.GetProjectServicesParams {
	locale, requestID := commonHeaders(ginCtx)
	return generated.GetProjectServicesParams{XGraftLocale: locale, XRequestId: requestID}
}

// bindGetProjectLogsParams 绑定项目日志查询参数，并在参数无效时中止请求。
// 返回解析后的参数和参数是否有效。
func bindGetProjectLogsParams(ginCtx *gin.Context, ctx *module.Context) (generated.GetProjectLogsParams, bool) {
	locale, requestID := commonHeaders(ginCtx)
	params := generated.GetProjectLogsParams{XGraftLocale: locale, XRequestId: requestID}
	if value, ok := optionalIntQuery[generated.ProjectLogsTail](ginCtx.Query("tail"), 1, maxProjectLogsTail); ok {
		params.Tail = value
	} else {
		abortInvalidQuery(ginCtx, ctx)
		return generated.GetProjectLogsParams{}, false
	}
	if value := strings.TrimSpace(ginCtx.Query("since")); value != "" {
		params.Since = &value
	}
	if value, ok := optionalBoolQuery(ginCtx.Query("timestamps")); ok {
		params.Timestamps = value
	} else {
		abortInvalidQuery(ginCtx, ctx)
		return generated.GetProjectLogsParams{}, false
	}
	if value, ok := optionalBoolQuery(ginCtx.Query("stdout")); ok {
		params.Stdout = value
	} else {
		abortInvalidQuery(ginCtx, ctx)
		return generated.GetProjectLogsParams{}, false
	}
	if value, ok := optionalBoolQuery(ginCtx.Query("stderr")); ok {
		params.Stderr = value
	} else {
		abortInvalidQuery(ginCtx, ctx)
		return generated.GetProjectLogsParams{}, false
	}
	return params, true
}

func bindGetProjectOverviewParams(ginCtx *gin.Context) generated.GetProjectOverviewParams {
	locale, requestID := commonHeaders(ginCtx)
	return generated.GetProjectOverviewParams{XGraftLocale: locale, XRequestId: requestID}
}

// bindGetProjectConfigurationParams 组装获取项目配置接口的请求参数。
//
// 它从请求头提取 locale 和 request ID，并填充到对应的生成参数中。
func bindGetProjectConfigurationParams(ginCtx *gin.Context) generated.GetProjectConfigurationParams {
	locale, requestID := commonHeaders(ginCtx)
	return generated.GetProjectConfigurationParams{XGraftLocale: locale, XRequestId: requestID}
}

// bindGetProjectConfigurationPreviewParams 构造项目配置预览接口的公共请求参数。
// 它包含从请求头提取的语言区域和请求 ID。
func bindGetProjectConfigurationPreviewParams(ginCtx *gin.Context) generated.GetProjectConfigurationPreviewParams {
	locale, requestID := commonHeaders(ginCtx)
	return generated.GetProjectConfigurationPreviewParams{XGraftLocale: locale, XRequestId: requestID}
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

func bindGetProjectFilesParams(ginCtx *gin.Context, query workspaceFileBrowseQuery) generated.GetProjectFilesParams {
	locale, requestID := commonHeaders(ginCtx)
	params := generated.GetProjectFilesParams{XGraftLocale: locale, XRequestId: requestID}
	if trimmed := strings.TrimSpace(query.Path); trimmed != "" {
		params.Path = &trimmed
	}
	if query.ShowHidden {
		showHidden := true
		params.ShowHidden = &showHidden
	}
	return params
}

func bindGetProjectFileContentParams(ginCtx *gin.Context, path string) generated.GetProjectFileContentParams {
	locale, requestID := commonHeaders(ginCtx)
	queryPath := generated.ProjectWorkspacePathQuery(path)
	return generated.GetProjectFileContentParams{XGraftLocale: locale, XRequestId: requestID, Path: &queryPath}
}

func bindPutProjectFileContentParams(ginCtx *gin.Context, path string) generated.PutProjectFileContentParams {
	locale, requestID := commonHeaders(ginCtx)
	queryPath := generated.ProjectWorkspacePathQuery(path)
	return generated.PutProjectFileContentParams{XGraftLocale: locale, XRequestId: requestID, Path: &queryPath}
}

// bindPutProjectFileAnnotationParams 构造更新项目文件注释所需的请求参数。
//
// Path 会被编码为工作区路径查询参数。
func bindPutProjectFileAnnotationParams(ginCtx *gin.Context, path string) generated.PutProjectFileAnnotationParams {
	locale, requestID := commonHeaders(ginCtx)
	queryPath := generated.ProjectWorkspacePathQuery(path)
	return generated.PutProjectFileAnnotationParams{XGraftLocale: locale, XRequestId: requestID, Path: &queryPath}
}

// bindGetProjectManagedRootParams 构造获取托管根信息请求的公共参数。
//
// 它包含请求的语言环境和请求 ID。
func bindGetProjectManagedRootParams(ginCtx *gin.Context) generated.GetProjectManagedRootParams {
	locale, requestID := commonHeaders(ginCtx)
	return generated.GetProjectManagedRootParams{XGraftLocale: locale, XRequestId: requestID}
}

// bindPostProjectCreateValidateParams 构造项目创建校验接口的公共请求参数。
// 它包含语言信息和请求 ID。
func bindPostProjectCreateValidateParams(ginCtx *gin.Context) generated.PostProjectCreateValidateParams {
	locale, requestID := commonHeaders(ginCtx)
	return generated.PostProjectCreateValidateParams{XGraftLocale: locale, XRequestId: requestID}
}

// bindPostProjectCreateParams 构造创建项目请求的公共请求头参数。
// @returns 包含 `XGraftLocale` 和 `XRequestId` 的创建项目请求参数。
func bindPostProjectCreateParams(ginCtx *gin.Context) generated.PostProjectCreateParams {
	locale, requestID := commonHeaders(ginCtx)
	return generated.PostProjectCreateParams{XGraftLocale: locale, XRequestId: requestID}
}

// bindPostProjectRefreshParams 构造项目刷新接口的请求头参数。
func bindPostProjectRefreshParams(ginCtx *gin.Context) generated.PostProjectRefreshParams {
	locale, requestID := commonHeaders(ginCtx)
	return generated.PostProjectRefreshParams{XGraftLocale: locale, XRequestId: requestID}
}

// bindPostProjectDeployParams 构造项目部署接口的通用请求参数。
func bindPostProjectDeployParams(ginCtx *gin.Context) generated.PostProjectDeployParams {
	locale, requestID := commonHeaders(ginCtx)
	return generated.PostProjectDeployParams{XGraftLocale: locale, XRequestId: requestID}
}

// bindPostProjectUpParams 组装项目启动接口的请求参数，包含语言环境和请求 ID。
func bindPostProjectUpParams(ginCtx *gin.Context) generated.PostProjectUpParams {
	locale, requestID := commonHeaders(ginCtx)
	return generated.PostProjectUpParams{XGraftLocale: locale, XRequestId: requestID}
}

// bindPostProjectStopParams 组装项目停止接口的公共请求参数。
// 它包含请求的语言标识和请求 ID。
func bindPostProjectStopParams(ginCtx *gin.Context) generated.PostProjectStopParams {
	locale, requestID := commonHeaders(ginCtx)
	return generated.PostProjectStopParams{XGraftLocale: locale, XRequestId: requestID}
}

// bindPostProjectRestartParams 构造重启项目接口的公共请求参数。
// 其中包含从请求头提取的语言环境和请求 ID。
func bindPostProjectRestartParams(ginCtx *gin.Context) generated.PostProjectRestartParams {
	locale, requestID := commonHeaders(ginCtx)
	return generated.PostProjectRestartParams{XGraftLocale: locale, XRequestId: requestID}
}

func bindPostProjectRedeployParams(ginCtx *gin.Context) generated.PostProjectRedeployParams {
	locale, requestID := commonHeaders(ginCtx)
	return generated.PostProjectRedeployParams{XGraftLocale: locale, XRequestId: requestID}
}

func bindPutProjectLifecycleConfigurationParams(ginCtx *gin.Context) generated.PutProjectLifecycleConfigurationParams {
	locale, requestID := commonHeaders(ginCtx)
	return generated.PutProjectLifecycleConfigurationParams{XGraftLocale: locale, XRequestId: requestID}
}

func bindPostProjectBatchActionsParams(ginCtx *gin.Context) generated.PostProjectBatchActionsParams {
	locale, requestID := commonHeaders(ginCtx)
	return generated.PostProjectBatchActionsParams{XGraftLocale: locale, XRequestId: requestID}
}

// bindPostProjectUnregisterParams 构造项目取消注册接口的请求参数。
// 它包含请求语言和请求 ID。
func bindPostProjectUnregisterParams(ginCtx *gin.Context) generated.PostProjectUnregisterParams {
	locale, requestID := commonHeaders(ginCtx)
	return generated.PostProjectUnregisterParams{XGraftLocale: locale, XRequestId: requestID}
}

// bindPostProjectDestroyParams 构造项目销毁接口的公共请求参数。
func bindPostProjectDestroyParams(ginCtx *gin.Context) generated.PostProjectDestroyParams {
	locale, requestID := commonHeaders(ginCtx)
	return generated.PostProjectDestroyParams{XGraftLocale: locale, XRequestId: requestID}
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
func toImportRequest(ginCtx *gin.Context, request generated.ProjectImportValidateRequest) ImportRequest {
	return ImportRequest{
		WorkingDirectory:             request.WorkingDirectory,
		DisplayName:                  request.DisplayName,
		ComposeFiles:                 slicePtrValue(request.ComposeFiles),
		EnvFiles:                     slicePtrValue(request.EnvFiles),
		CanonicalProjectNameOverride: request.CanonicalProjectNameOverride,
		ActorID:                      currentUserIDPointer(ginCtx),
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

// currentUserID returns the authenticated user's non-zero ID and whether it is available.
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
		r.writeLocalizedProjectError(ginCtx, http.StatusConflict, projectcontract.ProjectSavedViewConflict.String())
	case errors.Is(err, errProjectNotFound):
		r.writeLocalizedProjectError(ginCtx, http.StatusNotFound, projectcontract.ProjectSavedViewNotFound.String())
	default:
		r.writeLocalizedProjectError(ginCtx, http.StatusBadRequest, projectcontract.ProjectSavedViewInvalid.String())
	}
	ginCtx.Abort()
}

// mapLifecycleErrorCode 将生命周期错误映射为对应的错误码字符串。
// 当错误为 errProjectManagedFlow 时返回 ProjectManagedFlowUnsupported，
// 否则返回 ProjectUnsupportedLifecycle。
func mapLifecycleErrorCode(err error) string {
	if errors.Is(err, errProjectManagedFlow) {
		return projectcontract.ProjectManagedFlowUnsupported.String()
	}
	return projectcontract.ProjectUnsupportedLifecycle.String()
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
