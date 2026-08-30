package build

import (
	"errors"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gin-gonic/gin"

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
	group.POST("/input-snapshots", httpx.RequirePermission(ctx.I18n, auth, authorizer, buildcontract.BuildCreatePermission, publisher), func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxInputSnapshotUploadBytes+inputSnapshotMultipartOverhead)
		var request openapigen.PostBuildInputSnapshotMultipartRequestBody
		if err := c.Request.ParseMultipartForm(inputSnapshotMultipartOverhead); err != nil {
			httpx.WriteLocalizedError(c, ctx.I18n, http.StatusBadRequest, "common.invalidArgument", nil)
			return
		}
		defer func() {
			if c.Request.MultipartForm != nil {
				_ = c.Request.MultipartForm.RemoveAll()
			}
		}()
		_, header, err := c.Request.FormFile("archive")
		if err != nil || header == nil || header.Size == 0 {
			httpx.WriteLocalizedError(c, ctx.I18n, http.StatusBadRequest, "common.invalidArgument", nil)
			return
		}
		request.Archive.InitFromMultipart(header)
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
		httpx.WriteSuccess(c, http.StatusCreated, openapigen.EnvelopedBuildInputSnapshot{Data: openapigen.BuildInputSnapshot{SnapshotId: snapshot.ID, SourceKind: openapigen.BuildInputSnapshotSourceKind(snapshot.SourceKind), ContentDigest: snapshot.ContentDigest, LifecycleState: openapigen.BuildInputSnapshotLifecycleState("available")}})
	})
	group.GET("/input-snapshots", httpx.RequirePermission(ctx.I18n, auth, authorizer, buildcontract.BuildReadPermission, publisher), func(c *gin.Context) {
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
		httpx.WriteSuccess(c, http.StatusOK, openapigen.EnvelopedBuildInputSnapshotList{Data: openapigen.BuildInputSnapshotList{Items: items, Total: result.Total, Limit: query.Limit, Offset: query.Offset}})
	})
	group.GET("/workspaces", httpx.RequirePermission(ctx.I18n, auth, authorizer, buildcontract.BuildReadPermission, publisher), func(c *gin.Context) {
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
		httpx.WriteSuccess(c, http.StatusOK, openapigen.EnvelopedBuildWorkspaceList{Data: toBuildWorkspaceList(result, buildstore.WorkspaceListQuery{Limit: pagination.Limit, Offset: pagination.Offset})})
	})
	group.GET("/runtime-targets", httpx.RequirePermission(ctx.I18n, auth, authorizer, buildcontract.BuildReadPermission, publisher), func(c *gin.Context) {
		items, listErr := service.ListBuildTargets(c.Request.Context(), requestUserID(c))
		if listErr != nil {
			httpx.WriteLocalizedError(c, ctx.I18n, http.StatusInternalServerError, "common.internalError", nil)
			return
		}
		httpx.WriteSuccess(c, http.StatusOK, openapigen.BuildRuntimeTargetList{Items: mapBuildRuntimeTargets(items)})
	})
	group.GET("/builder-pools", httpx.RequirePermission(ctx.I18n, auth, authorizer, buildcontract.BuildReadPermission, publisher), func(c *gin.Context) {
		items, listErr := service.ListBuilderPools(c.Request.Context(), requestUserID(c))
		if listErr != nil {
			httpx.WriteLocalizedError(c, ctx.I18n, http.StatusInternalServerError, "common.internalError", nil)
			return
		}
		httpx.WriteSuccess(c, http.StatusOK, openapigen.BuildBuilderPoolList{Items: mapBuilderPools(items)})
	})
	group.GET("/artifacts", httpx.RequirePermission(ctx.I18n, auth, authorizer, buildcontract.BuildReadPermission, publisher), func(c *gin.Context) {
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
	})
	group.GET("/artifacts/:artifactId/publications", httpx.RequirePermission(ctx.I18n, auth, authorizer, buildcontract.BuildReadPermission, publisher), func(c *gin.Context) {
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
		httpx.WriteSuccess(c, http.StatusOK, openapigen.EnvelopedBuildArtifactPublicationList{Data: toBuildArtifactPublicationList(result, query.Limit, query.Offset)})
	})
	group.GET(buildJobsRoute, httpx.RequirePermission(ctx.I18n, auth, authorizer, buildcontract.BuildReadPermission, publisher), func(c *gin.Context) {
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
	})
	group.GET(buildJobsRoute+"/:buildId", httpx.RequirePermission(ctx.I18n, auth, authorizer, buildcontract.BuildReadPermission, publisher), func(c *gin.Context) {
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
	})
	group.POST(buildJobsRoute, httpx.RequirePermission(ctx.I18n, auth, authorizer, buildcontract.BuildCreatePermission, publisher), func(c *gin.Context) {
		var request openapigen.PostBuildJobJSONRequestBody
		if err := c.ShouldBindJSON(&request); err != nil {
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
			status, key := http.StatusInternalServerError, "common.internalError"
			if errors.Is(submitErr, moduleapi.ErrTaskSubmissionConflict) {
				status, key = http.StatusConflict, "common.invalidArgument"
			} else if errors.Is(submitErr, buildstore.ErrConflict) {
				status, key = http.StatusConflict, "common.invalidArgument"
			} else if errors.Is(submitErr, errInvalidBuildRequest) {
				status, key = http.StatusBadRequest, "common.invalidArgument"
			}
			httpx.WriteLocalizedError(c, ctx.I18n, status, key, nil)
			return
		}
		if receipt.TaskID > math.MaxInt64 {
			httpx.WriteLocalizedError(c, ctx.I18n, http.StatusInternalServerError, "common.internalError", nil)
			return
		}
		httpx.WriteSuccess(c, http.StatusAccepted, openapigen.TaskReceipt{TaskId: int64(receipt.TaskID), Status: openapigen.TaskStatus(receipt.Status)})
	})
	group.POST("/artifact-promotions", httpx.RequirePermission(ctx.I18n, auth, authorizer, buildcontract.BuildCreatePermission, publisher), func(c *gin.Context) {
		var request openapigen.PostBuildArtifactPromotionJSONRequestBody
		if err := c.ShouldBindJSON(&request); err != nil || !validArtifactPromotionRequest(request) {
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
			status, errorKey := http.StatusInternalServerError, "common.internalError"
			if errors.Is(submitErr, moduleapi.ErrTaskSubmissionConflict) || errors.Is(submitErr, buildstore.ErrConflict) {
				status, errorKey = http.StatusConflict, "common.invalidArgument"
			}
			httpx.WriteLocalizedError(c, ctx.I18n, status, errorKey, nil)
			return
		}
		if receipt.TaskID > math.MaxInt64 {
			httpx.WriteLocalizedError(c, ctx.I18n, http.StatusInternalServerError, "common.internalError", nil)
			return
		}
		httpx.WriteSuccess(c, http.StatusAccepted, openapigen.TaskReceipt{TaskId: int64(receipt.TaskID), Status: openapigen.TaskStatus(receipt.Status)})
	})
	return nil
}

func validArtifactPromotionRequest(request openapigen.PostBuildArtifactPromotionJSONRequestBody) bool {
	return strings.TrimSpace(request.ArtifactId) != "" &&
		strings.TrimSpace(request.PublicationId) != "" &&
		request.RuntimeTargetId > 0 &&
		request.Destination.Kind == openapigen.BuildArtifactPromotionCreateRequestDestinationKind("oci_registry") &&
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
