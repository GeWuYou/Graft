package rbac

import (
	"testing"

	rbaccontract "graft/server/plugins/rbac/contract"
)

func TestRegisterRBACMenuIncludesTitleKey(t *testing.T) {
	ctx, _ := newPluginTestContext(t, testRBACRepository{})

	menus := ctx.MenuRegistry.Items()
	if len(menus) != 3 {
		t.Fatalf("expected 3 registered menus, got %d", len(menus))
	}

	if menus[0].Path != "/access-control/overview" ||
		menus[0].TitleKey != rbaccontract.AccessControlOverviewMenuTitle.String() {
		t.Fatalf("unexpected overview menu: %#v", menus[0])
	}
	if menus[1].Path != "/access-control/roles" ||
		menus[1].TitleKey != rbaccontract.RoleListMenuTitle.String() ||
		menus[1].Permission != rbaccontract.RoleReadPermission.String() {
		t.Fatalf("unexpected role menu: %#v", menus[1])
	}
	if menus[2].Path != "/access-control/permissions" ||
		menus[2].TitleKey != rbaccontract.PermissionListMenuTitle.String() ||
		menus[2].Permission != rbaccontract.PermissionReadPermission.String() {
		t.Fatalf("unexpected permission menu: %#v", menus[2])
	}
}
