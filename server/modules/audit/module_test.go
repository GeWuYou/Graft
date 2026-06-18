// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

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
	"graft/server/internal/configregistry"
	"graft/server/internal/container"
	"graft/server/internal/cronx"
	"graft/server/internal/dashboard"
	"graft/server/internal/drilldown"
	"graft/server/internal/eventbus"
	"graft/server/internal/httpx"
	"graft/server/internal/i18n"
	"graft/server/internal/menu"
	"graft/server/internal/module"
	"graft/server/internal/moduleapi"
	"graft/server/internal/permission"
	"graft/server/internal/testassert"
	auditcontract "graft/server/modules/audit/contract"
	auditlocales "graft/server/modules/audit/locales"
	"graft/server/modules/audit/store"
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

func (r *memoryAuditRepository) ReadAuditLog(_ context.Context, id uint64) (store.AuditLog, error) {
	for _, item := range r.items {
		if item.ID == id {
			return item, nil
		}
	}
	return store.AuditLog{}, store.ErrAuditLogNotFound
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
		RiskGroups: []store.OverviewRiskGroup{
			{
				Key:   string(store.AuditBusinessCategoryAuthFailures),
				Count: 4,
			},
			{
				Key:   string(store.AuditBusinessCategoryPermissionDenials),
				Count: 3,
			},
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
		return defaultModuleTestPolicyRules(), nil
	}
	return append([]store.AuditPolicyRule(nil), r.rules...), nil
}

func (r *memoryAuditRepository) DeleteAuditLogsBefore(_ context.Context, createdBefore time.Time) (int64, error) {
	kept := r.items[:0]
	deleted := int64(0)
	for _, item := range r.items {
		if item.CreatedAt.Before(createdBefore) {
			deleted++
			continue
		}
		kept = append(kept, item)
	}
	r.items = kept
	return deleted, nil
}

type failingAuditRepository struct{}

func (failingAuditRepository) CreateAuditLog(context.Context, store.CreateAuditLogInput) (store.AuditLog, error) {
	return store.AuditLog{}, errors.New("write failed")
}

func (failingAuditRepository) ListAuditLogs(context.Context, store.ListAuditLogsQuery) (store.ListAuditLogsResult, error) {
	return store.ListAuditLogsResult{}, nil
}

func (failingAuditRepository) ReadAuditLog(context.Context, uint64) (store.AuditLog, error) {
	return store.AuditLog{}, errors.New("detail failed")
}

func (failingAuditRepository) ReadAuditOverview(context.Context, store.AuditTimePreset) (store.AuditOverview, error) {
	return store.AuditOverview{}, errors.New("overview failed")
}

func (failingAuditRepository) ReadIncident(context.Context, uint64) (store.AuditIncident, error) {
	return store.AuditIncident{}, errors.New("incident failed")
}

func (failingAuditRepository) ListAuditPolicyRules(context.Context) ([]store.AuditPolicyRule, error) {
	return defaultModuleTestPolicyRules(), nil
}

func (failingAuditRepository) DeleteAuditLogsBefore(context.Context, time.Time) (int64, error) {
	return 0, errors.New("delete failed")
}

type stubAuthService struct {
	user moduleapi.CurrentUser
}

func (s stubAuthService) CurrentUser(ctx context.Context) (*moduleapi.CurrentUser, error) {
	auth, ok := moduleapi.RequestAuthContextFromContext(ctx)
	if !ok || auth.Claims == nil {
		return nil, moduleapi.ErrUnauthenticated
	}

	user := s.user
	return &user, nil
}

func (s stubAuthService) ParseAccessToken(_ context.Context, token string) (*moduleapi.AccessTokenClaims, error) {
	if token == "" {
		return nil, moduleapi.ErrInvalidAccessToken
	}

	return &moduleapi.AccessTokenClaims{
		UserID:       s.user.ID,
		SessionID:    "session-1",
		TokenVersion: 1,
		ExpiresAt:    time.Now().UTC().Add(time.Minute),
		IssuedAt:     time.Now().UTC(),
	}, nil
}

type allowAuthorizer struct{}

