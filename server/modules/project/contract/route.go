package contract

const (
	// ProjectAPIGroup is the API route group for Compose project management.
	ProjectAPIGroup = "/ops/projects"
	// ProjectCollectionRoute identifies the project list route fragment.
	ProjectCollectionRoute = ""
	// ProjectSavedViewsRoute identifies the project-list saved-view collection route.
	ProjectSavedViewsRoute = "/saved-views"
	// ProjectSavedViewRoute identifies one project-list saved view.
	ProjectSavedViewRoute = "/saved-views/:viewId"
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
	// ProjectWorkspaceEntryRoute identifies project workspace entry mutation routes.
	ProjectWorkspaceEntryRoute = "/:id/files/entries"
	// ProjectWorkspaceRenameRoute identifies project workspace entry rename routes.
	ProjectWorkspaceRenameRoute = "/:id/files/rename"
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
	// ProjectCreationMethodsRoute identifies the project creation-method catalog route fragment.
	ProjectCreationMethodsRoute = "/creation-methods"
	// ProjectDiscoveryCandidatesRoute identifies the bounded discovery-candidate preview route fragment.
	ProjectDiscoveryCandidatesRoute = "/discovery-candidates"
	// ProjectCreationMethodSelectorRoute identifies the creation-method selector route fragment.
	ProjectCreationMethodSelectorRoute = "/create"
	// ProjectComposeRuntimeTargetsRoute identifies the selectable runtime-target catalog for Compose creation.
	ProjectComposeRuntimeTargetsRoute = "/create/runtime-targets"
	// ProjectManagedRootRoute identifies the managed-root metadata route fragment.
	ProjectManagedRootRoute = "/managed/root"
	// ProjectCreateValidateRoute identifies the managed-create validation route fragment.
	ProjectCreateValidateRoute = "/create/managed/validate"
	// ProjectApplicationNameAvailabilityRoute identifies the managed-create application-name preflight route fragment.
	ProjectApplicationNameAvailabilityRoute = "/create/application-name/availability"
	// ProjectCreateRoute identifies the managed-create route fragment.
	ProjectCreateRoute = "/create/managed"
	// ProjectCreateTemplateValidateRoute validates a runtime template source without materializing it.
	ProjectCreateTemplateValidateRoute = "/create/template/validate"
	// ProjectCreateTemplateRoute identifies the future template source route fragment.
	ProjectCreateTemplateRoute = "/create/template"
	// ProjectWorkspaceDefaultsRoute returns server-owned blank workspace defaults and available templates.
	ProjectWorkspaceDefaultsRoute = "/create/workspace-defaults"
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
	ProjectMenuRootPath = "/applications/projects"
	// ProjectMenuPath identifies the canonical web menu path for Compose project management.
	ProjectMenuPath = ProjectMenuRootPath
	// ProjectCreationMenuPath identifies the global creation-method selector route path.
	ProjectCreationMenuPath = ProjectMenuRootPath + "/create"
	// ProjectBlankCreateMenuPath identifies the blank-project create route path.
	ProjectBlankCreateMenuPath = ProjectCreationMenuPath + "/blank"
	// ProjectTemplateCreateMenuPath identifies the template source create route path.
	ProjectTemplateCreateMenuPath = ProjectCreationMenuPath + "/template"
	// ProjectImportCreateMenuPath identifies the import-project create route path.
	ProjectImportCreateMenuPath = ProjectCreationMenuPath + "/import"
	// ProjectDiscoveryCandidatesMenuPath identifies the hidden discovery-candidate preview route path.
	ProjectDiscoveryCandidatesMenuPath = ProjectCreationMenuPath + "/discovery"
)
