package audit

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"graft/server/internal/config"
	"graft/server/internal/container"
	"graft/server/internal/cronx"
	"graft/server/internal/eventbus"
	"graft/server/internal/httpx"
	"graft/server/internal/i18n"
	"graft/server/internal/menu"
	"graft/server/internal/permission"
	"graft/server/internal/plugin"
	"graft/server/internal/pluginapi"
	"graft/server/plugins/audit/store"
)

type memoryAuditRepository struct {
	items []store.AuditLog
	rules []store.AuditPolicyRule
}

func (r *memoryAuditRepository) CreateAuditLog(_ context.Context, input store.CreateAuditLogInput) (store.AuditLog, error) {
	record := store.AuditLog{
		ID:               uint64(len(r.items) + 1),
		ActorUserID:      input.ActorUserID,
		ActorUsername:    input.ActorUsername,
		ActorDisplayName: input.ActorDisplayName,
		Action:           input.Action,
		ResourceType:     input.ResourceType,
		ResourceID:       input.ResourceID,
		ResourceName:     input.ResourceName,
		Success:          input.Success,
		RequestID:        input.RequestID,
		IP:               input.IP,
		UserAgent:        input.UserAgent,
		Message:          input.Message,
		Metadata:         input.Metadata,
		CreatedAt:        input.CreatedAt,
	}
	r.items = append(r.items, record)
	return record, nil
}

func (r *memoryAuditRepository) ListAuditLogs(_ context.Context, _ store.ListAuditLogsQuery) (store.ListAuditLogsResult, error) {
	return store.ListAuditLogsResult{Items: append([]store.AuditLog(nil), r.items...), Total: len(r.items)}, nil
}

func (r *memoryAuditRepository) ReadAuditOverview(_ context.Context, window store.OverviewWindow) (store.AuditOverview, error) {
	return store.AuditOverview{
		Window: window,
		Summary: store.OverviewSummary{
			TotalLogs:           len(r.items),
			FailedOperations:    1,
			HighRiskEvents:      2,
			SensitiveOperations: 1,
		},
		FailedAuth: []store.OverviewItem{
			{
				ID:        1,
				Action:    "POST /api/auth/login",
				RequestID: "req-auth",
				Success:   false,
				CreatedAt: time.Now().UTC(),
			},
		},
		PermissionDenied: []store.OverviewItem{
			{
				ID:        2,
				Action:    "rbac.role.delete",
				RequestID: "req-role",
				Success:   false,
				CreatedAt: time.Now().UTC(),
			},
		},
		SensitiveOps: []store.OverviewItem{
			{
				ID:           3,
				Action:       "user.password.reset",
				ResourceType: "user",
				ResourceID:   "42",
				ResourceName: "alice",
				RequestID:    "req-user",
				Success:      true,
				CreatedAt:    time.Now().UTC(),
			},
		},
	}, nil
}

func (r *memoryAuditRepository) ListAuditPolicyRules(_ context.Context) ([]store.AuditPolicyRule, error) {
	if len(r.rules) == 0 {
		return defaultPluginTestPolicyRules(), nil
	}
	return append([]store.AuditPolicyRule(nil), r.rules...), nil
}

type failingAuditRepository struct{}

func (failingAuditRepository) CreateAuditLog(context.Context, store.CreateAuditLogInput) (store.AuditLog, error) {
	return store.AuditLog{}, errors.New("write failed")
}

func (failingAuditRepository) ListAuditLogs(context.Context, store.ListAuditLogsQuery) (store.ListAuditLogsResult, error) {
	return store.ListAuditLogsResult{}, nil
}

func (failingAuditRepository) ReadAuditOverview(context.Context, store.OverviewWindow) (store.AuditOverview, error) {
	return store.AuditOverview{}, errors.New("overview failed")
}

func (failingAuditRepository) ListAuditPolicyRules(context.Context) ([]store.AuditPolicyRule, error) {
	return defaultPluginTestPolicyRules(), nil
}

type stubAuthService struct {
	user pluginapi.CurrentUser
}

