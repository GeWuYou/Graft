package container

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"sync/atomic"
	"testing"

	"graft/server/internal/config"
	"graft/server/internal/configregistry"
	containerdi "graft/server/internal/container"
	"graft/server/internal/i18n"
	"graft/server/internal/menu"
	"graft/server/internal/module"
	"graft/server/internal/moduleapi"
	"graft/server/internal/permission"
	"graft/server/internal/realtime"
	"graft/server/internal/realtimeauth"
	containercontract "graft/server/modules/container/contract"
	containerlocales "graft/server/modules/container/locales"
	"graft/server/modules/container/terminal"
	systemconfiglocales "graft/server/modules/system-config/locales"
)

func TestModuleRegistersContainerFoundation(t *testing.T) {
	ctx := newTestContext()

	if err := NewModule().Register(ctx); err != nil {
		t.Fatalf("register module: %v", err)
	}

	assertPermissions(t, ctx.PermissionRegistry)
	assertMenu(t, ctx.MenuRegistry)
	assertModuleMessages(t, ctx.I18n)
	assertConfigDefinitions(t, ctx.ConfigRegistry, ctx.I18n)
}

func TestDescriptorDeclaresContainerModule(t *testing.T) {
	spec := NewModuleSpec()
	if spec.Name() != moduleID {
		t.Fatalf("expected module id %q, got %q", moduleID, spec.Name())
	}
	for _, dependency := range []string{"user", "auth", "rbac", "system-config"} {
		if !slices.Contains(spec.DependsOn(), dependency) {
			t.Fatalf("expected dependency %s in %#v", dependency, spec.DependsOn())
		}
	}
	if len(spec.MigrationDirs()) != 0 {
		t.Fatalf("container foundation must not declare migrations, got %#v", spec.MigrationDirs())
	}
	built, err := spec.Build(module.BuildContext{})
	if err != nil {
		t.Fatalf("build module: %v", err)
	}
	if built.Name() != moduleID {
		t.Fatalf("expected built module name %q, got %q", moduleID, built.Name())
	}
	if !slices.Contains(built.DependsOn(), "system-config") {
		t.Fatalf("expected built module dependencies to include system-config, got %#v", built.DependsOn())
	}
}

func TestModuleShutdownUsesServiceClosePath(t *testing.T) {
	t.Parallel()

	runtime := &moduleCloseRuntime{}
	service := &service{
		runtime:        runtime,
		statsCollector: &statsCollector{},
	}
	containerModule := &Module{service: service}

	if err := containerModule.Shutdown(&module.Context{LifecycleContext: context.Background()}); err != nil {
		t.Fatalf("shutdown module: %v", err)
	}

	if runtime.closeCalls.Load() != 1 {
		t.Fatalf("expected shutdown to close runtime exactly once, got %d", runtime.closeCalls.Load())
	}
}

func TestRouteAndConfigContractsStayCanonical(t *testing.T) {
	if containercontract.ContainerAPIGroup != "/ops/containers" {
		t.Fatalf("unexpected API group %q", containercontract.ContainerAPIGroup)
	}
	if containercontract.ContainerMenuPath != "/ops/containers" {
		t.Fatalf("unexpected menu path %q", containercontract.ContainerMenuPath)
	}
	if containercontract.ContainerDockerEndpointConfig.String() != "ops.container.docker.endpoint" {
		t.Fatalf("unexpected docker endpoint config key")
	}
	if containercontract.ContainerRuntimeEnabledConfig.String() != "ops.container.runtime.enabled" {
		t.Fatalf("unexpected runtime access config key")
	}
	if containercontract.ContainerResourceStatsCacheTTLConfig.String() != "ops.container.resource_stats.cache_ttl_seconds" {
		t.Fatalf("unexpected resource stats cache ttl config key")
	}
	if containercontract.ContainerResourceStatsCacheStaleWindowConfig.String() != "ops.container.resource_stats.stale_window_seconds" {
		t.Fatalf("unexpected resource stats stale window config key")
	}
	if containercontract.ContainerDangerousActionsEnabledConfig.String() != "ops.container.actions.dangerous_enabled" {
		t.Fatalf("unexpected dangerous actions config key")
	}
	if containercontract.ContainerShellEnabledConfig.String() != "ops.container.shell.enabled" {
		t.Fatalf("unexpected shell enabled config key")
	}
	if containercontract.ContainerEnvironmentPolicyConfig.String() != "ops.container.environment.policy" {
		t.Fatalf("unexpected environment policy config key")
	}
	if containercontract.ContainerEnvironmentMaskedCopyEnabledConfig.String() != "ops.container.environment.masked_copy_enabled" {
		t.Fatalf("unexpected environment masked copy config key")
	}
	for _, permissionCode := range expectedPermissionCodes() {
		if strings.Contains(permissionCode, "ops.docker") {
			t.Fatalf("permission %s must not use ops.docker", permissionCode)
		}
	}
}

