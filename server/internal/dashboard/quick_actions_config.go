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
	quickActionsConfigSchema = `{"type":"object","properties":{"enabled":{"type":"boolean","default":true,"x-i18n":{"titleKey":"systemConfig.fields.dashboardQuickActions.enabled.title","descriptionKey":"systemConfig.fields.dashboardQuickActions.enabled.description"}},"maxItems":{"type":"integer","minimum":1,"maximum":24,"default":4,"x-i18n":{"titleKey":"systemConfig.fields.dashboardQuickActions.maxItems.title","descriptionKey":"systemConfig.fields.dashboardQuickActions.maxItems.description"}},"strategy":{"type":"string","enum":["most_used","recent","hybrid"],"default":"hybrid","x-i18n":{"titleKey":"systemConfig.fields.dashboardQuickActions.strategy.title","descriptionKey":"systemConfig.fields.dashboardQuickActions.strategy.description","enumLabels":{"most_used":{"labelKey":"systemConfig.options.dashboardQuickActionStrategy.mostUsed","descriptionKey":"systemConfig.options.dashboardQuickActionStrategyDescriptions.mostUsed"},"recent":{"labelKey":"systemConfig.options.dashboardQuickActionStrategy.recent","descriptionKey":"systemConfig.options.dashboardQuickActionStrategyDescriptions.recent"},"hybrid":{"labelKey":"systemConfig.options.dashboardQuickActionStrategy.hybrid","descriptionKey":"systemConfig.options.dashboardQuickActionStrategyDescriptions.hybrid"}}}}},"required":["enabled","maxItems","strategy"],"additionalProperties":false,"x-i18n":{"titleKey":"systemConfig.items.dashboardQuickActions.title","descriptionKey":"systemConfig.items.dashboardQuickActions.description"}}`
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
			DomainLabel:         "",
			Group:               quickActionsConfigGroup,
			GroupKey:            quickActionsConfigGroupKey,
			GroupLabel:          "",
			GroupDescription:    "",
			GroupDescriptionKey: quickActionsConfigGroupDescKey,
			Title:               "",
			TitleKey:            quickActionsConfigTitleKey,
			Description:         "",
			DescriptionKey:      quickActionsConfigDescKey,
			Tags:                []string{"dashboard", "quick_actions"},
			Type:                configregistry.ValueTypeObject,
			Schema:              json.RawMessage(quickActionsConfigSchema),
			DefaultValue:        json.RawMessage(`{"enabled":true,"maxItems":4,"strategy":"hybrid"}`),
			RuntimeApplyMode:    configregistry.RuntimeApplyModeUnknown,
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

// RegisterQuickActionsConfigMessages verifies that all required i18n message keys are registered in the provided localizer for dashboard quick-actions configuration display across supported locales. It returns an error if any required key is missing for any locale.
func RegisterQuickActionsConfigMessages(localizer *i18n.Service) error {
	if localizer == nil {
		return errors.New("i18n service is required")
	}
	for _, locale := range []i18n.LocaleTag{i18n.LocaleZHCN, i18n.LocaleENUS} {
		for _, key := range quickActionsConfigMessageKeys() {
			matches := localizer.RegisteredMessageResources(locale, i18n.MessageKey(key))
			if len(matches) == 0 {
				return fmt.Errorf("register dashboard quick-actions config messages: locale resource %s missing key %s", locale, key)
			}
		}
	}
	return nil
}

// quickActionsConfigMessageKeys returns the list of message keys required for i18n configuration of dashboard quick actions.
func quickActionsConfigMessageKeys() []string {
	return []string{
		quickActionsConfigGroupKey,
		quickActionsConfigGroupDescKey,
		quickActionsConfigTitleKey,
		quickActionsConfigDescKey,
		quickActionsEnabledTitleKey,
		quickActionsEnabledDescKey,
		quickActionsMaxItemsTitleKey,
		quickActionsMaxItemsDescKey,
		quickActionsStrategyTitleKey,
		quickActionsStrategyDescKey,
		quickActionsStrategyMostUsedKey,
		quickActionsStrategyMostUsedDesc,
		quickActionsStrategyRecentKey,
		quickActionsStrategyRecentDesc,
		quickActionsStrategyHybridKey,
		quickActionsStrategyHybridDesc,
	}
}
