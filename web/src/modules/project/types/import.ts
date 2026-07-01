import type { paths } from '@/contracts/openapi/generated/schema';

import type { PROJECT_API_PATH } from '../contract/paths';
import type { ProjectCanonicalNameSource, ProjectFileKind, ProjectFileRole, ProjectImportResponse } from './project';

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

type ProjectImportPath = (typeof PROJECT_API_PATH)['IMPORT'];
type PostProjectImportOperation = paths[ProjectImportPath]['post'];
type PostProjectImportEnvelope = PostProjectImportOperation['responses'][200]['content']['application/json'];
type PostProjectImportPayload = PostProjectImportOperation['requestBody']['content']['application/json'];

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

export type ProjectImportFileEntry = {
  kind: ProjectFileKind;
  role: ProjectFileRole;
  absolute_path: string;
  display_path: string;
  order_index: number;
  exists_on_last_refresh: boolean;
  last_observed_hash?: string | null;
};

export type ProjectImportInspectRequest = PostProjectImportInspectPayload;
export type ProjectImportValidationStatus = 'ready' | 'conflict' | string;
export type ProjectImportInspectResponse = Omit<
  NonNullable<PostProjectImportInspectEnvelope['data']>,
  'compose_files' | 'env_files'
> & {
  canonical_project_name_source: ProjectCanonicalNameSource;
  compose_files: ProjectImportFileEntry[];
  env_files: ProjectImportFileEntry[];
};

export type ProjectImportExecuteRequest = PostProjectImportPayload;
export type ProjectImportExecuteResponse = NonNullable<PostProjectImportEnvelope['data']> & ProjectImportResponse;
