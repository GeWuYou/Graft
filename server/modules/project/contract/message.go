package contract

// ErrorCode identifies a stable project module error-code contract.
//
// Canonical owner: server/modules/project/contract.
type ErrorCode string

// MessageKey identifies a stable project module localization key contract.
//
// Canonical owner: server/modules/project/contract.
type MessageKey string

func (c ErrorCode) String() string { return string(c) }

func (c MessageKey) String() string { return string(c) }

const (
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
)
