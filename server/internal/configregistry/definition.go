// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

// Package configregistry owns module-registered system configuration definitions.
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

// ValueType identifies the JSON value shape accepted by one config definition.
type ValueType string

const (
	// ValueTypeString accepts JSON string values.
	ValueTypeString ValueType = "string"
	// ValueTypeNumber accepts JSON number values.
	ValueTypeNumber ValueType = "number"
	// ValueTypeInteger accepts JSON integer values.
	ValueTypeInteger ValueType = "integer"
	// ValueTypeBoolean accepts JSON boolean values.
	ValueTypeBoolean ValueType = "boolean"
	// ValueTypeObject accepts JSON object values.
	ValueTypeObject ValueType = "object"
	// ValueTypeArray accepts JSON array values.
	ValueTypeArray ValueType = "array"
)

// RuntimeApplyMode identifies how a changed config value is expected to apply at runtime.
type RuntimeApplyMode string

const (
	// RuntimeApplyModeUnknown means the current authority owners have not classified apply semantics yet.
	RuntimeApplyModeUnknown RuntimeApplyMode = "unknown"
	// RuntimeApplyModeRuntimeHot means the next runtime read should observe the new effective value without restart.
	RuntimeApplyModeRuntimeHot RuntimeApplyMode = "runtime_hot"
	// RuntimeApplyModeRestartRequired means the persisted value changes immediately, but runtime behavior changes only after restart.
	RuntimeApplyModeRestartRequired RuntimeApplyMode = "restart_required"
)

// Definition declares one module-owned system configuration key.
//
// Definitions are registered by modules during Register. They are canonical
// metadata and must not be copied into system_config_values as database truth.
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

// Snapshot returns an immutable copy safe for callers to retain.
func (d Definition) Snapshot() Definition {
	cloned := d
	cloned.Schema = cloneRawMessage(d.Schema)
	cloned.DefaultValue = cloneRawMessage(d.DefaultValue)
	cloned.Tags = slices.Clone(d.Tags)
	return cloned
}

// MaskedPlaceholder returns the canonical masked value sentinel for sensitive values.
func MaskedPlaceholder() string {
	return maskedPlaceholder
}

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

func validRuntimeApplyModes() []RuntimeApplyMode {
	return []RuntimeApplyMode{
		RuntimeApplyModeUnknown,
		RuntimeApplyModeRuntimeHot,
		RuntimeApplyModeRestartRequired,
	}
}

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

// InvalidJSONShape returns the expected shape name when value does not match the definition type.
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
