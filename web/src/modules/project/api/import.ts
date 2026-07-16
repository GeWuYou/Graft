import { request } from '@/utils/request';

import { APPLICATION_API_PATH, buildApplicationImportDirectoriesApiPath } from '../contract/paths';
import type {
  ApplicationImportDirectoryListQuery,
  ApplicationImportDirectoryListResponse,
  ApplicationImportDirectorySourcesResponse,
  ApplicationImportExecuteRequest,
  ApplicationImportExecuteResponse,
  ApplicationImportInspectResponse,
  ApplicationImportRuntimeCandidatesQuery,
  ApplicationImportRuntimeCandidatesResponse,
  ApplicationImportRuntimeInspectRequest,
} from '../types/import';

export function getApplicationImportDirectorySources() {
  return request.get<ApplicationImportDirectorySourcesResponse>({
    url: APPLICATION_API_PATH.IMPORT_DIRECTORY_SOURCES,
  });
}

export function getApplicationImportRuntimeCandidates(query?: ApplicationImportRuntimeCandidatesQuery) {
  return request.get<ApplicationImportRuntimeCandidatesResponse>({
    url: APPLICATION_API_PATH.IMPORT_RUNTIME_CANDIDATES,
    params: query as ApplicationImportRuntimeCandidatesQuery | undefined,
  });
}

export function getApplicationImportDirectories(query: ApplicationImportDirectoryListQuery) {
  return request.get<ApplicationImportDirectoryListResponse>({
    url: buildApplicationImportDirectoriesApiPath(),
    params: query,
  });
}

export function postApplicationImportRuntimeInspect(payload: ApplicationImportRuntimeInspectRequest) {
  return request.post<ApplicationImportInspectResponse>({
    url: APPLICATION_API_PATH.IMPORT_RUNTIME_INSPECT,
    data: payload,
  });
}

export function postApplicationImportExecute(payload: ApplicationImportExecuteRequest) {
  return request.post<ApplicationImportExecuteResponse>({
    url: APPLICATION_API_PATH.IMPORT,
    data: payload,
  });
}
