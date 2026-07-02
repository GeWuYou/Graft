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

/**
 * 获取项目导入目录来源列表。
 *
 * @returns 导入目录来源数据。
 */
export function getProjectImportDirectorySources() {
  return request.get<ProjectImportDirectorySourcesResponse>({
    url: PROJECT_API_PATH.IMPORT_DIRECTORY_SOURCES,
  });
}

/**
 * 获取运行时候选导入入口。
 *
 * @returns 候选列表与入口 authority 摘要
 */
export function getProjectImportRuntimeCandidates(query?: ProjectImportRuntimeCandidatesQuery) {
  return request.get<ProjectImportRuntimeCandidatesResponse>({
    url: PROJECT_API_PATH.IMPORT_RUNTIME_CANDIDATES,
    params: query as ProjectImportRuntimeCandidatesQuery | undefined,
  });
}

/**
 * 获取项目导入目录列表。
 *
 * @param query - 列表查询条件
 * @returns 导入目录列表响应
 */
export function getProjectImportDirectories(query: ProjectImportDirectoryListQuery) {
  return request.get<ProjectImportDirectoryListResponse>({
    url: buildProjectImportDirectoriesApiPath(),
    params: query,
  });
}

/**
 * 根据运行时候选创建导入检查会话。
 *
 * @param payload - 候选导入检查请求内容
 * @returns 导入检查结果
 */
export function postProjectImportRuntimeInspect(payload: ProjectImportRuntimeInspectRequest) {
  return request.post<ProjectImportInspectResponse>({
    url: PROJECT_API_PATH.IMPORT_RUNTIME_INSPECT,
    data: payload,
  });
}

/**
 * 执行项目导入。
 *
 * @param payload - 导入执行请求内容
 * @returns 导入执行结果
 */
export function postProjectImportExecute(payload: ProjectImportExecuteRequest) {
  return request.post<ProjectImportExecuteResponse>({
    url: PROJECT_API_PATH.IMPORT,
    data: payload,
  });
}
