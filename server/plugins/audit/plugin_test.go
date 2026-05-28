package audit

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
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

type failingAuditRepository struct{}

func (failingAuditRepository) CreateAuditLog(context.Context, store.CreateAuditLogInput) (store.AuditLog, error) {
	return store.AuditLog{}, errors.New("write failed")
}

func (failingAuditRepository) ListAuditLogs(context.Context, store.ListAuditLogsQuery) (store.ListAuditLogsResult, error) {
	return store.ListAuditLogsResult{}, nil
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

// TestRequestAuditMiddlewareCapturesAuthenticatedRequest 验证请求级自动审计会在
// 受保护路由完成后记录当前主体和稳定路由语义。
func TestRequestAuditMiddlewareCapturesAuthenticatedRequest(t *testing.T) {
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
	if len(repo.items) != 1 {
		t.Fatalf("expected one audit record, got %d", len(repo.items))
	}

	record := repo.items[0]
	assertAuditRecord(t, record, expectedAuditRecord{
		username:     "alice",
		displayName:  "Alice",
		action:       "GET /api/users/:id",
		resourceType: "users",
		resourceID:   "42",
	})
	if record.RequestID == "" {
		t.Fatal("expected request id to be recorded")
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

type expectedAuditRecord struct {
	username     string
	displayName  string
	action       string
	resourceType string
	resourceID   string
}

func assertAuditRecord(t *testing.T, record store.AuditLog, expected expectedAuditRecord) {
	t.Helper()

	if record.ActorUserID == nil || *record.ActorUserID != 7 {
		t.Fatalf("expected actor id 7, got %#v", record.ActorUserID)
	}
	if record.ActorUsername != expected.username || record.ActorDisplayName != expected.displayName {
		t.Fatalf("expected actor identity %s/%s, got %#v", expected.username, expected.displayName, record)
	}
	if record.Action != expected.action {
		t.Fatalf("expected stable action, got %q", record.Action)
	}
	if record.ResourceType != expected.resourceType {
		t.Fatalf("expected resource type %s, got %q", expected.resourceType, record.ResourceType)
	}
	if record.ResourceID != expected.resourceID {
		t.Fatalf("expected resource id %s, got %q", expected.resourceID, record.ResourceID)
	}
	if !record.Success || record.Message != "" {
		t.Fatalf("expected successful audit record, got %#v", record)
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
