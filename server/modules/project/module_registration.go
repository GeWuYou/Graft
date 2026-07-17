package project

import (
	"errors"

	"graft/server/internal/menu"
	"graft/server/internal/permission"
	projectcontract "graft/server/modules/project/contract"
)

const (
	operationsMenuOrderRoot = 50
	projectMenuOrderList    = 52
)

// registerPermissions 注册项目模块的权限项。
// 当权限注册器不可用时返回错误。
func registerPermissions(registry *permission.Registry, moduleName string) error {
	if registry == nil {
		return errors.New("permission registry is unavailable")
	}
	for _, item := range permissionItems(moduleName) {
		registry.Register(item)
	}
	return nil
}

// permissionItems 返回项目模块的权限目录条目列表，涵盖查看、导入、刷新、生命周期管理、销毁、创建、创建方式查看、发现候选和部署等权限。
func permissionItems(moduleName string) []permission.Item {
	return []permission.Item{
		{
			Code:           projectcontract.ApplicationViewPermission.String(),
			Name:           "",
			DisplayKey:     "rbac.permissionCatalog.applicationView.display",
			Description:    "",
			DescriptionKey: "rbac.permissionCatalog.applicationView.description",
			Module:         moduleName,
		},
		{
			Code:           projectcontract.ApplicationImportPermission.String(),
			Name:           "",
			DisplayKey:     "rbac.permissionCatalog.applicationImport.display",
			Description:    "",
			DescriptionKey: "rbac.permissionCatalog.applicationImport.description",
			Module:         moduleName,
		},
		{
			Code:           projectcontract.ApplicationRefreshPermission.String(),
			Name:           "",
			DisplayKey:     "rbac.permissionCatalog.applicationRefresh.display",
			Description:    "",
			DescriptionKey: "rbac.permissionCatalog.applicationRefresh.description",
			Module:         moduleName,
		},
		{
			Code:           projectcontract.ApplicationLifecyclePermission.String(),
			Name:           "",
			DisplayKey:     "rbac.permissionCatalog.applicationLifecycle.display",
			Description:    "",
			DescriptionKey: "rbac.permissionCatalog.applicationLifecycle.description",
			Module:         moduleName,
		},
		{
			Code:           projectcontract.ApplicationDestroyPermission.String(),
			Name:           "",
			DisplayKey:     "rbac.permissionCatalog.applicationDestroy.display",
			Description:    "",
			DescriptionKey: "rbac.permissionCatalog.applicationDestroy.description",
			Module:         moduleName,
		},
		{
			Code:           projectcontract.ApplicationCreatePermission.String(),
			Name:           "",
			DisplayKey:     "rbac.permissionCatalog.applicationCreate.display",
			Description:    "",
			DescriptionKey: "rbac.permissionCatalog.applicationCreate.description",
			Module:         moduleName,
		},
		{
			Code:           projectcontract.ApplicationCreationMethodViewPermission.String(),
			Name:           "",
			DisplayKey:     "rbac.permissionCatalog.applicationCreationMethodView.display",
			Description:    "",
			DescriptionKey: "rbac.permissionCatalog.applicationCreationMethodView.description",
			Module:         moduleName,
		},
		{
			Code:           projectcontract.ApplicationDiscoveryViewPermission.String(),
			Name:           "",
			DisplayKey:     "rbac.permissionCatalog.applicationDiscoveryView.display",
			Description:    "",
			DescriptionKey: "rbac.permissionCatalog.applicationDiscoveryView.description",
			Module:         moduleName,
		},
		{
			Code:           projectcontract.ApplicationTemplateManagePermission.String(),
			Name:           "",
			DisplayKey:     "rbac.permissionCatalog.applicationTemplateManage.display",
			Description:    "",
			DescriptionKey: "rbac.permissionCatalog.applicationTemplateManage.description",
			Module:         moduleName,
		},
		{
			Code:           projectcontract.ApplicationTemplatePublishPermission.String(),
			Name:           "",
			DisplayKey:     "rbac.permissionCatalog.applicationTemplatePublish.display",
			Description:    "",
			DescriptionKey: "rbac.permissionCatalog.applicationTemplatePublish.description",
			Module:         moduleName,
		},
		{
			Code:           projectcontract.ApplicationDeployPermission.String(),
			Name:           "",
			DisplayKey:     "rbac.permissionCatalog.applicationDeploy.display",
			Description:    "",
			DescriptionKey: "rbac.permissionCatalog.applicationDeploy.description",
			Module:         moduleName,
		},
	}
}

// registerMenu 注册项目模块的菜单项；菜单注册表不可用时返回错误。
// moduleName 写入每个菜单项的模块归属，供权限和模块装配边界保持一致。
func registerMenu(registry *menu.Registry, moduleName string) error {
	if registry == nil {
		return errors.New("menu registry is unavailable")
	}

	registry.Register(menu.Item{
		Code:       "application.list",
		ParentCode: "domain.application",
		Kind:       menu.NodeKindEntry,
		Title:      "",
		TitleKey:   projectcontract.ApplicationMenuTitle.String(),
		Path:       projectcontract.ApplicationMenuPath,
		Icon:       "application",
		Order:      projectMenuOrderList,
		Permission: projectcontract.ApplicationViewPermission.String(),
		Module:     moduleName,
	})
	registry.Register(menu.Item{
		Code:       "application.templates",
		ParentCode: "domain.application",
		Kind:       menu.NodeKindEntry,
		Title:      "",
		TitleKey:   projectcontract.ApplicationTemplateMenuTitle.String(),
		Path:       projectcontract.ApplicationTemplateManagementMenuPath,
		Icon:       "folder",
		Order:      projectMenuOrderList + 1,
		Permission: projectcontract.ApplicationTemplateManagePermission.String(),
		Module:     moduleName,
	})
	return nil
}
