// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package container

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"

	"graft/server/internal/config"
	"graft/server/internal/configregistry"
	"graft/server/internal/i18n"
	"graft/server/internal/menu"
	"graft/server/internal/module"
	"graft/server/internal/permission"
	containercontract "graft/server/modules/container/contract"
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
	if containercontract.ContainerDangerousActionsEnabledConfig.String() != "ops.container.actions.dangerous_enabled" {
		t.Fatalf("unexpected dangerous actions config key")
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
	return &module.Context{
		I18n:               i18n.MustNew(config.I18nConfig{DefaultLocale: "zh-CN", FallbackLocale: "zh-CN", SupportedLocales: []string{"zh-CN", "en-US"}}),
		MenuRegistry:       menu.NewRegistry(),
		PermissionRegistry: permission.NewRegistry(),
		ConfigRegistry:     configregistry.NewRegistry(),
	}
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
		title:                    "运维管理",
		titleKey:                 containercontract.OperationsMenuTitle.String(),
		path:                     "/ops",
		permission:               "",
		visibleWhenConfigEnabled: "",
	})
	assertMenuItem(t, items, expectedMenuItem{
		code:                     "container.list",
		title:                    "容器管理",
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
		containercontract.ContainerDangerousActionsDisabled.String(),
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

	assertRuntimeConfigSchema(t, registry)
	assertMaxTailConfigSchema(t, registry)
	assertEnvironmentPolicyConfigSchema(t, registry, localizer)
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
		containercontract.ContainerDangerousActionsEnabledConfig.String(),
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
