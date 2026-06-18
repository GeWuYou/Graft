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
	return nil
}
