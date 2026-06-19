// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package container

import (
	"encoding/json"
	"errors"
	"fmt"

	"graft/server/internal/configregistry"
	"graft/server/internal/i18n"
	containercontract "graft/server/modules/container/contract"
)

const (
	containerConfigDomain            = "ops"
	containerConfigDomainKey         = "systemConfig.domains.ops"
	containerConfigGeneralGroup      = "ops.container.general"
	containerConfigGeneralGroupKey   = "systemConfig.groups.ops.container.general"
	containerConfigGeneralDescKey    = "systemConfig.groups.ops.container.general.description"
	containerConfigRuntimeGroup      = "ops.container.runtime"
	containerConfigRuntimeGroupKey   = "systemConfig.groups.ops.container.runtime"
	containerConfigRuntimeDescKey    = "systemConfig.groups.ops.container.runtime.description"
	containerConfigLogsGroup         = "ops.container.logs"
	containerConfigLogsGroupKey      = "systemConfig.groups.ops.container.logs"
	containerConfigLogsDescKey       = "systemConfig.groups.ops.container.logs.description"
	containerConfigActionsGroup      = "ops.container.actions"
	containerConfigActionsGroupKey   = "systemConfig.groups.ops.container.actions"
	containerConfigActionsDescKey    = "systemConfig.groups.ops.container.actions.description"
	containerConfigShellGroup        = "ops.container.shell"
	containerConfigShellGroupKey     = "systemConfig.groups.ops.container.shell"
	containerConfigShellDescKey      = "systemConfig.groups.ops.container.shell.description"
	containerConfigDefinitionBaseOrd = 6200
	maxDockerEndpointLength          = 512
)

const (
	defaultContainerEnabled                 = false
	defaultContainerRuntime                 = "first-adapter"
	defaultContainerDockerEndpoint          = "unix:///var/run/docker.sock"
	defaultContainerLogsDefaultTail         = 200
	defaultContainerLogsMaxTail             = 2000
	defaultContainerDangerousActionsEnabled = false
	defaultContainerShellEnabled            = false
	defaultContainerEnvironmentPolicy       = containercontract.ContainerEnvironmentPolicyMasked
	defaultContainerEnvironmentMaskedCopy   = false
	containerConfigEstimatedKeysPerItem     = 8
)

// registerConfig registers container configuration definitions and i18n messages.
func registerConfig(localizer *i18n.Service, registry *configregistry.Registry) error {
	if err := registerConfigMessages(localizer); err != nil {
		return err
	}
	return registerConfigDefinitions(registry)
}

func registerConfigDefinitions(registry *configregistry.Registry) error {
	if registry == nil {
		return errors.New("config registry is required")
	}
	for index, definition := range configDefinitions() {
		definition.Order = containerConfigDefinitionBaseOrd + index
		if err := registry.Register(definition); err != nil {
			return fmt.Errorf("register container config definition %s: %w", definition.Key, err)
		}
	}
	return nil
}

