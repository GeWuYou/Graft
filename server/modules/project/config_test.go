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
