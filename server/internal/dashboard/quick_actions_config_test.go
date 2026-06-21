// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package dashboard

import (
	"encoding/json"
	"strings"
	"testing"

	"graft/server/internal/config"
	"graft/server/internal/configregistry"
	"graft/server/internal/i18n"
	"graft/server/internal/testassert"
	systemconfiglocales "graft/server/modules/system-config/locales"
)

func TestRegisterQuickActionsConfigDefinitionsUsesDomainGroupItemMetadata(t *testing.T) {
	t.Parallel()

	registry := configregistry.NewRegistry()
	if err := RegisterQuickActionsConfigDefinitions(registry); err != nil {
		t.Fatalf("register quick-actions config definitions: %v", err)
	}

	items := registry.Items()
	if len(items) != 1 {
		t.Fatalf("expected one quick-action config object item, got %#v", items)
	}
	for _, item := range items {
		assertQuickActionsHierarchyMetadata(t, item)
		assertQuickActionsSchemaI18nMetadata(t, item)
		assertQuickActionsDefaultValue(t, item)
	}
	assertOldQuickActionsConfigKeysRemoved(t, registry)
}

func assertQuickActionsHierarchyMetadata(t *testing.T, item configregistry.Definition) {
	t.Helper()

	if item.Domain != quickActionsConfigDomain ||
		item.DomainKey != quickActionsConfigDomainKey ||
		item.Group != quickActionsConfigGroup ||
		item.GroupKey != quickActionsConfigGroupKey ||
		item.GroupDescriptionKey != quickActionsConfigGroupDescKey {
		t.Fatalf("expected dashboard quick-action hierarchy metadata, got %#v", item)
	}
	if item.GroupLabel == "core / dashboard.quick_actions" {
		t.Fatalf("group label must be product-facing fallback, got %q", item.GroupLabel)
	}
}

func assertQuickActionsSchemaI18nMetadata(t *testing.T, item configregistry.Definition) {
	t.Helper()

	var schema struct {
		Type                 string                       `json:"type"`
		Required             []string                     `json:"required"`
		AdditionalProperties bool                         `json:"additionalProperties"`
		Properties           map[string]quickActionSchema `json:"properties"`
		XI18n                struct {
			TitleKey       string `json:"titleKey"`
			DescriptionKey string `json:"descriptionKey"`
		} `json:"x-i18n"`
	}
	if err := json.Unmarshal(item.Schema, &schema); err != nil {
		t.Fatalf("parse quick-actions config schema %s: %v", item.Key, err)
	}
	if item.Key != QuickActionsConfigKey || item.Type != configregistry.ValueTypeObject {
		t.Fatalf("expected canonical quick-action object config, got %#v", item)
	}
	if item.RuntimeApplyMode != configregistry.RuntimeApplyModeUnknown {
		t.Fatalf("expected dashboard quick-actions runtime apply mode to remain unknown, got %#v", item.RuntimeApplyMode)
	}
	if schema.Type != "object" || schema.AdditionalProperties {
		t.Fatalf("expected strict object schema, got %#v", schema)
	}
	if !testassert.SameStringSet(schema.Required, []string{"enabled", "maxItems", "strategy"}) {
		t.Fatalf("expected required quick-action fields, got %#v", schema.Required)
	}
	if schema.XI18n.TitleKey != quickActionsConfigTitleKey ||
		schema.XI18n.DescriptionKey != quickActionsConfigDescKey {
		t.Fatalf("expected object schema key-first metadata, got %#v", schema.XI18n)
	}
	assertQuickActionsFieldSchemas(t, schema.Properties)
}

func assertQuickActionsFieldSchemas(t *testing.T, properties map[string]quickActionSchema) {
	t.Helper()

	if len(properties) != 3 {
		t.Fatalf("expected three quick-action fields, got %#v", properties)
	}
	if string(properties["enabled"].Default) != "true" {
		t.Fatalf("expected enabled field default true, got %s", properties["enabled"].Default)
	}
	assertQuickActionsMaxItemsSchema(t, properties["maxItems"])
	assertQuickActionsStrategySchema(t, properties["strategy"])
}

func assertQuickActionsMaxItemsSchema(t *testing.T, schema quickActionSchema) {
	t.Helper()

	if string(schema.Default) != "4" ||
		schema.Minimum == nil ||
		*schema.Minimum != 1 ||
		schema.Maximum == nil ||
		*schema.Maximum != 24 {
		t.Fatalf("expected maxItems default and range constraints, got %#v", schema)
	}
}

func assertQuickActionsStrategySchema(t *testing.T, schema quickActionSchema) {
	t.Helper()

	if string(schema.Default) != `"hybrid"` || len(schema.Enum) != 3 {
		t.Fatalf("expected strategy default and enum values, got %#v", schema)
	}
	hybrid := schema.XI18n.EnumLabels["hybrid"]
	if hybrid.LabelKey != quickActionsStrategyHybridKey || hybrid.DescriptionKey != quickActionsStrategyHybridDesc {
		t.Fatalf("expected hybrid option key-first metadata, got %#v", hybrid)
	}
}

