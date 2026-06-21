// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package notification

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
	"go.uber.org/zap"

	"graft/server/internal/config"
	"graft/server/internal/configregistry"
	"graft/server/internal/container"
	"graft/server/internal/cronx"
	"graft/server/internal/dashboard"
	"graft/server/internal/eventbus"
	"graft/server/internal/i18n"
	"graft/server/internal/menu"
	"graft/server/internal/module"
	"graft/server/internal/moduleapi"
	"graft/server/internal/permission"
	"graft/server/internal/testassert"
	notificationcontract "graft/server/modules/notification/contract"
	notificationstore "graft/server/modules/notification/store"
	systemconfiglocales "graft/server/modules/system-config/locales"
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
		I18n:               mustNewNotificationModuleTestLocalizer(t),
		Services:           services,
		MenuRegistry:       menu.NewRegistry(),
		PermissionRegistry: permission.NewRegistry(),
		ConfigRegistry:     configregistry.NewRegistry(),
	}
	moduleInstance := NewModule(service, publisher)
	if err := moduleInstance.Register(ctx); err != nil {
		t.Fatalf("register module: %v", err)
	}
	if err := moduleInstance.Boot(ctx); !errors.Is(err, container.ErrServiceNotRegistered) {
		t.Fatalf("expected boot to require system-config resolver, got %v", err)
	}

	assertNotificationPermissionsRegistered(t, ctx.PermissionRegistry)
	assertNotificationPublisherRegistered(t, services)
	assertNotificationMenuNotRegistered(t, ctx.MenuRegistry)
	assertNotificationMenuTitleMessage(t, ctx.I18n, i18n.LocaleZHCN, "通知中心")
	assertNotificationMenuTitleMessage(t, ctx.I18n, i18n.LocaleENUS, "Notification Center")
	assertNotificationConfigRegistered(t, ctx.ConfigRegistry)
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
		I18n:               mustNewNotificationModuleTestLocalizer(t),
		EventBus:           eventbus.New(zap.NewNop()),
		Router:             engine.Group("/api"),
		Services:           services,
		MenuRegistry:       menu.NewRegistry(),
		PermissionRegistry: permission.NewRegistry(),
		CronRegistry:       cronx.NewRegistry(),
		ConfigRegistry:     configregistry.NewRegistry(),
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

func assertNotificationConfigRegistered(t *testing.T, registry *configregistry.Registry) {
	t.Helper()

	for _, key := range []string{
		notificationEnabledKey,
		notificationSourceScheduledTaskFailureEnabledKey,
		notificationSourceAuditIncidentEnabledKey,
		notificationDeliveryInAppEnabledKey,
		notificationDisplayKey,
	} {
		if _, ok := registry.Get(key); !ok {
			t.Fatalf("expected notification config %s to be registered", key)
		}
	}
	for _, key := range []string{
		"notification.display.show_read_days",
		"notification.display.popup_limit",
	} {
		if _, ok := registry.Get(key); ok {
			t.Fatalf("old notification display flat key %s must not be registered", key)
		}
	}
}

func TestModuleRegistersNotificationConfigI18nMetadata(t *testing.T) {
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
		I18n:               mustNewNotificationModuleTestLocalizer(t),
		Services:           services,
		MenuRegistry:       menu.NewRegistry(),
		PermissionRegistry: permission.NewRegistry(),
		ConfigRegistry:     configregistry.NewRegistry(),
	}
	if err := NewModule(service, publisher).Register(ctx); err != nil {
		t.Fatalf("register module: %v", err)
	}

	definitions := ctx.ConfigRegistry.Items()
	if len(definitions) != len(notificationConfigDefinitions()) {
		t.Fatalf("expected %d notification config definitions, got %d", len(notificationConfigDefinitions()), len(definitions))
	}
	for _, definition := range definitions {
		assertNotificationConfigDefinitionI18nKeys(t, definition)
		assertNotificationConfigDefinitionMessages(t, ctx.I18n, definition)
	}
	display, ok := ctx.ConfigRegistry.Get(notificationDisplayKey)
	if !ok {
		t.Fatal("expected canonical notification.display config definition")
	}
	assertNotificationDisplayConfigDefinition(t, display)
	assertSingleNotificationConfigMessage(t, ctx.I18n, i18n.LocaleZHCN, display.Key, "ShowReadDaysTitleKey", notificationConfigTitleKey(notificationDisplayShowReadDaysKey))
	assertSingleNotificationConfigMessage(t, ctx.I18n, i18n.LocaleENUS, display.Key, "PopupLimitDescriptionKey", notificationConfigDescriptionKey(notificationDisplayPopupLimitKey))
}

