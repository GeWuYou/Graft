package build

import (
	"context"
	"errors"
	buildopenapi "graft/server/internal/contract/openapi/build"
	"graft/server/internal/contract/openapi/strictsupport"
	"math"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gin-gonic/gin"

	messagecontract "graft/server/internal/contract/message"
	openapigen "graft/server/internal/contract/openapi/generated"
	"graft/server/internal/httpx"
	"graft/server/internal/module"
	"graft/server/internal/moduleapi"
	buildcontract "graft/server/modules/build/contract"
	buildstore "graft/server/modules/build/store"
)

const (
	buildJobsRoute                       = "/jobs"
	inputSnapshotMultipartOverhead int64 = 1 << 20
)

//nolint:gocognit,gocyclo,cyclop,funlen,maintidx // 单一 HTTP 边界将鉴权、查询和提交契约转换保持在一起。
func registerRoutes(ctx *module.Context, service *Service) error {
	if ctx == nil || ctx.Router == nil {
		return nil
	}
	if service == nil {
		return errors.New("build service is unavailable")
	}
	auth, err := module.ResolveService[moduleapi.AuthService](ctx.Services, (*moduleapi.AuthService)(nil))
	if err != nil {
		return err
	}
	authorizer, err := module.ResolveService[moduleapi.Authorizer](ctx.Services, (*moduleapi.Authorizer)(nil))
	if err != nil {
		return err
	}
	publisher := httpx.NewSecurityAuditPublisher(ctx.EventBus, ctx.Logger, moduleID)
	group := ctx.Router.Group("/build")
	group.Use(httpx.RequestIDMiddleware())
	uploadSnapshot := func(c *gin.Context, reader *multipart.Reader) {
		form, err := reader.ReadForm(inputSnapshotMultipartOverhead)
		if err != nil {
			httpx.WriteLocalizedError(c, ctx.I18n, http.StatusBadRequest, "common.invalidArgument", nil)
			return
		}
		defer func() { _ = form.RemoveAll() }()
		files := form.File["archive"]
		if len(files) == 0 || files[0].Size == 0 {
			httpx.WriteLocalizedError(c, ctx.I18n, http.StatusBadRequest, "common.invalidArgument", nil)
			return
		}
		var request openapigen.PostBuildInputSnapshotMultipartRequestBody
		request.Archive.InitFromMultipart(files[0])
		file, err := request.Archive.Reader()
		if err != nil {
			httpx.WriteLocalizedError(c, ctx.I18n, http.StatusBadRequest, "common.invalidArgument", nil)
			return
		}
		defer func() { _ = file.Close() }()
		snapshot, createErr := service.CreateInputSnapshot(c.Request.Context(), InputSnapshotUpload{Archive: file, Size: request.Archive.FileSize(), UserID: requestUserID(c)})
		if createErr != nil {
			status, key := http.StatusInternalServerError, "common.internalError"
			if errors.Is(createErr, errInvalidBuildRequest) || errors.Is(createErr, errInvalidInputSnapshotUpload) {
				status, key = http.StatusBadRequest, "common.invalidArgument"
			}
			httpx.WriteLocalizedError(c, ctx.I18n, status, key, nil)
			return
		}
		httpx.WriteSuccess(c, http.StatusCreated, openapigen.BuildInputSnapshot{SnapshotId: snapshot.ID, SourceKind: openapigen.BuildInputSnapshotSourceKind(snapshot.SourceKind), ContentDigest: snapshot.ContentDigest, LifecycleState: openapigen.BuildInputSnapshotLifecycleState("available")})
	}
	listSnapshots := func(c *gin.Context) {
		query, ok := buildPaginationQuery(c)
		if !ok {
			httpx.WriteLocalizedError(c, ctx.I18n, http.StatusBadRequest, "common.invalidArgument", nil)
			return
		}
		result, listErr := service.ListInputSnapshots(c.Request.Context(), requestUserID(c), query.Limit, query.Offset)
		if listErr != nil {
			httpx.WriteLocalizedError(c, ctx.I18n, http.StatusInternalServerError, "common.internalError", nil)
			return
		}
		items := make([]openapigen.BuildInputSnapshot, 0, len(result.Items))
		for _, snapshot := range result.Items {
			items = append(items, openapigen.BuildInputSnapshot{SnapshotId: snapshot.ID, SourceKind: openapigen.BuildInputSnapshotSourceKind(snapshot.SourceKind), ContentDigest: snapshot.ContentDigest, LifecycleState: openapigen.BuildInputSnapshotLifecycleState("available")})
		}
		httpx.WriteSuccess(c, http.StatusOK, openapigen.BuildInputSnapshotList{Items: items, Total: result.Total, Limit: query.Limit, Offset: query.Offset})
	}
	listWorkspaces := func(c *gin.Context) {
		pagination, ok := buildPaginationQuery(c)
		if !ok {
			httpx.WriteLocalizedError(c, ctx.I18n, http.StatusBadRequest, "common.invalidArgument", nil)
			return
		}
		search, ok := buildExactStringQuery(c, "search")
		if !ok {
			httpx.WriteLocalizedError(c, ctx.I18n, http.StatusBadRequest, "common.invalidArgument", nil)
			return
		}
		result, listErr := service.ListWorkspaces(c.Request.Context(), requestUserID(c), buildstore.WorkspaceListQuery{Limit: pagination.Limit, Offset: pagination.Offset, Search: search})
		if listErr != nil {
			httpx.WriteLocalizedError(c, ctx.I18n, http.StatusInternalServerError, "common.internalError", nil)
			return
		}
		httpx.WriteSuccess(c, http.StatusOK, toBuildWorkspaceList(result, buildstore.WorkspaceListQuery{Limit: pagination.Limit, Offset: pagination.Offset}))
	}
	listTargets := func(c *gin.Context) {
		items, listErr := service.ListBuildTargets(c.Request.Context(), requestUserID(c))
		if listErr != nil {
			httpx.WriteLocalizedError(c, ctx.I18n, http.StatusInternalServerError, "common.internalError", nil)
			return
		}
		httpx.WriteSuccess(c, http.StatusOK, openapigen.BuildRuntimeTargetList{Items: mapBuildRuntimeTargets(items)})
	}
	listPools := func(c *gin.Context) {
		items, listErr := service.ListBuilderPools(c.Request.Context(), requestUserID(c))
		if listErr != nil {
			httpx.WriteLocalizedError(c, ctx.I18n, http.StatusInternalServerError, "common.internalError", nil)
			return
		}
		httpx.WriteSuccess(c, http.StatusOK, openapigen.BuildBuilderPoolList{Items: mapBuilderPools(items)})
	}
	listArtifacts := func(c *gin.Context) {
		query, ok := buildPaginationQuery(c)
		if !ok {
			httpx.WriteLocalizedError(c, ctx.I18n, http.StatusBadRequest, "common.invalidArgument", nil)
			return
		}
		result, listErr := service.ListArtifacts(c.Request.Context(), query.Limit, query.Offset)
		if listErr != nil {
			httpx.WriteLocalizedError(c, ctx.I18n, http.StatusInternalServerError, "common.internalError", nil)
			return
		}
		httpx.WriteSuccess(c, http.StatusOK, toBuildArtifactList(result, query.Limit, query.Offset))
	}
	listPublications := func(c *gin.Context) {
		artifactID := strings.TrimSpace(c.Param("artifactId"))
		if artifactID == "" || utf8.RuneCountInString(artifactID) > 64 {
			httpx.WriteLocalizedError(c, ctx.I18n, http.StatusBadRequest, "common.invalidArgument", nil)
			return
		}
		query, ok := buildPaginationQuery(c)
		if !ok {
			httpx.WriteLocalizedError(c, ctx.I18n, http.StatusBadRequest, "common.invalidArgument", nil)
			return
		}
		result, listErr := service.ListArtifactPublications(c.Request.Context(), artifactID, query.Limit, query.Offset)
		if errors.Is(listErr, buildstore.ErrNotFound) {
			httpx.WriteLocalizedError(c, ctx.I18n, http.StatusNotFound, "common.notFound", nil)
			return
		}
		if listErr != nil {
			httpx.WriteLocalizedError(c, ctx.I18n, http.StatusInternalServerError, "common.internalError", nil)
			return
		}
		httpx.WriteSuccess(c, http.StatusOK, toBuildArtifactPublicationList(result, query.Limit, query.Offset))
	}
	listJobs := func(c *gin.Context) {
		query, ok := buildListQuery(c)
		if !ok {
			httpx.WriteLocalizedError(c, ctx.I18n, http.StatusBadRequest, "common.invalidArgument", nil)
			return
		}
		result, err := service.ListJobs(c.Request.Context(), requestUserID(c), query)
		if err != nil {
			httpx.WriteLocalizedError(c, ctx.I18n, http.StatusInternalServerError, "common.internalError", nil)
			return
		}
		httpx.WriteSuccess(c, http.StatusOK, toBuildJobList(result, query))
	}
	getJob := func(c *gin.Context) {
		job, err := service.GetJob(c.Request.Context(), requestUserID(c), c.Param("buildId"))
		if errors.Is(err, buildstore.ErrNotFound) {
			httpx.WriteLocalizedError(c, ctx.I18n, http.StatusNotFound, "common.notFound", nil)
			return
		}
		if errors.Is(err, errInvalidBuildID) {
			httpx.WriteLocalizedError(c, ctx.I18n, http.StatusBadRequest, "common.invalidArgument", nil)
			return
		}
		if err != nil {
			httpx.WriteLocalizedError(c, ctx.I18n, http.StatusInternalServerError, "common.internalError", nil)
			return
		}
		httpx.WriteSuccess(c, http.StatusOK, toBuildJobDetail(job))
	}
	createJob := func(c *gin.Context, request *buildopenapi.PostBuildJobJSONRequestBody) {
		if request == nil {
			httpx.WriteLocalizedError(c, ctx.I18n, http.StatusBadRequest, "common.invalidArgument", nil)
			return
		}
		key := c.GetHeader("Idempotency-Key")
		if strings.TrimSpace(key) == "" || utf8.RuneCountInString(key) > moduleapi.TaskIdempotencyKeyMaxRunes {
			httpx.WriteLocalizedError(c, ctx.I18n, http.StatusBadRequest, "common.invalidArgument", nil)
			return
		}
		inputSnapshotID := ""
		if request.InputSnapshotId != nil {
			inputSnapshotID = strings.TrimSpace(*request.InputSnapshotId)
		}
		workspaceID := ""
		if request.WorkspaceId != nil {
			workspaceID = strings.TrimSpace(*request.WorkspaceId)
		}
		if inputSnapshotID == "" && workspaceID == "" {
			httpx.WriteLocalizedError(c, ctx.I18n, http.StatusBadRequest, "common.invalidArgument", nil)
			return
		}
		requestedBy := uint64(0)
		if requestAuth, ok := moduleapi.RequestAuthContextFromContext(c.Request.Context()); ok && requestAuth.User != nil {
			requestedBy = requestAuth.User.ID
		}
		platforms := []string(nil)
		if request.Platforms != nil {
			platforms = append(platforms, *request.Platforms...)
		}
		runtimeTargetID := int64(0)
		if request.RuntimeTargetId != nil {
			runtimeTargetID = *request.RuntimeTargetId
		}
		builderPoolID := ""
		if request.BuilderPoolId != nil {
			builderPoolID = *request.BuilderPoolId
		}
		receipt, submitErr := service.SubmitExecutionPlan(c.Request.Context(), ExecutionPlanRequest{InputSnapshotID: inputSnapshotID, WorkspaceID: workspaceID, BuilderPoolID: builderPoolID, RuntimeTargetID: runtimeTargetID, TemplateRef: string(request.TemplateRef), Driver: string(request.Driver), Platforms: platforms, Destination: moduleapi.BuildDestination{Kind: string(request.Destination.Kind), ConnectionRef: request.Destination.ConnectionRef, RepositoryRef: request.Destination.RepositoryRef, Reference: request.Destination.Reference}, RequestedBy: requestedBy, IdempotencyKey: key})
		if submitErr != nil {
			status, key := buildSubmissionFailure(submitErr)
			httpx.WriteLocalizedError(c, ctx.I18n, status, key, nil)
			return
		}
		if receipt.TaskID > math.MaxInt64 {
			httpx.WriteLocalizedError(c, ctx.I18n, http.StatusInternalServerError, "common.internalError", nil)
			return
		}
		httpx.WriteSuccess(c, http.StatusAccepted, openapigen.TaskReceipt{TaskId: int64(receipt.TaskID), Status: openapigen.TaskStatus(receipt.Status)})
	}
	promoteArtifact := func(c *gin.Context, request *buildopenapi.PostBuildArtifactPromotionJSONRequestBody) {
		if request == nil || !validArtifactPromotionRequest(*request) {
			httpx.WriteLocalizedError(c, ctx.I18n, http.StatusBadRequest, "common.invalidArgument", nil)
			return
		}
		key := c.GetHeader("Idempotency-Key")
		if strings.TrimSpace(key) == "" || utf8.RuneCountInString(key) > moduleapi.TaskIdempotencyKeyMaxRunes {
			httpx.WriteLocalizedError(c, ctx.I18n, http.StatusBadRequest, "common.invalidArgument", nil)
			return
		}
		receipt, submitErr := service.SubmitArtifactPromotion(c.Request.Context(), ArtifactPromotionRequest{
			ArtifactID:      request.ArtifactId,
			PublicationID:   request.PublicationId,
			RuntimeTargetID: request.RuntimeTargetId,
			Destination: moduleapi.BuildDestination{
				Kind:          string(request.Destination.Kind),
				ConnectionRef: request.Destination.ConnectionRef,
				RepositoryRef: request.Destination.RepositoryRef,
				Reference:     request.Destination.Reference,
			},
			RequestedBy:    requestUserID(c),
			IdempotencyKey: key,
		})
		if submitErr != nil {
			status, key := buildSubmissionFailure(submitErr)
			httpx.WriteLocalizedError(c, ctx.I18n, status, key, nil)
			return
		}
		if receipt.TaskID > math.MaxInt64 {
			httpx.WriteLocalizedError(c, ctx.I18n, http.StatusInternalServerError, "common.internalError", nil)
			return
		}
		httpx.WriteSuccess(c, http.StatusAccepted, openapigen.TaskReceipt{TaskId: int64(receipt.TaskID), Status: openapigen.TaskStatus(receipt.Status)})
	}
	wrapper := buildopenapi.ServerInterfaceWrapper{
		Handler: buildopenapi.NewStrictHandlerWithOptions(buildStrictServer{uploadSnapshot: uploadSnapshot, listSnapshots: listSnapshots, listWorkspaces: listWorkspaces, listTargets: listTargets, listPools: listPools, listArtifacts: listArtifacts, listPublications: listPublications, listJobs: listJobs, getJob: getJob, createJob: createJob, promoteArtifact: promoteArtifact}, nil, buildopenapi.StrictGinServerOptions{
			RequestErrorHandlerFunc: strictsupport.RequestErrorHandler(ctx.I18n, "common.invalidArgument", nil),
			HandlerErrorFunc:        strictsupport.HandlerErrorHandler(ctx.I18n, ctx.Logger),
		}),
		ErrorHandler: strictsupport.BindingErrorHandler(ctx.I18n, "common.invalidArgument"),
	}
	group.POST("/input-snapshots", httpx.RequirePermission(ctx.I18n, auth, authorizer, buildcontract.BuildCreatePermission, publisher), limitBuildSnapshotBody, strictsupport.Route(wrapper.PostBuildInputSnapshot, nil))
	group.GET("/input-snapshots", httpx.RequirePermission(ctx.I18n, auth, authorizer, buildcontract.BuildReadPermission, publisher), strictsupport.Route(wrapper.GetBuildInputSnapshots, listSnapshots))
	group.GET("/workspaces", httpx.RequirePermission(ctx.I18n, auth, authorizer, buildcontract.BuildReadPermission, publisher), strictsupport.Route(wrapper.GetBuildWorkspaces, listWorkspaces))
	group.GET("/runtime-targets", httpx.RequirePermission(ctx.I18n, auth, authorizer, buildcontract.BuildReadPermission, publisher), strictsupport.Route(wrapper.GetBuildRuntimeTargets, nil))
	group.GET("/builder-pools", httpx.RequirePermission(ctx.I18n, auth, authorizer, buildcontract.BuildReadPermission, publisher), strictsupport.Route(wrapper.GetBuildBuilderPools, nil))
	group.GET("/artifacts", httpx.RequirePermission(ctx.I18n, auth, authorizer, buildcontract.BuildReadPermission, publisher), strictsupport.Route(wrapper.GetBuildArtifacts, listArtifacts))
	group.GET("/artifacts/:artifactId/publications", httpx.RequirePermission(ctx.I18n, auth, authorizer, buildcontract.BuildReadPermission, publisher), strictsupport.Route(wrapper.GetBuildArtifactPublications, listPublications))
	group.GET(buildJobsRoute, httpx.RequirePermission(ctx.I18n, auth, authorizer, buildcontract.BuildReadPermission, publisher), strictsupport.Route(wrapper.GetBuildJobs, listJobs))
	group.GET(buildJobsRoute+"/:buildId", httpx.RequirePermission(ctx.I18n, auth, authorizer, buildcontract.BuildReadPermission, publisher), strictsupport.Route(wrapper.GetBuildJob, getJob))
	group.POST(buildJobsRoute, httpx.RequirePermission(ctx.I18n, auth, authorizer, buildcontract.BuildCreatePermission, publisher), preserveBuildIdempotencyKey, strictsupport.Route(wrapper.PostBuildJob, nil))
	group.POST("/artifact-promotions", httpx.RequirePermission(ctx.I18n, auth, authorizer, buildcontract.BuildCreatePermission, publisher), preserveBuildIdempotencyKey, strictsupport.Route(wrapper.PostBuildArtifactPromotion, nil))

	return nil
}

