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
	group.POST(projectcontract.ProjectImportRoute, httpx.RequirePermission(ctx.I18n, authService, authorizer, projectcontract.ProjectImportPermission.String(), publisher), routes.handleImport)
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
	projectGeneratedHandler{}.PostProjectImport(bindCommonParams(ginCtx), request)
	result, err := r.service.Import(ginCtx.Request.Context(), toImportRequest(ginCtx, request))
	if err != nil {
		r.writeRouteError(ginCtx, err)
		return
	}
	httpx.WriteSuccess(ginCtx, http.StatusOK, result)
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
		details := map[string]any{"code": projectcontract.ProjectInvalidArgument.String()}
		if errors.Is(err, errProjectFileNotFound) {
			details["code"] = projectcontract.ProjectInvalidFileID.String()
		}
		httpx.WriteLocalizedErrorCode(ginCtx, r.ctx.I18n, http.StatusBadRequest, projectcontract.ProjectInvalidArgument.String(), messagecontract.CommonInvalidArgument.String(), details)
	case errors.Is(err, errProjectNotFound):
		httpx.WriteLocalizedErrorCode(ginCtx, r.ctx.I18n, http.StatusNotFound, projectcontract.ProjectNotFound.String(), messagecontract.CommonInvalidArgument.String(), map[string]any{"code": projectcontract.ProjectNotFound.String()})
	case errors.Is(err, errProjectConflict):
		httpx.WriteLocalizedErrorCode(ginCtx, r.ctx.I18n, http.StatusConflict, projectcontract.ProjectConflict.String(), messagecontract.CommonInvalidArgument.String(), map[string]any{"code": projectcontract.ProjectConflict.String()})
	case errors.Is(err, errProjectUnsupportedLifecycle), errors.Is(err, errProjectManagedFlow):
		httpx.WriteLocalizedErrorCode(ginCtx, r.ctx.I18n, http.StatusConflict, projectcontract.ProjectUnsupportedLifecycle.String(), messagecontract.CommonInvalidArgument.String(), map[string]any{
			"code":         mapLifecycleErrorCode(err),
			"actionResult": toActionResponse(action),
		})
	default:
		httpx.WriteLocalizedErrorCode(ginCtx, r.ctx.I18n, http.StatusInternalServerError, messagecontract.CommonInternalError.String(), messagecontract.CommonInternalError.String(), nil)
	}
	ginCtx.Abort()
}

type projectGeneratedHandler struct{}