func (allowAuthorizer) Authorize(_ context.Context, _ moduleapi.RequestAuthContext, _ string) error {
	return nil
}

type denyAuthorizer struct{}

func (denyAuthorizer) Authorize(_ context.Context, _ moduleapi.RequestAuthContext, _ string) error {
	return moduleapi.ErrPermissionDenied
}

func newModuleTestContext(t *testing.T, repo store.AuditRepository) (*module.Context, *gin.Engine, eventbus.Bus) {
	return newModuleTestContextWithLogger(t, repo, zap.NewNop())
}

func newModuleTestContextWithLogger(t *testing.T, repo store.AuditRepository, logger *zap.Logger) (*module.Context, *gin.Engine, eventbus.Bus) {
	t.Helper()

	gin.SetMode(gin.TestMode)
	engine := gin.New()
	bus := eventbus.New(zap.NewNop())
	localizer := i18n.MustNew(config.I18nConfig{DefaultLocale: "zh-CN", FallbackLocale: "zh-CN", SupportedLocales: []string{"zh-CN", "en-US"}})
	resources, err := auditlocales.EmbeddedLocaleResources()
	if err != nil {
		t.Fatalf("load audit locale resources: %v", err)
	}
	if err := localizer.RegisterEmbeddedLocaleResources(resources); err != nil {
		t.Fatalf("register audit locale resources: %v", err)
	}
	ctx := &module.Context{
		Logger:             logger,
		Config:             &config.Config{Audit: config.AuditConfig{LogRetention: 30 * 24 * time.Hour}},
		I18n:               localizer,
		EventBus:           bus,
		Router:             engine.Group("/api"),
		Services:           container.New(),
		MenuRegistry:       menu.NewRegistry(),
		PermissionRegistry: permission.NewRegistry(),
		CronRegistry:       cronx.NewRegistry(),
		ConfigRegistry:     configregistry.NewRegistry(),
		DashboardRegistry:  dashboard.NewRegistry(),
	}

	if err := ctx.Services.RegisterSingleton((*moduleapi.AuthService)(nil), func(container.Resolver) (any, error) {
		return stubAuthService{user: moduleapi.CurrentUser{ID: 7, Username: "alice", DisplayName: "Alice"}}, nil
	}); err != nil {
		t.Fatalf("register auth service: %v", err)
	}
	if err := ctx.Services.RegisterSingleton((*moduleapi.Authorizer)(nil), func(container.Resolver) (any, error) {
		return allowAuthorizer{}, nil
	}); err != nil {
		t.Fatalf("register authorizer: %v", err)
	}

	moduleInstance, err := NewModule(repo)
	if err != nil {
		t.Fatalf("build audit module: %v", err)
	}
	if err := moduleInstance.Register(ctx); err != nil {
		t.Fatalf("register audit module: %v", err)
	}

	return ctx, engine, bus
}

type stubScopeMetadataRepo struct {
	metadata map[string]drilldown.ScopeMetadata
}

func (r stubScopeMetadataRepo) GetScope(_ context.Context, module, scope string) (drilldown.ScopeMetadata, error) {
	if metadata, ok := r.metadata[module+":"+scope]; ok {
		return metadata, nil
	}
	return drilldown.ScopeMetadata{}, drilldown.ErrScopeNotFound
}

