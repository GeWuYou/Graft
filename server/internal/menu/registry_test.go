package menu

import "testing"

func TestRegistryValidateRejectsMalformedNavigationGraph(t *testing.T) {
	tests := []struct {
		name  string
		items []Item
	}{
		{"group path", []Item{{Code: "group", Kind: NodeKindGroup, Path: "/unexpected"}}},
		{"entry missing path", []Item{{Code: "entry", Kind: NodeKindEntry}}},
		{"unknown parent", []Item{{Code: "entry", Kind: NodeKindEntry, Path: "/entries", ParentCode: "missing"}}},
		{"entry parent", []Item{{Code: "entry", Kind: NodeKindEntry, Path: "/entries"}, {Code: "child", Kind: NodeKindEntry, Path: "/children", ParentCode: "entry"}}},
		{"cycle", []Item{{Code: "a", Kind: NodeKindGroup, ParentCode: "b"}, {Code: "b", Kind: NodeKindGroup, ParentCode: "a"}}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			registry := NewRegistry()
			for _, item := range test.items {
				registry.Register(item)
			}
			if err := registry.Validate(); err == nil {
				t.Fatal("expected invalid navigation graph")
			}
		})
	}
}

func TestRegistryValidateAcceptsExplicitDomainGraph(t *testing.T) {
	registry := NewRegistry()
	RegisterDomainGroups(registry)
	registry.Register(Item{Code: "project.list", ParentCode: "domain.application", Kind: NodeKindEntry, Path: "/projects"})
	if err := registry.Validate(); err != nil {
		t.Fatalf("validate graph: %v", err)
	}
}
