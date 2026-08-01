package build

import (
	"errors"
	"math"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/gin-gonic/gin"

	openapigen "graft/server/internal/contract/openapi/generated"
	"graft/server/internal/httpx"
	"graft/server/internal/module"
	"graft/server/internal/moduleapi"
	buildcontract "graft/server/modules/build/contract"
)

const buildJobsRoute = "/jobs"

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
		if request.ApplicationId < 1 {
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
		receipt, submitErr := service.Submit(c.Request.Context(), SubmitRequest{ApplicationID: uint64(request.ApplicationId), ContextPath: request.ContextPath, DockerfilePath: request.DockerfilePath, ImageRepository: request.ImageRepository, ImageTag: request.ImageTag, BuildArgs: args, RequestedBy: requestedBy, IdempotencyKey: key})
		if submitErr != nil {
			status := http.StatusBadRequest
			if errors.Is(submitErr, moduleapi.ErrTaskSubmissionConflict) {
				status = http.StatusConflict
			}
			httpx.WriteLocalizedError(c, ctx.I18n, status, "common.invalidArgument", nil)
			return
		}
		if receipt.TaskID > math.MaxInt64 {
			httpx.WriteLocalizedError(c, ctx.I18n, http.StatusInternalServerError, "common.invalidArgument", nil)
			return
		}
		httpx.WriteSuccess(c, http.StatusAccepted, openapigen.TaskReceipt{TaskId: int64(receipt.TaskID), Status: openapigen.TaskStatus(receipt.Status)})
	})
	return nil
}
