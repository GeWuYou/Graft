package httpx

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	authcontract "graft/server/internal/contract/auth"
	"graft/server/internal/contract/httpheader"
	messagecontract "graft/server/internal/contract/message"
	"graft/server/internal/event"
	"graft/server/internal/eventbus"
	"graft/server/internal/i18n"
	"graft/server/internal/moduleapi"
)

// NewAuditEvent 将稳定的审计 DTO 编码为 Runtime 事件 envelope。
// 发布方只负责生成 envelope；投递模式和 handler 生命周期仍由 Runtime 管理。
func NewAuditEvent(source string, payload moduleapi.AuditEvent) (event.Event, error) {
	encoded, err := json.Marshal(payload)
	if err != nil {
		return event.Event{}, fmt.Errorf("encode audit event: %w", err)
	}
	id, err := event.NewID()
	if err != nil {
		return event.Event{}, err
	}
	now := time.Now().UTC()
	occurredAt := payload.CreatedAt
	if occurredAt.IsZero() {
		occurredAt = now
	}
	return event.Event{
		ID:             id,
		Type:           event.Type(moduleapi.AuditRecordEventName),
		Version:        1,
		Source:         strings.TrimSpace(source),
		Payload:        encoded,
		OccurredAt:     occurredAt.UTC(),
		CreatedAt:      now,
		CorrelationID:  strings.TrimSpace(payload.RequestID),
		IdempotencyKey: strings.TrimSpace(payload.RequestID),
	}, nil
}

type securityAuditEventType string

//nolint:gosec // Stable audit event names are security semantics, not credentials.
const (
	securityAuditEventAuthTokenMissing  securityAuditEventType = "auth.token.missing"
	securityAuditEventAuthTokenExpired  securityAuditEventType = "auth.token.expired"
	securityAuditEventAuthTokenInvalid  securityAuditEventType = "auth.token.invalid"
	securityAuditEventAuthorizationDeny securityAuditEventType = "auth.permission.denied"
)

// SecurityAuditPublisher emits security-relevant auth/authz failures into the audit event path.
type SecurityAuditPublisher interface {
	Publish(*gin.Context, securityAuditEventType, int, string, map[string]any)
}

type eventBusSecurityAuditPublisher struct {
	bus    eventbus.Bus
	logger *zap.Logger
	source string
}

type requestAuthorization struct {
	ctx            context.Context
	requestAuth    moduleapi.RequestAuthContext
	localizer      *i18n.Service
	logger         *zap.Logger
	auditPublisher SecurityAuditPublisher
}

type securityAuditError struct {
	eventType  securityAuditEventType
	status     int
	messageKey string
	metadata   map[string]any
}

// RequirePermission 以真实请求鉴权上下文保护路由。
//
// 该中间件只负责从请求中提取访问令牌、解析当前主体并调用授权器，不直接
// 依赖任何具体模块实现。缺少登录态返回 401，认证成功但权限不足返回 403。
func RequirePermission(
	localizer *i18n.Service,
	authService moduleapi.AuthService,
	authorizer moduleapi.Authorizer,
	code string,
	auditPublishers ...SecurityAuditPublisher,
) gin.HandlerFunc {
	return RequirePermissionWithLogger(nil, localizer, authService, authorizer, code, auditPublishers...)
}

// RequirePermissionWithLogger 以显式 logger 依赖创建鉴权中间件。
// logger 为空时使用 nop logger，确保缺少认证基础设施时仍能安全返回内部错误。
func RequirePermissionWithLogger(
	logger *zap.Logger,
	localizer *i18n.Service,
	authService moduleapi.AuthService,
	authorizer moduleapi.Authorizer,
	code string,
	auditPublishers ...SecurityAuditPublisher,
) gin.HandlerFunc {
	if logger == nil {
		logger = zap.NewNop()
	}
	auditPublisher := firstSecurityAuditPublisher(auditPublishers...)
	return func(ctx *gin.Context) {
		_, personalTokenCaller := moduleapi.PersonalAccessTokenCallerFromContext(ctx.Request.Context())
		if authService == nil && !personalTokenCaller {
			abortAuthorizationInternalError(ctx, localizer, logger, errors.New("auth service is unavailable"))
			return
		}

		requestAuth, requestCtx, handled := authenticateRequest(ctx, localizer, logger, authService, auditPublisher)
		if handled {
			return
		}
		if authorizeRequest(
			requestAuthorization{
				ctx:            requestCtx,
				requestAuth:    requestAuth,
				localizer:      localizer,
				logger:         logger,
				auditPublisher: auditPublisher,
			},
			ctx,
			authorizer,
			code,
		) {
			return
		}

		ctx.Request = ctx.Request.WithContext(requestCtx)
		ctx.Next()
	}
}

