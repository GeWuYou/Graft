package httpx

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	authcontract "graft/server/internal/contract/auth"
	"graft/server/internal/contract/httpheader"
	messagecontract "graft/server/internal/contract/message"
	"graft/server/internal/eventbus"
	"graft/server/internal/i18n"
	"graft/server/internal/pluginapi"
)

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
	ctx         context.Context
	requestAuth pluginapi.RequestAuthContext
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
// 依赖任何具体插件实现。缺少登录态返回 401，认证成功但权限不足返回 403。
func RequirePermission(
	localizer *i18n.Service,
	authService pluginapi.AuthService,
	authorizer pluginapi.Authorizer,
	code string,
	auditPublishers ...SecurityAuditPublisher,
) gin.HandlerFunc {
	auditPublisher := firstSecurityAuditPublisher(auditPublishers...)
	return func(ctx *gin.Context) {
		EnsureRequestID(ctx)

		if authService == nil {
			AbortLocalizedError(ctx, localizer, http.StatusInternalServerError, messagecontract.CommonInternalError.String(), nil)
			return
		}

		requestAuth, requestCtx, handled := authenticateRequest(ctx, localizer, authService, auditPublisher)
		if handled {
			return
		}
		if authorizeRequest(
			requestAuthorization{ctx: requestCtx, requestAuth: requestAuth},
			ctx,
			localizer,
			authorizer,
			code,
			auditPublisher,
		) {
			return
		}

		ctx.Request = ctx.Request.WithContext(requestCtx)
		ctx.Next()
	}
}

func authenticateRequest(
	ctx *gin.Context,
	localizer *i18n.Service,
	authService pluginapi.AuthService,
	auditPublisher SecurityAuditPublisher,
) (pluginapi.RequestAuthContext, context.Context, bool) {
	requestToken, ok := extractBearerToken(ctx.Request)
	if !ok {
		publishSecurityAudit(ctx, auditPublisher, securityAuditEventAuthTokenMissing, http.StatusUnauthorized, messagecontract.AuthTokenMissing.String(), nil)
		AbortLocalizedError(ctx, localizer, http.StatusUnauthorized, messagecontract.AuthTokenMissing.String(), nil)
		return pluginapi.RequestAuthContext{}, nil, true
	}

	claims, err := authService.ParseAccessToken(ctx.Request.Context(), requestToken)
	if err != nil {
		writeAccessTokenError(ctx, localizer, err, auditPublisher)
		return pluginapi.RequestAuthContext{}, nil, true
	}

	requestAuth := pluginapi.RequestAuthContext{Claims: claims}
	requestCtx := pluginapi.WithRequestAuthContext(ctx.Request.Context(), requestAuth)
	user, err := authService.CurrentUser(requestCtx)
	if err != nil {
		writeCurrentUserError(ctx, localizer, err, auditPublisher)
		return pluginapi.RequestAuthContext{}, nil, true
	}

	requestAuth.User = user
	requestCtx = pluginapi.WithRequestAuthContext(ctx.Request.Context(), requestAuth)
	return requestAuth, requestCtx, false
}

func authorizeRequest(
	request requestAuthorization,
	ctx *gin.Context,
	localizer *i18n.Service,
	authorizer pluginapi.Authorizer,
	code string,
	auditPublisher SecurityAuditPublisher,
) bool {
	if strings.TrimSpace(code) == "" {
		return false
	}

	if authorizer == nil {
		AbortLocalizedError(ctx, localizer, http.StatusInternalServerError, messagecontract.CommonInternalError.String(), nil)
		return true
	}
	if err := authorizer.Authorize(request.ctx, request.requestAuth, code); err != nil {
		writeAuthorizationError(ctx, localizer, code, err, auditPublisher)
		return true
	}

	return false
}

