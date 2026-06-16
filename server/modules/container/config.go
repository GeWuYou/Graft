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
)

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
	for configKey, copy := range enUSContainerConfigCopy() {
		if configKey == key {
			return copy[0]
		}
	}
	return key
}

func containerConfigDescriptionFallback(key string) string {
	for configKey, copy := range enUSContainerConfigCopy() {
		if configKey == key {
			return copy[1]
		}
	}
	return key
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
	for _, registration := range configMessageRegistrations() {
		if err := localizer.RegisterMessages(registration); err != nil {
			return fmt.Errorf("register container config messages: %w", err)
		}
	}
	return nil
}

func configMessageRegistrations() []i18n.Registration {
	return []i18n.Registration{
		{
			Namespace: "system-config",
			Locale:    i18n.LocaleZHCN,
			Messages: containerConfigMessages(map[string]string{
				containerConfigDomainKey:       "运维管理",
				containerConfigGeneralGroupKey: "容器管理",
				containerConfigGeneralDescKey:  "控制容器管理能力的基础开关。",
				containerConfigRuntimeGroupKey: "运行时",
				containerConfigRuntimeDescKey:  "控制本地容器运行时适配器。",
				containerConfigLogsGroupKey:    "日志",
				containerConfigLogsDescKey:     "控制容器日志读取上限。",
				containerConfigActionsGroupKey: "高危操作",
				containerConfigActionsDescKey:  "控制容器启停和重启操作。",
			}, zhCNContainerConfigCopy(), zhCNContainerEnvironmentPolicyEnumCopy()),
		},
		{
			Namespace: "system-config",
			Locale:    i18n.LocaleENUS,
			Messages: containerConfigMessages(map[string]string{
				containerConfigDomainKey:       "Operations",
				containerConfigGeneralGroupKey: "Container Management",
				containerConfigGeneralDescKey:  "Control the container management baseline.",
				containerConfigRuntimeGroupKey: "Runtime",
				containerConfigRuntimeDescKey:  "Control the local container runtime adapter.",
				containerConfigLogsGroupKey:    "Logs",
				containerConfigLogsDescKey:     "Control bounded container log reads.",
				containerConfigActionsGroupKey: "Dangerous Actions",
				containerConfigActionsDescKey:  "Control container start, stop, and restart actions.",
			}, enUSContainerConfigCopy(), enUSContainerEnvironmentPolicyEnumCopy()),
		},
	}
}

func containerConfigMessages(prefix map[string]string, definitions map[string][2]string, enumDefinitions map[string][2]string) []i18n.MessageResource {
	enumMessages := containerEnvironmentPolicyEnumMessages(enumDefinitions)
	messages := make([]i18n.MessageResource, 0, len(prefix)+len(definitions)*2+len(enumMessages))
	for key, text := range prefix {
		messages = append(messages, i18n.MessageResource{Key: i18n.MessageKey(key), Text: text})
	}
	for key, copy := range definitions {
		messages = append(messages,
			i18n.MessageResource{Key: i18n.MessageKey(containerConfigTitleKey(key)), Text: copy[0]},
			i18n.MessageResource{Key: i18n.MessageKey(containerConfigDescriptionKey(key)), Text: copy[1]},
		)
	}
	messages = append(messages, enumMessages...)
	return messages
}

func containerEnvironmentPolicyEnumMessages(definitions map[string][2]string) []i18n.MessageResource {
	const messagesPerEnvironmentPolicyEnum = 2

	key := containercontract.ContainerEnvironmentPolicyConfig.String()
	messages := make([]i18n.MessageResource, 0, len(definitions)*messagesPerEnvironmentPolicyEnum)
	for value, row := range definitions {
		messages = append(messages,
			i18n.MessageResource{Key: i18n.MessageKey(fmt.Sprintf("systemConfig.container.%s.enum.%s.label", key, value)), Text: row[0]},
			i18n.MessageResource{Key: i18n.MessageKey(fmt.Sprintf("systemConfig.container.%s.enum.%s.description", key, value)), Text: row[1]},
		)
	}
	return messages
}

func zhCNContainerEnvironmentPolicyEnumCopy() map[string][2]string {
	return map[string][2]string{
		containercontract.ContainerEnvironmentPolicyHidden.String(): {"隐藏", "仅返回环境变量名称和敏感标记。"},
		containercontract.ContainerEnvironmentPolicyMasked.String(): {"脱敏", "敏感环境变量值脱敏，非敏感值正常返回。"},
		containercontract.ContainerEnvironmentPolicyPlain.String():  {"明文", "不按策略脱敏，直接返回环境变量值。"},
	}
}

func enUSContainerEnvironmentPolicyEnumCopy() map[string][2]string {
	return map[string][2]string{
		containercontract.ContainerEnvironmentPolicyHidden.String(): {"Hidden", "Return only environment variable names and sensitivity metadata."},
		containercontract.ContainerEnvironmentPolicyMasked.String(): {"Masked", "Mask sensitive environment variable values and show non-sensitive values."},
		containercontract.ContainerEnvironmentPolicyPlain.String():  {"Plain", "Return environment variable values without policy masking."},
	}
}

func zhCNContainerConfigCopy() map[string][2]string {
	return map[string][2]string{
		containercontract.ContainerRuntimeEnabledConfig.String():          {"启用容器运行时访问", "是否允许容器管理访问已配置的容器运行时。"},
		containercontract.ContainerRuntimeConfig.String():                 {"容器运行时", "容器管理使用的运行时适配器。"},
		containercontract.ContainerDockerEndpointConfig.String():          {"容器运行时 endpoint", "首个本地容器运行时适配器使用的 endpoint。"},
		containercontract.ContainerLogsDefaultTailConfig.String():         {"默认日志行数", "容器日志读取的默认返回行数。"},
		containercontract.ContainerLogsMaxTailConfig.String():             {"最大日志行数", "容器日志读取允许的最大返回行数。"},
		containercontract.ContainerDangerousActionsEnabledConfig.String(): {"启用容器高危操作", "是否允许容器启动、停止和重启等高危操作。"},
		containercontract.ContainerEnvironmentPolicyConfig.String():       {"环境变量值展示策略", "控制容器详情读取时如何返回环境变量值。"},
	}
}

func enUSContainerConfigCopy() map[string][2]string {
	return map[string][2]string{
		containercontract.ContainerRuntimeEnabledConfig.String():          {"Container Runtime Access Enabled", "Whether container management may access the configured runtime."},
		containercontract.ContainerRuntimeConfig.String():                 {"Container Runtime", "Runtime adapter used by container management."},
		containercontract.ContainerDockerEndpointConfig.String():          {"Container Runtime Endpoint", "Local runtime endpoint used by the first container adapter."},
		containercontract.ContainerLogsDefaultTailConfig.String():         {"Default Log Tail", "Default number of log lines returned by container log reads."},
		containercontract.ContainerLogsMaxTailConfig.String():             {"Maximum Log Tail", "Maximum number of log lines allowed for container log reads."},
		containercontract.ContainerDangerousActionsEnabledConfig.String(): {"Dangerous Container Actions Enabled", "Whether start, stop, restart, and future high-risk actions are enabled."},
		containercontract.ContainerEnvironmentPolicyConfig.String():       {"Environment Value Display Policy", "Controls how container environment variable values are returned by detail reads."},
	}
}
