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

const (
	// QuickActionsEnabledConfigKey controls whether personalized dashboard quick actions are shown.
	QuickActionsEnabledConfigKey = "dashboard.quick_actions.enabled"
	// QuickActionsMaxItemsConfigKey limits the personalized dashboard quick-action count.
	QuickActionsMaxItemsConfigKey = "dashboard.quick_actions.max_items"
	// QuickActionsStrategyConfigKey selects the personalized dashboard quick-action ranking strategy.
	QuickActionsStrategyConfigKey = "dashboard.quick_actions.strategy"

	quickActionsConfigDomain         = "dashboard"
	quickActionsConfigDomainKey      = "systemConfig.domains.dashboard"
	quickActionsConfigGroup          = "quick_actions"
	quickActionsConfigGroupKey       = "systemConfig.groups.dashboardQuickActions"
	quickActionsConfigGroupDescKey   = "systemConfig.groupDescriptions.dashboardQuickActions"
	quickActionsEnabledTitleKey      = "systemConfig.items.dashboardQuickActionsEnabled.title"
	quickActionsEnabledDescKey       = "systemConfig.items.dashboardQuickActionsEnabled.description"
	quickActionsMaxItemsTitleKey     = "systemConfig.items.dashboardQuickActionsMaxItems.title"
	quickActionsMaxItemsDescKey      = "systemConfig.items.dashboardQuickActionsMaxItems.description"
	quickActionsStrategyTitleKey     = "systemConfig.items.dashboardQuickActionsStrategy.title"
	quickActionsStrategyDescKey      = "systemConfig.items.dashboardQuickActionsStrategy.description"
	quickActionsStrategyMostUsedKey  = "systemConfig.options.dashboardQuickActionStrategy.mostUsed"
	quickActionsStrategyMostUsedDesc = "systemConfig.options.dashboardQuickActionStrategyDescriptions.mostUsed"
	quickActionsStrategyRecentKey    = "systemConfig.options.dashboardQuickActionStrategy.recent"
	quickActionsStrategyRecentDesc   = "systemConfig.options.dashboardQuickActionStrategyDescriptions.recent"
	quickActionsStrategyHybridKey    = "systemConfig.options.dashboardQuickActionStrategy.hybrid"
	quickActionsStrategyHybridDesc   = "systemConfig.options.dashboardQuickActionStrategyDescriptions.hybrid"
	quickActionsConfigDefinitionBase = 120
	quickActionsStrategyConfigOrder  = quickActionsConfigDefinitionBase + 2
)

const (
	quickActionsEnabledSchema  = `{"type":"boolean","x-i18n":{"titleKey":"systemConfig.items.dashboardQuickActionsEnabled.title","descriptionKey":"systemConfig.items.dashboardQuickActionsEnabled.description"}}`
	quickActionsMaxItemsSchema = `{"type":"integer","minimum":1,"maximum":24,"default":4,"title":"Maximum quick actions","description":"Maximum personalized entries shown on the dashboard home page.","x-i18n":{"titleKey":"systemConfig.items.dashboardQuickActionsMaxItems.title","descriptionKey":"systemConfig.items.dashboardQuickActionsMaxItems.description"}}`
	quickActionsStrategySchema = `{"type":"string","enum":["most_used","recent","hybrid"],"default":"hybrid","title":"Quick action strategy","description":"Personalized quick action ranking strategy.","x-i18n":{"titleKey":"systemConfig.items.dashboardQuickActionsStrategy.title","descriptionKey":"systemConfig.items.dashboardQuickActionsStrategy.description","enumLabels":{"most_used":{"labelKey":"systemConfig.options.dashboardQuickActionStrategy.mostUsed","descriptionKey":"systemConfig.options.dashboardQuickActionStrategyDescriptions.mostUsed"},"recent":{"labelKey":"systemConfig.options.dashboardQuickActionStrategy.recent","descriptionKey":"systemConfig.options.dashboardQuickActionStrategyDescriptions.recent"},"hybrid":{"labelKey":"systemConfig.options.dashboardQuickActionStrategy.hybrid","descriptionKey":"systemConfig.options.dashboardQuickActionStrategyDescriptions.hybrid"}}}}`
)

