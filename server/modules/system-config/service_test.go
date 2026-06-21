// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package systemconfig

import (
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"graft/server/internal/cachex"
	cachebackend "graft/server/internal/cachex/backend"
	"graft/server/internal/config"
	"graft/server/internal/configregistry"
	"graft/server/internal/container"
	generated "graft/server/internal/contract/openapi/generated"
	"graft/server/internal/dashboard"
	"graft/server/internal/i18n"
	"graft/server/internal/menu"
	"graft/server/internal/module"
	"graft/server/internal/moduleapi"
	"graft/server/internal/permission"
	systemconfiglocales "graft/server/modules/system-config/locales"
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

func TestServiceIsBooleanConfigEnabledUsesEffectiveValueAndFallback(t *testing.T) {
	service := newTestService(t, configregistry.Definition{
		Key:          "notification.enabled",
		Module:       "notification",
		Domain:       "notification",
		Group:        "notification.general",
		Title:        "Notification enabled",
		Type:         configregistry.ValueTypeBoolean,
		DefaultValue: json.RawMessage(`true`),
	})

	if got := service.IsBooleanConfigEnabled(context.Background(), "notification.enabled", false); !got {
		t.Fatalf("expected boolean default true")
	}
	if _, err := service.Update(context.Background(), "notification.enabled", json.RawMessage(`false`), nil); err != nil {
		t.Fatalf("update boolean config: %v", err)
	}
	if got := service.IsBooleanConfigEnabled(context.Background(), "notification.enabled", true); got {
		t.Fatalf("expected boolean override false")
	}
	if got := service.IsBooleanConfigEnabled(context.Background(), "missing.key", true); !got {
		t.Fatalf("expected missing boolean config to use fallback")
	}
}

func TestServiceCachesFullOverrideSnapshotAcrossReads(t *testing.T) {
	repo := newMemoryRepo()
	service := newTestServiceWithRepo(t, repo, configregistry.Definition{
		Key:          "notification.enabled",
		Module:       "notification",
		Group:        "notification.general",
		Title:        "Notification enabled",
		Type:         configregistry.ValueTypeBoolean,
		DefaultValue: json.RawMessage(`true`),
	})
	if _, err := service.Update(context.Background(), "notification.enabled", json.RawMessage(`false`), nil); err != nil {
		t.Fatalf("seed override: %v", err)
	}
	service.invalidateSnapshotCacheForKey("notification.enabled", snapshotInvalidationActionUpdate)
	repo.resetReadCounters()

	first, err := service.Get(context.Background(), "notification.enabled")
	if err != nil {
		t.Fatalf("first cached get: %v", err)
	}
	second, err := service.Get(context.Background(), "notification.enabled")
	if err != nil {
		t.Fatalf("second cached get: %v", err)
	}
	if string(first.EffectiveValue) != "false" || string(second.EffectiveValue) != "false" {
		t.Fatalf("expected cached override value false, got %#v and %#v", first, second)
	}
	if _, err := service.List(context.Background()); err != nil {
		t.Fatalf("list cached configs: %v", err)
	}
	if repo.listOverridesCalls() != 1 {
		t.Fatalf("expected one snapshot load for repeated reads, got %d", repo.listOverridesCalls())
	}
	if repo.getOverrideCalls() != 0 {
		t.Fatalf("expected snapshot cache reads to avoid per-key override lookups, got %d", repo.getOverrideCalls())
	}
}

func TestServiceInvalidatesLocalSnapshotAfterUpdateAndReset(t *testing.T) {
	repo := newMemoryRepo()
	service := newTestServiceWithRepo(t, repo, configregistry.Definition{
		Key:          "notification.enabled",
		Module:       "notification",
		Group:        "notification.general",
		Title:        "Notification enabled",
		Type:         configregistry.ValueTypeBoolean,
		DefaultValue: json.RawMessage(`true`),
	})

	before, err := service.Get(context.Background(), "notification.enabled")
	if err != nil {
		t.Fatalf("get default before update: %v", err)
	}
	if string(before.EffectiveValue) != "true" {
		t.Fatalf("expected default true before update, got %#v", before)
	}
	if repo.listOverridesCalls() != 1 {
		t.Fatalf("expected initial snapshot load, got %d", repo.listOverridesCalls())
	}

	updated, err := service.Update(context.Background(), "notification.enabled", json.RawMessage(`false`), nil)
	if err != nil {
		t.Fatalf("update override: %v", err)
	}
	if string(updated.EffectiveValue) != "false" {
		t.Fatalf("expected refreshed override false after update, got %#v", updated)
	}
	if repo.listOverridesCalls() != 2 {
		t.Fatalf("expected snapshot reload after update invalidation, got %d", repo.listOverridesCalls())
	}

	reset, err := service.Reset(context.Background(), "notification.enabled")
	if err != nil {
		t.Fatalf("reset override: %v", err)
	}
	if string(reset.EffectiveValue) != "true" || reset.HasOverride {
		t.Fatalf("expected refreshed default after reset, got %#v", reset)
	}
	if repo.listOverridesCalls() != 3 {
		t.Fatalf("expected snapshot reload after reset invalidation, got %d", repo.listOverridesCalls())
	}

	debugState := service.SnapshotCacheDebugState()
	if debugState.InvalidateCount != 2 {
		t.Fatalf("expected two local invalidations, got %#v", debugState)
	}
	if debugState.LastInvalidationSource != string(snapshotInvalidationSourceLocal) {
		t.Fatalf("expected local invalidation source, got %#v", debugState)
	}
	if debugState.LastInvalidationAction != string(snapshotInvalidationActionReset) {
		t.Fatalf("expected reset invalidation action, got %#v", debugState)
	}
	if debugState.LastInvalidationKey != "notification.enabled" {
		t.Fatalf("expected invalidation key notification.enabled, got %#v", debugState)
	}
}

func TestServiceUpdateAndResetInvalidateSharedSnapshotCache(t *testing.T) {
	repo := newMemoryRepo()
	manager := newTestCacheManager(t)
	service := newTestServiceWithRepoAndManager(t, repo, manager, configregistry.Definition{
		Key:          "notification.enabled",
		Module:       "notification",
		Group:        "notification.general",
		Title:        "Notification enabled",
		Type:         configregistry.ValueTypeBoolean,
		DefaultValue: json.RawMessage(`true`),
	})
	peer := newTestServiceWithRepoAndManager(t, repo, manager, configregistry.Definition{
		Key:          "notification.enabled",
		Module:       "notification",
		Group:        "notification.general",
		Title:        "Notification enabled",
		Type:         configregistry.ValueTypeBoolean,
		DefaultValue: json.RawMessage(`true`),
	})
	if _, err := peer.Get(context.Background(), "notification.enabled"); err != nil {
		t.Fatalf("warm shared snapshot: %v", err)
	}
	repo.resetReadCounters()

	updated, err := service.Update(context.Background(), "notification.enabled", json.RawMessage(`false`), nil)
	if err != nil {
		t.Fatalf("update override: %v", err)
	}
	if string(updated.EffectiveValue) != "false" {
		t.Fatalf("expected updated value false, got %#v", updated)
	}

	peerItem, err := peer.Get(context.Background(), "notification.enabled")
	if err != nil {
		t.Fatalf("peer get after update invalidation: %v", err)
	}
	if string(peerItem.EffectiveValue) != "false" {
		t.Fatalf("expected peer to observe updated value through shared cache invalidation, got %#v", peerItem)
	}
	if repo.listOverridesCalls() != 1 {
		t.Fatalf("expected one reload after shared cache invalidation, got %d", repo.listOverridesCalls())
	}

	reset, err := service.Reset(context.Background(), "notification.enabled")
	if err != nil {
		t.Fatalf("reset override: %v", err)
	}
	if string(reset.EffectiveValue) != "true" || reset.HasOverride {
		t.Fatalf("expected reset to restore default, got %#v", reset)
	}
	resetPeerItem, err := peer.Get(context.Background(), "notification.enabled")
	if err != nil {
		t.Fatalf("peer get after reset invalidation: %v", err)
	}
	if string(resetPeerItem.EffectiveValue) != "true" || resetPeerItem.HasOverride {
		t.Fatalf("expected peer to observe reset default, got %#v", resetPeerItem)
	}
	if repo.listOverridesCalls() != 2 {
		t.Fatalf("expected second reload after reset invalidation, got %d", repo.listOverridesCalls())
	}
}

func TestServiceSingleflightCollapsesConcurrentSnapshotMisses(t *testing.T) {
	repo := newMemoryRepo()
	repo.listOverridesBlock = make(chan struct{})
	service := newTestServiceWithRepo(t, repo, configregistry.Definition{
		Key:          "notification.enabled",
		Module:       "notification",
		Group:        "notification.general",
		Title:        "Notification enabled",
		Type:         configregistry.ValueTypeBoolean,
		DefaultValue: json.RawMessage(`true`),
	})

	const readers = 8
	results := make(chan error, readers)
	var wg sync.WaitGroup
	wg.Add(readers)
	for range readers {
		go func() {
			defer wg.Done()
			item, err := service.Get(context.Background(), "notification.enabled")
			if err != nil {
				results <- err
				return
			}
			if string(item.EffectiveValue) != "true" {
				results <- errors.New("unexpected effective value")
				return
			}
			results <- nil
		}()
	}

	if !repo.waitForListOverridesCalls(1, time.Second) {
		t.Fatalf("expected concurrent reads to queue behind the same snapshot load, got %d starts", repo.listOverridesCalls())
	}
	close(repo.listOverridesBlock)
	wg.Wait()
	close(results)
	for err := range results {
		if err != nil {
			t.Fatalf("concurrent read failed: %v", err)
		}
	}
	if repo.listOverridesCalls() != 1 {
		t.Fatalf("expected singleflight to collapse concurrent snapshot loads to one query, got %d", repo.listOverridesCalls())
	}

	debugState := service.SnapshotCacheDebugState()
	if debugState.MissCount != readers {
		t.Fatalf("expected one miss observation per concurrent reader, got %#v", debugState)
	}
	if debugState.LoadCount != 1 {
		t.Fatalf("expected one snapshot load in debug state, got %#v", debugState)
	}
}

func TestServiceSnapshotCacheDebugStateTracksHitMissAndLoadCounts(t *testing.T) {
	repo := newMemoryRepo()
	service := newTestServiceWithRepo(t, repo, configregistry.Definition{
		Key:          "notification.enabled",
		Module:       "notification",
		Group:        "notification.general",
		Title:        "Notification enabled",
		Type:         configregistry.ValueTypeBoolean,
		DefaultValue: json.RawMessage(`true`),
	})

	initial := service.SnapshotCacheDebugState()
	if initial.Cached || initial.HitCount != 0 || initial.MissCount != 0 || initial.LoadCount != 0 {
		t.Fatalf("expected zeroed debug state before reads, got %#v", initial)
	}

	if _, err := service.Get(context.Background(), "notification.enabled"); err != nil {
		t.Fatalf("first get: %v", err)
	}
	if _, err := service.Get(context.Background(), "notification.enabled"); err != nil {
		t.Fatalf("second get: %v", err)
	}

	debugState := service.SnapshotCacheDebugState()
	if !debugState.Cached {
		t.Fatalf("expected snapshot cache to remain warm, got %#v", debugState)
	}
	if debugState.CachedOverrideCount != 0 {
		t.Fatalf("expected zero overrides in warm cache, got %#v", debugState)
	}
	if debugState.MissCount != 1 || debugState.HitCount != 1 || debugState.LoadCount != 1 {
		t.Fatalf("expected one miss, one hit, and one load, got %#v", debugState)
	}
	if debugState.LastLoadedOverrideCount != 0 || debugState.LastLoadAt == nil {
		t.Fatalf("expected last load metadata, got %#v", debugState)
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
	service := newTestServiceWithRepoAndUsers(t, repo, newTestCacheManager(t), nil, configregistry.Definition{
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
	resources, err := systemconfiglocales.EmbeddedLocaleResources()
	if err != nil {
		t.Fatalf("load system-config locale resources: %v", err)
	}
	if err := localizer.RegisterEmbeddedLocaleResources(resources); err != nil {
		t.Fatalf("register system-config locale resources: %v", err)
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

	if len(registry.Items()) != 0 {
		t.Fatalf("expected system-config dashboard registry to stay empty, got %#v", registry.Items())
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

func TestServiceRejectsScalarValueOutsideSchemaConstraints(t *testing.T) {
	testCases := []struct {
		name         string
		definition   configregistry.Definition
		invalidValue json.RawMessage
	}{
		{
			name: "integer range",
			definition: configregistry.Definition{
				Key:          "dashboard.quick_actions.max_items",
				Module:       "dashboard",
				Group:        "quick_actions",
				Title:        "Maximum quick actions",
				Type:         configregistry.ValueTypeInteger,
				Schema:       json.RawMessage(`{"type":"integer","minimum":1,"maximum":24}`),
				DefaultValue: json.RawMessage(`4`),
			},
			invalidValue: json.RawMessage(`25`),
		},
		{
			name: "string enum",
			definition: configregistry.Definition{
				Key:          "dashboard.quick_actions.strategy",
				Module:       "dashboard",
				Group:        "quick_actions",
				Title:        "Quick action strategy",
				Type:         configregistry.ValueTypeString,
				Schema:       json.RawMessage(`{"type":"string","enum":["most_used","recent","hybrid"]}`),
				DefaultValue: json.RawMessage(`"hybrid"`),
			},
			invalidValue: json.RawMessage(`"unknown"`),
		},
		{
			name: "string length",
			definition: configregistry.Definition{
				Key:          "auth.password_policy",
				Module:       "auth",
				Group:        "security",
				Title:        "Password policy",
				Type:         configregistry.ValueTypeString,
				Schema:       json.RawMessage(`{"type":"string","minLength":3,"maxLength":8}`),
				DefaultValue: json.RawMessage(`"medium"`),
			},
			invalidValue: json.RawMessage(`"xy"`),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			service := newTestService(t, tc.definition)
			if _, err := service.Update(context.Background(), tc.definition.Key, tc.invalidValue, nil); !errors.Is(err, errInvalidConfigValue) {
				t.Fatalf("expected scalar schema validation error, got %v", err)
			}
		})
	}
}

func TestServiceAcceptsScalarValueInsideSchemaConstraints(t *testing.T) {
	definition := configregistry.Definition{
		Key:          "dashboard.quick_actions.strategy",
		Module:       "dashboard",
		Group:        "quick_actions",
		Title:        "Quick action strategy",
		Type:         configregistry.ValueTypeString,
		Schema:       json.RawMessage(`{"type":"string","enum":["most_used","recent","hybrid"]}`),
		DefaultValue: json.RawMessage(`"hybrid"`),
	}
	service := newTestService(t, definition)

	item, err := service.Update(context.Background(), definition.Key, json.RawMessage(`"recent"`), nil)
	if err != nil {
		t.Fatalf("update valid scalar override: %v", err)
	}
	if string(item.EffectiveValue) != `"recent"` {
		t.Fatalf("expected valid scalar override, got %#v", item)
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
			Key:                 "httpx.access-log-retention-cleanup",
			Module:              "core.httpx",
			Domain:              "logs",
			DomainKey:           "systemConfig.domains.logs",
			DomainLabel:         "Logs",
			Group:               "log.retention",
			GroupKey:            "systemConfig.groups.coreHttpxLogRetention",
			GroupLabel:          "Access log retention",
			GroupDescription:    "Manage access log cleanup retention and batch policy.",
			GroupDescriptionKey: "systemConfig.groupDescriptions.coreHttpxLogRetention",
			Title:               "Access log retention cleanup",
			TitleKey:            "systemConfig.items.accessLogRetentionCleanup.title",
			Description:         "Default cleanup configuration for access-log retention jobs.",
			DescriptionKey:      "systemConfig.items.accessLogRetentionCleanup.description",
			Tags:                []string{"httpx", "log.retention"},
			Type:                configregistry.ValueTypeObject,
			Schema: json.RawMessage(
				`{"type":"object","properties":{"retentionDays":{"type":"integer","title":"Log retention days","x-i18n":{"titleKey":"systemConfig.fields.retentionDays.title","unitKey":"systemConfig.units.days"}}}}`,
			),
			DefaultValue:     json.RawMessage(`{"retentionDays":30}`),
			RuntimeApplyMode: configregistry.RuntimeApplyModeRuntimeHot,
		},
		DefaultValue:   json.RawMessage(`{"retentionDays":30}`),
		EffectiveValue: json.RawMessage(`{"retentionDays":30}`),
	})

	assertMappedLocalizationMetadata(t, item)
	assertMappedStructuredSchema(t, item)
	assertMappedRuntimeApplyMode(t, item)
}

func assertMappedLocalizationMetadata(t *testing.T, item generated.SystemConfigItem) {
	t.Helper()

	if item.GroupKey == nil || *item.GroupKey != "systemConfig.groups.coreHttpxLogRetention" {
		t.Fatalf("expected group key in response, got %#v", item.GroupKey)
	}
	if item.DomainKey == nil || *item.DomainKey != "systemConfig.domains.logs" {
		t.Fatalf("expected domain key in response, got %#v", item.DomainKey)
	}
	if item.GroupDescriptionKey == nil || *item.GroupDescriptionKey != "systemConfig.groupDescriptions.coreHttpxLogRetention" {
		t.Fatalf("expected group description key in response, got %#v", item.GroupDescriptionKey)
	}
	if item.TitleKey == nil || *item.TitleKey != "systemConfig.items.accessLogRetentionCleanup.title" {
		t.Fatalf("expected title key in response, got %#v", item.TitleKey)
	}
	if item.Tags == nil || len(*item.Tags) != 2 || (*item.Tags)[0] != "httpx" {
		t.Fatalf("expected tags in response, got %#v", item.Tags)
	}
}

func assertMappedStructuredSchema(t *testing.T, item generated.SystemConfigItem) {
	t.Helper()

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

func assertMappedRuntimeApplyMode(t *testing.T, item generated.SystemConfigItem) {
	t.Helper()

	if item.RuntimeApplyMode != generated.SystemConfigItemRuntimeApplyMode(configregistry.RuntimeApplyModeRuntimeHot) {
		t.Fatalf("expected runtime apply mode in response, got %#v", item.RuntimeApplyMode)
	}
}

func newTestService(t *testing.T, definition configregistry.Definition) *Service {
	t.Helper()
	return newTestServiceWithRepo(t, newMemoryRepo(), definition)
}

func newTestServiceWithRepo(t *testing.T, repo *memoryRepo, definition configregistry.Definition) *Service {
	t.Helper()
	return newTestServiceWithRepoAndUsers(t, repo, newTestCacheManager(t), testUserService{
		users: map[uint64]moduleapi.UserSummary{
			7:  {ID: 7, Username: "bob", Display: "Bob"},
			42: {ID: 42, Username: "alice", Display: "Alice"},
		},
	}, definition)
}

func newTestServiceWithRepoAndManager(
	t *testing.T,
	repo *memoryRepo,
	manager *cachex.Manager,
	definition configregistry.Definition,
) *Service {
	t.Helper()
	return newTestServiceWithRepoAndUsers(t, repo, manager, testUserService{
		users: map[uint64]moduleapi.UserSummary{
			7:  {ID: 7, Username: "bob", Display: "Bob"},
			42: {ID: 42, Username: "alice", Display: "Alice"},
		},
	}, definition)
}

func newTestServiceWithRepoAndUsers(
	t *testing.T,
	repo *memoryRepo,
	manager *cachex.Manager,
	users moduleapi.UserService,
	definition configregistry.Definition,
) *Service {
	t.Helper()
	definition = normalizeTestDefinition(definition)
	registry := configregistry.NewRegistry()
	if err := registry.Register(definition); err != nil {
		t.Fatalf("register definition: %v", err)
	}
	cache, err := manager.NewCache(systemConfigSnapshotCacheName)
	if err != nil {
		t.Fatalf("new test snapshot cache: %v", err)
	}
	service, err := NewService(registry, repo, ServiceOptions{
		Users:  users,
		Cache:  cache,
		Logger: zap.NewNop(),
	})
	if err != nil {
		t.Fatalf("create service: %v", err)
	}
	return service
}

func newTestCacheManager(t *testing.T) *cachex.Manager {
	t.Helper()
	manager, err := cachex.NewManager(cachex.ManagerOptions{
		Backend:   cachebackend.NewMemory(),
		Namespace: "test-runtime",
	})
	if err != nil {
		t.Fatalf("new test cache manager: %v", err)
	}
	return manager
}

func normalizeTestDefinition(definition configregistry.Definition) configregistry.Definition {
	if definition.Domain == "" {
		definition.Domain = definition.Module
	}
	return definition
}

type memoryRepo struct {
	mu sync.Mutex

	values map[string]json.RawMessage
	audit  map[string]systemconfigstore.Override

	listOverridesStarted atomic.Int32
	getOverrideStarted   atomic.Int32
	listOverridesBlock   chan struct{}
}

func newMemoryRepo() *memoryRepo {
	return &memoryRepo{
		values: make(map[string]json.RawMessage),
		audit:  make(map[string]systemconfigstore.Override),
	}
}

func (r *memoryRepo) ListOverrides(_ context.Context) ([]systemconfigstore.Override, error) {
	r.listOverridesStarted.Add(1)
	if r.listOverridesBlock != nil {
		<-r.listOverridesBlock
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	overrides := make([]systemconfigstore.Override, 0, len(r.audit))
	for key, override := range r.audit {
		override.Key = key
		override.Value = cloneRawMessage(r.values[key])
		overrides = append(overrides, cloneOverride(override))
	}
	return overrides, nil
}

func (r *memoryRepo) GetOverride(_ context.Context, key string) (systemconfigstore.Override, error) {
	r.getOverrideStarted.Add(1)

	r.mu.Lock()
	defer r.mu.Unlock()

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

	r.mu.Lock()
	defer r.mu.Unlock()

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
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.values, key)
	delete(r.audit, key)
	return nil
}

func (r *memoryRepo) resetReadCounters() {
	r.listOverridesStarted.Store(0)
	r.getOverrideStarted.Store(0)
}

func (r *memoryRepo) listOverridesCalls() int {
	return int(r.listOverridesStarted.Load())
}

func (r *memoryRepo) getOverrideCalls() int {
	return int(r.getOverrideStarted.Load())
}

func (r *memoryRepo) waitForListOverridesCalls(min int, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if r.listOverridesCalls() >= min {
			return true
		}
		time.Sleep(5 * time.Millisecond)
	}
	return r.listOverridesCalls() >= min
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
