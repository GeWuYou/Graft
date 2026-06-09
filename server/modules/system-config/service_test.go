// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package systemconfig

import (
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"graft/server/internal/config"
	"graft/server/internal/configregistry"
	"graft/server/internal/container"
	"graft/server/internal/dashboard"
	"graft/server/internal/i18n"
	"graft/server/internal/menu"
	"graft/server/internal/module"
	"graft/server/internal/moduleapi"
	"graft/server/internal/permission"
	systemconfigcontract "graft/server/modules/system-config/contract"
	systemconfigstore "graft/server/modules/system-config/store"
)

func TestServiceMasksSensitiveValues(t *testing.T) {
	service := newTestService(t, configregistry.Definition{
		Key:          "auth.jwt_secret",
		Module:       "auth",
		Group:        "security",
		Title:        "JWT Secret",
		Type:         configregistry.ValueTypeString,
		DefaultValue: json.RawMessage(`"secret"`),
		Sensitive:    true,
	})

	item, err := service.Get(context.Background(), "auth.jwt_secret")
	if err != nil {
		t.Fatalf("get sensitive config: %v", err)
	}
	if !item.Masked || item.EffectiveValue != nil || item.DefaultValue != nil {
		t.Fatalf("sensitive values must be masked, got %#v", item)
	}
}

func TestServiceListsDefaultsAndStoresOverridesOnly(t *testing.T) {
	repo := newMemoryRepo()
	service := newTestServiceWithRepo(t, repo, configregistry.Definition{
		Key:          "scheduler.timeout",
		Module:       "scheduler",
		Group:        "runtime",
		Title:        "Timeout",
		Type:         configregistry.ValueTypeString,
		DefaultValue: json.RawMessage(`"30s"`),
	})

	assertDefaultVisibleWithoutOverride(t, service, repo)
	assertUpdateStoresOneOverride(t, service, repo)
	assertResetDeletesOverride(t, service, repo)
}

func TestServiceResolveDefaultConfigReturnsEffectiveOverride(t *testing.T) {
	repo := newMemoryRepo()
	service := newTestServiceWithRepo(t, repo, configregistry.Definition{
		Key:          "httpx.access-log-retention-cleanup",
		Module:       "core.httpx",
		Group:        "log.retention",
		Title:        "Access log retention cleanup",
		Type:         configregistry.ValueTypeObject,
		DefaultValue: json.RawMessage(`{"retentionDays":30,"batchSize":1000}`),
	})
	if _, err := service.Update(
		context.Background(),
		"httpx.access-log-retention-cleanup",
		json.RawMessage(`{"retentionDays":45,"batchSize":2000}`),
		nil,
	); err != nil {
		t.Fatalf("update override: %v", err)
	}

	value, err := service.ResolveDefaultConfig(context.Background(), "httpx.access-log-retention-cleanup")
	if err != nil {
		t.Fatalf("resolve default config: %v", err)
	}
	if value != `{"retentionDays":45,"batchSize":2000}` {
		t.Fatalf("expected effective override, got %s", value)
	}
}

func TestServiceResolveDefaultConfigRejectsSensitiveDefinitions(t *testing.T) {
	service := newTestService(t, configregistry.Definition{
		Key:          "auth.jwt_secret",
		Module:       "auth",
		Group:        "security",
		Title:        "JWT Secret",
		Type:         configregistry.ValueTypeString,
		DefaultValue: json.RawMessage(`"secret"`),
		Sensitive:    true,
	})

	if _, err := service.ResolveDefaultConfig(context.Background(), "auth.jwt_secret"); !errors.Is(err, errSensitiveConfig) {
		t.Fatalf("expected sensitive default config error, got %v", err)
	}
}

func TestCurrentUserIDReadsRequestAuthContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	userID := uint64(42)
	request := httptest.NewRequest("PUT", "/system-config/scheduler.timeout", nil)
	request = request.WithContext(moduleapi.WithRequestAuthContext(request.Context(), moduleapi.RequestAuthContext{
		User: &moduleapi.CurrentUser{ID: userID},
	}))
	ginCtx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ginCtx.Request = request

	got := currentUserID(ginCtx)
	if got == nil || *got != userID {
		t.Fatalf("expected current user id %d, got %#v", userID, got)
	}
}

func TestModuleRegisterRequiresUserService(t *testing.T) {
	service := newTestService(t, configregistry.Definition{
		Key:          "scheduler.timeout",
		Module:       "scheduler",
		Group:        "runtime",
		Title:        "Timeout",
		Type:         configregistry.ValueTypeString,
		DefaultValue: json.RawMessage(`"30s"`),
	})
	moduleInstance, err := NewModule(service)
	if err != nil {
		t.Fatalf("create module: %v", err)
	}

	err = moduleInstance.Register(&module.Context{
		Services: container.New(),
	})
	if err == nil {
		t.Fatalf("expected missing user service error")
	}
	if !errors.Is(err, container.ErrServiceNotRegistered) {
		t.Fatalf("expected service not registered error, got %v", err)
	}
}

func TestModuleRegisterBindsUserServiceForUpdatedByUsername(t *testing.T) {
	repo := newMemoryRepo()
	service := newTestServiceWithRepoAndUsers(t, repo, nil, configregistry.Definition{
		Key:          "scheduler.timeout",
		Module:       "scheduler",
		Group:        "runtime",
		Title:        "Timeout",
		Type:         configregistry.ValueTypeString,
		DefaultValue: json.RawMessage(`"30s"`),
	})
	dashboardRegistry := registerSystemConfigModuleWithUserService(t, service)
	assertSystemConfigQuickLink(t, dashboardRegistry)

	userID := uint64(42)
	item, err := service.Update(context.Background(), "scheduler.timeout", json.RawMessage(`"60s"`), &userID)
	if err != nil {
		t.Fatalf("update override: %v", err)
	}
	if item.UpdatedByName != "alice" {
		t.Fatalf("expected updated_by username alice, got %#v", item.UpdatedByName)
	}
	mapped := toItem(item)
	if mapped.UpdatedByUsername == nil || *mapped.UpdatedByUsername != "alice" {
		t.Fatalf("expected response username alice, got %#v", mapped.UpdatedByUsername)
	}
}

func registerSystemConfigModuleWithUserService(t *testing.T, service *Service) *dashboard.Registry {
	t.Helper()

	moduleInstance, err := NewModule(service)
	if err != nil {
		t.Fatalf("create module: %v", err)
	}

	services := container.New()
	if err := services.RegisterSingleton((*moduleapi.UserService)(nil), func(container.Resolver) (any, error) {
		return testUserService{
			users: map[uint64]moduleapi.UserSummary{
				42: {ID: 42, Username: "alice", Display: "Alice"},
			},
		}, nil
	}); err != nil {
		t.Fatalf("register user service: %v", err)
	}

	localizer, err := i18n.New(config.I18nConfig{
		SupportedLocales: []string{"zh-CN", "en-US"},
	})
	if err != nil {
		t.Fatalf("create i18n service: %v", err)
	}

	dashboardRegistry := dashboard.NewRegistry()
	if err := moduleInstance.Register(&module.Context{
		Services:           services,
		I18n:               localizer,
		MenuRegistry:       menu.NewRegistry(),
		PermissionRegistry: permission.NewRegistry(),
		DashboardRegistry:  dashboardRegistry,
	}); err != nil {
		t.Fatalf("register system config module: %v", err)
	}

	return dashboardRegistry
}