func (projectGeneratedHandler) GetProjects(generated.GetProjectsParams) {}
func (projectGeneratedHandler) PostProjectImportValidate(generated.PostProjectImportValidateParams, generated.PostProjectImportValidateJSONRequestBody) {
}
func (projectGeneratedHandler) PostProjectImport(generated.PostProjectImportParams, generated.PostProjectImportJSONRequestBody) {
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

func bindListParams(ginCtx *gin.Context, ctx *module.Context) (generated.GetProjectsParams, bool) {
	locale, requestID := commonHeaders(ginCtx)
	query := ginCtx.Request.URL.Query()
	params := generated.GetProjectsParams{
		XGraftLocale:      locale,
		XRequestId:        requestID,
		SourceKind:        optionalTypedQuery[generated.ProjectSourceKind](query.Get("source_kind")),
		DriftStatus:       optionalTypedQuery[generated.ProjectDriftStatus](query.Get("drift_status")),
		LastRefreshStatus: optionalTypedQuery[generated.ProjectRefreshStatus](query.Get("last_refresh_status")),
	}
	var ok bool
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

func bindJSON[T any](ginCtx *gin.Context, ctx *module.Context, target *T) bool {
	if err := ginCtx.ShouldBindJSON(target); err != nil {
		httpx.AbortLocalizedError(ginCtx, ctx.I18n, http.StatusBadRequest, messagecontract.CommonInvalidArgument.String(), map[string]any{"field": "body"})
		return false
	}
	return true
}

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

func bindCommonParams(ginCtx *gin.Context) generated.PostProjectImportParams {
	locale, requestID := commonHeaders(ginCtx)
	return generated.PostProjectImportParams{XGraftLocale: locale, XRequestId: requestID}
}

func bindImportValidateParams(ginCtx *gin.Context) generated.PostProjectImportValidateParams {
	locale, requestID := commonHeaders(ginCtx)
	return generated.PostProjectImportValidateParams{XGraftLocale: locale, XRequestId: requestID}
}

func bindGetProjectParams(ginCtx *gin.Context) generated.GetProjectParams {
	locale, requestID := commonHeaders(ginCtx)
	return generated.GetProjectParams{XGraftLocale: locale, XRequestId: requestID}
}

func bindGetProjectServicesParams(ginCtx *gin.Context) generated.GetProjectServicesParams {
	locale, requestID := commonHeaders(ginCtx)
	return generated.GetProjectServicesParams{XGraftLocale: locale, XRequestId: requestID}
}

func bindGetProjectConfigurationParams(ginCtx *gin.Context) generated.GetProjectConfigurationParams {
	locale, requestID := commonHeaders(ginCtx)
	return generated.GetProjectConfigurationParams{XGraftLocale: locale, XRequestId: requestID}
}

func bindGetProjectConfigurationPreviewParams(ginCtx *gin.Context) generated.GetProjectConfigurationPreviewParams {
	locale, requestID := commonHeaders(ginCtx)
	return generated.GetProjectConfigurationPreviewParams{XGraftLocale: locale, XRequestId: requestID}
}

func bindGetProjectConfigurationFileParams(ginCtx *gin.Context) generated.GetProjectConfigurationFileParams {
	locale, requestID := commonHeaders(ginCtx)
	return generated.GetProjectConfigurationFileParams{XGraftLocale: locale, XRequestId: requestID}
}

func bindPostProjectConfigurationDiffParams(ginCtx *gin.Context) generated.PostProjectConfigurationDiffParams {
	locale, requestID := commonHeaders(ginCtx)
	return generated.PostProjectConfigurationDiffParams{XGraftLocale: locale, XRequestId: requestID}
}

func bindPostProjectConfigurationValidateParams(ginCtx *gin.Context) generated.PostProjectConfigurationValidateParams {
	locale, requestID := commonHeaders(ginCtx)
	return generated.PostProjectConfigurationValidateParams{XGraftLocale: locale, XRequestId: requestID}
}

func bindGetProjectManagedRootParams(ginCtx *gin.Context) generated.GetProjectManagedRootParams {
	locale, requestID := commonHeaders(ginCtx)
	return generated.GetProjectManagedRootParams{XGraftLocale: locale, XRequestId: requestID}
}

func bindPostProjectCreateValidateParams(ginCtx *gin.Context) generated.PostProjectCreateValidateParams {
	locale, requestID := commonHeaders(ginCtx)
	return generated.PostProjectCreateValidateParams{XGraftLocale: locale, XRequestId: requestID}
}

func bindPostProjectCreateParams(ginCtx *gin.Context) generated.PostProjectCreateParams {
	locale, requestID := commonHeaders(ginCtx)
	return generated.PostProjectCreateParams{XGraftLocale: locale, XRequestId: requestID}
}

func bindPostProjectRefreshParams(ginCtx *gin.Context) generated.PostProjectRefreshParams {
	locale, requestID := commonHeaders(ginCtx)
	return generated.PostProjectRefreshParams{XGraftLocale: locale, XRequestId: requestID}
}

func bindPostProjectDeployParams(ginCtx *gin.Context) generated.PostProjectDeployParams {
	locale, requestID := commonHeaders(ginCtx)
	return generated.PostProjectDeployParams{XGraftLocale: locale, XRequestId: requestID}
}

func bindPostProjectUpParams(ginCtx *gin.Context) generated.PostProjectUpParams {
	locale, requestID := commonHeaders(ginCtx)
	return generated.PostProjectUpParams{XGraftLocale: locale, XRequestId: requestID}
}

func bindPostProjectDownParams(ginCtx *gin.Context) generated.PostProjectDownParams {
	locale, requestID := commonHeaders(ginCtx)
	return generated.PostProjectDownParams{XGraftLocale: locale, XRequestId: requestID}
}

func bindPostProjectRestartParams(ginCtx *gin.Context) generated.PostProjectRestartParams {
	locale, requestID := commonHeaders(ginCtx)
	return generated.PostProjectRestartParams{XGraftLocale: locale, XRequestId: requestID}
}

func bindPostProjectUnregisterParams(ginCtx *gin.Context) generated.PostProjectUnregisterParams {
	locale, requestID := commonHeaders(ginCtx)
	return generated.PostProjectUnregisterParams{XGraftLocale: locale, XRequestId: requestID}
}

func bindPostProjectDestroyParams(ginCtx *gin.Context) generated.PostProjectDestroyParams {
	locale, requestID := commonHeaders(ginCtx)
	return generated.PostProjectDestroyParams{XGraftLocale: locale, XRequestId: requestID}
}

func commonHeaders(ginCtx *gin.Context) (*string, *string) {
	locale := ginCtx.GetHeader(string(httpheader.Locale))
	requestID := httpx.EnsureRequestID(ginCtx)
	return &locale, &requestID
}

func optionalTypedQuery[T ~string](raw string) *T {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil
	}
	value := T(trimmed)
	return &value
}

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

func abortInvalidQuery(ginCtx *gin.Context, ctx *module.Context) {
	httpx.AbortLocalizedError(ginCtx, ctx.I18n, http.StatusBadRequest, messagecontract.CommonInvalidArgument.String(), map[string]any{"field": "query"})
}

func intPtrValue[T ~int](value *T) int {
	if value == nil {
		return 0
	}
	return int(*value)
}

func stringPtrValue[T ~string](value *T) string {
	if value == nil {
		return ""
	}
	return string(*value)
}

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

func slicePtrValue(value *[]string) []string {
	if value == nil {
		return nil
	}
	return append([]string(nil), (*value)...)
}

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

func mapLifecycleErrorCode(err error) string {
	if errors.Is(err, errProjectManagedFlow) {
		return projectcontract.ProjectManagedFlowUnsupported.String()
	}
	return projectcontract.ProjectUnsupportedLifecycle.String()
}

func resolveAuthService(ctx *module.Context) (moduleapi.AuthService, error) {
	return module.ResolveService[moduleapi.AuthService](ctx.Services, (*moduleapi.AuthService)(nil))
}

func resolveAuthorizer(ctx *module.Context) (moduleapi.Authorizer, error) {
	return module.ResolveService[moduleapi.Authorizer](ctx.Services, (*moduleapi.Authorizer)(nil))
}
