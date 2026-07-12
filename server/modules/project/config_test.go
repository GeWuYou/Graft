package project

import (
	"testing"

	"graft/server/internal/configregistry"
)

func TestProjectManagedRootDefinitionSchemaRegisters(t *testing.T) {
	registry := configregistry.NewRegistry()

	if err := registry.Register(projectManagedRootDefinition()); err != nil {
		t.Fatalf("register project managed root definition: %v", err)
	}
}

func TestProjectConfigDefinitionsPreserveOpsDomainContract(t *testing.T) {
	for _, definition := range configDefinitions() {
		if definition.Domain != "ops" || definition.DomainKey != "systemConfig.domains.ops" {
			t.Fatalf("expected stable ops config domain metadata, got %#v", definition)
		}
	}
}
