package contract

const (
	// ProjectAPIGroup 标识 Compose 项目管理 API 路由组。
	ProjectAPIGroup = "/ops/projects"
	// ProjectCollectionRoute 标识项目列表接口的路径片段。
	ProjectCollectionRoute = ""
	// ProjectSavedViewsRoute 标识项目列表保存视图集合接口的路径片段。
	ProjectSavedViewsRoute = "/saved-views"
	// ProjectSavedViewRoute 标识单个项目列表保存视图接口的路径片段。
	ProjectSavedViewRoute = "/saved-views/:viewId"
	// ProjectDetailRoute 标识单个项目摘要接口的路径片段。
	ProjectDetailRoute = "/:id"
	// ProjectServicesRoute 标识项目服务聚合接口的路径片段。
	ProjectServicesRoute = "/:id/services"
	// ProjectLogsRoute 标识项目聚合日志接口的路径片段。
	ProjectLogsRoute = "/:id/logs"
	// ProjectOverviewRoute 标识项目运行时概览接口的路径片段。
	ProjectOverviewRoute = "/:id/overview"
	// ProjectConfigurationRoute 标识项目配置元数据接口的路径片段。
	ProjectConfigurationRoute = "/:id/configuration"
	// ProjectConfigurationPreviewRoute 标识规范化配置预览接口的路径片段。
	ProjectConfigurationPreviewRoute = "/:id/configuration/preview"
	// ProjectWorkspaceFilesRoute 标识按需加载项目根目录文件树接口的路径片段。
	ProjectWorkspaceFilesRoute = "/:id/files"
	// ProjectWorkspaceFileContentRoute 标识按路径读写项目文件接口的路径片段。
	ProjectWorkspaceFileContentRoute = "/:id/files/content"
	// ProjectWorkspaceFileAnnotationRoute 标识按路径写入工作区注释接口的路径片段。
	ProjectWorkspaceFileAnnotationRoute = "/:id/files/annotation"
	// ProjectWorkspaceEntryRoute 标识项目工作区条目变更接口的路径片段。
	ProjectWorkspaceEntryRoute = "/:id/files/entries"
	// ProjectWorkspaceRenameRoute 标识项目工作区条目重命名接口的路径片段。
	ProjectWorkspaceRenameRoute = "/:id/files/rename"
	// ProjectImportValidateRoute 标识项目导入校验接口的路径片段。
	ProjectImportValidateRoute = "/import/validate"
	// ProjectImportRuntimeCandidatesRoute 标识运行时驱动的导入候选列表接口路径片段。
	ProjectImportRuntimeCandidatesRoute = "/import/runtime-candidates"
	// ProjectImportRuntimeInspectRoute 标识运行时驱动的导入检查接口路径片段。
	ProjectImportRuntimeInspectRoute = "/import/runtime-inspect"
	// ProjectImportInspectRoute 标识项目导入检查接口的路径片段。
	ProjectImportInspectRoute = "/import/inspect"
	// ProjectImportRoute 标识项目导入并注册接口的路径片段。
	ProjectImportRoute = "/import"
	// ProjectImportDirectorySourcesRoute 标识可用导入目录源根接口的路径片段。
	ProjectImportDirectorySourcesRoute = "/import/directory-sources"
	// ProjectImportDirectoriesRoute 标识按根目录相对路径浏览导入目录接口的路径片段。
	ProjectImportDirectoriesRoute = "/import/directories"
	// ProjectCreationMethodsRoute 标识项目创建方式目录接口的路径片段。
	ProjectCreationMethodsRoute = "/creation-methods"
	// ProjectDiscoveryCandidatesRoute 标识有界发现候选预览接口的路径片段。
	ProjectDiscoveryCandidatesRoute = "/discovery-candidates"
	// ProjectCreationMethodSelectorRoute 标识创建方式选择器接口的路径片段。
	ProjectCreationMethodSelectorRoute = "/create"
	// ProjectComposeRuntimeTargetsRoute 标识 Compose 创建可选运行时目标目录接口的路径片段。
	ProjectComposeRuntimeTargetsRoute = "/create/runtime-targets"
	// ProjectManagedRootRoute 标识受管根目录元数据接口的路径片段。
	ProjectManagedRootRoute = "/managed/root"
	// ProjectCreateValidateRoute 标识受管创建校验接口的路径片段。
	ProjectCreateValidateRoute = "/create/managed/validate"
	// ProjectApplicationNameAvailabilityRoute 标识受管创建应用名称预检接口的路径片段。
	ProjectApplicationNameAvailabilityRoute = "/create/application-name/availability"
	// ProjectCreateRoute 标识受管创建接口的路径片段。
	ProjectCreateRoute = "/create/managed"
	// ProjectCreateTemplateValidateRoute validates a runtime template source without materializing it.
	ProjectCreateTemplateValidateRoute = "/create/template/validate"
	// ProjectCreateTemplateRoute 标识模板来源创建接口的路径片段。
	ProjectCreateTemplateRoute = "/create/template"
	// ProjectWorkspaceDefaultsRoute 标识返回服务端拥有的空白工作区默认值和可用模板的接口路径片段。
	ProjectWorkspaceDefaultsRoute = "/create/workspace-defaults"
	// ProjectRefreshRoute 标识静态刷新接口的路径片段。
	ProjectRefreshRoute = "/:id/refresh"
	// ProjectUpRoute 标识 Compose 启动接口的路径片段。
	ProjectUpRoute = "/:id/up"
	// ProjectStopRoute 标识 Compose 停止接口的路径片段。
	ProjectStopRoute = "/:id/stop"
	// ProjectRestartRoute 标识 Compose 重启接口的路径片段。
	ProjectRestartRoute = "/:id/restart"
	// ProjectRedeployRoute 标识 Compose 重部署接口的路径片段。
	ProjectRedeployRoute = "/:id/redeploy"
	// ProjectLifecycleConfigurationRoute 标识项目生命周期配置接口的路径片段。
	ProjectLifecycleConfigurationRoute = "/:id/lifecycle-configuration"
	// ProjectUnregisterRoute 标识项目注销接口的路径片段。
	ProjectUnregisterRoute = "/:id/unregister"
	// ProjectDestroyRoute 标识受保护销毁接口的路径片段。
	ProjectDestroyRoute = "/:id/destroy"
	// ProjectBatchActionsRoute 标识批量动作接口的路径片段。
	ProjectBatchActionsRoute = "/batch-actions"
	// ProjectMenuRootPath 标识运维页面的 Web 菜单根路径。
	ProjectMenuRootPath = "/applications/projects"
	// ProjectMenuPath 标识 Compose 项目管理的规范 Web 菜单路径。
	ProjectMenuPath = ProjectMenuRootPath
	// ProjectCreationMenuPath 标识全局创建方式选择器路由路径。
	ProjectCreationMenuPath = ProjectMenuRootPath + "/create"
	// ProjectBlankCreateMenuPath 标识空白项目创建路由路径。
	ProjectBlankCreateMenuPath = ProjectCreationMenuPath + "/blank"
	// ProjectTemplateCreateMenuPath 标识模板来源创建路由路径。
	ProjectTemplateCreateMenuPath = ProjectCreationMenuPath + "/template"
	// ProjectImportCreateMenuPath 标识导入项目创建路由路径。
	ProjectImportCreateMenuPath = ProjectCreationMenuPath + "/import"
	// ProjectDiscoveryCandidatesMenuPath 标识隐藏的发现候选预览路由路径。
	ProjectDiscoveryCandidatesMenuPath = ProjectCreationMenuPath + "/discovery"
)