func assertSystemConfigQuickLink(t *testing.T, registry *dashboard.Registry) {
	t.Helper()

	quickLinks := registry.QuickLinks()
	if len(quickLinks) != 1 {
		t.Fatalf("expected system-config quick link, got %#v", quickLinks)
	}

	link := quickLinks[0]
	if link.ID != systemConfigQuickLinkID ||
		link.ModuleKey != moduleID ||
		link.TitleKey != systemconfigcontract.SystemConfigMenuTitle.String() ||
		link.RouteLocation != systemconfigcontract.SystemConfigMenuPath ||
		link.Order != systemConfigQuickLinkOrder {
		t.Fatalf("unexpected system-config quick link: %#v", link)
	}
	if len(link.RequiredPermissions) != 1 || link.RequiredPermissions[0] != systemconfigcontract.SystemConfigReadPermission.String() {
		t.Fatalf("unexpected system-config quick link permissions: %#v", link.RequiredPermissions)
	}
}

func assertDefaultVisibleWithoutOverride(t *testing.T, service *Service, repo *memoryRepo) {
	t.Helper()

	items, err := service.List(context.Background())
	if err != nil {
		t.Fatalf("list default config: %v", err)
	}
	if len(items) != 1 || items[0].HasOverride || string(items[0].EffectiveValue) != `"30s"` {
		t.Fatalf("expected listed module default without override, got %#v", items)
	}
	if len(repo.values) != 0 {
		t.Fatalf("expected list to avoid copying defaults into overrides, got %d rows", len(repo.values))
	}

	item, err := service.Get(context.Background(), "scheduler.timeout")
	if err != nil {
		t.Fatalf("get default config: %v", err)
	}
	if item.HasOverride || string(item.EffectiveValue) != `"30s"` {
		t.Fatalf("expected get to return module default without override, got %#v", item)
	}
	if item.Status != ValueStatusDefault || item.CreatedAt != nil || item.UpdatedAt != nil {
		t.Fatalf("expected default status without audit fields, got %#v", item)
	}
}

func assertUpdateStoresOneOverride(t *testing.T, service *Service, repo *memoryRepo) {
	t.Helper()

	userID := uint64(42)
	item, err := service.Update(context.Background(), "scheduler.timeout", json.RawMessage(`"60s"`), &userID)
	if err != nil {
		t.Fatalf("update override: %v", err)
	}
	if !item.HasOverride || string(item.EffectiveValue) != `"60s"` {
		t.Fatalf("expected effective override, got %#v", item)
	}
	if item.Status != ValueStatusModified {
		t.Fatalf("expected modified status, got %#v", item.Status)
	}
	assertNewOverrideAudit(t, item, userID)
	if len(repo.values) != 1 {
		t.Fatalf("expected only one override row, got %d", len(repo.values))
	}

	updatingUserID := uint64(7)
	updated, err := service.Update(context.Background(), "scheduler.timeout", json.RawMessage(`"90s"`), &updatingUserID)
	if err != nil {
		t.Fatalf("update existing override: %v", err)
	}
	assertUpdatedOverrideAudit(t, updated, userID, updatingUserID)
}

func assertNewOverrideAudit(t *testing.T, item ValueSnapshot, userID uint64) {
	t.Helper()

	if item.CreatedBy == nil || *item.CreatedBy != userID {
		t.Fatalf("expected created_by user %d on override, got %#v", userID, item)
	}
	if item.UpdatedBy == nil || *item.UpdatedBy != userID {
		t.Fatalf("expected updated_by user %d on override, got %#v", userID, item)
	}
	if item.CreatedAt == nil || item.UpdatedAt == nil {
		t.Fatalf("expected audit timestamps on override, got %#v", item)
	}
}

func assertUpdatedOverrideAudit(t *testing.T, item ValueSnapshot, createdBy uint64, updatedBy uint64) {
	t.Helper()

	if item.CreatedBy == nil || *item.CreatedBy != createdBy {
		t.Fatalf("expected created_by to stay %d, got %#v", createdBy, item)
	}
	if item.UpdatedBy == nil || *item.UpdatedBy != updatedBy {
		t.Fatalf("expected updated_by to change to %d, got %#v", updatedBy, item)
	}
}

