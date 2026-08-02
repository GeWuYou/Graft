package update

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	messagecontract "graft/server/internal/contract/message"
	"graft/server/internal/event"
	"graft/server/internal/httpx"
	"graft/server/internal/i18n"
	"graft/server/internal/logger"
	"graft/server/internal/module"
	"graft/server/internal/moduleapi"
	updatecontract "graft/server/modules/update/contract"
)

// registerRoutes 安装只读发现与显式检查路由；检查不触发升级副作用。
func registerRoutes(ctx *module.Context, service *Service, rollout *RolloutService, diagnostics FailureDiagnosticStore) error {
	if ctx == nil || ctx.Router == nil {
		return nil
	}
	auth, err := resolveAuth(ctx)
	if err != nil {
		return err
	}
	authorizer, err := resolveAuthorizer(ctx)
	if err != nil {
		return err
	}
	group := ctx.Router.Group(updatecontract.UpdateGroup)
	group.Use(httpx.RequestIDMiddleware())
	publisher := httpx.NewSecurityAuditPublisher(ctx.EventBus, ctx.Logger, moduleID)
	if diagnostics == nil {
		return errors.New("platform-update failure diagnostic store is unavailable")
	}
	handlers := updateRouteHandlers{localizer: ctx.I18n, auth: auth, rollout: rollout, diagnostics: diagnostics, appLogger: ctx.AppLogger, auditPublisher: ctx.EventPublisher}
	group.GET(updatecontract.UpdateStatusRoute, httpx.RequirePermission(ctx.I18n, auth, authorizer, updatecontract.UpdateReadPermission.String()), func(ginCtx *gin.Context) {
		httpx.WriteSuccess(ginCtx, http.StatusOK, statusForUpdateViewer(ginCtx.Request.Context(), authorizer, service.Status()))
	})
	group.POST(updatecontract.UpdateCheckRoute, httpx.RequirePermission(ctx.I18n, auth, authorizer, updatecontract.UpdateCheckPermission.String()), func(ginCtx *gin.Context) {
		httpx.WriteSuccess(ginCtx, http.StatusOK, statusForUpdateViewer(ginCtx.Request.Context(), authorizer, service.Check(ginCtx.Request.Context())))
	})
	if rollout == nil {
		return errors.New("platform-update rollout service is unavailable")
	}
	group.GET(updatecontract.UpdateOperationCollectionRoute, httpx.RequirePermission(ctx.I18n, auth, authorizer, updatecontract.UpdateReadPermission.String()), handlers.list)
	group.GET(updatecontract.UpdateActiveOperationRoute, httpx.RequirePermission(ctx.I18n, auth, authorizer, updatecontract.UpdateReadPermission.String()), handlers.getActive)
	group.GET(updatecontract.UpdateFailureDiagnosticRoute, httpx.RequirePermission(ctx.I18n, auth, authorizer, updatecontract.UpdateManagePermission.String(), publisher), handlers.getFailureDiagnostic)
	group.GET(updatecontract.UpdateOperationDiagnosticRoute, httpx.RequirePermission(ctx.I18n, auth, authorizer, updatecontract.UpdateManagePermission.String(), publisher), handlers.getOperationFailureDiagnostic)
	group.GET(updatecontract.UpdateOperationEventsRoute, httpx.RequirePermission(ctx.I18n, auth, authorizer, updatecontract.UpdateReadPermission.String()), handlers.getEvents)
	group.GET(updatecontract.UpdateOperationRoute, httpx.RequirePermission(ctx.I18n, auth, authorizer, updatecontract.UpdateReadPermission.String()), handlers.get)
	group.POST(updatecontract.UpdateOperationCollectionRoute, httpx.RequirePermission(ctx.I18n, auth, authorizer, updatecontract.UpdateManagePermission.String(), publisher), handlers.start)
	group.POST(updatecontract.UpdateOperationRecoveryRoute, httpx.RequirePermission(ctx.I18n, auth, authorizer, updatecontract.UpdateManagePermission.String(), publisher), handlers.recover)
	return nil
}

type updateRouteHandlers struct {
	localizer      *i18n.Service
	auth           moduleapi.AuthService
	rollout        *RolloutService
	diagnostics    FailureDiagnosticStore
	appLogger      logger.AppLogger
	auditPublisher event.Publisher
}

