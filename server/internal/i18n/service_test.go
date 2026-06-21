package i18n

import (
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"testing/fstest"

	"graft/server/internal/config"
)

func newTestService() *Service {
	return MustNew(config.I18nConfig{
		DefaultLocale:    "zh-CN",
		FallbackLocale:   "en-US",
		SupportedLocales: []string{"zh-CN", "en-US"},
	})
}

func newTestServiceWithModuleLocales(t *testing.T) *Service {
	t.Helper()

	service := newTestService()
	resources := []EmbeddedLocaleResource{
		{
			Namespace: "audit",
			Locale:    LocaleZHCN,
			Source:    "audit/zh-CN.yaml",
			Data: []byte(strings.Join([]string{
				"menu.audit.title: 安全审计",
				"audit.target.user: 用户",
				"audit.target.auth: 认证",
				"dashboard.widget.auditRiskEvents.title: 审计风险事件",
				"scheduler.job.auditLogRetentionCleanup.title: 审计日志保留清理",
			}, "\n")),
		},
		{
			Namespace: "audit",
			Locale:    LocaleENUS,
			Source:    "audit/en-US.yaml",
			Data: []byte(strings.Join([]string{
				"menu.audit.title: Security Audit",
				"audit.target.user: User",
				"audit.target.auth: Authentication",
				"dashboard.widget.auditRiskEvents.title: Audit Risk Events",
				"scheduler.job.auditLogRetentionCleanup.title: Audit Log Retention Cleanup",
			}, "\n")),
		},
		{
			Namespace: "scheduler",
			Locale:    LocaleZHCN,
			Source:    "scheduler/zh-CN.yaml",
			Data: []byte(strings.Join([]string{
				"menu.server.scheduled_tasks.title: 定时任务",
				"dashboard.widget.schedulerTaskAttention.title: 定时任务关注项",
			}, "\n")),
		},
		{
			Namespace: "scheduler",
			Locale:    LocaleENUS,
			Source:    "scheduler/en-US.yaml",
			Data: []byte(strings.Join([]string{
				"menu.server.scheduled_tasks.title: Scheduled Tasks",
				"dashboard.widget.schedulerTaskAttention.title: Scheduled Task Attention",
			}, "\n")),
		},
		{
			Namespace: "container",
			Locale:    LocaleZHCN,
			Source:    "container/zh-CN.yaml",
			Data:      []byte("menu.ops.title: 运维管理\nmenu.ops.container.title: 容器管理\n"),
		},
		{
			Namespace: "container",
			Locale:    LocaleENUS,
			Source:    "container/en-US.yaml",
			Data:      []byte("menu.ops.title: Operations\nmenu.ops.container.title: Container Management\n"),
		},
		{
			Namespace: "monitor",
			Locale:    LocaleZHCN,
			Source:    "monitor/zh-CN.yaml",
			Data:      []byte("dashboard.widget.monitorSystemHealth.title: 系统健康\n"),
		},
		{
			Namespace: "monitor",
			Locale:    LocaleENUS,
			Source:    "monitor/en-US.yaml",
			Data:      []byte("dashboard.widget.monitorSystemHealth.title: System Health\n"),
		},
	}
	if err := service.RegisterEmbeddedLocaleResources(resources); err != nil {
		t.Fatalf("register synthetic module locale resources: %v", err)
	}
	return service
}

func newTestServiceWithModuleRuntimeLocale(t *testing.T) *Service {
	t.Helper()

	service := newTestService()
	if err := service.RegisterEmbeddedLocaleResources([]EmbeddedLocaleResource{
		{
			Namespace: "module-runtime",
			Locale:    LocaleZHCN,
			Source:    "module-runtime/zh-CN.yaml",
			Data:      []byte("menu.modulesRuntime.title: 模块运行时\n"),
		},
		{
			Namespace: "module-runtime",
			Locale:    LocaleENUS,
			Source:    "module-runtime/en-US.yaml",
			Data:      []byte("menu.modulesRuntime.title: Module Runtime\n"),
		},
	}); err != nil {
		t.Fatalf("register synthetic module-runtime locale resources: %v", err)
	}
	return service
}