type quickActionSchema struct {
	Type    string          `json:"type"`
	Default json.RawMessage `json:"default"`
	Enum    []string        `json:"enum"`
	Minimum *float64        `json:"minimum"`
	Maximum *float64        `json:"maximum"`
	XI18n   struct {
		TitleKey       string `json:"titleKey"`
		DescriptionKey string `json:"descriptionKey"`
		EnumLabels     map[string]struct {
			LabelKey       string `json:"labelKey"`
			DescriptionKey string `json:"descriptionKey"`
		} `json:"enumLabels"`
	} `json:"x-i18n"`
}

func assertQuickActionsDefaultValue(t *testing.T, item configregistry.Definition) {
	t.Helper()

	if string(item.DefaultValue) != `{"enabled":true,"maxItems":4,"strategy":"hybrid"}` {
		t.Fatalf("expected quick-action object default value, got %s", item.DefaultValue)
	}
}

func assertOldQuickActionsConfigKeysRemoved(t *testing.T, registry *configregistry.Registry) {
	t.Helper()

	for _, key := range []string{
		"dashboard.quick_actions.enabled",
		"dashboard.quick_actions.max_items",
		"dashboard.quick_actions.strategy",
	} {
		if _, ok := registry.Get(key); ok {
			t.Fatalf("old dashboard quick-action flat key %s must not be registered", key)
		}
	}
}

func TestRegisterQuickActionsConfigMessagesUsesProductFacingChineseCopy(t *testing.T) {
	t.Parallel()

	localizer := i18n.MustNew(config.I18nConfig{
		DefaultLocale:    string(i18n.LocaleZHCN),
		FallbackLocale:   string(i18n.LocaleENUS),
		SupportedLocales: []string{string(i18n.LocaleZHCN), string(i18n.LocaleENUS)},
	})
	resources, err := systemconfiglocales.EmbeddedLocaleResources()
	if err != nil {
		t.Fatalf("load system-config locale resources: %v", err)
	}
	if err := localizer.RegisterEmbeddedLocaleResources(resources); err != nil {
		t.Fatalf("register system-config locale resources: %v", err)
	}

	if err := RegisterQuickActionsConfigMessages(localizer); err != nil {
		t.Fatalf("register quick-actions config messages: %v", err)
	}

	zhMessage := localizer.Lookup(i18n.LookupRequest{
		Namespace: "system-config",
		Locale:    i18n.LocaleZHCN,
		Key:       i18n.MessageKey(quickActionsConfigGroupKey),
	})
	if zhMessage != "工作台快捷入口" {
		t.Fatalf("expected zh-CN localized dashboard quick-actions group label, got %q", zhMessage)
	}

	enMessage := localizer.Lookup(i18n.LookupRequest{
		Namespace: "system-config",
		Locale:    i18n.LocaleENUS,
		Key:       i18n.MessageKey(quickActionsStrategyHybridDesc),
	})
	if enMessage != "Rank by recent visits, usage frequency, and system recommendations." {
		t.Fatalf("expected en-US localized hybrid description, got %q", enMessage)
	}

	matches := localizer.RegisteredMessageResources(i18n.LocaleZHCN, i18n.MessageKey(quickActionsConfigGroupKey))
	if len(matches) != 1 ||
		matches[0].Key != "system-config."+quickActionsConfigGroupKey ||
		matches[0].Text != "工作台快捷入口" {
		t.Fatalf("expected registered zh-CN dashboard quick-actions diagnostic, got %#v", matches)
	}

	ids := localizer.RegisteredMessageKeyIDs(i18n.LocaleENUS, i18n.MessageKey(quickActionsStrategyHybridDesc))
	if len(ids) != 1 || ids[0] != "system-config."+quickActionsStrategyHybridDesc {
		t.Fatalf("expected registered en-US dashboard quick-actions diagnostic key, got %#v", ids)
	}
}

func TestRegisterQuickActionsConfigMessagesRequiresPreRegisteredSystemConfigResources(t *testing.T) {
	t.Parallel()

	localizer := i18n.MustNew(config.I18nConfig{
		DefaultLocale:    string(i18n.LocaleZHCN),
		FallbackLocale:   string(i18n.LocaleENUS),
		SupportedLocales: []string{string(i18n.LocaleZHCN), string(i18n.LocaleENUS)},
	})

	err := RegisterQuickActionsConfigMessages(localizer)
	if err == nil {
		t.Fatal("expected missing pre-registered system-config locale resources to fail")
	}
	if !strings.Contains(err.Error(), "missing key") {
		t.Fatalf("expected missing key error, got %v", err)
	}
}
