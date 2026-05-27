package rbac

import (
	"testing"

	rbaccontract "graft/server/plugins/rbac/contract"
)

func TestRegisterRBACMenuIncludesTitleKey(t *testing.T) {
	ctx, _ := newPluginTestContext(t, testRBACRepository{})

	menus := ctx.MenuRegistry.Items()
	if len(menus) != 4 {
		t.Fatalf("expected 4 registered menus, got %d", len(menus))
	}

	if menus[0].Path != "/access-control" ||
		menus[0].TitleKey != rbaccontract.AccessControlMenuTitle.String() {
		t.Fatalf("unexpected root menu: %#v", menus[0])
	}
	if menus[1].Path != "/access-control/overview" ||
		menus[1].Icon != "dashboard" ||
		menus[1].TitleKey != rbaccontract.AccessControlOverviewMenuTitle.String() {
		t.Fatalf("unexpected overview menu: %#v", menus[1])
	}
	if menus[2].Path != "/access-control/roles" ||
		menus[2].TitleKey != rbaccontract.RoleListMenuTitle.String() ||
		menus[2].Permission != rbaccontract.RoleReadPermission.String() {
		t.Fatalf("unexpected role menu: %#v", menus[2])
	}
	if menus[3].Path != "/access-control/permissions" ||
		menus[3].Icon != "lock-on" ||
		menus[3].TitleKey != rbaccontract.PermissionListMenuTitle.String() ||
		menus[3].Permission != rbaccontract.PermissionReadPermission.String() {
		t.Fatalf("unexpected permission menu: %#v", menus[3])
	}
}
