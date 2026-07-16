package contract

const (
	// ApplicationAPIGroup 标识 Compose Application 管理 API 路由组。
	ApplicationAPIGroup = "/ops/applications"
	// ApplicationCollectionRoute 标识应用列表接口的路径片段。
	ApplicationCollectionRoute = ""
	// ApplicationTemplatesRoute 必须在 /:applicationId 前注册，避免被应用详情动态路由吞掉。
	ApplicationTemplatesRoute = "/templates"
	// ApplicationTemplateDetailRoute 标识模板详情和草稿编辑接口。
	ApplicationTemplateDetailRoute = "/templates/:templateId"
	// ApplicationTemplateDeriveRoute 标识从已发布版本派生草稿的接口。
	ApplicationTemplateDeriveRoute = "/templates/:templateId/derive"
	// ApplicationTemplatePublishRoute 标识模板草稿发布接口。
	ApplicationTemplatePublishRoute = "/templates/:templateId/publish"
	// ApplicationTemplateArchiveRoute 标识模板归档接口。
	ApplicationTemplateArchiveRoute = "/templates/:templateId/archive"
	// ApplicationTemplateLegacyImportRoute 标识管理员显式导入旧目录模板接口。
	ApplicationTemplateLegacyImportRoute = "/templates/import-legacy"
	// ApplicationSavedViewsRoute 标识应用列表保存视图集合接口的路径片段。
	ApplicationSavedViewsRoute = "/saved-views"
	// ApplicationSavedViewRoute 标识单个应用列表保存视图接口的路径片段。
	ApplicationSavedViewRoute = "/saved-views/:viewId"
	// ApplicationDetailRoute 标识单个应用摘要接口的路径片段。
	ApplicationDetailRoute = "/:applicationId"
	// ApplicationServicesRoute 标识应用服务聚合接口的路径片段。
	ApplicationServicesRoute = "/:applicationId/services"
	// ApplicationLogsRoute 标识应用聚合日志接口的路径片段。
	ApplicationLogsRoute = "/:applicationId/logs"
	// ApplicationOverviewRoute 标识应用运行时概览接口的路径片段。
	ApplicationOverviewRoute = "/:applicationId/overview"
	// ApplicationConfigurationRoute 标识应用配置元数据接口的路径片段。
	ApplicationConfigurationRoute = "/:applicationId/configuration"
	// ApplicationConfigurationPreviewRoute 标识规范化配置预览接口的路径片段。
	ApplicationConfigurationPreviewRoute = "/:applicationId/configuration/preview"
	// ApplicationWorkspaceFilesRoute 标识按需加载应用工作区文件树接口的路径片段。
	ApplicationWorkspaceFilesRoute = "/:applicationId/files"
	// ApplicationWorkspaceFileContentRoute 标识按路径读写应用文件接口的路径片段。
	ApplicationWorkspaceFileContentRoute = "/:applicationId/files/content"
	// ApplicationWorkspaceFileAnnotationRoute 标识按路径写入工作区注释接口的路径片段。
	ApplicationWorkspaceFileAnnotationRoute = "/:applicationId/files/annotation"
	// ApplicationWorkspaceEntryRoute 标识应用工作区条目变更接口的路径片段。
	ApplicationWorkspaceEntryRoute = "/:applicationId/files/entries"
	// ApplicationWorkspaceRenameRoute 标识应用工作区条目重命名接口的路径片段。
	ApplicationWorkspaceRenameRoute = "/:applicationId/files/rename"
	// ApplicationImportValidateRoute 标识应用导入校验接口的路径片段。
	ApplicationImportValidateRoute = "/import/validate"
	// ApplicationImportRuntimeCandidatesRoute 标识运行时驱动的导入候选列表接口路径片段。
	ApplicationImportRuntimeCandidatesRoute = "/import/runtime-candidates"
	// ApplicationImportRuntimeInspectRoute 标识运行时驱动的导入检查接口路径片段。
	ApplicationImportRuntimeInspectRoute = "/import/runtime-inspect"
	// ApplicationImportInspectRoute 标识应用导入检查接口的路径片段。
	ApplicationImportInspectRoute = "/import/inspect"
	// ApplicationImportRoute 标识应用导入并注册接口的路径片段。
	ApplicationImportRoute = "/import"
	// ApplicationImportDirectorySourcesRoute 标识可用导入目录源根接口的路径片段。
	ApplicationImportDirectorySourcesRoute = "/import/directory-sources"
	// ApplicationImportDirectoriesRoute 标识按根目录相对路径浏览导入目录接口的路径片段。
	ApplicationImportDirectoriesRoute = "/import/directories"
	// ApplicationCreationMethodsRoute 标识应用创建方式目录接口的路径片段。
	ApplicationCreationMethodsRoute = "/creation-methods"
	// ApplicationDiscoveryCandidatesRoute 标识有界发现候选预览接口的路径片段。
	ApplicationDiscoveryCandidatesRoute = "/discovery-candidates"
	// ApplicationCreationMethodSelectorRoute 标识创建方式选择器接口的路径片段。
	ApplicationCreationMethodSelectorRoute = "/create"
	// ApplicationComposeRuntimeTargetsRoute 标识 Compose 创建可选运行时目标目录接口的路径片段。
	ApplicationComposeRuntimeTargetsRoute = "/create/runtime-targets"
	// ApplicationManagedRootRoute 标识受管根目录元数据接口的路径片段。
	ApplicationManagedRootRoute = "/managed/root"
	// ApplicationCreateValidateRoute 标识受管创建校验接口的路径片段。
	ApplicationCreateValidateRoute = "/create/managed/validate"
	// ApplicationNameAvailabilityRoute 标识受管创建应用名称预检接口的路径片段。
	ApplicationNameAvailabilityRoute = "/create/application-name/availability"
	// ApplicationCreateRoute 标识受管创建接口的路径片段。
	ApplicationCreateRoute = "/create/managed"
	// ApplicationRefreshRoute 标识静态刷新接口的路径片段。
	ApplicationRefreshRoute = "/:applicationId/refresh"
	// ApplicationUpRoute 标识 Compose 启动接口的路径片段。
	ApplicationUpRoute = "/:applicationId/up"
	// ApplicationStopRoute 标识 Compose 停止接口的路径片段。
	ApplicationStopRoute = "/:applicationId/stop"
	// ApplicationRestartRoute 标识 Compose 重启接口的路径片段。
	ApplicationRestartRoute = "/:applicationId/restart"
	// ApplicationRedeployRoute 标识 Compose 重部署接口的路径片段。
	ApplicationRedeployRoute = "/:applicationId/redeploy"
	// ApplicationLifecycleConfigurationRoute 标识应用生命周期配置接口的路径片段。
	ApplicationLifecycleConfigurationRoute = "/:applicationId/lifecycle-configuration"
	// ApplicationUnregisterRoute 标识应用注销接口的路径片段。
	ApplicationUnregisterRoute = "/:applicationId/unregister"
	// ApplicationDestroyRoute 标识受保护销毁接口的路径片段。
	ApplicationDestroyRoute = "/:applicationId/destroy"
	// ApplicationBatchActionsRoute 标识批量动作接口的路径片段。
	ApplicationBatchActionsRoute = "/batch-actions"
	// ApplicationMenuRootPath 标识运维页面的 Web 菜单根路径。
	ApplicationMenuRootPath = "/applications"
	// ApplicationMenuPath 标识 Compose Application 管理的规范 Web 菜单路径。
	ApplicationMenuPath = ApplicationMenuRootPath
	// ApplicationCreationMenuPath 标识全局创建方式选择器路由路径。
	ApplicationCreationMenuPath = ApplicationMenuRootPath + "/create"
	// ApplicationBlankCreateMenuPath 标识空白应用创建路由路径。
	ApplicationBlankCreateMenuPath = ApplicationCreationMenuPath + "/blank"
	// ApplicationTemplateCreateMenuPath 标识模板来源创建路由路径。
	ApplicationTemplateCreateMenuPath = ApplicationCreationMenuPath + "/template"
	// ApplicationImportCreateMenuPath 标识导入应用创建路由路径。
	ApplicationImportCreateMenuPath = ApplicationCreationMenuPath + "/import"
	// ApplicationDiscoveryCandidatesMenuPath 标识隐藏的发现候选预览路由路径。
	ApplicationDiscoveryCandidatesMenuPath = ApplicationCreationMenuPath + "/discovery"
)
