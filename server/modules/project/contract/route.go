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
	// ProjectSourcesRoute identifies the project source catalog route fragment.
	ProjectSourcesRoute = "/sources"
	// ProjectDiscoveryCandidatesRoute identifies the bounded discovery-candidate preview route fragment.
	ProjectDiscoveryCandidatesRoute = "/discovery-candidates"
	// ProjectCreateSourceSelectorRoute identifies the source selector route fragment.
	ProjectCreateSourceSelectorRoute = "/create"
	// ProjectManagedRootRoute identifies the managed-root metadata route fragment.
	ProjectManagedRootRoute = "/managed/root"
	// ProjectCreateValidateRoute identifies the managed-create validation route fragment.
	ProjectCreateValidateRoute = "/create/managed/validate"
	// ProjectCreateRoute identifies the managed-create route fragment.
	ProjectCreateRoute = "/create/managed"
	// ProjectCreateGitRoute identifies the future git source route fragment.
	ProjectCreateGitRoute = "/create/git"
	// ProjectCreateTemplateRoute identifies the future template source route fragment.
	ProjectCreateTemplateRoute = "/create/template"
	// ProjectCreateRemoteHostRoute identifies the future remote-host source route fragment.
	ProjectCreateRemoteHostRoute = "/create/remote-host"
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
	// ProjectSourceMenuPath identifies the global source selector route path.
	ProjectSourceMenuPath = "/ops/projects/create"
	// ProjectManagedCreateMenuPath identifies the managed source create route path.
	ProjectManagedCreateMenuPath = "/ops/projects/create/managed"
	// ProjectGitCreateMenuPath identifies the git source create route path.
	ProjectGitCreateMenuPath = "/ops/projects/create/git"
	// ProjectTemplateCreateMenuPath identifies the template source create route path.
	ProjectTemplateCreateMenuPath = "/ops/projects/create/template"
	// ProjectRemoteHostCreateMenuPath identifies the remote-host source create route path.
	ProjectRemoteHostCreateMenuPath = "/ops/projects/create/remote-host"
	// ProjectDiscoveryCandidatesMenuPath identifies the hidden discovery-candidate preview route path.
	ProjectDiscoveryCandidatesMenuPath = "/ops/projects/create/discovery"
)
