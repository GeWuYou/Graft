package contract

// ErrorCode identifies a stable project module error-code contract.
//
// Canonical owner: server/modules/project/contract.
type ErrorCode string

// MessageKey identifies a stable project module localization key contract.
//
// Canonical owner: server/modules/project/contract.
type MessageKey string

// PermissionCode identifies a stable project module permission contract.
//
// Canonical owner: server/modules/project/contract.
type PermissionCode string

// ConfigKey identifies a stable project module system-config contract.
//
// Canonical owner: server/modules/project/contract.
type ConfigKey string

// ConfigMessageKey identifies a stable project system-config localization key contract.
//
// Canonical owner: server/modules/project/contract.
type ConfigMessageKey string

func (c ErrorCode) String() string { return string(c) }

func (c MessageKey) String() string { return string(c) }

func (c PermissionCode) String() string { return string(c) }

func (c ConfigKey) String() string { return string(c) }

func (c ConfigMessageKey) String() string { return string(c) }

const (
	// ProjectWorkspaceAnnotationMaxLength identifies the stable max length for one workspace annotation payload.
	ProjectWorkspaceAnnotationMaxLength = 500
)

const (
	// ProjectMenuTitle identifies the project-management menu title.
	ProjectMenuTitle MessageKey = "menu.project.title"
	// ProjectInvalidID identifies path or payload project identifiers that fail validation.
	ProjectInvalidID ErrorCode = "ops.project.error.invalidProjectId"
	// ProjectInvalidFileID identifies path file identifiers that fail validation.
	ProjectInvalidFileID ErrorCode = "ops.project.error.invalidFileId"
	// ProjectInvalidArgument identifies malformed project request input.
	ProjectInvalidArgument ErrorCode = "ops.project.error.invalidArgument"
	// ProjectInvalidCompose identifies invalid Compose content in a managed create request.
	ProjectInvalidCompose ErrorCode = "ops.project.error.invalidCompose"
	// ProjectWorkspaceUnsafe identifies a managed workspace that cannot be safely reused.
	ProjectWorkspaceUnsafe ErrorCode = "ops.project.error.workspaceUnsafe"
	// ProjectWorkspaceWriteFailed identifies a managed workspace that could not be materialized.
	ProjectWorkspaceWriteFailed ErrorCode = "ops.project.error.workspaceWriteFailed"
	// ProjectApplicationNameRequired identifies managed-create requests without an application name.
	ProjectApplicationNameRequired ErrorCode = "ops.project.error.applicationNameRequired"
	// ProjectInvalidApplicationName identifies application names that are not Compose-safe.
	ProjectInvalidApplicationName ErrorCode = "ops.project.error.invalidApplicationName"
	// ProjectApplicationNameOccupied identifies managed application names already reserved by a live project.
	ProjectApplicationNameOccupied ErrorCode = "ops.project.error.applicationNameOccupied"
	// ProjectInvalidCanonicalProjectName identifies invalid canonical project names.
	ProjectInvalidCanonicalProjectName ErrorCode = "ops.project.error.invalidCanonicalProjectName"
	// ProjectConflict identifies project-registration uniqueness conflicts.
	ProjectConflict ErrorCode = "ops.project.error.conflict"
	// ProjectComposeProjectNameOccupied identifies a runtime-local Compose name collision.
	ProjectComposeProjectNameOccupied ErrorCode = "ops.project.error.composeProjectNameOccupied"
	// ProjectNotFound identifies unknown project records.
	ProjectNotFound ErrorCode = "ops.project.error.notFound"
	// ProjectUnsupportedLifecycle identifies lifecycle requests blocked by project ownership or phase scope.
	ProjectUnsupportedLifecycle ErrorCode = "ops.project.error.unsupportedLifecycle"
	// ProjectRuntimeUnavailable identifies Compose lifecycle requests blocked by an unavailable selected Runtime Target.
	ProjectRuntimeUnavailable ErrorCode = "ops.project.error.runtimeUnavailable"
	// ProjectImportValidationFailed identifies invalid Compose import payloads or parse failures.
	ProjectImportValidationFailed ErrorCode = "ops.project.error.importValidationFailed"
	// ProjectManagedRootUnconfigured identifies managed-create flows blocked by missing managed-root authority.
	ProjectManagedRootUnconfigured ErrorCode = "ops.project.error.managedRootUnconfigured"
	// ProjectManagedRootInvalid identifies managed-create flows blocked by invalid managed-root authority.
	ProjectManagedRootInvalid ErrorCode = "ops.project.error.managedRootInvalid"
	// ProjectManagedFlowUnsupported identifies flows blocked because the project is not a managed project.
	ProjectManagedFlowUnsupported ErrorCode = "ops.project.error.managedFlowUnsupported"
	// ProjectSourceUnsupported identifies source-specific flows that are defined but not implemented in the current phase.
	ProjectSourceUnsupported ErrorCode = "ops.project.error.sourceUnsupported"
	// ProjectDirectoryBrowseForbidden identifies directory browse requests outside the configured authority roots.
	ProjectDirectoryBrowseForbidden ErrorCode = "ops.project.error.directoryBrowseForbidden"
	// ProjectInspectionExpired identifies stale or expired import inspection sessions.
	ProjectInspectionExpired ErrorCode = "ops.project.error.inspectionExpired"
	// ProjectInspectionStale identifies import requests whose inspection snapshot no longer matches the filesystem.
	ProjectInspectionStale ErrorCode = "ops.project.error.inspectionStale"
	// ProjectSavedViewInvalid identifies a saved-view payload that the project consumer rejects.
	ProjectSavedViewInvalid ErrorCode = "ops.project.error.savedViewInvalid"
	// ProjectSavedViewConflict identifies a duplicate saved-view display name on the project list.
	ProjectSavedViewConflict ErrorCode = "ops.project.error.savedViewConflict"
	// ProjectSavedViewNotFound identifies an unknown or foreign saved view.
	ProjectSavedViewNotFound ErrorCode = "ops.project.error.savedViewNotFound"
)

