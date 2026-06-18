// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package container

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"graft/server/internal/config"
	internalcontainer "graft/server/internal/container"
	"graft/server/internal/eventbus"
	"graft/server/internal/i18n"
	"graft/server/internal/module"
	"graft/server/internal/moduleapi"
	containercontract "graft/server/modules/container/contract"
	containerlocales "graft/server/modules/container/locales"
)

func TestRoutesRequireContainerPermissions(t *testing.T) {
	t.Parallel()

	authorizer := &recordingAuthorizer{}
	ctx, engine := newRouteTestContext(authorizer)
	service, err := newService(containerServiceOptions{
		runtime:                 fakeRuntime{},
		enabled:                 true,
		dangerousActionsEnabled: true,
		defaultTail:             defaultContainerLogsDefaultTail,
		maxTail:                 defaultContainerLogsMaxTail,
	})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	if err := registerRoutes(ctx, moduleID, service); err != nil {
		t.Fatalf("register routes: %v", err)
	}

	assertDetailRoutePermission(t, engine, authorizer)
	assertLogsRoutePermission(t, engine, authorizer)
	assertMountUsageRoutePermission(t, engine, authorizer)
	assertRemoveRoutePermission(t, engine, authorizer)
	assertBatchActionRoutePermission(t, engine, authorizer)
}

func assertDetailRoutePermission(t *testing.T, engine *gin.Engine, authorizer *recordingAuthorizer) {
	t.Helper()

	authorizer.reset()
	response := httptest.NewRecorder()
	engine.ServeHTTP(response, authorizedRequest(http.MethodGet, "/api/ops/containers/abc123"))
	if response.Code != http.StatusOK {
		t.Fatalf("expected detail 200, got %d: %s", response.Code, response.Body.String())
	}
	if !slices.Contains(authorizer.permissions, containercontract.ContainerDetailPermission.String()) {
		t.Fatalf("expected detail permission, got %#v", authorizer.permissions)
	}
	if !slices.Contains(authorizer.permissions, containercontract.ContainerEnvironmentPermission.String()) {
		t.Fatalf("expected environment permission check, got %#v", authorizer.permissions)
	}
}

func assertLogsRoutePermission(t *testing.T, engine *gin.Engine, authorizer *recordingAuthorizer) {
	t.Helper()

	authorizer.reset()
	response := httptest.NewRecorder()
	engine.ServeHTTP(response, authorizedRequest(http.MethodGet, "/api/ops/containers/abc123/logs?tail=20&stdout=true"))
	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
	}
	if !slices.Contains(authorizer.permissions, containercontract.ContainerLogsPermission.String()) {
		t.Fatalf("expected logs permission, got %#v", authorizer.permissions)
	}
}

func assertMountUsageRoutePermission(t *testing.T, engine *gin.Engine, authorizer *recordingAuthorizer) {
	t.Helper()

	authorizer.reset()
	response := httptest.NewRecorder()
	engine.ServeHTTP(response, authorizedRequest(http.MethodGet, "/api/ops/containers/abc123/mounts/usage"))
	if response.Code != http.StatusOK {
		t.Fatalf("expected mount usage list 200, got %d: %s", response.Code, response.Body.String())
	}
	if !slices.Contains(authorizer.permissions, containercontract.ContainerDetailPermission.String()) {
		t.Fatalf("expected mount usage list detail permission, got %#v", authorizer.permissions)
	}
	if !strings.Contains(response.Body.String(), `"mount_id":"m_`) {
		t.Fatalf("expected mount_id response field, got %s", response.Body.String())
	}

	authorizer.reset()
	mountID := fakeMountID()
	response = httptest.NewRecorder()
	engine.ServeHTTP(response, authorizedJSONRequest(http.MethodPost, "/api/ops/containers/abc123/mounts/"+mountID+"/usage/refresh", `{}`))
	if response.Code != http.StatusOK {
		t.Fatalf("expected mount usage refresh 200, got %d: %s", response.Code, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), `"size_display":"1 KiB"`) {
		t.Fatalf("expected size_display response field, got %s", response.Body.String())
	}
}

func assertRemoveRoutePermission(t *testing.T, engine *gin.Engine, authorizer *recordingAuthorizer) {
	t.Helper()

	authorizer.reset()
	response := httptest.NewRecorder()
	engine.ServeHTTP(response, authorizedJSONRequest(http.MethodPost, "/api/ops/containers/abc123/remove", `{"force":true}`))
	if response.Code != http.StatusOK {
		t.Fatalf("expected remove 200, got %d: %s", response.Code, response.Body.String())
	}
	if !slices.Contains(authorizer.permissions, containercontract.ContainerRemovePermission.String()) {
		t.Fatalf("expected remove permission, got %#v", authorizer.permissions)
	}
}

func assertBatchActionRoutePermission(t *testing.T, engine *gin.Engine, authorizer *recordingAuthorizer) {
	t.Helper()

	authorizer.reset()
	response := httptest.NewRecorder()
	engine.ServeHTTP(response, authorizedJSONRequest(http.MethodPost, "/api/ops/containers/batch-actions", `{"action":"start","ids":["abc123"]}`))
	if response.Code != http.StatusOK {
		t.Fatalf("expected batch start 200, got %d: %s", response.Code, response.Body.String())
	}
	if !slices.Contains(authorizer.permissions, containercontract.ContainerStartPermission.String()) {
		t.Fatalf("expected batch start permission, got %#v", authorizer.permissions)
	}

	authorizer.reset()
	response = httptest.NewRecorder()
	engine.ServeHTTP(response, authorizedJSONRequest(http.MethodPost, "/api/ops/containers/batch-actions", `{"action":"remove","ids":["abc123"],"force":true}`))
	if response.Code != http.StatusOK {
		t.Fatalf("expected batch remove 200, got %d: %s", response.Code, response.Body.String())
	}
	if !slices.Contains(authorizer.permissions, containercontract.ContainerRemovePermission.String()) {
		t.Fatalf("expected batch remove to require remove permission, got %#v", authorizer.permissions)
	}
}

