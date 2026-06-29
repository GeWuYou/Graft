package container

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"

	"graft/server/internal/eventbus"
	"graft/server/internal/httpx"
	"graft/server/internal/moduleapi"
	containercontract "graft/server/modules/container/contract"
)

func (s *service) publishActionAudit(ctx context.Context, result ActionResult, options ActionOptions, err error) {
	detached := startDetachedAuditContext(ctx, s)
	if !detached.ok {
		return
	}
	defer detached.cancel()
	action := actionAuditContract(result.Action).String()
	messageKey, message := auditErrorMessageFields(err)
	metadata := map[string]any{
		"container_id":   result.ID,
		"container_name": result.Name,
		"image":          result.Image,
		"action":         action,
		"runtime":        firstNonEmpty(result.Runtime, runtimeNameDocker),
		"endpoint":       safeEndpointLabel(s.runtimeOptions.endpoint),
		"force":          options.Force,
		"result":         auditResult(err),
		"error":          messageKey,
		"status_before":  result.StatusBefore,
		"status_after":   result.StatusAfter,
		"orchestrator_type": firstNonEmpty(
			result.Orchestrator.Type,
			containerOrchestratorUnknown,
		),
		"source_group_kind":   strings.TrimSpace(result.Orchestrator.GroupScopeKind),
		"source_group_value":  strings.TrimSpace(result.Orchestrator.GroupValue),
		"source_member_kind":  strings.TrimSpace(result.Orchestrator.MemberScopeKind),
		"source_member_value": strings.TrimSpace(result.Orchestrator.MemberValue),
	}
	enrichAuditMetadataWithRequestContext(detached.ctx, metadata, "")
	event := moduleapi.AuditEvent{
		Kind:          moduleapi.AuditEventKindDomain,
		Operator:      currentAuditOperator(detached.ctx),
		Action:        action,
		ResourceType:  containerResourceType,
		ResourceID:    firstNonEmpty(result.ID, result.Name),
		ResourceName:  result.Name,
		StatusCode:    auditStatusCode(err),
		Success:       err == nil,
		MessageKey:    messageKey,
		Message:       message,
		Metadata:      metadata,
		RequestMethod: "",
		RequestPath:   "",
	}
	s.publishAuditEvent(detached.ctx, event, "publish container audit event failed")
}

func (s *service) publishBatchActionAudit(ctx context.Context, result BatchActionResult, options ActionOptions) {
	detached := startDetachedAuditContext(ctx, s)
	if !detached.ok {
		return
	}
	defer detached.cancel()

	requestID := firstNonEmpty(strings.TrimSpace(result.RequestID), requestIDFromContext(ctx))
	resourceID := requestID
	if resourceID == "" {
		resourceID = batchAuditResourceID(result.Action, detached.now)
	}
	metadata := map[string]any{
		"batch":           true,
		"batch_action":    batchActionAuditContract(result.Action).String(),
		"requested_total": result.Total,
		"requested_ids":   batchRequestedIDs(result.Items),
		"success_count":   result.SuccessCount,
		"failed_count":    result.FailedCount,
		"failed_ids":      batchFailedIDs(result.Items),
		"force":           options.Force,
	}
	enrichAuditMetadataWithRequestContext(detached.ctx, metadata, requestID)
	event := moduleapi.AuditEvent{
		Kind:         moduleapi.AuditEventKindDomain,
		Operator:     currentAuditOperator(detached.ctx),
		Action:       batchActionAuditContract(result.Action).String(),
		ResourceType: containerBatchResourceType,
		ResourceID:   resourceID,
		ResourceName: strings.TrimSpace(result.Action) + " x" + strconv.Itoa(result.Total),
		StatusCode:   batchAuditStatusCode(result),
		Success:      result.FailedCount == 0,
		MessageKey:   strings.TrimSpace(result.MessageKey),
		Message:      strings.TrimSpace(result.Message),
		Metadata:     metadata,
	}
	s.publishAuditEvent(detached.ctx, event, "publish container batch audit event failed")
}

type detachedAuditRuntime struct {
	ctx    context.Context
	cancel context.CancelFunc
	ok     bool
	now    time.Time
}

func startDetachedAuditContext(ctx context.Context, s *service) detachedAuditRuntime {
	if s == nil || s.auditBus == nil {
		return detachedAuditRuntime{}
	}
	auditCtx, cancel := detachedAuditContext(ctx)
	return detachedAuditRuntime{
		ctx:    auditCtx,
		cancel: cancel,
		ok:     true,
		now:    time.Now().UTC(),
	}
}

func batchAuditResourceID(action string, now time.Time) string {
	return "batch:" + strings.TrimSpace(action) + ":" + strconv.FormatInt(now.UnixNano(), 10)
}

