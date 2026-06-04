package httpx

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"graft/server/internal/config"
	"graft/server/internal/eventbus"
	"graft/server/internal/i18n"
	"graft/server/internal/moduleapi"
)

func newTestLocalizer() *i18n.Service {
	return i18n.MustNew(config.I18nConfig{
		DefaultLocale:    "zh-CN",
		FallbackLocale:   "zh-CN",
		SupportedLocales: []string{"zh-CN", "en-US"},
	})
}

type testAuthService struct {
	parseAccessToken func(ctx context.Context, token string) (*moduleapi.AccessTokenClaims, error)
	currentUser      func(ctx context.Context) (*moduleapi.CurrentUser, error)
}

func (s testAuthService) CurrentUser(ctx context.Context) (*moduleapi.CurrentUser, error) {
	return s.currentUser(ctx)
}

func (s testAuthService) ParseAccessToken(ctx context.Context, token string) (*moduleapi.AccessTokenClaims, error) {
	return s.parseAccessToken(ctx, token)
}

type testAuthorizer struct {
	authorize func(ctx context.Context, request moduleapi.RequestAuthContext, permission string) error
}

func (a testAuthorizer) Authorize(ctx context.Context, request moduleapi.RequestAuthContext, permission string) error {
	return a.authorize(ctx, request, permission)
}

func newBearerRequest(path string, token string) *http.Request {
	request := httptest.NewRequest(http.MethodGet, path, nil)
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	return request
}

func newAuthenticatedClaims() *moduleapi.AccessTokenClaims {
	return &moduleapi.AccessTokenClaims{
		UserID:    7,
		SessionID: "session-1",
		IssuedAt:  time.Now(),
		ExpiresAt: time.Now().Add(time.Minute),
	}
}

func newAuthenticatedUser() *moduleapi.CurrentUser {
	return &moduleapi.CurrentUser{ID: 7, Username: "alice", DisplayName: "Alice"}
}

func newAuthenticatedAuthService() testAuthService {
	return testAuthService{
		parseAccessToken: func(context.Context, string) (*moduleapi.AccessTokenClaims, error) {
			return newAuthenticatedClaims(), nil
		},
		currentUser: func(context.Context) (*moduleapi.CurrentUser, error) {
			return newAuthenticatedUser(), nil
		},
	}
}

func assertAuthenticatedRequestAllowed(t *testing.T, permission string) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	recorder := httptest.NewRecorder()
	ctx, engine := gin.CreateTestContext(recorder)
	engine.Use(RequirePermission(nil, newAuthenticatedAuthService(), nil, permission))
	engine.GET("/api/profile", func(inner *gin.Context) {
		inner.Status(http.StatusOK)
	})

	ctx.Request = newBearerRequest("/api/profile", "token-1")
	engine.HandleContext(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}
}

func assertPermissionRejectsTokenError(t *testing.T, requestToken string, parseErr error, wantKey string, wantCode string) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	localizer := newTestLocalizer()
	authService := testAuthService{
		parseAccessToken: func(context.Context, string) (*moduleapi.AccessTokenClaims, error) {
			return nil, parseErr
		},
		currentUser: func(context.Context) (*moduleapi.CurrentUser, error) {
			t.Fatal("current user should not be called when token parse fails")
			return nil, nil
		},
	}
	authorizer := testAuthorizer{
		authorize: func(context.Context, moduleapi.RequestAuthContext, string) error {
			t.Fatal("authorize should not be called when token parse fails")
			return nil
		},
	}

	recorder := httptest.NewRecorder()
	ctx, engine := gin.CreateTestContext(recorder)
	engine.Use(RequirePermission(localizer, authService, authorizer, "user.read"))
	engine.GET("/api/users/:id", func(inner *gin.Context) {
		inner.Status(http.StatusOK)
	})

	ctx.Request = newBearerRequest("/api/users/1", requestToken)
	engine.HandleContext(ctx)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, recorder.Code)
	}

	var payload ErrorResponse
	if err := json.NewDecoder(recorder.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.MessageKey != wantKey || payload.Code != wantCode {
		t.Fatalf("expected token error contract %s/%s, got %#v", wantKey, wantCode, payload)
	}
}