func assertResetDeletesOverride(t *testing.T, service *Service, repo *memoryRepo) {
	t.Helper()

	item, err := service.Reset(context.Background(), "scheduler.timeout")
	if err != nil {
		t.Fatalf("reset override: %v", err)
	}
	if item.HasOverride || string(item.EffectiveValue) != `"30s"` {
		t.Fatalf("expected reset to return module default without override, got %#v", item)
	}
	if item.Status != ValueStatusDefault || item.CreatedBy != nil || item.UpdatedBy != nil {
		t.Fatalf("expected reset snapshot without audit fields, got %#v", item)
	}
	if len(repo.values) != 0 {
		t.Fatalf("expected reset to delete override row, got %d rows", len(repo.values))
	}
}

func TestServiceRejectsMismatchedValueType(t *testing.T) {
	service := newTestService(t, configregistry.Definition{
		Key:          "audit.retention_days",
		Module:       "audit",
		Group:        "retention",
		Title:        "Retention Days",
		Type:         configregistry.ValueTypeInteger,
		DefaultValue: json.RawMessage(`30`),
	})

	if _, err := service.Update(context.Background(), "audit.retention_days", json.RawMessage(`"30"`), nil); err == nil {
		t.Fatal("expected value type error")
	}
}

func TestServiceRejectsObjectValueOutsideSchemaConstraints(t *testing.T) {
	service := newTestService(t, configregistry.Definition{
		Key:          "httpx.access-log-retention-cleanup",
		Module:       "core.httpx",
		Group:        "log.retention",
		Title:        "Access Log Retention Cleanup",
		Type:         configregistry.ValueTypeObject,
		Schema:       json.RawMessage(`{"type":"object","properties":{"retentionDays":{"type":"integer","minimum":1,"maximum":365},"batchSize":{"type":"integer","minimum":1,"maximum":10000}},"additionalProperties":false}`),
		DefaultValue: json.RawMessage(`{"retentionDays":30,"batchSize":1000}`),
	})

	if _, err := service.Update(context.Background(), "httpx.access-log-retention-cleanup", json.RawMessage(`{"retentionDays":366,"batchSize":1000}`), nil); err == nil {
		t.Fatal("expected schema validation error")
	}
	if _, err := service.Update(context.Background(), "httpx.access-log-retention-cleanup", json.RawMessage(`{"retentionDays":30,"batchSize":1000,"extra":true}`), nil); err == nil {
		t.Fatal("expected additional property validation error")
	}
}

func TestToItemIncludesLocalizationMetadataAndStructuredSchema(t *testing.T) {
	item := toItem(ValueSnapshot{
		Definition: configregistry.Definition{
			Key:            "httpx.access-log-retention-cleanup",
			Module:         "core.httpx",
			Group:          "log.retention",
			GroupKey:       "systemConfig.groups.coreHttpxLogRetention",
			GroupLabel:     "core.httpx / log.retention",
			Title:          "Access log retention cleanup",
			TitleKey:       "systemConfig.items.accessLogRetentionCleanup.title",
			Description:    "Default cleanup configuration for access-log retention jobs.",
			DescriptionKey: "systemConfig.items.accessLogRetentionCleanup.description",
			Tags:           []string{"httpx", "log.retention"},
			Type:           configregistry.ValueTypeObject,
			Schema: json.RawMessage(
				`{"type":"object","properties":{"retentionDays":{"type":"integer","title":"Log retention days","x-i18n":{"titleKey":"systemConfig.fields.retentionDays.title","unitKey":"systemConfig.units.days"}}}}`,
			),
			DefaultValue: json.RawMessage(`{"retentionDays":30}`),
		},
		DefaultValue:   json.RawMessage(`{"retentionDays":30}`),
		EffectiveValue: json.RawMessage(`{"retentionDays":30}`),
	})

	if item.GroupKey == nil || *item.GroupKey != "systemConfig.groups.coreHttpxLogRetention" {
		t.Fatalf("expected group key in response, got %#v", item.GroupKey)
	}
	if item.TitleKey == nil || *item.TitleKey != "systemConfig.items.accessLogRetentionCleanup.title" {
		t.Fatalf("expected title key in response, got %#v", item.TitleKey)
	}
	if item.Tags == nil || len(*item.Tags) != 2 || (*item.Tags)[0] != "httpx" {
		t.Fatalf("expected tags in response, got %#v", item.Tags)
	}
	properties, ok := item.ConfigSchema["properties"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected structured config schema properties, got %#v", item.ConfigSchema)
	}
	retentionDays, ok := properties["retentionDays"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected retentionDays schema, got %#v", properties)
	}
	i18nExtension, ok := retentionDays["x-i18n"].(map[string]interface{})
	if !ok || i18nExtension["unitKey"] != "systemConfig.units.days" {
		t.Fatalf("expected x-i18n unit metadata, got %#v", retentionDays)
	}
}