func newModuleTestContextWithDrilldown(
	t *testing.T,
	repo store.AuditRepository,
	scopes []string,
) (*module.Context, *gin.Engine, eventbus.Bus) {
	t.Helper()

	gin.SetMode(gin.TestMode)
	engine := gin.New()
	bus := eventbus.New(zap.NewNop())
	localizer := i18n.MustNew(config.I18nConfig{DefaultLocale: "zh-CN", FallbackLocale: "zh-CN", SupportedLocales: []string{"zh-CN", "en-US"}})
	resources, err := auditlocales.EmbeddedLocaleResources()
	if err != nil {
		t.Fatalf("load audit locale resources: %v", err)
	}
	if err := localizer.RegisterEmbeddedLocaleResources(resources); err != nil {
		t.Fatalf("register audit locale resources: %v", err)
	}
	ctx := &module.Context{
		Logger:             zap.NewNop(),
		Config:             &config.Config{Audit: config.AuditConfig{LogRetention: 30 * 24 * time.Hour}},
		I18n:               localizer,
		EventBus:           bus,
		Router:             engine.Group("/api"),
		Services:           container.New(),
		MenuRegistry:       menu.NewRegistry(),
		PermissionRegistry: permission.NewRegistry(),
		CronRegistry:       cronx.NewRegistry(),
		ConfigRegistry:     configregistry.NewRegistry(),
		DashboardRegistry:  dashboard.NewRegistry(),
	}

	if err := ctx.Services.RegisterSingleton((*moduleapi.AuthService)(nil), func(container.Resolver) (any, error) {
		return stubAuthService{user: moduleapi.CurrentUser{ID: 7, Username: "alice", DisplayName: "Alice"}}, nil
	}); err != nil {
		t.Fatalf("register auth service: %v", err)
	}
	if err := ctx.Services.RegisterSingleton((*moduleapi.Authorizer)(nil), func(container.Resolver) (any, error) {
		return allowAuthorizer{}, nil
	}); err != nil {
		t.Fatalf("register authorizer: %v", err)
	}

	scopeMetadata := make(map[string]drilldown.ScopeMetadata, len(scopes))
	for index, scope := range scopes {
		scopeMetadata["audit:"+scope] = drilldown.ScopeMetadata{
			ID:           uint64(index + 1),
			Module:       "audit",
			Scope:        scope,
			Name:         scope,
			Description:  "test scope",
			TargetType:   "log_query",
			TargetModule: "audit",
			TargetPage:   "audit_logs",
			Enabled:      true,
			SortOrder:    index + 1,
		}
	}

	drilldownService, err := drilldown.NewService[ListQuery, ListQuery](
		stubScopeMetadataRepo{metadata: scopeMetadata},
		newAuditScopeResolver(),
	)
	if err != nil {
		t.Fatalf("build drilldown service: %v", err)
	}

	moduleInstance, err := NewModuleWithDrilldown(repo, drilldownService)
	if err != nil {
		t.Fatalf("build audit module with drilldown: %v", err)
	}
	if err := moduleInstance.Register(ctx); err != nil {
		t.Fatalf("register audit module: %v", err)
	}

	return ctx, engine, bus
}

func TestRegisterRegistersAuditRiskEventsDashboardWidget(t *testing.T) {
	ctx, _, _ := newModuleTestContext(t, &memoryAuditRepository{})

	widget, ok := ctx.DashboardRegistry.Get(auditRiskEventsWidgetID)
	if !ok {
		t.Fatalf("expected audit risk events dashboard widget to be registered")
	}
	if widget.ModuleKey != moduleID {
		t.Fatalf("expected module key %q, got %q", moduleID, widget.ModuleKey)
	}
	if widget.Type != dashboard.WidgetTypeAlertList {
		t.Fatalf("expected alert-list widget, got %q", widget.Type)
	}
	if widget.RouteLocation != "/audit/overview" {
		t.Fatalf("expected audit overview route, got %q", widget.RouteLocation)
	}
	if len(widget.RequiredPermissions) != 1 || widget.RequiredPermissions[0] != "audit.read" {
		t.Fatalf("unexpected required permissions: %#v", widget.RequiredPermissions)
	}

	quickLinks := ctx.DashboardRegistry.QuickLinks()
	if len(quickLinks) != 2 {
		t.Fatalf("expected audit quick links, got %#v", quickLinks)
	}
	assertAuditQuickLink(t, quickLinks[0], auditOverviewQuickLinkID, auditcontract.AuditOverviewMenuTitle.String(), auditcontract.AuditOverviewMenuPath, auditOverviewQuickLinkOrder)
	assertAuditQuickLink(t, quickLinks[1], auditLogsQuickLinkID, auditcontract.AuditLogMenuTitle.String(), auditcontract.AuditLogsMenuPath, auditLogsQuickLinkOrder)
}

