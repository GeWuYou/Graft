package contract

// ErrorCode 表示项目模块对外稳定的错误码契约。
//
// 规范 owner：server/modules/project/contract。
type ErrorCode string

// MessageKey 表示项目模块对外稳定的本地化消息键契约。
//
// 规范 owner：server/modules/project/contract。
type MessageKey string

// PermissionCode 表示项目模块对外稳定的权限码契约。
//
// 规范 owner：server/modules/project/contract。
type PermissionCode string

// ConfigKey 表示项目模块对外稳定的系统配置键契约。
//
// 规范 owner：server/modules/project/contract。
type ConfigKey string

// ConfigMessageKey 表示项目系统配置使用的稳定本地化消息键契约。
//
// 规范 owner：server/modules/project/contract。
type ConfigMessageKey string

// String 返回错误码的稳定线值，供 HTTP 错误响应和本地化查找使用。
func (c ErrorCode) String() string { return string(c) }

// String 返回消息键的稳定线值，供本地化资源查找使用。
func (c MessageKey) String() string { return string(c) }

// String 返回权限码的稳定线值，供权限注册和鉴权匹配使用。
func (c PermissionCode) String() string { return string(c) }

// String 返回配置键的稳定线值，供系统配置注册和读取使用。
func (c ConfigKey) String() string { return string(c) }

// String 返回配置消息键的稳定线值，供系统配置本地化资源查找使用。
func (c ConfigMessageKey) String() string { return string(c) }

const (
	// ApplicationWorkspaceAnnotationMaxLength 限制单条工作区注释长度，避免注释载荷无界增长。
	ApplicationWorkspaceAnnotationMaxLength = 500
)