const (
	// ProjectImportValidated identifies a successful import validation response.
	ProjectImportValidated MessageKey = "ops.project.import.validated"
	// ProjectImported identifies a successful import-and-register response.
	ProjectImported MessageKey = "ops.project.import.completed"
	// ProjectRefreshCompleted identifies a successful static refresh response.
	ProjectRefreshCompleted MessageKey = "ops.project.refresh.completed"
	// ProjectLifecycleAccepted identifies an accepted future lifecycle execution response.
	ProjectLifecycleAccepted MessageKey = "ops.project.lifecycle.accepted"
	// ProjectLifecycleBlocked identifies a guarded lifecycle response.
	ProjectLifecycleBlocked MessageKey = "ops.project.lifecycle.blocked"
	// ProjectUpCompleted identifies a successful compose up response.
	ProjectUpCompleted MessageKey = "ops.project.up.completed"
	// ProjectStopCompleted identifies a successful compose stop response.
	ProjectStopCompleted MessageKey = "ops.project.stop.completed"
	// ProjectRestartCompleted identifies a successful compose restart response.
	ProjectRestartCompleted MessageKey = "ops.project.restart.completed"
	// ProjectRedeployCompleted identifies a successful compose redeploy response.
	ProjectRedeployCompleted MessageKey = "ops.project.redeploy.completed"
	// ProjectUnregisterCompleted identifies a successful unregister response.
	ProjectUnregisterCompleted MessageKey = "ops.project.unregister.completed"
	// ProjectDestroyCompleted identifies a successful guarded destroy response.
	ProjectDestroyCompleted MessageKey = "ops.project.destroy.completed"
	// ProjectManagedCreateValidated identifies a successful managed-create validation response.
	ProjectManagedCreateValidated MessageKey = "ops.project.create.validated"
	// ProjectManagedCreateAccepted identifies an accepted managed-create response.
	ProjectManagedCreateAccepted MessageKey = "ops.project.create.accepted"
	// ProjectDeployCompleted identifies a successful managed configuration deploy response.
	ProjectDeployCompleted MessageKey = "ops.project.deploy.completed"
	// ProjectDiscoveryCandidatesReady identifies a successful discovery-candidate preview response.
	ProjectDiscoveryCandidatesReady MessageKey = "ops.project.discovery.candidates.ready"
	// ProjectDirectorySourcesReady identifies a successful import directory-source listing response.
	ProjectDirectorySourcesReady MessageKey = "ops.project.import.directorySources.ready"
	// ProjectDirectoryBrowseReady identifies a successful import directory browse response.
	ProjectDirectoryBrowseReady MessageKey = "ops.project.import.directories.ready"
	// ProjectImportInspected identifies a successful import inspection response.
	ProjectImportInspected MessageKey = "ops.project.import.inspected"
)