// TestResolveLocaleUsesConfiguredFallbackOrder 验证 locale 解析会按请求、
// 会话、默认语言、回退语言的固定顺序收敛。
func TestResolveLocaleUsesConfiguredFallbackOrder(t *testing.T) {
	service := newTestService()

	if locale := service.ResolveLocale("en-US", "zh-CN"); locale != "en-US" {
		t.Fatalf("expected request locale to win, got %q", locale)
	}
	if locale := service.ResolveLocale("", "en-US"); locale != "en-US" {
		t.Fatalf("expected session locale to win, got %q", locale)
	}
	if locale := service.ResolveLocale("", ""); locale != "zh-CN" {
		t.Fatalf("expected default locale fallback, got %q", locale)
	}
	if locale := service.ResolveLocale("@@@", ""); locale != "zh-CN" {
		t.Fatalf("expected invalid locale to fall back to default, got %q", locale)
	}
}

// TestResolveRequestLocalePrefersExplicitHeader 验证平台自定义请求头会优先于
// Accept-Language 参与 locale 解析。
func TestResolveRequestLocalePrefersExplicitHeader(t *testing.T) {
	service := newTestService()
	request := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	request.Header.Set(LocaleHeader, "en-US")
	request.Header.Set("Accept-Language", "zh-CN,zh;q=0.9")

	locale := service.ResolveRequestLocale(request, "")
	if locale != "en-US" {
		t.Fatalf("expected explicit locale header to win, got %q", locale)
	}
}

// TestResolveRequestLocaleFallsBackToAcceptLanguage 验证缺少显式 locale 头时，
// 服务会回退解析 Accept-Language。
func TestResolveRequestLocaleFallsBackToAcceptLanguage(t *testing.T) {
	service := newTestService()
	request := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	request.Header.Set("Accept-Language", "en-US,en;q=0.8")

	locale := service.ResolveRequestLocale(request, "")
	if locale != "en-US" {
		t.Fatalf("expected Accept-Language locale, got %q", locale)
	}
}

// TestMessageFallsBackToConfiguredLocalesAndKey 验证文案查找会优先命中解析后
// 的语言目录，缺失时再返回稳定 message key。
func TestMessageFallsBackToConfiguredLocalesAndKey(t *testing.T) {
	service := newTestService()

	if message := service.Message("en-US", "auth.missing_actor"); message != "Missing request actor" {
		t.Fatalf("expected en-US catalog message, got %q", message)
	}
	if message := service.Message("zh-CN", "auth.invalid_credentials"); message != "用户名或密码错误" {
		t.Fatalf("expected zh-CN auth login message, got %q", message)
	}
	if message := service.Message("en-US", "auth.invalid_refresh_session"); message != "Invalid or expired refresh session" {
		t.Fatalf("expected en-US auth refresh message, got %q", message)
	}
	if message := service.Message("en-US", "auth.session_not_found"); message != "Session not found or already inactive" {
		t.Fatalf("expected en-US auth session-not-found message, got %q", message)
	}
	if message := service.Message("en-US", "common.conjunction"); message != "and" {
		t.Fatalf("expected en-US shared conjunction message, got %q", message)
	}
	if message := service.Message("zh-CN", "common.copyright"); message != "Copyright (C) 2021-2026 Tencent. All Rights Reserved" {
		t.Fatalf("expected zh-CN shared copyright message, got %q", message)
	}
	if message := service.Message("en-US", "menu.server.title"); message != "Service Management" {
		t.Fatalf("expected en-US shared server menu title, got %q", message)
	}
	if message := service.Message("en-US", "missing.key"); message != "missing.key" {
		t.Fatalf("expected missing key fallback, got %q", message)
	}
}

func TestRegisterMessagesAddsNamespaceScopedResources(t *testing.T) {
	service := newTestService()

	if err := service.RegisterMessages(Registration{
		Namespace: "user",
		Locale:    LocaleENUS,
		Messages: []MessageResource{
			{Key: "menu.list", Text: "User List"},
		},
	}); err != nil {
		t.Fatalf("register messages: %v", err)
	}

	message := service.Lookup(LookupRequest{
		Namespace: "user",
		Locale:    LocaleENUS,
		Key:       "menu.list",
	})
	if message != "User List" {
		t.Fatalf("expected namespaced message, got %q", message)
	}
}

