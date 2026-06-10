// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package notification

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
	"go.uber.org/zap"

	"graft/server/internal/config"
	"graft/server/internal/container"
	"graft/server/internal/cronx"
	"graft/server/internal/dashboard"
	"graft/server/internal/eventbus"
	"graft/server/internal/i18n"
	"graft/server/internal/menu"
	"graft/server/internal/module"
	"graft/server/internal/moduleapi"
	"graft/server/internal/permission"
	notificationcontract "graft/server/modules/notification/contract"
	notificationstore "graft/server/modules/notification/store"
)

func TestModuleRegistersPermissionsAndPublisher(t *testing.T) {
	repository := &moduleTestRepository{}
	service, err := NewService(repository)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	publisher, err := NewPublisher(repository)
	if err != nil {
		t.Fatalf("new publisher: %v", err)
	}

	services := container.New()
	if err := services.RegisterSingleton((*moduleapi.RBACAccessService)(nil), func(container.Resolver) (any, error) {
		return permissionFanoutRBAC{userIDs: []uint64{42}}, nil
	}); err != nil {
		t.Fatalf("register rbac access service: %v", err)
	}
	ctx := &module.Context{
		I18n:               i18n.MustNew(config.I18nConfig{DefaultLocale: "zh-CN", FallbackLocale: "zh-CN", SupportedLocales: []string{"zh-CN", "en-US"}}),
		Services:           services,
		MenuRegistry:       menu.NewRegistry(),
		PermissionRegistry: permission.NewRegistry(),
	}
	if err := NewModule(service, publisher).Register(ctx); err != nil {
		t.Fatalf("register module: %v", err)
	}

	assertNotificationPermissionsRegistered(t, ctx.PermissionRegistry)
	assertNotificationPublisherRegistered(t, services)
	assertNotificationMenuNotRegistered(t, ctx.MenuRegistry)
	assertNotificationMenuTitleMessage(t, ctx.I18n, i18n.LocaleZHCN, "通知中心")
	assertNotificationMenuTitleMessage(t, ctx.I18n, i18n.LocaleENUS, "Notification Center")
}

func assertNotificationPermissionsRegistered(t *testing.T, registry *permission.Registry) {
	t.Helper()

	registered := make(map[string]struct{}, len(registry.Items()))
	for _, item := range registry.Items() {
		registered[item.Code] = struct{}{}
	}
	for _, code := range []string{
		notificationcontract.NotificationViewPermission.String(),
		notificationcontract.NotificationReadPermission.String(),
		notificationcontract.NotificationManagePermission.String(),
	} {
		if _, ok := registered[code]; !ok {
			t.Fatalf("expected permission %s to be registered", code)
		}
	}
}

func assertNotificationPublisherRegistered(t *testing.T, services *container.Container) {
	t.Helper()

	resolved, err := services.Resolve((*moduleapi.NotificationPublisher)(nil))
	if err != nil {
		t.Fatalf("resolve notification publisher: %v", err)
	}
	if _, ok := resolved.(moduleapi.NotificationPublisher); !ok {
		t.Fatalf("unexpected publisher service type %T", resolved)
	}
}

func assertNotificationMenuNotRegistered(t *testing.T, registry *menu.Registry) {
	t.Helper()

	menus := registry.Items()
	for _, item := range menus {
		if item.Path == "/notifications" || item.Code == "notification.list" {
			t.Fatalf("notification center must not be registered in sidebar menus, got %#v", menus)
		}
	}
}

