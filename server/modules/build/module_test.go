package build

import (
	"slices"
	"testing"

	"graft/server/internal/menu"
	"graft/server/internal/module"
	"graft/server/internal/permission"
	buildcontract "graft/server/modules/build/contract"
)

func TestModuleRegistersBuildPermissionsAndMenu(t *testing.T) {
	menuRegistry := menu.NewRegistry()
	menu.RegisterDomainGroups(menuRegistry)
	permissionRegistry := permission.NewRegistry()

	if err := NewModule().Register(&module.Context{
		MenuRegistry:       menuRegistry,
		PermissionRegistry: permissionRegistry,
	}); err != nil {
		t.Fatalf("register build module: %v", err)
	}

	permissions := permissionRegistry.Items()
	for _, code := range []string{
		buildcontract.BuildReadPermission,
		buildcontract.BuildCreatePermission,
		buildcontract.BuildCancelPermission,
		buildcontract.BuildRetryPermission,
	} {
		if !slices.ContainsFunc(permissions, func(item permission.Item) bool {
			return item.Code == code && item.Module == moduleID
		}) {
			t.Fatalf("expected build permission %q, got %#v", code, permissions)
		}
	}

	if err := menuRegistry.Validate(); err != nil {
		t.Fatalf("validate build menu: %v", err)
	}
	menus := menuRegistry.Items()
	if !slices.ContainsFunc(menus, func(item menu.Item) bool {
		return item.Code == "build.jobs" &&
			item.ParentCode == "domain.build" &&
			item.Path == "/build/jobs" &&
			item.Permission == buildcontract.BuildReadPermission &&
			item.Module == moduleID
	}) {
		t.Fatalf("expected build jobs menu, got %#v", menus)
	}
}