func validArtifactPromotionRequest(request buildopenapi.PostBuildArtifactPromotionJSONRequestBody) bool {
	return strings.TrimSpace(request.ArtifactId) != "" &&
		strings.TrimSpace(request.PublicationId) != "" &&
		request.RuntimeTargetId > 0 &&
		request.Destination.Kind == "oci_registry" &&
		strings.TrimSpace(request.Destination.ConnectionRef) != "" &&
		strings.TrimSpace(request.Destination.RepositoryRef) != "" &&
		strings.TrimSpace(request.Destination.Reference) != ""
}

func requestUserID(c *gin.Context) uint64 {
	if requestAuth, ok := moduleapi.RequestAuthContextFromContext(c.Request.Context()); ok && requestAuth.User != nil {
		return requestAuth.User.ID
	}
	return 0
}

func buildListQuery(c *gin.Context) (buildstore.ListQuery, bool) {
	query, ok := buildPaginationQuery(c)
	if !ok {
		return buildstore.ListQuery{}, false
	}
	if !bindBuildHistoryFilters(c, &query) {
		return buildstore.ListQuery{}, false
	}
	if query.CreatedAfter != nil && query.CreatedBefore != nil && query.CreatedAfter.After(*query.CreatedBefore) {
		return buildstore.ListQuery{}, false
	}
	return query, true
}

