import type { components, paths } from '@/contracts/openapi/generated/schema';

import type { APPLICATION_API_PATH } from '../contract/paths';
import type { ApplicationComposeProjectNameSource, ApplicationImportResponse } from './project';

type ApplicationImportDirectorySourcesPath = (typeof APPLICATION_API_PATH)['IMPORT_DIRECTORY_SOURCES'];
type GetApplicationImportDirectorySourcesOperation = paths[ApplicationImportDirectorySourcesPath]['get'];
type GetApplicationImportDirectorySourcesEnvelope =
  GetApplicationImportDirectorySourcesOperation['responses'][200]['content']['application/json'];

type ApplicationImportDirectoriesPath = (typeof APPLICATION_API_PATH)['IMPORT_DIRECTORIES'];
type GetApplicationImportDirectoriesOperation = paths[ApplicationImportDirectoriesPath]['get'];
type GetApplicationImportDirectoriesEnvelope =
  GetApplicationImportDirectoriesOperation['responses'][200]['content']['application/json'];
type GetApplicationImportDirectoriesQuery = NonNullable<
  GetApplicationImportDirectoriesOperation['parameters']['query']
>;

type ApplicationImportInspectPath = (typeof APPLICATION_API_PATH)['IMPORT_INSPECT'];
type PostApplicationImportInspectOperation = paths[ApplicationImportInspectPath]['post'];
type PostApplicationImportInspectEnvelope =
  PostApplicationImportInspectOperation['responses'][200]['content']['application/json'];
type PostApplicationImportInspectPayload =
  PostApplicationImportInspectOperation['requestBody']['content']['application/json'];

type ApplicationImportRuntimeCandidatesPath = (typeof APPLICATION_API_PATH)['IMPORT_RUNTIME_CANDIDATES'];
type GetApplicationImportRuntimeCandidatesOperation = paths[ApplicationImportRuntimeCandidatesPath]['get'];
type GetApplicationImportRuntimeCandidatesEnvelope =
  GetApplicationImportRuntimeCandidatesOperation['responses'][200]['content']['application/json'];
type GetApplicationImportRuntimeCandidatesQuery = NonNullable<
  GetApplicationImportRuntimeCandidatesOperation['parameters']['query']
>;

type ApplicationImportRuntimeInspectPath = (typeof APPLICATION_API_PATH)['IMPORT_RUNTIME_INSPECT'];
type PostApplicationImportRuntimeInspectOperation = paths[ApplicationImportRuntimeInspectPath]['post'];
type PostApplicationImportRuntimeInspectEnvelope =
  PostApplicationImportRuntimeInspectOperation['responses'][200]['content']['application/json'];
type PostApplicationImportRuntimeInspectPayload =
  PostApplicationImportRuntimeInspectOperation['requestBody']['content']['application/json'];

type ApplicationImportPath = (typeof APPLICATION_API_PATH)['IMPORT'];
type PostApplicationImportOperation = paths[ApplicationImportPath]['post'];
type PostApplicationImportEnvelope = PostApplicationImportOperation['responses'][200]['content']['application/json'];
type PostApplicationImportPayload = PostApplicationImportOperation['requestBody']['content']['application/json'];

type ApplicationSchemas = components['schemas'];

export type ApplicationImportDirectoryProvider = string;

export type ApplicationImportDirectorySource = NonNullable<
  GetApplicationImportDirectorySourcesEnvelope['data']
>['items'][number];
export type ApplicationImportDirectorySourcesResponse = NonNullable<
  GetApplicationImportDirectorySourcesEnvelope['data']
>;

export type ApplicationImportDirectoryRef = {
  provider: string;
  root_id: string;
  path: string;
};

export type ApplicationImportDirectoryListItem = NonNullable<
  NonNullable<GetApplicationImportDirectoriesEnvelope['data']>['directories']
>[number];
export type ApplicationImportDirectoryListResponse = NonNullable<GetApplicationImportDirectoriesEnvelope['data']>;
export type ApplicationImportDirectoryListQuery = GetApplicationImportDirectoriesQuery;

export type ApplicationImportDirectoryInspectFileEntry = ApplicationSchemas['application-import-inspect-file-item'];

export type ApplicationImportDirectoryInspectRequest = PostApplicationImportInspectPayload;
export type ApplicationImportDirectoryInspectValidationStatus = 'ready' | 'conflict' | string;
export type ApplicationImportDirectoryInspectResponse = Omit<
  NonNullable<PostApplicationImportInspectEnvelope['data']>,
  'compose_files' | 'env_files'
> & {
  compose_project_name_source: ApplicationComposeProjectNameSource;
  compose_files: ApplicationImportDirectoryInspectFileEntry[];
  env_files: ApplicationImportDirectoryInspectFileEntry[];
};

export type ApplicationImportExecuteRequest = PostApplicationImportPayload;
export type ApplicationImportExecuteResponse = NonNullable<PostApplicationImportEnvelope['data']> &
  ApplicationImportResponse;

export type ApplicationImportRuntimeCandidateStatus = ApplicationSchemas['application-import-runtime-candidate-status'];
export type ApplicationImportRuntimeCandidateAvailability =
  ApplicationSchemas['application-import-runtime-candidate-availability'];
export type ApplicationImportRuntimeWorkspacePathSource =
  ApplicationSchemas['application-import-runtime-workspace-path-source'];
export type ApplicationImportRuntimeCandidate = NonNullable<
  GetApplicationImportRuntimeCandidatesEnvelope['data']
>['items'][number];
export type ApplicationImportRuntimeCandidatesResponse = NonNullable<
  GetApplicationImportRuntimeCandidatesEnvelope['data']
>;
export type ApplicationImportRuntimeCandidatesQuery = GetApplicationImportRuntimeCandidatesQuery;
export type ApplicationImportRuntimeCandidateFilterCounts = ApplicationImportRuntimeCandidatesResponse['filter_counts'];
export type ApplicationImportRuntimeInspectRequest = PostApplicationImportRuntimeInspectPayload;
export type ApplicationImportRuntimeInspectNetworkResource =
  ApplicationSchemas['application-import-runtime-network-resource'];
export type ApplicationImportRuntimeInspectVolumeResource =
  ApplicationSchemas['application-import-runtime-volume-resource'];
export type ApplicationImportRuntimeMember = ApplicationSchemas['application-import-runtime-member'];
export type ApplicationImportRuntimeInspectResponse = Omit<
  NonNullable<PostApplicationImportRuntimeInspectEnvelope['data']>,
  'networks' | 'volumes'
> & {
  networks: Array<ApplicationImportRuntimeInspectNetworkResource | string> | null;
  volumes: Array<ApplicationImportRuntimeInspectVolumeResource | string> | null;
};
export type ApplicationImportRuntimeCandidateInspectRequest = ApplicationImportRuntimeInspectRequest;
export type ApplicationImportInspectResponse = ApplicationImportRuntimeInspectResponse;
