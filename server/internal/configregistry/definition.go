// Package configregistry 管理模块注册的系统配置定义，并保持配置元数据的稳定注册边界。
package configregistry

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"

	"graft/server/internal/scheduler"
)

const maskedPlaceholder = "******"

var keyPattern = regexp.MustCompile(`^[a-z][a-z0-9]*(?:[._-][a-z0-9]+)*$`)

// ValueType 标识配置定义允许的 JSON 值形状。
type ValueType string

const (
	// ValueTypeString 接受 JSON 字符串值。
	ValueTypeString ValueType = "string"
	// ValueTypeNumber 接受 JSON 数字值。
	ValueTypeNumber ValueType = "number"
	// ValueTypeInteger 接受 JSON 整数值。
	ValueTypeInteger ValueType = "integer"
	// ValueTypeBoolean 接受 JSON 布尔值。
	ValueTypeBoolean ValueType = "boolean"
	// ValueTypeObject 接受 JSON 对象值。
	ValueTypeObject ValueType = "object"
	// ValueTypeArray 接受 JSON 数组值。
	ValueTypeArray ValueType = "array"
)

// RuntimeApplyMode 标识配置值变更在运行时生效的方式。
type RuntimeApplyMode string

const (
	// RuntimeApplyModeUnknown 表示当前 authority owner 尚未分类该配置的生效语义。
	RuntimeApplyModeUnknown RuntimeApplyMode = "unknown"
	// RuntimeApplyModeRuntimeHot 表示下一次运行时读取即可观察新值，无需重启。
	RuntimeApplyModeRuntimeHot RuntimeApplyMode = "runtime_hot"
	// RuntimeApplyModeRestartRequired 表示持久化值立即变化，但运行时行为要到重启后才变化。
	RuntimeApplyModeRestartRequired RuntimeApplyMode = "restart_required"
)

// Definition 声明一个模块拥有的系统配置键；模块在 Register 阶段注册它，
// 它是配置元数据真相，不应复制到 system_config_values 作为数据库真相。
type Definition struct {
	Key                 string
	Module              string
	Domain              string
	DomainKey           string
	DomainLabel         string
	Group               string
	GroupKey            string
	GroupLabel          string
	GroupDescription    string
	GroupDescriptionKey string
	Title               string
	TitleKey            string
	Description         string
	DescriptionKey      string
	Tags                []string
	Type                ValueType
	Schema              json.RawMessage
	DefaultValue        json.RawMessage
	Sensitive           bool
	Required            bool
	RestartRequired     bool
	RuntimeApplyMode    RuntimeApplyMode
	Permission          string
	Order               int
}

// Snapshot 返回可安全长期持有的副本；可变 JSON 与标签切片会被复制。
func (d Definition) Snapshot() Definition {
	cloned := d
	cloned.Schema = cloneRawMessage(d.Schema)
	cloned.DefaultValue = cloneRawMessage(d.DefaultValue)
	cloned.Tags = slices.Clone(d.Tags)
	return cloned
}

// MaskedPlaceholder 返回敏感配置值使用的规范脱敏占位符。
func MaskedPlaceholder() string {
	return maskedPlaceholder
}

// validateDefinition 验证配置键、必需元数据、值类型、运行时应用模式、Schema 和默认值。
func validateDefinition(definition Definition) error {
	key := strings.TrimSpace(definition.Key)
	if key == "" {
		return errors.New("config definition key is required")
	}
	if !keyPattern.MatchString(key) {
		return fmt.Errorf("config definition key %q is invalid", definition.Key)
	}
	if err := validateRequiredDefinitionMetadata(definition, key); err != nil {
		return err
	}
	if !slices.Contains(validValueTypes(), definition.Type) {
		return fmt.Errorf("config definition %s type %q is invalid", key, definition.Type)
	}
	if !slices.Contains(validRuntimeApplyModes(), definition.RuntimeApplyMode) {
		return fmt.Errorf("config definition %s runtime apply mode %q is invalid", key, definition.RuntimeApplyMode)
	}
	if err := validateJSONObject(definition.Schema, "schema", key); err != nil {
		return err
	}
	if err := validateDefaultValue(definition.DefaultValue, definition.Type, definition.Schema, key); err != nil {
		return err
	}
	return nil
}