func newTestContext() *module.Context {
	localizer := i18n.MustNew(config.I18nConfig{DefaultLocale: "zh-CN", FallbackLocale: "zh-CN", SupportedLocales: []string{"zh-CN", "en-US"}})
	resources, err := containerlocales.EmbeddedLocaleResources()
	if err != nil {
		panic(fmt.Sprintf("load container locale resources: %v", err))
	}
	if err := localizer.RegisterEmbeddedLocaleResources(resources); err != nil {
		panic(fmt.Sprintf("register container locale resources: %v", err))
	}
	systemConfigResources, err := systemconfiglocales.EmbeddedLocaleResources()
	if err != nil {
		panic(fmt.Sprintf("load system-config locale resources: %v", err))
	}
	if err := localizer.RegisterEmbeddedLocaleResources(systemConfigResources); err != nil {
		panic(fmt.Sprintf("register system-config locale resources: %v", err))
	}
	services := containerdi.New()
	if err := services.RegisterSingleton((*realtimeauth.Service)(nil), func(containerdi.Resolver) (any, error) {
		return realtimeauth.NewMemoryService(), nil
	}); err != nil {
		panic(fmt.Sprintf("register realtime ticket service: %v", err))
	}
	if err := services.RegisterSingleton((*realtime.Hub)(nil), func(containerdi.Resolver) (any, error) {
		return realtime.NewHub(), nil
	}); err != nil {
		panic(fmt.Sprintf("register realtime hub: %v", err))
	}
	if err := services.RegisterSingleton((*realtime.TopicIssuerRegistry)(nil), func(containerdi.Resolver) (any, error) {
		return realtime.NewTopicIssuerRegistry(), nil
	}); err != nil {
		panic(fmt.Sprintf("register realtime topic issuer registry: %v", err))
	}
	if err := services.RegisterSingleton((*moduleapi.Authorizer)(nil), func(containerdi.Resolver) (any, error) {
		return moduleAuthorizerStub{}, nil
	}); err != nil {
		panic(fmt.Sprintf("register authorizer: %v", err))
	}
	return &module.Context{
		I18n:               localizer,
		MenuRegistry:       menu.NewRegistry(),
		PermissionRegistry: permission.NewRegistry(),
		ConfigRegistry:     configregistry.NewRegistry(),
		Services:           services,
	}
}

type moduleAuthorizerStub struct{}

func (moduleAuthorizerStub) Authorize(context.Context, moduleapi.RequestAuthContext, string) error {
	return nil
}

type moduleCloseRuntime struct {
	closeCalls atomic.Int64
}

func (*moduleCloseRuntime) Info(context.Context) (RuntimeInfo, error)          { return RuntimeInfo{}, nil }
func (*moduleCloseRuntime) List(context.Context, ListQuery) ([]Summary, error) { return nil, nil }
func (*moduleCloseRuntime) Detail(context.Context, Ref) (Detail, error)        { return Detail{}, nil }
func (*moduleCloseRuntime) Mounts(context.Context, Ref) ([]Mount, error)       { return nil, nil }
func (*moduleCloseRuntime) MountUsage(context.Context, Ref, string) (MountUsage, error) {
	return MountUsage{}, nil
}
func (*moduleCloseRuntime) Logs(context.Context, Ref, LogQuery) (Logs, error) { return Logs{}, nil }
func (*moduleCloseRuntime) Shell(context.Context, Ref, string) (terminal.Session, error) {
	return nil, nil
}
func (*moduleCloseRuntime) Start(context.Context, Ref) (ActionResult, error) {
	return ActionResult{}, nil
}
func (*moduleCloseRuntime) Stop(context.Context, Ref) (ActionResult, error) {
	return ActionResult{}, nil
}
func (*moduleCloseRuntime) Restart(context.Context, Ref) (ActionResult, error) {
	return ActionResult{}, nil
}
func (*moduleCloseRuntime) Remove(context.Context, Ref, RemoveOptions) (ActionResult, error) {
	return ActionResult{}, nil
}
func (r *moduleCloseRuntime) Close() error {
	r.closeCalls.Add(1)
	return nil
}

