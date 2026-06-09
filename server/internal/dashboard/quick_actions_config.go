// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package dashboard

import (
	"encoding/json"
	"errors"
	"fmt"

	"graft/server/internal/configregistry"
	"graft/server/internal/i18n"
)

// Dashboard quick-action config keys are the config-center authority consumed by the web dashboard.
const (
	QuickActionsEnabledConfigKey  = "dashboard.quick_actions.enabled"
	QuickActionsMaxItemsConfigKey = "dashboard.quick_actions.max_items"
	QuickActionsStrategyConfigKey = "dashboard.quick_actions.strategy"

	quickActionsConfigGroupKey       = "systemConfig.groups.dashboardQuickActions"
	quickActionsEnabledTitleKey      = "systemConfig.items.dashboardQuickActionsEnabled.title"
	quickActionsEnabledDescKey       = "systemConfig.items.dashboardQuickActionsEnabled.description"
	quickActionsMaxItemsTitleKey     = "systemConfig.items.dashboardQuickActionsMaxItems.title"
	quickActionsMaxItemsDescKey      = "systemConfig.items.dashboardQuickActionsMaxItems.description"
	quickActionsStrategyTitleKey     = "systemConfig.items.dashboardQuickActionsStrategy.title"
	quickActionsStrategyDescKey      = "systemConfig.items.dashboardQuickActionsStrategy.description"
	quickActionsStrategyMostUsedKey  = "systemConfig.options.dashboardQuickActionStrategy.mostUsed"
	quickActionsStrategyRecentKey    = "systemConfig.options.dashboardQuickActionStrategy.recent"
	quickActionsStrategyHybridKey    = "systemConfig.options.dashboardQuickActionStrategy.hybrid"
	quickActionsConfigDefinitionBase = 120
	quickActionsStrategyConfigOrder  = quickActionsConfigDefinitionBase + 2
)

const (
	quickActionsEnabledSchema  = `{"type":"boolean"}`
	quickActionsMaxItemsSchema = `{"type":"integer","minimum":1,"maximum":24,"default":8,"title":"Maximum quick actions","description":"Maximum personalized entries shown on the dashboard home page."}`
	quickActionsStrategySchema = `{"type":"string","enum":["most_used","recent","hybrid"],"default":"hybrid","title":"Quick action strategy","description":"Personalized quick action ranking strategy.","x-i18n":{"enumLabels":{"most_used":"systemConfig.options.dashboardQuickActionStrategy.mostUsed","recent":"systemConfig.options.dashboardQuickActionStrategy.recent","hybrid":"systemConfig.options.dashboardQuickActionStrategy.hybrid"}}}`
)