func assertAuditQuickLink(t *testing.T, link dashboard.QuickLinkDefinition, id string, titleKey string, routeLocation string, order int) {
	t.Helper()
	if link.ID != id ||
		link.ModuleKey != moduleID ||
		link.TitleKey != titleKey ||
		link.RouteLocation != routeLocation ||
		link.Order != order {
		t.Fatalf("unexpected audit quick link: %#v", link)
	}
	if len(link.RequiredPermissions) != 1 || link.RequiredPermissions[0] != auditcontract.AuditReadPermission.String() {
		t.Fatalf("unexpected audit quick link permissions: %#v", link.RequiredPermissions)
	}
}

type authOnlyOverviewRepository struct {
	memoryAuditRepository
}

func (r *authOnlyOverviewRepository) ReadAuditOverview(_ context.Context, window store.AuditTimePreset) (store.AuditOverview, error) {
	return store.AuditOverview{
		TimePreset: window,
		Summary: store.OverviewSummary{
			TotalLogs:           1,
			FailedOperations:    1,
			HighRiskEvents:      1,
			SensitiveOperations: 0,
		},
		RiskGroups: []store.OverviewRiskGroup{
			{
				Key:   string(store.AuditBusinessCategoryAuthFailures),
				Count: 1,
			},
		},
		FailedAuth: []store.OverviewItem{
			{
				ID:        1,
				Action:    "auth.token.expired",
				RequestID: "req-auth",
				Success:   false,
				Message:   "auth.token_expired",
				CreatedAt: time.Now().UTC(),
			},
		},
	}, nil
}

func TestAuditRiskEventsDashboardWidgetLoadsRiskPayload(t *testing.T) {
	ctx, _, _ := newModuleTestContext(t, &memoryAuditRepository{})
	widget, ok := ctx.DashboardRegistry.Get(auditRiskEventsWidgetID)
	if !ok {
		t.Fatalf("expected audit risk events dashboard widget to be registered")
	}

	payload, err := widget.Loader.Load(context.Background(), dashboard.WidgetRequest{})
	if err != nil {
		t.Fatalf("load audit risk events widget: %v", err)
	}
	items, ok := payload["items"].([]map[string]any)
	if !ok {
		t.Fatalf("expected alert-list items payload, got %#v", payload["items"])
	}
	if len(items) == 0 {
		t.Fatalf("expected risk events payload to include attention items")
	}
	itemsByID := map[string]map[string]any{}
	for _, item := range items {
		id, _ := item["id"].(string)
		itemsByID[id] = item
	}
	if itemsByID["audit.high-risk"]["route_location"] != "/audit/logs?preset=last_24h&risk_levels=HIGH%2CCRITICAL" {
		t.Fatalf("expected high risk item to drill into audit logs, got %#v", itemsByID["audit.high-risk"])
	}
	if itemsByID["audit.failed-operations"]["route_location"] != "/audit/logs?preset=last_24h&results=FAILED%2CDENIED%2CERROR" {
		t.Fatalf("expected failed operations item to drill into audit logs, got %#v", itemsByID["audit.failed-operations"])
	}
	if itemsByID["audit.failed-auth"]["route_location"] != "/audit/logs?business_category=auth_failures&preset=last_24h" {
		t.Fatalf("expected failed auth item to drill into audit scope, got %#v", itemsByID["audit.failed-auth"])
	}
	if itemsByID["audit.high-risk"]["count"] != 2 {
		t.Fatalf("expected high risk count to come from overview summary, got %#v", itemsByID["audit.high-risk"])
	}
	if itemsByID["audit.failed-operations"]["count"] != 1 {
		t.Fatalf("expected failed operations count to come from overview summary, got %#v", itemsByID["audit.failed-operations"])
	}
	if itemsByID["audit.failed-auth"]["count"] != 4 {
		t.Fatalf("expected failed auth count to come from risk group count, got %#v", itemsByID["audit.failed-auth"])
	}
	if _, ok := itemsByID["audit.permission-denied"]; ok {
		t.Fatalf("permission denied is no longer a dashboard security entry, got %#v", itemsByID["audit.permission-denied"])
	}
}

