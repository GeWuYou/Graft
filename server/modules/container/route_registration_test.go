// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package container

import (
	"context"
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
)

func TestRoutesRequireContainerPermissions(t *testing.T) {
	t.Parallel()

	authorizer := &recordingAuthorizer{}
	ctx, engine := newRouteTestContext(authorizer)
	service, err := newService(containerServiceOptions{
		runtime:     fakeRuntime{},
		enabled:     true,
		defaultTail: defaultContainerLogsDefaultTail,
		maxTail:     defaultContainerLogsMaxTail,
	})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	if err := registerRoutes(ctx, moduleID, service); err != nil {
		t.Fatalf("register routes: %v", err)
	}

	response := httptest.NewRecorder()
	engine.ServeHTTP(response, authorizedRequest(http.MethodGet, "/api/ops/containers/abc123/logs?tail=20&stdout=true"))
	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
	}
	if !slices.Contains(authorizer.permissions, containercontract.ContainerLogsPermission.String()) {
		t.Fatalf("expected logs permission, got %#v", authorizer.permissions)
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

func newRegisteredRouteTestService(t *testing.T) (*module.Context, *gin.Engine) {
	t.Helper()

	ctx, engine := newRouteTestContext(&recordingAuthorizer{})
	service, err := newService(containerServiceOptions{
		runtime:     fakeRuntime{},
		enabled:     true,
		defaultTail: defaultContainerLogsDefaultTail,
		maxTail:     defaultContainerLogsMaxTail,
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
		I18n: i18n.MustNew(config.I18nConfig{
			DefaultLocale:  "zh-CN",
			FallbackLocale: "zh-CN",
			SupportedLocales: []string{
				"zh-CN",
				"en-US",
			},
		}),
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
