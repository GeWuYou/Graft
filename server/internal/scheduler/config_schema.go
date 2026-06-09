// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package scheduler

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"
)

type configSchema struct {
	Type                 string                          `json:"type"`
	Properties           map[string]configPropertySchema `json:"properties"`
	Required             []string                        `json:"required"`
	AdditionalProperties bool                            `json:"additionalProperties"`
}

type configPropertySchema struct {
	Type        string   `json:"type"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Enum        []any    `json:"enum"`
	Minimum     *float64 `json:"minimum"`
	Maximum     *float64 `json:"maximum"`
	MinLength   *int     `json:"minLength"`
	MaxLength   *int     `json:"maxLength"`
}

// ConfigValidationError carries the field path that should be returned in the
// existing error envelope data.field slot.
type ConfigValidationError struct {
	Field      string
	Reason     string
	ReasonCode string
	Constraint string
	Minimum    any
	Maximum    any
	Expected   any
	Actual     any
}

func (e ConfigValidationError) Error() string {
	if e.Reason == "" {
		return e.Field
	}
	return fmt.Sprintf("%s: %s", e.Field, e.Reason)
}

// Details returns the stable structured error payload for HTTP response data.
func (e ConfigValidationError) Details() map[string]any {
	details := map[string]any{
		"field":  e.Field,
		"reason": e.Reason,
	}
	if e.ReasonCode != "" {
		details["reason_code"] = e.ReasonCode
	}
	if e.Constraint != "" {
		details["constraint"] = e.Constraint
	}
	if e.Minimum != nil {
		details["minimum"] = e.Minimum
	}
	if e.Maximum != nil {
		details["maximum"] = e.Maximum
	}
	if e.Expected != nil {
		details["expected"] = e.Expected
	}
	if e.Actual != nil {
		details["actual"] = e.Actual
	}
	return details
}

// ValidateConfigSchema validates the scheduler-owned JSON Schema subset.
func ValidateConfigSchema(schemaJSON string) error {
	schema, err := decodeConfigSchema(schemaJSON)
	if err != nil {
		return err
	}
	if schema.Type != "" && schema.Type != "object" {
		return configTypeError("config_schema.type", "object", schema.Type)
	}
	for name, property := range schema.Properties {
		field := "config_schema.properties." + name
		if err := validateConfigPropertySchema(field, property); err != nil {
			return err
		}
	}
	return nil
}

func validateConfigPropertySchema(field string, property configPropertySchema) error {
	if !validConfigPropertyType(property.Type) {
		return ConfigValidationError{
			Field:      field + ".type",
			Reason:     "unsupported type",
			ReasonCode: "unsupported_type",
			Constraint: "type",
			Actual:     property.Type,
		}
	}
	if err := validateConfigPropertyLengthSchema(field, property); err != nil {
		return err
	}
	return validateConfigPropertyRangeSchema(field, property)
}

func validateConfigPropertyLengthSchema(field string, property configPropertySchema) error {
	if property.MinLength != nil && *property.MinLength < 0 {
		return ConfigValidationError{Field: field + ".minLength", Reason: "must be non-negative", ReasonCode: "below_minimum", Constraint: "minimum", Minimum: 0, Actual: *property.MinLength}
	}
	if property.MaxLength != nil && *property.MaxLength < 0 {
		return ConfigValidationError{Field: field + ".maxLength", Reason: "must be non-negative", ReasonCode: "below_minimum", Constraint: "minimum", Minimum: 0, Actual: *property.MaxLength}
	}
	if property.MinLength != nil && property.MaxLength != nil && *property.MinLength > *property.MaxLength {
		return ConfigValidationError{Field: field + ".minLength", Reason: "must be less than or equal to maxLength", ReasonCode: "above_maximum", Constraint: "maxLength", Maximum: *property.MaxLength, Actual: *property.MinLength}
	}
	return nil
}

func validateConfigPropertyRangeSchema(field string, property configPropertySchema) error {
	if property.Minimum != nil && property.Maximum != nil && *property.Minimum > *property.Maximum {
		return ConfigValidationError{Field: field + ".minimum", Reason: "must be less than or equal to maximum", ReasonCode: "above_maximum", Constraint: "maximum", Maximum: *property.Maximum, Actual: *property.Minimum}
	}
	return nil
}

// ValidateConfigJSON validates an effective_config JSON object against the
// scheduler MVP flat-object JSON Schema subset.
func ValidateConfigJSON(schemaJSON string, configJSON string) error {
	schema, err := decodeConfigSchema(schemaJSON)
	if err != nil {
		return err
	}
	config, err := decodeConfigObject(configJSON)
	if err != nil {
		return err
	}
	return validateConfigObject(schema, config)
}

func sanitizeConfigJSON(schemaJSON string, configJSON string) (string, error) {
	schema, err := decodeConfigSchema(schemaJSON)
	if err != nil {
		return "", err
	}
	config, err := decodeConfigObject(configJSON)
	if err != nil {
		return "", err
	}
	sanitized := make(map[string]any, len(config))
	for name, value := range config {
		property, ok := schema.Properties[name]
		if !ok {
			if schema.AdditionalProperties {
				sanitized[name] = value
			}
			continue
		}
		if err := validateConfigValue("config_json."+name, property, value); err != nil {
			continue
		}
		sanitized[name] = value
	}
	encoded, err := json.Marshal(sanitized)
	if err != nil {
		return "", ConfigValidationError{Field: "config_json", Reason: "must be a JSON object", ReasonCode: "invalid_json", Constraint: "type", Expected: "object"}
	}
	return string(encoded), nil
}

func decodeConfigSchema(schemaJSON string) (configSchema, error) {
	var schema configSchema
	if err := json.Unmarshal([]byte(defaultJSONObject(schemaJSON)), &schema); err != nil {
		return configSchema{}, ConfigValidationError{Field: "config_schema", Reason: "must be a JSON object", ReasonCode: "invalid_json", Constraint: "type", Expected: "object"}
	}
	if schema.Properties == nil {
		schema.Properties = map[string]configPropertySchema{}
	}
	return schema, nil
}

func decodeConfigObject(configJSON string) (map[string]any, error) {
	var config map[string]any
	if err := json.Unmarshal([]byte(defaultJSONObject(configJSON)), &config); err != nil {
		return nil, ConfigValidationError{Field: "config_json", Reason: "must be a JSON object", ReasonCode: "invalid_json", Constraint: "type", Expected: "object"}
	}
	return config, nil
}

func validateConfigObject(schema configSchema, config map[string]any) error {
	if err := validateRequiredConfigFields(schema, config); err != nil {
		return err
	}
	if err := validateAdditionalConfigFields(schema, config); err != nil {
		return err
	}
	return validateConfigProperties(schema, config)
}

func validateRequiredConfigFields(schema configSchema, config map[string]any) error {
	for _, name := range schema.Required {
		if _, ok := config[name]; !ok {
			return ConfigValidationError{Field: "config_json." + name, Reason: "is required", ReasonCode: "required", Constraint: "required"}
		}
	}
	return nil
}

func validateAdditionalConfigFields(schema configSchema, config map[string]any) error {
	if schema.AdditionalProperties {
		return nil
	}
	for name := range config {
		if _, ok := schema.Properties[name]; !ok {
			return ConfigValidationError{Field: "config_json." + name, Reason: "is not allowed", ReasonCode: "additional_property", Constraint: "additionalProperties", Actual: name}
		}
	}
	return nil
}

func validateConfigProperties(schema configSchema, config map[string]any) error {
	for name, property := range schema.Properties {
		value, ok := config[name]
		if !ok {
			continue
		}
		if err := validateConfigValue("config_json."+name, property, value); err != nil {
			return err
		}
	}
	return nil
}

func validateConfigValue(field string, property configPropertySchema, value any) error {
	var err error
	switch property.Type {
	case "string":
		err = validateStringConfigValue(field, property, value)
	case "integer":
		err = validateIntegerConfigValue(field, property, value)
	case "number":
		err = validateNumberConfigValue(field, property, value)
	case "boolean":
		err = validateBooleanConfigValue(field, value)
	case "":
		return nil
	default:
		return ConfigValidationError{Field: field, Reason: "uses unsupported schema type", ReasonCode: "unsupported_type", Constraint: "type", Actual: property.Type}
	}
	if err != nil {
		return err
	}
	if len(property.Enum) > 0 && !enumContains(property.Type, property.Enum, value) {
		return ConfigValidationError{Field: field, Reason: "must match enum", ReasonCode: "enum", Constraint: "enum", Expected: property.Enum, Actual: value}
	}
	return nil
}

func validateStringConfigValue(field string, property configPropertySchema, value any) error {
	text, ok := value.(string)
	if !ok {
		return configTypeError(field, "string", value)
	}
	if property.MinLength != nil && len(text) < *property.MinLength {
		return ConfigValidationError{Field: field, Reason: "is too short", ReasonCode: "too_short", Constraint: "minLength", Minimum: *property.MinLength, Actual: len(text)}
	}
	if property.MaxLength != nil && len(text) > *property.MaxLength {
		return ConfigValidationError{Field: field, Reason: "is too long", ReasonCode: "too_long", Constraint: "maxLength", Maximum: *property.MaxLength, Actual: len(text)}
	}
	return nil
}

func validateIntegerConfigValue(field string, property configPropertySchema, value any) error {
	number, ok := value.(float64)
	if !ok || math.Trunc(number) != number {
		return configTypeError(field, "integer", value)
	}
	return validateNumberRange(field, property, number)
}

func validateNumberConfigValue(field string, property configPropertySchema, value any) error {
	number, ok := value.(float64)
	if !ok {
		return configTypeError(field, "number", value)
	}
	return validateNumberRange(field, property, number)
}

func validateBooleanConfigValue(field string, value any) error {
	if _, ok := value.(bool); !ok {
		return configTypeError(field, "boolean", value)
	}
	return nil
}

func validateNumberRange(field string, property configPropertySchema, value float64) error {
	if property.Minimum != nil && value < *property.Minimum {
		return ConfigValidationError{Field: field, Reason: "is below minimum", ReasonCode: "below_minimum", Constraint: "minimum", Minimum: numberDetailValue(*property.Minimum), Actual: numberDetailValue(value)}
	}
	if property.Maximum != nil && value > *property.Maximum {
		return ConfigValidationError{Field: field, Reason: "is above maximum", ReasonCode: "above_maximum", Constraint: "maximum", Maximum: numberDetailValue(*property.Maximum), Actual: numberDetailValue(value)}
	}
	return nil
}

func configTypeError(field string, expected string, actual any) ConfigValidationError {
	return ConfigValidationError{
		Field:      field,
		Reason:     "must be " + expected,
		ReasonCode: "type_mismatch",
		Constraint: "type",
		Expected:   expected,
		Actual:     actual,
	}
}

func numberDetailValue(value float64) any {
	if math.Trunc(value) == value {
		return int64(value)
	}
	return value
}

func enumContains(propertyType string, values []any, candidate any) bool {
	for _, value := range values {
		if enumValueMatches(propertyType, value, candidate) {
			return true
		}
	}
	return false
}

func enumValueMatches(propertyType string, value any, candidate any) bool {
	switch propertyType {
	case "string":
		return stringEnumValueMatches(value, candidate)
	case "integer":
		return numericEnumValueMatches(value, candidate, true)
	case "number":
		return numericEnumValueMatches(value, candidate, false)
	case "boolean":
		return boolEnumValueMatches(value, candidate)
	default:
		return fmt.Sprint(value) == fmt.Sprint(candidate)
	}
}

func stringEnumValueMatches(value any, candidate any) bool {
	expected, ok := value.(string)
	actual, actualOK := candidate.(string)
	return ok && actualOK && expected == actual
}

func numericEnumValueMatches(value any, candidate any, requireInteger bool) bool {
	expected, ok := value.(float64)
	actual, actualOK := candidate.(float64)
	if !ok || !actualOK {
		return false
	}
	if requireInteger && (math.Trunc(expected) != expected || math.Trunc(actual) != actual) {
		return false
	}
	return expected == actual
}

func boolEnumValueMatches(value any, candidate any) bool {
	expected, ok := value.(bool)
	actual, actualOK := candidate.(bool)
	return ok && actualOK && expected == actual
}

func validConfigPropertyType(value string) bool {
	switch strings.TrimSpace(value) {
	case "", "string", "integer", "number", "boolean":
		return true
	default:
		return false
	}
}

func defaultJSONObject(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "{}"
	}
	return trimmed
}