func (s stubAuthService) CurrentUser(ctx context.Context) (*pluginapi.CurrentUser, error) {
	auth, ok := pluginapi.RequestAuthContextFromContext(ctx)
	if !ok || auth.Claims == nil {
		return nil, pluginapi.ErrUnauthenticated
	}

	user := s.user
	return &user, nil
}

func (s stubAuthService) ParseAccessToken(_ context.Context, token string) (*pluginapi.AccessTokenClaims, error) {
	if token == "" {
		return nil, pluginapi.ErrInvalidAccessToken
	}

	return &pluginapi.AccessTokenClaims{
		UserID:       s.user.ID,
		SessionID:    "session-1",
		TokenVersion: 1,
		ExpiresAt:    time.Now().UTC().Add(time.Minute),
		IssuedAt:     time.Now().UTC(),
	}, nil
}

type allowAuthorizer struct{}

func (allowAuthorizer) Authorize(_ context.Context, _ pluginapi.RequestAuthContext, _ string) error {
	return nil
}

type denyAuthorizer struct{}

func (denyAuthorizer) Authorize(_ context.Context, _ pluginapi.RequestAuthContext, _ string) error {
	return pluginapi.ErrPermissionDenied
}

func newPluginTestContext(t *testing.T, repo store.AuditRepository) (*plugin.Context, *gin.Engine, eventbus.Bus) {
	t.Helper()

	gin.SetMode(gin.TestMode)
	engine := gin.New()
	bus := eventbus.New(zap.NewNop())
	ctx := &plugin.Context{
		Logger:             zap.NewNop(),
		Config:             &config.Config{},
		I18n:               i18n.New(config.I18nConfig{DefaultLocale: "zh-CN", FallbackLocale: "zh-CN", SupportedLocales: []string{"zh-CN", "en-US"}}),
		EventBus:           bus,
		Router:             engine.Group("/api"),
		Services:           container.New(),
		MenuRegistry:       menu.NewRegistry(),
		PermissionRegistry: permission.NewRegistry(),
		CronRegistry:       cronx.NewRegistry(),
	}

	if err := ctx.Services.RegisterSingleton((*pluginapi.AuthService)(nil), func(container.Resolver) (any, error) {
		return stubAuthService{user: pluginapi.CurrentUser{ID: 7, Username: "alice", DisplayName: "Alice"}}, nil
	}); err != nil {
		t.Fatalf("register auth service: %v", err)
	}
	if err := ctx.Services.RegisterSingleton((*pluginapi.Authorizer)(nil), func(container.Resolver) (any, error) {
		return allowAuthorizer{}, nil
	}); err != nil {
		t.Fatalf("register authorizer: %v", err)
	}

	pluginInstance, err := NewPlugin(repo)
	if err != nil {
		t.Fatalf("build audit plugin: %v", err)
	}
	if err := pluginInstance.Register(ctx); err != nil {
		t.Fatalf("register audit plugin: %v", err)
	}

	return ctx, engine, bus
}

// TestRequestAuditMiddlewareSkipsUnmatchedRequest 验证未命中策略的普通请求不会落库。
func TestRequestAuditMiddlewareSkipsUnmatchedRequest(t *testing.T) {
	repo := &memoryAuditRepository{}
	ctx, engine, _ := newPluginTestContext(t, repo)
	authService := stubAuthService{user: pluginapi.CurrentUser{ID: 7, Username: "alice", DisplayName: "Alice"}}
	authorizer := allowAuthorizer{}

	ctx.Router.GET("/users/:id", httpx.RequirePermission(ctx.I18n, authService, authorizer, "user.read"), func(ginCtx *gin.Context) {
		ginCtx.JSON(http.StatusOK, gin.H{"ok": true})
	})

	request := httptest.NewRequest(http.MethodGet, "/api/users/42", nil)
	request.Header.Set("Authorization", "Bearer token")
	request.Header.Set("User-Agent", "audit-test")
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}
	if len(repo.items) != 0 {
		t.Fatalf("expected request to be skipped by policy, got %d audit records", len(repo.items))
	}
}

