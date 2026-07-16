import { request } from '@/utils/request';

import { buildProjectImportDirectoriesApiPath, PROJECT_API_PATH } from '../contract/paths';
import type {
  ProjectImportDirectoryListQuery,
  ProjectImportDirectoryListResponse,
  ProjectImportDirectorySourcesResponse,
  ProjectImportExecuteRequest,
  ProjectImportExecuteResponse,
  ProjectImportInspectResponse,
  ProjectImportRuntimeCandidatesQuery,
  ProjectImportRuntimeCandidatesResponse,
  ProjectImportRuntimeInspectRequest,
} from '../types/import';

export function getProjectImportDirectorySources() {
  return request.get<ProjectImportDirectorySourcesResponse>({
    url: PROJECT_API_PATH.IMPORT_DIRECTORY_SOURCES,
  });
}

export function getProjectImportRuntimeCandidates(query?: ProjectImportRuntimeCandidatesQuery) {
  return request.get<ProjectImportRuntimeCandidatesResponse>({
    url: PROJECT_API_PATH.IMPORT_RUNTIME_CANDIDATES,
    params: query as ProjectImportRuntimeCandidatesQuery | undefined,
  });
}

export function getProjectImportDirectories(query: ProjectImportDirectoryListQuery) {
  return request.get<ProjectImportDirectoryListResponse>({
    url: buildProjectImportDirectoriesApiPath(),
    params: query,
  });
}

export function postProjectImportRuntimeInspect(payload: ProjectImportRuntimeInspectRequest) {
  return request.post<ProjectImportInspectResponse>({
    url: PROJECT_API_PATH.IMPORT_RUNTIME_INSPECT,
    data: payload,
  });
}

export function postProjectImportExecute(payload: ProjectImportExecuteRequest) {
  return request.post<ProjectImportExecuteResponse>({
    url: PROJECT_API_PATH.IMPORT,
    data: payload,
  });
}