// configDefinitions returns the list of all container configuration definitions in registration order.
func configDefinitions() []configregistry.Definition {
	return []configregistry.Definition{
		containerBooleanDefinition(containerDefinitionSpec{
			key:                 containercontract.ContainerRuntimeEnabledConfig.String(),
			group:               containerConfigGeneralGroup,
			fallbackTitle:       "",
			fallbackDescription: "",
			defaultValue:        mustRawJSON(defaultContainerEnabled),
		}),
		containerRuntimeDefinition(),
		containerEndpointDefinition(),
		containerIntegerDefinition(containerIntegerDefinitionSpec{
			containerDefinitionSpec: containerDefinitionSpec{
				key:                 containercontract.ContainerLogsDefaultTailConfig.String(),
				group:               containerConfigLogsGroup,
				fallbackTitle:       "",
				fallbackDescription: "",
				defaultValue:        mustRawJSON(defaultContainerLogsDefaultTail),
			},
			defaultNumber: defaultContainerLogsDefaultTail,
			minimum:       1,
			maximum:       defaultContainerLogsMaxTail,
		}),
		containerIntegerDefinition(containerIntegerDefinitionSpec{
			containerDefinitionSpec: containerDefinitionSpec{
				key:                 containercontract.ContainerLogsMaxTailConfig.String(),
				group:               containerConfigLogsGroup,
				fallbackTitle:       "",
				fallbackDescription: "",
				defaultValue:        mustRawJSON(defaultContainerLogsMaxTail),
			},
			defaultNumber: defaultContainerLogsMaxTail,
			minimum:       defaultContainerLogsDefaultTail,
			maximum:       defaultContainerLogsMaxTail,
		}),
		containerBooleanDefinition(containerDefinitionSpec{
			key:                 containercontract.ContainerDangerousActionsEnabledConfig.String(),
			group:               containerConfigActionsGroup,
			fallbackTitle:       "",
			fallbackDescription: "",
			defaultValue:        mustRawJSON(defaultContainerDangerousActionsEnabled),
		}),
		containerBooleanDefinition(containerDefinitionSpec{
			key:                 containercontract.ContainerShellEnabledConfig.String(),
			group:               containerConfigShellGroup,
			fallbackTitle:       "",
			fallbackDescription: "",
			defaultValue:        mustRawJSON(defaultContainerShellEnabled),
		}),
		containerEnvironmentPolicyDefinition(),
		containerBooleanDefinition(containerDefinitionSpec{
			key:                 containercontract.ContainerEnvironmentMaskedCopyEnabledConfig.String(),
			group:               containerConfigGeneralGroup,
			fallbackTitle:       "",
			fallbackDescription: "",
			defaultValue:        mustRawJSON(defaultContainerEnvironmentMaskedCopy),
		}),
	}
}

func containerRuntimeDefinition() configregistry.Definition {
	return baseContainerDefinition(containerDefinitionSpec{
		key:                 containercontract.ContainerRuntimeConfig.String(),
		group:               containerConfigRuntimeGroup,
		fallbackTitle:       "",
		fallbackDescription: "",
		valueType:           configregistry.ValueTypeString,
		defaultValue:        mustRawJSON(defaultContainerRuntime),
		schema:              containerRuntimeSchema(),
	})
}

func containerEndpointDefinition() configregistry.Definition {
	definition := baseContainerDefinition(containerDefinitionSpec{
		key:                 containercontract.ContainerDockerEndpointConfig.String(),
		group:               containerConfigRuntimeGroup,
		fallbackTitle:       "",
		fallbackDescription: "",
		valueType:           configregistry.ValueTypeString,
		defaultValue:        mustRawJSON(defaultContainerDockerEndpoint),
		schema:              containerStringSchema(containercontract.ContainerDockerEndpointConfig.String(), 1, maxDockerEndpointLength),
	})
	definition.RestartRequired = true
	return definition
}

func containerEnvironmentPolicyDefinition() configregistry.Definition {
	return baseContainerDefinition(containerDefinitionSpec{
		key:                 containercontract.ContainerEnvironmentPolicyConfig.String(),
		group:               containerConfigGeneralGroup,
		fallbackTitle:       "",
		fallbackDescription: "",
		valueType:           configregistry.ValueTypeString,
		defaultValue:        mustRawJSON(defaultContainerEnvironmentPolicy.String()),
		schema:              containerEnvironmentPolicySchema(),
	})
}

type containerDefinitionSpec struct {
	key                 string
	group               string
	fallbackTitle       string
	fallbackDescription string
	valueType           configregistry.ValueType
	defaultValue        json.RawMessage
	schema              json.RawMessage
}

type containerIntegerDefinitionSpec struct {
	containerDefinitionSpec
	defaultNumber int
	minimum       int
	maximum       int
}

func containerBooleanDefinition(spec containerDefinitionSpec) configregistry.Definition {
	spec.valueType = configregistry.ValueTypeBoolean
	spec.schema = containerBooleanSchema(spec.key)
	return baseContainerDefinition(spec)
}

func containerIntegerDefinition(spec containerIntegerDefinitionSpec) configregistry.Definition {
	definitionSpec := spec.containerDefinitionSpec
	definitionSpec.valueType = configregistry.ValueTypeInteger
	definitionSpec.schema = containerIntegerSchema(definitionSpec.key, spec.defaultNumber, spec.minimum, spec.maximum)
	return baseContainerDefinition(definitionSpec)
}

