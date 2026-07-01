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
 * 发送导入检查请求。
 *
 * @param payload - 导入检查请求体
 * @returns 导入检查结果
 */
export function postProjectImportInspect(payload: ProjectImportInspectRequest) {
  return request.post<ProjectImportInspectResponse>({
    url: PROJECT_API_PATH.IMPORT_INSPECT,
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
