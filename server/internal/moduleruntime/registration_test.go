// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package moduleruntime

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"graft/server/internal/config"
	"graft/server/internal/httpx"
	"graft/server/internal/i18n"
	"graft/server/internal/menu"
	"graft/server/internal/module"
	"graft/server/internal/moduleapi"
	"graft/server/internal/permission"
)

func TestRegisterExposesProtectedSnapshotRoutesAndMetadata(t *testing.T) {
	gin.SetMode(gin.TestMode)

	localizer := i18n.MustNew(config.I18nConfig{
		DefaultLocale:    "zh-CN",
		FallbackLocale:   "en-US",
		SupportedLocales: []string{"zh-CN", "en-US"},
	})
	menuRegistry := menu.NewRegistry()
	menuRegistry.Register(menu.Item{Code: "monitor.section", Path: "/server"})
	permissionRegistry := permission.NewRegistry()
	authorizer := recordingAuthorizer{}

	engine := gin.New()
	err := Register(
		Registration{
			I18n:               localizer,
			MenuRegistry:       menuRegistry,
			PermissionRegistry: permissionRegistry,
			Config: &config.Config{
				Modules: config.ModulesConfig{Enabled: []string{"rbac"}},
			},
			Specs: []module.Spec{
				{ID: "user", Builder: module.BuilderFunc(noopBuilder)},
				{ID: "rbac", Dependencies: []string{"user"}, Builder: module.BuilderFunc(noopBuilder)},
			},
		},
		engine.Group("/api"),
		allowAuthService{},
		&authorizer,
	)
	if err != nil {
		t.Fatalf("register module runtime: %v", err)
	}

	assertRegisteredMetadata(t, menuRegistry.Items(), permissionRegistry.Items())
	assertMenuTitleKeysRegistered(t, localizer, menuRegistry.Items())

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/modules/runtime", nil)
	request.Header.Set("Authorization", "Bearer token")
	engine.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, recorder.Code, recorder.Body.String())
	}
	if authorizer.permission != PermissionRead {
		t.Fatalf("expected permission %q, got %q", PermissionRead, authorizer.permission)
	}

	var payload httpx.SuccessResponse[Snapshot]
	if err := json.NewDecoder(recorder.Body).Decode(&payload); err != nil {
		t.Fatalf("decode snapshot response: %v", err)
	}
	if payload.Data.Summary.TotalModules != 2 || payload.Data.Summary.EnabledModules != 1 {
		t.Fatalf("unexpected snapshot summary: %#v", payload.Data.Summary)
	}
	if string(payload.Data.Items[1].RuntimeStatus) != runtimeStatusDegraded {
		t.Fatalf("expected rbac degraded by disabled dependency, got %#v", payload.Data.Items[1])
	}
}

func TestRegisterExposesProtectedDetailRoute(t *testing.T) {
	gin.SetMode(gin.TestMode)

	engine := gin.New()
	if err := Register(
		Registration{
			I18n: i18n.MustNew(config.I18nConfig{
				DefaultLocale:    "zh-CN",
				FallbackLocale:   "en-US",
				SupportedLocales: []string{"zh-CN", "en-US"},
			}),
			MenuRegistry:       menu.NewRegistry(),
			PermissionRegistry: permission.NewRegistry(),
			Specs: []module.Spec{
				{ID: "audit", MigrationPath: []string{"modules/audit/migrations"}, Builder: module.BuilderFunc(noopBuilder)},
			},
		},
		engine.Group("/api"),
		allowAuthService{},
		&recordingAuthorizer{},
	); err != nil {
		t.Fatalf("register module runtime: %v", err)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/modules/runtime/audit", nil)
	request.Header.Set("Authorization", "Bearer token")
	engine.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, recorder.Code, recorder.Body.String())
	}

	var payload httpx.SuccessResponse[Item]
	if err := json.NewDecoder(recorder.Body).Decode(&payload); err != nil {
		t.Fatalf("decode detail response: %v", err)
	}
	if payload.Data.ModuleKey != "audit" || string(payload.Data.SchemaStatus.Status) != schemaStatusDeclared {
		t.Fatalf("unexpected module detail: %#v", payload.Data)
	}
}

func TestRegisterAddsRootMenuWhenServerSectionIsMissing(t *testing.T) {
	gin.SetMode(gin.TestMode)

	localizer := i18n.MustNew(config.I18nConfig{
		DefaultLocale:    "zh-CN",
		FallbackLocale:   "en-US",
		SupportedLocales: []string{"zh-CN", "en-US"},
	})
	menuRegistry := menu.NewRegistry()
	permissionRegistry := permission.NewRegistry()
	engine := gin.New()

	if err := Register(
		Registration{
			I18n:               localizer,
			MenuRegistry:       menuRegistry,
			PermissionRegistry: permissionRegistry,
		},
		engine.Group("/api"),
		allowAuthService{},
		&recordingAuthorizer{},
	); err != nil {
		t.Fatalf("register module runtime: %v", err)
	}

	menus := menuRegistry.Items()
	assertRegisteredMetadata(t, menus, permissionRegistry.Items())
	assertMenuTitleKeysRegistered(t, localizer, menus)
}

