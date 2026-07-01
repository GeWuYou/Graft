package project

import (
	"errors"
	"fmt"
	"math"
	"net/http"
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
)

type routeRuntime struct {
	ctx     *module.Context
	service *Service
}

const minimumProjectListLimit = 1

type boundProjectConfigurationDraft[T any] struct {
	projectID   uint64
	generatedID int64
	request     T
}

// registerRoutes 为项目模块注册路由并挂载权限校验与请求追踪中间件。
// 当路由器不可用时直接返回；当服务缺失时返回错误。
// 注册的接口覆盖项目列表、导入、创建、详情、配置、刷新、部署及生命周期和销毁操作。
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

	routes := routeRuntime{ctx: ctx, service: service}
	publisher := httpx.NewSecurityAuditPublisher(ctx.EventBus, ctx.Logger, moduleName)
	group := ctx.Router.Group(projectcontract.ProjectAPIGroup)
	group.Use(httpx.RequestIDMiddleware())
	group.GET(projectcontract.ProjectCollectionRoute, httpx.RequirePermission(ctx.I18n, authService, authorizer, projectcontract.ProjectViewPermission.String(), publisher), routes.handleList)
	group.POST(projectcontract.ProjectImportValidateRoute, httpx.RequirePermission(ctx.I18n, authService, authorizer, projectcontract.ProjectImportPermission.String(), publisher), routes.handleImportValidate)
	group.POST(projectcontract.ProjectImportInspectRoute, httpx.RequirePermission(ctx.I18n, authService, authorizer, projectcontract.ProjectImportPermission.String(), publisher), routes.handleImportInspect)
	group.POST(projectcontract.ProjectImportRoute, httpx.RequirePermission(ctx.I18n, authService, authorizer, projectcontract.ProjectImportPermission.String(), publisher), routes.handleImport)
	group.GET(projectcontract.ProjectImportDirectorySourcesRoute, httpx.RequirePermission(ctx.I18n, authService, authorizer, projectcontract.ProjectImportPermission.String(), publisher), routes.handleImportDirectorySources)
	group.GET(projectcontract.ProjectImportDirectoriesRoute, httpx.RequirePermission(ctx.I18n, authService, authorizer, projectcontract.ProjectImportPermission.String(), publisher), routes.handleImportDirectories)
	group.GET(projectcontract.ProjectSourcesRoute, httpx.RequirePermission(ctx.I18n, authService, authorizer, projectcontract.ProjectSourceViewPermission.String(), publisher), routes.handleSources)
	group.GET(projectcontract.ProjectDiscoveryCandidatesRoute, httpx.RequirePermission(ctx.I18n, authService, authorizer, projectcontract.ProjectDiscoveryViewPermission.String(), publisher), routes.handleDiscoveryCandidates)
	group.GET(projectcontract.ProjectManagedRootRoute, httpx.RequirePermission(ctx.I18n, authService, authorizer, projectcontract.ProjectCreatePermission.String(), publisher), routes.handleManagedRoot)
	group.POST(projectcontract.ProjectCreateValidateRoute, httpx.RequirePermission(ctx.I18n, authService, authorizer, projectcontract.ProjectCreatePermission.String(), publisher), routes.handleCreateValidate)
	group.POST(projectcontract.ProjectCreateRoute, httpx.RequirePermission(ctx.I18n, authService, authorizer, projectcontract.ProjectCreatePermission.String(), publisher), routes.handleCreate)
	group.GET(projectcontract.ProjectDetailRoute, httpx.RequirePermission(ctx.I18n, authService, authorizer, projectcontract.ProjectViewPermission.String(), publisher), routes.handleDetail)
	group.GET(projectcontract.ProjectServicesRoute, httpx.RequirePermission(ctx.I18n, authService, authorizer, projectcontract.ProjectViewPermission.String(), publisher), routes.handleServices)
	group.GET(projectcontract.ProjectConfigurationRoute, httpx.RequirePermission(ctx.I18n, authService, authorizer, projectcontract.ProjectViewPermission.String(), publisher), routes.handleConfiguration)
	group.GET(projectcontract.ProjectConfigurationPreviewRoute, httpx.RequirePermission(ctx.I18n, authService, authorizer, projectcontract.ProjectViewPermission.String(), publisher), routes.handleConfigurationPreview)
	group.GET(projectcontract.ProjectConfigurationFileRoute, httpx.RequirePermission(ctx.I18n, authService, authorizer, projectcontract.ProjectViewPermission.String(), publisher), routes.handleConfigurationFile)
	group.POST(projectcontract.ProjectConfigurationDiffRoute, httpx.RequirePermission(ctx.I18n, authService, authorizer, projectcontract.ProjectDeployPermission.String(), publisher), routes.handleConfigurationDiff)
	group.POST(projectcontract.ProjectConfigurationValidateRoute, httpx.RequirePermission(ctx.I18n, authService, authorizer, projectcontract.ProjectDeployPermission.String(), publisher), routes.handleConfigurationValidate)
	group.POST(projectcontract.ProjectRefreshRoute, httpx.RequirePermission(ctx.I18n, authService, authorizer, projectcontract.ProjectRefreshPermission.String(), publisher), routes.handleRefresh)
	group.POST(projectcontract.ProjectDeployRoute, httpx.RequirePermission(ctx.I18n, authService, authorizer, projectcontract.ProjectDeployPermission.String(), publisher), routes.handleDeploy)
	group.POST(projectcontract.ProjectUpRoute, httpx.RequirePermission(ctx.I18n, authService, authorizer, projectcontract.ProjectLifecyclePermission.String(), publisher), routes.handleUp)
	group.POST(projectcontract.ProjectDownRoute, httpx.RequirePermission(ctx.I18n, authService, authorizer, projectcontract.ProjectLifecyclePermission.String(), publisher), routes.handleDown)
	group.POST(projectcontract.ProjectRestartRoute, httpx.RequirePermission(ctx.I18n, authService, authorizer, projectcontract.ProjectLifecyclePermission.String(), publisher), routes.handleRestart)
	group.POST(projectcontract.ProjectUnregisterRoute, httpx.RequirePermission(ctx.I18n, authService, authorizer, projectcontract.ProjectDestroyPermission.String(), publisher), routes.handleUnregister)
	group.POST(projectcontract.ProjectDestroyRoute, httpx.RequirePermission(ctx.I18n, authService, authorizer, projectcontract.ProjectDestroyPermission.String(), publisher), routes.handleDestroy)
	return nil
}