func TestRegisterMessagesRejectsDuplicateKeyWithinNamespaceAndLocale(t *testing.T) {
	service := newTestService()

	registration := Registration{
		Namespace: "rbac",
		Locale:    LocaleZHCN,
		Messages: []MessageResource{
			{Key: "menu.list", Text: "角色管理"},
		},
	}
	if err := service.RegisterMessages(registration); err != nil {
		t.Fatalf("register messages: %v", err)
	}

	if err := service.RegisterMessages(registration); err == nil {
		t.Fatal("expected duplicate registration to fail")
	}
}

func TestRegisterMessagesRejectsUnsupportedLocale(t *testing.T) {
	service := newTestService()

	err := service.RegisterMessages(Registration{
		Namespace: "user",
		Locale:    "fr-FR",
		Messages: []MessageResource{
			{Key: "menu.list", Text: "Utilisateurs"},
		},
	})
	if err == nil {
		t.Fatal("expected unsupported locale error")
	}
}

func TestFreezeBlocksLateMessageRegistration(t *testing.T) {
	service := newTestService()

	if err := service.Freeze(); err != nil {
		t.Fatalf("freeze service: %v", err)
	}
	if !service.IsFrozen() {
		t.Fatal("expected service to become frozen")
	}

	err := service.RegisterMessages(Registration{
		Namespace: "user",
		Locale:    LocaleZHCN,
		Messages: []MessageResource{
			{Key: "menu.list", Text: "用户管理"},
		},
	})
	if err == nil {
		t.Fatal("expected frozen registry to reject registration")
	}
}

func TestLookupFallsBackToExplicitFallbackMessage(t *testing.T) {
	service := newTestService()

	message := service.Lookup(LookupRequest{
		Namespace:       "user",
		Locale:          LocaleENUS,
		Key:             "menu.missing",
		FallbackMessage: "Fallback Copy",
	})
	if message != "Fallback Copy" {
		t.Fatalf("expected explicit fallback message, got %q", message)
	}
}

func TestLookupUsesModuleNamespaceAndFallbackMessage(t *testing.T) {
	service := newTestService()

	if err := service.RegisterMessages(Registration{
		Namespace: "user",
		Locale:    LocaleENUS,
		Messages: []MessageResource{
			{Key: "menu.user_list.title", Text: "User Management"},
		},
	}); err != nil {
		t.Fatalf("register messages: %v", err)
	}

	message := service.Lookup(LookupRequest{
		Namespace:       "user",
		Locale:          LocaleENUS,
		Key:             "menu.user_list.title",
		FallbackMessage: "用户管理",
	})
	if message != "User Management" {
		t.Fatalf("expected module namespace message, got %q", message)
	}

	message = service.Lookup(LookupRequest{
		Namespace:       "user",
		Locale:          LocaleZHCN,
		Key:             "menu.user_list.title",
		FallbackMessage: "用户管理",
	})
	if message != "User Management" {
		t.Fatalf("expected fallback locale catalog message, got %q", message)
	}

	message = service.Lookup(LookupRequest{
		Namespace:       "user",
		Locale:          LocaleZHCN,
		Key:             "menu.profile.title",
		FallbackMessage: "个人中心",
	})
	if message != "个人中心" {
		t.Fatalf("expected explicit fallback title message, got %q", message)
	}
}

func TestRegisteredMessageKeyIDsFindsBareKeyAcrossNamespaces(t *testing.T) {
	service := newTestServiceWithModuleRuntimeLocale(t)

	matches := service.RegisteredMessageKeyIDs(LocaleENUS, "menu.modulesRuntime.title")
	if len(matches) != 1 || matches[0] != "module-runtime.menu.modulesRuntime.title" {
		t.Fatalf("expected module runtime canonical key, got %v", matches)
	}

	if matches := service.RegisteredMessageKeyIDs(LocaleZHCN, "menu.modulesRuntime.title"); len(matches) != 1 ||
		matches[0] != "module-runtime.menu.modulesRuntime.title" {
		t.Fatalf("expected zh-CN canonical key match, got %v", matches)
	}
}

func TestRegisteredMessageResourcesFindsRegisteredTextAcrossNamespaces(t *testing.T) {
	service := newTestServiceWithModuleRuntimeLocale(t)

	matches := service.RegisteredMessageResources(LocaleENUS, "menu.modulesRuntime.title")
	if len(matches) != 1 ||
		matches[0].Key != "module-runtime.menu.modulesRuntime.title" ||
		matches[0].Text != "Module Runtime" {
		t.Fatalf("expected module runtime message resource, got %v", matches)
	}
}