// RequirePersonalAccessToken 使用 auth 模块验证的个人 API Token 保护非 REST transport 入口。
//
// 它复用 request-id、统一错误和安全审计链路，并把认证结果写入稳定请求 context；
// 后续 transport 仍必须通过 moduleapi.Authorizer 完成每个业务 permission 的 RBAC 判断。
func RequirePersonalAccessToken(
	localizer *i18n.Service,
	tokenService moduleapi.PersonalAccessTokenService,
	auditPublishers ...SecurityAuditPublisher,
) gin.HandlerFunc {
	logger := zap.NewNop()
	auditPublisher := firstSecurityAuditPublisher(auditPublishers...)
	return func(ctx *gin.Context) {
		if tokenService == nil {
			abortAuthorizationInternalError(ctx, localizer, logger, errors.New("personal access token service is unavailable"))
			return
		}
		requestToken, ok := extractBearerToken(ctx.Request)
		if !ok {
			writeMissingTokenAuditError(ctx, localizer, auditPublisher)
			return
		}
		caller, err := tokenService.AuthenticatePersonalAccessToken(ctx.Request.Context(), requestToken)
		if err != nil {
			writePersonalAccessTokenError(ctx, localizer, logger, err, auditPublisher)
			return
		}

		user := caller.User
		requestCtx := moduleapi.WithRequestAuthContext(ctx.Request.Context(), moduleapi.RequestAuthContext{User: &user})
		requestCtx = moduleapi.WithPersonalAccessTokenCaller(requestCtx, caller)
		ctx.Request = ctx.Request.WithContext(requestCtx)
		ctx.Next()
	}
}

func authenticateRequest(
	ctx *gin.Context,
	localizer *i18n.Service,
	logger *zap.Logger,
	authService moduleapi.AuthService,
	auditPublisher SecurityAuditPublisher,
) (moduleapi.RequestAuthContext, context.Context, bool) {
	if caller, ok := moduleapi.PersonalAccessTokenCallerFromContext(ctx.Request.Context()); ok {
		requestAuth := moduleapi.RequestAuthContext{User: &caller.User}
		requestCtx := moduleapi.WithRequestAuthContext(ctx.Request.Context(), requestAuth)
		return requestAuth, requestCtx, false
	}
	requestToken, ok := extractBearerToken(ctx.Request)
	if !ok {
		publishSecurityAudit(ctx, auditPublisher, securityAuditEventAuthTokenMissing, http.StatusUnauthorized, messagecontract.AuthTokenMissing.String(), nil)
		AbortLocalizedError(ctx, localizer, http.StatusUnauthorized, messagecontract.AuthTokenMissing.String(), nil)
		return moduleapi.RequestAuthContext{}, nil, true
	}

	claims, err := authService.ParseAccessToken(ctx.Request.Context(), requestToken)
	if err != nil {
		writeAccessTokenError(ctx, localizer, logger, err, auditPublisher)
		return moduleapi.RequestAuthContext{}, nil, true
	}

	requestAuth := moduleapi.RequestAuthContext{Claims: claims}
	requestCtx := moduleapi.WithRequestAuthContext(ctx.Request.Context(), requestAuth)
	user, err := authService.CurrentUser(requestCtx)
	if err != nil {
		writeCurrentUserError(ctx, localizer, logger, err, auditPublisher)
		return moduleapi.RequestAuthContext{}, nil, true
	}

	requestAuth.User = user
	requestCtx = moduleapi.WithRequestAuthContext(ctx.Request.Context(), requestAuth)
	return requestAuth, requestCtx, false
}

