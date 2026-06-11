// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package configregistry

import (
	"encoding/json"
	"testing"
)

func TestRegistryRegisterAndGetDefinition(t *testing.T) {
	registry := NewRegistry()
	definition := testDefinition("scheduler.default_timeout")

	if err := registry.Register(definition); err != nil {
		t.Fatalf("register definition: %v", err)
	}

	got, ok := registry.Get("scheduler.default_timeout")
	if !ok {
		t.Fatal("definition was not registered")
	}
	if got.Key != definition.Key || got.Module != definition.Module || got.Sensitive {
		t.Fatalf("unexpected definition snapshot: %#v", got)
	}
	assertLocalizedMetadata(t, got)

	got.DefaultValue[0] = '{'
	got.Tags[0] = "mutated"
	again, _ := registry.Get("scheduler.default_timeout")
	if string(again.DefaultValue) != `"30s"` {
		t.Fatalf("registry leaked mutable default value: %s", string(again.DefaultValue))
	}
	if again.Tags[0] != "retention" {
		t.Fatalf("registry leaked mutable tags: %#v", again.Tags)
	}
}

func assertLocalizedMetadata(t *testing.T, got Definition) {
	t.Helper()

	if got.DomainKey != "test.domain" || got.DomainLabel != "Test Domain" {
		t.Fatalf("expected localized domain metadata, got %#v", got)
	}
	if got.GroupKey != "test.group" || got.GroupLabel != "Test Group" {
		t.Fatalf("expected localized group metadata, got %#v", got)
	}
	if got.GroupDescriptionKey != "test.group.description" || got.GroupDescription != "Test group description" {
		t.Fatalf("expected localized group description metadata, got %#v", got)
	}
	if len(got.Tags) != 1 || got.Tags[0] != "retention" {
		t.Fatalf("expected normalized tags, got %#v", got.Tags)
	}
}

func TestRegistryRejectsDuplicateDefinition(t *testing.T) {
	registry := NewRegistry()
	if err := registry.Register(testDefinition("audit.retention_days")); err != nil {
		t.Fatalf("register first definition: %v", err)
	}
	if err := registry.Register(testDefinition("audit.retention_days")); err == nil {
		t.Fatal("expected duplicate definition error")
	}
}

func TestRegistryValidatesDefaultValueShape(t *testing.T) {
	definition := testDefinition("auth.password_policy")
	definition.Type = ValueTypeInteger
	definition.DefaultValue = json.RawMessage(`"thirty"`)

	if err := NewRegistry().Register(definition); err == nil {
		t.Fatal("expected invalid default value error")
	}
}

func TestRegistryValidatesScalarDefaultValueAgainstSchema(t *testing.T) {
	testCases := []struct {
		name         string
		key          string
		valueType    ValueType
		schema       json.RawMessage
		defaultValue json.RawMessage
	}{
		{
			name:         "integer range",
			key:          "dashboard.quick_actions.max_items",
			valueType:    ValueTypeInteger,
			schema:       json.RawMessage(`{"type":"integer","minimum":1,"maximum":24}`),
			defaultValue: json.RawMessage(`0`),
		},
		{
			name:         "string enum",
			key:          "dashboard.quick_actions.strategy",
			valueType:    ValueTypeString,
			schema:       json.RawMessage(`{"type":"string","enum":["hybrid","recent"]}`),
			defaultValue: json.RawMessage(`"unknown"`),
		},
		{
			name:         "string length",
			key:          "auth.password_policy",
			valueType:    ValueTypeString,
			schema:       json.RawMessage(`{"type":"string","minLength":3,"maxLength":8}`),
			defaultValue: json.RawMessage(`"xy"`),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			definition := testDefinition(tc.key)
			definition.Type = tc.valueType
			definition.Schema = tc.schema
			definition.DefaultValue = tc.defaultValue

			if err := NewRegistry().Register(definition); err == nil {
				t.Fatal("expected invalid scalar schema default value error")
			}
		})
	}
}

func TestRegistryRequiresDomain(t *testing.T) {
	definition := testDefinition("auth.password_policy")
	definition.Domain = ""

	if err := NewRegistry().Register(definition); err == nil {
		t.Fatal("expected missing domain error")
	}
}

func TestMaskedPlaceholder(t *testing.T) {
	if MaskedPlaceholder() == "" {
		t.Fatal("masked placeholder must be stable and non-empty")
	}
}

func testDefinition(key string) Definition {
	return Definition{
		Key:                 key,
		Module:              "test",
		Domain:              " test ",
		DomainKey:           " test.domain ",
		DomainLabel:         " Test Domain ",
		Group:               "test",
		GroupKey:            " test.group ",
		GroupLabel:          " Test Group ",
		GroupDescription:    " Test group description ",
		GroupDescriptionKey: " test.group.description ",
		Title:               "Test",
		Tags:                []string{" retention ", ""},
		Type:                ValueTypeString,
		Schema:              json.RawMessage(`{"type":"string"}`),
		DefaultValue:        json.RawMessage(`"30s"`),
		Order:               10,
	}
}