func buildPaginationQuery(c *gin.Context) (buildstore.ListQuery, bool) {
	query := buildstore.ListQuery{Limit: buildstore.DefaultListLimit}
	if raw := c.Query("limit"); raw != "" {
		limit, err := strconv.Atoi(raw)
		if err != nil || limit < 1 || limit > buildstore.MaxListLimit {
			return buildstore.ListQuery{}, false
		}
		query.Limit = limit
	}
	if raw := c.Query("offset"); raw != "" {
		offset, err := strconv.Atoi(raw)
		if err != nil || offset < 0 {
			return buildstore.ListQuery{}, false
		}
		query.Offset = offset
	}
	return query, true
}

func bindBuildHistoryFilters(c *gin.Context, query *buildstore.ListQuery) bool {
	search, ok := buildExactStringQuery(c, "search")
	if !ok {
		return false
	}
	query.Search = search
	imageRepository, ok := buildExactStringQuery(c, "image_repository")
	if !ok {
		return false
	}
	query.ImageRepository = imageRepository
	imageTag, ok := buildExactStringQuery(c, "image_tag")
	if !ok {
		return false
	}
	query.ImageTag = imageTag
	status, ok := buildStatusQuery(c)
	if !ok {
		return false
	}
	query.BuildStatus = status
	builderID, ok := buildUint64Query(c, "builder_id")
	if !ok {
		return false
	}
	query.BuilderID = builderID
	createdAfter, ok := buildTimeQuery(c, "created_after")
	if !ok {
		return false
	}
	query.CreatedAfter = createdAfter
	createdBefore, ok := buildTimeQuery(c, "created_before")
	if !ok {
		return false
	}
	query.CreatedBefore = createdBefore
	return true
}

