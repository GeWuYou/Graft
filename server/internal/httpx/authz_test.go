package httpx

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

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

	recorder := httptest.NewRecorder()
	ctx, engine := gin.CreateTestContext(recorder)
	engine.Use(RequirePermission("user.read"))
	engine.GET("/api/users/:id", func(inner *gin.Context) {
		inner.Status(http.StatusOK)
	})

	request := httptest.NewRequest(http.MethodGet, "/api/users/1", nil)
	ctx.Request = request
	engine.HandleContext(ctx)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, recorder.Code)
	}
}

// TestRequirePermissionRejectsMissingPermission 验证身份存在但缺少所需权限码
// 时，请求会被拒绝为无权限。
func TestRequirePermissionRejectsMissingPermission(t *testing.T) {
	gin.SetMode(gin.TestMode)

	recorder := httptest.NewRecorder()
	ctx, engine := gin.CreateTestContext(recorder)
	engine.Use(RequirePermission("user.read"))
	engine.GET("/api/users/:id", func(inner *gin.Context) {
		inner.Status(http.StatusOK)
	})

	request := httptest.NewRequest(http.MethodGet, "/api/users/1", nil)
	request.Header.Set(actorHeader, "alice")
	request.Header.Set(permissionsHeader, "dashboard.view")
	ctx.Request = request
	engine.HandleContext(ctx)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d", http.StatusForbidden, recorder.Code)
	}
}

// TestRequirePermissionAllowsAuthorizedRequest 验证身份和权限都满足时，请求
// 可以继续进入后续处理链。
func TestRequirePermissionAllowsAuthorizedRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)

	recorder := httptest.NewRecorder()
	ctx, engine := gin.CreateTestContext(recorder)
	engine.Use(RequirePermission("user.read"))
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