// validateRequiredDefinitionMetadata 验证定义体的必需元数据字段。
// 它检查 Module、Domain 和 Group 非空，且 Title 或 TitleKey 至少有一个提供。
func validateRequiredDefinitionMetadata(definition Definition, key string) error {
	if strings.TrimSpace(definition.Module) == "" {
		return fmt.Errorf("config definition %s module is required", key)
	}
	if strings.TrimSpace(definition.Domain) == "" {
		return fmt.Errorf("config definition %s domain is required", key)
	}
	if strings.TrimSpace(definition.Group) == "" {
		return fmt.Errorf("config definition %s group is required", key)
	}
	if strings.TrimSpace(definition.Title) == "" && strings.TrimSpace(definition.TitleKey) == "" {
		return fmt.Errorf("config definition %s title or title key is required", key)
	}
	return nil
}

// validValueTypes 返回所有允许的 ValueType 值。
func validValueTypes() []ValueType {
	return []ValueType{
		ValueTypeString,
		ValueTypeNumber,
		ValueTypeInteger,
		ValueTypeBoolean,
		ValueTypeObject,
		ValueTypeArray,
	}
}

// validRuntimeApplyModes 返回所有有效的 RuntimeApplyMode 取值。
func validRuntimeApplyModes() []RuntimeApplyMode {
	return []RuntimeApplyMode{
		RuntimeApplyModeUnknown,
		RuntimeApplyModeRuntimeHot,
		RuntimeApplyModeRestartRequired,
	}
}

// validateJSONObject 验证 raw 为空或表示 JSON 对象；数组、标量和非法 JSON 均返回错误。
func validateJSONObject(raw json.RawMessage, label string, key string) error {
	if len(raw) == 0 {
		return nil
	}
	var decoded any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return fmt.Errorf("config definition %s %s is invalid JSON: %w", key, label, err)
	}
	if _, ok := decoded.(map[string]any); !ok {
		return fmt.Errorf("config definition %s %s must be a JSON object", key, label)
	}
	return nil
}

func validateDefaultValue(raw json.RawMessage, valueType ValueType, schema json.RawMessage, key string) error {
	if len(raw) == 0 {
		return fmt.Errorf("config definition %s default value is required", key)
	}
	var decoded any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return fmt.Errorf("config definition %s default value is invalid JSON: %w", key, err)
	}

	if expected := InvalidJSONShape(decoded, valueType); expected != "" {
		return fmt.Errorf("config definition %s default value must be %s", key, expected)
	}
	if len(schema) > 0 {
		if err := validateValueSchema(valueType, schema, raw); err != nil {
			return fmt.Errorf("config definition %s default value does not match schema: %w", key, err)
		}
	}
	return nil
}

func validateValueSchema(valueType ValueType, schema json.RawMessage, value json.RawMessage) error {
	switch valueType {
	case ValueTypeObject:
		return scheduler.ValidateConfigJSON(string(schema), string(value))
	case ValueTypeString, ValueTypeNumber, ValueTypeInteger, ValueTypeBoolean:
		return scheduler.ValidateScalarConfigJSON(string(schema), string(value), string(valueType))
	default:
		return nil
	}
}

// InvalidJSONShape 在值与定义类型不匹配时返回期望的 JSON 形状名称。
func InvalidJSONShape(value any, valueType ValueType) string {
	switch valueType {
	case ValueTypeString:
		return invalidShapeUnless(isJSONString(value), "string")
	case ValueTypeNumber:
		return invalidShapeUnless(isJSONNumber(value), "number")
	case ValueTypeInteger:
		return invalidShapeUnless(isJSONInteger(value), "integer")
	case ValueTypeBoolean:
		return invalidShapeUnless(isJSONBoolean(value), "boolean")
	case ValueTypeObject:
		return invalidShapeUnless(isJSONObject(value), "object")
	case ValueTypeArray:
		return invalidShapeUnless(isJSONArray(value), "array")
	default:
		return "supported JSON value"
	}
}

func invalidShapeUnless(valid bool, expected string) string {
	if valid {
		return ""
	}
	return expected
}

func isJSONString(value any) bool {
	_, ok := value.(string)
	return ok
}

func isJSONNumber(value any) bool {
	_, ok := value.(float64)
	return ok
}

func isJSONInteger(value any) bool {
	number, ok := value.(float64)
	return ok && number == float64(int64(number))
}

func isJSONBoolean(value any) bool {
	_, ok := value.(bool)
	return ok
}

func isJSONObject(value any) bool {
	_, ok := value.(map[string]any)
	return ok
}

func isJSONArray(value any) bool {
	_, ok := value.([]any)
	return ok
}

func cloneRawMessage(raw json.RawMessage) json.RawMessage {
	if len(raw) == 0 {
		return nil
	}
	cloned := make(json.RawMessage, len(raw))
	copy(cloned, raw)
	return cloned
}
