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
	if got.GroupKey != "test.group" || got.GroupLabel != "Test Group" || len(got.Tags) != 1 || got.Tags[0] != "retention" {
		t.Fatalf("expected localized group metadata and normalized tags, got %#v", got)
	}

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

func TestMaskedPlaceholder(t *testing.T) {
	if MaskedPlaceholder() == "" {
		t.Fatal("masked placeholder must be stable and non-empty")
	}
}

func testDefinition(key string) Definition {
	return Definition{
		Key:          key,
		Module:       "test",
		Group:        "test",
		GroupKey:     " test.group ",
		GroupLabel:   " Test Group ",
		Title:        "Test",
		Tags:         []string{" retention ", ""},
		Type:         ValueTypeString,
		Schema:       json.RawMessage(`{"type":"string"}`),
		DefaultValue: json.RawMessage(`"30s"`),
		Order:        10,
	}
}
