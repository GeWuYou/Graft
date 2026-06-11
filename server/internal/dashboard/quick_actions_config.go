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
	// QuickActionsConfigKey controls personalized dashboard quick-action visibility and ranking.
	QuickActionsConfigKey = "dashboard.quick_actions"

	quickActionsConfigDomain         = "dashboard"
	quickActionsConfigDomainKey      = "systemConfig.domains.dashboard"
	quickActionsConfigGroup          = "quick_actions"
	quickActionsConfigGroupKey       = "systemConfig.groups.dashboardQuickActions"
	quickActionsConfigGroupDescKey   = "systemConfig.groupDescriptions.dashboardQuickActions"
	quickActionsConfigTitleKey       = "systemConfig.items.dashboardQuickActions.title"
	quickActionsConfigDescKey        = "systemConfig.items.dashboardQuickActions.description"
	quickActionsEnabledTitleKey      = "systemConfig.fields.dashboardQuickActions.enabled.title"
	quickActionsEnabledDescKey       = "systemConfig.fields.dashboardQuickActions.enabled.description"
	quickActionsMaxItemsTitleKey     = "systemConfig.fields.dashboardQuickActions.maxItems.title"
	quickActionsMaxItemsDescKey      = "systemConfig.fields.dashboardQuickActions.maxItems.description"
	quickActionsStrategyTitleKey     = "systemConfig.fields.dashboardQuickActions.strategy.title"
	quickActionsStrategyDescKey      = "systemConfig.fields.dashboardQuickActions.strategy.description"
	quickActionsStrategyMostUsedKey  = "systemConfig.options.dashboardQuickActionStrategy.mostUsed"
	quickActionsStrategyMostUsedDesc = "systemConfig.options.dashboardQuickActionStrategyDescriptions.mostUsed"
	quickActionsStrategyRecentKey    = "systemConfig.options.dashboardQuickActionStrategy.recent"
	quickActionsStrategyRecentDesc   = "systemConfig.options.dashboardQuickActionStrategyDescriptions.recent"
	quickActionsStrategyHybridKey    = "systemConfig.options.dashboardQuickActionStrategy.hybrid"
	quickActionsStrategyHybridDesc   = "systemConfig.options.dashboardQuickActionStrategyDescriptions.hybrid"
	quickActionsConfigDefinitionBase = 120
)

const (
	quickActionsConfigSchema = `{"type":"object","title":"Dashboard quick actions","description":"Dashboard home quick-action visibility and ranking defaults.","properties":{"enabled":{"type":"boolean","default":true,"title":"Enabled","description":"Controls whether personalized dashboard quick actions are shown.","x-i18n":{"titleKey":"systemConfig.fields.dashboardQuickActions.enabled.title","descriptionKey":"systemConfig.fields.dashboardQuickActions.enabled.description"}},"maxItems":{"type":"integer","minimum":1,"maximum":24,"default":4,"title":"Maximum quick actions","description":"Maximum personalized entries shown on the dashboard home page.","x-i18n":{"titleKey":"systemConfig.fields.dashboardQuickActions.maxItems.title","descriptionKey":"systemConfig.fields.dashboardQuickActions.maxItems.description"}},"strategy":{"type":"string","enum":["most_used","recent","hybrid"],"default":"hybrid","title":"Quick action strategy","description":"Personalized quick action ranking strategy.","x-i18n":{"titleKey":"systemConfig.fields.dashboardQuickActions.strategy.title","descriptionKey":"systemConfig.fields.dashboardQuickActions.strategy.description","enumLabels":{"most_used":{"labelKey":"systemConfig.options.dashboardQuickActionStrategy.mostUsed","descriptionKey":"systemConfig.options.dashboardQuickActionStrategyDescriptions.mostUsed"},"recent":{"labelKey":"systemConfig.options.dashboardQuickActionStrategy.recent","descriptionKey":"systemConfig.options.dashboardQuickActionStrategyDescriptions.recent"},"hybrid":{"labelKey":"systemConfig.options.dashboardQuickActionStrategy.hybrid","descriptionKey":"systemConfig.options.dashboardQuickActionStrategyDescriptions.hybrid"}}}}},"required":["enabled","maxItems","strategy"],"additionalProperties":false,"x-i18n":{"titleKey":"systemConfig.items.dashboardQuickActions.title","descriptionKey":"systemConfig.items.dashboardQuickActions.description"}}`
)

// RegisterQuickActionsConfigDefinitions exposes dashboard quick-action defaults as config-center authority.
func RegisterQuickActionsConfigDefinitions(registry *configregistry.Registry) error {
	if registry == nil {
		return errors.New("config registry is required")
	}

	definitions := []configregistry.Definition{
		{
			Key:                 QuickActionsConfigKey,
			Module:              moduleKeyCore,
			Domain:              quickActionsConfigDomain,
			DomainKey:           quickActionsConfigDomainKey,
			DomainLabel:         "Dashboard",
			Group:               quickActionsConfigGroup,
			GroupKey:            quickActionsConfigGroupKey,
			GroupLabel:          "Quick actions",
			GroupDescription:    "Manage dashboard home quick-action visibility and ranking.",
			GroupDescriptionKey: quickActionsConfigGroupDescKey,
			Title:               "Dashboard quick actions",
			TitleKey:            quickActionsConfigTitleKey,
			Description:         "Dashboard home quick-action visibility and ranking defaults.",
			DescriptionKey:      quickActionsConfigDescKey,
			Tags:                []string{"dashboard", "quick_actions"},
			Type:                configregistry.ValueTypeObject,
			Schema:              json.RawMessage(quickActionsConfigSchema),
			DefaultValue:        json.RawMessage(`{"enabled":true,"maxItems":4,"strategy":"hybrid"}`),
			Order:               quickActionsConfigDefinitionBase,
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
			configTitle:        "工作台快捷入口",
			configDesc:         "工作台首页快捷入口的显示与排序默认配置。",
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
			configTitle:        "Dashboard Quick Actions",
			configDesc:         "Dashboard home quick-action visibility and ranking defaults.",
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
	configTitle        string
	configDesc         string
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
		{Key: i18n.MessageKey(quickActionsConfigTitleKey), Text: texts.configTitle},
		{Key: i18n.MessageKey(quickActionsConfigDescKey), Text: texts.configDesc},
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