func (r routeRuntime) handleList(ginCtx *gin.Context) {
	params, ok := bindListParams(ginCtx, r.ctx)
	if !ok {
		return
	}
	projectGeneratedHandler{}.GetProjects(params)
	result, err := r.service.List(ginCtx.Request.Context(), ListQuery{
		Limit:             intPtrValue(params.Limit),
		Offset:            intPtrValue(params.Offset),
		SourceKind:        stringPtrValue(params.SourceKind),
		DriftStatus:       stringPtrValue(params.DriftStatus),
		LastRefreshStatus: stringPtrValue(params.LastRefreshStatus),
	})
	if err != nil {
		r.writeRouteError(ginCtx, err)
		return
	}
	httpx.WriteSuccess(ginCtx, http.StatusOK, toProjectListResponse(result))
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

func (r routeRuntime) handleImport(ginCtx *gin.Context) {
	var request generated.PostProjectImportJSONRequestBody
	if !bindJSON(ginCtx, r.ctx, &request) {
		return
	}
	projectGeneratedHandler{}.PostProjectImport(bindPostProjectImportParams(ginCtx), request)
	result, err := r.service.ImportByInspection(ginCtx.Request.Context(), ImportExecuteRequest{
		InspectionID:                 request.InspectionId,
		DisplayName:                  request.DisplayName,
		CanonicalProjectNameOverride: request.CanonicalProjectNameOverride,
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

func (r routeRuntime) handleSources(ginCtx *gin.Context) {
	projectGeneratedHandler{}.GetProjectSources(bindGetProjectSourcesParams(ginCtx))
	result, err := r.service.SourceCatalog(ginCtx.Request.Context())
	if err != nil {
		r.writeRouteError(ginCtx, err)
		return
	}
	httpx.WriteSuccess(ginCtx, http.StatusOK, toSourceCatalogResponse(result))
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
	result, err := r.service.ValidateManagedCreate(ginCtx.Request.Context(), toManagedCreateRequest(request))
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
	result, err := r.service.CreateManagedProject(ginCtx.Request.Context(), toManagedCreateExecuteRequest(request), currentUserIDPointer(ginCtx))
	if err != nil {
		r.writeRouteError(ginCtx, err)
		return
	}
	httpx.WriteSuccess(ginCtx, http.StatusCreated, toManagedCreateResponse(result))
}

func (r routeRuntime) handleDetail(ginCtx *gin.Context) {
	projectID, generatedID, ok := bindProjectID(ginCtx, r.ctx)
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

func (r routeRuntime) handleServices(ginCtx *gin.Context) {
	projectID, generatedID, ok := bindProjectID(ginCtx, r.ctx)
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

func (r routeRuntime) handleConfiguration(ginCtx *gin.Context) {
	projectID, generatedID, ok := bindProjectID(ginCtx, r.ctx)
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
	projectID, generatedID, ok := bindProjectID(ginCtx, r.ctx)
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

func (r routeRuntime) handleConfigurationFile(ginCtx *gin.Context) {
	projectID, generatedProjectID, ok := bindProjectID(ginCtx, r.ctx)
	if !ok {
		return
	}
	fileID, generatedFileID, ok := bindProjectFileID(ginCtx, r.ctx)
	if !ok {
		return
	}
	projectGeneratedHandler{}.GetProjectConfigurationFile(generatedProjectID, generatedFileID, bindGetProjectConfigurationFileParams(ginCtx))
	result, err := r.service.ConfigurationFile(ginCtx.Request.Context(), projectID, fileID)
	if err != nil {
		r.writeRouteError(ginCtx, err)
		return
	}
	httpx.WriteSuccess(ginCtx, http.StatusOK, toConfigurationFileResponse(result))
}

func (r routeRuntime) handleConfigurationDiff(ginCtx *gin.Context) {
	bound, ok := bindProjectConfigurationDiffRequest(ginCtx, r)
	if !ok {
		return
	}
	projectGeneratedHandler{}.PostProjectConfigurationDiff(bound.generatedID, bindPostProjectConfigurationDiffParams(ginCtx), bound.request)
	result, err := r.service.DiffConfiguration(ginCtx.Request.Context(), bound.projectID, toConfigurationDiffRequest(bound.request))
	if err != nil {
		r.writeRouteError(ginCtx, err)
		return
	}
	httpx.WriteSuccess(ginCtx, http.StatusOK, toConfigurationDiffResponse(result))
}

func (r routeRuntime) handleConfigurationValidate(ginCtx *gin.Context) {
	bound, ok := bindProjectConfigurationValidateRequest(ginCtx, r)
	if !ok {
		return
	}
	projectGeneratedHandler{}.PostProjectConfigurationValidate(bound.generatedID, bindPostProjectConfigurationValidateParams(ginCtx), bound.request)
	result, err := r.service.ValidateConfiguration(ginCtx.Request.Context(), bound.projectID, toConfigurationValidateRequest(bound.request))
	if err != nil {
		r.writeRouteError(ginCtx, err)
		return
	}
	httpx.WriteSuccess(ginCtx, http.StatusOK, toConfigurationValidateResponse(result))
}

func (r routeRuntime) handleRefresh(ginCtx *gin.Context) {
	projectID, generatedID, ok := bindProjectID(ginCtx, r.ctx)
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
	projectID, generatedID, ok := bindProjectID(ginCtx, r.ctx)
	if !ok {
		return
	}
	var request generated.ProjectDeployRequest
	if !bindJSON(ginCtx, r.ctx, &request) {
		return
	}
	projectGeneratedHandler{}.PostProjectDeploy(generatedID, bindPostProjectDeployParams(ginCtx), request)
	result, err := r.service.DeployConfiguration(ginCtx.Request.Context(), projectID, toDeployRequest(request), currentUserIDPointer(ginCtx))
	if err != nil {
		r.writeRouteError(ginCtx, err)
		return
	}
	httpx.WriteSuccess(ginCtx, http.StatusOK, toDeployResponse(result))
}

func (r routeRuntime) handleUp(ginCtx *gin.Context) {
	projectID, generatedID, ok := bindProjectID(ginCtx, r.ctx)
	if !ok {
		return
	}
	projectGeneratedHandler{}.PostProjectUp(generatedID, bindPostProjectUpParams(ginCtx))
	result, err := r.service.Up(ginCtx.Request.Context(), projectID, currentUserIDPointer(ginCtx))
	if err != nil {
		r.writeRouteErrorWithAction(ginCtx, err, result)
		return
	}
	httpx.WriteSuccess(ginCtx, http.StatusOK, toActionResponse(result))
}

func (r routeRuntime) handleDown(ginCtx *gin.Context) {
	projectID, generatedID, ok := bindProjectID(ginCtx, r.ctx)
	if !ok {
		return
	}
	projectGeneratedHandler{}.PostProjectDown(generatedID, bindPostProjectDownParams(ginCtx))
	result, err := r.service.Down(ginCtx.Request.Context(), projectID, currentUserIDPointer(ginCtx))
	if err != nil {
		r.writeRouteErrorWithAction(ginCtx, err, result)
		return
	}
	httpx.WriteSuccess(ginCtx, http.StatusOK, toActionResponse(result))
}

func (r routeRuntime) handleRestart(ginCtx *gin.Context) {
	projectID, generatedID, ok := bindProjectID(ginCtx, r.ctx)
	if !ok {
		return
	}
	projectGeneratedHandler{}.PostProjectRestart(generatedID, bindPostProjectRestartParams(ginCtx))
	result, err := r.service.Restart(ginCtx.Request.Context(), projectID, currentUserIDPointer(ginCtx))
	if err != nil {
		r.writeRouteErrorWithAction(ginCtx, err, result)
		return
	}
	httpx.WriteSuccess(ginCtx, http.StatusOK, toActionResponse(result))
}

func (r routeRuntime) handleUnregister(ginCtx *gin.Context) {
	projectID, generatedID, ok := bindProjectID(ginCtx, r.ctx)
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
	projectID, generatedID, ok := bindProjectID(ginCtx, r.ctx)
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
		DeleteWorkingDirectory:      request.DeleteWorkingDirectory,
		ConfirmCanonicalProjectName: request.ConfirmCanonicalProjectName,
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
	switch {
	case errors.Is(err, errProjectInvalidArgument), errors.Is(err, errProjectImportValidation), errors.Is(err, errProjectFileNotFound):
		r.writeInvalidArgumentError(ginCtx, err)
	case errors.Is(err, errProjectNotFound):
		r.writeLocalizedProjectError(ginCtx, http.StatusNotFound, projectcontract.ProjectNotFound.String())
	case errors.Is(err, errProjectConflict):
		r.writeLocalizedProjectError(ginCtx, http.StatusConflict, projectcontract.ProjectConflict.String())
	case errors.Is(err, errProjectDirectoryForbidden):
		r.writeLocalizedProjectError(ginCtx, http.StatusForbidden, projectcontract.ProjectDirectoryBrowseForbidden.String())
	case errors.Is(err, errProjectInspectionExpired):
		r.writeLocalizedProjectError(ginCtx, http.StatusConflict, projectcontract.ProjectInspectionExpired.String())
	case errors.Is(err, errProjectInspectionStale):
		r.writeLocalizedProjectError(ginCtx, http.StatusConflict, projectcontract.ProjectInspectionStale.String())
	case errors.Is(err, errProjectDestroyBlocked):
		r.writeLocalizedActionError(ginCtx, http.StatusConflict, projectcontract.ProjectConflict.String(), map[string]any{
			"code":         projectcontract.ProjectConflict.String(),
			"actionResult": toActionResponse(action),
		})
	case errors.Is(err, errProjectUnsupportedLifecycle), errors.Is(err, errProjectManagedFlow):
		r.writeLocalizedActionError(ginCtx, http.StatusConflict, projectcontract.ProjectUnsupportedLifecycle.String(), map[string]any{
			"code":         mapLifecycleErrorCode(err),
			"actionResult": toActionResponse(action),
		})
	default:
		httpx.WriteLocalizedErrorCode(ginCtx, r.ctx.I18n, http.StatusInternalServerError, messagecontract.CommonInternalError.String(), messagecontract.CommonInternalError.String(), nil)
	}
	ginCtx.Abort()
}

func (r routeRuntime) writeInvalidArgumentError(ginCtx *gin.Context, err error) {
	code := projectcontract.ProjectInvalidArgument.String()
	if errors.Is(err, errProjectFileNotFound) {
		code = projectcontract.ProjectInvalidFileID.String()
	}
	r.writeLocalizedActionError(ginCtx, http.StatusBadRequest, projectcontract.ProjectInvalidArgument.String(), map[string]any{"code": code})
}

func (r routeRuntime) writeLocalizedProjectError(ginCtx *gin.Context, status int, code string) {
	r.writeLocalizedActionError(ginCtx, status, code, map[string]any{"code": code})
}

func (r routeRuntime) writeLocalizedActionError(ginCtx *gin.Context, status int, code string, details map[string]any) {
	httpx.WriteLocalizedErrorCode(ginCtx, r.ctx.I18n, status, code, messagecontract.CommonInvalidArgument.String(), details)
}

type projectGeneratedHandler struct{}

func (projectGeneratedHandler) GetProjects(generated.GetProjectsParams)             {}
func (projectGeneratedHandler) GetProjectSources(generated.GetProjectSourcesParams) {}
func (projectGeneratedHandler) GetProjectDiscoveryCandidates(generated.GetProjectDiscoveryCandidatesParams) {
}
func (projectGeneratedHandler) PostProjectImportValidate(generated.PostProjectImportValidateParams, generated.PostProjectImportValidateJSONRequestBody) {
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
func (projectGeneratedHandler) GetProject(int64, generated.GetProjectParams)                 {}
func (projectGeneratedHandler) GetProjectServices(int64, generated.GetProjectServicesParams) {}
func (projectGeneratedHandler) GetProjectConfiguration(int64, generated.GetProjectConfigurationParams) {
}
func (projectGeneratedHandler) GetProjectConfigurationPreview(int64, generated.GetProjectConfigurationPreviewParams) {
}
func (projectGeneratedHandler) GetProjectConfigurationFile(int64, int64, generated.GetProjectConfigurationFileParams) {
}
func (projectGeneratedHandler) PostProjectConfigurationDiff(int64, generated.PostProjectConfigurationDiffParams, generated.ProjectConfigurationDiffRequest) {
}
func (projectGeneratedHandler) PostProjectConfigurationValidate(int64, generated.PostProjectConfigurationValidateParams, generated.ProjectConfigurationValidateRequest) {
}
func (projectGeneratedHandler) PostProjectRefresh(int64, generated.PostProjectRefreshParams) {}
func (projectGeneratedHandler) PostProjectDeploy(int64, generated.PostProjectDeployParams, generated.ProjectDeployRequest) {
}
func (projectGeneratedHandler) PostProjectUp(int64, generated.PostProjectUpParams)           {}
func (projectGeneratedHandler) PostProjectDown(int64, generated.PostProjectDownParams)       {}
func (projectGeneratedHandler) PostProjectRestart(int64, generated.PostProjectRestartParams) {}
func (projectGeneratedHandler) PostProjectUnregister(int64, generated.PostProjectUnregisterParams) {
}
func (projectGeneratedHandler) PostProjectDestroy(int64, generated.PostProjectDestroyParams, generated.PostProjectDestroyJSONRequestBody) {
}

// bindListParams 绑定项目列表查询参数和公共请求头。
// 它解析 source_kind、drift_status、last_refresh_status、limit 和 offset，并在分页参数无效时中止请求。
func bindListParams(ginCtx *gin.Context, ctx *module.Context) (generated.GetProjectsParams, bool) {
	locale, requestID := commonHeaders(ginCtx)
	query := ginCtx.Request.URL.Query()
	params := generated.GetProjectsParams{
		XGraftLocale: locale,
		XRequestId:   requestID,
	}
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
	lastRefreshStatus, ok := optionalValidatedEnumQuery(query.Get("last_refresh_status"), generated.ProjectRefreshStatus.Valid)
	if !ok {
		abortInvalidQuery(ginCtx, ctx)
		return generated.GetProjectsParams{}, false
	}
	params.SourceKind = sourceKind
	params.DriftStatus = driftStatus
	params.LastRefreshStatus = lastRefreshStatus
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

// bindProjectConfigurationDiffRequest 绑定配置差异请求及其项目标识。
// 成功时返回包含项目 ID、生成 ID 和请求体的结果；绑定失败时返回 false。
func bindProjectConfigurationDiffRequest(ginCtx *gin.Context, r routeRuntime) (boundProjectConfigurationDraft[generated.ProjectConfigurationDiffRequest], bool) {
	var request generated.ProjectConfigurationDiffRequest
	projectID, generatedID, ok := bindProjectConfigurationDraftRequest(ginCtx, r, &request)
	if !ok {
		return boundProjectConfigurationDraft[generated.ProjectConfigurationDiffRequest]{}, false
	}
	return boundProjectConfigurationDraft[generated.ProjectConfigurationDiffRequest]{
		projectID:   projectID,
		generatedID: generatedID,
		request:     request,
	}, true
}

// bindProjectConfigurationValidateRequest 绑定项目配置校验请求及其路径标识。
// 成功时返回包含 projectID、generatedID 和请求体的绑定结果；绑定失败时返回 false。
func bindProjectConfigurationValidateRequest(ginCtx *gin.Context, r routeRuntime) (boundProjectConfigurationDraft[generated.ProjectConfigurationValidateRequest], bool) {
	var request generated.ProjectConfigurationValidateRequest
	projectID, generatedID, ok := bindProjectConfigurationDraftRequest(ginCtx, r, &request)
	if !ok {
		return boundProjectConfigurationDraft[generated.ProjectConfigurationValidateRequest]{}, false
	}
	return boundProjectConfigurationDraft[generated.ProjectConfigurationValidateRequest]{
		projectID:   projectID,
		generatedID: generatedID,
		request:     request,
	}, true
}

// bindProjectConfigurationDraftRequest 绑定项目标识和配置草稿请求体。
// 成功时返回项目 ID、生成 ID 和 true。
func bindProjectConfigurationDraftRequest[T any](ginCtx *gin.Context, r routeRuntime, request *T) (uint64, int64, bool) {
	projectID, generatedID, ok := bindProjectID(ginCtx, r.ctx)
	if !ok {
		return 0, 0, false
	}
	if !bindJSON(ginCtx, r.ctx, request) {
		return 0, 0, false
	}
	return projectID, generatedID, true
}

// bindJSON 绑定请求体中的 JSON 到目标对象。
//
// 绑定失败时，会中止当前请求并返回 400 Bad Request 的本地化参数错误，错误字段为 `body`。
func bindJSON[T any](ginCtx *gin.Context, ctx *module.Context, target *T) bool {
	if err := ginCtx.ShouldBindJSON(target); err != nil {
		httpx.AbortLocalizedError(ginCtx, ctx.I18n, http.StatusBadRequest, messagecontract.CommonInvalidArgument.String(), map[string]any{"field": "body"})
		return false
	}
	return true
}

// bindProjectID 解析并校验项目路由参数 id。
// 解析失败、值为 0 或超出 int64 范围时，会返回本地化的参数错误并中止请求。
func bindProjectID(ginCtx *gin.Context, ctx *module.Context) (uint64, int64, bool) {
	raw := strings.TrimSpace(ginCtx.Param("id"))
	value, err := strconv.ParseUint(raw, 10, 64)
	if err != nil || value == 0 {
		httpx.WriteLocalizedErrorCode(ginCtx, ctx.I18n, http.StatusBadRequest, projectcontract.ProjectInvalidID.String(), messagecontract.CommonInvalidArgument.String(), map[string]any{"field": "id", "code": projectcontract.ProjectInvalidID.String()})
		ginCtx.Abort()
		return 0, 0, false
	}
	if value > math.MaxInt64 {
		httpx.WriteLocalizedErrorCode(ginCtx, ctx.I18n, http.StatusBadRequest, projectcontract.ProjectInvalidID.String(), messagecontract.CommonInvalidArgument.String(), map[string]any{"field": "id", "code": projectcontract.ProjectInvalidID.String()})
		ginCtx.Abort()
		return 0, 0, false
	}
	return value, int64(value), true
}

// bindProjectFileID 解析并校验路由参数中的文件 ID。
// 成功时返回文件 ID 及其 int64 形式；校验失败时写入本地化错误并中止请求。
func bindProjectFileID(ginCtx *gin.Context, ctx *module.Context) (uint64, int64, bool) {
	raw := strings.TrimSpace(ginCtx.Param("fileId"))
	value, err := strconv.ParseUint(raw, 10, 64)
	if err != nil || value == 0 || value > math.MaxInt64 {
		httpx.WriteLocalizedErrorCode(ginCtx, ctx.I18n, http.StatusBadRequest, projectcontract.ProjectInvalidFileID.String(), messagecontract.CommonInvalidArgument.String(), map[string]any{"field": "fileId", "code": projectcontract.ProjectInvalidFileID.String()})
		ginCtx.Abort()
		return 0, 0, false
	}
	return value, int64(value), true
}

// 返回包含请求语言和请求 ID 的参数结构体。
func bindImportValidateParams(ginCtx *gin.Context) generated.PostProjectImportValidateParams {
	locale, requestID := commonHeaders(ginCtx)
	return generated.PostProjectImportValidateParams{XGraftLocale: locale, XRequestId: requestID}
}

func bindPostProjectImportParams(ginCtx *gin.Context) generated.PostProjectImportParams {
	locale, requestID := commonHeaders(ginCtx)
	return generated.PostProjectImportParams{XGraftLocale: locale, XRequestId: requestID}
}

func bindPostProjectImportInspectParams(ginCtx *gin.Context) generated.PostProjectImportInspectParams {
	locale, requestID := commonHeaders(ginCtx)
	return generated.PostProjectImportInspectParams{XGraftLocale: locale, XRequestId: requestID}
}

func bindGetProjectSourcesParams(ginCtx *gin.Context) generated.GetProjectSourcesParams {
	locale, requestID := commonHeaders(ginCtx)
	return generated.GetProjectSourcesParams{XGraftLocale: locale, XRequestId: requestID}
}

func bindGetProjectDiscoveryCandidatesParams(ginCtx *gin.Context) generated.GetProjectDiscoveryCandidatesParams {
	locale, requestID := commonHeaders(ginCtx)
	return generated.GetProjectDiscoveryCandidatesParams{XGraftLocale: locale, XRequestId: requestID}
}

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

// bindGetProjectConfigurationFileParams 构造获取项目配置文件接口的请求参数。
// 该参数包含语言环境和请求 ID。
func bindGetProjectConfigurationFileParams(ginCtx *gin.Context) generated.GetProjectConfigurationFileParams {
	locale, requestID := commonHeaders(ginCtx)
	return generated.GetProjectConfigurationFileParams{XGraftLocale: locale, XRequestId: requestID}
}

// bindPostProjectConfigurationDiffParams 组装项目配置 diff 请求的公共请求头参数。
func bindPostProjectConfigurationDiffParams(ginCtx *gin.Context) generated.PostProjectConfigurationDiffParams {
	locale, requestID := commonHeaders(ginCtx)
	return generated.PostProjectConfigurationDiffParams{XGraftLocale: locale, XRequestId: requestID}
}

// bindPostProjectConfigurationValidateParams 构造配置校验接口的公共请求参数。
// 它包含从请求头提取的语言和请求 ID。
func bindPostProjectConfigurationValidateParams(ginCtx *gin.Context) generated.PostProjectConfigurationValidateParams {
	locale, requestID := commonHeaders(ginCtx)
	return generated.PostProjectConfigurationValidateParams{XGraftLocale: locale, XRequestId: requestID}
}

// bindGetProjectManagedRootParams 构造获取托管根信息请求的公共参数。
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

// bindPostProjectDownParams 组装项目下线接口的公共请求参数。
// 它包含请求的语言标识和请求 ID。
func bindPostProjectDownParams(ginCtx *gin.Context) generated.PostProjectDownParams {
	locale, requestID := commonHeaders(ginCtx)
	return generated.PostProjectDownParams{XGraftLocale: locale, XRequestId: requestID}
}

// bindPostProjectRestartParams 构造重启项目接口的公共请求参数。
// 其中包含从请求头提取的语言环境和请求 ID。
func bindPostProjectRestartParams(ginCtx *gin.Context) generated.PostProjectRestartParams {
	locale, requestID := commonHeaders(ginCtx)
	return generated.PostProjectRestartParams{XGraftLocale: locale, XRequestId: requestID}
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
// 否则返回用户 ID 的指针。
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

// mapLifecycleErrorCode 将生命周期错误映射为对应的错误码字符串。
// 当错误为 errProjectManagedFlow 时返回 ProjectManagedFlowUnsupported，
// 否则返回 ProjectUnsupportedLifecycle。
func mapLifecycleErrorCode(err error) string {
	if errors.Is(err, errProjectManagedFlow) {
		return projectcontract.ProjectManagedFlowUnsupported.String()
	}
	return projectcontract.ProjectUnsupportedLifecycle.String()
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