func writeAccessTokenError(ctx *gin.Context, localizer *i18n.Service, err error, auditPublisher SecurityAuditPublisher) {
	switch {
	case errors.Is(err, pluginapi.ErrExpiredAccessToken):
		writeSecurityAuditError(ctx, localizer, auditPublisher, securityAuditError{
			eventType:  securityAuditEventAuthTokenExpired,
			status:     http.StatusUnauthorized,
			messageKey: messagecontract.AuthTokenExpired.String(),
		})
	case errors.Is(err, pluginapi.ErrInvalidAccessToken):
		writeInvalidTokenAuditError(ctx, localizer, auditPublisher)
	default:
		AbortLocalizedError(ctx, localizer, http.StatusInternalServerError, messagecontract.CommonInternalError.String(), nil)
	}
}

func writeCurrentUserError(ctx *gin.Context, localizer *i18n.Service, err error, auditPublisher SecurityAuditPublisher) {
	switch {
	case errors.Is(err, pluginapi.ErrInvalidAccessToken):
		writeInvalidTokenAuditError(ctx, localizer, auditPublisher)
	case errors.Is(err, pluginapi.ErrUnauthenticated):
		writeMissingTokenAuditError(ctx, localizer, auditPublisher)
	default:
		AbortLocalizedError(ctx, localizer, http.StatusInternalServerError, messagecontract.CommonInternalError.String(), nil)
	}
}

func writeAuthorizationError(
	ctx *gin.Context,
	localizer *i18n.Service,
	code string,
	err error,
	auditPublisher SecurityAuditPublisher,
) {
	switch {
	case errors.Is(err, pluginapi.ErrPermissionDenied):
		writeSecurityAuditError(ctx, localizer, auditPublisher, securityAuditError{
			eventType:  securityAuditEventAuthorizationDeny,
			status:     http.StatusForbidden,
			messageKey: messagecontract.AuthForbidden.String(),
			metadata: map[string]any{
				"permission": code,
			},
		})
	case errors.Is(err, pluginapi.ErrInvalidAccessToken):
		writeInvalidTokenAuditError(ctx, localizer, auditPublisher)
	case errors.Is(err, pluginapi.ErrUnauthenticated):
		writeMissingTokenAuditError(ctx, localizer, auditPublisher)
	default:
		AbortLocalizedError(ctx, localizer, http.StatusInternalServerError, messagecontract.CommonInternalError.String(), nil)
	}
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
	requestPath := currentRequestAuditPath(ctx)
	metadata = canonicalizeSecurityEventMetadata(
		metadata,
		eventType,
		p.source,
		requestID,
		requestPath,
		strings.TrimSpace(ctx.Request.Method),
		status,
	)

	event := pluginapi.AuditEvent{
		Kind:          pluginapi.AuditEventKindSecurity,
		Action:        string(eventType),
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
	if requestAuth, ok := pluginapi.RequestAuthContextFromContext(ctx.Request.Context()); ok && requestAuth.User != nil {
		user := *requestAuth.User
		event.Operator = &user
		event.Metadata["actorId"] = strconv.FormatUint(user.ID, 10)
		event.Metadata["actorType"] = "user"
	}

	if err := p.bus.Publish(ctx.Request.Context(), eventbus.Event{
		Name:    pluginapi.AuditRecordEventName,
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

func canonicalizeSecurityEventMetadata(
	metadata map[string]any,
	eventType securityAuditEventType,
	source string,
	requestID string,
	route string,
	method string,
	status int,
) map[string]any {
	cloned := cloneSecurityAuditMetadata(metadata)
	if cloned == nil {
		cloned = map[string]any{}
	}

	cloned["requestId"] = requestID
	cloned["traceId"] = requestID
	cloned["route"] = route
	cloned["method"] = method
	cloned["path"] = route
	cloned["status"] = status
	cloned["plugin"] = firstNonEmptyTrimmed(source, "httpx")
	cloned["component"] = "httpx.authz"
	cloned["eventType"] = string(eventType)
	cloned["riskLevel"] = "CRITICAL"

	if permission, ok := cloned["permission"].(string); ok && strings.TrimSpace(permission) != "" {
		cloned["targetType"] = "permission"
		cloned["targetId"] = strings.TrimSpace(permission)
	}

	return cloned
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
