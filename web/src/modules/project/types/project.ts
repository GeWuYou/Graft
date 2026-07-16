import type { components, paths } from '@/contracts/openapi/generated/schema';

import type { APPLICATION_API_PATH } from '../contract/paths';

export type ApplicationSourceType = components['schemas']['ApplicationSourceType'];
export type ApplicationOwnershipMode = components['schemas']['ApplicationOwnershipMode'];
export type ApplicationDriftStatus = components['schemas']['ApplicationDriftStatus'];
export type ApplicationComposeProjectNameSource = components['schemas']['ApplicationComposeProjectNameSource'];
export type ApplicationFileKind = components['schemas']['ApplicationFileKind'];
export type ApplicationFileRole = components['schemas']['ApplicationFileRole'];
export type ApplicationFileItem = components['schemas']['ApplicationFileItem'];
export type ApplicationContainerCounts = components['schemas']['ApplicationContainerCounts'];
export type ApplicationListItem = components['schemas']['ApplicationListItem'];
export type ApplicationListResponse = components['schemas']['ApplicationListResponse'];
export type ApplicationCreationMethodType = components['schemas']['ApplicationCreationMethodType'];
export type ApplicationCreationMethodAvailability = components['schemas']['ApplicationCreationMethodAvailability'];
export type ApplicationCreationMethod = components['schemas']['ApplicationCreationMethod'];
export type ApplicationCreationMethodCatalogResponse =
  components['schemas']['ApplicationCreationMethodCatalogResponse'];
export type ApplicationComposeRuntimeTarget = components['schemas']['ApplicationComposeRuntimeTarget'];
export type ApplicationComposeRuntimeTargetCatalogResponse =
  components['schemas']['ApplicationComposeRuntimeTargetCatalogResponse'];
export type ApplicationImportValidateRequest = components['schemas']['ApplicationImportValidateRequest'];
export type ApplicationImportValidateResponse = components['schemas']['ApplicationImportValidateResponse'];
export type ApplicationImportResponse = components['schemas']['ApplicationImportResponse'];
export type ApplicationActivityAuthority = components['schemas']['application-activity-authority'];
export type ApplicationDiscoveryCandidateKind = components['schemas']['application-discovery-candidate-kind'];
export type ApplicationDiscoveryCandidateStatus = components['schemas']['application-discovery-candidate-status'];
export type ApplicationDiscoveryCandidate = components['schemas']['application-discovery-candidate'];
export type ApplicationDiscoveryCandidatesResponse = components['schemas']['application-discovery-candidates-response'];
export type ApplicationDetailResponse = components['schemas']['ApplicationDetailResponse'];
export type ApplicationLogEntryStream = components['schemas']['ApplicationLogEntryStream'];
export type ApplicationLogEntrySource = components['schemas']['ApplicationLogEntrySource'];
export type ApplicationLogEntry = components['schemas']['ApplicationLogEntry'];
export type ApplicationLogResponse = components['schemas']['ApplicationLogResponse'];
export type ApplicationOverviewHealthSummary = components['schemas']['ApplicationOverviewHealthSummary'];
export type ApplicationOverviewResourceSummary = components['schemas']['ApplicationOverviewResourceSummary'];
export type ApplicationOverviewServiceItem = components['schemas']['ApplicationOverviewServiceItem'];
export type ApplicationOverviewResponse = components['schemas']['ApplicationOverviewResponse'];
export type ApplicationServiceItem = components['schemas']['ApplicationServiceItem'];
export type ApplicationServicesResponse = components['schemas']['ApplicationServicesResponse'];
export type ApplicationManagedRootStatus = components['schemas']['application-managed-root-status'];
export type ApplicationManagedRootResponse = components['schemas']['application-managed-root-response'];
export type ApplicationCreateRequest = components['schemas']['ApplicationCreateRequest'];
export type ApplicationCreateResponse = components['schemas']['application-create-response'];
export type ApplicationCreateValidateRequest = components['schemas']['application-create-validate-request'];
export type ApplicationCreateValidateResponse = components['schemas']['application-create-validate-response'];
export type ApplicationApplicationNameAvailabilityRequest =
  components['schemas']['application-name-availability-request'];
