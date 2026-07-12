package runtimetarget

import (
	"errors"

	"graft/server/internal/i18n"
	"graft/server/internal/menu"
	"graft/server/internal/module"
	"graft/server/internal/permission"
	contract "graft/server/modules/runtime-target/contract"
)

const runtimeTargetMenuOrder = 40

func registerModuleMetadata(ctx *module.Context, moduleName string) error {
	if ctx == nil || ctx.I18n == nil || ctx.PermissionRegistry == nil || ctx.MenuRegistry == nil {
		return errors.New("runtime target module context is unavailable")
	}
	for _, locale := range []i18n.LocaleTag{i18n.LocaleZHCN, i18n.LocaleENUS} {
		if len(ctx.I18n.RegisteredMessageResources(locale, i18n.MessageKey(contract.MenuTitle))) == 0 {
			return errors.New("runtime target locale resource is unavailable")
		}
	}
	for _, item := range []permission.Item{
		{Code: contract.ViewPermission, DisplayKey: "rbac.permissionCatalog.runtimeTargetView.display", DescriptionKey: "rbac.permissionCatalog.runtimeTargetView.description", Module: moduleName},
		{Code: contract.ManagePermission, DisplayKey: "rbac.permissionCatalog.runtimeTargetManage.display", DescriptionKey: "rbac.permissionCatalog.runtimeTargetManage.description", Module: moduleName},
		{Code: contract.RefreshPermission, DisplayKey: "rbac.permissionCatalog.runtimeTargetRefresh.display", DescriptionKey: "rbac.permissionCatalog.runtimeTargetRefresh.description", Module: moduleName},
	} {
		ctx.PermissionRegistry.Register(item)
	}
	ctx.MenuRegistry.Register(menu.Item{Code: "runtime-target.list", ParentCode: "domain.infrastructure", Kind: menu.NodeKindEntry, TitleKey: contract.MenuTitle, Path: contract.MenuPath, Icon: "server", Order: runtimeTargetMenuOrder, Permission: contract.ViewPermission, Module: moduleName})
	return nil
}
