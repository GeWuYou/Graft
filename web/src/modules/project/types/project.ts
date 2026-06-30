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