// mayViewComposeCandidates 将宿主机路径限定在已通过更新管理权限校验的请求中。
// 读取状态失败时按拒绝处理，避免 read 权限意外获得升级配置细节。
func mayViewComposeCandidates(ctx context.Context, authorizer moduleapi.Authorizer) bool {
	requestAuth, ok := moduleapi.RequestAuthContextFromContext(ctx)
	return ok && authorizer != nil && authorizer.Authorize(ctx, requestAuth, updatecontract.UpdateManagePermission.String()) == nil
}

func statusForUpdateViewer(ctx context.Context, authorizer moduleapi.Authorizer, status Status) Status {
	canManage := mayViewComposeCandidates(ctx, authorizer)
	if !canManage {
		status = status.withoutComposeCandidates()
	}
	return status.withReadiness(canManage)
}

func (h updateRouteHandlers) list(c *gin.Context) {
	limit := 20
	if raw := c.Query("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 || parsed > 100 {
			httpx.WriteLocalizedError(c, h.localizer, http.StatusBadRequest, messagecontract.CommonInvalidArgument.String(), nil)
			return
		}
		limit = parsed
	}
	items, err := h.rollout.ListOperations(c.Request.Context(), limit)
	if err != nil {
		if h.appLogger != nil {
			h.appLogger.Named("modules.update.routes").Error(c.Request.Context(), "list platform update operations failed",
				logger.StringField(logger.FieldOperation, "platform_update.operations.list"),
				logger.StringField(logger.FieldRequestID, rolloutRequestID(c.Request.Context())),
				logger.ErrorField(err),
			)
		}
		httpx.WriteLocalizedError(c, h.localizer, http.StatusInternalServerError, messagecontract.CommonInternalError.String(), nil)
		return
	}
	httpx.WriteSuccess(c, http.StatusOK, items)
}

func (h updateRouteHandlers) get(c *gin.Context) {
	item, err := h.rollout.GetOperation(c.Request.Context(), c.Param("operationID"))
	if errors.Is(err, errUpdateOperationNotFound) {
		httpx.WriteLocalizedError(c, h.localizer, http.StatusNotFound, messagecontract.CommonNotFound.String(), nil)
		return
	}
	if errors.Is(err, errRunnerStateUnavailable) {
		httpx.WriteLocalizedError(c, h.localizer, http.StatusServiceUnavailable, messagecontract.CommonInternalError.String(), nil)
		return
	}
	if err != nil {
		httpx.WriteLocalizedError(c, h.localizer, http.StatusBadRequest, messagecontract.CommonInvalidArgument.String(), nil)
		return
	}
	httpx.WriteSuccess(c, http.StatusOK, item)
}

func (h updateRouteHandlers) getActive(c *gin.Context) {
	item, err := h.rollout.GetActiveOperation(c.Request.Context())
	if errors.Is(err, errActiveUpdateOperationNotFound) {
		var noActive *OperationView
		httpx.WriteSuccess(c, http.StatusOK, noActive)
		return
	}
	if errors.Is(err, errRunnerStateUnavailable) {
		httpx.WriteLocalizedError(c, h.localizer, http.StatusServiceUnavailable, messagecontract.CommonInternalError.String(), nil)
		return
	}
	if err != nil {
		httpx.WriteLocalizedError(c, h.localizer, http.StatusInternalServerError, messagecontract.CommonInternalError.String(), nil)
		return
	}
	httpx.WriteSuccess(c, http.StatusOK, item)
}