// RegisterQuickActionsConfigDefinitions exposes dashboard quick-action defaults as config-center authority.
func RegisterQuickActionsConfigDefinitions(registry *configregistry.Registry) error {
	if registry == nil {
		return errors.New("config registry is required")
	}

	definitions := []configregistry.Definition{
		{
			Key:            QuickActionsEnabledConfigKey,
			Module:         moduleKeyCore,
			Group:          "dashboard.quick_actions",
			GroupKey:       quickActionsConfigGroupKey,
			GroupLabel:     "core / dashboard.quick_actions",
			Title:          "Dashboard quick actions enabled",
			TitleKey:       quickActionsEnabledTitleKey,
			Description:    "Controls whether personalized dashboard quick actions are shown.",
			DescriptionKey: quickActionsEnabledDescKey,
			Tags:           []string{"dashboard", "quick_actions"},
			Type:           configregistry.ValueTypeBoolean,
			Schema:         json.RawMessage(quickActionsEnabledSchema),
			DefaultValue:   json.RawMessage("true"),
			Order:          quickActionsConfigDefinitionBase,
		},
		{
			Key:            QuickActionsMaxItemsConfigKey,
			Module:         moduleKeyCore,
			Group:          "dashboard.quick_actions",
			GroupKey:       quickActionsConfigGroupKey,
			GroupLabel:     "core / dashboard.quick_actions",
			Title:          "Dashboard quick actions maximum items",
			TitleKey:       quickActionsMaxItemsTitleKey,
			Description:    "Maximum personalized entries shown on the dashboard home page.",
			DescriptionKey: quickActionsMaxItemsDescKey,
			Tags:           []string{"dashboard", "quick_actions"},
			Type:           configregistry.ValueTypeInteger,
			Schema:         json.RawMessage(quickActionsMaxItemsSchema),
			DefaultValue:   json.RawMessage("8"),
			Order:          quickActionsConfigDefinitionBase + 1,
		},
		{
			Key:            QuickActionsStrategyConfigKey,
			Module:         moduleKeyCore,
			Group:          "dashboard.quick_actions",
			GroupKey:       quickActionsConfigGroupKey,
			GroupLabel:     "core / dashboard.quick_actions",
			Title:          "Dashboard quick actions ranking strategy",
			TitleKey:       quickActionsStrategyTitleKey,
			Description:    "Personalized quick action ranking strategy.",
			DescriptionKey: quickActionsStrategyDescKey,
			Tags:           []string{"dashboard", "quick_actions"},
			Type:           configregistry.ValueTypeString,
			Schema:         json.RawMessage(quickActionsStrategySchema),
			DefaultValue:   json.RawMessage(`"hybrid"`),
			Order:          quickActionsStrategyConfigOrder,
		},
	}

	for _, definition := range definitions {
		if err := registry.Register(definition); err != nil {
			return fmt.Errorf("register dashboard quick-actions config definition %s: %w", definition.Key, err)
		}
	}
	return nil
}

// RegisterQuickActionsConfigMessages registers system-config display metadata for dashboard quick actions.
func RegisterQuickActionsConfigMessages(localizer *i18n.Service) error {
	if localizer == nil {
		return errors.New("i18n service is required")
	}

	for _, registration := range []i18n.Registration{
		quickActionsConfigRegistration(i18n.LocaleZHCN, []string{
			"Dashboard 快捷入口",
			"控制工作台首页是否展示个性化快捷入口。",
			"快捷入口最大数量",
			"工作台首页默认展示的个性化入口数量。",
			"快捷入口排序策略",
			"个性化快捷入口的推荐排序策略。",
			"最常使用",
			"最近访问",
			"综合推荐",
		}),
		quickActionsConfigRegistration(i18n.LocaleENUS, []string{
			"Dashboard Quick Actions",
			"Controls whether personalized dashboard quick actions are shown.",
			"Quick Action Maximum Items",
			"Maximum personalized entries shown on the dashboard home page.",
			"Quick Action Ranking Strategy",
			"Personalized quick action ranking strategy.",
			"Most Used",
			"Recent",
			"Hybrid",
		}),
	} {
		if err := localizer.RegisterMessages(registration); err != nil {
			return fmt.Errorf("register dashboard quick-actions config messages: %w", err)
		}
	}
	return nil
}

func quickActionsConfigRegistration(locale i18n.LocaleTag, texts []string) i18n.Registration {
	keys := []string{
		quickActionsEnabledTitleKey,
		quickActionsEnabledDescKey,
		quickActionsMaxItemsTitleKey,
		quickActionsMaxItemsDescKey,
		quickActionsStrategyTitleKey,
		quickActionsStrategyDescKey,
		quickActionsStrategyMostUsedKey,
		quickActionsStrategyRecentKey,
		quickActionsStrategyHybridKey,
	}
	messages := []i18n.MessageResource{
		{Key: i18n.MessageKey(quickActionsConfigGroupKey), Text: "core / dashboard.quick_actions"},
	}
	for index, key := range keys {
		messages = append(messages, i18n.MessageResource{Key: i18n.MessageKey(key), Text: texts[index]})
	}
	return i18n.Registration{
		Namespace: "system-config",
		Locale:    locale,
		Messages:  messages,
	}
}