const (
	// ProjectViewPermission identifies read access to project registry and readonly detail surfaces.
	ProjectViewPermission PermissionCode = "ops.project.view"
	// ProjectImportPermission identifies import validate and import registration access.
	ProjectImportPermission PermissionCode = "ops.project.import"
	// ProjectRefreshPermission identifies static refresh access.
	ProjectRefreshPermission PermissionCode = "ops.project.refresh"
	// ProjectLifecyclePermission identifies lifecycle action access.
	ProjectLifecyclePermission PermissionCode = "ops.project.lifecycle"
	// ProjectDestroyPermission identifies unregister and destroy access.
	ProjectDestroyPermission PermissionCode = "ops.project.destroy"
	// ProjectCreatePermission identifies managed-create contract and future create execution access.
	ProjectCreatePermission PermissionCode = "ops.project.create"
	// ProjectCreationMethodViewPermission identifies access to the project creation-method catalog.
	ProjectCreationMethodViewPermission PermissionCode = "ops.project.creation-method.view"
	// ProjectDiscoveryViewPermission identifies access to bounded directory scan and auto-discovery candidate previews.
	ProjectDiscoveryViewPermission PermissionCode = "ops.project.discovery.view"
	// ProjectDeployPermission identifies managed configuration diff, validate, and deploy access.
	ProjectDeployPermission PermissionCode = "ops.project.deploy"
)

const (
	// ApplicationRootDirectoryConfig stores the canonical application root directory.
	ApplicationRootDirectoryConfig ConfigKey = "ops.application.root_directory"
	// ProjectImportAllowedRootsConfig stores operator-allowlisted browse roots for import flows.
	ProjectImportAllowedRootsConfig ConfigKey = "ops.project.import.allowed_roots"
	// ProjectWorkspaceHiddenDirectoriesConfig stores default-hidden heavy directories for the configuration workspace tree.
	ProjectWorkspaceHiddenDirectoriesConfig ConfigKey = "ops.project.workspace.hidden_directories"
	// ProjectWorkspaceFileTooltipRulesConfig stores ordered default tooltip rules matched against file basenames.
	ProjectWorkspaceFileTooltipRulesConfig ConfigKey = "ops.project.workspace.file_tooltip_rules"
	// ProjectWorkspaceDirectoryTooltipRulesConfig stores ordered default tooltip rules matched against directory basenames.
	ProjectWorkspaceDirectoryTooltipRulesConfig ConfigKey = "ops.project.workspace.directory_tooltip_rules"
	// ProjectBlankCreatePrefillDefaultTemplateConfig controls whether blank creation starts from templates/default.
	ProjectBlankCreatePrefillDefaultTemplateConfig ConfigKey = "ops.project.create.blank_prefill_default_template"
)