// RegisterQuickActionsConfigDefinitions exposes dashboard quick-action defaults as config-center authority.
func RegisterQuickActionsConfigDefinitions(registry *configregistry.Registry) error {
	if registry == nil {
		return errors.New("config registry is required")
	}

	definitions := []configregistry.Definition{
		{
			Key:                 QuickActionsEnabledConfigKey,
			Module:              moduleKeyCore,
			Domain:              quickActionsConfigDomain,
			DomainKey:           quickActionsConfigDomainKey,
			DomainLabel:         "Dashboard",
			Group:               quickActionsConfigGroup,
			GroupKey:            quickActionsConfigGroupKey,
			GroupLabel:          "Quick actions",
			GroupDescription:    "Manage dashboard home quick-action visibility and ranking.",
			GroupDescriptionKey: quickActionsConfigGroupDescKey,
			Title:               "Dashboard quick actions enabled",
			TitleKey:            quickActionsEnabledTitleKey,
			Description:         "Controls whether personalized dashboard quick actions are shown.",
			DescriptionKey:      quickActionsEnabledDescKey,
			Tags:                []string{"dashboard", "quick_actions"},
			Type:                configregistry.ValueTypeBoolean,
			Schema:              json.RawMessage(quickActionsEnabledSchema),
			DefaultValue:        json.RawMessage("true"),
			Order:               quickActionsConfigDefinitionBase,
		},
		{
			Key:                 QuickActionsMaxItemsConfigKey,
			Module:              moduleKeyCore,
			Domain:              quickActionsConfigDomain,
			DomainKey:           quickActionsConfigDomainKey,
			DomainLabel:         "Dashboard",
			Group:               quickActionsConfigGroup,
			GroupKey:            quickActionsConfigGroupKey,
			GroupLabel:          "Quick actions",
			GroupDescription:    "Manage dashboard home quick-action visibility and ranking.",
			GroupDescriptionKey: quickActionsConfigGroupDescKey,
			Title:               "Dashboard quick actions maximum items",
			TitleKey:            quickActionsMaxItemsTitleKey,
			Description:         "Maximum personalized entries shown on the dashboard home page.",
			DescriptionKey:      quickActionsMaxItemsDescKey,
			Tags:                []string{"dashboard", "quick_actions"},
			Type:                configregistry.ValueTypeInteger,
			Schema:              json.RawMessage(quickActionsMaxItemsSchema),
			DefaultValue:        json.RawMessage("4"),
			Order:               quickActionsConfigDefinitionBase + 1,
		},
		{
			Key:                 QuickActionsStrategyConfigKey,
			Module:              moduleKeyCore,
			Domain:              quickActionsConfigDomain,
			DomainKey:           quickActionsConfigDomainKey,
			DomainLabel:         "Dashboard",
			Group:               quickActionsConfigGroup,
			GroupKey:            quickActionsConfigGroupKey,
			GroupLabel:          "Quick actions",
			GroupDescription:    "Manage dashboard home quick-action visibility and ranking.",
			GroupDescriptionKey: quickActionsConfigGroupDescKey,
			Title:               "Dashboard quick actions ranking strategy",
			TitleKey:            quickActionsStrategyTitleKey,
			Description:         "Personalized quick action ranking strategy.",
			DescriptionKey:      quickActionsStrategyDescKey,
			Tags:                []string{"dashboard", "quick_actions"},
			Type:                configregistry.ValueTypeString,
			Schema:              json.RawMessage(quickActionsStrategySchema),
			DefaultValue:        json.RawMessage(`"hybrid"`),
			Order:               quickActionsStrategyConfigOrder,
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
		quickActionsConfigRegistration(i18n.LocaleZHCN, quickActionsConfigTexts{
			domainLabel:        "工作台配置",
			groupLabel:         "工作台快捷入口",
			groupDesc:          "管理首页快捷入口的显示与排序策略。",
			enabledTitle:       "是否启用",
			enabledDesc:        "控制工作台首页是否展示个性化快捷入口。",
			maxItemsTitle:      "最大数量",
			maxItemsDesc:       "工作台首页默认展示的个性化入口数量。",
			strategyTitle:      "排序策略",
			strategyDesc:       "个性化快捷入口的推荐排序策略。",
			mostUsedOption:     "最常使用",
			mostUsedOptionDesc: "优先展示使用频率最高的快捷入口。",
			recentOption:       "最近访问",
			recentOptionDesc:   "优先展示最近访问过的快捷入口。",
			hybridOption:       "综合推荐",
			hybridOptionDesc:   "根据最近访问、使用频率和系统推荐结果综合排序。",
		}),
		quickActionsConfigRegistration(i18n.LocaleENUS, quickActionsConfigTexts{
			domainLabel:        "Dashboard Configuration",
			groupLabel:         "Dashboard Quick Actions",
			groupDesc:          "Manage dashboard home quick-action visibility and ranking.",
			enabledTitle:       "Enabled",
			enabledDesc:        "Controls whether personalized dashboard quick actions are shown.",
			maxItemsTitle:      "Maximum Items",
			maxItemsDesc:       "Maximum personalized entries shown on the dashboard home page.",
			strategyTitle:      "Ranking Strategy",
			strategyDesc:       "Personalized quick action ranking strategy.",
			mostUsedOption:     "Most Used",
			mostUsedOptionDesc: "Prioritize the quick actions used most often.",
			recentOption:       "Recent",
			recentOptionDesc:   "Prioritize quick actions visited most recently.",
			hybridOption:       "Hybrid",
			hybridOptionDesc:   "Rank by recent visits, usage frequency, and system recommendations.",
		}),
	} {
		if err := localizer.RegisterMessages(registration); err != nil {
			return fmt.Errorf("register dashboard quick-actions config messages: %w", err)
		}
	}
	return nil
}

type quickActionsConfigTexts struct {
	domainLabel        string
	groupLabel         string
	groupDesc          string
	enabledTitle       string
	enabledDesc        string
	maxItemsTitle      string
	maxItemsDesc       string
	strategyTitle      string
	strategyDesc       string
	mostUsedOption     string
	mostUsedOptionDesc string
	recentOption       string
	recentOptionDesc   string
	hybridOption       string
	hybridOptionDesc   string
}

func quickActionsConfigRegistration(locale i18n.LocaleTag, texts quickActionsConfigTexts) i18n.Registration {
	messages := []i18n.MessageResource{
		{Key: i18n.MessageKey(quickActionsConfigDomainKey), Text: texts.domainLabel},
		{Key: i18n.MessageKey(quickActionsConfigGroupKey), Text: texts.groupLabel},
		{Key: i18n.MessageKey(quickActionsConfigGroupDescKey), Text: texts.groupDesc},
		{Key: i18n.MessageKey(quickActionsEnabledTitleKey), Text: texts.enabledTitle},
		{Key: i18n.MessageKey(quickActionsEnabledDescKey), Text: texts.enabledDesc},
		{Key: i18n.MessageKey(quickActionsMaxItemsTitleKey), Text: texts.maxItemsTitle},
		{Key: i18n.MessageKey(quickActionsMaxItemsDescKey), Text: texts.maxItemsDesc},
		{Key: i18n.MessageKey(quickActionsStrategyTitleKey), Text: texts.strategyTitle},
		{Key: i18n.MessageKey(quickActionsStrategyDescKey), Text: texts.strategyDesc},
		{Key: i18n.MessageKey(quickActionsStrategyMostUsedKey), Text: texts.mostUsedOption},
		{Key: i18n.MessageKey(quickActionsStrategyMostUsedDesc), Text: texts.mostUsedOptionDesc},
		{Key: i18n.MessageKey(quickActionsStrategyRecentKey), Text: texts.recentOption},
		{Key: i18n.MessageKey(quickActionsStrategyRecentDesc), Text: texts.recentOptionDesc},
		{Key: i18n.MessageKey(quickActionsStrategyHybridKey), Text: texts.hybridOption},
		{Key: i18n.MessageKey(quickActionsStrategyHybridDesc), Text: texts.hybridOptionDesc},
	}
	return i18n.Registration{
		Namespace: "system-config",
		Locale:    locale,
		Messages:  messages,
	}
}