const (
	// ApplicationMenuTitle 标识应用管理菜单标题的本地化键。
	ApplicationMenuTitle MessageKey = "menu.application.title"
	// ApplicationTemplateMenuTitle 标识应用模板管理菜单标题的本地化键。
	ApplicationTemplateMenuTitle MessageKey = "menu.application.templates.title"
	// ApplicationInvalidID 标识路径或载荷中的应用公开标识未通过校验。
	ApplicationInvalidID ErrorCode = "ops.application.error.invalidApplicationId"
	// ApplicationInvalidFileID 标识路径中的应用文件标识未通过校验。
	ApplicationInvalidFileID ErrorCode = "ops.application.error.invalidFileId"
	// ApplicationInvalidArgument 标识应用请求参数格式无效。
	ApplicationInvalidArgument ErrorCode = "ops.application.error.invalidArgument"
	// ApplicationInvalidCompose 标识受管创建请求中的 Compose 内容无效。
	ApplicationInvalidCompose ErrorCode = "ops.application.error.invalidCompose"
	// ApplicationWorkspaceUnsafe 标识受管工作区不满足安全复用条件。
	ApplicationWorkspaceUnsafe ErrorCode = "ops.application.error.workspaceUnsafe"
	// ApplicationWorkspaceWriteFailed 标识受管工作区写入失败。
	ApplicationWorkspaceWriteFailed ErrorCode = "ops.application.error.workspaceWriteFailed"
	// ApplicationNameRequired 标识受管创建请求缺少应用名称。
	ApplicationNameRequired ErrorCode = "ops.application.error.applicationNameRequired"
	// ApplicationInvalidApplicationName 标识应用名称不满足 Compose 安全命名约束。
	ApplicationInvalidApplicationName ErrorCode = "ops.application.error.invalidApplicationName"
	// ApplicationNameOccupied 标识受管应用名称已被存活应用占用。
	ApplicationNameOccupied ErrorCode = "ops.application.error.applicationNameOccupied"
	// ApplicationInvalidComposeProjectName 标识 Compose 运行时名称无效。
	ApplicationInvalidComposeProjectName ErrorCode = "ops.application.error.invalidComposeProjectName"
	// ApplicationConflict 标识应用注册记录违反唯一性约束。
	ApplicationConflict ErrorCode = "ops.application.error.conflict"
	// ApplicationComposeProjectNameOccupied 标识同一运行时目标内的 Compose 名称冲突。
	ApplicationComposeProjectNameOccupied ErrorCode = "ops.application.error.composeProjectNameOccupied"
	// ApplicationNotFound 标识未找到存活应用记录。
	ApplicationNotFound ErrorCode = "ops.application.error.notFound"
	// ApplicationUnsupportedLifecycle 标识应用所有权或当前阶段不允许执行生命周期请求。
	ApplicationUnsupportedLifecycle ErrorCode = "ops.application.error.unsupportedLifecycle"
	// ApplicationRuntimeUnavailable 标识选定 Runtime Target 不可用，无法执行 Compose 生命周期请求。
	ApplicationRuntimeUnavailable ErrorCode = "ops.application.error.runtimeUnavailable"
	// ApplicationImportValidationFailed 标识 Compose 导入载荷无效或解析失败。
	ApplicationImportValidationFailed ErrorCode = "ops.application.error.importValidationFailed"
	// ApplicationManagedRootUnconfigured 标识受管根目录未配置，无法执行受管创建。
	ApplicationManagedRootUnconfigured ErrorCode = "ops.application.error.managedRootUnconfigured"
	// ApplicationManagedRootInvalid 标识受管根目录配置无效，无法执行受管创建。
	ApplicationManagedRootInvalid ErrorCode = "ops.application.error.managedRootInvalid"
	// ApplicationManagedFlowUnsupported 标识当前应用不是受管应用，无法执行对应流程。
	ApplicationManagedFlowUnsupported ErrorCode = "ops.application.error.managedFlowUnsupported"
	// ApplicationSourceUnsupported 标识当前阶段尚未实现给定来源类型的流程。
	ApplicationSourceUnsupported ErrorCode = "ops.application.error.sourceUnsupported"
	// ApplicationDirectoryBrowseForbidden 标识目录浏览请求超出配置的权威根目录。
	ApplicationDirectoryBrowseForbidden ErrorCode = "ops.application.error.directoryBrowseForbidden"
	// ApplicationInspectionExpired 标识导入检查会话已过期。
	ApplicationInspectionExpired ErrorCode = "ops.application.error.inspectionExpired"
	// ApplicationInspectionStale 标识导入检查快照与当前文件系统状态不再一致。
	ApplicationInspectionStale ErrorCode = "ops.application.error.inspectionStale"
	// ApplicationSavedViewInvalid 标识应用列表保存视图载荷无效。
	ApplicationSavedViewInvalid ErrorCode = "ops.application.error.savedViewInvalid"
	// ApplicationSavedViewConflict 标识应用列表保存视图名称重复。
	ApplicationSavedViewConflict ErrorCode = "ops.application.error.savedViewConflict"
	// ApplicationSavedViewNotFound 标识保存视图不存在或不属于当前用户。
	ApplicationSavedViewNotFound ErrorCode = "ops.application.error.savedViewNotFound"
)

const (
	// ApplicationImportValidated 标识应用导入校验成功。
	ApplicationImportValidated MessageKey = "ops.application.import.validated"
	// ApplicationImported 标识应用导入并注册成功。
	ApplicationImported MessageKey = "ops.application.import.completed"
	// ApplicationRefreshCompleted 标识应用静态刷新成功。
	ApplicationRefreshCompleted MessageKey = "ops.application.refresh.completed"
	// ApplicationLifecycleAccepted 标识应用生命周期执行请求已接受。
	ApplicationLifecycleAccepted MessageKey = "ops.application.lifecycle.accepted"
	// ApplicationLifecycleBlocked 标识应用生命周期请求被守卫规则阻止。
	ApplicationLifecycleBlocked MessageKey = "ops.application.lifecycle.blocked"
	// ApplicationUpCompleted 标识 Compose 启动成功。
	ApplicationUpCompleted MessageKey = "ops.application.up.completed"
	// ApplicationStopCompleted 标识 Compose 停止成功。
	ApplicationStopCompleted MessageKey = "ops.application.stop.completed"
	// ApplicationRestartCompleted 标识 Compose 重启成功。
	ApplicationRestartCompleted MessageKey = "ops.application.restart.completed"
	// ApplicationRedeployCompleted 标识 Compose 重部署成功。
	ApplicationRedeployCompleted MessageKey = "ops.application.redeploy.completed"
	// ApplicationUnregisterCompleted 标识应用注销成功。
	ApplicationUnregisterCompleted MessageKey = "ops.application.unregister.completed"
	// ApplicationDestroyCompleted 标识受保护的应用销毁成功。
	ApplicationDestroyCompleted MessageKey = "ops.application.destroy.completed"
	// ApplicationManagedCreateValidated 标识受管应用创建校验成功。
	ApplicationManagedCreateValidated MessageKey = "ops.application.create.validated"
	// ApplicationManagedCreateAccepted 标识受管应用创建请求已接受。
	ApplicationManagedCreateAccepted MessageKey = "ops.application.create.accepted"
	// ApplicationDiscoveryCandidatesReady 标识应用发现候选预览已就绪。
	ApplicationDiscoveryCandidatesReady MessageKey = "ops.application.discovery.candidates.ready"
	// ApplicationDirectorySourcesReady 标识导入目录来源列表已就绪。
	ApplicationDirectorySourcesReady MessageKey = "ops.application.import.directorySources.ready"
	// ApplicationDirectoryBrowseReady 标识导入目录浏览结果已就绪。
	ApplicationDirectoryBrowseReady MessageKey = "ops.application.import.directories.ready"
	// ApplicationImportInspected 标识应用导入检查成功。
	ApplicationImportInspected MessageKey = "ops.application.import.inspected"
)

