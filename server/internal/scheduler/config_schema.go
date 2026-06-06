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
	Type           string   `json:"type"`
	Title          string   `json:"title"`
	Description    string   `json:"description"`
	TitleKey       string   `json:"x-title-key"`
	DescriptionKey string   `json:"x-description-key"`
	Enum           []any    `json:"enum"`
	Minimum        *float64 `json:"minimum"`
	Maximum        *float64 `json:"maximum"`
	MinLength      *int     `json:"minLength"`
	MaxLength      *int     `json:"maxLength"`
}

// ConfigValidationError carries the field path that should be returned in the
// existing error envelope data.field slot.
type ConfigValidationError struct {
	Field  string
	Reason string
}

func (e ConfigValidationError) Error() string {
	if e.Reason == "" {
		return e.Field
	}
	return fmt.Sprintf("%s: %s", e.Field, e.Reason)
}

// ValidateConfigSchema validates the scheduler-owned JSON Schema subset.
func ValidateConfigSchema(schemaJSON string) error {
	schema, err := decodeConfigSchema(schemaJSON)
	if err != nil {
		return err
	}
	if schema.Type != "" && schema.Type != "object" {
		return ConfigValidationError{Field: "config_schema.type", Reason: "must be object"}
	}
	for name, property := range schema.Properties {
		field := "config_schema.properties." + name
		if !validConfigPropertyType(property.Type) {
			return ConfigValidationError{Field: field + ".type", Reason: "unsupported type"}
		}
		if property.MinLength != nil && *property.MinLength < 0 {
			return ConfigValidationError{Field: field + ".minLength", Reason: "must be non-negative"}
		}
		if property.MaxLength != nil && *property.MaxLength < 0 {
			return ConfigValidationError{Field: field + ".maxLength", Reason: "must be non-negative"}
		}
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

func decodeConfigSchema(schemaJSON string) (configSchema, error) {
	var schema configSchema
	if err := json.Unmarshal([]byte(defaultJSONObject(schemaJSON)), &schema); err != nil {
		return configSchema{}, ConfigValidationError{Field: "config_schema", Reason: "must be a JSON object"}
	}
	if schema.Properties == nil {
		schema.Properties = map[string]configPropertySchema{}
	}
	return schema, nil
}

func decodeConfigObject(configJSON string) (map[string]any, error) {
	var config map[string]any
	if err := json.Unmarshal([]byte(defaultJSONObject(configJSON)), &config); err != nil {
		return nil, ConfigValidationError{Field: "config_json", Reason: "must be a JSON object"}
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
			return ConfigValidationError{Field: "config_json." + name, Reason: "is required"}
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
			return ConfigValidationError{Field: "config_json." + name, Reason: "is not allowed"}
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
		return ConfigValidationError{Field: field, Reason: "uses unsupported schema type"}
	}
	if err != nil {
		return err
	}
	if len(property.Enum) > 0 && !enumContains(property.Enum, value) {
		return ConfigValidationError{Field: field, Reason: "must match enum"}
	}
	return nil
}

func validateStringConfigValue(field string, property configPropertySchema, value any) error {
	text, ok := value.(string)
	if !ok {
		return ConfigValidationError{Field: field, Reason: "must be string"}
	}
	if property.MinLength != nil && len(text) < *property.MinLength {
		return ConfigValidationError{Field: field, Reason: "is too short"}
	}
	if property.MaxLength != nil && len(text) > *property.MaxLength {
		return ConfigValidationError{Field: field, Reason: "is too long"}
	}
	return nil
}

func validateIntegerConfigValue(field string, property configPropertySchema, value any) error {
	number, ok := value.(float64)
	if !ok || math.Trunc(number) != number {
		return ConfigValidationError{Field: field, Reason: "must be integer"}
	}
	return validateNumberRange(field, property, number)
}

func validateNumberConfigValue(field string, property configPropertySchema, value any) error {
	number, ok := value.(float64)
	if !ok {
		return ConfigValidationError{Field: field, Reason: "must be number"}
	}
	return validateNumberRange(field, property, number)
}

func validateBooleanConfigValue(field string, value any) error {
	if _, ok := value.(bool); !ok {
		return ConfigValidationError{Field: field, Reason: "must be boolean"}
	}
	return nil
}

func validateNumberRange(field string, property configPropertySchema, value float64) error {
	if property.Minimum != nil && value < *property.Minimum {
		return ConfigValidationError{Field: field, Reason: "is below minimum"}
	}
	if property.Maximum != nil && value > *property.Maximum {
		return ConfigValidationError{Field: field, Reason: "is above maximum"}
	}
	return nil
}

func enumContains(values []any, candidate any) bool {
	for _, value := range values {
		if fmt.Sprint(value) == fmt.Sprint(candidate) {
			return true
		}
	}
	return false
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