func buildStatusQuery(c *gin.Context) (*buildstore.StatusFilter, bool) {
	raw, present := c.GetQuery("build_status")
	if !present {
		return nil, true
	}
	status := buildstore.StatusFilter(strings.TrimSpace(raw))
	switch status {
	case buildstore.StatusFilterQueued, buildstore.StatusFilterRunning, buildstore.StatusFilterSuccess, buildstore.StatusFilterFailed, buildstore.StatusFilterCancelled:
		return &status, true
	default:
		return nil, false
	}
}

func buildUint64Query(c *gin.Context, key string) (*uint64, bool) {
	raw, present := c.GetQuery(key)
	if !present {
		return nil, true
	}
	value, err := strconv.ParseUint(strings.TrimSpace(raw), 10, 64)
	if err != nil || value == 0 {
		return nil, false
	}
	return &value, true
}

func buildExactStringQuery(c *gin.Context, key string) (*string, bool) {
	raw, present := c.GetQuery(key)
	if !present {
		return nil, true
	}
	value := strings.TrimSpace(raw)
	if value == "" || utf8.RuneCountInString(value) > 255 {
		return nil, false
	}
	return &value, true
}

func buildTimeQuery(c *gin.Context, key string) (*time.Time, bool) {
	raw, present := c.GetQuery(key)
	if !present {
		return nil, true
	}
	value, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return nil, false
	}
	return &value, true
}

