package systemconfig

import (
	"encoding/json"
	"math"
	"time"

	"graft/server/internal/configregistry"
	generated "graft/server/internal/contract/openapi/generated"
)

// toListResponse 将提供的 ValueSnapshots 转换成一个包含映射项和总数的 SystemConfigListResponse。
func toListResponse(items []ValueSnapshot) generated.SystemConfigListResponse {
	mapped := make([]generated.SystemConfigItem, 0, len(items))
	for _, item := range items {
		mapped = append(mapped, toItem(item))
	}
	return generated.SystemConfigListResponse{
		Items: mapped,
		Total: len(mapped),
	}
}

// toItem 将 ValueSnapshot 转换为 generated.SystemConfigItem 用于 API 响应，敏感值会被掩盖。
func toItem(snapshot ValueSnapshot) generated.SystemConfigItem {
	definition := snapshot.Definition
	return generated.SystemConfigItem{
		Key:                 definition.Key,
		Module:              definition.Module,
		Domain:              optionalString(definition.Domain),
		DomainKey:           optionalString(definition.DomainKey),
		DomainLabel:         optionalString(definition.DomainLabel),
		Group:               definition.Group,
		GroupKey:            optionalString(definition.GroupKey),
		GroupLabel:          optionalString(definition.GroupLabel),
		GroupDescription:    optionalString(definition.GroupDescription),
		GroupDescriptionKey: optionalString(definition.GroupDescriptionKey),
		Title:               optionalString(definition.Title),
		TitleKey:            optionalString(definition.TitleKey),
		Description:         optionalString(definition.Description),
		DescriptionKey:      optionalString(definition.DescriptionKey),
		Tags:                optionalStrings(definition.Tags),
		Type:                generated.SystemConfigItemType(definition.Type),
		ConfigSchema:        rawJSONMap(definition.Schema),
		Status:              generated.SystemConfigItemStatus(snapshot.Status),
		DefaultValue:        visibleValue(snapshot.DefaultValue, definition.Sensitive),
		EffectiveValue:      visibleValue(snapshot.EffectiveValue, definition.Sensitive),
		OverrideValue:       visibleValue(snapshot.OverrideValue, definition.Sensitive),
		HasOverride:         snapshot.HasOverride,
		UpdatedAt:           optionalTime(snapshot.UpdatedAt),
		UpdatedByUserId:     optionalInt64FromUint64(snapshot.UpdatedBy),
		UpdatedByUsername:   optionalString(snapshot.UpdatedByName),
		Sensitive:           definition.Sensitive,
		Masked:              snapshot.Masked,
		RestartRequired:     definition.RestartRequired,
		RuntimeApplyMode:    generated.SystemConfigItemRuntimeApplyMode(definition.RuntimeApplyMode),
		Permission:          optionalString(definition.Permission),
		Order:               optionalInt(definition.Order),
		MaskedPlaceholder:   maskedPointer(definition),
	}
}

func optionalTime(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	utc := value.UTC()
	return &utc
}

func rawJSONMap(raw json.RawMessage) map[string]interface{} {
	var decoded map[string]interface{}
	if len(raw) == 0 {
		return map[string]interface{}{}
	}
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return map[string]interface{}{}
	}
	return decoded
}

func visibleValue(raw json.RawMessage, sensitive bool) *string {
	if sensitive || len(raw) == 0 {
		return nil
	}
	value := string(raw)
	return &value
}

func optionalString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func optionalInt(value int) *int {
	if value == 0 {
		return nil
	}
	return &value
}

func optionalInt64FromUint64(value *uint64) *int64 {
	if value == nil || *value > uint64(math.MaxInt64) {
		return nil
	}
	converted := int64(*value)
	return &converted
}

func optionalStrings(values []string) *[]string {
	if len(values) == 0 {
		return nil
	}
	cloned := append([]string(nil), values...)
	return &cloned
}

func maskedPointer(definition configregistry.Definition) *string {
	if !definition.Sensitive {
		return nil
	}
	value := configregistry.MaskedPlaceholder()
	return &value
}