func newTestService(t *testing.T, definition configregistry.Definition) *Service {
	t.Helper()
	return newTestServiceWithRepo(t, newMemoryRepo(), definition)
}

func newTestServiceWithRepo(t *testing.T, repo *memoryRepo, definition configregistry.Definition) *Service {
	t.Helper()
	return newTestServiceWithRepoAndUsers(t, repo, testUserService{
		users: map[uint64]moduleapi.UserSummary{
			7:  {ID: 7, Username: "bob", Display: "Bob"},
			42: {ID: 42, Username: "alice", Display: "Alice"},
		},
	}, definition)
}

func newTestServiceWithRepoAndUsers(
	t *testing.T,
	repo *memoryRepo,
	users moduleapi.UserService,
	definition configregistry.Definition,
) *Service {
	t.Helper()
	registry := configregistry.NewRegistry()
	if err := registry.Register(definition); err != nil {
		t.Fatalf("register definition: %v", err)
	}
	service, err := NewService(registry, repo, users)
	if err != nil {
		t.Fatalf("create service: %v", err)
	}
	return service
}

type memoryRepo struct {
	values map[string]json.RawMessage
	audit  map[string]systemconfigstore.Override
}

func newMemoryRepo() *memoryRepo {
	return &memoryRepo{
		values: make(map[string]json.RawMessage),
		audit:  make(map[string]systemconfigstore.Override),
	}
}

func (r *memoryRepo) GetOverride(_ context.Context, key string) (systemconfigstore.Override, error) {
	value, ok := r.values[key]
	if !ok {
		return systemconfigstore.Override{}, systemconfigstore.ErrOverrideNotFound
	}
	override := r.audit[key]
	override.Key = key
	override.Value = cloneRawMessage(value)
	return override, nil
}

func (r *memoryRepo) SetOverride(_ context.Context, key string, value json.RawMessage, userID *uint64) (systemconfigstore.Override, error) {
	if len(value) == 0 {
		return systemconfigstore.Override{}, errors.New("value is required")
	}
	r.values[key] = cloneRawMessage(value)
	override := r.audit[key]
	now := time.Now().UTC()
	override.Key = key
	override.Value = cloneRawMessage(value)
	if override.CreatedAt.IsZero() {
		override.CreatedAt = now
		override.CreatedBy = cloneUint64Pointer(userID)
	}
	override.UpdatedAt = now
	override.UpdatedBy = cloneUint64Pointer(userID)
	r.audit[key] = override
	return override, nil
}

func (r *memoryRepo) DeleteOverride(_ context.Context, key string) error {
	delete(r.values, key)
	delete(r.audit, key)
	return nil
}

type testUserService struct {
	users map[uint64]moduleapi.UserSummary
}

func (s testUserService) GetUserByID(_ context.Context, id uint64) (moduleapi.UserSummary, error) {
	user, ok := s.users[id]
	if !ok {
		return moduleapi.UserSummary{}, moduleapi.ErrUserNotFound
	}
	return user, nil
}

func (s testUserService) CountUsers(context.Context) (int, error) {
	return len(s.users), nil
}
