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
	defaultContainerEnvironmentPolicy       = containercontract.ContainerEnvironmentPolicyMasked
	defaultContainerEnvironmentMaskedCopy   = false
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
			fallbackTitle:       "Container runtime access enabled",
			fallbackDescription: "Whether container management may access the configured runtime.",
			defaultValue:        mustRawJSON(defaultContainerEnabled),
		}),
		containerRuntimeDefinition(),
		containerEndpointDefinition(),
		containerIntegerDefinition(containerIntegerDefinitionSpec{
			containerDefinitionSpec: containerDefinitionSpec{
				key:                 containercontract.ContainerLogsDefaultTailConfig.String(),
				group:               containerConfigLogsGroup,
				fallbackTitle:       "Default log tail",
				fallbackDescription: "Default number of log lines returned by container log reads.",
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
				fallbackTitle:       "Maximum log tail",
				fallbackDescription: "Maximum number of log lines allowed for container log reads.",
				defaultValue:        mustRawJSON(defaultContainerLogsMaxTail),
			},
			defaultNumber: defaultContainerLogsMaxTail,
			minimum:       defaultContainerLogsDefaultTail,
			maximum:       defaultContainerLogsMaxTail,
		}),
		containerBooleanDefinition(containerDefinitionSpec{
			key:                 containercontract.ContainerDangerousActionsEnabledConfig.String(),
			group:               containerConfigActionsGroup,
			fallbackTitle:       "Dangerous actions enabled",
			fallbackDescription: "Whether start, stop, and restart actions are enabled.",
			defaultValue:        mustRawJSON(defaultContainerDangerousActionsEnabled),
		}),
		containerEnvironmentPolicyDefinition(),
		containerBooleanDefinition(containerDefinitionSpec{
			key:                 containercontract.ContainerEnvironmentMaskedCopyEnabledConfig.String(),
			group:               containerConfigGeneralGroup,
			fallbackTitle:       "Masked environment copy enabled",
			fallbackDescription: "Whether masked environment values may expose copy-only raw values to authorized users.",
			defaultValue:        mustRawJSON(defaultContainerEnvironmentMaskedCopy),
		}),
	}
}

func containerRuntimeDefinition() configregistry.Definition {
	return baseContainerDefinition(containerDefinitionSpec{
		key:                 containercontract.ContainerRuntimeConfig.String(),
		group:               containerConfigRuntimeGroup,
		fallbackTitle:       "Container runtime",
		fallbackDescription: "Runtime adapter used by container management.",
		valueType:           configregistry.ValueTypeString,
		defaultValue:        mustRawJSON(defaultContainerRuntime),
		schema:              containerRuntimeSchema(),
	})
}

