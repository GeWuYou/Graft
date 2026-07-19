import { OPENAPI_RUNTIME_PATH } from '@/contracts/generated/openapi-runtime-paths';
import { request } from '@/utils/request';

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
    url: OPENAPI_RUNTIME_PATH.getApplicationImportDirectorySources,
  });
}

export function getApplicationImportRuntimeCandidates(query?: ApplicationImportRuntimeCandidatesQuery) {
  return request.get<ApplicationImportRuntimeCandidatesResponse>({
    url: OPENAPI_RUNTIME_PATH.getApplicationImportRuntimeCandidates,
    params: query as ApplicationImportRuntimeCandidatesQuery | undefined,
  });
}

export function getApplicationImportDirectories(query: ApplicationImportDirectoryListQuery) {
  return request.get<ApplicationImportDirectoryListResponse>({
    url: OPENAPI_RUNTIME_PATH.getApplicationImportDirectories,
    params: query,
  });
}

export function postApplicationImportRuntimeInspect(payload: ApplicationImportRuntimeInspectRequest) {
  return request.post<ApplicationImportInspectResponse>({
    url: OPENAPI_RUNTIME_PATH.postApplicationImportRuntimeInspect,
    data: payload,
  });
}

export function postApplicationImportExecute(payload: ApplicationImportExecuteRequest) {
  return request.post<ApplicationImportExecuteResponse>({
    url: OPENAPI_RUNTIME_PATH.postApplicationImport,
    data: payload,
  });
}