const (
	// ApplicationViewPermission 允许读取应用注册表及只读详情。
	ApplicationViewPermission PermissionCode = "ops.application.view"
	// ApplicationImportPermission 允许校验并注册导入应用。
	ApplicationImportPermission PermissionCode = "ops.application.import"
	// ApplicationRefreshPermission 允许刷新应用静态配置投影。
	ApplicationRefreshPermission PermissionCode = "ops.application.refresh"
	// ApplicationLifecyclePermission 允许执行应用生命周期动作。
	ApplicationLifecyclePermission PermissionCode = "ops.application.lifecycle"
	// ApplicationDestroyPermission 允许注销或销毁应用。
	ApplicationDestroyPermission PermissionCode = "ops.application.destroy"
	// ApplicationCreatePermission 允许校验并执行受管应用创建。
	ApplicationCreatePermission PermissionCode = "ops.application.create"
	// ApplicationCreationMethodViewPermission 允许读取应用创建方式目录。
	ApplicationCreationMethodViewPermission PermissionCode = "ops.application.creation-method.view"
	// ApplicationDiscoveryViewPermission 允许执行有界目录扫描并查看发现候选。
	ApplicationDiscoveryViewPermission PermissionCode = "ops.application.discovery.view"
	// ApplicationTemplateManagePermission 允许维护 Application 模板草稿与归档状态。
	ApplicationTemplateManagePermission PermissionCode = "ops.application.template.manage"
	// ApplicationTemplatePublishPermission 允许发布不可变 Application 模板版本。
	ApplicationTemplatePublishPermission PermissionCode = "ops.application.template.publish"
	// ApplicationDeployPermission 允许比较、校验并部署受管应用配置。
	ApplicationDeployPermission PermissionCode = "ops.application.deploy"
)

const (
	// ApplicationRootDirectoryConfig 保存受管应用工作区的规范根目录。
	ApplicationRootDirectoryConfig ConfigKey = "ops.application.root_directory"
	// ApplicationImportAllowedRootsConfig 保存操作员允许导入流程浏览的根目录白名单。
	ApplicationImportAllowedRootsConfig ConfigKey = "ops.application.import.allowed_roots"
	// ApplicationWorkspaceHiddenDirectoriesConfig 保存配置工作区文件树默认隐藏的大目录名称。
	ApplicationWorkspaceHiddenDirectoriesConfig ConfigKey = "ops.application.workspace.hidden_directories"
	// ApplicationWorkspaceFileTooltipRulesConfig 保存按文件基础名匹配的有序默认提示规则。
	ApplicationWorkspaceFileTooltipRulesConfig ConfigKey = "ops.application.workspace.file_tooltip_rules"
	// ApplicationWorkspaceDirectoryTooltipRulesConfig 保存按目录基础名匹配的有序默认提示规则。
	ApplicationWorkspaceDirectoryTooltipRulesConfig ConfigKey = "ops.application.workspace.directory_tooltip_rules"
)

