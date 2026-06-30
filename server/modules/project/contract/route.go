package contract

const (
	// ProjectAPIGroup is the API route group for Compose project management.
	ProjectAPIGroup = "/ops/projects"
	// ProjectCollectionRoute identifies the project list route fragment.
	ProjectCollectionRoute = ""
	// ProjectDetailRoute identifies one project summary route fragment.
	ProjectDetailRoute = "/:id"
	// ProjectServicesRoute identifies the project services aggregation route fragment.
	ProjectServicesRoute = "/:id/services"
	// ProjectConfigurationRoute identifies the project configuration metadata route fragment.
	ProjectConfigurationRoute = "/:id/configuration"
	// ProjectConfigurationPreviewRoute identifies the normalized configuration preview route fragment.
	ProjectConfigurationPreviewRoute = "/:id/configuration/preview"
	// ProjectConfigurationFileRoute identifies the single-file content route fragment.
	ProjectConfigurationFileRoute = "/:id/configuration/files/:fileId"
	// ProjectConfigurationDiffRoute identifies the managed configuration draft diff route fragment.
	ProjectConfigurationDiffRoute = "/:id/configuration/diff"
	// ProjectConfigurationValidateRoute identifies the managed configuration draft validate route fragment.
	ProjectConfigurationValidateRoute = "/:id/configuration/validate"
	// ProjectImportValidateRoute identifies the import validation route fragment.
	ProjectImportValidateRoute = "/import/validate"
	// ProjectImportRoute identifies the import-and-register route fragment.
	ProjectImportRoute = "/import"
	// ProjectManagedRootRoute identifies the managed-root metadata route fragment.
	ProjectManagedRootRoute = "/managed-root"
	// ProjectCreateValidateRoute identifies the managed-create validation route fragment.
	ProjectCreateValidateRoute = "/create/validate"
	// ProjectCreateRoute identifies the managed-create route fragment.
	ProjectCreateRoute = "/create"
	// ProjectRefreshRoute identifies the static refresh route fragment.
	ProjectRefreshRoute = "/:id/refresh"
	// ProjectUpRoute identifies the compose up route fragment.
	ProjectUpRoute = "/:id/up"
	// ProjectDownRoute identifies the compose down route fragment.
	ProjectDownRoute = "/:id/down"
	// ProjectRestartRoute identifies the compose restart route fragment.
	ProjectRestartRoute = "/:id/restart"
	// ProjectUnregisterRoute identifies the unregister route fragment.
	ProjectUnregisterRoute = "/:id/unregister"
	// ProjectDestroyRoute identifies the guarded destroy route fragment.
	ProjectDestroyRoute = "/:id/destroy"
	// ProjectDeployRoute identifies the managed configuration deploy route fragment.
	ProjectDeployRoute = "/:id/deploy"
	// ProjectMenuRootPath identifies the web menu root path for operations.
	ProjectMenuRootPath = "/ops"
	// ProjectMenuPath identifies the canonical web menu path for Compose project management.
	ProjectMenuPath = "/ops/projects"
)