func TestModuleRegisterMountsNotificationRoutesUnderInjectedAPIRoot(t *testing.T) {
	gin.SetMode(gin.TestMode)
	now := time.Date(2026, 6, 9, 8, 0, 0, 0, time.UTC)
	repository := &moduleTestRepository{
		items: []notificationstore.Notification{routeTestNotification(100, 42, "Unread item", now, nil)},
	}
	service, err := NewService(repository)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	publisher, err := NewPublisher(repository)
	if err != nil {
		t.Fatalf("new publisher: %v", err)
	}

	engine := gin.New()
	services := container.New()
	registerNotificationModuleTestServices(t, services)
	ctx := &module.Context{
		Logger:             zap.NewNop(),
		Config:             &config.Config{},
		I18n:               i18n.MustNew(config.I18nConfig{DefaultLocale: "zh-CN", FallbackLocale: "zh-CN", SupportedLocales: []string{"zh-CN", "en-US"}}),
		EventBus:           eventbus.New(zap.NewNop()),
		Router:             engine.Group("/api"),
		Services:           services,
		MenuRegistry:       menu.NewRegistry(),
		PermissionRegistry: permission.NewRegistry(),
		CronRegistry:       cronx.NewRegistry(),
		DashboardRegistry:  dashboard.NewRegistry(),
	}

	if err := NewModule(service, publisher).Register(ctx); err != nil {
		t.Fatalf("register module: %v", err)
	}

	request := httptest.NewRequest(http.MethodGet, "/api/notifications?page_size=5", nil)
	request.Header.Set("Authorization", "Bearer token")
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected /api/notifications to be mounted, got status %d body=%s", recorder.Code, recorder.Body.String())
	}

	var response struct {
		Data struct {
			Items []json.RawMessage `json:"items"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(response.Data.Items) != 1 {
		t.Fatalf("expected one notification item, got %d", len(response.Data.Items))
	}
}

func TestDescriptorBuildDoesNotResolveRBACAccessService(t *testing.T) {
	t.Parallel()

	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			t.Fatalf("close sqlite db: %v", err)
		}
	}()

	services := container.New()
	if err := services.RegisterSingleton((*sql.DB)(nil), func(container.Resolver) (any, error) {
		return db, nil
	}); err != nil {
		t.Fatalf("register sql db: %v", err)
	}

	descriptor := NewModuleSpec()
	if _, err := descriptor.Build(module.BuildContext{Services: services}); err != nil {
		t.Fatalf("build notification without rbac access service: %v", err)
	}
}

type moduleTestRepository struct {
	items []notificationstore.Notification
}

func registerNotificationModuleTestServices(t *testing.T, services *container.Container) {
	t.Helper()
	if err := services.RegisterSingleton((*moduleapi.AuthService)(nil), func(container.Resolver) (any, error) {
		return notificationModuleTestAuthService{}, nil
	}); err != nil {
		t.Fatalf("register auth service: %v", err)
	}
	if err := services.RegisterSingleton((*moduleapi.Authorizer)(nil), func(container.Resolver) (any, error) {
		return notificationModuleTestAuthorizer{}, nil
	}); err != nil {
		t.Fatalf("register authorizer: %v", err)
	}
	if err := services.RegisterSingleton((*moduleapi.RBACAccessService)(nil), func(container.Resolver) (any, error) {
		return permissionFanoutRBAC{userIDs: []uint64{42}}, nil
	}); err != nil {
		t.Fatalf("register rbac access service: %v", err)
	}
}

func assertNotificationMenuTitleMessage(t *testing.T, localizer *i18n.Service, locale i18n.LocaleTag, expected string) {
	t.Helper()

	matches := localizer.RegisteredMessageResources(locale, i18n.MessageKey(notificationcontract.NotificationMenuTitle.String()))
	if len(matches) != 1 {
		t.Fatalf("expected one notification menu title message for %s, got %#v", locale, matches)
	}
	if matches[0].Text != expected {
		t.Fatalf("expected notification menu title %q for %s, got %#v", expected, locale, matches[0])
	}
}

type notificationModuleTestAuthService struct{}

func (notificationModuleTestAuthService) CurrentUser(context.Context) (*moduleapi.CurrentUser, error) {
	return &moduleapi.CurrentUser{ID: 42, Username: "alice", DisplayName: "Alice"}, nil
}

func (notificationModuleTestAuthService) ParseAccessToken(context.Context, string) (*moduleapi.AccessTokenClaims, error) {
	return &moduleapi.AccessTokenClaims{
		UserID:       42,
		SessionID:    "session-1",
		TokenVersion: 1,
		IssuedAt:     time.Now().UTC(),
		ExpiresAt:    time.Now().UTC().Add(time.Minute),
	}, nil
}

type notificationModuleTestAuthorizer struct{}

func (notificationModuleTestAuthorizer) Authorize(context.Context, moduleapi.RequestAuthContext, string) error {
	return nil
}

func (r *moduleTestRepository) CreateEvent(context.Context, notificationstore.CreateEventInput) (notificationstore.Event, bool, error) {
	return notificationstore.Event{}, false, nil
}

func (r *moduleTestRepository) CreateDeliveries(context.Context, []notificationstore.CreateDeliveryInput) ([]notificationstore.Delivery, error) {
	return nil, nil
}

func (r *moduleTestRepository) List(_ context.Context, query notificationstore.ListQuery) (notificationstore.ListResult, error) {
	items := make([]notificationstore.Notification, 0, len(r.items))
	for _, item := range r.items {
		if item.Delivery.RecipientUserID != query.RecipientUserID {
			continue
		}
		items = append(items, item)
	}
	return notificationstore.ListResult{Items: items, Total: len(items)}, nil
}

func (r *moduleTestRepository) Get(context.Context, uint64, uint64) (notificationstore.Notification, error) {
	return notificationstore.Notification{}, nil
}

func (r *moduleTestRepository) UnreadCount(context.Context, uint64) (int, error) {
	return 0, nil
}

func (r *moduleTestRepository) MarkRead(context.Context, uint64, uint64, time.Time) (notificationstore.Delivery, error) {
	return notificationstore.Delivery{}, nil
}

func (r *moduleTestRepository) MarkAllRead(context.Context, uint64, time.Time) (int, error) {
	return 0, nil
}

func (r *moduleTestRepository) MarkAllReadMatching(context.Context, notificationstore.ListQuery, time.Time) (int, error) {
	return 0, nil
}

func (r *moduleTestRepository) DeleteDelivery(context.Context, uint64, uint64, time.Time) error {
	return nil
}
