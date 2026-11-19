package backup

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	messagecontract "graft/server/internal/contract/message"
	backupopenapi "graft/server/internal/contract/openapi/backup"
	"graft/server/internal/contract/openapi/strictsupport"
	"graft/server/internal/httpx"
	"graft/server/internal/i18n"
	"graft/server/internal/logger/logsafe"
	"graft/server/internal/module"
	"graft/server/internal/moduleapi"
	backupcontract "graft/server/modules/backup/contract"
)

const (
	defaultBackupListLimit = 20
	maxBackupListLimit     = 100
)

// registerRoutes 安装备份提交和安全历史读取面；工件路径及内容始终留在模块私有边界。
func registerRoutes(ctx *module.Context, service *Service) error {
	if ctx == nil || ctx.Router == nil {
		return nil
	}
	auth, err := module.ResolveService[moduleapi.AuthService](ctx.Services, (*moduleapi.AuthService)(nil))
	if err != nil {
		return err
	}
	authorizer, err := module.ResolveService[moduleapi.Authorizer](ctx.Services, (*moduleapi.Authorizer)(nil))
	if err != nil {
		return err
	}
	group := ctx.Router.Group(backupcontract.BackupGroup)
	group.Use(httpx.RequestIDMiddleware())
	publisher := httpx.NewSecurityAuditPublisher(ctx.EventBus, ctx.Logger, moduleID)
	handlers := backupRouteHandlers{service: service, auth: auth, localizer: ctx.I18n, logger: ctx.Logger}
	// 生成 wrapper 承担路径参数与请求体绑定；权限中间件与 handler envelope 写入保持迁移前语义。
	wrapper := backupopenapi.ServerInterfaceWrapper{
		Handler: backupopenapi.NewStrictHandlerWithOptions(
			backupStrictServer{handlers: handlers},
			nil,
			backupopenapi.StrictGinServerOptions{
				RequestErrorHandlerFunc: func(c *gin.Context, err error) { handlers.writeError(c, http.StatusBadRequest, err) },
				HandlerErrorFunc:        strictsupport.HandlerErrorHandler(ctx.I18n, ctx.Logger),
			},
		),
		ErrorHandler: strictsupport.BindingErrorHandler(ctx.I18n, messagecontract.CommonInvalidArgument.String()),
	}
	group.GET("", httpx.RequirePermission(ctx.I18n, auth, authorizer, backupcontract.BackupReadPermission, publisher), strictsupport.Route(wrapper.ListPlatformBackups, handlers.list))
	group.GET(backupcontract.BackupDetailRoute, httpx.RequirePermission(ctx.I18n, auth, authorizer, backupcontract.BackupReadPermission, publisher), strictsupport.Route(wrapper.GetPlatformBackup, handlers.detail))
	group.POST("", httpx.RequirePermission(ctx.I18n, auth, authorizer, backupcontract.BackupCreatePermission, publisher), strictsupport.Route(wrapper.PostPlatformBackup, handlers.create))
	return nil
}

// backupStrictServer 将备份路由主体适配到生成的 strict server 契约；
// 编译期断言保证 OpenAPI 备份操作变化时必须在适配层补齐。
type backupStrictServer struct {
	handlers backupRouteHandlers
}

var _ backupopenapi.StrictServerInterface = backupStrictServer{}

func (s backupStrictServer) ListPlatformBackups(ctx context.Context, _ backupopenapi.ListPlatformBackupsRequestObject) (backupopenapi.ListPlatformBackupsResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.handlers.list)
}

func (s backupStrictServer) GetPlatformBackup(ctx context.Context, _ backupopenapi.GetPlatformBackupRequestObject) (backupopenapi.GetPlatformBackupResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.handlers.detail)
}

// PostPlatformBackup 接收生成绑定的请求体，保留备份保留期默认值。
func (s backupStrictServer) PostPlatformBackup(ctx context.Context, req backupopenapi.PostPlatformBackupRequestObject) (backupopenapi.PostPlatformBackupResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, func(c *gin.Context) {
		retention := ""
		if req.Body != nil && req.Body.Retention != nil {
			retention = string(*req.Body.Retention)
		}
		s.handlers.submit(c, retention)
	})
}

type backupRouteHandlers struct {
	service   *Service
	auth      moduleapi.AuthService
	localizer *i18n.Service
	logger    *zap.Logger
}

