package rbac

import (
	"testing"

	"graft/server/internal/menu"
	rbaccontract "graft/server/plugins/rbac/contract"
)

func TestRegisterRBACMenuIncludesTitleKey(t *testing.T) {
	ctx, _ := newPluginTestContext(t, testRBACRepository{})

	menus := ctx.MenuRegistry.Items()
	if len(menus) != 4 {
		t.Fatalf("expected 4 registered menus, got %d", len(menus))
	}

	assertRBACMenuItem(t, menus[0], expectedRBACMenuItem{
		path:     "/access-control",
		titleKey: rbaccontract.AccessControlMenuTitle.String(),
		icon:     "secured",
		order:    0,
	})
	assertRBACMenuItem(
		t,
		menus[1],
		expectedRBACMenuItem{
			path:     "/access-control/overview",
			titleKey: rbaccontract.AccessControlOverviewMenuTitle.String(),
			icon:     "dashboard",
			order:    1,
		},
	)
	assertRBACMenuItem(
		t,
		menus[2],
		expectedRBACMenuItem{
			path:       "/access-control/roles",
			titleKey:   rbaccontract.RoleListMenuTitle.String(),
			icon:       "secured",
			order:      3,
			permission: rbaccontract.RoleReadPermission.String(),
		},
	)
	assertRBACMenuItem(
		t,
		menus[3],
		expectedRBACMenuItem{
			path:       "/access-control/permissions",
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