func buildSubmissionFailure(err error) (int, string) {
	switch {
	case errors.Is(err, moduleapi.ErrTaskOwnerBusy):
		return http.StatusConflict, messagecontract.CommonTaskResourceBusy.String()
	case errors.Is(err, moduleapi.ErrTaskExecutionUnavailable):
		return http.StatusServiceUnavailable, messagecontract.CommonTaskExecutionUnavailable.String()
	case errors.Is(err, moduleapi.ErrTaskSubmissionConflict), errors.Is(err, buildstore.ErrConflict):
		return http.StatusConflict, "common.invalidArgument"
	case errors.Is(err, errInvalidBuildRequest):
		return http.StatusBadRequest, "common.invalidArgument"
	default:
		return http.StatusInternalServerError, "common.internalError"
	}
}

// buildStrictServer 将现有 HTTP 响应写入接入生成的请求契约。
type buildStrictServer struct {
	uploadSnapshot   func(*gin.Context, *multipart.Reader)
	listSnapshots    gin.HandlerFunc
	listWorkspaces   gin.HandlerFunc
	listTargets      gin.HandlerFunc
	listPools        gin.HandlerFunc
	listArtifacts    gin.HandlerFunc
	listPublications gin.HandlerFunc
	listJobs         gin.HandlerFunc
	getJob           gin.HandlerFunc
	createJob        func(*gin.Context, *buildopenapi.PostBuildJobJSONRequestBody)
	promoteArtifact  func(*gin.Context, *buildopenapi.PostBuildArtifactPromotionJSONRequestBody)
}