const (
	// ApplicationRootDirectoryConfigTitle 标识应用根目录配置标题的本地化键。
	ApplicationRootDirectoryConfigTitle ConfigMessageKey = "systemConfig.application.ops.application.root_directory.title"
	// ApplicationRootDirectoryConfigDescription 标识应用根目录配置说明的本地化键。
	ApplicationRootDirectoryConfigDescription ConfigMessageKey = "systemConfig.application.ops.application.root_directory.description"
	// ApplicationCreateConfigGroupTitle 标识应用创建配置组标题的本地化键。
	ApplicationCreateConfigGroupTitle ConfigMessageKey = "systemConfig.groups.ops.application.create"
	// ApplicationCreateConfigGroupDescription 标识应用创建配置组说明的本地化键。
	ApplicationCreateConfigGroupDescription ConfigMessageKey = "systemConfig.groups.ops.application.create.description"
	// ApplicationImportAllowedRootsConfigTitle 标识导入根目录白名单配置标题的本地化键。
	ApplicationImportAllowedRootsConfigTitle ConfigMessageKey = "systemConfig.application.ops.application.import.allowed_roots.title"
	// ApplicationImportAllowedRootsConfigDescription 标识导入根目录白名单配置说明的本地化键。
	ApplicationImportAllowedRootsConfigDescription ConfigMessageKey = "systemConfig.application.ops.application.import.allowed_roots.description"
	// ApplicationImportConfigGroupTitle 标识应用导入配置组标题的本地化键。
	ApplicationImportConfigGroupTitle ConfigMessageKey = "systemConfig.groups.ops.application.import"
	// ApplicationImportConfigGroupDescription 标识应用导入配置组说明的本地化键。
	ApplicationImportConfigGroupDescription ConfigMessageKey = "systemConfig.groups.ops.application.import.description"
	// ApplicationWorkspaceHiddenDirectoriesConfigTitle 标识工作区隐藏目录配置标题的本地化键。
	ApplicationWorkspaceHiddenDirectoriesConfigTitle ConfigMessageKey = "systemConfig.application.ops.application.workspace.hidden_directories.title"
	// ApplicationWorkspaceHiddenDirectoriesConfigDescription 标识工作区隐藏目录配置说明的本地化键。
	ApplicationWorkspaceHiddenDirectoriesConfigDescription ConfigMessageKey = "systemConfig.application.ops.application.workspace.hidden_directories.description"
	// ApplicationWorkspaceConfigGroupTitle 标识工作区配置组标题的本地化键。
	ApplicationWorkspaceConfigGroupTitle ConfigMessageKey = "systemConfig.groups.ops.application.workspace"
	// ApplicationWorkspaceConfigGroupDescription 标识工作区配置组说明的本地化键。
	ApplicationWorkspaceConfigGroupDescription ConfigMessageKey = "systemConfig.groups.ops.application.workspace.description"
	// ApplicationWorkspaceHiddenDirectoriesPlaceholder 标识工作区隐藏目录编辑器占位文案的本地化键。
	ApplicationWorkspaceHiddenDirectoriesPlaceholder ConfigMessageKey = "systemConfig.application.ops.application.workspace.hidden_directories.placeholder"
	// ApplicationWorkspaceFileTooltipRulesConfigTitle 标识工作区文件提示规则配置标题的本地化键。
	ApplicationWorkspaceFileTooltipRulesConfigTitle ConfigMessageKey = "systemConfig.application.ops.application.workspace.file_tooltip_rules.title"
	// ApplicationWorkspaceFileTooltipRulesConfigDescription 标识工作区文件提示规则配置说明的本地化键。
	ApplicationWorkspaceFileTooltipRulesConfigDescription ConfigMessageKey = "systemConfig.application.ops.application.workspace.file_tooltip_rules.description"
	// ApplicationWorkspaceDirectoryTooltipRulesConfigTitle 标识工作区目录提示规则配置标题的本地化键。
	ApplicationWorkspaceDirectoryTooltipRulesConfigTitle ConfigMessageKey = "systemConfig.application.ops.application.workspace.directory_tooltip_rules.title"
	// ApplicationWorkspaceDirectoryTooltipRulesConfigDescription 标识工作区目录提示规则配置说明的本地化键。
	ApplicationWorkspaceDirectoryTooltipRulesConfigDescription ConfigMessageKey = "systemConfig.application.ops.application.workspace.directory_tooltip_rules.description"
)
