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
	buildJobsRoute = "/jobs"
)

//nolint:gocognit,gocyclo,cyclop // 单一 HTTP 提交边界将身份、幂等键与生成契约的转换保持在一起。
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
	group := ctx.Router.Group("/api/build")
	group.Use(httpx.RequestIDMiddleware())
	group.GET(buildJobsRoute, httpx.RequirePermission(ctx.I18n, auth, authorizer, buildcontract.BuildReadPermission, publisher), func(c *gin.Context) {
		query, ok := buildListQuery(c)
		if !ok {
			httpx.WriteLocalizedError(c, ctx.I18n, http.StatusBadRequest, "common.invalidArgument", nil)
			return
		}
		result, err := service.ListJobs(c.Request.Context(), query)
		if err != nil {
			httpx.WriteLocalizedError(c, ctx.I18n, http.StatusInternalServerError, "common.internalError", nil)
			return
		}
		httpx.WriteSuccess(c, http.StatusOK, toBuildJobList(result, query))
	})
	group.GET(buildJobsRoute+"/:buildId", httpx.RequirePermission(ctx.I18n, auth, authorizer, buildcontract.BuildReadPermission, publisher), func(c *gin.Context) {
		job, err := service.GetJob(c.Request.Context(), c.Param("buildId"))
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
		if strings.TrimSpace(request.ApplicationId) == "" {
			httpx.WriteLocalizedError(c, ctx.I18n, http.StatusBadRequest, "common.invalidArgument", nil)
			return
		}
		requestedBy := uint64(0)
		if requestAuth, ok := moduleapi.RequestAuthContextFromContext(c.Request.Context()); ok && requestAuth.User != nil {
			requestedBy = requestAuth.User.ID
		}
		args := make([]moduleapi.DockerImageBuildArg, 0)
		if request.BuildArgs != nil {
			args = make([]moduleapi.DockerImageBuildArg, 0, len(*request.BuildArgs))
			for _, item := range *request.BuildArgs {
				args = append(args, moduleapi.DockerImageBuildArg{Name: item.Name, Value: item.Value})
			}
		}
		receipt, submitErr := service.Submit(c.Request.Context(), SubmitRequest{ApplicationID: request.ApplicationId, ContextPath: request.ContextPath, DockerfilePath: request.DockerfilePath, ImageRepository: request.ImageRepository, ImageTag: request.ImageTag, BuildArgs: args, RequestedBy: requestedBy, IdempotencyKey: key})
		if submitErr != nil {
			status, key := http.StatusInternalServerError, "common.internalError"
			if errors.Is(submitErr, moduleapi.ErrTaskSubmissionConflict) {
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
	return nil
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
	applicationID, ok := buildApplicationIDQuery(c)
	if !ok {
		return false
	}
	query.ApplicationID = applicationID
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

func buildApplicationIDQuery(c *gin.Context) (*string, bool) {
	raw, present := c.GetQuery("application_id")
	if !present {
		return nil, true
	}
	value := strings.TrimSpace(raw)
	if value == "" {
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
	if value == "" || len(value) > 255 {
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
