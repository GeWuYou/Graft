package httpx

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

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
