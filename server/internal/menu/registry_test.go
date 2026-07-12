package menu

import "testing"

func TestRegistryValidateRejectsMalformedNavigationGraph(t *testing.T) {
	tests := []struct {
		name  string
		items []Item
	}{
		{"group path", []Item{{Code: "group", Kind: NodeKindGroup, Path: "/unexpected"}}},
		{"group incomplete section", []Item{{Code: "group", Kind: NodeKindGroup, SectionKey: "runtime"}}},
		{"entry incomplete section", []Item{{Code: "entry", Kind: NodeKindEntry, Path: "/entries", SectionTitleKey: "menu.section.runtime"}}},
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

func TestRegistryValidateAcceptsSectionedNestedGroup(t *testing.T) {
	registry := NewRegistry()
	RegisterDomainGroups(registry)
	registry.Register(Item{
		Code:            "docker",
		ParentCode:      "domain.infrastructure",
		Kind:            NodeKindGroup,
		SectionKey:      RuntimeSectionKey,
		SectionTitleKey: "menu.section.runtime",
	})
	registry.Register(Item{Code: "container.list", ParentCode: "docker", Kind: NodeKindEntry, Path: "/infrastructure/docker/containers"})
	if err := registry.Validate(); err != nil {
		t.Fatalf("validate graph: %v", err)
	}
}

func TestRegistryValidateRejectsEmptyDuplicateAndInvalidKinds(t *testing.T) {
	tests := []struct {
		name  string
		items []Item
	}{
		{"empty code", []Item{{Kind: NodeKindEntry, Path: "/entries"}}},
		{"duplicate code", []Item{{Code: "entry", Kind: NodeKindEntry, Path: "/entries"}, {Code: "entry", Kind: NodeKindEntry, Path: "/other"}}},
		{"invalid kind", []Item{{Code: "entry", Kind: NodeKind("unknown"), Path: "/entries"}}},
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

func TestRegistryRegisterNormalizesEmptyKind(t *testing.T) {
	registry := NewRegistry()
	registry.Register(Item{Code: "entry", Path: "/entries"})

	items := registry.Items()
	if len(items) != 1 || items[0].Kind != NodeKindEntry {
		t.Fatalf("expected empty kind normalized to entry, got %#v", items)
	}
}

func TestRegisterDomainGroupsProvidesIcons(t *testing.T) {
	registry := NewRegistry()
	RegisterDomainGroups(registry)

	expectedIcons := map[string]string{
		"domain.application":    "application",
		"domain.infrastructure": "infrastructure",
		"domain.build":          "build",
		"domain.resources":      "resources",
		"domain.observability":  "observability",
		"domain.security":       "security",
		"domain.platform":       "platform",
	}
	for _, item := range registry.Items() {
		if expected, ok := expectedIcons[item.Code]; !ok || item.Icon != expected {
			t.Fatalf("unexpected domain icon for %q: %#v", item.Code, item)
		}
	}
}