func assertNotificationDisplayConfigDefinition(t *testing.T, definition configregistry.Definition) {
	t.Helper()

	if definition.Type != configregistry.ValueTypeObject {
		t.Fatalf("expected notification.display object type, got %#v", definition)
	}
	if definition.RuntimeApplyMode != configregistry.RuntimeApplyModeUnknown {
		t.Fatalf("expected notification.display runtime apply mode to remain unknown, got %#v", definition.RuntimeApplyMode)
	}
	if string(definition.DefaultValue) != `{"showReadDays":7,"popupLimit":5}` {
		t.Fatalf("expected notification.display object default, got %s", definition.DefaultValue)
	}
	var schema struct {
		Type                 string `json:"type"`
		AdditionalProperties bool   `json:"additionalProperties"`
		Required             []string
		Properties           map[string]struct {
			Type    string          `json:"type"`
			Default json.RawMessage `json:"default"`
			Minimum *float64        `json:"minimum"`
			Maximum *float64        `json:"maximum"`
			XI18n   struct {
				TitleKey       string `json:"titleKey"`
				DescriptionKey string `json:"descriptionKey"`
			} `json:"x-i18n"`
		} `json:"properties"`
	}
	if err := json.Unmarshal(definition.Schema, &schema); err != nil {
		t.Fatalf("decode notification display schema: %v", err)
	}
	if schema.Type != "object" || schema.AdditionalProperties {
		t.Fatalf("expected strict notification display object schema, got %#v", schema)
	}
	if !testassert.SameStringSet(schema.Required, []string{"showReadDays", "popupLimit"}) {
		t.Fatalf("expected notification display required fields, got %#v", schema.Required)
	}
	showReadDays := schema.Properties["showReadDays"]
	if string(showReadDays.Default) != "7" ||
		showReadDays.XI18n.TitleKey != notificationConfigTitleKey(notificationDisplayShowReadDaysKey) ||
		showReadDays.XI18n.DescriptionKey != notificationConfigDescriptionKey(notificationDisplayShowReadDaysKey) {
		t.Fatalf("expected showReadDays schema metadata, got %#v", showReadDays)
	}
	popupLimit := schema.Properties["popupLimit"]
	if string(popupLimit.Default) != "5" ||
		popupLimit.XI18n.TitleKey != notificationConfigTitleKey(notificationDisplayPopupLimitKey) ||
		popupLimit.XI18n.DescriptionKey != notificationConfigDescriptionKey(notificationDisplayPopupLimitKey) {
		t.Fatalf("expected popupLimit schema metadata, got %#v", popupLimit)
	}
}

func TestModuleBootBindsSystemConfigResolverOnce(t *testing.T) {
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
	registerNotificationModuleTestServices(t, services)
	resolver := &notificationModuleTestSystemConfigResolver{values: map[string]bool{
		notificationEnabledKey: false,
	}}
	if err := services.RegisterSingleton((*moduleapi.SystemConfigResolver)(nil), func(container.Resolver) (any, error) {
		return resolver, nil
	}); err != nil {
		t.Fatalf("register system config resolver: %v", err)
	}
	ctx := &module.Context{
		I18n:               mustNewNotificationModuleTestLocalizer(t),
		Services:           services,
		MenuRegistry:       menu.NewRegistry(),
		PermissionRegistry: permission.NewRegistry(),
		ConfigRegistry:     configregistry.NewRegistry(),
	}
	moduleInstance := NewModule(service, publisher)
	if err := moduleInstance.Register(ctx); err != nil {
		t.Fatalf("register module: %v", err)
	}
	if err := moduleInstance.Boot(ctx); err != nil {
		t.Fatalf("boot module: %v", err)
	}

	_, err = publisher.Publish(context.Background(), moduleapi.PublishNotificationInput{
		Title:        "Title",
		Message:      "Message",
		Severity:     moduleapi.NotificationSeverity(notificationcontract.SeverityWarning),
		Category:     moduleapi.NotificationCategory(notificationcontract.CategorySecurity),
		SourceModule: "audit",
		EventType:    "incident",
		Navigation: moduleapi.NotificationNavigation{
			Kind:    moduleapi.NotificationNavigationKind(notificationcontract.NavigationAuditLog),
			Payload: json.RawMessage(`{}`),
		},
		Metadata: json.RawMessage(`{}`),
		Target: moduleapi.NotificationTarget{
			Type: moduleapi.NotificationTargetType(notificationcontract.TargetUser),
			Ref:  "42",
		},
	})
	if err != nil {
		t.Fatalf("publish notification: %v", err)
	}
	if resolver.calls != 1 {
		t.Fatalf("expected bound resolver to be called once, got %d", resolver.calls)
	}
}