// TestRequestAuditMiddlewareCapturesLocalizedErrorKey 验证失败请求会把统一错误
// 响应的稳定 message key 收敛为审计错误信息。
func TestRequestAuditMiddlewareCapturesLocalizedErrorKey(t *testing.T) {
	repo := &memoryAuditRepository{}
	ctx, engine, _ := newPluginTestContext(t, repo)

	ctx.Router.POST("/auth/login", func(ginCtx *gin.Context) {
		httpx.WriteLocalizedError(ginCtx, ctx.I18n, http.StatusBadRequest, "common.invalid_argument", nil)
	})

	request := httptest.NewRequest(http.MethodPost, "/api/auth/login", nil)
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", recorder.Code)
	}
	if len(repo.items) != 1 {
		t.Fatalf("expected one audit record, got %d", len(repo.items))
	}
	if repo.items[0].Success {
		t.Fatal("expected failed request audit record")
	}
	if repo.items[0].Message != "common.invalid_argument" {
		t.Fatalf("expected stable error message key, got %q", repo.items[0].Message)
	}
}

func TestRequirePermissionPublishesSecurityAuditEvent(t *testing.T) {
	repo := &memoryAuditRepository{}
	ctx, engine, _ := newPluginTestContext(t, repo)

	ctx.Router.GET(
		"/roles",
		httpx.RequirePermission(
			ctx.I18n,
			stubAuthService{user: pluginapi.CurrentUser{ID: 7, Username: "alice", DisplayName: "Alice"}},
			denyAuthorizer{},
			"rbac.role.read",
			httpx.NewSecurityAuditPublisher(ctx.EventBus, ctx.Logger, "test"),
		),
		func(ginCtx *gin.Context) {
			ginCtx.JSON(http.StatusOK, gin.H{"ok": true})
		},
	)

	request := httptest.NewRequest(http.MethodGet, "/api/roles", nil)
	request.Header.Set("Authorization", "Bearer token")
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d", recorder.Code)
	}
	if len(repo.items) != 1 {
		t.Fatalf("expected one security audit record, got %d", len(repo.items))
	}
	if repo.items[0].Action != "auth.permission.denied" {
		t.Fatalf("expected permission denied audit action, got %q", repo.items[0].Action)
	}
}

func defaultPluginTestPolicyRules() []store.AuditPolicyRule {
	return []store.AuditPolicyRule{
		{
			Name:        "request.healthz.exclude",
			Source:      store.AuditSourceRequest,
			Enabled:     true,
			Priority:    1,
			Effect:      store.AuditPolicyEffectExclude,
			Method:      http.MethodGet,
			PathPattern: "/healthz",
			MatchType:   store.AuditPolicyMatchTypeExact,
		},
		{
			Name:        "request.monitor.exclude",
			Source:      store.AuditSourceRequest,
			Enabled:     true,
			Priority:    2,
			Effect:      store.AuditPolicyEffectExclude,
			Method:      http.MethodGet,
			PathPattern: "/api/monitor",
			MatchType:   store.AuditPolicyMatchTypePrefix,
		},
		{
			Name:        "request.audit.overview.exclude",
			Source:      store.AuditSourceRequest,
			Enabled:     true,
			Priority:    3,
			Effect:      store.AuditPolicyEffectExclude,
			Method:      http.MethodGet,
			PathPattern: "/api/audit/overview",
			MatchType:   store.AuditPolicyMatchTypeExact,
		},
		{
			Name:        "request.audit.logs.exclude",
			Source:      store.AuditSourceRequest,
			Enabled:     true,
			Priority:    4,
			Effect:      store.AuditPolicyEffectExclude,
			Method:      http.MethodGet,
			PathPattern: "/api/audit/logs",
			MatchType:   store.AuditPolicyMatchTypeExact,
		},
		{
			Name:      "security.auth.permission_denied",
			Source:    store.AuditSourceSecurityEvent,
			Enabled:   true,
			Priority:  10,
			Effect:    store.AuditPolicyEffectInclude,
			EventType: "auth.permission.denied",
			MatchType: store.AuditPolicyMatchTypeExact,
		},
		{
			Name:        "request.auth.login",
			Source:      store.AuditSourceRequest,
			Enabled:     true,
			Priority:    20,
			Effect:      store.AuditPolicyEffectInclude,
			Method:      http.MethodPost,
			PathPattern: "/api/auth/login",
			MatchType:   store.AuditPolicyMatchTypeExact,
		},
		{
			Name:      "domain.user.password.reset",
			Source:    store.AuditSourceDomainEvent,
			Enabled:   true,
			Priority:  30,
			Effect:    store.AuditPolicyEffectInclude,
			EventType: "user.password.reset",
			MatchType: store.AuditPolicyMatchTypeExact,
		},
		{
			Name:      "domain.user.profile.update",
			Source:    store.AuditSourceDomainEvent,
			Enabled:   true,
			Priority:  31,
			Effect:    store.AuditPolicyEffectInclude,
			EventType: "user.profile.update",
			MatchType: store.AuditPolicyMatchTypeExact,
		},
	}
}