func baseContainerDefinition(spec containerDefinitionSpec) configregistry.Definition {
	metadata := containerConfigGroupMetadata(spec.group)
	return configregistry.Definition{
		Key:                 spec.key,
		Module:              moduleID,
		Domain:              containerConfigDomain,
		DomainKey:           containerConfigDomainKey,
		DomainLabel:         "",
		Group:               spec.group,
		GroupKey:            metadata.key,
		GroupLabel:          "",
		GroupDescription:    metadata.description,
		GroupDescriptionKey: metadata.descriptionKey,
		Title:               spec.fallbackTitle,
		TitleKey:            containerConfigTitleKey(spec.key),
		Description:         spec.fallbackDescription,
		DescriptionKey:      containerConfigDescriptionKey(spec.key),
		Tags:                []string{"ops", "container", spec.group},
		Type:                spec.valueType,
		Schema:              spec.schema,
		DefaultValue:        spec.defaultValue,
		Permission:          containercontract.ContainerViewPermission.String(),
	}
}

type containerConfigGroupInfo struct {
	key            string
	label          string
	descriptionKey string
	description    string
}

// containerConfigGroupMetadata returns the configuration group metadata including the group key and description i18n key for the given group. If the group is not recognized, the general group metadata is returned.
func containerConfigGroupMetadata(group string) containerConfigGroupInfo {
	switch group {
	case containerConfigRuntimeGroup:
		return containerConfigGroupInfo{
			key:            containerConfigRuntimeGroupKey,
			label:          "",
			descriptionKey: containerConfigRuntimeDescKey,
			description:    "",
		}
	case containerConfigLogsGroup:
		return containerConfigGroupInfo{
			key:            containerConfigLogsGroupKey,
			label:          "",
			descriptionKey: containerConfigLogsDescKey,
			description:    "",
		}
	case containerConfigActionsGroup:
		return containerConfigGroupInfo{
			key:            containerConfigActionsGroupKey,
			label:          "",
			descriptionKey: containerConfigActionsDescKey,
			description:    "",
		}
	case containerConfigShellGroup:
		return containerConfigGroupInfo{
			key:            containerConfigShellGroupKey,
			label:          "",
			descriptionKey: containerConfigShellDescKey,
			description:    "",
		}
	default:
		return containerConfigGroupInfo{
			key:            containerConfigGeneralGroupKey,
			label:          "",
			descriptionKey: containerConfigGeneralDescKey,
			description:    "",
		}
	}
}

func containerRuntimeSchema() json.RawMessage {
	return json.RawMessage(fmt.Sprintf(
		`{"type":"string","enum":["first-adapter"],"default":%q,"x-i18n":{"titleKey":%q,"descriptionKey":%q}}`,
		defaultContainerRuntime,
		containerConfigTitleKey(containercontract.ContainerRuntimeConfig.String()),
		containerConfigDescriptionKey(containercontract.ContainerRuntimeConfig.String()),
	))
}

func containerEnvironmentPolicySchema() json.RawMessage {
	key := containercontract.ContainerEnvironmentPolicyConfig.String()
	hiddenPolicy := containercontract.ContainerEnvironmentPolicyHidden.String()
	maskedPolicy := containercontract.ContainerEnvironmentPolicyMasked.String()
	plainPolicy := containercontract.ContainerEnvironmentPolicyPlain.String()
	return json.RawMessage(fmt.Sprintf(
		`{"type":"string","enum":[%q,%q,%q],"default":%q,"x-i18n":{"titleKey":%q,"descriptionKey":%q,"enumLabels":{"hidden":{"labelKey":"systemConfig.container.%s.enum.hidden.label","descriptionKey":"systemConfig.container.%s.enum.hidden.description"},"masked":{"labelKey":"systemConfig.container.%s.enum.masked.label","descriptionKey":"systemConfig.container.%s.enum.masked.description"},"plain":{"labelKey":"systemConfig.container.%s.enum.plain.label","descriptionKey":"systemConfig.container.%s.enum.plain.description"}}}}`,
		hiddenPolicy,
		maskedPolicy,
		plainPolicy,
		defaultContainerEnvironmentPolicy.String(),
		containerConfigTitleKey(key),
		containerConfigDescriptionKey(key),
		key,
		key,
		key,
		key,
		key,
		key,
	))
}

