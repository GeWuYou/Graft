package network

import (
	"encoding/json"
	"fmt"

	"graft/server/internal/configregistry"
)

const outboundConfigOrder = 125

func registerOutboundConfig(registry *configregistry.Registry) error {
	if registry == nil {
		return fmt.Errorf("config registry is unavailable")
	}
	return registry.Register(configregistry.Definition{
		Key:              outboundConfigKey,
		Module:           moduleID,
		Domain:           "network",
		DomainKey:        "systemConfig.domains.network",
		DomainLabel:      "Network",
		Group:            "network.outbound",
		GroupKey:         "systemConfig.groups.network.outbound",
		GroupLabel:       "Outbound Network",
		Title:            "Outbound network policy",
		TitleKey:         "systemConfig.network.network.outbound.title",
		Description:      "Platform proxy and no-proxy policy for outbound HTTP(S) requests.",
		DescriptionKey:   "systemConfig.network.network.outbound.description",
		Tags:             []string{"network", "outbound"},
		Type:             configregistry.ValueTypeObject,
		Schema:           outboundConfigSchema(),
		DefaultValue:     json.RawMessage(`{"enabled":false,"http_proxy":"","https_proxy":"","no_proxy":[],"authentication":null}`),
		ModuleManaged:    true,
		RuntimeApplyMode: configregistry.RuntimeApplyModeRuntimeHot,
		Order:            outboundConfigOrder,
	})
}

func outboundConfigSchema() json.RawMessage {
	// system-config 当前 JSON Schema 校验器不支持数组和 null，因此字段级校验由 decodeOutboundPolicy 完成。
	return json.RawMessage(`{"type":"object","additionalProperties":true,"properties":{"authentication":{"description":"Reserved for future Secret Management.","x-i18n":{"descriptionKey":"network.outbound.authentication.description"}}},"x-i18n":{"titleKey":"systemConfig.network.network.outbound.title","descriptionKey":"systemConfig.network.network.outbound.description"}}`)
}