var _ buildopenapi.StrictServerInterface = buildStrictServer{}

func (s buildStrictServer) PostBuildInputSnapshot(ctx context.Context, req buildopenapi.PostBuildInputSnapshotRequestObject) (buildopenapi.PostBuildInputSnapshotResponseObject, error) {
	if req.Body == nil {
		return nil, strictsupport.ErrMissingRequestBody
	}
	return nil, strictsupport.Dispatch(ctx, func(c *gin.Context) { s.uploadSnapshot(c, req.Body) })
}

func limitBuildSnapshotBody(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxInputSnapshotUploadBytes+inputSnapshotMultipartOverhead)
	c.Next()
}

func (s buildStrictServer) GetBuildInputSnapshots(ctx context.Context, _ buildopenapi.GetBuildInputSnapshotsRequestObject) (buildopenapi.GetBuildInputSnapshotsResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.listSnapshots)
}

func (s buildStrictServer) GetBuildWorkspaces(ctx context.Context, _ buildopenapi.GetBuildWorkspacesRequestObject) (buildopenapi.GetBuildWorkspacesResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.listWorkspaces)
}

func (s buildStrictServer) GetBuildRuntimeTargets(ctx context.Context, _ buildopenapi.GetBuildRuntimeTargetsRequestObject) (buildopenapi.GetBuildRuntimeTargetsResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.listTargets)
}