func TestEmbeddedLocaleResourcesIncludePhase4DisplayKeys(t *testing.T) {
	service := newTestServiceWithModuleLocales(t)

	keys := []string{
		"menu.notification.title",
		"menu.audit.title",
		"menu.server.scheduled_tasks.title",
		"menu.ops.title",
		"menu.ops.container.title",
		"menu.logCenter.title",
		"menu.accessLog.title",
		"menu.appLog.title",
		"dashboard.widget.accessLogRequestAttention.title",
		"dashboard.widget.schedulerTaskAttention.title",
		"dashboard.widget.auditRiskEvents.title",
		"scheduler.job.auditLogRetentionCleanup.title",
		"scheduler.job.accessLogRetentionCleanup.title",
		"scheduler.job.appLogRetentionCleanup.title",
	}

	for _, locale := range []LocaleTag{LocaleZHCN, LocaleENUS} {
		for _, key := range keys {
			matches := service.RegisteredMessageResources(locale, MessageKey(key))
			if len(matches) != 1 {
				t.Fatalf("expected one embedded message for %s %q, got %#v", locale, key, matches)
			}
			if matches[0].Text == "" {
				t.Fatalf("expected non-empty embedded message for %s %q", locale, key)
			}
		}
	}
}

func TestEmbeddedLocaleResourcesIncludeAuditTargetLabelKeys(t *testing.T) {
	service := newTestServiceWithModuleLocales(t)

	for _, tc := range []struct {
		locale   LocaleTag
		key      string
		expected string
	}{
		{locale: LocaleZHCN, key: "audit.target.user", expected: "用户"},
		{locale: LocaleENUS, key: "audit.target.user", expected: "User"},
		{locale: LocaleZHCN, key: "audit.target.auth", expected: "认证"},
		{locale: LocaleENUS, key: "audit.target.auth", expected: "Authentication"},
	} {
		message := service.Lookup(LookupRequest{
			Namespace: "audit",
			Locale:    tc.locale,
			Key:       MessageKey(tc.key),
		})
		if message != tc.expected {
			t.Fatalf("expected embedded audit target label %q for %s %q, got %q", tc.expected, tc.locale, tc.key, message)
		}

		matches := service.RegisteredMessageResources(tc.locale, MessageKey(tc.key))
		if len(matches) != 1 {
			t.Fatalf("expected one embedded audit target label resource for %s %q, got %#v", tc.locale, tc.key, matches)
		}
	}
}

func TestEmbeddedCoreLocaleResourcesProvideDefaultCatalogMessages(t *testing.T) {
	service := newTestService()

	expectedKeys := []string{
		"core.auth.invalid_credentials",
		"core.auth.token_missing",
		"core.auth.token_expired",
		"core.auth.token_invalid",
		"core.auth.forbidden",
		"core.auth.invalid_refresh_session",
		"core.auth.password_policy_violation",
		"core.auth.password_reuse_forbidden",
		"core.auth.current_password_invalid",
		"core.auth.missing_actor",
		"core.auth.missing_permission",
		"core.auth.session_not_found",
		"core.common.conjunction",
		"core.common.copyright",
		"core.common.internal_error",
		"core.common.invalid_argument",
		"core.menu.server.title",
		"core.rbac.cannot_remove_own_admin_role",
		"core.rbac.builtin_admin_permissions_immutable",
		"core.permission.not_found",
		"core.role.not_found",
		"core.user.not_found",
	}

	for _, locale := range []LocaleTag{LocaleZHCN, LocaleENUS} {
		coreMatches := service.RegisteredMessageKeys(CoreNamespace, locale)
		if len(coreMatches) < len(expectedKeys) {
			t.Fatalf("expected at least %d core keys for %s, got %d", len(expectedKeys), locale, len(coreMatches))
		}
		for _, key := range expectedKeys {
			if !slices.Contains(coreMatches, key) {
				t.Fatalf("expected core namespace to include %s for %s, got %v", key, locale, coreMatches)
			}
		}
	}
}

func TestParseLocaleResourceName(t *testing.T) {
	namespace, locale, err := parseLocaleResourceName("system-config.zh-CN.yaml")
	if err != nil {
		t.Fatalf("parse locale resource name: %v", err)
	}
	if namespace != "system-config" {
		t.Fatalf("expected system-config namespace, got %q", namespace)
	}
	if locale != LocaleZHCN {
		t.Fatalf("expected zh-CN locale, got %q", locale)
	}
}