func (h updateRouteHandlers) getEvents(c *gin.Context) {
	operationID := c.Param("operationID")
	if !runnerOperationID.MatchString(operationID) {
		httpx.WriteLocalizedError(c, h.localizer, http.StatusBadRequest, messagecontract.CommonInvalidArgument.String(), nil)
		return
	}
	afterRevision := uint64(0)
	if raw := c.Query("after_revision"); raw != "" {
		parsed, err := strconv.ParseUint(raw, 10, 64)
		if err != nil {
			httpx.WriteLocalizedError(c, h.localizer, http.StatusBadRequest, messagecontract.CommonInvalidArgument.String(), nil)
			return
		}
		afterRevision = parsed
	}
	events, err := h.rollout.GetOperationEvents(c.Request.Context(), operationID, afterRevision, maxRunnerStateEventReplay)
	if errors.Is(err, errUpdateOperationNotFound) {
		httpx.WriteLocalizedError(c, h.localizer, http.StatusNotFound, messagecontract.CommonNotFound.String(), nil)
		return
	}
	if errors.Is(err, errRunnerStateUnavailable) {
		httpx.WriteLocalizedError(c, h.localizer, http.StatusServiceUnavailable, messagecontract.CommonInternalError.String(), nil)
		return
	}
	if err != nil {
		httpx.WriteLocalizedError(c, h.localizer, http.StatusBadRequest, messagecontract.CommonInvalidArgument.String(), nil)
		return
	}
	httpx.WriteSuccess(c, http.StatusOK, events)
}

func (h updateRouteHandlers) getFailureDiagnostic(c *gin.Context) {
	requestID := strings.TrimSpace(c.Param("requestID"))
	if requestID == "" {
		httpx.WriteLocalizedError(c, h.localizer, http.StatusBadRequest, messagecontract.CommonInvalidArgument.String(), nil)
		return
	}
	value, err := h.diagnostics.GetFailureDiagnostic(c.Request.Context(), requestID)
	if errors.Is(err, errUpdateFailureDiagnosticNotFound) {
		httpx.WriteLocalizedError(c, h.localizer, http.StatusNotFound, messagecontract.CommonNotFound.String(), nil)
		return
	}
	if err != nil {
		httpx.WriteLocalizedError(c, h.localizer, http.StatusInternalServerError, messagecontract.CommonInternalError.String(), nil)
		return
	}
	h.publishDiagnosticReadAudit(c.Request.Context(), value)
	httpx.WriteSuccess(c, http.StatusOK, value)
}

func (h updateRouteHandlers) getOperationFailureDiagnostic(c *gin.Context) {
	operationID := c.Param("operationID")
	if !runnerOperationID.MatchString(operationID) {
		httpx.WriteLocalizedError(c, h.localizer, http.StatusBadRequest, messagecontract.CommonInvalidArgument.String(), nil)
		return
	}
	value, err := h.diagnostics.GetFailureDiagnosticByOperation(c.Request.Context(), operationID)
	if errors.Is(err, errUpdateFailureDiagnosticNotFound) {
		httpx.WriteLocalizedError(c, h.localizer, http.StatusNotFound, messagecontract.CommonNotFound.String(), nil)
		return
	}
	if err != nil {
		httpx.WriteLocalizedError(c, h.localizer, http.StatusInternalServerError, messagecontract.CommonInternalError.String(), nil)
		return
	}
	h.publishDiagnosticReadAudit(c.Request.Context(), value)
	httpx.WriteSuccess(c, http.StatusOK, value)
}

