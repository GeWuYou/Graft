package user

import (
	"context"
	"testing"

	"graft/server/internal/menu"
)

func TestFilterBootstrapMenusPrunesUnauthorizedAndEmptyDomainGroups(t *testing.T) {
	registry := menu.NewRegistry()
	menu.RegisterDomainGroups(registry)
	registry.Register(menu.Item{Code: "project.list", ParentCode: "domain.application", Kind: menu.NodeKindEntry, Path: "/projects", Permission: "project.read"})
	menus := filterBootstrapMenus(context.Background(), registry, map[string]struct{}{}, nil)
	if len(menus) != 0 {
		t.Fatalf("expected all empty or unauthorized groups pruned, got %#v", menus)
	}

	menus = filterBootstrapMenus(context.Background(), registry, map[string]struct{}{"project.read": {}}, nil)
	if len(menus) != 2 || menus[0].Code != "project.list" || menus[1].Code != "domain.application" {
		t.Fatalf("expected explicit application group and project entry, got %#v", menus)
	}
}