const (
	// ApplicationRootDirectoryConfigTitle identifies the application-root config title localization key.
	ApplicationRootDirectoryConfigTitle ConfigMessageKey = "systemConfig.project.ops.application.root_directory.title"
	// ApplicationRootDirectoryConfigDescription identifies the application-root config description localization key.
	ApplicationRootDirectoryConfigDescription ConfigMessageKey = "systemConfig.project.ops.application.root_directory.description"
	// ApplicationCreateConfigGroupTitle identifies the application-create config group title localization key.
	ApplicationCreateConfigGroupTitle ConfigMessageKey = "systemConfig.groups.ops.application.create"
	// ApplicationCreateConfigGroupDescription identifies the application-create config group description localization key.
	ApplicationCreateConfigGroupDescription ConfigMessageKey = "systemConfig.groups.ops.application.create.description"
	// ProjectImportAllowedRootsConfigTitle identifies the allowlisted import roots config title localization key.
	ProjectImportAllowedRootsConfigTitle ConfigMessageKey = "systemConfig.project.ops.project.import.allowed_roots.title"
	// ProjectImportAllowedRootsConfigDescription identifies the allowlisted import roots config description localization key.
	ProjectImportAllowedRootsConfigDescription ConfigMessageKey = "systemConfig.project.ops.project.import.allowed_roots.description"
	// ProjectImportConfigGroupTitle identifies the project-import config group title localization key.
	ProjectImportConfigGroupTitle ConfigMessageKey = "systemConfig.groups.ops.project.import"
	// ProjectImportConfigGroupDescription identifies the project-import config group description localization key.
	ProjectImportConfigGroupDescription ConfigMessageKey = "systemConfig.groups.ops.project.import.description"
	// ProjectWorkspaceHiddenDirectoriesConfigTitle identifies the workspace hidden-directory config title localization key.
	ProjectWorkspaceHiddenDirectoriesConfigTitle ConfigMessageKey = "systemConfig.project.ops.project.workspace.hidden_directories.title"
	// ProjectWorkspaceHiddenDirectoriesConfigDescription identifies the workspace hidden-directory config description localization key.
	ProjectWorkspaceHiddenDirectoriesConfigDescription ConfigMessageKey = "systemConfig.project.ops.project.workspace.hidden_directories.description"
	// ProjectWorkspaceConfigGroupTitle identifies the workspace config group title localization key.
	ProjectWorkspaceConfigGroupTitle ConfigMessageKey = "systemConfig.groups.ops.project.workspace"
	// ProjectWorkspaceConfigGroupDescription identifies the workspace config group description localization key.
	ProjectWorkspaceConfigGroupDescription ConfigMessageKey = "systemConfig.groups.ops.project.workspace.description"
	// ProjectWorkspaceHiddenDirectoriesPlaceholder identifies the workspace hidden-directory editor placeholder key.
	ProjectWorkspaceHiddenDirectoriesPlaceholder ConfigMessageKey = "systemConfig.project.ops.project.workspace.hidden_directories.placeholder"
	// ProjectWorkspaceFileTooltipRulesConfigTitle identifies the workspace file-tooltip rules config title localization key.
	ProjectWorkspaceFileTooltipRulesConfigTitle ConfigMessageKey = "systemConfig.project.ops.project.workspace.file_tooltip_rules.title"
	// ProjectWorkspaceFileTooltipRulesConfigDescription identifies the workspace file-tooltip rules config description localization key.
	ProjectWorkspaceFileTooltipRulesConfigDescription ConfigMessageKey = "systemConfig.project.ops.project.workspace.file_tooltip_rules.description"
	// ProjectWorkspaceDirectoryTooltipRulesConfigTitle identifies the workspace directory-tooltip rules config title localization key.
	ProjectWorkspaceDirectoryTooltipRulesConfigTitle ConfigMessageKey = "systemConfig.project.ops.project.workspace.directory_tooltip_rules.title"
	// ProjectWorkspaceDirectoryTooltipRulesConfigDescription identifies the workspace directory-tooltip rules config description localization key.
	ProjectWorkspaceDirectoryTooltipRulesConfigDescription ConfigMessageKey = "systemConfig.project.ops.project.workspace.directory_tooltip_rules.description"
	// ProjectBlankCreatePrefillDefaultTemplateConfigTitle identifies the blank-create template prefill config title.
	ProjectBlankCreatePrefillDefaultTemplateConfigTitle ConfigMessageKey = "systemConfig.project.ops.project.create.blank_prefill_default_template.title"
	// ProjectBlankCreatePrefillDefaultTemplateConfigDescription identifies the blank-create template prefill config description.
	ProjectBlankCreatePrefillDefaultTemplateConfigDescription ConfigMessageKey = "systemConfig.project.ops.project.create.blank_prefill_default_template.description"
)
