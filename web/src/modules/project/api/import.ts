import { request } from '@/utils/request';

import { buildProjectImportDirectoriesApiPath, PROJECT_API_PATH } from '../contract/paths';
import type {
  ProjectImportDirectoryListQuery,
  ProjectImportDirectoryListResponse,
  ProjectImportDirectorySourcesResponse,
  ProjectImportExecuteRequest,
  ProjectImportExecuteResponse,
  ProjectImportInspectRequest,
  ProjectImportInspectResponse,
} from '../types/import';

export function getProjectImportDirectorySources() {
  return request.get<ProjectImportDirectorySourcesResponse>({
    url: PROJECT_API_PATH.IMPORT_DIRECTORY_SOURCES,
  });
}

export function getProjectImportDirectories(query: ProjectImportDirectoryListQuery) {
  return request.get<ProjectImportDirectoryListResponse>({
    url: buildProjectImportDirectoriesApiPath(),
    params: query,
  });
}

export function postProjectImportInspect(payload: ProjectImportInspectRequest) {
  return request.post<ProjectImportInspectResponse>({
    url: PROJECT_API_PATH.IMPORT_INSPECT,
    data: payload,
  });
}

export function postProjectImportExecute(payload: ProjectImportExecuteRequest) {
  return request.post<ProjectImportExecuteResponse>({
    url: PROJECT_API_PATH.IMPORT,
    data: payload,
  });
}
