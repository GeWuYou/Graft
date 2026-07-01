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
export type ProjectActivityAuthority = components['schemas']['project-activity-authority'];
export type ProjectDiscoveryCandidateKind = components['schemas']['project-discovery-candidate-kind'];
export type ProjectDiscoveryCandidateStatus = components['schemas']['project-discovery-candidate-status'];
export type ProjectDiscoveryCandidate = components['schemas']['project-discovery-candidate'];
export type ProjectDiscoveryCandidatesResponse = components['schemas']['project-discovery-candidates-response'];
export type ProjectDetailResponse = components['schemas']['ProjectDetailResponse'];
export type ProjectServiceItem = components['schemas']['ProjectServiceItem'];
export type ProjectServicesResponse = components['schemas']['ProjectServicesResponse'];
export type ProjectManagedRootStatus = components['schemas']['project-managed-root-status'];
export type ProjectManagedRootResponse = components['schemas']['project-managed-root-response'];
export type ProjectCreateValidateRequest = components['schemas']['project-create-validate-request'];
export type ProjectCreateValidateResponse = components['schemas']['project-create-validate-response'];
export type ProjectCreateRequest = components['schemas']['ProjectCreateRequest'];
export type ProjectCreateResponse = components['schemas']['project-create-response'];
export type ProjectConfigurationMetadataResponse = components['schemas']['ProjectConfigurationMetadataResponse'];
export type ProjectConfigurationPreviewResponse = components['schemas']['ProjectConfigurationPreviewResponse'];
export type ProjectConfigurationFileResponse = components['schemas']['ProjectConfigurationFileResponse'];
export type ProjectConfigurationDiffRequest = components['schemas']['project-configuration-diff-request'];
export type ProjectConfigurationDiffResponse = components['schemas']['project-configuration-diff-response'];
export type ProjectConfigurationValidateRequest = components['schemas']['project-configuration-validate-request'];
export type ProjectConfigurationValidateResponse = components['schemas']['project-configuration-validate-response'];
export type ProjectDeployRequest = components['schemas']['project-deploy-request'];
export type ProjectDeployResponse = components['schemas']['project-deploy-response'];
export type ProjectActionResponse = components['schemas']['ProjectActionResponse'];
export type ProjectRuntimeStatus = ProjectDetailResponse['runtime_status'];

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