func assertPermissions(t *testing.T, registry *permission.Registry) {
	t.Helper()

	items := registry.Items()
	if len(items) != len(expectedPermissionCodes()) {
		t.Fatalf("expected %d permissions, got %#v", len(expectedPermissionCodes()), items)
	}
	for _, code := range expectedPermissionCodes() {
		if !slices.ContainsFunc(items, func(item permission.Item) bool {
			return item.Code == code && item.Module == moduleID && item.Category == "api" &&
				strings.TrimSpace(item.DisplayKey) != "" && strings.TrimSpace(item.DescriptionKey) != ""
		}) {
			t.Fatalf("expected registered permission %s in %#v", code, items)
		}
	}
}

func expectedPermissionCodes() []string {
	return []string{
		containercontract.ContainerViewPermission.String(),
		containercontract.ContainerDetailPermission.String(),
		containercontract.ContainerEnvironmentPermission.String(),
		containercontract.ContainerLogsPermission.String(),
		containercontract.ContainerShellPermission.String(),
		containercontract.ContainerStartPermission.String(),
		containercontract.ContainerStopPermission.String(),
		containercontract.ContainerRestartPermission.String(),
		containercontract.ContainerRemovePermission.String(),
	}
}

func assertMenu(t *testing.T, registry *menu.Registry) {
	t.Helper()

	items := registry.Items()
	if len(items) != 2 {
		t.Fatalf("expected root and container menu items, got %#v", items)
	}
	assertMenuItem(t, items, expectedMenuItem{
		code:                     "ops.root",
		title:                    "",
		titleKey:                 containercontract.OperationsMenuTitle.String(),
		path:                     "/ops",
		permission:               "",
		visibleWhenConfigEnabled: "",
	})
	assertMenuItem(t, items, expectedMenuItem{
		code:                     "container.list",
		title:                    "",
		titleKey:                 containercontract.ContainerMenuTitle.String(),
		path:                     "/ops/containers",
		permission:               containercontract.ContainerViewPermission.String(),
		visibleWhenConfigEnabled: containercontract.ContainerRuntimeEnabledConfig.String(),
	})
	for _, item := range items {
		if strings.Contains(item.Path, "/server") || strings.Contains(item.Title, "服务器") {
			t.Fatalf("container menu must stay under operations IA, got %#v", item)
		}
	}
}

type expectedMenuItem struct {
	code                     string
	title                    string
	titleKey                 string
	path                     string
	permission               string
	visibleWhenConfigEnabled string
}

func assertMenuItem(t *testing.T, items []menu.Item, expected expectedMenuItem) {
	t.Helper()

	if !slices.ContainsFunc(items, func(item menu.Item) bool {
		return item.Code == expected.code && item.Title == expected.title && item.TitleKey == expected.titleKey &&
			item.Path == expected.path && item.Permission == expected.permission &&
			item.VisibleWhenConfigEnabled == expected.visibleWhenConfigEnabled && item.Module == moduleID
	}) {
		t.Fatalf("expected menu item %s in %#v", expected.code, items)
	}
}

func assertModuleMessages(t *testing.T, localizer *i18n.Service) {
	t.Helper()

	for _, key := range []string{
		containercontract.OperationsMenuTitle.String(),
		containercontract.ContainerMenuTitle.String(),
		containercontract.ContainerInvalidRef.String(),
		containercontract.ContainerShellDisabled.String(),
		containercontract.ContainerShellInvalidSize.String(),
		containercontract.ContainerDangerousActionsDisabled.String(),
		containercontract.ContainerAuditShellSessionStarted.String(),
		containercontract.ContainerActionRemoveCompleted.String(),
		containercontract.ContainerBatchActionPartial.String(),
	} {
		assertRegisteredMessage(t, localizer, i18n.LocaleZHCN, key)
		assertRegisteredMessage(t, localizer, i18n.LocaleENUS, key)
	}
}