func TestAuditRiskEventsDashboardWidgetDoesNotFallbackPermissionDenialsToAuthFailures(t *testing.T) {
	ctx, _, _ := newModuleTestContext(t, &authOnlyOverviewRepository{})
	widget, ok := ctx.DashboardRegistry.Get(auditRiskEventsWidgetID)
	if !ok {
		t.Fatalf("expected audit risk events dashboard widget to be registered")
	}

	payload, err := widget.Loader.Load(context.Background(), dashboard.WidgetRequest{})
	if err != nil {
		t.Fatalf("load audit risk events widget: %v", err)
	}
	items, ok := payload["items"].([]map[string]any)
	if !ok {
		t.Fatalf("expected alert-list items payload, got %#v", payload["items"])
	}
	for _, item := range items {
		if item["id"] == "audit.permission-denied" {
			t.Fatalf("permission denial item must not be synthesized from auth failures: %#v", item)
		}
	}
}

// TestRequestAuditMiddlewareSkipsUnmatchedRequest 验证未命中策略的普通请求不会落库。
func TestRequestAuditMiddlewareSkipsUnmatchedRequest(t *testing.T) {
	repo := &memoryAuditRepository{}
	ctx, engine, _ := newModuleTestContext(t, repo)
	authService := stubAuthService{user: moduleapi.CurrentUser{ID: 7, Username: "alice", DisplayName: "Alice"}}
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
	ctx, engine, _ := newModuleTestContext(t, repo)

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
	_, engine, _ := newModuleTestContext(t, repo)

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
	_, engine, _ := newModuleTestContext(t, repo)

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

func TestAuditLogsRouteAcceptsRepeatedSortParams(t *testing.T) {
	repo := &memoryAuditRepository{}
	_, engine, _ := newModuleTestContext(t, repo)

	request := httptest.NewRequest(http.MethodGet, "/api/audit/logs?sort=created_at:desc&sort=created_at:asc", nil)
	request.Header.Set("Authorization", "Bearer token")
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}
}

func TestAuditLogsRouteRejectsUnknownQueryKeys(t *testing.T) {
	repo := &memoryAuditRepository{}
	_, engine, _ := newModuleTestContext(t, repo)

	request := httptest.NewRequest(http.MethodGet, "/api/audit/logs?sort_by=created_at", nil)
	request.Header.Set("Authorization", "Bearer token")
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", recorder.Code)
	}
}

func TestAuditLogDetailRouteReturnsEvidenceRecord(t *testing.T) {
	actorID := uint64(7)
	repo := &memoryAuditRepository{
		items: []store.AuditLog{
			{
				ID:               42,
				ActorUserID:      &actorID,
				ActorUsername:    "alice",
				ActorDisplayName: "Alice",
				Action:           "auth.token.expired",
				ResourceType:     "auth",
				ResourceID:       "token",
				ResourceName:     "access token",
				Success:          false,
				RequestID:        "req-42",
				IP:               "127.0.0.1",
				UserAgent:        "vitest",
				Message:          "Token expired",
				Metadata:         json.RawMessage(`{"traceId":"trace-42","route":"/api/auth/bootstrap"}`),
				CreatedAt:        time.Date(2026, 6, 13, 12, 0, 0, 0, time.UTC),
			},
		},
	}
	_, engine, _ := newModuleTestContext(t, repo)

	request := httptest.NewRequest(http.MethodGet, "/api/audit/logs/42", nil)
	request.Header.Set("Authorization", "Bearer token")
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", recorder.Code, recorder.Body.String())
	}
	data := testassert.DecodeSuccessData[map[string]any](t, recorder)
	if data["id"] != float64(42) || data["request_id"] != "req-42" || data["action"] != "auth.token.expired" {
		t.Fatalf("expected audit detail record, got %#v", data)
	}
}

