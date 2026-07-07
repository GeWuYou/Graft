import type { components, paths } from '@/contracts/openapi/generated/schema';

import type { PROJECT_API_PATH } from '../contract/paths';

export type ProjectSourceKind = components['schemas']['ProjectSourceKind'];
export type ProjectHostScope = components['schemas']['ProjectHostScope'];
export type ProjectOwnershipMode = components['schemas']['ProjectOwnershipMode'];
export type ProjectRefreshStatus = components['schemas']['ProjectRefreshStatus'];
export type ProjectDriftStatus = components['schemas']['ProjectDriftStatus'];
export type ProjectCanonicalNameSource = components['schemas']['ProjectCanonicalNameSource'];
export type ProjectFileKind = components['schemas']['ProjectFileKind'];
export type ProjectFileRole = components['schemas']['ProjectFileRole'];
export type ProjectFileItem = components['schemas']['ProjectFileItem'];
export type ProjectContainerCounts = components['schemas']['ProjectContainerCounts'];
export type ProjectListItem = components['schemas']['ProjectListItem'];
export type ProjectListResponse = components['schemas']['ProjectListResponse'];
export type ProjectSourceEntryType = components['schemas']['ProjectSourceEntryType'];
export type ProjectSourceEntryStatus = components['schemas']['ProjectSourceEntryStatus'];
export type ProjectSourceEntry = components['schemas']['ProjectSourceEntry'];
export type ProjectSourceCatalogResponse = components['schemas']['ProjectSourceCatalogResponse'];
export type ProjectImportValidateRequest = components['schemas']['ProjectImportValidateRequest'];
export type ProjectImportValidateResponse = components['schemas']['ProjectImportValidateResponse'];
export type ProjectImportResponse = components['schemas']['ProjectImportResponse'];
export type ProjectActivityAuthority = components['schemas']['project-activity-authority'];
export type ProjectDiscoveryCandidateKind = components['schemas']['project-discovery-candidate-kind'];
export type ProjectDiscoveryCandidateStatus = components['schemas']['project-discovery-candidate-status'];
export type ProjectDiscoveryCandidate = components['schemas']['project-discovery-candidate'];
export type ProjectDiscoveryCandidatesResponse = components['schemas']['project-discovery-candidates-response'];
export type ProjectDetailResponse = components['schemas']['ProjectDetailResponse'];
export type ProjectLogEntryStream = components['schemas']['ProjectLogEntryStream'];
export type ProjectLogEntrySource = components['schemas']['ProjectLogEntrySource'];
export type ProjectLogEntry = components['schemas']['ProjectLogEntry'];
export type ProjectLogResponse = components['schemas']['ProjectLogResponse'];
export type ProjectOverviewHealthSummary = components['schemas']['ProjectOverviewHealthSummary'];
export type ProjectOverviewResourceSummary = components['schemas']['ProjectOverviewResourceSummary'];
export type ProjectOverviewServiceItem = components['schemas']['ProjectOverviewServiceItem'];
export type ProjectOverviewResponse = components['schemas']['ProjectOverviewResponse'];
export type ProjectServiceItem = components['schemas']['ProjectServiceItem'];
export type ProjectServicesResponse = components['schemas']['ProjectServicesResponse'];
export type ProjectManagedRootStatus = components['schemas']['project-managed-root-status'];
export type ProjectManagedRootResponse = components['schemas']['project-managed-root-response'];
export type ProjectCreateRequest = components['schemas']['ProjectCreateRequest'];
export type ProjectCreateResponse = components['schemas']['project-create-response'];
export type ProjectCreateValidateRequest = components['schemas']['project-create-validate-request'];
export type ProjectCreateValidateResponse = components['schemas']['project-create-validate-response'];
export type ProjectConfigurationMetadataResponse = components['schemas']['ProjectConfigurationMetadataResponse'];
export type ProjectConfigurationPreviewResponse = components['schemas']['ProjectConfigurationPreviewResponse'];
export type ProjectActionResponse = components['schemas']['ProjectActionResponse'];
export type ProjectBatchActionRequest = components['schemas']['project-batch-action-request'];
export type ProjectBatchActionItem = components['schemas']['project-batch-action-item'];
export type ProjectBatchActionResponse = components['schemas']['project-batch-action-response'];
export type ProjectDestroyRequest = components['schemas']['project-destroy-request'];
export type ProjectRuntimeStatus = ProjectDetailResponse['runtime_status'];
export type ProjectLifecycleReviewStatus = components['schemas']['project-lifecycle-review-status'];
export type ProjectLifecycleStrategyKind = components['schemas']['project-lifecycle-strategy-kind'];
export type ProjectLifecycleActionKey = 'up' | 'stop' | 'restart' | 'redeploy';
export type ProjectLifecycleGeneratedCommand = components['schemas']['project-lifecycle-generated-command'];
export type ProjectLifecycleConfigurationModel = components['schemas']['project-lifecycle-configuration'];
export type ProjectLifecycleConfigurationUpdateRequest =
  components['schemas']['project-lifecycle-configuration-request'];
export type ProjectLifecycleConfigurationSavedResponse =
  components['schemas']['project-lifecycle-configuration-response'];

export type ProjectLifecycleCommandStep = {
  title_key: string;
  command: string;
};

export type ProjectLifecycleCommandPreview = Partial<Record<ProjectLifecycleActionKey, ProjectLifecycleCommandStep[]>>;

