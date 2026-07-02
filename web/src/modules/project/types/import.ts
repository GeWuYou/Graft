import type { components, paths } from '@/contracts/openapi/generated/schema';

import type { PROJECT_API_PATH } from '../contract/paths';
import type { ProjectCanonicalNameSource, ProjectImportResponse } from './project';

type ProjectImportDirectorySourcesPath = (typeof PROJECT_API_PATH)['IMPORT_DIRECTORY_SOURCES'];
type GetProjectImportDirectorySourcesOperation = paths[ProjectImportDirectorySourcesPath]['get'];
type GetProjectImportDirectorySourcesEnvelope =
  GetProjectImportDirectorySourcesOperation['responses'][200]['content']['application/json'];

type ProjectImportDirectoriesPath = (typeof PROJECT_API_PATH)['IMPORT_DIRECTORIES'];
type GetProjectImportDirectoriesOperation = paths[ProjectImportDirectoriesPath]['get'];
type GetProjectImportDirectoriesEnvelope =
  GetProjectImportDirectoriesOperation['responses'][200]['content']['application/json'];
type GetProjectImportDirectoriesQuery = NonNullable<GetProjectImportDirectoriesOperation['parameters']['query']>;

type ProjectImportInspectPath = (typeof PROJECT_API_PATH)['IMPORT_INSPECT'];
type PostProjectImportInspectOperation = paths[ProjectImportInspectPath]['post'];
type PostProjectImportInspectEnvelope =
  PostProjectImportInspectOperation['responses'][200]['content']['application/json'];
type PostProjectImportInspectPayload = PostProjectImportInspectOperation['requestBody']['content']['application/json'];

type ProjectImportRuntimeCandidatesPath = (typeof PROJECT_API_PATH)['IMPORT_RUNTIME_CANDIDATES'];
type GetProjectImportRuntimeCandidatesOperation = paths[ProjectImportRuntimeCandidatesPath]['get'];
type GetProjectImportRuntimeCandidatesEnvelope =
  GetProjectImportRuntimeCandidatesOperation['responses'][200]['content']['application/json'];
type GetProjectImportRuntimeCandidatesQuery = NonNullable<
  GetProjectImportRuntimeCandidatesOperation['parameters']['query']
>;

type ProjectImportRuntimeInspectPath = (typeof PROJECT_API_PATH)['IMPORT_RUNTIME_INSPECT'];
type PostProjectImportRuntimeInspectOperation = paths[ProjectImportRuntimeInspectPath]['post'];
type PostProjectImportRuntimeInspectEnvelope =
  PostProjectImportRuntimeInspectOperation['responses'][200]['content']['application/json'];
type PostProjectImportRuntimeInspectPayload =
  PostProjectImportRuntimeInspectOperation['requestBody']['content']['application/json'];

type ProjectImportPath = (typeof PROJECT_API_PATH)['IMPORT'];
type PostProjectImportOperation = paths[ProjectImportPath]['post'];
type PostProjectImportEnvelope = PostProjectImportOperation['responses'][200]['content']['application/json'];
type PostProjectImportPayload = PostProjectImportOperation['requestBody']['content']['application/json'];

type ProjectSchemas = components['schemas'];

export type ProjectImportDirectoryProvider = string;

export type ProjectImportDirectorySource = NonNullable<
  GetProjectImportDirectorySourcesEnvelope['data']
>['items'][number];
export type ProjectImportDirectorySourcesResponse = NonNullable<GetProjectImportDirectorySourcesEnvelope['data']>;

export type ProjectImportDirectoryRef = {
  provider: string;
  root_id: string;
  path: string;
};

export type ProjectImportDirectoryListItem = NonNullable<
  NonNullable<GetProjectImportDirectoriesEnvelope['data']>['directories']
>[number];
export type ProjectImportDirectoryListResponse = NonNullable<GetProjectImportDirectoriesEnvelope['data']>;
export type ProjectImportDirectoryListQuery = GetProjectImportDirectoriesQuery;

export type ProjectImportDirectoryInspectFileEntry = ProjectSchemas['project-import-inspect-file-item'];

export type ProjectImportDirectoryInspectRequest = PostProjectImportInspectPayload;
export type ProjectImportDirectoryInspectValidationStatus = 'ready' | 'conflict' | string;
export type ProjectImportDirectoryInspectResponse = Omit<
  NonNullable<PostProjectImportInspectEnvelope['data']>,
  'compose_files' | 'env_files'
> & {
  canonical_project_name_source: ProjectCanonicalNameSource;
  compose_files: ProjectImportDirectoryInspectFileEntry[];
  env_files: ProjectImportDirectoryInspectFileEntry[];
};

export type ProjectImportExecuteRequest = PostProjectImportPayload;
export type ProjectImportExecuteResponse = NonNullable<PostProjectImportEnvelope['data']> & ProjectImportResponse;

export type ProjectImportRuntimeCandidateStatus = ProjectSchemas['project-import-runtime-candidate-status'];
export type ProjectImportRuntimeCandidateAvailability = ProjectSchemas['project-import-runtime-candidate-availability'];
export type ProjectImportRuntimeWorkingDirectorySource =
  ProjectSchemas['project-import-runtime-working-directory-source'];
export type ProjectImportRuntimeCandidate = NonNullable<
  GetProjectImportRuntimeCandidatesEnvelope['data']
>['items'][number];
export type ProjectImportRuntimeCandidatesResponse = NonNullable<GetProjectImportRuntimeCandidatesEnvelope['data']>;
export type ProjectImportRuntimeCandidatesQuery = GetProjectImportRuntimeCandidatesQuery;
export type ProjectImportRuntimeCandidateFilterCounts = ProjectImportRuntimeCandidatesResponse['filter_counts'];
export type ProjectImportRuntimeInspectRequest = PostProjectImportRuntimeInspectPayload;
export type ProjectImportRuntimeInspectNetworkResource = ProjectSchemas['project-import-runtime-network-resource'];
export type ProjectImportRuntimeInspectVolumeResource = ProjectSchemas['project-import-runtime-volume-resource'];
export type ProjectImportRuntimeMember = ProjectSchemas['project-import-runtime-member'];
export type ProjectImportRuntimeInspectResponse = Omit<
  NonNullable<PostProjectImportRuntimeInspectEnvelope['data']>,
  'networks' | 'volumes'
> & {
  networks: Array<ProjectImportRuntimeInspectNetworkResource | string> | null;
  volumes: Array<ProjectImportRuntimeInspectVolumeResource | string> | null;
};
export type ProjectImportRuntimeCandidateInspectRequest = ProjectImportRuntimeInspectRequest;
export type ProjectImportInspectResponse = ProjectImportRuntimeInspectResponse;