func TestAuditLogDetailRouteKeepsTargetLabelWireShape(t *testing.T) {
	actorID := uint64(7)
	repo := &memoryAuditRepository{
		items: []store.AuditLog{
			{
				ID:               42,
				ActorUserID:      &actorID,
				ActorUsername:    "alice",
				ActorDisplayName: "Alice",
				Action:           "auth.token.expired",
				ResourceType:     "auth",
				TargetType:       "AUTH",
				TargetLabel:      "Authentication",
				ResourceID:       "token",
				ResourceName:     "",
				Success:          false,
				RequestID:        "req-42",
				IP:               "127.0.0.1",
				UserAgent:        "vitest",
				Message:          "Token expired",
				Metadata:         json.RawMessage(`{"traceId":"trace-42","route":"/api/auth/bootstrap"}`),
				CreatedAt:        time.Date(2026, 6, 13, 12, 0, 0, 0, time.UTC),
			},
		},
	}
	_, engine, _ := newModuleTestContext(t, repo)

	request := httptest.NewRequest(http.MethodGet, "/api/audit/logs/42", nil)
	request.Header.Set("Authorization", "Bearer token")
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", recorder.Code, recorder.Body.String())
	}
	data := testassert.DecodeSuccessData[map[string]any](t, recorder)
	if data["target_label"] != "Authentication" || data["target_type"] != "AUTH" {
		t.Fatalf("expected localized target_label in existing wire field, got %#v", data)
	}
	if _, ok := data["target_label_key"]; ok {
		t.Fatalf("wire shape must remain unchanged, got %#v", data)
	}
}

func TestAuditLogDetailRouteReturnsNotFound(t *testing.T) {
	repo := &memoryAuditRepository{}
	_, engine, _ := newModuleTestContext(t, repo)

	request := httptest.NewRequest(http.MethodGet, "/api/audit/logs/404", nil)
	request.Header.Set("Authorization", "Bearer token")
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d body=%s", recorder.Code, recorder.Body.String())
	}
	response := testassert.DecodeErrorResponse(t, recorder)
	if field := response.Details["field"]; field != "id" {
		t.Fatalf("expected invalid detail field id, got %#v", field)
	}
}

func TestAuditLogsRouteAcceptsRegisteredDrilldownScopes(t *testing.T) {
	repo := &memoryAuditRepository{}
	scopes := []string{
		"failed_operations",
		"high_risk_operations",
		"sensitive_operations",
		"auth_failures",
		"permission_denials",
		"rbac_changes",
		"critical_security",
	}
	_, engine, _ := newModuleTestContextWithDrilldown(t, repo, scopes)

	for _, scope := range scopes {
		request := httptest.NewRequest(http.MethodGet, "/api/audit/logs?scope="+scope, nil)
		request.Header.Set("Authorization", "Bearer token")
		recorder := httptest.NewRecorder()
		engine.ServeHTTP(recorder, request)

		if recorder.Code != http.StatusOK {
			t.Fatalf("expected status 200 for scope %q, got %d", scope, recorder.Code)
		}
	}
}

func TestAuditLogsRouteRejectsUnknownDrilldownScope(t *testing.T) {
	repo := &memoryAuditRepository{}
	_, engine, _ := newModuleTestContextWithDrilldown(t, repo, []string{"sensitive_operations"})

	request := httptest.NewRequest(http.MethodGet, "/api/audit/logs?scope=failed_operations", nil)
	request.Header.Set("Authorization", "Bearer token")
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", recorder.Code)
	}

	response := testassert.DecodeErrorResponse(t, recorder)
	if field := response.Details["field"]; field != "scope" {
		t.Fatalf("expected invalid scope field, got %#v", field)
	}
}

