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
	// ProjectMenuTitle identifies the project-management menu title.
	ProjectMenuTitle MessageKey = "menu.ops.project.title"
	// ProjectSourceMenuTitle identifies the hidden source-selector route title.
	ProjectSourceMenuTitle MessageKey = "menu.ops.project.source.title"
	// ProjectInvalidID identifies path or payload project identifiers that fail validation.
	ProjectInvalidID ErrorCode = "ops.project.error.invalidProjectId"
	// ProjectInvalidFileID identifies path file identifiers that fail validation.
	ProjectInvalidFileID ErrorCode = "ops.project.error.invalidFileId"
	// ProjectInvalidArgument identifies malformed project request input.
	ProjectInvalidArgument ErrorCode = "ops.project.error.invalidArgument"
	// ProjectConflict identifies project-registration uniqueness conflicts.
	ProjectConflict ErrorCode = "ops.project.error.conflict"
	// ProjectNotFound identifies unknown project records.
	ProjectNotFound ErrorCode = "ops.project.error.notFound"
	// ProjectUnsupportedLifecycle identifies lifecycle requests blocked by project ownership or phase scope.
	ProjectUnsupportedLifecycle ErrorCode = "ops.project.error.unsupportedLifecycle"
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
	// ProjectDownCompleted identifies a successful compose down response.
	ProjectDownCompleted MessageKey = "ops.project.down.completed"
	// ProjectRestartCompleted identifies a successful compose restart response.
	ProjectRestartCompleted MessageKey = "ops.project.restart.completed"
	// ProjectUnregisterCompleted identifies a successful unregister response.
	ProjectUnregisterCompleted MessageKey = "ops.project.unregister.completed"
	// ProjectDestroyCompleted identifies a successful guarded destroy response.
	ProjectDestroyCompleted MessageKey = "ops.project.destroy.completed"
	// ProjectManagedCreateValidated identifies a successful managed-create validation response.
	ProjectManagedCreateValidated MessageKey = "ops.project.create.validated"
	// ProjectManagedCreateAccepted identifies an accepted managed-create response.
	ProjectManagedCreateAccepted MessageKey = "ops.project.create.accepted"
	// ProjectConfigurationValidated identifies a successful managed configuration validation response.
	ProjectConfigurationValidated MessageKey = "ops.project.configuration.validated"
	// ProjectDeployCompleted identifies a successful managed configuration deploy response.
	ProjectDeployCompleted MessageKey = "ops.project.deploy.completed"
	// ProjectSourceCatalogReady identifies a successful project source catalog response.
	ProjectSourceCatalogReady MessageKey = "ops.project.source.catalog.ready"
	// ProjectDiscoveryCandidatesReady identifies a successful discovery-candidate preview response.
	ProjectDiscoveryCandidatesReady MessageKey = "ops.project.discovery.candidates.ready"
	// ProjectSourceManagedDescription identifies the managed source catalog description key.
	ProjectSourceManagedDescription MessageKey = "ops.project.source.managed.description"
	// ProjectSourceGitDescription identifies the git source catalog description key.
	ProjectSourceGitDescription MessageKey = "ops.project.source.git.description"
	// ProjectSourceTemplateDescription identifies the template source catalog description key.
	ProjectSourceTemplateDescription MessageKey = "ops.project.source.template.description"
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
	// ProjectSourceViewPermission identifies access to the Phase 3 source selector and source catalog boundary.
	ProjectSourceViewPermission PermissionCode = "ops.project.source.view"
	// ProjectDiscoveryViewPermission identifies access to bounded directory scan and auto-discovery candidate previews.
	ProjectDiscoveryViewPermission PermissionCode = "ops.project.discovery.view"
	// ProjectDeployPermission identifies managed configuration diff, validate, and deploy access.
	ProjectDeployPermission PermissionCode = "ops.project.deploy"
)

const (
	// ProjectManagedRootConfig stores the canonical managed-project root directory.
	ProjectManagedRootConfig ConfigKey = "ops.project.managed.root_directory"
)

const (
	// ProjectManagedRootConfigTitle identifies the managed-root config title localization key.
	ProjectManagedRootConfigTitle ConfigMessageKey = "systemConfig.project.ops.project.managed.root_directory.title"
	// ProjectManagedRootConfigDescription identifies the managed-root config description localization key.
	ProjectManagedRootConfigDescription ConfigMessageKey = "systemConfig.project.ops.project.managed.root_directory.description"
)
