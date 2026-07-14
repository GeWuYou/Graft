package project

import (
	"context"
	"encoding/json"
	"testing"

	"graft/server/internal/configregistry"
)

func TestApplicationRootDirectoryDefinitionRegisters(t *testing.T) {
	registry := configregistry.NewRegistry()

	if err := registry.Register(applicationRootDirectoryDefinition()); err != nil {
		t.Fatalf("register application root directory definition: %v", err)
	}
}

func TestApplicationRootDirectoryDefinitionUsesCanonicalApplicationContract(t *testing.T) {
	definition := applicationRootDirectoryDefinition()
	if definition.Key != "ops.application.root_directory" {
		t.Fatalf("unexpected key: %q", definition.Key)
	}
	if definition.Group != "ops.application.create" {
		t.Fatalf("unexpected group: %q", definition.Group)
	}

	var defaultValue string
	if err := json.Unmarshal(definition.DefaultValue, &defaultValue); err != nil {
		t.Fatalf("decode default value: %v", err)
	}
	if defaultValue != "/opt/graft/apps" {
		t.Fatalf("unexpected default root: %q", defaultValue)
	}
	if !json.Valid(definition.Schema) {
		t.Fatalf("expected valid JSON schema: %s", definition.Schema)
	}
}

func TestManagedRootKeepsExplicitEmptyApplicationRootDisabled(t *testing.T) {
	service, err := NewService(
		&stubProjectRepository{},
		WithSystemConfigResolver(stubCompositeConfigResolver{
			values: map[string]string{"ops.application.root_directory": `""`},
		}),
	)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	info, err := service.ManagedRoot(context.Background())
	if err != nil {
		t.Fatalf("resolve managed root: %v", err)
	}
	if info.SupportsManagedCreate || info.ConfiguredRootDirectory != nil || info.Status != "unconfigured" {
		t.Fatalf("expected explicit empty root to disable managed creation, got %#v", info)
	}
}

func TestProjectConfigDefinitionsPreserveOpsDomainContract(t *testing.T) {
	for _, definition := range configDefinitions() {
		if definition.Domain != "ops" || definition.DomainKey != "systemConfig.domains.ops" {
			t.Fatalf("expected stable ops config domain metadata, got %#v", definition)
		}
	}
}