func TestRequirePermissionPublishesSecurityAuditEvent(t *testing.T) {
	repo := &memoryAuditRepository{}
	ctx, engine, _ := newModuleTestContext(t, repo)

	ctx.Router.GET(
		"/roles",
		httpx.RequirePermission(
			ctx.I18n,
			stubAuthService{user: moduleapi.CurrentUser{ID: 7, Username: "alice", DisplayName: "Alice"}},
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

func defaultModuleTestPolicyRules() []store.AuditPolicyRule {
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
	_, engine, _ := newModuleTestContext(t, repo)

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
	_, _, bus := newModuleTestContext(t, repo)

	requestCtx := httpx.WithRequestAuditContext(
		moduleapi.WithRequestAuthContext(context.Background(), moduleapi.RequestAuthContext{
			User: &moduleapi.CurrentUser{ID: 21, Username: "ctx-admin", DisplayName: "Context Admin"},
		}),
		httpx.RequestAuditContext{
			RequestID: "req-domain-1",
			TraceID:   "req-domain-1",
			Route:     "/api/users",
			Method:    http.MethodPost,
			ClientIP:  "203.0.113.10",
			UserAgent: "audit-module-test",
		},
	)

	err := bus.Publish(requestCtx, eventbus.Event{
		Name: string(moduleapi.AuditRecordEventName),
		Payload: moduleapi.AuditEvent{
			Operator:     &moduleapi.CurrentUser{ID: 9, Username: "bob"},
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
	_, _, bus := newModuleTestContext(t, repo)

	requestCtx := httpx.WithRequestAuditContext(
		moduleapi.WithRequestAuthContext(context.Background(), moduleapi.RequestAuthContext{
			User: &moduleapi.CurrentUser{ID: 22, Username: "ctx-user", DisplayName: "Context User"},
		}),
		httpx.RequestAuditContext{
			RequestID: "req-domain-2",
			TraceID:   "req-domain-2",
			Route:     "/api/roles/:id/status",
			Method:    http.MethodPost,
			ClientIP:  "203.0.113.22",
			UserAgent: "audit-module-test-auth",
		},
	)

	err := bus.Publish(requestCtx, eventbus.Event{
		Name: string(moduleapi.AuditRecordEventName),
		Payload: moduleapi.AuditEvent{
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
	_, _, bus := newModuleTestContext(t, repo)

	err := bus.Publish(context.Background(), eventbus.Event{
		Name: string(moduleapi.AuditRecordEventName),
		Payload: &moduleapi.AuditEvent{
			Operator:     &moduleapi.CurrentUser{ID: 10, Username: "carol"},
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
	ctx, _, bus := newModuleTestContext(t, failingAuditRepository{})

	if err := ctx.EventBus.Subscribe("noop", func(context.Context, eventbus.Event) error { return nil }); err != nil {
		t.Fatalf("subscribe noop: %v", err)
	}

	err := bus.Publish(context.Background(), eventbus.Event{
		Name: string(moduleapi.AuditRecordEventName),
		Payload: moduleapi.AuditEvent{
			Operator:     &moduleapi.CurrentUser{ID: 10, Username: "carol"},
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
	_, _, bus := newModuleTestContextWithLogger(t, repo, logger)

	err := bus.Publish(context.Background(), eventbus.Event{
		Name: string(moduleapi.AuditRecordEventName),
		Payload: moduleapi.AuditEvent{
			Kind:         moduleapi.AuditEventKindSecurity,
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
	_, _, bus := newModuleTestContext(t, repo)

	err := bus.Publish(context.Background(), eventbus.Event{
		Name: string(moduleapi.AuditRecordEventName),
		Payload: moduleapi.AuditEvent{
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
	_, engine, _ := newModuleTestContext(t, repo)

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
		rules: append(defaultModuleTestPolicyRules(), store.AuditPolicyRule{
			Name:      "domain.rbac.role.delete",
			Source:    store.AuditSourceDomainEvent,
			Enabled:   true,
			Priority:  40,
			Effect:    store.AuditPolicyEffectInclude,
			EventType: "rbac.role.delete",
			MatchType: store.AuditPolicyMatchTypeExact,
		}),
	}
	_, _, bus := newModuleTestContext(t, repo)

	err := bus.Publish(context.Background(), eventbus.Event{
		Name: string(moduleapi.AuditRecordEventName),
		Payload: moduleapi.AuditEvent{
			Kind:         moduleapi.AuditEventKindDomain,
			Operator:     &moduleapi.CurrentUser{ID: 9, Username: "bob"},
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
	ctx, engine, _ := newModuleTestContext(t, repo)

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
	_, engine, _ := newModuleTestContext(t, repo)

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