func (s buildStrictServer) GetBuildBuilderPools(ctx context.Context, _ buildopenapi.GetBuildBuilderPoolsRequestObject) (buildopenapi.GetBuildBuilderPoolsResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.listPools)
}

func (s buildStrictServer) GetBuildArtifacts(ctx context.Context, _ buildopenapi.GetBuildArtifactsRequestObject) (buildopenapi.GetBuildArtifactsResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.listArtifacts)
}

func (s buildStrictServer) GetBuildArtifactPublications(ctx context.Context, _ buildopenapi.GetBuildArtifactPublicationsRequestObject) (buildopenapi.GetBuildArtifactPublicationsResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.listPublications)
}

func (s buildStrictServer) GetBuildJobs(ctx context.Context, _ buildopenapi.GetBuildJobsRequestObject) (buildopenapi.GetBuildJobsResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.listJobs)
}

func (s buildStrictServer) GetBuildJob(ctx context.Context, _ buildopenapi.GetBuildJobRequestObject) (buildopenapi.GetBuildJobResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.getJob)
}

func (s buildStrictServer) PostBuildJob(ctx context.Context, req buildopenapi.PostBuildJobRequestObject) (buildopenapi.PostBuildJobResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, func(c *gin.Context) { s.createJob(c, req.Body) })
}

func (s buildStrictServer) PostBuildArtifactPromotion(ctx context.Context, req buildopenapi.PostBuildArtifactPromotionRequestObject) (buildopenapi.PostBuildArtifactPromotionResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, func(c *gin.Context) { s.promoteArtifact(c, req.Body) })
}

// preserveBuildIdempotencyKey 保持任务提交端点采用首个幂等请求头的旧行为。
func preserveBuildIdempotencyKey(c *gin.Context) {
	const name = "Idempotency-Key"
	if values := c.Request.Header.Values(name); len(values) > 1 {
		c.Request.Header.Set(name, values[0])
	}
	c.Next()
}
