package user

import (
	"testing"

	"graft/server/internal/menu"
	usercontract "graft/server/plugins/user/contract"
)

func TestRegisterUserMenuIncludesTitleKey(t *testing.T) {
	ctx, _ := newPluginTestContext(t, pluginTestUserRepository{}, &pluginTestAuthRepository{})

	menus := ctx.MenuRegistry.Items()
	if len(menus) == 0 {
		t.Fatalf("expected registered menu items, got %#v", menus)
	}

	menu := menus[0]
	if menu.Code != "user.list" ||
		menu.TitleKey != usercontract.UserListMenuTitle.String() ||
		menu.Path != "/access-control/users" ||
		menu.Permission != usercontract.UserReadPermission.String() {
		t.Fatalf("expected canonical user menu contract, got %#v", menu)
	}
}

func TestFilterBootstrapMenusIncludesTitleKeyAndFallback(t *testing.T) {
	menus := filterBootstrapMenus(testMenuRegistry(), map[string]struct{}{
		usercontract.UserReadPermission.String(): {},
	})

	if len(menus) != 2 {
		t.Fatalf("expected filtered menus to keep user and public entries, got %#v", menus)
	}
	if menus[0].Code != "user.list" ||
		menus[0].TitleKey != usercontract.UserListMenuTitle.String() ||
		menus[0].Title != "用户管理" {
		t.Fatalf("expected canonical user bootstrap menu, got %#v", menus[0])
	}
	if menus[1].Code != "profile.self" || menus[1].TitleKey != "" || menus[1].Title != "个人中心" {
		t.Fatalf("expected fallback-only public menu, got %#v", menus[1])
	}
}

func testMenuRegistry() *menu.Registry {
	registry := menu.NewRegistry()
	registerUserMenu(registry, "user")
	registry.Register(menu.Item{
		Code:  "profile.self",
		Title: "个人中心",
		Path:  "/profile",
		Icon:  "user",
	})

	return registry
}