// TestRegisterSubscribesActiveAuditEvents 验证主动审计事件会通过 event bus
// 订阅路径落入统一仓储。
func TestRegisterSubscribesActiveAuditEvents(t *testing.T) {
	repo := &memoryAuditRepository{}
	_, _, bus := newPluginTestContext(t, repo)

	err := bus.Publish(context.Background(), eventbus.Event{
		Name: pluginapi.AuditRecordEventName,
		Payload: pluginapi.AuditEvent{
			Operator:     &pluginapi.CurrentUser{ID: 9, Username: "bob"},
			Action:       "user.password.reset",
			ResourceType: "user",
			ResourceID:   "9",
			Success:      true,
		},
	})
	if err != nil {
		t.Fatalf("publish audit event: %v", err)
	}
	if len(repo.items) != 1 {
		t.Fatalf("expected one audit record, got %d", len(repo.items))
	}
	if repo.items[0].Action != "user.password.reset" {
		t.Fatalf("expected active audit action to be preserved, got %q", repo.items[0].Action)
	}
	if repo.items[0].ActorUserID == nil || *repo.items[0].ActorUserID != 9 {
		t.Fatalf("expected actor id 9, got %#v", repo.items[0].ActorUserID)
	}
}

// TestRegisterSubscribesActiveAuditEventPointers 验证主动审计事件同时兼容值类型和指针类型载荷。
func TestRegisterSubscribesActiveAuditEventPointers(t *testing.T) {
	repo := &memoryAuditRepository{}
	_, _, bus := newPluginTestContext(t, repo)

	err := bus.Publish(context.Background(), eventbus.Event{
		Name: pluginapi.AuditRecordEventName,
		Payload: &pluginapi.AuditEvent{
			Operator:     &pluginapi.CurrentUser{ID: 10, Username: "carol"},
			Action:       "user.profile.update",
			ResourceType: "user",
			ResourceID:   "10",
			Success:      true,
		},
	})
	if err != nil {
		t.Fatalf("publish audit event pointer payload: %v", err)
	}
	if len(repo.items) != 1 {
		t.Fatalf("expected one audit record, got %d", len(repo.items))
	}
	if repo.items[0].Action != "user.profile.update" {
		t.Fatalf("expected pointer payload action to be preserved, got %q", repo.items[0].Action)
	}
}

func TestRegisterSwallowsActiveAuditWriteErrors(t *testing.T) {
	ctx, _, bus := newPluginTestContext(t, failingAuditRepository{})

	if err := ctx.EventBus.Subscribe("noop", func(context.Context, eventbus.Event) error { return nil }); err != nil {
		t.Fatalf("subscribe noop: %v", err)
	}

	err := bus.Publish(context.Background(), eventbus.Event{
		Name: pluginapi.AuditRecordEventName,
		Payload: pluginapi.AuditEvent{
			Operator:     &pluginapi.CurrentUser{ID: 10, Username: "carol"},
			Action:       "user.profile.update",
			ResourceType: "user",
			ResourceID:   "10",
			Success:      true,
		},
	})
	if err != nil {
		t.Fatalf("expected active audit failure to be swallowed, got %v", err)
	}
}