// TestRequirePermissionRejectsMissingBearerToken 验证缺少访问令牌时会被后端守卫直接拒绝。
func TestRequirePermissionRejectsMissingBearerToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	localizer := newTestLocalizer()
	authService := testAuthService{
		parseAccessToken: func(context.Context, string) (*moduleapi.AccessTokenClaims, error) {
			t.Fatal("parse access token should not be called without bearer token")
			return nil, nil
		},
		currentUser: func(context.Context) (*moduleapi.CurrentUser, error) {
			t.Fatal("current user should not be called without bearer token")
			return nil, nil
		},
	}
	authorizer := testAuthorizer{
		authorize: func(context.Context, moduleapi.RequestAuthContext, string) error {
			t.Fatal("authorize should not be called without bearer token")
			return nil
		},
	}

	recorder := httptest.NewRecorder()
	ctx, engine := gin.CreateTestContext(recorder)
	engine.Use(RequirePermission(localizer, authService, authorizer, "user.read"))
	engine.GET("/api/users/:id", func(inner *gin.Context) {
		inner.Status(http.StatusOK)
	})

	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/users/1", nil)
	engine.HandleContext(ctx)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, recorder.Code)
	}

	var payload ErrorResponse
	if err := json.NewDecoder(recorder.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.MessageKey != "auth.token_missing" || payload.Code != "AUTH_TOKEN_MISSING" {
		t.Fatalf("expected missing token contract, got %#v", payload)
	}
}

// TestRequirePermissionRejectsPermissionDenied 验证认证成功但缺少权限时会返回 403。
func TestRequirePermissionRejectsPermissionDenied(t *testing.T) {
	gin.SetMode(gin.TestMode)
	localizer := newTestLocalizer()
	authService := testAuthService{
		parseAccessToken: func(context.Context, string) (*moduleapi.AccessTokenClaims, error) {
			return newAuthenticatedClaims(), nil
		},
		currentUser: func(context.Context) (*moduleapi.CurrentUser, error) {
			return newAuthenticatedUser(), nil
		},
	}
	authorizer := testAuthorizer{
		authorize: func(context.Context, moduleapi.RequestAuthContext, string) error {
			return moduleapi.ErrPermissionDenied
		},
	}

	recorder := httptest.NewRecorder()
	ctx, engine := gin.CreateTestContext(recorder)
	engine.Use(RequirePermission(localizer, authService, authorizer, "user.read"))
	engine.GET("/api/users/:id", func(inner *gin.Context) {
		inner.Status(http.StatusOK)
	})

	request := newBearerRequest("/api/users/1", "token-1")
	request.Header.Set(i18n.LocaleHeader, "en-US")
	ctx.Request = request
	engine.HandleContext(ctx)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d", http.StatusForbidden, recorder.Code)
	}

	var payload ErrorResponse
	if err := json.NewDecoder(recorder.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.MessageKey != "auth.forbidden" || payload.Code != "AUTH_FORBIDDEN" {
		t.Fatalf("expected forbidden message key, got %#v", payload)
	}
	if payload.Locale != "en-US" {
		t.Fatalf("expected requested locale to be echoed, got %#v", payload)
	}
	if payload.Details["permission"] != "user.read" {
		t.Fatalf("expected denied permission to be echoed, got %#v", payload)
	}
}

// TestRequirePermissionAllowsAuthorizedRequest 验证 bearer token 和授权判断都通过时，请求可以继续执行。
func TestRequirePermissionAllowsAuthorizedRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)

	authService := testAuthService{
		parseAccessToken: func(context.Context, string) (*moduleapi.AccessTokenClaims, error) {
			return newAuthenticatedClaims(), nil
		},
		currentUser: func(ctx context.Context) (*moduleapi.CurrentUser, error) {
			requestAuth, ok := moduleapi.RequestAuthContextFromContext(ctx)
			if !ok || requestAuth.Claims == nil || requestAuth.Claims.UserID != 7 {
				t.Fatalf("expected request auth claims to be populated before CurrentUser, got %#v, ok=%v", requestAuth, ok)
			}
			return newAuthenticatedUser(), nil
		},
	}
	authorizer := testAuthorizer{
		authorize: func(_ context.Context, request moduleapi.RequestAuthContext, _ string) error {
			if request.User == nil || request.User.ID != 7 {
				t.Fatalf("expected request user to be populated before Authorize, got %#v", request.User)
			}
			return nil
		},
	}

	recorder := httptest.NewRecorder()
	ctx, engine := gin.CreateTestContext(recorder)
	engine.Use(RequirePermission(nil, authService, authorizer, "user.read"))
	engine.GET("/api/users/:id", func(inner *gin.Context) {
		requestAuth, ok := moduleapi.RequestAuthContextFromContext(inner.Request.Context())
		if !ok || requestAuth.User == nil || requestAuth.User.ID != 7 {
			t.Fatalf("expected handler context to carry current user, got %#v, ok=%v", requestAuth, ok)
		}
		inner.Status(http.StatusOK)
	})

	ctx.Request = newBearerRequest("/api/users/1", "token-1")
	engine.HandleContext(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}
}

