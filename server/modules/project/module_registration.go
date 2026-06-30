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

func registerPermissions(registry *permission.Registry, moduleName string) error {
	if registry == nil {
		return errors.New("permission registry is unavailable")
	}
	for _, item := range permissionItems(moduleName) {
		registry.Register(item)
	}
	return nil
}

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
	}
}

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