func TestParseLocaleResourceNameRejectsInvalidFormat(t *testing.T) {
	if _, _, err := parseLocaleResourceName("system-config.yaml"); err == nil {
		t.Fatal("expected invalid locale resource name to fail")
	}
}

func TestLoadLocaleRegistrationsParsesFlatYAML(t *testing.T) {
	registrations, err := loadLocaleRegistrations(fstest.MapFS{
		"locales/system-config.en-US.yaml": {
			Data: []byte("systemConfig.domains.dashboard: Dashboard\nsystemConfig.groups.quickActions: Quick Actions\n"),
		},
		"locales/system-config.zh-CN.yaml": {
			Data: []byte("systemConfig.domains.dashboard: 工作台配置\n"),
		},
	})
	if err != nil {
		t.Fatalf("load locale registrations: %v", err)
	}
	if len(registrations) != 2 {
		t.Fatalf("expected 2 registrations, got %d", len(registrations))
	}
	if registrations[0].Namespace != "system-config" || registrations[0].Locale != LocaleENUS {
		t.Fatalf("expected sorted en-US registration first, got %+v", registrations[0])
	}
	if len(registrations[0].Messages) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(registrations[0].Messages))
	}
	if registrations[0].Messages[0].Key != "systemConfig.domains.dashboard" ||
		registrations[0].Messages[0].Text != "Dashboard" {
		t.Fatalf("unexpected first message: %+v", registrations[0].Messages[0])
	}
}

func TestLoadLocaleRegistrationsIncludesNestedModuleResources(t *testing.T) {
	registrations, err := loadLocaleRegistrations(fstest.MapFS{
		"locales/core.en-US.yaml": {
			Data: []byte("core.errors.unknown: Unknown error\n"),
		},
		"locales/modules/rbac.en-US.yaml": {
			Data: []byte("rbac.permissionCatalog.users.display: Users\n"),
		},
		"locales/modules/rbac.zh-CN.yaml": {
			Data: []byte("rbac.permissionCatalog.users.display: 用户\n"),
		},
	})
	if err != nil {
		t.Fatalf("load locale registrations: %v", err)
	}
	if len(registrations) != 3 {
		t.Fatalf("expected 3 registrations, got %d", len(registrations))
	}
	if registrations[0].Namespace != "core" || registrations[0].Locale != LocaleENUS {
		t.Fatalf("expected core en-US registration first, got %+v", registrations[0])
	}
	if registrations[1].Namespace != "rbac" || registrations[1].Locale != LocaleENUS {
		t.Fatalf("expected nested module en-US registration second, got %+v", registrations[1])
	}
	if registrations[2].Namespace != "rbac" || registrations[2].Locale != LocaleZHCN {
		t.Fatalf("expected nested module zh-CN registration third, got %+v", registrations[2])
	}
}

func TestEmbeddedLocaleResourcesReturnsDeferredModuleRegistrations(t *testing.T) {
	previousFS := embeddedLocaleFS
	embeddedLocaleFS = fstest.MapFS{
		"locales/core.en-US.yaml": {
			Data: []byte("core.common.ok: OK\n"),
		},
		"locales/modules/rbac.en-US.yaml": {
			Data: []byte("rbac.permissionCatalog.users.display: Users\n"),
		},
		"locales/modules/rbac.zh-CN.yaml": {
			Data: []byte("rbac.permissionCatalog.users.display: 用户\n"),
		},
	}
	t.Cleanup(func() {
		embeddedLocaleFS = previousFS
	})

	resources, err := EmbeddedLocaleResources()
	if err != nil {
		t.Fatalf("embedded locale resources: %v", err)
	}
	if len(resources) != 2 {
		t.Fatalf("expected 2 deferred resources, got %d", len(resources))
	}
	if resources[0].Namespace != "rbac" || resources[0].Locale != LocaleENUS {
		t.Fatalf("unexpected first deferred resource: %+v", resources[0])
	}
	if resources[1].Namespace != "rbac" || resources[1].Locale != LocaleZHCN {
		t.Fatalf("unexpected second deferred resource: %+v", resources[1])
	}
}