func assertConfigDefinitions(t *testing.T, registry *configregistry.Registry, localizer *i18n.Service) {
	t.Helper()

	definitions := registry.Items()
	if len(definitions) != len(configDefinitions()) {
		t.Fatalf("expected %d config definitions, got %#v", len(configDefinitions()), definitions)
	}
	for _, key := range expectedConfigKeys() {
		definition, ok := registry.Get(key)
		if !ok {
			t.Fatalf("expected config definition %s", key)
		}
		assertConfigDefinitionMetadata(t, definition, localizer)
	}

	endpoint, _ := registry.Get(containercontract.ContainerDockerEndpointConfig.String())
	if !endpoint.RestartRequired {
		t.Fatalf("container runtime endpoint must be restart-required")
	}
	if endpoint.RuntimeApplyMode != configregistry.RuntimeApplyModeRestartRequired {
		t.Fatalf("container runtime endpoint must expose restart-required apply mode, got %#v", endpoint.RuntimeApplyMode)
	}
	runtime, _ := registry.Get(containercontract.ContainerRuntimeConfig.String())
	if !runtime.RestartRequired {
		t.Fatalf("container runtime type must be restart-required")
	}
	if runtime.RuntimeApplyMode != configregistry.RuntimeApplyModeRestartRequired {
		t.Fatalf("container runtime type must expose restart-required apply mode, got %#v", runtime.RuntimeApplyMode)
	}
	assertContainerRuntimeHotConfigModes(t, registry)

	assertRuntimeConfigSchema(t, registry)
	assertMaxTailConfigSchema(t, registry)
	assertEnvironmentPolicyConfigSchema(t, registry, localizer)
}

func assertContainerRuntimeHotConfigModes(t *testing.T, registry *configregistry.Registry) {
	t.Helper()

	for _, key := range []string{
		containercontract.ContainerRuntimeEnabledConfig.String(),
		containercontract.ContainerLogsDefaultTailConfig.String(),
		containercontract.ContainerLogsMaxTailConfig.String(),
		containercontract.ContainerResourceStatsCacheTTLConfig.String(),
		containercontract.ContainerResourceStatsCacheStaleWindowConfig.String(),
		containercontract.ContainerDangerousActionsEnabledConfig.String(),
		containercontract.ContainerShellEnabledConfig.String(),
		containercontract.ContainerEnvironmentPolicyConfig.String(),
		containercontract.ContainerEnvironmentMaskedCopyEnabledConfig.String(),
	} {
		definition, ok := registry.Get(key)
		if !ok {
			t.Fatalf("expected container config definition %s", key)
		}
		if definition.RuntimeApplyMode != configregistry.RuntimeApplyModeRuntimeHot {
			t.Fatalf("expected runtime-hot apply mode for %s, got %#v", key, definition.RuntimeApplyMode)
		}
	}
}

func assertRuntimeConfigSchema(t *testing.T, registry *configregistry.Registry) {
	t.Helper()

	runtime, _ := registry.Get(containercontract.ContainerRuntimeConfig.String())
	var runtimeSchema struct {
		Type string   `json:"type"`
		Enum []string `json:"enum"`
	}
	if err := json.Unmarshal(runtime.Schema, &runtimeSchema); err != nil {
		t.Fatalf("decode runtime schema: %v", err)
	}
	if runtimeSchema.Type != "string" || !slices.Contains(runtimeSchema.Enum, defaultContainerRuntime) {
		t.Fatalf("expected runtime enum schema, got %#v", runtimeSchema)
	}
}

func assertMaxTailConfigSchema(t *testing.T, registry *configregistry.Registry) {
	t.Helper()

	maxTail, _ := registry.Get(containercontract.ContainerLogsMaxTailConfig.String())
	var maxTailSchema struct {
		Type    string   `json:"type"`
		Minimum *float64 `json:"minimum"`
		Maximum *float64 `json:"maximum"`
	}
	if err := json.Unmarshal(maxTail.Schema, &maxTailSchema); err != nil {
		t.Fatalf("decode max tail schema: %v", err)
	}
	if maxTailSchema.Type != "integer" || maxTailSchema.Maximum == nil || *maxTailSchema.Maximum != defaultContainerLogsMaxTail {
		t.Fatalf("expected max tail integer schema, got %#v", maxTailSchema)
	}

	resourceTTL, _ := registry.Get(containercontract.ContainerResourceStatsCacheTTLConfig.String())
	var resourceTTLSchema struct {
		Type    string   `json:"type"`
		Minimum *float64 `json:"minimum"`
		Maximum *float64 `json:"maximum"`
	}
	if err := json.Unmarshal(resourceTTL.Schema, &resourceTTLSchema); err != nil {
		t.Fatalf("decode resource ttl schema: %v", err)
	}
	if resourceTTLSchema.Type != "integer" || resourceTTLSchema.Minimum == nil || *resourceTTLSchema.Minimum != 1 {
		t.Fatalf("expected resource ttl integer schema, got %#v", resourceTTLSchema)
	}
}