func TestRequirePermissionInjectsCanonicalRequestAuditContext(t *testing.T) {
	gin.SetMode(gin.TestMode)

	authService := testAuthService{
		parseAccessToken: func(context.Context, string) (*moduleapi.AccessTokenClaims, error) {
			return newAuthenticatedClaims(), nil
		},
		currentUser: func(context.Context) (*moduleapi.CurrentUser, error) {
			return newAuthenticatedUser(), nil
		},
	}
	authorizer := testAuthorizer{
		authorize: func(_ context.Context, _ moduleapi.RequestAuthContext, _ string) error {
			return nil
		},
	}

	recorder := httptest.NewRecorder()
	ctx, engine := gin.CreateTestContext(recorder)
	engine.Use(RequirePermission(nil, authService, authorizer, "user.read"))
	engine.GET("/api/users/:id", func(inner *gin.Context) {
		auditCtx, ok := RequestAuditContextFromContext(inner.Request.Context())
		if !ok {
			t.Fatal("expected request audit context in handler context")
		}
		if auditCtx.RequestID == "" || auditCtx.TraceID != auditCtx.RequestID {
			t.Fatalf("expected canonical request/trace ids, got %#v", auditCtx)
		}
		if auditCtx.Route != "/api/users/:id" || auditCtx.Method != http.MethodGet {
			t.Fatalf("expected route and method in audit context, got %#v", auditCtx)
		}
		if auditCtx.ClientIP != "198.51.100.8" || auditCtx.UserAgent != "authz-test" {
			t.Fatalf("expected client metadata in audit context, got %#v", auditCtx)
		}
		inner.Status(http.StatusOK)
	})

	ctx.Request = newBearerRequest("/api/users/1", "token-1")
	ctx.Request.Header.Set("User-Agent", "authz-test")
	ctx.Request.Header.Set("X-Forwarded-For", "198.51.100.8")
	engine.HandleContext(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}
}

func TestSecurityAuditPublisherPublishesPermissionDeniedAuditEvent(t *testing.T) {
	gin.SetMode(gin.TestMode)

	bus := eventbus.New(nil)
	events := make([]moduleapi.AuditEvent, 0, 1)
	if err := bus.Subscribe(string(moduleapi.AuditRecordEventName), func(_ context.Context, event eventbus.Event) error {
		payload, ok := event.Payload.(moduleapi.AuditEvent)
		if !ok {
			t.Fatalf("expected audit event payload, got %#v", event.Payload)
		}
		events = append(events, payload)
		return nil
	}); err != nil {
		t.Fatalf("subscribe audit event: %v", err)
	}

	authService := newAuthenticatedAuthService()
	authorizer := testAuthorizer{
		authorize: func(context.Context, moduleapi.RequestAuthContext, string) error {
			return moduleapi.ErrPermissionDenied
		},
	}
	publisher := NewSecurityAuditPublisher(bus, nil, "test-authz")

	recorder := httptest.NewRecorder()
	ctx, engine := gin.CreateTestContext(recorder)
	engine.Use(RequirePermission(newTestLocalizer(), authService, authorizer, "user.read", publisher))
	engine.GET("/api/users/:id", func(inner *gin.Context) {
		inner.Status(http.StatusOK)
	})

	request := newBearerRequest("/api/users/1", "token-1")
	request.Header.Set("User-Agent", "security-test")
	ctx.Request = request
	engine.HandleContext(ctx)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d", http.StatusForbidden, recorder.Code)
	}
	if len(events) != 1 {
		t.Fatalf("expected one security audit event, got %d", len(events))
	}

	assertPermissionDeniedSecurityAuditEvent(t, events[0])
}