func TestRoutesRejectInvalidRef(t *testing.T) {
	t.Parallel()

	_, engine := newRegisteredRouteTestService(t)
	response := httptest.NewRecorder()
	engine.ServeHTTP(response, authorizedRequest(http.MethodGet, "/api/ops/containers/bad%00id"))
	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", response.Code, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), containercontract.ContainerInvalidRef.String()) {
		t.Fatalf("expected invalid ref message key, got %s", response.Body.String())
	}
}

func TestRoutesRejectInvalidLogQuery(t *testing.T) {
	t.Parallel()

	_, engine := newRegisteredRouteTestService(t)
	response := httptest.NewRecorder()
	engine.ServeHTTP(response, authorizedRequest(http.MethodGet, "/api/ops/containers/abc123/logs?since=not-a-time"))
	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", response.Code, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), containercontract.ContainerInvalidLogQuery.String()) {
		t.Fatalf("expected invalid log query message key, got %s", response.Body.String())
	}
}

func TestRoutesRejectInvalidListQuery(t *testing.T) {
	t.Parallel()

	_, engine := newRegisteredRouteTestService(t)
	response := httptest.NewRecorder()
	engine.ServeHTTP(response, authorizedRequest(http.MethodGet, "/api/ops/containers?limit=0"))
	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", response.Code, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), "limit") {
		t.Fatalf("expected invalid limit field, got %s", response.Body.String())
	}
}

func TestRoutesRejectInvalidBatchAction(t *testing.T) {
	t.Parallel()

	_, engine := newRegisteredRouteTestService(t)
	response := httptest.NewRecorder()
	engine.ServeHTTP(response, authorizedJSONRequest(http.MethodPost, "/api/ops/containers/batch-actions", `{"action":"prune","ids":["abc123"]}`))
	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", response.Code, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), containercontract.ContainerInvalidBatchAction.String()) {
		t.Fatalf("expected invalid batch action message key, got %s", response.Body.String())
	}
}

func newRegisteredRouteTestService(t *testing.T) (*module.Context, *gin.Engine) {
	t.Helper()

	ctx, engine := newRouteTestContext(&recordingAuthorizer{})
	service, err := newService(containerServiceOptions{
		runtime:                 fakeRuntime{},
		enabled:                 true,
		dangerousActionsEnabled: true,
		defaultTail:             defaultContainerLogsDefaultTail,
		maxTail:                 defaultContainerLogsMaxTail,
	})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	if err := registerRoutes(ctx, moduleID, service); err != nil {
		t.Fatalf("register routes: %v", err)
	}
	return ctx, engine
}

func newRouteTestContext(authorizer moduleapi.Authorizer) (*module.Context, *gin.Engine) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	localizer := i18n.MustNew(config.I18nConfig{
		DefaultLocale:  "zh-CN",
		FallbackLocale: "zh-CN",
		SupportedLocales: []string{
			"zh-CN",
			"en-US",
		},
	})
	resources, err := containerlocales.EmbeddedLocaleResources()
	if err != nil {
		panic(fmt.Sprintf("load container locale resources: %v", err))
	}
	if err := localizer.RegisterEmbeddedLocaleResources(resources); err != nil {
		panic(fmt.Sprintf("register container locale resources: %v", err))
	}
	services := internalcontainer.New()
	if err := services.RegisterSingleton((*moduleapi.AuthService)(nil), func(internalcontainer.Resolver) (any, error) {
		return routeTestAuthService{}, nil
	}); err != nil {
		panic(err)
	}
	if err := services.RegisterSingleton((*moduleapi.Authorizer)(nil), func(internalcontainer.Resolver) (any, error) {
		return authorizer, nil
	}); err != nil {
		panic(err)
	}
	return &module.Context{
		Logger: zap.NewNop(),
		I18n:    localizer,
		EventBus: eventbus.New(zap.NewNop()),
		Router:   engine.Group("/api"),
		Services: services,
	}, engine
}

func authorizedRequest(method string, path string) *http.Request {
	request := httptest.NewRequest(method, path, nil)
	request.Header.Set("Authorization", "Bearer route-test-token")
	return request
}

func authorizedJSONRequest(method string, path string, body string) *http.Request {
	request := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	request.Header.Set("Authorization", "Bearer route-test-token")
	request.Header.Set("Content-Type", "application/json")
	return request
}

type routeTestAuthService struct{}

func (routeTestAuthService) CurrentUser(context.Context) (*moduleapi.CurrentUser, error) {
	return &moduleapi.CurrentUser{ID: 7, Username: "admin", DisplayName: "Admin"}, nil
}

func (routeTestAuthService) ParseAccessToken(context.Context, string) (*moduleapi.AccessTokenClaims, error) {
	return &moduleapi.AccessTokenClaims{UserID: 7, SessionID: "session-1", ExpiresAt: time.Now().UTC().Add(time.Hour)}, nil
}

type recordingAuthorizer struct {
	permissions []string
}

func (a *recordingAuthorizer) Authorize(_ context.Context, _ moduleapi.RequestAuthContext, permission string) error {
	a.permissions = append(a.permissions, permission)
	return nil
}

func (a *recordingAuthorizer) reset() {
	a.permissions = nil
}