export type ProjectLifecycleConfigurationDraft = {
  strategy_kind: ProjectLifecycleStrategyKind;
  working_directory: string;
  compose_files: string[];
  canonical_project_name: string;
  profiles: string[];
  down_before_redeploy: boolean;
  pull_before_redeploy: boolean;
  build_before_up: boolean;
  force_recreate: boolean;
  wait_after_up: boolean;
  prune_images_after_redeploy: boolean;
  additional_args: string;
  review_status?: ProjectLifecycleReviewStatus | null;
  generated_commands?: ProjectLifecycleCommandPreview | null;
};

export type ProjectListItemWithLifecycle = ProjectListItem;
export type ProjectListResponseWithLifecycle = ProjectListResponse;
export type ProjectDetailResponseWithLifecycle = ProjectDetailResponse;

type ProjectListPath = (typeof PROJECT_API_PATH)['LIST'];
type GetProjectListOperation = paths[ProjectListPath]['get'];

export type ProjectListQuery = NonNullable<GetProjectListOperation['parameters']['query']>;

export type ProjectFilters = {
  keyword: string;
  sourceKind: ProjectSourceKind | 'all';
  driftStatus: ProjectDriftStatus | 'all';
  lastRefreshStatus: ProjectRefreshStatus | 'all';
};

export type ProjectActivityStream = 'events' | 'logs';

export type ProjectServiceContainerMember = ProjectServiceItem['container_members'][number];

export type ProjectBatchAction = ProjectBatchActionRequest['action'];

export type ProjectConfigurationFileResponse = {
  content: string;
  download_name: string;
  encoding?: string | null;
  file_id: number;
  kind: ProjectFileKind;
  path: string;
  read_only?: boolean;
};

export type ProjectConfigurationDiffRequest = {
  compose_file_content?: string;
  env_file_content?: string;
};

export type ProjectConfigurationDiffResponse = {
  canonical_project_name: string;
  current_config_hash: string;
  files: Array<{
    changed: boolean;
    current_content: string;
    current_hash: string;
    kind: string;
    path: string;
    proposed_content: string;
    proposed_hash: string;
  }>;
  has_changes: boolean;
  ownership_mode: ProjectOwnershipMode | string;
  project_id: number;
  proposed_config_hash: string;
  warnings: string[];
};

export type ProjectConfigurationValidateRequest = {
  compose_file_content?: string;
  env_file_content?: string;
};

export type ProjectConfigurationValidateResponse = {
  canonical_project_name: string;
  declared_service_names: string[];
  normalized_compose_yaml: string;
  ownership_mode: ProjectOwnershipMode | string;
  project_id: number;
  proposed_config_hash: string;
  warnings: string[];
};

export type ProjectDeployRequest = {
  compose_file_content?: string;
  env_file_content?: string;
};

export type ProjectDeployResponse = ProjectActionResponse;

export type ProjectWorkspaceNodeType = 'file' | 'directory';
export type ProjectWorkspaceFileKind = 'directory' | 'compose' | 'env' | 'config' | 'text' | 'binary' | 'unsupported';
export type ProjectWorkspaceLanguageHint =
  'yaml' | 'json' | 'dotenv' | 'ini' | 'toml' | 'properties' | 'shell' | 'dockerfile' | 'plaintext' | (string & {});

export type ProjectWorkspaceTreeItem = {
  name: string;
  relative_path: string;
  node_type: ProjectWorkspaceNodeType;
  file_kind: ProjectWorkspaceFileKind;
  editable: boolean;
  language_hint?: ProjectWorkspaceLanguageHint | null;
  size_bytes?: number | null;
  hidden_by_default?: boolean;
  has_children?: boolean;
  tooltip?: string | null;
  tooltip_source?: string | null;
  project_note?: string | null;
};

export type ProjectWorkspaceFilesResponse = {
  root_path: string;
  current_path: string;
  parent_path?: string | null;
  items: ProjectWorkspaceTreeItem[];
  has_more_hidden?: boolean;
};

export type ProjectWorkspaceFilesQuery = {
  path?: string;
  show_hidden?: boolean;
};

export type ProjectWorkspaceFileContentResponse = {
  relative_path: string;
  file_kind: ProjectWorkspaceFileKind;
  language_hint?: ProjectWorkspaceLanguageHint | null;
  editable: boolean;
  encoding?: string | null;
  content: string;
  size_bytes?: number | null;
};

export type ProjectWorkspaceFileContentQuery = {
  path: string;
};

export type ProjectWorkspaceFileSaveRequest = {
  content: string;
};

export type ProjectWorkspaceFileSaveResponse = {
  relative_path: string;
  saved_at?: string | null;
  content_hash?: string | null;
  size_bytes?: number | null;
};

type ProjectWorkspaceFileAnnotationPath = (typeof PROJECT_API_PATH)['FILES_ANNOTATION'];
type PutProjectWorkspaceFileAnnotationOperation = paths[ProjectWorkspaceFileAnnotationPath]['put'];

export type ProjectWorkspaceFileAnnotationRequest =
  PutProjectWorkspaceFileAnnotationOperation['requestBody']['content']['application/json'];
export type ProjectWorkspaceFileAnnotationResponse =
  PutProjectWorkspaceFileAnnotationOperation['responses'][200]['content']['application/json'] extends {
    data: infer T;
  }
    ? T
    : ProjectWorkspaceTreeItem;