func (h updateRouteHandlers) start(c *gin.Context) {
	var request struct {
		TargetVersion string `json:"target_version"`
		CandidateKey  string `json:"compose_candidate_key,omitempty"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		httpx.WriteLocalizedError(c, h.localizer, http.StatusBadRequest, messagecontract.CommonInvalidArgument.String(), nil)
		return
	}
	actor, err := h.auth.CurrentUser(c.Request.Context())
	if err != nil || actor == nil {
		httpx.WriteLocalizedError(c, h.localizer, http.StatusUnauthorized, "auth.unauthenticated", nil)
		return
	}
	requestID := httpx.EnsureRequestID(c)
	requestAudit, _ := httpx.RequestAuditContextFromContext(c.Request.Context())
	requestAudit.RequestID = requestID
	c.Request = c.Request.WithContext(httpx.WithRequestAuditContext(c.Request.Context(), requestAudit))
	operation, err := h.rollout.Start(c.Request.Context(), StartRolloutInput{
		RequestedBy:   actor.ID,
		TargetVersion: request.TargetVersion,
		CandidateKey:  request.CandidateKey,
	})
	if err != nil {
		h.writeStartFailure(c, actor.ID, request.TargetVersion, request.CandidateKey, err)
		return
	}
	httpx.WriteSuccess(c, http.StatusAccepted, OperationLaunchAcknowledgement{OperationID: operation.OperationID, RunnerID: operation.RunnerID})
}

func (h updateRouteHandlers) recover(c *gin.Context) {
	operationID := c.Param("operationID")
	if !runnerOperationID.MatchString(operationID) {
		httpx.WriteLocalizedError(c, h.localizer, http.StatusBadRequest, messagecontract.CommonInvalidArgument.String(), nil)
		return
	}
	operation, err := h.rollout.Recover(c.Request.Context(), operationID)
	if errors.Is(err, errUpdateOperationNotFound) {
		httpx.WriteLocalizedError(c, h.localizer, http.StatusNotFound, messagecontract.CommonNotFound.String(), nil)
		return
	}
	if errors.Is(err, errRunnerStateUnavailable) {
		httpx.WriteLocalizedError(c, h.localizer, http.StatusServiceUnavailable, messagecontract.CommonInternalError.String(), nil)
		return
	}
	if errors.Is(err, errRecoveryConflict) {
		httpx.WriteLocalizedError(c, h.localizer, http.StatusConflict, messagecontract.CommonInvalidArgument.String(), nil)
		return
	}
	if err != nil {
		httpx.WriteLocalizedError(c, h.localizer, http.StatusBadRequest, messagecontract.CommonInvalidArgument.String(), nil)
		return
	}
	httpx.WriteSuccess(c, http.StatusAccepted, OperationLaunchAcknowledgement{OperationID: operation.OperationID, RunnerID: operation.RunnerID})
}

func (h updateRouteHandlers) writeStartFailure(c *gin.Context, actorID uint64, targetVersion, _ string, err error) {
	code, stage, operationID := rolloutFailureDetails(err)
	status := rolloutFailureHTTPStatus(code)
	requestID := httpx.EnsureRequestID(c)
	diagnostic := newFailureDiagnostic(requestID, operationID, targetVersion, code, stage, err)
	if operationID != "" && h.rollout != nil && h.rollout.operations != nil {
		if operation, getErr := h.rollout.operations.Get(c.Request.Context(), operationID); getErr == nil {
			diagnostic.TaskID = operation.TaskID
		}
	}
	if h.diagnostics != nil {
		if createErr := h.diagnostics.CreateFailureDiagnostic(c.Request.Context(), diagnostic, actorID); createErr != nil {
			h.logDiagnosticPersistenceFailure(c.Request.Context(), createErr, diagnostic)
		}
	}
	h.logStartFailure(c.Request.Context(), actorID, status, diagnostic)
	h.publishStartFailureAudit(c.Request.Context(), actorID, status, diagnostic)
	httpx.WriteLocalizedErrorCode(c, h.localizer, status, code, rolloutFailureMessageKey(code), rolloutFailureResponseData(code))
}

func (h updateRouteHandlers) logStartFailure(ctx context.Context, actorID uint64, status int, diagnostic FailureDiagnostic) {
	if h.appLogger == nil {
		return
	}
	h.appLogger.Named("modules.update.route").Error(ctx, updateFailureDiagnosticSummary,
		logger.StringField(logger.FieldOperation, "platform_update.rollout_start"),
		logger.Uint64Field("actor_id", actorID),
		logger.IntField("http_status", status),
		logger.StringField("failure_code", diagnostic.FailureCode),
		logger.StringField("failure_stage", diagnostic.FailureStage),
		logger.StringField("operation_id", diagnostic.OperationID),
		logger.Uint64Field("task_id", diagnostic.TaskID),
		logger.StringField("target_version", diagnostic.TargetVersion),
		logger.StringField(logger.FieldError, diagnostic.Detail),
	)
}

func (h updateRouteHandlers) logDiagnosticPersistenceFailure(ctx context.Context, err error, diagnostic FailureDiagnostic) {
	if h.appLogger == nil {
		return
	}
	h.appLogger.Named("modules.update.route").Error(ctx, "persist platform update failure diagnostic failed",
		logger.StringField(logger.FieldOperation, "platform_update.failure_diagnostic_create"),
		logger.StringField("failure_code", diagnostic.FailureCode),
		logger.StringField("failure_stage", diagnostic.FailureStage),
		logger.StringField(logger.FieldError, sanitizeRolloutError(err)),
	)
}

func (h updateRouteHandlers) publishStartFailureAudit(ctx context.Context, actorID uint64, status int, diagnostic FailureDiagnostic) {
	if h.auditPublisher == nil {
		return
	}
	requestAudit, _ := httpx.RequestAuditContextFromContext(ctx)
	payload := moduleapi.AuditEvent{Kind: moduleapi.AuditEventKindDomain, Operator: &moduleapi.CurrentUser{ID: actorID}, Action: "platform.update.compose", ResourceType: "platform_update", ResourceID: diagnostic.OperationID, ResourceName: diagnostic.TargetVersion, RequestID: diagnostic.RequestID, RequestMethod: requestAudit.Method, RequestPath: requestAudit.Route, IP: requestAudit.ClientIP, UserAgent: requestAudit.UserAgent, StatusCode: status, Success: false, Message: diagnostic.FailureCode, Metadata: map[string]any{"failure_stage": diagnostic.FailureStage, "task_id": diagnostic.TaskID, "target_version": diagnostic.TargetVersion}, CreatedAt: time.Now().UTC()}
	h.publishAuditEvent(ctx, payload, diagnostic)
}

func (h updateRouteHandlers) publishDiagnosticReadAudit(ctx context.Context, diagnostic FailureDiagnostic) {
	if h.auditPublisher == nil {
		return
	}
	requestAuth, _ := moduleapi.RequestAuthContextFromContext(ctx)
	requestAudit, _ := httpx.RequestAuditContextFromContext(ctx)
	payload := moduleapi.AuditEvent{Kind: moduleapi.AuditEventKindDomain, Operator: requestAuth.User, Action: "platform.update.failure_diagnostic.read", ResourceType: "platform_update_failure_diagnostic", ResourceID: diagnostic.RequestID, ResourceName: diagnostic.TargetVersion, RequestID: requestAudit.RequestID, RequestMethod: requestAudit.Method, RequestPath: requestAudit.Route, IP: requestAudit.ClientIP, UserAgent: requestAudit.UserAgent, StatusCode: http.StatusOK, Success: true, Metadata: map[string]any{"failure_code": diagnostic.FailureCode, "failure_stage": diagnostic.FailureStage}, CreatedAt: time.Now().UTC()}
	h.publishAuditEvent(ctx, payload, diagnostic)
}

func (h updateRouteHandlers) publishAuditEvent(ctx context.Context, payload moduleapi.AuditEvent, diagnostic FailureDiagnostic) {
	envelope, err := httpx.NewAuditEvent(moduleID, payload)
	if err != nil {
		h.logAuditPublishFailure(ctx, err, diagnostic)
		return
	}
	if _, err := h.auditPublisher.Publish(ctx, envelope, event.PublishOptions{Delivery: event.DeliveryDurable}); err != nil {
		h.logAuditPublishFailure(ctx, err, diagnostic)
	}
}

func (h updateRouteHandlers) logAuditPublishFailure(ctx context.Context, err error, diagnostic FailureDiagnostic) {
	if h.appLogger == nil {
		return
	}
	h.appLogger.Named("modules.update.route").Error(ctx, "publish platform update diagnostic audit event failed",
		logger.StringField(logger.FieldOperation, "platform_update.failure_diagnostic_audit"),
		logger.StringField("failure_stage", diagnostic.FailureStage),
		logger.StringField(logger.FieldError, sanitizeRolloutError(err)),
	)
}

func resolveAuth(ctx *module.Context) (moduleapi.AuthService, error) {
	if ctx == nil || ctx.Services == nil {
		return nil, errors.New("module services are unavailable")
	}
	value, err := ctx.Services.Resolve((*moduleapi.AuthService)(nil))
	if err != nil {
		return nil, fmt.Errorf("resolve auth service: %w", err)
	}
	service, ok := value.(moduleapi.AuthService)
	if !ok || service == nil {
		return nil, fmt.Errorf("resolved auth service has unexpected type %T", value)
	}
	return service, nil
}
func resolveAuthorizer(ctx *module.Context) (moduleapi.Authorizer, error) {
	value, err := ctx.Services.Resolve((*moduleapi.Authorizer)(nil))
	if err != nil {
		return nil, fmt.Errorf("resolve authorizer: %w", err)
	}
	service, ok := value.(moduleapi.Authorizer)
	if !ok || service == nil {
		return nil, fmt.Errorf("resolved authorizer has unexpected type %T", value)
	}
	return service, nil
}