func TestRegisterDetailRouteReturnsNotFoundForUnknownModule(t *testing.T) {
	gin.SetMode(gin.TestMode)

	engine := gin.New()
	if err := Register(
		Registration{
			I18n: i18n.MustNew(config.I18nConfig{
				DefaultLocale:    "zh-CN",
				FallbackLocale:   "en-US",
				SupportedLocales: []string{"zh-CN", "en-US"},
			}),
			MenuRegistry:       menu.NewRegistry(),
			PermissionRegistry: permission.NewRegistry(),
			Specs:              []module.Spec{{ID: "audit", Builder: module.BuilderFunc(noopBuilder)}},
		},
		engine.Group("/api"),
		allowAuthService{},
		&recordingAuthorizer{},
	); err != nil {
		t.Fatalf("register module runtime: %v", err)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/modules/runtime/missing", nil)
	request.Header.Set("Authorization", "Bearer token")
	engine.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, recorder.Code)
	}
}

func TestRegisterRejectsMissingRouteDependencies(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name        string
		router      gin.IRouter
		authService moduleapi.AuthService
		authorizer  moduleapi.Authorizer
		want        string
	}{
		{
			name:        "router",
			authService: allowAuthService{},
			authorizer:  &recordingAuthorizer{},
			want:        "module runtime router is unavailable",
		},
		{
			name:       "auth service",
			router:     gin.New().Group("/api"),
			authorizer: &recordingAuthorizer{},
			want:       "module runtime auth service is unavailable",
		},
		{
			name:        "authorizer",
			router:      gin.New().Group("/api"),
			authService: allowAuthService{},
			want:        "module runtime authorizer is unavailable",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registration := Registration{
				I18n: i18n.MustNew(config.I18nConfig{
					DefaultLocale:    "zh-CN",
					FallbackLocale:   "en-US",
					SupportedLocales: []string{"zh-CN", "en-US"},
				}),
				MenuRegistry:       menu.NewRegistry(),
				PermissionRegistry: permission.NewRegistry(),
			}
			err := Register(registration, tt.router, tt.authService, tt.authorizer)
			if err == nil || err.Error() != tt.want {
				t.Fatalf("expected %q, got %v", tt.want, err)
			}
		})
	}
}

func assertRegisteredMetadata(t *testing.T, menus []menu.Item, permissions []permission.Item) {
	t.Helper()

	if len(permissions) != 1 || permissions[0].Code != PermissionRead {
		t.Fatalf("expected module runtime read permission, got %#v", permissions)
	}

	rootCount := 0
	foundRuntimeMenu := false
	for _, item := range menus {
		if item.Path == menuRootPath {
			rootCount++
		}
		if item.Path == menuRuntimePath {
			foundRuntimeMenu = true
			if item.TitleKey != menuModulesRuntimeTitleKey || item.Permission != PermissionRead {
				t.Fatalf("unexpected runtime menu item: %#v", item)
			}
		}
	}
	if rootCount != 1 {
		t.Fatalf("expected existing /server root not to be duplicated, got %d", rootCount)
	}
	if !foundRuntimeMenu {
		t.Fatalf("expected %s menu item in %#v", menuRuntimePath, menus)
	}
}

func assertMenuTitleKeysRegistered(t *testing.T, localizer *i18n.Service, menus []menu.Item) {
	t.Helper()

	for _, item := range menus {
		if item.TitleKey == "" {
			continue
		}

		for _, locale := range []i18n.LocaleTag{i18n.LocaleZHCN, i18n.LocaleENUS} {
			assertMenuTitleKeyRegistered(t, localizer, item, locale)
		}
	}
}

func assertMenuTitleKeyRegistered(t *testing.T, localizer *i18n.Service, item menu.Item, locale i18n.LocaleTag) {
	t.Helper()

	matches := localizer.RegisteredMessageResources(locale, i18n.MessageKey(item.TitleKey))
	if len(matches) == 0 {
		t.Fatalf("menu %q title_key %q is not registered for locale %s", item.Code, item.TitleKey, locale)
	}
	for _, match := range matches {
		assertRegisteredMenuTitleMessage(t, item, locale, match)
	}
}

func assertRegisteredMenuTitleMessage(t *testing.T, item menu.Item, locale i18n.LocaleTag, message i18n.MessageResource) {
	t.Helper()

	if message.Text == "" || message.Text == item.TitleKey {
		t.Fatalf("menu %q title_key %q has invalid registered message for locale %s: %#v", item.Code, item.TitleKey, locale, message)
	}
	if locale == i18n.LocaleENUS && message.Text == item.Title {
		t.Fatalf("menu %q title_key %q falls back to zh-CN title for locale %s: %#v", item.Code, item.TitleKey, locale, message)
	}
}

type allowAuthService struct{}

func (allowAuthService) CurrentUser(context.Context) (*moduleapi.CurrentUser, error) {
	return &moduleapi.CurrentUser{ID: 1, Username: "alice"}, nil
}

func (allowAuthService) ParseAccessToken(context.Context, string) (*moduleapi.AccessTokenClaims, error) {
	return &moduleapi.AccessTokenClaims{
		UserID:    1,
		SessionID: "session-1",
		IssuedAt:  time.Now(),
		ExpiresAt: time.Now().Add(time.Minute),
	}, nil
}

type recordingAuthorizer struct {
	permission string
}

func (a *recordingAuthorizer) Authorize(_ context.Context, _ moduleapi.RequestAuthContext, permission string) error {
	a.permission = permission
	return nil
}