func assertPermissionDeniedSecurityAuditEvent(t *testing.T, event moduleapi.AuditEvent) {
	t.Helper()

	if event.Kind != moduleapi.AuditEventKindSecurity {
		t.Fatalf("expected security event kind, got %q", event.Kind)
	}
	if event.Action != string(securityAuditEventAuthorizationDeny) {
		t.Fatalf("expected permission-denied action, got %q", event.Action)
	}
	if event.Operator == nil || event.Operator.ID != 7 {
		t.Fatalf("expected authenticated operator, got %#v", event.Operator)
	}
	if event.RequestPath != "/api/users/:id" || event.RequestMethod != http.MethodGet {
		t.Fatalf("expected canonical route/method, got %#v", event)
	}
	if event.ResourceType != "permission" || event.ResourceID != "user.read" || event.ResourceName != "user.read" {
		t.Fatalf("expected permission target resource context, got %#v", event)
	}
	if event.StatusCode != http.StatusForbidden || event.Success {
		t.Fatalf("expected failed 403 event, got %#v", event)
	}

	assertSecurityAuditMetadata(t, event.Metadata, map[string]any{
		"component":  "httpx.authz",
		"eventType":  string(securityAuditEventAuthorizationDeny),
		"method":     http.MethodGet,
		"module":     "test-authz",
		"path":       "/api/users/:id",
		"permission": "user.read",
		"riskLevel":  "CRITICAL",
		"route":      "/api/users/:id",
		"status":     http.StatusForbidden,
		"targetId":   "user.read",
		"targetName": "user.read",
		"targetType": "permission",
	})
}

func TestSecurityAuditPublisherPublishesMissingTokenAuditEvent(t *testing.T) {
	gin.SetMode(gin.TestMode)

	bus := eventbus.New(nil)
	events := make([]moduleapi.AuditEvent, 0, 1)
	if err := bus.Subscribe(string(moduleapi.AuditRecordEventName), func(_ context.Context, event eventbus.Event) error {
		payload, ok := event.Payload.(moduleapi.AuditEvent)
		if !ok {
			t.Fatalf("expected audit event payload, got %#v", event.Payload)
		}
		events = append(events, payload)
		return nil
	}); err != nil {
		t.Fatalf("subscribe audit event: %v", err)
	}

	authService := testAuthService{
		parseAccessToken: func(context.Context, string) (*moduleapi.AccessTokenClaims, error) {
			t.Fatal("parse access token should not be called without bearer token")
			return nil, nil
		},
		currentUser: func(context.Context) (*moduleapi.CurrentUser, error) {
			t.Fatal("current user should not be called without bearer token")
			return nil, nil
		},
	}
	authorizer := testAuthorizer{
		authorize: func(context.Context, moduleapi.RequestAuthContext, string) error {
			t.Fatal("authorize should not be called without bearer token")
			return nil
		},
	}
	publisher := NewSecurityAuditPublisher(bus, nil, "test-authz")

	recorder := httptest.NewRecorder()
	ctx, engine := gin.CreateTestContext(recorder)
	engine.Use(RequirePermission(newTestLocalizer(), authService, authorizer, "user.read", publisher))
	engine.GET("/api/users/:id", func(inner *gin.Context) {
		inner.Status(http.StatusOK)
	})

	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/users/1", nil)
	engine.HandleContext(ctx)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, recorder.Code)
	}
	if len(events) != 1 {
		t.Fatalf("expected one security audit event, got %d", len(events))
	}

	event := events[0]
	if event.Kind != moduleapi.AuditEventKindSecurity {
		t.Fatalf("expected security event kind, got %q", event.Kind)
	}
	if event.Action != string(securityAuditEventAuthTokenMissing) {
		t.Fatalf("expected token-missing action, got %q", event.Action)
	}
	if event.Operator != nil {
		t.Fatalf("missing-token event should not have an operator, got %#v", event.Operator)
	}
	if event.RequestPath != "/api/users/:id" || event.RequestMethod != http.MethodGet {
		t.Fatalf("expected canonical route/method, got %#v", event)
	}
	if event.ResourceType != "auth" || event.ResourceID != string(securityAuditEventAuthTokenMissing) {
		t.Fatalf("expected auth target resource context, got %#v", event)
	}
	if event.StatusCode != http.StatusUnauthorized || event.Success {
		t.Fatalf("expected failed 401 event, got %#v", event)
	}

	assertSecurityAuditMetadata(t, event.Metadata, map[string]any{
		"component":  "httpx.authz",
		"eventType":  string(securityAuditEventAuthTokenMissing),
		"method":     http.MethodGet,
		"module":     "test-authz",
		"path":       "/api/users/:id",
		"riskLevel":  "CRITICAL",
		"route":      "/api/users/:id",
		"status":     http.StatusUnauthorized,
		"targetId":   string(securityAuditEventAuthTokenMissing),
		"targetName": string(securityAuditEventAuthTokenMissing),
		"targetType": "auth",
	})
}

