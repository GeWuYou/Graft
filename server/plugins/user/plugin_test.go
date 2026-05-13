package user

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"graft/server/internal/config"
	"graft/server/internal/container"
	"graft/server/internal/cronx"
	"graft/server/internal/httpx"
	"graft/server/internal/i18n"
	"graft/server/internal/menu"
	"graft/server/internal/permission"
	"graft/server/internal/plugin"
	"graft/server/internal/pluginapi"
	"graft/server/internal/store"
)

type pluginTestStoreFactory struct {
	users store.UserRepository
}

func (f pluginTestStoreFactory) Users() store.UserRepository {
	return f.users
}

type pluginTestUserRepository struct {
	getByID func(ctx context.Context, id uint64) (store.User, error)
}

func (r pluginTestUserRepository) GetByID(ctx context.Context, id uint64) (store.User, error) {
	if r.getByID == nil {
		return store.User{}, store.ErrUserNotFound
	}

	return r.getByID(ctx, id)
}

func newPluginTestContext(t *testing.T, repo store.UserRepository) (*plugin.Context, *gin.Engine) {
	t.Helper()

	gin.SetMode(gin.TestMode)
	engine := gin.New()
	ctx := &plugin.Context{
		Logger:             zap.NewNop(),
		I18n:               i18n.New(config.I18nConfig{DefaultLocale: "zh-CN", FallbackLocale: "zh-CN", SupportedLocales: []string{"zh-CN", "en-US"}}),
		Router:             engine.Group("/api"),
		Services:           container.New(),
		Stores:             pluginTestStoreFactory{users: repo},
		MenuRegistry:       menu.NewRegistry(),
		PermissionRegistry: permission.NewRegistry(),
		CronRegistry:       cronx.NewRegistry(),
	}

	if err := NewPlugin().Register(ctx); err != nil {
		t.Fatalf("register plugin: %v", err)
	}

	return ctx, engine
}

func newAuthorizedRequest(path string) *http.Request {
	request := httptest.NewRequest(http.MethodGet, path, nil)
	request.Header.Set("X-Graft-Actor", "alice")
	request.Header.Set("X-Graft-Permissions", "user.read")
	return request
}

// TestRegisterPublishesContracts 验证用户插件注册时会暴露权限、菜单与稳定
// 的跨插件用户服务。
func TestRegisterPublishesContracts(t *testing.T) {
	ctx, _ := newPluginTestContext(t, pluginTestUserRepository{
		getByID: func(context.Context, uint64) (store.User, error) {
			return store.User{
				ID:        7,
				Username:  "alice",
				Display:   "Alice",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}, nil
		},
	})

	if items := ctx.PermissionRegistry.Items(); len(items) != 1 || items[0].Code != "user.read" {
		t.Fatalf("expected one user.read permission, got %#v", items)
	}
	if items := ctx.MenuRegistry.Items(); len(items) != 1 || items[0].Path != "/users" {
		t.Fatalf("expected one /users menu item, got %#v", items)
	}

	svcAny, err := ctx.Services.Resolve((*pluginapi.UserService)(nil))
	if err != nil {
		t.Fatalf("resolve user service: %v", err)
	}

	summary, err := svcAny.(pluginapi.UserService).GetUserByID(context.Background(), 7)
	if err != nil {
		t.Fatalf("get user by id: %v", err)
	}
	if summary.ID != 7 || summary.Username != "alice" || summary.Display != "Alice" {
		t.Fatalf("expected stable user summary, got %#v", summary)
	}
}

// TestUserRouteRejectsInvalidID 验证用户查询路由会把非法 ID 收敛为 400
// JSON 响应，而不是继续访问仓储。
func TestUserRouteRejectsInvalidID(t *testing.T) {
	_, engine := newPluginTestContext(t, pluginTestUserRepository{
		getByID: func(context.Context, uint64) (store.User, error) {
			t.Fatal("user repository should not be called for invalid ids")
			return store.User{}, nil
		},
	})

	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, newAuthorizedRequest("/api/users/not-a-number"))

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, recorder.Code)
	}

	var payload httpx.ErrorResponse
	if err := json.NewDecoder(recorder.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.MessageKey != "common.invalid_argument" || payload.Locale != "zh-CN" {
		t.Fatalf("expected localized invalid argument contract, got %#v", payload)
	}
	if payload.Message != "请求参数不合法" || payload.Error != payload.Message {
		t.Fatalf("expected parse error payload, got %#v", payload)
	}
	if payload.Details["field"] != "id" {
		t.Fatalf("expected field detail to be id, got %#v", payload)
	}
}

// TestUserRouteReturnsNotFoundContract 验证仓储未命中时，路由会返回 404
// 与稳定错误消息，便于前端后续接入统一空态分支。
func TestUserRouteReturnsNotFoundContract(t *testing.T) {
	_, engine := newPluginTestContext(t, pluginTestUserRepository{
		getByID: func(context.Context, uint64) (store.User, error) {
			return store.User{}, store.ErrUserNotFound
		},
	})

	recorder := httptest.NewRecorder()
	request := newAuthorizedRequest("/api/users/7")
	request.Header.Set(i18n.LocaleHeader, "en-US")
	engine.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, recorder.Code)
	}

	var payload httpx.ErrorResponse
	if err := json.NewDecoder(recorder.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.MessageKey != "user.not_found" || payload.Locale != "en-US" {
		t.Fatalf("expected localized not found contract, got %#v", payload)
	}
	if payload.Message != "用户不存在" || payload.Error != payload.Message {
		t.Fatalf("expected not found payload, got %#v", payload)
	}
}

// TestUserRouteReturnsSummary 验证用户查询成功时会返回跨插件稳定 DTO，而不
// 直接泄漏仓储层内部结构。
func TestUserRouteReturnsSummary(t *testing.T) {
	_, engine := newPluginTestContext(t, pluginTestUserRepository{
		getByID: func(context.Context, uint64) (store.User, error) {
			return store.User{
				ID:        7,
				Username:  "alice",
				Display:   "Alice",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}, nil
		},
	})

	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, newAuthorizedRequest("/api/users/7"))

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}

	var payload pluginapi.UserSummary
	if err := json.NewDecoder(recorder.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.ID != 7 || payload.Username != "alice" || payload.Display != "Alice" {
		t.Fatalf("expected stable user summary payload, got %#v", payload)
	}
}

// TestUserRouteRequiresPermissionMiddleware 验证插件路由仍复用统一的后端
// 权限守卫契约，而不是在插件内部发散独立鉴权格式。
func TestUserRouteRequiresPermissionMiddleware(t *testing.T) {
	_, engine := newPluginTestContext(t, pluginTestUserRepository{})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/users/7", nil)
	request.Header.Set("X-Graft-Actor", "alice")
	request.Header.Set("X-Graft-Permissions", "dashboard.view")
	engine.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d", http.StatusForbidden, recorder.Code)
	}

	var payload httpx.ErrorResponse
	if err := json.NewDecoder(recorder.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.MessageKey != "auth.missing_permission" {
		t.Fatalf("expected permission middleware payload, got %#v", payload)
	}
	if payload.Details["permission"] != "user.read" {
		t.Fatalf("expected denied permission to be user.read, got %#v", payload)
	}
}
