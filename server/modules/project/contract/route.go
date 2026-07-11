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
	// ProjectLogsRoute identifies the project-owned aggregated logs route fragment.
	ProjectLogsRoute = "/:id/logs"
	// ProjectOverviewRoute identifies the project runtime overview route fragment.
	ProjectOverviewRoute = "/:id/overview"
	// ProjectConfigurationRoute identifies the project configuration metadata route fragment.
	ProjectConfigurationRoute = "/:id/configuration"
	// ProjectConfigurationPreviewRoute identifies the normalized configuration preview route fragment.
	ProjectConfigurationPreviewRoute = "/:id/configuration/preview"
	// ProjectWorkspaceFilesRoute identifies the lazy-loaded project-root file tree route fragment.
	ProjectWorkspaceFilesRoute = "/:id/files"
	// ProjectWorkspaceFileContentRoute identifies the path-based project file read/write route fragment.
	ProjectWorkspaceFileContentRoute = "/:id/files/content"
	// ProjectWorkspaceFileAnnotationRoute identifies the path-based workspace annotation write route fragment.
	ProjectWorkspaceFileAnnotationRoute = "/:id/files/annotation"
	// ProjectImportValidateRoute identifies the import validation route fragment.
	ProjectImportValidateRoute = "/import/validate"
	// ProjectImportRuntimeCandidatesRoute identifies the runtime-driven import candidate list route fragment.
	ProjectImportRuntimeCandidatesRoute = "/import/runtime-candidates"
	// ProjectImportRuntimeInspectRoute identifies the runtime-driven import inspection route fragment.
	ProjectImportRuntimeInspectRoute = "/import/runtime-inspect"
	// ProjectImportInspectRoute identifies the import inspection route fragment.
	ProjectImportInspectRoute = "/import/inspect"
	// ProjectImportRoute identifies the import-and-register route fragment.
	ProjectImportRoute = "/import"
	// ProjectImportDirectorySourcesRoute identifies the available import directory source roots route fragment.
	ProjectImportDirectorySourcesRoute = "/import/directory-sources"
	// ProjectImportDirectoriesRoute identifies the root-relative import directory browse route fragment.
	ProjectImportDirectoriesRoute = "/import/directories"
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
	// ProjectCreateGitValidateRoute validates a Git source without materializing it.
	ProjectCreateGitValidateRoute = "/create/git/validate"
	// ProjectCreateGitRoute identifies the future git source route fragment.
	ProjectCreateGitRoute = "/create/git"
	// ProjectCreateTemplateValidateRoute validates a bundled template source without materializing it.
	ProjectCreateTemplateValidateRoute = "/create/template/validate"
	// ProjectCreateTemplateRoute identifies the future template source route fragment.
	ProjectCreateTemplateRoute = "/create/template"
	// ProjectCreateRemoteHostRoute identifies the future remote-host source route fragment.
	ProjectCreateRemoteHostRoute = "/create/remote-host"
	// ProjectRefreshRoute identifies the static refresh route fragment.
	ProjectRefreshRoute = "/:id/refresh"
	// ProjectUpRoute identifies the compose up route fragment.
	ProjectUpRoute = "/:id/up"
	// ProjectStopRoute identifies the compose stop route fragment.
	ProjectStopRoute = "/:id/stop"
	// ProjectRestartRoute identifies the compose restart route fragment.
	ProjectRestartRoute = "/:id/restart"
	// ProjectRedeployRoute identifies the compose redeploy route fragment.
	ProjectRedeployRoute = "/:id/redeploy"
	// ProjectLifecycleConfigurationRoute identifies the lifecycle configuration route fragment.
	ProjectLifecycleConfigurationRoute = "/:id/lifecycle-configuration"
	// ProjectUnregisterRoute identifies the unregister route fragment.
	ProjectUnregisterRoute = "/:id/unregister"
	// ProjectDestroyRoute identifies the guarded destroy route fragment.
	ProjectDestroyRoute = "/:id/destroy"
	// ProjectBatchActionsRoute identifies the batch-action route fragment.
	ProjectBatchActionsRoute = "/batch-actions"
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
