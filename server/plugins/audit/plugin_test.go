package audit

import (
	"context"
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
	"graft/server/internal/store"
)

type pluginTestStoreFactory struct {
	audit store.AuditRepository
}

func (f pluginTestStoreFactory) Audit() store.AuditRepository { return f.audit }
func (f pluginTestStoreFactory) Users() store.UserRepository  { return nil }
func (f pluginTestStoreFactory) Auth() store.AuthRepository   { return nil }
func (f pluginTestStoreFactory) RBAC() store.RBACRepository   { return nil }

type memoryAuditRepository struct {
	items []store.AuditLog
}

func (r *memoryAuditRepository) CreateAuditLog(_ context.Context, input store.CreateAuditLogInput) (store.AuditLog, error) {
	record := store.AuditLog{
		ID:            uint64(len(r.items) + 1),
		OperatorID:    input.OperatorID,
		OperatorName:  input.OperatorName,
		Action:        input.Action,
		ResourceType:  input.ResourceType,
		ResourceID:    input.ResourceID,
		RequestMethod: input.RequestMethod,
		RequestPath:   input.RequestPath,
		IP:            input.IP,
		UserAgent:     input.UserAgent,
		Success:       input.Success,
		ErrorMessage:  input.ErrorMessage,
		CreatedAt:     input.CreatedAt,
	}
	r.items = append(r.items, record)
	return record, nil
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
		Stores:             pluginTestStoreFactory{audit: repo},
		MenuRegistry:       menu.NewRegistry(),
		PermissionRegistry: permission.NewRegistry(),
		CronRegistry:       cronx.NewRegistry(),
	}

	if err := NewPlugin().Register(ctx); err != nil {
		t.Fatalf("register audit plugin: %v", err)
	}

	return ctx, engine, bus
}

// TestRequestAuditMiddlewareCapturesAuthenticatedRequest 验证请求级自动审计会在
// 受保护路由完成后记录当前主体和稳定路由语义。
func TestRequestAuditMiddlewareCapturesAuthenticatedRequest(t *testing.T) {
	repo := &memoryAuditRepository{}
	ctx, engine, _ := newPluginTestContext(t, repo)

	if err := ctx.Services.RegisterSingleton((*pluginapi.AuthService)(nil), func(_ container.Resolver) (any, error) {
		return stubAuthService{user: pluginapi.CurrentUser{ID: 7, Username: "alice", DisplayName: "Alice"}}, nil
	}); err != nil {
		t.Fatalf("register auth service: %v", err)
	}
	if err := ctx.Services.RegisterSingleton((*pluginapi.Authorizer)(nil), func(_ container.Resolver) (any, error) {
		return allowAuthorizer{}, nil
	}); err != nil {
		t.Fatalf("register authorizer: %v", err)
	}

	ctx.Router.GET("/users/:id", httpx.RequirePermission(ctx.I18n, ctx.Services, "user.read"), func(ginCtx *gin.Context) {
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
	assertAuditRecord(t, record, "Alice", "GET /api/users/:id", "users", "42")
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
	if repo.items[0].ErrorMessage != "common.invalid_argument" {
		t.Fatalf("expected stable error message key, got %q", repo.items[0].ErrorMessage)
	}
}

func assertAuditRecord(t *testing.T, record store.AuditLog, operatorName string, action string, resourceType string, resourceID string) {
	t.Helper()

	if record.OperatorID == nil || *record.OperatorID != 7 {
		t.Fatalf("expected operator id 7, got %#v", record.OperatorID)
	}
	if record.OperatorName != operatorName {
		t.Fatalf("expected operator name %s, got %q", operatorName, record.OperatorName)
	}
	if record.Action != action {
		t.Fatalf("expected stable action, got %q", record.Action)
	}
	if record.ResourceType != resourceType {
		t.Fatalf("expected resource type %s, got %q", resourceType, record.ResourceType)
	}
	if record.ResourceID != resourceID {
		t.Fatalf("expected resource id %s, got %q", resourceID, record.ResourceID)
	}
	if !record.Success || record.ErrorMessage != "" {
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
	if repo.items[0].OperatorID == nil || *repo.items[0].OperatorID != 9 {
		t.Fatalf("expected operator id 9, got %#v", repo.items[0].OperatorID)
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