func TestLoadLocaleRegistrationsRejectsDuplicateKeys(t *testing.T) {
	_, err := loadLocaleRegistrations(fstest.MapFS{
		"locales/system-config.en-US.yaml": {
			Data: []byte("systemConfig.domains.dashboard: Dashboard\nsystemConfig.domains.dashboard: Dashboard Duplicate\n"),
		},
	})
	if err == nil {
		t.Fatal("expected duplicate key validation to fail")
	}
}

func TestLoadLocaleRegistrationsRejectsNestedValues(t *testing.T) {
	_, err := loadLocaleRegistrations(fstest.MapFS{
		"locales/system-config.en-US.yaml": {
			Data: []byte("systemConfig:\n  domains.dashboard: Dashboard\n"),
		},
	})
	if err == nil {
		t.Fatal("expected nested mapping validation to fail")
	}
}

func TestRegisterLocaleResourcesReusesRegisterMessagesValidation(t *testing.T) {
	service := newTestService()
	if err := service.Freeze(); err != nil {
		t.Fatalf("freeze service: %v", err)
	}

	err := service.registerLocaleResources(fstest.MapFS{
		"locales/system-config.en-US.yaml": {
			Data: []byte("systemConfig.domains.dashboard: Dashboard\n"),
		},
	})
	if err == nil {
		t.Fatal("expected frozen registry validation to fail")
	}
}

func TestRegisterEmbeddedLocaleResourcesReusesRegisterMessagesValidation(t *testing.T) {
	service := newTestService()
	if err := service.Freeze(); err != nil {
		t.Fatalf("freeze service: %v", err)
	}

	err := service.RegisterEmbeddedLocaleResources([]EmbeddedLocaleResource{
		{
			Namespace: "rbac",
			Locale:    LocaleENUS,
			Source:    "locales/modules/rbac.en-US.yaml",
			Data:      []byte("rbac.permissionCatalog.users.display: Users\n"),
		},
	})
	if err == nil {
		t.Fatal("expected frozen registry validation to fail")
	}
}

func TestRegisterEmbeddedLocaleResourcesAddsMessagesToCatalog(t *testing.T) {
	previousFS := embeddedLocaleFS
	embeddedLocaleFS = fstest.MapFS{
		"locales/core.en-US.yaml": {
			Data: []byte("core.common.ok: OK\n"),
		},
		"locales/core.zh-CN.yaml": {
			Data: []byte("core.common.ok: 好\n"),
		},
	}
	t.Cleanup(func() {
		embeddedLocaleFS = previousFS
	})

	service := newTestService()
	err := service.RegisterEmbeddedLocaleResources([]EmbeddedLocaleResource{
		{
			Namespace: "rbac",
			Locale:    LocaleENUS,
			Source:    "modules/rbac/en-US.yaml",
			Data:      []byte("rbac.permissionCatalog.users.display: Users\n"),
		},
	})
	if err != nil {
		t.Fatalf("register embedded locale resources: %v", err)
	}

	matches := service.RegisteredMessageResources(LocaleENUS, "rbac.permissionCatalog.users.display")
	if len(matches) != 1 || matches[0].Text != "Users" {
		t.Fatalf("expected registered embedded locale message, got %#v", matches)
	}
}

func TestNewRegistersEmbeddedLocaleResources(t *testing.T) {
	previousFS := embeddedLocaleFS
	embeddedLocaleFS = fstest.MapFS{
		"locales/dashboard.en-US.yaml": {
			Data: []byte("dashboard.quickActions.title: Quick Actions\n"),
		},
		"locales/dashboard.zh-CN.yaml": {
			Data: []byte("dashboard.quickActions.title: 快捷入口\n"),
		},
		"locales/modules/rbac.en-US.yaml": {
			Data: []byte("rbac.permissionCatalog.users.display: Users\n"),
		},
	}
	t.Cleanup(func() {
		embeddedLocaleFS = previousFS
	})

	service := newTestService()
	message := service.Lookup(LookupRequest{
		Namespace: "dashboard",
		Locale:    LocaleENUS,
		Key:       "dashboard.quickActions.title",
	})
	if message != "Quick Actions" {
		t.Fatalf("expected embedded locale resource to be registered, got %q", message)
	}
	if nested := service.Lookup(LookupRequest{
		Namespace: "rbac",
		Locale:    LocaleENUS,
		Key:       "rbac.permissionCatalog.users.display",
	}); nested != "Users" {
		t.Fatalf("expected nested embedded locale resource to be registered, got %q", nested)
	}
}