func assertNotificationConfigDefinitionI18nKeys(t *testing.T, definition configregistry.Definition) {
	t.Helper()

	required := map[string]string{
		"DomainKey":           definition.DomainKey,
		"GroupKey":            definition.GroupKey,
		"GroupDescriptionKey": definition.GroupDescriptionKey,
		"TitleKey":            definition.TitleKey,
		"DescriptionKey":      definition.DescriptionKey,
	}
	for field, value := range required {
		if strings.TrimSpace(value) == "" {
			t.Fatalf("expected notification config %s to have %s", definition.Key, field)
		}
	}
	if definition.TitleKey != notificationConfigTitleKey(definition.Key) {
		t.Fatalf("expected notification config %s title key %q, got %q", definition.Key, notificationConfigTitleKey(definition.Key), definition.TitleKey)
	}
	if definition.DescriptionKey != notificationConfigDescriptionKey(definition.Key) {
		t.Fatalf("expected notification config %s description key %q, got %q", definition.Key, notificationConfigDescriptionKey(definition.Key), definition.DescriptionKey)
	}
	switch definition.Type {
	case configregistry.ValueTypeBoolean:
		if definition.RuntimeApplyMode != configregistry.RuntimeApplyModeRuntimeHot {
			t.Fatalf("expected boolean notification config %s to be runtime-hot, got %#v", definition.Key, definition.RuntimeApplyMode)
		}
	default:
		if definition.Key != notificationDisplayKey && definition.RuntimeApplyMode != configregistry.RuntimeApplyModeUnknown {
			t.Fatalf("expected non-boolean notification config %s to remain unknown, got %#v", definition.Key, definition.RuntimeApplyMode)
		}
	}
}

func assertNotificationConfigDefinitionMessages(t *testing.T, localizer *i18n.Service, definition configregistry.Definition) {
	t.Helper()

	for _, locale := range []i18n.LocaleTag{i18n.LocaleZHCN, i18n.LocaleENUS} {
		for field, key := range map[string]string{
			"DomainKey":           definition.DomainKey,
			"GroupKey":            definition.GroupKey,
			"GroupDescriptionKey": definition.GroupDescriptionKey,
			"TitleKey":            definition.TitleKey,
			"DescriptionKey":      definition.DescriptionKey,
		} {
			assertSingleNotificationConfigMessage(t, localizer, locale, definition.Key, field, key)
		}
	}
}

func assertSingleNotificationConfigMessage(t *testing.T, localizer *i18n.Service, locale i18n.LocaleTag, configKey string, field string, key string) {
	t.Helper()

	matches := localizer.RegisteredMessageResources(locale, i18n.MessageKey(key))
	if len(matches) != 1 {
		t.Fatalf("expected one %s message for notification config %s locale %s key %q, got %#v", field, configKey, locale, key, matches)
	}
	if strings.TrimSpace(matches[0].Text) == "" {
		t.Fatalf("expected non-empty %s message for notification config %s locale %s key %q", field, configKey, locale, key)
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

// registerNotificationModuleTestServices 注册 notification 模块 Register 阶段所需的跨模块测试服务。
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

func mustNewNotificationModuleTestLocalizer(t *testing.T) *i18n.Service {
	t.Helper()

	localizer := i18n.MustNew(config.I18nConfig{
		DefaultLocale:    "zh-CN",
		FallbackLocale:   "zh-CN",
		SupportedLocales: []string{"zh-CN", "en-US"},
	})
	resources, err := systemconfiglocales.EmbeddedLocaleResources()
	if err != nil {
		t.Fatalf("load system-config locale resources: %v", err)
	}
	if err := localizer.RegisterEmbeddedLocaleResources(resources); err != nil {
		t.Fatalf("register system-config locale resources: %v", err)
	}
	return localizer
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

type notificationModuleTestSystemConfigResolver struct {
	values map[string]bool
	calls  int
}

func (r *notificationModuleTestSystemConfigResolver) IsBooleanConfigEnabled(_ context.Context, key string, fallback bool) bool {
	if r == nil || r.values == nil {
		return fallback
	}
	r.calls++
	value, ok := r.values[key]
	if !ok {
		return fallback
	}
	return value
}

func (r *notificationModuleTestSystemConfigResolver) ResolveDefaultConfig(_ context.Context, _ string) (string, error) {
	return "", errors.New("config unavailable")
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