func enrichAuditMetadataWithRequestContext(auditCtx context.Context, metadata map[string]any, fallbackRequestID string) {
	if metadata == nil {
		return
	}
	if requestAudit, ok := httpx.RequestAuditContextFromContext(auditCtx); ok {
		metadata["requestId"] = firstNonEmpty(fallbackRequestID, requestAudit.RequestID)
		metadata["traceId"] = requestAudit.TraceID
		return
	}
	if strings.TrimSpace(fallbackRequestID) != "" {
		metadata["requestId"] = strings.TrimSpace(fallbackRequestID)
	}
}

func auditErrorMessageFields(err error) (string, string) {
	if err == nil {
		return "", ""
	}
	return messageKeyForError(err).String(), fallbackMessageForError(err)
}

func (s *service) publishAuditEvent(ctx context.Context, event moduleapi.AuditEvent, failureMessage string) {
	if s == nil || s.auditBus == nil {
		return
	}
	if publishErr := s.auditBus.Publish(ctx, eventbus.Event{
		Name:    string(moduleapi.AuditRecordEventName),
		Source:  s.moduleName,
		Payload: event,
	}); publishErr != nil && s.logger != nil {
		s.logger.Warn(failureMessage,
			zap.String("module", s.moduleName),
			zap.String("action", event.Action),
			zap.Error(publishErr),
		)
	}
}

// actionAuditContract 将容器动作字符串映射为审计动作类型。
// 预定义动作会转换为对应的容器审计动作；其他值会按原字符串生成审计动作。
func actionAuditContract(action string) containercontract.AuditAction {
	return auditActionContract(action, false)
}

// 对已知动作返回对应的批量审计动作；其他值返回去除首尾空白后的原始动作。
func batchActionAuditContract(action string) containercontract.AuditAction {
	return auditActionContract(action, true)
}

func auditActionContract(action string, batch bool) containercontract.AuditAction {
	normalized := strings.TrimSpace(action)
	if batch {
		switch normalized {
		case containerActionStart:
			return containercontract.ContainerAuditActionBatchStart
		case containerActionStop:
			return containercontract.ContainerAuditActionBatchStop
		case containerActionRemove:
			return containercontract.ContainerAuditActionBatchRemove
		default:
			return containercontract.AuditAction(normalized)
		}
	}
	switch normalized {
	case containerActionStart:
		return containercontract.ContainerAuditActionStart
	case containerActionStop:
		return containercontract.ContainerAuditActionStop
	case containerActionRemove:
		return containercontract.ContainerAuditActionRemove
	default:
		return containercontract.AuditAction(normalized)
	}
}

// batchRequestedIDs 提取批量动作中每个条目的请求资源 ID。
// 返回的每个 ID 优先使用条目结果中的 ID，若为空则使用条目自身的 ID。
func batchRequestedIDs(items []BatchActionItem) []string {
	ids := make([]string, 0, len(items))
	for _, item := range items {
		ids = append(ids, firstNonEmpty(item.Result.ID, item.ID))
	}
	return ids
}

// batchFailedIDs 返回批量动作中执行失败项的 ID 列表。
// 每个失败项优先使用结果中的 ID，必要时使用请求中的 ID。
func batchFailedIDs(items []BatchActionItem) []string {
	ids := make([]string, 0, len(items))
	for _, item := range items {
		if item.Success {
			continue
		}
		ids = append(ids, firstNonEmpty(item.Result.ID, item.ID))
	}
	return ids
}

// batchAuditStatusCode 根据批量动作结果返回审计状态码。
// 当存在失败项时返回 `409 Conflict`，否则返回 `200 OK`。
func batchAuditStatusCode(result BatchActionResult) int {
	if result.FailedCount > 0 {
		return http.StatusConflict
	}
	return http.StatusOK
}

// currentAuditOperator 提取当前请求中的审计操作者信息。
// 当请求上下文中存在用户时，返回其副本；否则返回 nil。
func currentAuditOperator(ctx context.Context) *moduleapi.CurrentUser {
	requestAuth, ok := moduleapi.RequestAuthContextFromContext(ctx)
	if !ok || requestAuth.User == nil {
		return nil
	}
	user := *requestAuth.User
	return &user
}

func auditResult(err error) string {
	if err != nil {
		return "failed"
	}
	return "success"
}

// auditStatusCode 将错误转换为审计状态码。
// @returns 错误为 nil 时返回 http.StatusOK；否则返回与该错误对应的状态码。
func auditStatusCode(err error) int {
	if err == nil {
		return http.StatusOK
	}
	return statusForError(err)
}

func detachedAuditContext(ctx context.Context) (context.Context, context.CancelFunc) {
	auditCtx, cancel := context.WithTimeout(context.Background(), containerAuditPublishTimeout)
	if requestAudit, ok := httpx.RequestAuditContextFromContext(ctx); ok {
		auditCtx = httpx.WithRequestAuditContext(auditCtx, requestAudit)
	}
	if requestAuth, ok := moduleapi.RequestAuthContextFromContext(ctx); ok {
		auditCtx = moduleapi.WithRequestAuthContext(auditCtx, requestAuth)
	}
	return auditCtx, cancel
}