func TestAuditReadRoutesStayOutOfAuditLogByPolicy(t *testing.T) {
	repo := &memoryAuditRepository{}
	_, engine, _ := newPluginTestContext(t, repo)

	for _, path := range []string{"/api/audit/logs", "/api/audit/overview?window=7d"} {
		request := httptest.NewRequest(http.MethodGet, path, nil)
		request.Header.Set("Authorization", "Bearer token")
		recorder := httptest.NewRecorder()
		engine.ServeHTTP(recorder, request)

		if recorder.Code != http.StatusOK {
			t.Fatalf("expected status 200 for %s, got %d", path, recorder.Code)
		}
	}

	if len(repo.items) != 0 {
		t.Fatalf("expected audit read routes to be excluded by policy, got %d records", len(repo.items))
	}
}

func TestRegisterRecordsRBACDomainEventWhenPolicyAllows(t *testing.T) {
	repo := &memoryAuditRepository{
		rules: append(defaultPluginTestPolicyRules(), store.AuditPolicyRule{
			Name:      "domain.rbac.role.delete",
			Source:    store.AuditSourceDomainEvent,
			Enabled:   true,
			Priority:  40,
			Effect:    store.AuditPolicyEffectInclude,
			EventType: "rbac.role.delete",
			MatchType: store.AuditPolicyMatchTypeExact,
		}),
	}
	_, _, bus := newPluginTestContext(t, repo)

	err := bus.Publish(context.Background(), eventbus.Event{
		Name: pluginapi.AuditRecordEventName,
		Payload: pluginapi.AuditEvent{
			Kind:         pluginapi.AuditEventKindDomain,
			Operator:     &pluginapi.CurrentUser{ID: 9, Username: "bob"},
			Action:       "rbac.role.delete",
			ResourceType: "role",
			ResourceID:   "12",
			ResourceName: "ops-admin",
			Success:      true,
		},
	})
	if err != nil {
		t.Fatalf("publish audit event: %v", err)
	}
	if len(repo.items) != 1 {
		t.Fatalf("expected one rbac audit record, got %d", len(repo.items))
	}
	if repo.items[0].Action != "rbac.role.delete" {
		t.Fatalf("expected rbac role delete action, got %q", repo.items[0].Action)
	}
}

func TestRegisterExposesAuditReadSurface(t *testing.T) {
	repo := &memoryAuditRepository{}
	ctx, engine, _ := newPluginTestContext(t, repo)

	foundPermission := false
	for _, item := range ctx.PermissionRegistry.Items() {
		if item.Code == "audit.read" {
			foundPermission = true
			break
		}
	}
	if !foundPermission {
		t.Fatal("expected audit.read permission to be registered")
	}

	items := ctx.MenuRegistry.Items()
	if len(items) != 3 {
		t.Fatalf("expected 3 audit menu items, got %#v", items)
	}
	if items[0].Path != "/audit" || items[0].TitleKey != "menu.audit.title" || items[0].Order != 200 {
		t.Fatalf("unexpected audit root menu: %#v", items[0])
	}
	if items[1].Path != "/audit/overview" || items[1].TitleKey != "menu.audit.overview.title" || items[1].Order != 201 {
		t.Fatalf("unexpected audit overview menu: %#v", items[1])
	}
	if items[2].Path != "/audit/logs" || items[2].TitleKey != "menu.audit.logs.title" || items[2].Order != 202 {
		t.Fatalf("unexpected audit logs menu: %#v", items[2])
	}

	request := httptest.NewRequest(http.MethodGet, "/api/audit/logs", nil)
	request.Header.Set("Authorization", "Bearer token")
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}
}

func TestAuditOverviewRouteReturnsPayload(t *testing.T) {
	repo := &memoryAuditRepository{}
	_, engine, _ := newPluginTestContext(t, repo)

	request := httptest.NewRequest(http.MethodGet, "/api/audit/overview?window=7d", nil)
	request.Header.Set("Authorization", "Bearer token")
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}
	if !strings.Contains(recorder.Body.String(), `"window":"7d"`) {
		t.Fatalf("expected overview window in response, got %s", recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `"failed_auth"`) {
		t.Fatalf("expected failed_auth in response, got %s", recorder.Body.String())
	}
}
