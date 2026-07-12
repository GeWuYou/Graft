package user

import (
	"context"
	"strings"
	"testing"

	"graft/server/internal/menu"
	"graft/server/internal/moduleapi"
)

func TestFilterBootstrapMenusPrunesUnauthorizedAndEmptyDomainGroups(t *testing.T) {
	registry := menu.NewRegistry()
	menu.RegisterDomainGroups(registry)
	registry.Register(menu.Item{Code: "project.list", ParentCode: "domain.application", Kind: menu.NodeKindEntry, Path: "/applications/projects", Permission: "project.read"})
	menus := filterBootstrapMenus(context.Background(), registry, map[string]struct{}{}, nil)
	if len(menus) != 0 {
		t.Fatalf("expected all empty or unauthorized groups pruned, got %#v", menus)
	}

	menus = filterBootstrapMenus(context.Background(), registry, map[string]struct{}{"project.read": {}}, nil)
	if len(menus) != 2 || menus[0].Code != "project.list" || menus[1].Code != "domain.application" {
		t.Fatalf("expected explicit application group and project entry, got %#v", menus)
	}
}

func TestCompareBootstrapMenusUsesParentCodeForEqualOrder(t *testing.T) {
	parent := bootstrapMenuResponse{Code: "domain.security", Order: 10}
	child := bootstrapMenuResponse{Code: "user.list", ParentCode: "domain.security", Order: 10}
	otherChild := bootstrapMenuResponse{Code: "monitor.list", ParentCode: "domain.observability", Order: 10}

	if compareBootstrapMenus(parent, child) >= 0 {
		t.Fatalf("expected parent group to precede child with equal order")
	}
	if compareBootstrapMenus(otherChild, child) >= 0 {
		t.Fatalf("expected parent code to order equal-order entries")
	}
}

func TestFilterBootstrapMenusPreservesVisualSectionMetadata(t *testing.T) {
	registry := menu.NewRegistry()
	menu.RegisterDomainGroups(registry)
	registry.Register(menu.Item{
		Code:            "user.list",
		ParentCode:      "domain.security",
		Kind:            menu.NodeKindEntry,
		Path:            "/security/users",
		Permission:      "user.read",
		SectionKey:      menu.AccessControlSectionKey,
		SectionTitleKey: menu.AccessControlSectionTitleKey,
	})

	menus := filterBootstrapMenus(context.Background(), registry, map[string]struct{}{"user.read": {}}, nil)
	for _, item := range menus {
		if item.Code == "user.list" && (item.SectionKey != menu.AccessControlSectionKey || item.SectionTitleKey != menu.AccessControlSectionTitleKey) {
			t.Fatalf("expected access-control metadata, got %#v", item)
		}
	}
}

func TestBootstrapReaderReadReturnsInvalidMenuGraph(t *testing.T) {
	registry := menu.NewRegistry()
	registry.Register(menu.Item{Code: "invalid-group", Kind: menu.NodeKindGroup, Path: "/unexpected"})
	reader := bootstrapReader{menuRegistry: registry}
	ctx := moduleapi.WithRequestAuthContext(context.Background(), moduleapi.RequestAuthContext{
		User: &moduleapi.CurrentUser{ID: 1},
	})

	_, err := reader.Read(ctx, nil)
	if err == nil || !strings.Contains(err.Error(), "must not declare a path") {
		t.Fatalf("expected malformed menu graph error, got %v", err)
	}
}
