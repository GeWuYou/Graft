package rbac

import (
	"testing"

	rbaccontract "graft/server/plugins/rbac/contract"
)

func TestRegisterRBACMenuIncludesTitleKey(t *testing.T) {
	ctx, _ := newPluginTestContext(t, testRBACRepository{})

	menus := ctx.MenuRegistry.Items()
	if len(menus) != 1 {
		t.Fatalf("expected 1 registered menu, got %d", len(menus))
	}

	menu := menus[0]
	if menu.Path != rbaccontract.RolesGroup ||
		menu.TitleKey != rbaccontract.RoleListMenuTitle.String() ||
		menu.Permission != rbaccontract.RoleReadPermission.String() {
		t.Fatalf("unexpected registered menu: %#v", menu)
	}
}
