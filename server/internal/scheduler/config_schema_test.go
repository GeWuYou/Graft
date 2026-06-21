package scheduler

import (
	"errors"
	"testing"
)

func TestValidateConfigSchemaRejectsInvertedBounds(t *testing.T) {
	testCases := []struct {
		name   string
		schema string
	}{
		{
			name:   "string length bounds",
			schema: `{"type":"object","properties":{"name":{"type":"string","minLength":5,"maxLength":3}}}`,
		},
		{
			name:   "number bounds",
			schema: `{"type":"object","properties":{"count":{"type":"integer","minimum":10,"maximum":2}}}`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if err := ValidateConfigSchema(tc.schema); err == nil {
				t.Fatal("expected invalid schema bounds")
			}
		})
	}
}

func TestValidateConfigJSONEnumKeepsValueTypesStrict(t *testing.T) {
	schema := `{"type":"object","properties":{"mode":{"type":"string","enum":["42"]},"count":{"type":"integer","enum":[42]}}}`
	if err := ValidateConfigJSON(schema, `{"mode":"42","count":42}`); err != nil {
		t.Fatalf("expected matching enum values: %v", err)
	}
	if err := ValidateConfigJSON(schema, `{"mode":42,"count":42}`); err == nil {
		t.Fatal("expected numeric value to fail string enum/type validation")
	}
	if err := ValidateConfigJSON(schema, `{"mode":"42","count":"42"}`); err == nil {
		t.Fatal("expected string value to fail integer enum/type validation")
	}
}

func TestValidateScalarConfigJSONAppliesEnumRangeAndLength(t *testing.T) {
	testCases := []struct {
		name         string
		schema       string
		value        string
		expectedType string
	}{
		{
			name:   "string enum",
			schema: `{"type":"string","enum":["hybrid","recent"]}`,
			value:  `"unknown"`,
		},
		{
			name:   "integer range",
			schema: `{"type":"integer","minimum":1,"maximum":24}`,
			value:  `25`,
		},
		{
			name:   "string length",
			schema: `{"type":"string","minLength":3,"maxLength":5}`,
			value:  `"xy"`,
		},
		{
			name:         "definition type fallback",
			schema:       `{"minimum":1,"maximum":24}`,
			value:        `0`,
			expectedType: "integer",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if err := ValidateScalarConfigJSON(tc.schema, tc.value, tc.expectedType); err == nil {
				t.Fatal("expected scalar schema validation error")
			}
		})
	}
}

func TestValidateConfigJSONReturnsStructuredRangeDetails(t *testing.T) {
	schema := `{"type":"object","properties":{"batchSize":{"type":"integer","minimum":1,"maximum":10000}},"additionalProperties":false}`

	err := ValidateConfigJSON(schema, `{"batchSize":100000}`)

	var configErr ConfigValidationError
	if !errors.As(err, &configErr) {
		t.Fatalf("expected ConfigValidationError, got %T %v", err, err)
	}
	if configErr.Field != "config_json.batchSize" ||
		configErr.ReasonCode != "above_maximum" ||
		configErr.Constraint != "maximum" ||
		configErr.Maximum != int64(10000) ||
		configErr.Actual != int64(100000) {
		t.Fatalf("unexpected validation details: %#v", configErr)
	}
	details := configErr.Details()
	if details["field"] != "config_json.batchSize" ||
		details["reason_code"] != "above_maximum" ||
		details["maximum"] != int64(10000) ||
		details["actual"] != int64(100000) {
		t.Fatalf("unexpected response details: %#v", details)
	}
}

func TestValidateConfigJSONReturnsStructuredTypeEnumAndAdditionalPropertyDetails(t *testing.T) {
	testCases := []struct {
		name       string
		schema     string
		config     string
		field      string
		reasonCode string
		constraint string
	}{
		{
			name:       "type mismatch",
			schema:     `{"type":"object","properties":{"batchSize":{"type":"integer"}},"additionalProperties":false}`,
			config:     `{"batchSize":"100"}`,
			field:      "config_json.batchSize",
			reasonCode: "type_mismatch",
			constraint: "type",
		},
		{
			name:       "enum mismatch",
			schema:     `{"type":"object","properties":{"mode":{"type":"string","enum":["safe"]}},"additionalProperties":false}`,
			config:     `{"mode":"force"}`,
			field:      "config_json.mode",
			reasonCode: "enum",
			constraint: "enum",
		},
		{
			name:       "additional property",
			schema:     `{"type":"object","properties":{"batchSize":{"type":"integer"}},"additionalProperties":false}`,
			config:     `{"batchSize":100,"dryRun":true}`,
			field:      "config_json.dryRun",
			reasonCode: "additional_property",
			constraint: "additionalProperties",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateConfigJSON(tc.schema, tc.config)
			var configErr ConfigValidationError
			if !errors.As(err, &configErr) {
				t.Fatalf("expected ConfigValidationError, got %T %v", err, err)
			}
			if configErr.Field != tc.field || configErr.ReasonCode != tc.reasonCode || configErr.Constraint != tc.constraint {
				t.Fatalf("unexpected validation details: %#v", configErr)
			}
		})
	}
}