func assertSecurityAuditMetadata(t *testing.T, metadata map[string]any, want map[string]any) {
	t.Helper()

	for key, wantValue := range want {
		if got := metadata[key]; got != wantValue {
			t.Fatalf("metadata[%q]: expected %#v, got %#v in %#v", key, wantValue, got, metadata)
		}
	}
	for _, key := range []string{"requestId", "traceId"} {
		value, ok := metadata[key].(string)
		if !ok || value == "" {
			t.Fatalf("expected non-empty %s in metadata, got %#v", key, metadata)
		}
	}
	if metadata["requestId"] != metadata["traceId"] {
		t.Fatalf("expected MVP traceId to match requestId, got %#v", metadata)
	}
}

// TestRequirePermissionAllowsAuthenticatedRequestWhenPermissionCodeBlank 验证空权限码只要求建立登录态。
func TestRequirePermissionAllowsAuthenticatedRequestWhenPermissionCodeBlank(t *testing.T) {
	assertAuthenticatedRequestAllowed(t, " ")
}

// TestRequirePermissionAllowsBlankPermissionWithoutAuthorizer 验证空权限码路由不会隐式要求 Authorizer 已注册。
func TestRequirePermissionAllowsBlankPermissionWithoutAuthorizer(t *testing.T) {
	assertAuthenticatedRequestAllowed(t, "")
}

// TestRequirePermissionMapsInvalidTokenToUnauthorized 验证无效 token 会收敛为未登录响应。
func TestRequirePermissionMapsInvalidTokenToUnauthorized(t *testing.T) {
	assertPermissionRejectsTokenError(t, "bad-token", moduleapi.ErrInvalidAccessToken, "auth.token_invalid", "AUTH_TOKEN_INVALID")
}

// TestRequirePermissionMapsExpiredTokenToUnauthorized 验证过期 token 会收敛为稳定的过期响应，
// 以便前端仅对该分支触发 refresh。
func TestRequirePermissionMapsExpiredTokenToUnauthorized(t *testing.T) {
	assertPermissionRejectsTokenError(t, "expired-token", moduleapi.ErrExpiredAccessToken, "auth.token_expired", "AUTH_TOKEN_EXPIRED")
}

// TestRequirePermissionFailsClosedWhenAuthDependenciesMissing 验证未装配 auth 依赖时会拒绝请求而不是继续执行。
func TestRequirePermissionFailsClosedWhenAuthDependenciesMissing(t *testing.T) {
	gin.SetMode(gin.TestMode)
	localizer := newTestLocalizer()

	recorder := httptest.NewRecorder()
	ctx, engine := gin.CreateTestContext(recorder)
	handled := false
	engine.Use(RequirePermission(localizer, nil, nil, "user.read"))
	engine.GET("/api/users/:id", func(inner *gin.Context) {
		handled = true
		inner.Status(http.StatusOK)
	})

	ctx.Request = newBearerRequest("/api/users/1", "token-1")
	engine.HandleContext(ctx)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, recorder.Code)
	}
	if handled {
		t.Fatal("expected request to fail closed before reaching handler")
	}
}

// TestRequirePermissionFailsClosedWhenAuthorizerMissing 验证非空权限码路由缺少授权器时会拒绝请求。
func TestRequirePermissionFailsClosedWhenAuthorizerMissing(t *testing.T) {
	gin.SetMode(gin.TestMode)
	localizer := newTestLocalizer()

	recorder := httptest.NewRecorder()
	ctx, engine := gin.CreateTestContext(recorder)
	handled := false
	engine.Use(RequirePermission(localizer, newAuthenticatedAuthService(), nil, "user.read"))
	engine.GET("/api/users/:id", func(inner *gin.Context) {
		handled = true
		inner.Status(http.StatusOK)
	})

	ctx.Request = newBearerRequest("/api/users/1", "token-1")
	engine.HandleContext(ctx)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, recorder.Code)
	}
	if handled {
		t.Fatal("expected request to fail closed before reaching handler")
	}
}

var (
	_ moduleapi.AuthService = testAuthService{}
	_ moduleapi.Authorizer  = testAuthorizer{}
	_                       = errors.Is
)
