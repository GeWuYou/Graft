package backup

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	messagecontract "graft/server/internal/contract/message"
	"graft/server/internal/httpx"
	"graft/server/internal/i18n"
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
	handlers := backupRouteHandlers{service: service, auth: auth, localizer: ctx.I18n}
	group.GET("", httpx.RequirePermission(ctx.I18n, auth, authorizer, backupcontract.BackupReadPermission, publisher), handlers.list)
	group.GET(backupcontract.BackupDetailRoute, httpx.RequirePermission(ctx.I18n, auth, authorizer, backupcontract.BackupReadPermission, publisher), handlers.detail)
	group.POST("", httpx.RequirePermission(ctx.I18n, auth, authorizer, backupcontract.BackupCreatePermission, publisher), handlers.create)
	return nil
}

type backupRouteHandlers struct {
	service   *Service
	auth      moduleapi.AuthService
	localizer *i18n.Service
}

func (h backupRouteHandlers) create(c *gin.Context) {
	var request struct {
		Retention string `json:"retention"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		h.writeError(c, http.StatusBadRequest, err)
		return
	}
	retention := ManualRetention(strings.TrimSpace(request.Retention))
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
	httpx.WriteSuccess(c, http.StatusAccepted, map[string]any{"task_id": receipt.TaskID, "status": receipt.Status})
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
	httpx.WriteSuccess(c, http.StatusOK, map[string]any{"items": items, "total": total, "limit": limit, "offset": offset})
}

func (h backupRouteHandlers) detail(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		h.writeError(c, http.StatusBadRequest, err)
		return
	}
	item, err := h.service.GetSummary(c.Request.Context(), id)
	if err != nil {
		h.writeError(c, backupHTTPStatus(err), err)
		return
	}
	httpx.WriteSuccess(c, http.StatusOK, item)
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
	case errors.Is(err, moduleapi.ErrTaskSubmissionConflict):
		return http.StatusConflict
	default:
		return http.StatusInternalServerError
	}
}

func (h backupRouteHandlers) writeError(c *gin.Context, status int, _ error) {
	message := messagecontract.CommonInternalError.String()
	if status == http.StatusBadRequest {
		message = messagecontract.CommonInvalidArgument.String()
	}
	if status == http.StatusNotFound {
		message = messagecontract.CommonNotFound.String()
	}
	if status == http.StatusUnauthorized {
		message = "auth.unauthenticated"
	}
	httpx.WriteLocalizedError(c, h.localizer, status, message, nil)
}