func authorizeRequest(
	request requestAuthorization,
	ctx *gin.Context,
	authorizer moduleapi.Authorizer,
	code string,
) bool {
	ctx.Request = ctx.Request.WithContext(request.ctx)
	if strings.TrimSpace(code) == "" {
		if _, personalToken := moduleapi.PersonalAccessTokenCallerFromContext(request.ctx); personalToken {
			writeAuthorizationError(ctx, request.localizer, request.logger, code, moduleapi.ErrPermissionDenied, request.auditPublisher)
			return true
		}
		return false
	}

	if authorizer == nil {
		abortAuthorizationInternalError(ctx, request.localizer, request.logger, errors.New("authorizer is unavailable"))
		return true
	}
	if err := authorizer.Authorize(request.ctx, request.requestAuth, code); err != nil {
		writeAuthorizationError(ctx, request.localizer, request.logger, code, err, request.auditPublisher)
		return true
	}
	if caller, ok := moduleapi.PersonalAccessTokenCallerFromContext(request.ctx); ok && !personalTokenHasScope(caller.Scopes, code) {
		writeAuthorizationError(ctx, request.localizer, request.logger, code, moduleapi.ErrPermissionDenied, request.auditPublisher)
		return true
	}

	return false
}

func personalTokenHasScope(scopes []string, permission string) bool {
	for _, scope := range scopes {
		if scope == permission {
			return true
		}
	}
	return false
}

func writeAccessTokenError(ctx *gin.Context, localizer *i18n.Service, logger *zap.Logger, err error, auditPublisher SecurityAuditPublisher) {
	switch {
	case errors.Is(err, moduleapi.ErrExpiredAccessToken):
		writeSecurityAuditError(ctx, localizer, auditPublisher, securityAuditError{
			eventType:  securityAuditEventAuthTokenExpired,
			status:     http.StatusUnauthorized,
			messageKey: messagecontract.AuthTokenExpired.String(),
		})
	case errors.Is(err, moduleapi.ErrInvalidAccessToken):
		writeInvalidTokenAuditError(ctx, localizer, auditPublisher)
	default:
		abortAuthorizationInternalError(ctx, localizer, logger, err)
	}
}

func writePersonalAccessTokenError(ctx *gin.Context, localizer *i18n.Service, logger *zap.Logger, err error, auditPublisher SecurityAuditPublisher) {
	switch {
	case errors.Is(err, moduleapi.ErrExpiredPersonalAccessToken):
		writeSecurityAuditError(ctx, localizer, auditPublisher, securityAuditError{
			eventType:  securityAuditEventAuthTokenExpired,
			status:     http.StatusUnauthorized,
			messageKey: messagecontract.AuthTokenExpired.String(),
		})
	case errors.Is(err, moduleapi.ErrInvalidPersonalAccessToken):
		writeInvalidTokenAuditError(ctx, localizer, auditPublisher)
	default:
		abortAuthorizationInternalError(ctx, localizer, logger, err)
	}
}

func writeCurrentUserError(ctx *gin.Context, localizer *i18n.Service, logger *zap.Logger, err error, auditPublisher SecurityAuditPublisher) {
	switch {
	case errors.Is(err, moduleapi.ErrInvalidAccessToken):
		writeInvalidTokenAuditError(ctx, localizer, auditPublisher)
	case errors.Is(err, moduleapi.ErrUnauthenticated):
		writeMissingTokenAuditError(ctx, localizer, auditPublisher)
	default:
		abortAuthorizationInternalError(ctx, localizer, logger, err)
	}
}

func writeAuthorizationError(
	ctx *gin.Context,
	localizer *i18n.Service,
	logger *zap.Logger,
	code string,
	err error,
	auditPublisher SecurityAuditPublisher,
) {
	switch {
	case errors.Is(err, moduleapi.ErrPermissionDenied):
		writeSecurityAuditError(ctx, localizer, auditPublisher, securityAuditError{
			eventType:  securityAuditEventAuthorizationDeny,
			status:     http.StatusForbidden,
			messageKey: messagecontract.AuthForbidden.String(),
			metadata: map[string]any{
				"permission": code,
			},
		})
	case errors.Is(err, moduleapi.ErrInvalidAccessToken):
		writeInvalidTokenAuditError(ctx, localizer, auditPublisher)
	case errors.Is(err, moduleapi.ErrUnauthenticated):
		writeMissingTokenAuditError(ctx, localizer, auditPublisher)
	default:
		abortAuthorizationInternalError(ctx, localizer, logger, err)
	}
}