export type ApplicationApplicationNameAvailabilityResponse =
  components['schemas']['application-name-availability-response'];
export type ApplicationTemplate = components['schemas']['application-template-response'];
export type ApplicationTemplateVersion = components['schemas']['application-template-version'];
export type ApplicationTemplateListResponse = components['schemas']['application-template-list-response'];
export type ApplicationTemplateDraftRequest = components['schemas']['application-template-draft-request'];
export type ApplicationWorkspaceManifestFile = components['schemas']['application-workspace-manifest-file'];
export type ApplicationWorkspaceEntry = components['schemas']['application-workspace-entry'];
export type ApplicationWorkspaceDraftFile = {
  path: string;
  node_type?: 'file';
  content: string;
};
export type ApplicationWorkspaceDraftDirectory = {
  path: string;
  node_type: 'directory';
};
export type ApplicationWorkspaceDraftEntry = ApplicationWorkspaceDraftFile | ApplicationWorkspaceDraftDirectory;
export type ApplicationWorkspaceRenameRequest = components['schemas']['application-workspace-entry-rename-request'];
export type ApplicationConfigurationMetadataResponse =
  components['schemas']['ApplicationConfigurationMetadataResponse'];
export type ApplicationConfigurationPreviewResponse = components['schemas']['ApplicationConfigurationPreviewResponse'];
export type ApplicationActionResponse = components['schemas']['ApplicationActionResponse'];
export type ApplicationTaskReceipt = components['schemas']['TaskReceipt'];
export type ApplicationBatchActionRequest = components['schemas']['application-batch-action-request'];
export type ApplicationBatchActionItem = components['schemas']['application-batch-action-item'];
export type ApplicationBatchActionResponse = components['schemas']['application-batch-action-response'];
export type ApplicationDestroyRequest = components['schemas']['application-destroy-request'];
export type ApplicationRuntimeStatus = ApplicationDetailResponse['runtime_status'];
export type ApplicationLifecycleReviewStatus = components['schemas']['application-lifecycle-review-status'];
export type ApplicationLifecycleStrategyKind = components['schemas']['application-lifecycle-strategy-kind'];
export type ApplicationLifecycleActionKey = 'up' | 'stop' | 'restart' | 'redeploy';
export type ApplicationLifecycleGeneratedCommand = components['schemas']['application-lifecycle-generated-command'];
export type ApplicationLifecycleConfigurationModel = components['schemas']['application-lifecycle-configuration'];
export type ApplicationLifecycleConfigurationUpdateRequest =
  components['schemas']['application-lifecycle-configuration-request'];
export type ApplicationLifecycleConfigurationSavedResponse =
  components['schemas']['application-lifecycle-configuration-response'];

export type ApplicationLifecycleCommandStep = {
  title_key: string;
  command: string;
  absolute_command?: string;
};

export type ApplicationLifecycleCommandPreview = Partial<
  Record<ApplicationLifecycleActionKey, ApplicationLifecycleCommandStep[]>
>;

export type ApplicationLifecycleConfigurationDraft = {
  strategy_kind: ApplicationLifecycleStrategyKind;
  workspace_path: string;
  compose_files: string[];
  compose_project_name: string;
  profiles: string[];
  down_before_redeploy: boolean;
  pull_before_redeploy: boolean;
  build_before_up: boolean;
  force_recreate: boolean;
  remove_orphans: boolean;
  wait_after_up: boolean;
  wait_timeout_seconds: number;
  renew_anon_volumes: boolean;
  prune_images_after_redeploy: boolean;
  additional_args: string;
  review_status?: ApplicationLifecycleReviewStatus | null;
  generated_commands?: ApplicationLifecycleCommandPreview | null;
};

export type ApplicationListItemWithLifecycle = ApplicationListItem;
export type ApplicationListResponseWithLifecycle = ApplicationListResponse;
export type ApplicationDetailResponseWithLifecycle = ApplicationDetailResponse;

type ApplicationListPath = (typeof APPLICATION_API_PATH)['LIST'];
type GetApplicationListOperation = paths[ApplicationListPath]['get'];

export type ApplicationListQuery = NonNullable<GetApplicationListOperation['parameters']['query']>;