func (h backupRouteHandlers) create(c *gin.Context) {
	var request struct {
		Retention string `json:"retention"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		h.writeError(c, http.StatusBadRequest, err)
		return
	}
	h.submit(c, request.Retention)
}

func (h backupRouteHandlers) submit(c *gin.Context, requestedRetention string) {
	retention := ManualRetention(strings.TrimSpace(requestedRetention))
	if retention == "" {
		retention = ManualRetentionThirtyDays
	}
	retainUntil, err := ManualRetentionDeadline(retention, time.Now().UTC())
	if err != nil {
		h.writeError(c, http.StatusBadRequest, err)
		return
	}
	actor, err := h.auth.CurrentUser(c.Request.Context())
	if err != nil || actor == nil {
		h.writeError(c, http.StatusUnauthorized, err)
		return
	}
	idempotencyKey := c.GetHeader("Idempotency-Key")
	if strings.TrimSpace(idempotencyKey) == "" {
		h.writeError(c, http.StatusBadRequest, moduleapi.ErrBackupInvalidInput)
		return
	}
	receipt, err := h.service.SubmitManualBackup(c.Request.Context(), manualBackupOperationID(idempotencyKey, time.Now().UTC()), actor.ID, retainUntil, idempotencyKey)
	if err != nil {
		h.writeError(c, backupHTTPStatus(err), err)
		return
	}
	httpx.WriteSuccess(c, http.StatusAccepted, backupTaskReceiptResponse{TaskID: receipt.TaskID, Status: receipt.Status})
}

func (h backupRouteHandlers) list(c *gin.Context) {
	limit, offset, err := backupListPage(c)
	if err != nil {
		h.writeError(c, http.StatusBadRequest, err)
		return
	}
	items, total, err := h.service.ListSummaries(c.Request.Context(), limit, offset)
	if err != nil {
		h.writeError(c, backupHTTPStatus(err), err)
		return
	}
	httpx.WriteSuccess(c, http.StatusOK, backupListResponse{Items: backupSummaryResponses(items), Total: total, Limit: limit, Offset: offset})
}

func (h backupRouteHandlers) detail(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		h.writeError(c, http.StatusBadRequest, err)
		return
	}
	item, err := h.service.Get(c.Request.Context(), id)
	if err != nil {
		h.writeError(c, backupHTTPStatus(err), err)
		return
	}
	httpx.WriteSuccess(c, http.StatusOK, toBackupDetailResponse(item))
}

func backupListPage(c *gin.Context) (int, int, error) {
	limit, offset := defaultBackupListLimit, 0
	if raw := c.Query("limit"); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value < 1 || value > maxBackupListLimit {
			return 0, 0, moduleapi.ErrBackupInvalidInput
		}
		limit = value
	}
	if raw := c.Query("offset"); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value < 0 {
			return 0, 0, moduleapi.ErrBackupInvalidInput
		}
		offset = value
	}
	return limit, offset, nil
}

func backupHTTPStatus(err error) int {
	switch {
	case errors.Is(err, moduleapi.ErrBackupNotFound):
		return http.StatusNotFound
	case errors.Is(err, moduleapi.ErrBackupInvalidInput):
		return http.StatusBadRequest
	case errors.Is(err, moduleapi.ErrTaskExecutionUnavailable):
		return http.StatusServiceUnavailable
	case errors.Is(err, moduleapi.ErrTaskOwnerBusy), errors.Is(err, moduleapi.ErrTaskSubmissionConflict):
		return http.StatusConflict
	default:
		return http.StatusInternalServerError
	}
}

func (h backupRouteHandlers) writeError(c *gin.Context, status int, err error) {
	messageKey := messagecontract.CommonInternalError.String()
	switch status {
	case http.StatusBadRequest:
		messageKey = messagecontract.CommonInvalidArgument.String()
	case http.StatusNotFound:
		messageKey = messagecontract.CommonNotFound.String()
	case http.StatusUnauthorized:
		messageKey = "auth.unauthenticated"
	case http.StatusConflict:
		messageKey = messagecontract.CommonInvalidArgument.String()
	case http.StatusServiceUnavailable:
		messageKey = messagecontract.CommonTaskExecutionUnavailable.String()
	case http.StatusInternalServerError:
		logsafe.Error(h.logger, "backup request failed", zap.Error(err))
	}
	httpx.WriteLocalizedError(c, h.localizer, status, messageKey, nil)
}