// abortAuthorizationInternalError 保留鉴权基础设施失败的 cause，由 HTTP fallback 记录一次关联错误。
func abortAuthorizationInternalError(ctx *gin.Context, localizer *i18n.Service, logger *zap.Logger, err error) {
	AbortAppError(ctx, localizer, logger, err)
}

func writeInvalidTokenAuditError(ctx *gin.Context, localizer *i18n.Service, auditPublisher SecurityAuditPublisher) {
	writeSecurityAuditError(ctx, localizer, auditPublisher, securityAuditError{
		eventType:  securityAuditEventAuthTokenInvalid,
		status:     http.StatusUnauthorized,
		messageKey: messagecontract.AuthTokenInvalid.String(),
	})
}

func writeMissingTokenAuditError(ctx *gin.Context, localizer *i18n.Service, auditPublisher SecurityAuditPublisher) {
	writeSecurityAuditError(ctx, localizer, auditPublisher, securityAuditError{
		eventType:  securityAuditEventAuthTokenMissing,
		status:     http.StatusUnauthorized,
		messageKey: messagecontract.AuthTokenMissing.String(),
	})
}

func writeSecurityAuditError(
	ctx *gin.Context,
	localizer *i18n.Service,
	auditPublisher SecurityAuditPublisher,
	auditError securityAuditError,
) {
	publishSecurityAudit(
		ctx,
		auditPublisher,
		auditError.eventType,
		auditError.status,
		auditError.messageKey,
		auditError.metadata,
	)
	AbortLocalizedError(ctx, localizer, auditError.status, auditError.messageKey, auditError.metadata)
}

// NewSecurityAuditPublisher builds a request-guard publisher backed by the shared event bus.
func NewSecurityAuditPublisher(bus eventbus.Bus, logger *zap.Logger, source string) SecurityAuditPublisher {
	if bus == nil {
		return nil
	}
	if logger == nil {
		logger = zap.NewNop()
	}

	return eventBusSecurityAuditPublisher{
		bus:    bus,
		logger: logger,
		source: strings.TrimSpace(source),
	}
}

func firstSecurityAuditPublisher(publishers ...SecurityAuditPublisher) SecurityAuditPublisher {
	for _, publisher := range publishers {
		if publisher != nil {
			return publisher
		}
	}
	return nil
}

func publishSecurityAudit(
	ctx *gin.Context,
	publisher SecurityAuditPublisher,
	eventType securityAuditEventType,
	status int,
	messageKey string,
	metadata map[string]any,
) {
	if publisher == nil {
		return
	}

	publisher.Publish(ctx, eventType, status, messageKey, metadata)
}

func (p eventBusSecurityAuditPublisher) Publish(
	ctx *gin.Context,
	eventType securityAuditEventType,
	status int,
	messageKey string,
	metadata map[string]any,
) {
	if p.bus == nil || ctx == nil || ctx.Request == nil {
		return
	}

	requestID := EnsureRequestID(ctx)
	traceID := EnsureTraceID(ctx)
	requestPath := currentRequestAuditPath(ctx)
	metadata = canonicalizeSecurityEventMetadata(
		metadata,
		securityEventContext{
			eventType: eventType,
			source:    p.source,
			requestID: requestID,
			traceID:   traceID,
			route:     requestPath,
			method:    strings.TrimSpace(ctx.Request.Method),
			status:    status,
		},
	)

	event := moduleapi.AuditEvent{
		Kind:          moduleapi.AuditEventKindSecurity,
		Action:        string(eventType),
		ResourceType:  securityEventResourceType(metadata),
		ResourceID:    securityEventResourceID(metadata),
		ResourceName:  securityEventResourceName(metadata),
		RequestMethod: strings.TrimSpace(ctx.Request.Method),
		RequestPath:   requestPath,
		StatusCode:    status,
		RequestID:     requestID,
		IP:            strings.TrimSpace(ctx.ClientIP()),
		UserAgent:     strings.TrimSpace(ctx.Request.UserAgent()),
		Success:       false,
		Message:       strings.TrimSpace(messageKey),
		Metadata:      cloneSecurityAuditMetadata(metadata),
	}
	if requestAuth, ok := moduleapi.RequestAuthContextFromContext(ctx.Request.Context()); ok && requestAuth.User != nil {
		user := *requestAuth.User
		event.Operator = &user
		event.Metadata["actorId"] = strconv.FormatUint(user.ID, 10)
		event.Metadata["actorType"] = "user"
	}

	if err := p.bus.Publish(ctx.Request.Context(), eventbus.Event{
		Name:    string(moduleapi.AuditRecordEventName),
		Source:  firstNonEmptyTrimmed(p.source, "httpx"),
		Payload: event,
	}); err != nil {
		p.logger.Warn("publish security audit event failed",
			zap.String("source", firstNonEmptyTrimmed(p.source, "httpx")),
			zap.String("event_type", string(eventType)),
			zap.Error(err),
		)
	}
}