func containerBooleanSchema(key string) json.RawMessage {
	return json.RawMessage(fmt.Sprintf(
		`{"type":"boolean","x-i18n":{"titleKey":%q,"descriptionKey":%q}}`,
		containerConfigTitleKey(key),
		containerConfigDescriptionKey(key),
	))
}

func containerIntegerSchema(key string, defaultValue int, minimum int, maximum int) json.RawMessage {
	return json.RawMessage(fmt.Sprintf(
		`{"type":"integer","minimum":%d,"maximum":%d,"default":%d,"x-i18n":{"titleKey":%q,"descriptionKey":%q,"unitKey":"systemConfig.units.rows"}}`,
		minimum,
		maximum,
		defaultValue,
		containerConfigTitleKey(key),
		containerConfigDescriptionKey(key),
	))
}

func containerStringSchema(key string, minimumLength int, maximumLength int) json.RawMessage {
	return json.RawMessage(fmt.Sprintf(
		`{"type":"string","minLength":%d,"maxLength":%d,"x-i18n":{"titleKey":%q,"descriptionKey":%q}}`,
		minimumLength,
		maximumLength,
		containerConfigTitleKey(key),
		containerConfigDescriptionKey(key),
	))
}

func containerConfigTitleKey(key string) string {
	return "systemConfig.container." + key + ".title"
}

func containerConfigDescriptionKey(key string) string {
	return "systemConfig.container." + key + ".description"
}

func mustRawJSON(value any) json.RawMessage {
	raw, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return raw
}

func registerConfigMessages(localizer *i18n.Service) error {
	if localizer == nil {
		return errors.New("i18n service is required")
	}
	keys, err := containerConfigMessageKeys()
	if err != nil {
		return err
	}
	for _, locale := range []i18n.LocaleTag{i18n.LocaleZHCN, i18n.LocaleENUS} {
		for _, key := range keys {
			matches := localizer.RegisteredMessageResources(locale, i18n.MessageKey(key))
			if len(matches) == 0 {
				return fmt.Errorf("register container config messages: locale resource %s missing key %s", locale, key)
			}
		}
	}
	return nil
}

func containerConfigMessageKeys() ([]string, error) {
	keys := make([]string, 0, len(configDefinitions())*containerConfigEstimatedKeysPerItem)
	seen := make(map[string]struct{}, len(configDefinitions())*containerConfigEstimatedKeysPerItem)
	appendKey := func(key string) {
		if key == "" {
			return
		}
		if _, exists := seen[key]; exists {
			return
		}
		seen[key] = struct{}{}
		keys = append(keys, key)
	}

	for _, definition := range configDefinitions() {
		appendKey(definition.DomainKey)
		appendKey(definition.GroupKey)
		appendKey(definition.GroupDescriptionKey)
		appendKey(definition.TitleKey)
		appendKey(definition.DescriptionKey)
		if err := appendContainerSchemaMessageKeys(definition.Key, definition.Schema, appendKey); err != nil {
			return nil, err
		}
	}

	return keys, nil
}

func appendContainerSchemaMessageKeys(configKey string, raw json.RawMessage, appendKey func(string)) error {
	if len(raw) == 0 {
		return nil
	}

	var decoded any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return fmt.Errorf("register container config messages: decode schema for %s: %w", configKey, err)
	}

	collectContainerSchemaMessageKeys(decoded, appendKey)
	return nil
}

func collectContainerSchemaMessageKeys(node any, appendKey func(string)) {
	switch typed := node.(type) {
	case map[string]any:
		for key, value := range typed {
			switch key {
			case "titleKey", "descriptionKey", "labelKey", "unitKey":
				if text, ok := value.(string); ok {
					appendKey(text)
				}
			}
			collectContainerSchemaMessageKeys(value, appendKey)
		}
	case []any:
		for _, item := range typed {
			collectContainerSchemaMessageKeys(item, appendKey)
		}
	}
}
