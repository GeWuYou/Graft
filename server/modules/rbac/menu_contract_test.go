package rbac

import (
	"testing"

	"graft/server/internal/menu"
	rbaccontract "graft/server/modules/rbac/contract"
)

func TestRegisterRBACMenuIncludesTitleKey(t *testing.T) {
	ctx, _ := newModuleTestContext(t, testRBACRepository{})

	menus := ctx.MenuRegistry.Items()
	if len(menus) != 3 {
		t.Fatalf("expected 3 registered menus, got %d", len(menus))
	}

	assertRBACMenuItem(
		t,
		menus[0],
		expectedRBACMenuItem{
			path:     "/access-control/overview",
			titleKey: rbaccontract.AccessControlOverviewMenuTitle.String(),
			icon:     "dashboard",
			order:    1,
		},
	)
	assertRBACMenuItem(
		t,
		menus[1],
		expectedRBACMenuItem{
			path:       "/roles",
			titleKey:   rbaccontract.RoleListMenuTitle.String(),
			icon:       "secured",
			order:      3,
			permission: rbaccontract.RoleReadPermission.String(),
		},
	)
	assertRBACMenuItem(
		t,
		menus[2],
		expectedRBACMenuItem{
			path:       "/permissions",
			titleKey:   rbaccontract.PermissionListMenuTitle.String(),
			icon:       "lock-on",
			order:      4,
			permission: rbaccontract.PermissionReadPermission.String(),
		},
	)
}

type expectedRBACMenuItem struct {
	path       string
	titleKey   string
	icon       string
	order      int
	permission string
}

func assertRBACMenuItem(t *testing.T, item menu.Item, expected expectedRBACMenuItem) {
	t.Helper()

	if item.Path != expected.path {
		t.Fatalf("unexpected menu path: %#v", item)
	}
	if item.TitleKey != expected.titleKey {
		t.Fatalf("unexpected menu title key: %#v", item)
	}
	if item.Icon != expected.icon {
		t.Fatalf("unexpected menu icon: %#v", item)
	}
	if item.Order != expected.order {
		t.Fatalf("unexpected menu order: %#v", item)
	}
	if item.Permission != expected.permission {
		t.Fatalf("unexpected menu permission: %#v", item)
	}
}
