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

// 列表包含查看、导入、刷新、生命周期管理、销毁、创建和部署相关的权限。
func permissionItems(moduleName string) []permission.Item {
	return []permission.Item{
		{
			Code:           projectcontract.ProjectViewPermission.String(),
			Name:           "View compose projects",
			DisplayKey:     "rbac.permissionCatalog.projectView.display",
			Description:    "Read registered Compose project summaries and readonly details.",
			DescriptionKey: "rbac.permissionCatalog.projectView.description",
			Category:       "api",
			Module:         moduleName,
		},
		{
			Code:           projectcontract.ProjectImportPermission.String(),
			Name:           "Import compose projects",
			DisplayKey:     "rbac.permissionCatalog.projectImport.display",
			Description:    "Validate and import external Compose projects into the project registry.",
			DescriptionKey: "rbac.permissionCatalog.projectImport.description",
			Category:       "api",
			Module:         moduleName,
		},
		{
			Code:           projectcontract.ProjectRefreshPermission.String(),
			Name:           "Refresh compose projects",
			DisplayKey:     "rbac.permissionCatalog.projectRefresh.display",
			Description:    "Refresh project snapshots and readonly configuration projections.",
			DescriptionKey: "rbac.permissionCatalog.projectRefresh.description",
			Category:       "api",
			Module:         moduleName,
		},
		{
			Code:           projectcontract.ProjectLifecyclePermission.String(),
			Name:           "Manage compose project lifecycle",
			DisplayKey:     "rbac.permissionCatalog.projectLifecycle.display",
			Description:    "Run bounded Compose lifecycle actions such as up, down, and restart.",
			DescriptionKey: "rbac.permissionCatalog.projectLifecycle.description",
			Category:       "api",
			Module:         moduleName,
		},
		{
			Code:           projectcontract.ProjectDestroyPermission.String(),
			Name:           "Unregister or destroy compose projects",
			DisplayKey:     "rbac.permissionCatalog.projectDestroy.display",
			Description:    "Run unregister and guarded destroy actions for project records.",
			DescriptionKey: "rbac.permissionCatalog.projectDestroy.description",
			Category:       "api",
			Module:         moduleName,
		},
		{
			Code:           projectcontract.ProjectCreatePermission.String(),
			Name:           "Create managed compose projects",
			DisplayKey:     "rbac.permissionCatalog.projectCreate.display",
			Description:    "Inspect managed-root authority and access future managed project create workflows.",
			DescriptionKey: "rbac.permissionCatalog.projectCreate.description",
			Category:       "api",
			Module:         moduleName,
		},
		{
			Code:           projectcontract.ProjectSourceViewPermission.String(),
			Name:           "View compose project source entrypoints",
			DisplayKey:     "rbac.permissionCatalog.projectSourceView.display",
			Description:    "Inspect the Phase 3 source catalog and source-selector routes for managed, git, and template project flows.",
			DescriptionKey: "rbac.permissionCatalog.projectSourceView.description",
			Category:       "api",
			Module:         moduleName,
		},
		{
			Code:           projectcontract.ProjectDiscoveryViewPermission.String(),
			Name:           "View compose project discovery candidates",
			DisplayKey:     "rbac.permissionCatalog.projectDiscoveryView.display",
			Description:    "Inspect bounded local directory-scan and auto-discovery candidate previews without registering projects.",
			DescriptionKey: "rbac.permissionCatalog.projectDiscoveryView.description",
			Category:       "api",
			Module:         moduleName,
		},
		{
			Code:           projectcontract.ProjectDeployPermission.String(),
			Name:           "Deploy managed compose project drafts",
			DisplayKey:     "rbac.permissionCatalog.projectDeploy.display",
			Description:    "Diff, validate, and deploy managed project configuration drafts.",
			DescriptionKey: "rbac.permissionCatalog.projectDeploy.description",
			Category:       "api",
			Module:         moduleName,
		},
	}
}

// registerMenu 为项目模块注册菜单项。
// 当 registry 为空时，返回错误；否则注册“Operations”根菜单和“Compose Projects”列表菜单，并将 moduleName 作为模块标识写入菜单项。
// @param moduleName 模块名称。
// @returns 注册失败时返回错误，成功时返回 nil。
func registerMenu(registry *menu.Registry, moduleName string) error {
	if registry == nil {
		return errors.New("menu registry is unavailable")
	}

	registry.Register(menu.Item{
		Code:       "ops.root",
		Title:      "Operations",
		TitleKey:   "menu.ops.title",
		Path:       projectcontract.ProjectMenuRootPath,
		Icon:       "tools",
		Order:      operationsMenuOrderRoot,
		Permission: "",
		Module:     moduleName,
	})
	registry.Register(menu.Item{
		Code:       "project.list",
		Title:      "Compose Projects",
		TitleKey:   projectcontract.ProjectMenuTitle.String(),
		Path:       projectcontract.ProjectMenuPath,
		Icon:       "folder-open",
		Order:      projectMenuOrderList,
		Permission: projectcontract.ProjectViewPermission.String(),
		Module:     moduleName,
	})
	return nil
}
