// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package dashboard

import (
	"encoding/json"
	"testing"

	"graft/server/internal/config"
	"graft/server/internal/configregistry"
	"graft/server/internal/i18n"
)

func TestRegisterQuickActionsConfigDefinitionsUsesDomainGroupItemMetadata(t *testing.T) {
	t.Parallel()

	registry := configregistry.NewRegistry()
	if err := RegisterQuickActionsConfigDefinitions(registry); err != nil {
		t.Fatalf("register quick-actions config definitions: %v", err)
	}

	items := registry.Items()
	if len(items) != 3 {
		t.Fatalf("expected three quick-action config items, got %#v", items)
	}
	for _, item := range items {
		assertQuickActionsHierarchyMetadata(t, item)
		assertQuickActionsSchemaI18nMetadata(t, item)
		assertQuickActionsDefaultValue(t, item)
	}
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
		Default json.RawMessage `json:"default"`
		XI18n   struct {
			TitleKey       string `json:"titleKey"`
			DescriptionKey string `json:"descriptionKey"`
			EnumLabels     map[string]struct {
				LabelKey       string `json:"labelKey"`
				DescriptionKey string `json:"descriptionKey"`
			} `json:"enumLabels"`
		} `json:"x-i18n"`
	}
	if err := json.Unmarshal(item.Schema, &schema); err != nil {
		t.Fatalf("parse quick-actions config schema %s: %v", item.Key, err)
	}
	if schema.XI18n.TitleKey != expectedQuickActionsTitleKey(item.Key) ||
		schema.XI18n.DescriptionKey != expectedQuickActionsDescriptionKey(item.Key) {
		t.Fatalf("expected schema key-first metadata for %s, got %#v", item.Key, schema.XI18n)
	}
	if item.Key == QuickActionsMaxItemsConfigKey && string(schema.Default) != "4" {
		t.Fatalf("expected max-items schema default 4, got %s", schema.Default)
	}
	if item.Key != QuickActionsStrategyConfigKey {
		return
	}

	if len(schema.XI18n.EnumLabels) != 3 {
		t.Fatalf("expected strategy enum labels, got %#v", schema.XI18n.EnumLabels)
	}
	hybrid := schema.XI18n.EnumLabels["hybrid"]
	if hybrid.LabelKey != quickActionsStrategyHybridKey || hybrid.DescriptionKey != quickActionsStrategyHybridDesc {
		t.Fatalf("expected hybrid option key-first metadata, got %#v", hybrid)
	}
}

func assertQuickActionsDefaultValue(t *testing.T, item configregistry.Definition) {
	t.Helper()

	if item.Key == QuickActionsMaxItemsConfigKey && string(item.DefaultValue) != "4" {
		t.Fatalf("expected max-items default value 4, got %s", item.DefaultValue)
	}
}

func expectedQuickActionsTitleKey(key string) string {
	return map[string]string{
		QuickActionsEnabledConfigKey:  quickActionsEnabledTitleKey,
		QuickActionsMaxItemsConfigKey: quickActionsMaxItemsTitleKey,
		QuickActionsStrategyConfigKey: quickActionsStrategyTitleKey,
	}[key]
}

func expectedQuickActionsDescriptionKey(key string) string {
	return map[string]string{
		QuickActionsEnabledConfigKey:  quickActionsEnabledDescKey,
		QuickActionsMaxItemsConfigKey: quickActionsMaxItemsDescKey,
		QuickActionsStrategyConfigKey: quickActionsStrategyDescKey,
	}[key]
}

func TestRegisterQuickActionsConfigMessagesUsesProductFacingChineseCopy(t *testing.T) {
	t.Parallel()

	localizer := i18n.MustNew(config.I18nConfig{
		DefaultLocale:    string(i18n.LocaleZHCN),
		FallbackLocale:   string(i18n.LocaleENUS),
		SupportedLocales: []string{string(i18n.LocaleZHCN), string(i18n.LocaleENUS)},
	})
	if err := RegisterQuickActionsConfigMessages(localizer); err != nil {
		t.Fatalf("register quick-actions config messages: %v", err)
	}

	matches := localizer.RegisteredMessageResources(i18n.LocaleZHCN, i18n.MessageKey(quickActionsConfigGroupKey))
	if len(matches) != 1 || matches[0].Text != "工作台快捷入口" {
		t.Fatalf("expected localized dashboard quick-actions group label, got %#v", matches)
	}
}
