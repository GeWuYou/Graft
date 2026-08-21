package container

import (
	"context"
	"net/http"
	"strings"

	"go.uber.org/zap"

	"graft/server/internal/eventbus"
	"graft/server/internal/httpx"
	"graft/server/internal/moduleapi"
	containercontract "graft/server/modules/container/contract"
)

func (s *service) publishActionAudit(ctx context.Context, result actionAuditSnapshot, options ActionOptions, err error) {
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

func (s *service) publishLifecycleTaskSubmissionAudit(ctx context.Context, ref Ref, action string, options ActionOptions, receipt moduleapi.TaskReceipt, err error) {
	detached := startDetachedAuditContext(ctx, s)
	if !detached.ok {
		return
	}
	defer detached.cancel()
	auditAction := actionAuditContract(action).String()
	messageKey, message := auditErrorMessageFields(err)
	metadata := map[string]any{
		"container_id":    ref.Value,
		"action":          auditAction,
		"runtime":         runtimeNameDocker,
		"force":           options.Force,
		"submission":      auditResult(err),
		"task_id":         receipt.TaskID,
		"task_status":     receipt.Status,
		"execution_state": "not_started",
		"error":           messageKey,
	}
	enrichAuditMetadataWithRequestContext(detached.ctx, metadata, "")
	s.publishAuditEvent(detached.ctx, moduleapi.AuditEvent{
		Kind:         moduleapi.AuditEventKindDomain,
		Operator:     currentAuditOperator(detached.ctx),
		Action:       auditAction,
		ResourceType: containerResourceType,
		ResourceID:   ref.Value,
		ResourceName: ref.Value,
		StatusCode:   auditStatusCode(err),
		Success:      err == nil,
		MessageKey:   messageKey,
		Message:      message,
		Metadata:     metadata,
	}, "publish container lifecycle task submission audit event failed")
}

func (s *service) publishDockerImageAudit(ctx context.Context, action containercontract.AuditAction, imageID, target string, force bool, err error) {
	detached := startDetachedAuditContext(ctx, s)
	if !detached.ok {
		return
	}
	defer detached.cancel()
	messageKey, message := auditErrorMessageFields(err)
	metadata := map[string]any{
		"image_id": imageID,
		"target":   target,
		"runtime":  runtimeNameDocker,
		"force":    force,
		"result":   auditResult(err),
		"error":    messageKey,
	}
	enrichAuditMetadataWithRequestContext(detached.ctx, metadata, "")
	s.publishAuditEvent(detached.ctx, moduleapi.AuditEvent{
		Kind:          moduleapi.AuditEventKindDomain,
		Operator:      currentAuditOperator(detached.ctx),
		Action:        action.String(),
		ResourceType:  "docker_image",
		ResourceID:    imageID,
		ResourceName:  imageID,
		StatusCode:    auditStatusCode(err),
		Success:       err == nil,
		MessageKey:    messageKey,
		Message:       message,
		Metadata:      metadata,
		RequestMethod: "",
		RequestPath:   "",
	}, "publish docker image audit event failed")
}

type detachedAuditRuntime struct {
	ctx    context.Context
	cancel context.CancelFunc
	ok     bool
}

// startDetachedAuditContext 创建用于审计发布的独立上下文。
// 当服务或审计总线不可用时，返回零值结果。
func startDetachedAuditContext(ctx context.Context, s *service) detachedAuditRuntime {
	if s == nil || s.auditBus == nil {
		return detachedAuditRuntime{}
	}
	auditCtx, cancel := detachedAuditContext(ctx)
	return detachedAuditRuntime{
		ctx:    auditCtx,
		cancel: cancel,
		ok:     true,
	}
}

// enrichAuditMetadataWithRequestContext 补充审计元数据中的请求和追踪标识。
// 如果上下文中存在请求审计信息，则写入 requestId 和 traceId；否则在提供了 fallbackRequestID 时写入 requestId。
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
	return auditActionContract(action)
}

func auditActionContract(action string) containercontract.AuditAction {
	normalized := strings.TrimSpace(action)
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

func (s *service) publishDockerVolumeAudit(ctx context.Context, volume DockerVolume, force bool, err error) {
	detached := startDetachedAuditContext(ctx, s)
	if !detached.ok {
		return
	}
	defer detached.cancel()
	messageKey, message := auditErrorMessageFields(err)
	metadata := map[string]any{"name": volume.Name, "driver": volume.Driver, "scope": volume.Scope, "force": force, "result": auditResult(err), "error": messageKey}
	if volume.ReferenceCount != nil {
		metadata["reference_count"] = *volume.ReferenceCount
	}
	enrichAuditMetadataWithRequestContext(detached.ctx, metadata, "")
	s.publishAuditEvent(detached.ctx, moduleapi.AuditEvent{Kind: moduleapi.AuditEventKindDomain, Operator: currentAuditOperator(detached.ctx), Action: containercontract.ContainerAuditActionVolumeRemove.String(), ResourceType: "docker_volume", ResourceID: volume.Name, ResourceName: volume.Name, StatusCode: auditStatusCode(err), Success: err == nil, MessageKey: messageKey, Message: message, Metadata: metadata}, "publish Docker volume audit event failed")
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