export type ApplicationDeploymentAdapterKind = ApplicationListItem['deployment_adapter_kind'];
export type ApplicationProvider = NonNullable<ApplicationListItem['runtime_target']>['provider'];
export type ApplicationSavedView = components['schemas']['application-saved-view'];
export type ApplicationSavedViewRequest = components['schemas']['application-saved-view-request'];
export type ApplicationSavedViewQueryState = ApplicationSavedViewRequest['query_state'];

export type ApplicationFilters = {
  keyword: string;
  deploymentAdapterKind: ApplicationDeploymentAdapterKind | 'all';
  runtimeTargetId: number | undefined;
  provider: ApplicationProvider | 'all';
  sourceType: ApplicationSourceType | 'all';
  runtimeStatus: ApplicationRuntimeStatus | 'all';
  driftStatus: ApplicationDriftStatus | 'all';
};

export type ApplicationActivityStream = 'events' | 'logs';

export type ApplicationServiceContainerMember = ApplicationServiceItem['container_members'][number];

export type ApplicationBatchAction = ApplicationBatchActionRequest['action'];

export type ApplicationConfigurationFileResponse = {
  content: string;
  download_name: string;
  encoding?: string | null;
  file_id: number;
  kind: ApplicationFileKind;
  path: string;
  read_only?: boolean;
};

export type ApplicationDeployRequest = {
  compose_file_content?: string;
  env_file_content?: string;
};

export type ApplicationDeployResponse = ApplicationActionResponse;

export type ApplicationWorkspaceNodeType = 'file' | 'directory';
export type ApplicationWorkspaceFileKind =
  'directory' | 'compose' | 'env' | 'config' | 'text' | 'binary' | 'unsupported';
export type ApplicationWorkspaceLanguageHint =
  | 'yaml'
  | 'json'
  | 'dotenv'
  | 'ini'
  | 'toml'
  | 'properties'
  | 'xml'
  | 'sql'
  | 'markdown'
  | 'shell'
  | 'dockerfile'
  | 'hcl'
  | 'powershell'
  | 'plaintext'
  | (string & {});

export type ApplicationWorkspaceTreeItem = {
  name: string;
  relative_path: string;
  node_type: ApplicationWorkspaceNodeType;
  file_kind: ApplicationWorkspaceFileKind;
  readable: boolean;
  editable: boolean;
  language_hint?: ApplicationWorkspaceLanguageHint | null;
  size_bytes?: number | null;
  hidden_by_default?: boolean;
  has_children?: boolean;
  tooltip?: string | null;
  tooltip_source?: string | null;
  application_note?: string | null;
};

export type ApplicationWorkspaceFilesResponse = {
  root_path: string;
  current_path: string;
  parent_path?: string | null;
  items: ApplicationWorkspaceTreeItem[];
  has_more_hidden?: boolean;
};

export type ApplicationWorkspaceFilesQuery = {
  path?: string;
  show_hidden?: boolean;
};

export type ApplicationWorkspaceFileContentResponse = {
  relative_path: string;
  file_kind: ApplicationWorkspaceFileKind;
  language_hint?: ApplicationWorkspaceLanguageHint | null;
  readable: boolean;
  editable: boolean;
  encoding?: string | null;
  content: string;
  size_bytes?: number | null;
};

export type ApplicationWorkspaceFileContentQuery = {
  path: string;
};

export type ApplicationWorkspaceFileSaveRequest = {
  content: string;
};

export type ApplicationWorkspaceFileSaveResponse = {
  relative_path: string;
  saved_at?: string | null;
  content_hash?: string | null;
  size_bytes?: number | null;
};

type ApplicationWorkspaceFileAnnotationPath = (typeof APPLICATION_API_PATH)['FILES_ANNOTATION'];
type PutApplicationWorkspaceFileAnnotationOperation = paths[ApplicationWorkspaceFileAnnotationPath]['put'];

export type ApplicationWorkspaceFileAnnotationRequest =
  PutApplicationWorkspaceFileAnnotationOperation['requestBody']['content']['application/json'];
export type ApplicationWorkspaceFileAnnotationResponse =
  PutApplicationWorkspaceFileAnnotationOperation['responses'][200]['content']['application/json'] extends {
    data: infer T;
  }
    ? T
    : ApplicationWorkspaceTreeItem;
