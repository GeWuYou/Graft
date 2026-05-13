package httpx

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"graft/server/internal/config"
	"graft/server/internal/i18n"
)

func newTestLocalizer() *i18n.Service {
	return i18n.New(config.I18nConfig{
		DefaultLocale:    "zh-CN",
		FallbackLocale:   "zh-CN",
		SupportedLocales: []string{"zh-CN", "en-US"},
	})
}

// TestSessionFromRequestParsesActorAndPermissions 验证请求头会被解析为
// 显式会话信息，并过滤空白权限项。
func TestSessionFromRequestParsesActorAndPermissions(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/api/users/1", nil)
	request.Header.Set(actorHeader, "alice")
	request.Header.Set(permissionsHeader, " user.read , dashboard.view ,, ")

	session := SessionFromRequest(request)

	if session.Actor != "alice" {
		t.Fatalf("expected actor alice, got %q", session.Actor)
	}
	if !session.HasPermission("user.read") {
		t.Fatal("expected parsed permissions to include user.read")
	}
	if !session.HasPermission("dashboard.view") {
		t.Fatal("expected parsed permissions to include dashboard.view")
	}
}

// TestRequirePermissionRejectsMissingActor 验证缺少身份头时会被后端权限守卫
// 直接拒绝，而不是继续执行受保护路由。
func TestRequirePermissionRejectsMissingActor(t *testing.T) {
	gin.SetMode(gin.TestMode)
	localizer := newTestLocalizer()

	recorder := httptest.NewRecorder()
	ctx, engine := gin.CreateTestContext(recorder)
	engine.Use(RequirePermission(localizer, "user.read"))
	engine.GET("/api/users/:id", func(inner *gin.Context) {
		inner.Status(http.StatusOK)
	})

	request := httptest.NewRequest(http.MethodGet, "/api/users/1", nil)
	ctx.Request = request
	engine.HandleContext(ctx)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, recorder.Code)
	}

	var payload ErrorResponse
	if err := json.NewDecoder(recorder.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.MessageKey != "auth.missing_actor" {
		t.Fatalf("expected missing actor message key, got %#v", payload)
	}
	if payload.Message != "缺少请求身份信息" || payload.Error != payload.Message {
		t.Fatalf("expected localized missing actor message, got %#v", payload)
	}
	if payload.Locale != "zh-CN" {
		t.Fatalf("expected zh-CN locale, got %#v", payload)
	}
}

// TestRequirePermissionRejectsMissingPermission 验证身份存在但缺少所需权限码
// 时，请求会被拒绝为无权限。
func TestRequirePermissionRejectsMissingPermission(t *testing.T) {
	gin.SetMode(gin.TestMode)
	localizer := newTestLocalizer()

	recorder := httptest.NewRecorder()
	ctx, engine := gin.CreateTestContext(recorder)
	engine.Use(RequirePermission(localizer, "user.read"))
	engine.GET("/api/users/:id", func(inner *gin.Context) {
		inner.Status(http.StatusOK)
	})

	request := httptest.NewRequest(http.MethodGet, "/api/users/1", nil)
	request.Header.Set(actorHeader, "alice")
	request.Header.Set(permissionsHeader, "dashboard.view")
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
	if payload.MessageKey != "auth.missing_permission" {
		t.Fatalf("expected missing permission message key, got %#v", payload)
	}
	if payload.Message != "缺少所需权限" || payload.Error != payload.Message {
		t.Fatalf("expected localized missing permission message, got %#v", payload)
	}
	if payload.Locale != "en-US" {
		t.Fatalf("expected requested locale to be echoed, got %#v", payload)
	}
	if payload.Details["permission"] != "user.read" {
		t.Fatalf("expected denied permission to be echoed, got %#v", payload)
	}
}

// TestRequirePermissionAllowsAuthorizedRequest 验证身份和权限都满足时，请求
// 可以继续进入后续处理链。
func TestRequirePermissionAllowsAuthorizedRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)

	recorder := httptest.NewRecorder()
	ctx, engine := gin.CreateTestContext(recorder)
	engine.Use(RequirePermission(nil, "user.read"))
	engine.GET("/api/users/:id", func(inner *gin.Context) {
		inner.Status(http.StatusOK)
	})

	request := httptest.NewRequest(http.MethodGet, "/api/users/1", nil)
	request.Header.Set(actorHeader, "alice")
	request.Header.Set(permissionsHeader, "dashboard.view,user.read")
	ctx.Request = request
	engine.HandleContext(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}
}

// TestRequirePermissionAllowsAuthenticatedRequestWhenPermissionCodeBlank 验证空
// 权限码只要求存在调用者身份，不会额外阻塞已登录请求。
func TestRequirePermissionAllowsAuthenticatedRequestWhenPermissionCodeBlank(t *testing.T) {
	gin.SetMode(gin.TestMode)

	recorder := httptest.NewRecorder()
	ctx, engine := gin.CreateTestContext(recorder)
	engine.Use(RequirePermission(nil, " "))
	engine.GET("/api/profile", func(inner *gin.Context) {
		inner.Status(http.StatusOK)
	})

	request := httptest.NewRequest(http.MethodGet, "/api/profile", nil)
	request.Header.Set(actorHeader, "alice")
	ctx.Request = request
	engine.HandleContext(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}
}
