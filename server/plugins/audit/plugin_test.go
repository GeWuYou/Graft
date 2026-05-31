package audit

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"

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

func (r *memoryAuditRepository) ReadAuditOverview(_ context.Context, window store.AuditTimePreset) (store.AuditOverview, error) {
	return store.AuditOverview{
		TimePreset: window,
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

func (r *memoryAuditRepository) ReadIncident(_ context.Context, eventID uint64) (store.AuditIncident, error) {
	for _, item := range r.items {
		if item.ID == eventID {
			return store.AuditIncident{
				SeedEvent: item,
				Incident: store.AuditIncidentSummary{
					IncidentKey:       "incident:req:" + item.RequestID,
					Title:             "Audit incident",
					Summary:           "Seed event drilldown",
					RiskLevel:         store.AuditRiskLevelHigh,
					StartedAt:         item.CreatedAt,
					EndedAt:           item.CreatedAt,
					CorrelationReason: "Correlated by stable request_id first.",
				},
				RelatedEvents: []store.AuditLog{item},
				RelatedRequests: []store.AuditIncidentRequest{
					{
						RequestID:  item.RequestID,
						EventCount: 1,
						StartedAt:  item.CreatedAt,
						EndedAt:    item.CreatedAt,
					},
				},
				MonitorContext: store.AuditIncidentMonitorContext{
					State:  store.MonitorContextStateUnavailable,
					Reason: "Current monitor authority only supports bounded evidence links and short-retention trend context for this incident workflow.",
				},
			}, nil
		}
	}
	return store.AuditIncident{}, store.ErrIncidentNotFound
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

func (failingAuditRepository) ReadAuditOverview(context.Context, store.AuditTimePreset) (store.AuditOverview, error) {
	return store.AuditOverview{}, errors.New("overview failed")
}

func (failingAuditRepository) ReadIncident(context.Context, uint64) (store.AuditIncident, error) {
	return store.AuditIncident{}, errors.New("incident failed")
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
	return newPluginTestContextWithLogger(t, repo, zap.NewNop())
}

func newPluginTestContextWithLogger(t *testing.T, repo store.AuditRepository, logger *zap.Logger) (*plugin.Context, *gin.Engine, eventbus.Bus) {
	t.Helper()

	gin.SetMode(gin.TestMode)
	engine := gin.New()
	bus := eventbus.New(zap.NewNop())
	ctx := &plugin.Context{
		Logger:             logger,
		Config:             &config.Config{},
		I18n:               i18n.MustNew(config.I18nConfig{DefaultLocale: "zh-CN", FallbackLocale: "zh-CN", SupportedLocales: []string{"zh-CN", "en-US"}}),
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

func TestAuditLogsRouteAcceptsCanonicalFilters(t *testing.T) {
	repo := &memoryAuditRepository{}
	_, engine, _ := newPluginTestContext(t, repo)

	request := httptest.NewRequest(http.MethodGet, "/api/audit/logs?keyword=login&actor=alice&session_id=session-1&source=REQUEST", nil)
	request.Header.Set("Authorization", "Bearer token")
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}
}

func TestAuditLogsRouteAcceptsBracketedArrayFilters(t *testing.T) {
	repo := &memoryAuditRepository{}
	_, engine, _ := newPluginTestContext(t, repo)

	request := httptest.NewRequest(
		http.MethodGet,
		"/api/audit/logs?action_keywords[]=delete&action_keywords[]=reset&resource_types[]=auth&resource_types[]=session&results[]=FAILED&risk_levels[]=HIGH",
		nil,
	)
	request.Header.Set("Authorization", "Bearer token")
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
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
			Name:        "request.audit.incidents.exclude",
			Source:      store.AuditSourceRequest,
			Enabled:     true,
			Priority:    5,
			Effect:      store.AuditPolicyEffectExclude,
			Method:      http.MethodGet,
			PathPattern: "/api/audit/incidents/",
			MatchType:   store.AuditPolicyMatchTypePrefix,
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

func TestAuditIncidentEndpointReturnsAuditOwnedIncident(t *testing.T) {
	repo := &memoryAuditRepository{
		items: []store.AuditLog{
			{
				ID:               7,
				Action:           "auth.permission.denied",
				RequestID:        "req-incident-1",
				ActorDisplayName: "Alice",
				ActorUsername:    "alice",
				Success:          false,
				Message:          "common.forbidden",
				CreatedAt:        time.Date(2026, 5, 29, 12, 0, 0, 0, time.UTC),
			},
		},
	}
	_, engine, _ := newPluginTestContext(t, repo)

	request := httptest.NewRequest(http.MethodGet, "/api/audit/incidents/7", nil)
	request.Header.Set("Authorization", "Bearer token")
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}
	body := recorder.Body.String()
	if !strings.Contains(body, "\"incident_key\":\"incident:req:req-incident-1\"") {
		t.Fatalf("expected canonical incident key in response body, got %s", body)
	}
	if !strings.Contains(body, "\"request_id\":\"req-incident-1\"") {
		t.Fatalf("expected stable request id in response body, got %s", body)
	}
}

// TestRegisterSubscribesActiveAuditEvents 验证主动审计事件会通过 event bus
// 订阅路径落入统一仓储。
func TestRegisterSubscribesActiveAuditEvents(t *testing.T) {
	repo := &memoryAuditRepository{}
	_, _, bus := newPluginTestContext(t, repo)

	requestCtx := httpx.WithRequestAuditContext(
		pluginapi.WithRequestAuthContext(context.Background(), pluginapi.RequestAuthContext{
			User: &pluginapi.CurrentUser{ID: 21, Username: "ctx-admin", DisplayName: "Context Admin"},
		}),
		httpx.RequestAuditContext{
			RequestID: "req-domain-1",
			TraceID:   "req-domain-1",
			Route:     "/api/users",
			Method:    http.MethodPost,
			ClientIP:  "203.0.113.10",
			UserAgent: "audit-plugin-test",
		},
	)

	err := bus.Publish(requestCtx, eventbus.Event{
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
	if repo.items[0].RequestID != "req-domain-1" {
		t.Fatalf("expected request id from context, got %#v", repo.items[0])
	}

	var metadata map[string]any
	if err := json.Unmarshal(repo.items[0].Metadata, &metadata); err != nil {
		t.Fatalf("unmarshal metadata: %v", err)
	}
	if metadata["traceId"] != "req-domain-1" {
		t.Fatalf("expected traceId from context, got %#v", metadata)
	}
	if metadata["actorId"] != "9" {
		t.Fatalf("expected explicit operator actorId, got %#v", metadata)
	}
	if metadata["trace_id"] != "req-domain-1" {
		t.Fatalf("expected legacy aliases to remain, got %#v", metadata)
	}
}

func TestRegisterSubscribesActiveAuditEventsFallsBackToRequestAuthActor(t *testing.T) {
	repo := &memoryAuditRepository{}
	_, _, bus := newPluginTestContext(t, repo)

	requestCtx := httpx.WithRequestAuditContext(
		pluginapi.WithRequestAuthContext(context.Background(), pluginapi.RequestAuthContext{
			User: &pluginapi.CurrentUser{ID: 22, Username: "ctx-user", DisplayName: "Context User"},
		}),
		httpx.RequestAuditContext{
			RequestID: "req-domain-2",
			TraceID:   "req-domain-2",
			Route:     "/api/roles/:id/status",
			Method:    http.MethodPost,
			ClientIP:  "203.0.113.22",
			UserAgent: "audit-plugin-test-auth",
		},
	)

	err := bus.Publish(requestCtx, eventbus.Event{
		Name: pluginapi.AuditRecordEventName,
		Payload: pluginapi.AuditEvent{
			Action:       "user.profile.update",
			ResourceType: "user",
			ResourceID:   "22",
			Success:      true,
		},
	})
	if err != nil {
		t.Fatalf("publish audit event: %v", err)
	}
	if len(repo.items) != 1 {
		t.Fatalf("expected one audit record, got %d", len(repo.items))
	}
	if repo.items[0].ActorUserID == nil || *repo.items[0].ActorUserID != 22 {
		t.Fatalf("expected actor id 22 from request auth, got %#v", repo.items[0].ActorUserID)
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

func TestRegisterWarnsWhenSecurityAuditEventIsSkippedByPolicy(t *testing.T) {
	repo := &memoryAuditRepository{
		rules: []store.AuditPolicyRule{
			{
				Name:      "domain.user.profile.update",
				Source:    store.AuditSourceDomainEvent,
				Enabled:   true,
				Priority:  10,
				Effect:    store.AuditPolicyEffectInclude,
				EventType: "user.profile.update",
				MatchType: store.AuditPolicyMatchTypeExact,
			},
		},
	}
	core, observed := observer.New(zap.WarnLevel)
	logger := zap.New(core)
	_, _, bus := newPluginTestContextWithLogger(t, repo, logger)

	err := bus.Publish(context.Background(), eventbus.Event{
		Name: pluginapi.AuditRecordEventName,
		Payload: pluginapi.AuditEvent{
			Kind:         pluginapi.AuditEventKindSecurity,
			Action:       "auth.permission.denied",
			RequestPath:  "/api/roles/12",
			ResourceType: "role",
			ResourceID:   "12",
			Success:      false,
		},
	})
	if err != nil {
		t.Fatalf("publish security audit event: %v", err)
	}
	if len(repo.items) != 0 {
		t.Fatalf("expected skipped security event to avoid persistence, got %d records", len(repo.items))
	}

	entries := observed.FilterMessage("skip security audit candidate by policy").All()
	if len(entries) != 1 {
		t.Fatalf("expected one warn log for skipped security event, got %d", len(entries))
	}
	fields := entries[0].ContextMap()
	if fields["action"] != "auth.permission.denied" || fields["path"] != "/api/roles/12" {
		t.Fatalf("expected warn log to preserve candidate context, got %#v", fields)
	}
}

func TestRegisterSubscribesActiveAuditEventsWithoutHTTPContextDoesNotPanic(t *testing.T) {
	repo := &memoryAuditRepository{}
	_, _, bus := newPluginTestContext(t, repo)

	err := bus.Publish(context.Background(), eventbus.Event{
		Name: pluginapi.AuditRecordEventName,
		Payload: pluginapi.AuditEvent{
			Action:       "user.profile.update",
			ResourceType: "user",
			ResourceID:   "33",
			Success:      true,
		},
	})
	if err != nil {
		t.Fatalf("publish audit event: %v", err)
	}
	if len(repo.items) != 1 {
		t.Fatalf("expected one audit record, got %d", len(repo.items))
	}
	if repo.items[0].RequestID != "" {
		t.Fatalf("expected empty request id without HTTP context, got %#v", repo.items[0])
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

	request := httptest.NewRequest(http.MethodGet, "/api/audit/overview?preset=last_7d", nil)
	request.Header.Set("Authorization", "Bearer token")
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}
	if !strings.Contains(recorder.Body.String(), `"time_preset":"last_7d"`) {
		t.Fatalf("expected overview preset in response, got %s", recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `"failed_auth"`) {
		t.Fatalf("expected failed_auth in response, got %s", recorder.Body.String())
	}
}