func containerEndpointDefinition() configregistry.Definition {
	definition := baseContainerDefinition(containerDefinitionSpec{
		key:                 containercontract.ContainerDockerEndpointConfig.String(),
		group:               containerConfigRuntimeGroup,
		fallbackTitle:       "Container runtime endpoint",
		fallbackDescription: "Local runtime endpoint used by the first container adapter.",
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
		fallbackTitle:       "Environment value display policy",
		fallbackDescription: "Controls how container environment variable values are returned by detail reads.",
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
		DomainLabel:         "Operations",
		Group:               spec.group,
		GroupKey:            metadata.key,
		GroupLabel:          metadata.label,
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

func containerConfigGroupMetadata(group string) containerConfigGroupInfo {
	switch group {
	case containerConfigRuntimeGroup:
		return containerConfigGroupInfo{
			key:            containerConfigRuntimeGroupKey,
			label:          "Container runtime",
			descriptionKey: containerConfigRuntimeDescKey,
			description:    "Control the local container runtime adapter.",
		}
	case containerConfigLogsGroup:
		return containerConfigGroupInfo{
			key:            containerConfigLogsGroupKey,
			label:          "Container logs",
			descriptionKey: containerConfigLogsDescKey,
			description:    "Control bounded container log reads.",
		}
	case containerConfigActionsGroup:
		return containerConfigGroupInfo{
			key:            containerConfigActionsGroupKey,
			label:          "Container actions",
			descriptionKey: containerConfigActionsDescKey,
			description:    "Control high-risk container operations.",
		}
	default:
		return containerConfigGroupInfo{
			key:            containerConfigGeneralGroupKey,
			label:          "Container management",
			descriptionKey: containerConfigGeneralDescKey,
			description:    "Control the container management baseline.",
		}
	}
}

func containerRuntimeSchema() json.RawMessage {
	return json.RawMessage(fmt.Sprintf(
		`{"type":"string","enum":["first-adapter"],"default":%q,"title":"Container runtime","description":"Runtime adapter used by container management.","x-i18n":{"titleKey":%q,"descriptionKey":%q}}`,
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
		`{"type":"string","enum":[%q,%q,%q],"default":%q,"title":%q,"description":%q,"x-i18n":{"titleKey":%q,"descriptionKey":%q,"enumLabels":{"hidden":{"labelKey":"systemConfig.container.%s.enum.hidden.label","descriptionKey":"systemConfig.container.%s.enum.hidden.description"},"masked":{"labelKey":"systemConfig.container.%s.enum.masked.label","descriptionKey":"systemConfig.container.%s.enum.masked.description"},"plain":{"labelKey":"systemConfig.container.%s.enum.plain.label","descriptionKey":"systemConfig.container.%s.enum.plain.description"}}}}`,
		hiddenPolicy,
		maskedPolicy,
		plainPolicy,
		defaultContainerEnvironmentPolicy.String(),
		containerConfigTitleFallback(key),
		containerConfigDescriptionFallback(key),
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
		`{"type":"boolean","title":%q,"description":%q,"x-i18n":{"titleKey":%q,"descriptionKey":%q}}`,
		containerConfigTitleFallback(key),
		containerConfigDescriptionFallback(key),
		containerConfigTitleKey(key),
		containerConfigDescriptionKey(key),
	))
}

func containerIntegerSchema(key string, defaultValue int, minimum int, maximum int) json.RawMessage {
	return json.RawMessage(fmt.Sprintf(
		`{"type":"integer","minimum":%d,"maximum":%d,"default":%d,"title":%q,"description":%q,"x-i18n":{"titleKey":%q,"descriptionKey":%q,"unitKey":"systemConfig.units.rows"}}`,
		minimum,
		maximum,
		defaultValue,
		containerConfigTitleFallback(key),
		containerConfigDescriptionFallback(key),
		containerConfigTitleKey(key),
		containerConfigDescriptionKey(key),
	))
}

func containerStringSchema(key string, minimumLength int, maximumLength int) json.RawMessage {
	return json.RawMessage(fmt.Sprintf(
		`{"type":"string","minLength":%d,"maxLength":%d,"title":%q,"description":%q,"x-i18n":{"titleKey":%q,"descriptionKey":%q}}`,
		minimumLength,
		maximumLength,
		containerConfigTitleFallback(key),
		containerConfigDescriptionFallback(key),
		containerConfigTitleKey(key),
		containerConfigDescriptionKey(key),
	))
}

func containerConfigTitleFallback(key string) string {
	switch key {
	case containercontract.ContainerRuntimeEnabledConfig.String():
		return "Container Runtime Access Enabled"
	case containercontract.ContainerRuntimeConfig.String():
		return "Container Runtime"
	case containercontract.ContainerDockerEndpointConfig.String():
		return "Container Runtime Endpoint"
	case containercontract.ContainerLogsDefaultTailConfig.String():
		return "Default Log Tail"
	case containercontract.ContainerLogsMaxTailConfig.String():
		return "Maximum Log Tail"
	case containercontract.ContainerDangerousActionsEnabledConfig.String():
		return "Dangerous Container Actions Enabled"
	case containercontract.ContainerEnvironmentPolicyConfig.String():
		return "Environment Value Display Policy"
	case containercontract.ContainerEnvironmentMaskedCopyEnabledConfig.String():
		return "Masked Environment Copy Enabled"
	default:
		return key
	}
}

func containerConfigDescriptionFallback(key string) string {
	switch key {
	case containercontract.ContainerRuntimeEnabledConfig.String():
		return "Whether container management may access the configured runtime."
	case containercontract.ContainerRuntimeConfig.String():
		return "Runtime adapter used by container management."
	case containercontract.ContainerDockerEndpointConfig.String():
		return "Local runtime endpoint used by the first container adapter."
	case containercontract.ContainerLogsDefaultTailConfig.String():
		return "Default number of log lines returned by container log reads."
	case containercontract.ContainerLogsMaxTailConfig.String():
		return "Maximum number of log lines allowed for container log reads."
	case containercontract.ContainerDangerousActionsEnabledConfig.String():
		return "Whether start, stop, restart, and future high-risk actions are enabled."
	case containercontract.ContainerEnvironmentPolicyConfig.String():
		return "Controls how container environment variable values are returned by detail reads."
	case containercontract.ContainerEnvironmentMaskedCopyEnabledConfig.String():
		return "Allows authorized users to copy raw values for masked environment variables."
	default:
		return key
	}
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
	return nil
}
