package contract

// SourceKind identifies the stable project source contract.
//
// Canonical owner: server/modules/project/contract.
type SourceKind string

// HostScope identifies the stable host-scope contract for project registration.
type HostScope string

// OwnershipMode identifies the stable ownership contract for project lifecycle guards.
type OwnershipMode string

// ManagedRootStatus identifies the stable managed-root readiness contract for managed create.
type ManagedRootStatus string

// DriftStatus identifies the stable drift-state contract for project configuration checks.
type DriftStatus string

// CanonicalProjectNameSource identifies the stable source contract for canonical project names.
type CanonicalProjectNameSource string

// FileKind identifies the stable project file kind contract.
type FileKind string

// FileRole identifies the stable project file role contract.
type FileRole string

// LifecycleStrategyKind identifies the stable project lifecycle execution strategy contract.
type LifecycleStrategyKind string

// LifecycleReviewStatus identifies the stable lifecycle configuration review-state contract.
type LifecycleReviewStatus string

const (
	// SourceKindImported marks an externally created Compose project imported into Graft.
	SourceKindImported SourceKind = "imported"
	// SourceKindManaged marks a future Graft-created managed project.
	SourceKindManaged SourceKind = "managed"
	// SourceKindTemplate marks a future template-derived project source.
	SourceKindTemplate SourceKind = "template"

	// HostScopeLocal marks the Phase 1 local-host-only project scope.
	HostScopeLocal HostScope = "local"

	// OwnershipModeExternal marks a project backed by an external working directory.
	OwnershipModeExternal OwnershipMode = "external"
	// OwnershipModeManagedRootDedicated marks a project whose working directory is owned by a managed root.
	OwnershipModeManagedRootDedicated OwnershipMode = "managed-root-dedicated"

	// ManagedRootStatusUnconfigured marks a missing managed-root system configuration.
	ManagedRootStatusUnconfigured ManagedRootStatus = "unconfigured"
	// ManagedRootStatusReady marks a managed-root configuration that passed bounded authority checks.
	ManagedRootStatusReady ManagedRootStatus = "ready"
	// ManagedRootStatusInvalid marks a managed-root configuration that failed bounded authority checks.
	ManagedRootStatusInvalid ManagedRootStatus = "invalid"

	// DriftStatusUnknown marks a project whose drift state is not yet known.
	DriftStatusUnknown DriftStatus = "unknown"
	// DriftStatusClean marks a project whose current observed files match the last successful snapshot.
	DriftStatusClean DriftStatus = "clean"
	// DriftStatusChanged marks a project whose observed files differ from the last successful snapshot.
	DriftStatusChanged DriftStatus = "changed"
	// DriftStatusMissing marks a project whose required files are missing.
	DriftStatusMissing DriftStatus = "missing"

	// CanonicalProjectNameSourceComputed marks a runtime identity computed from Compose inputs.
	CanonicalProjectNameSourceComputed CanonicalProjectNameSource = "computed"
	// CanonicalProjectNameSourceOverride marks a runtime identity explicitly overridden at import time.
	CanonicalProjectNameSourceOverride CanonicalProjectNameSource = "override"

	// FileKindCompose marks a Compose definition file.
	FileKindCompose FileKind = "compose"
	// FileKindEnv marks an environment file.
	FileKindEnv FileKind = "env"

	// FileRolePrimary marks the primary Compose definition file.
	FileRolePrimary FileRole = "primary"
	// FileRoleOverride marks an ordered Compose override file.
	FileRoleOverride FileRole = "override"
	// FileRoleEnv marks an environment file consumed during parsing.
	FileRoleEnv FileRole = "env"

	// LifecycleStrategyKindStandard identifies the project-owned standard compose lifecycle strategy.
	LifecycleStrategyKindStandard LifecycleStrategyKind = "standard"

	// LifecycleReviewStatusReviewRequired identifies imported or changed lifecycle configs that still need operator confirmation.
	LifecycleReviewStatusReviewRequired LifecycleReviewStatus = "review_required"
	// LifecycleReviewStatusConfirmed identifies lifecycle configs that can execute compose actions.
	LifecycleReviewStatusConfirmed LifecycleReviewStatus = "confirmed"
)

// String returns the wire-format value.
func (v SourceKind) String() string { return string(v) }

// String returns the wire-format value.
func (v HostScope) String() string { return string(v) }

// String returns the wire-format value.
func (v OwnershipMode) String() string { return string(v) }

// String returns the wire-format value.
func (v ManagedRootStatus) String() string { return string(v) }

// String returns the wire-format value.
func (v DriftStatus) String() string { return string(v) }

// String returns the wire-format value.
func (v CanonicalProjectNameSource) String() string { return string(v) }

// String returns the wire-format value.
func (v FileKind) String() string { return string(v) }

// String returns the wire-format value.
func (v FileRole) String() string { return string(v) }

// String returns the wire-format value.
func (v LifecycleStrategyKind) String() string { return string(v) }

// String returns the wire-format value.
func (v LifecycleReviewStatus) String() string { return string(v) }