func assertEnvironmentPolicyConfigSchema(t *testing.T, registry *configregistry.Registry, localizer *i18n.Service) {
	t.Helper()

	environmentPolicy, _ := registry.Get(containercontract.ContainerEnvironmentPolicyConfig.String())
	var environmentPolicySchema struct {
		Type  string   `json:"type"`
		Enum  []string `json:"enum"`
		XI18n struct {
			TitleKey   string `json:"titleKey"`
			EnumLabels map[string]struct {
				LabelKey       string `json:"labelKey"`
				DescriptionKey string `json:"descriptionKey"`
			} `json:"enumLabels"`
		} `json:"x-i18n"`
	}
	if err := json.Unmarshal(environmentPolicy.Schema, &environmentPolicySchema); err != nil {
		t.Fatalf("decode environment policy schema: %v", err)
	}
	if environmentPolicySchema.Type != "string" ||
		!slices.Contains(environmentPolicySchema.Enum, containercontract.ContainerEnvironmentPolicyHidden.String()) ||
		!slices.Contains(environmentPolicySchema.Enum, containercontract.ContainerEnvironmentPolicyMasked.String()) ||
		!slices.Contains(environmentPolicySchema.Enum, containercontract.ContainerEnvironmentPolicyPlain.String()) {
		t.Fatalf("expected environment policy enum schema, got %#v", environmentPolicySchema)
	}
	for value, metadata := range environmentPolicySchema.XI18n.EnumLabels {
		if strings.TrimSpace(metadata.LabelKey) == "" || strings.TrimSpace(metadata.DescriptionKey) == "" {
			t.Fatalf("expected enum i18n metadata for %s, got %#v", value, metadata)
		}
		assertRegisteredMessage(t, localizer, i18n.LocaleZHCN, metadata.LabelKey)
		assertRegisteredMessage(t, localizer, i18n.LocaleENUS, metadata.LabelKey)
		assertRegisteredMessage(t, localizer, i18n.LocaleZHCN, metadata.DescriptionKey)
		assertRegisteredMessage(t, localizer, i18n.LocaleENUS, metadata.DescriptionKey)
	}
}

func expectedConfigKeys() []string {
	return []string{
		containercontract.ContainerRuntimeEnabledConfig.String(),
		containercontract.ContainerRuntimeConfig.String(),
		containercontract.ContainerDockerEndpointConfig.String(),
		containercontract.ContainerLogsDefaultTailConfig.String(),
		containercontract.ContainerLogsMaxTailConfig.String(),
		containercontract.ContainerResourceStatsCacheTTLConfig.String(),
		containercontract.ContainerResourceStatsCacheStaleWindowConfig.String(),
		containercontract.ContainerDangerousActionsEnabledConfig.String(),
		containercontract.ContainerShellEnabledConfig.String(),
		containercontract.ContainerEnvironmentPolicyConfig.String(),
		containercontract.ContainerEnvironmentMaskedCopyEnabledConfig.String(),
	}
}

func assertConfigDefinitionMetadata(t *testing.T, definition configregistry.Definition, localizer *i18n.Service) {
	t.Helper()

	if definition.Module != moduleID || definition.Domain != containerConfigDomain {
		t.Fatalf("expected container-owned config definition, got %#v", definition)
	}
	if definition.Permission != containercontract.ContainerViewPermission.String() {
		t.Fatalf("expected config permission %s, got %#v", containercontract.ContainerViewPermission, definition)
	}
	for _, key := range []string{
		definition.DomainKey,
		definition.GroupKey,
		definition.GroupDescriptionKey,
		definition.TitleKey,
		definition.DescriptionKey,
	} {
		if strings.TrimSpace(key) == "" {
			t.Fatalf("expected config i18n metadata keys, got %#v", definition)
		}
		assertRegisteredMessage(t, localizer, i18n.LocaleZHCN, key)
		assertRegisteredMessage(t, localizer, i18n.LocaleENUS, key)
	}
	if !strings.HasPrefix(definition.TitleKey, "systemConfig.container.") {
		t.Fatalf("expected container config title key, got %q", definition.TitleKey)
	}
}

func assertRegisteredMessage(t *testing.T, localizer *i18n.Service, locale i18n.LocaleTag, key string) {
	t.Helper()

	matches := localizer.RegisteredMessageResources(locale, i18n.MessageKey(key))
	if len(matches) != 1 {
		t.Fatalf("expected one locale %s message for %q, got %#v", locale, key, matches)
	}
	if strings.TrimSpace(matches[0].Text) == "" {
		t.Fatalf("expected non-empty locale %s message for %q", locale, key)
	}
}