func currentRequestAuditPath(ctx *gin.Context) string {
	if ctx == nil || ctx.Request == nil {
		return ""
	}
	if route := strings.TrimSpace(ctx.FullPath()); route != "" {
		return route
	}
	return strings.TrimSpace(ctx.Request.URL.Path)
}

func cloneSecurityAuditMetadata(metadata map[string]any) map[string]any {
	if len(metadata) == 0 {
		return nil
	}

	cloned := make(map[string]any, len(metadata))
	for key, value := range metadata {
		cloned[key] = value
	}
	return cloned
}

type securityEventContext struct {
	eventType securityAuditEventType
	source    string
	requestID string
	traceID   string
	route     string
	method    string
	status    int
}

func canonicalizeSecurityEventMetadata(metadata map[string]any, eventCtx securityEventContext) map[string]any {
	cloned := cloneSecurityAuditMetadata(metadata)
	if cloned == nil {
		cloned = map[string]any{}
	}

	cloned["requestId"] = eventCtx.requestID
	cloned["traceId"] = eventCtx.traceID
	cloned["route"] = eventCtx.route
	cloned["method"] = eventCtx.method
	cloned["path"] = eventCtx.route
	cloned["status"] = eventCtx.status
	cloned["module"] = firstNonEmptyTrimmed(eventCtx.source, "httpx")
	cloned["component"] = "httpx.authz"
	cloned["eventType"] = string(eventCtx.eventType)
	cloned["riskLevel"] = "CRITICAL"

	if permission, ok := cloned["permission"].(string); ok && strings.TrimSpace(permission) != "" {
		cloned["targetType"] = "permission"
		cloned["targetId"] = strings.TrimSpace(permission)
		cloned["targetName"] = strings.TrimSpace(permission)
	} else {
		cloned["targetType"] = "auth"
		cloned["targetId"] = string(eventCtx.eventType)
		cloned["targetName"] = string(eventCtx.eventType)
	}

	return cloned
}

func securityEventResourceType(metadata map[string]any) string {
	if value := stringMetadata(metadata, "targetType"); value != "" {
		return value
	}
	return "auth"
}

func securityEventResourceID(metadata map[string]any) string {
	if value := stringMetadata(metadata, "targetId"); value != "" {
		return value
	}
	return stringMetadata(metadata, "eventType")
}

func securityEventResourceName(metadata map[string]any) string {
	if value := stringMetadata(metadata, "targetName"); value != "" {
		return value
	}
	return securityEventResourceID(metadata)
}

func stringMetadata(metadata map[string]any, key string) string {
	if len(metadata) == 0 {
		return ""
	}
	value, ok := metadata[key].(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(value)
}

func firstNonEmptyTrimmed(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func extractBearerToken(request *http.Request) (string, bool) {
	if request == nil {
		return "", false
	}

	prefix := authcontract.Bearer.Prefix()
	header := strings.TrimSpace(request.Header.Get(httpheader.Authorization.String()))
	if header == "" {
		return "", false
	}
	if !strings.HasPrefix(strings.ToLower(header), strings.ToLower(prefix)) {
		return "", false
	}

	token := strings.TrimSpace(header[len(prefix):])
	if token == "" {
		return "", false
	}

	return token, true
}
