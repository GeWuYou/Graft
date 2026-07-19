import { buildOpenApiRuntimePath, OPENAPI_RUNTIME_PATH } from '@/contracts/generated/openapi-runtime-paths';
import type { paths } from '@/contracts/openapi/generated/schema';
import { request } from '@/utils/request';

import type {
  ContainerActionResponse,
  ContainerBatchActionRequest,
  ContainerBatchActionResponse,
  ContainerDetailRecord,
  ContainerListQueryWithOrchestrator,
  ContainerLogQuery,
  ContainerLogResponse,
  ContainerMountUsage,
  ContainerMountUsageListResponse,
  ContainerMountUsagePathParams,
  ContainerMountUsageRefreshPathParams,
  ContainerRemoveRequest,
  ContainerRuntimeEventsPathParams,
  ContainerRuntimeEventsResponse,
  ContainerShellSessionRequest,
  ContainerShellSessionResponse,
} from '../types/container';

type ContainerListPath = typeof OPENAPI_RUNTIME_PATH.getContainers;
type GetContainersOperation = paths[ContainerListPath]['get'];
type GetContainersEnvelope = GetContainersOperation['responses'][200]['content']['application/json'];
type GetContainersData = NonNullable<GetContainersEnvelope['data']>;

type ContainerDetailPath = typeof OPENAPI_RUNTIME_PATH.getContainer;
type GetContainerOperation = paths[ContainerDetailPath]['get'];
type GetContainerEnvelope = GetContainerOperation['responses'][200]['content']['application/json'];
type GetContainerData = NonNullable<GetContainerEnvelope['data']>;
type GetContainerPathParams = GetContainerOperation['parameters']['path'];

type ContainerLogsPath = typeof OPENAPI_RUNTIME_PATH.getContainerLogs;
type GetContainerLogsOperation = paths[ContainerLogsPath]['get'];
type GetContainerLogsEnvelope = GetContainerLogsOperation['responses'][200]['content']['application/json'];
type GetContainerLogsData = NonNullable<GetContainerLogsEnvelope['data']>;
type GetContainerLogsPathParams = GetContainerLogsOperation['parameters']['path'];

type ContainerEventsPath = typeof OPENAPI_RUNTIME_PATH.getContainerEvents;
type GetContainerEventsOperation = paths[ContainerEventsPath]['get'];
type GetContainerEventsEnvelope = GetContainerEventsOperation['responses'][200]['content']['application/json'];
type GetContainerEventsData = NonNullable<GetContainerEventsEnvelope['data']>;

type ContainerMountUsagePath = typeof OPENAPI_RUNTIME_PATH.getContainerMountUsage;
type GetContainerMountUsageOperation = paths[ContainerMountUsagePath]['get'];
type GetContainerMountUsageEnvelope = GetContainerMountUsageOperation['responses'][200]['content']['application/json'];
type GetContainerMountUsageData = NonNullable<GetContainerMountUsageEnvelope['data']>;

type ContainerMountUsageRefreshPath = typeof OPENAPI_RUNTIME_PATH.postContainerMountUsageRefresh;
type PostContainerMountUsageRefreshOperation = paths[ContainerMountUsageRefreshPath]['post'];
type PostContainerMountUsageRefreshEnvelope =
  PostContainerMountUsageRefreshOperation['responses'][200]['content']['application/json'];
type PostContainerMountUsageRefreshData = NonNullable<PostContainerMountUsageRefreshEnvelope['data']>;

type ContainerShellSessionsPath = typeof OPENAPI_RUNTIME_PATH.postContainerShellSession;
type PostContainerShellSessionOperation = paths[ContainerShellSessionsPath]['post'];
type PostContainerShellSessionEnvelope =
  PostContainerShellSessionOperation['responses'][200]['content']['application/json'];
type PostContainerShellSessionData = NonNullable<PostContainerShellSessionEnvelope['data']>;
type PostContainerShellSessionPathParams = PostContainerShellSessionOperation['parameters']['path'];
type PostContainerShellSessionRequest = NonNullable<
  PostContainerShellSessionOperation['requestBody']
>['content']['application/json'];

type ContainerStartPath = typeof OPENAPI_RUNTIME_PATH.postContainerStart;
type PostContainerStartOperation = paths[ContainerStartPath]['post'];
type PostContainerStartEnvelope = PostContainerStartOperation['responses'][200]['content']['application/json'];
type PostContainerStartData = NonNullable<PostContainerStartEnvelope['data']>;
type PostContainerStartPathParams = PostContainerStartOperation['parameters']['path'];

type ContainerStopPath = typeof OPENAPI_RUNTIME_PATH.postContainerStop;
type PostContainerStopOperation = paths[ContainerStopPath]['post'];
type PostContainerStopEnvelope = PostContainerStopOperation['responses'][200]['content']['application/json'];
type PostContainerStopData = NonNullable<PostContainerStopEnvelope['data']>;
type PostContainerStopPathParams = PostContainerStopOperation['parameters']['path'];

type ContainerRestartPath = typeof OPENAPI_RUNTIME_PATH.postContainerRestart;
type PostContainerRestartOperation = paths[ContainerRestartPath]['post'];
type PostContainerRestartEnvelope = PostContainerRestartOperation['responses'][200]['content']['application/json'];
type PostContainerRestartData = NonNullable<PostContainerRestartEnvelope['data']>;
type PostContainerRestartPathParams = PostContainerRestartOperation['parameters']['path'];

type ContainerRemovePath = typeof OPENAPI_RUNTIME_PATH.postContainerRemove;
type PostContainerRemoveOperation = paths[ContainerRemovePath]['post'];
type PostContainerRemoveEnvelope = PostContainerRemoveOperation['responses'][200]['content']['application/json'];
type PostContainerRemoveData = NonNullable<PostContainerRemoveEnvelope['data']>;
type PostContainerRemovePathParams = PostContainerRemoveOperation['parameters']['path'];
type PostContainerRemoveRequest = NonNullable<
  PostContainerRemoveOperation['requestBody']
>['content']['application/json'];

type ContainerBatchActionsPath = typeof OPENAPI_RUNTIME_PATH.postContainerBatchActions;
type PostContainerBatchActionsOperation = paths[ContainerBatchActionsPath]['post'];
type PostContainerBatchActionsEnvelope =
  PostContainerBatchActionsOperation['responses'][200]['content']['application/json'];
type PostContainerBatchActionsData = NonNullable<PostContainerBatchActionsEnvelope['data']>;
type PostContainerBatchActionsRequest = NonNullable<
  PostContainerBatchActionsOperation['requestBody']
>['content']['application/json'];

export type ContainerListResponse = GetContainersData;

type DockerImagesOperation = paths['/api/ops/docker/images']['get'];
type DockerVolumeBatchRemoveOperation = paths['/api/ops/docker/volumes/batch-remove']['post'];
export type DockerVolumeBatchRemoveRequest =
  DockerVolumeBatchRemoveOperation['requestBody']['content']['application/json'];
export type DockerVolumeBatchRemoveResult = NonNullable<
  DockerVolumeBatchRemoveOperation['responses'][200]['content']['application/json']['data']
>;
export type DockerImageListQuery = NonNullable<DockerImagesOperation['parameters']['query']>;
type DockerImagesData = NonNullable<DockerImagesOperation['responses'][200]['content']['application/json']['data']>;
type DockerNetworksData = NonNullable<
  paths['/api/ops/docker/networks']['get']['responses'][200]['content']['application/json']['data']
>;
type DockerVolumesData = NonNullable<
  paths['/api/ops/docker/volumes']['get']['responses'][200]['content']['application/json']['data']
>;
export type DockerVolumeDetail = NonNullable<
  paths['/api/ops/docker/volumes/{id}']['get']['responses'][200]['content']['application/json']['data']
>;
export type DockerVolumeListQuery = NonNullable<paths['/api/ops/docker/volumes']['get']['parameters']['query']>;
export type DockerVolumeListResponse = DockerVolumesData;
type DockerSystemData = NonNullable<
  paths['/api/ops/docker/system']['get']['responses'][200]['content']['application/json']['data']
>;

export type DockerImageRecord = DockerImagesData['items'][number];

export const getDockerImages = (query?: DockerImageListQuery) =>
  request.get<DockerImagesData>({
    url: OPENAPI_RUNTIME_PATH.getDockerImages,
    params: query,
  }) as Promise<DockerImagesData>;

export const getDockerImage = (imageId: string) =>
  request.get<DockerImageRecord>({
    url: buildOpenApiRuntimePath('getDockerImage', { id: imageId }),
  }) as Promise<DockerImageRecord>;
export const getDockerNetworks = () =>
  request.get<DockerNetworksData>({ url: OPENAPI_RUNTIME_PATH.getDockerNetworks }) as Promise<DockerNetworksData>;
export const getDockerVolumes = () =>
  request.get<DockerVolumesData>({ url: OPENAPI_RUNTIME_PATH.getDockerVolumes }) as Promise<DockerVolumesData>;
export const listDockerVolumes = (query?: DockerVolumeListQuery) =>
  request.get<DockerVolumesData>({
    url: OPENAPI_RUNTIME_PATH.getDockerVolumes,
    params: query,
  }) as Promise<DockerVolumesData>;
export const getDockerVolume = (id: string) =>
  request.get<DockerVolumeDetail>({
    url: buildOpenApiRuntimePath('getDockerVolume', { id }),
  }) as Promise<DockerVolumeDetail>;
export const removeDockerVolume = (id: string, options: { force?: boolean } = {}) =>
  request.post({ url: buildOpenApiRuntimePath('postDockerVolumeRemove', { id }), data: options });
export function batchRemoveDockerVolumes(payload: DockerVolumeBatchRemoveRequest) {
  return request.post<DockerVolumeBatchRemoveResult>({
    url: OPENAPI_RUNTIME_PATH.postDockerVolumeBatchRemove,
    data: payload,
  });
}
export const getDockerSystem = () =>
  request.get<DockerSystemData>({ url: OPENAPI_RUNTIME_PATH.getDockerSystem }) as Promise<DockerSystemData>;

/**
 * 获取容器列表。
 *
 * @param query - 用于筛选、分页和编排来源定位的可选查询条件
 * @returns 容器列表响应数据
 */
export function getContainers(query?: ContainerListQueryWithOrchestrator) {
  return request.get<GetContainersData>({
    url: OPENAPI_RUNTIME_PATH.getContainers,
    params: query,
  }) as Promise<ContainerListResponse>;
}

/**
 * 检索指定容器的详细信息。
 *
 * @param containerId - 容器的唯一标识符
 * @returns 容器的详细信息
 */
export function getContainer(containerId: GetContainerPathParams['id']) {
  return request.get<GetContainerData>({
    url: buildOpenApiRuntimePath('getContainer', { id: containerId }),
  }) as Promise<ContainerDetailRecord>;
}

/**
 * 获取容器日志。
 *
 * @param containerId - 容器 ID
 * @param query - 用于筛选、分页或限制日志范围的查询条件
 * @returns 容器日志响应数据
 */
export function getContainerLogs(containerId: GetContainerLogsPathParams['id'], query: ContainerLogQuery) {
  return request.get<GetContainerLogsData>({
    url: buildOpenApiRuntimePath('getContainerLogs', { id: containerId }),
    params: query,
  }) as Promise<ContainerLogResponse>;
}

/**
 * 获取容器的运行时事件。
 *
 * @param containerId - 容器 ID
 * @returns 容器运行时事件列表
 */
export function getContainerEvents(containerId: ContainerRuntimeEventsPathParams['id']) {
  return request.get<GetContainerEventsData>({
    url: buildOpenApiRuntimePath('getContainerEvents', { id: containerId }),
  }) as Promise<ContainerRuntimeEventsResponse>;
}

/**
 * 获取容器挂载的使用量信息。
 *
 * @param containerId - 容器 ID
 * @returns 容器挂载使用量列表
 */
export function getContainerMountUsage(containerId: ContainerMountUsagePathParams['id']) {
  return request.get<GetContainerMountUsageData>({
    url: buildOpenApiRuntimePath('getContainerMountUsage', { id: containerId }),
  }) as Promise<ContainerMountUsageListResponse>;
}

/**
 * 刷新指定容器挂载的使用量信息。
 *
 * @param containerId - 容器 ID
 * @param mountId - 挂载 ID
 * @returns 刷新后的挂载使用量信息
 */
export function postContainerMountUsageRefresh(
  containerId: ContainerMountUsageRefreshPathParams['id'],
  mountId: ContainerMountUsageRefreshPathParams['mountId'],
) {
  return request.post<PostContainerMountUsageRefreshData>({
    url: buildOpenApiRuntimePath('postContainerMountUsageRefresh', { id: containerId, mountId }),
  }) as Promise<ContainerMountUsage>;
}

/**
 * 为指定容器创建终端会话。
 *
 * @param containerId - 容器 ID
 * @param body - 终端会话请求参数
 * @returns 创建后的终端会话响应
 */
export function postContainerShellSession(
  containerId: PostContainerShellSessionPathParams['id'],
  body: ContainerShellSessionRequest & PostContainerShellSessionRequest,
) {
  return request.post<PostContainerShellSessionData>({
    url: buildOpenApiRuntimePath('postContainerShellSession', { id: containerId }),
    data: body,
  }) as Promise<ContainerShellSessionResponse>;
}

/**
 * 启动容器。
 *
 * @param containerId - 容器 ID
 * @returns 容器操作响应
 */
export function startContainer(containerId: PostContainerStartPathParams['id']) {
  return request.post<PostContainerStartData>({
    url: buildOpenApiRuntimePath('postContainerStart', { id: containerId }),
  }) as Promise<ContainerActionResponse>;
}

export function stopContainer(containerId: PostContainerStopPathParams['id']) {
  return request.post<PostContainerStopData>({
    url: buildOpenApiRuntimePath('postContainerStop', { id: containerId }),
  }) as Promise<ContainerActionResponse>;
}

export function restartContainer(containerId: PostContainerRestartPathParams['id']) {
  return request.post<PostContainerRestartData>({
    url: buildOpenApiRuntimePath('postContainerRestart', { id: containerId }),
  }) as Promise<ContainerActionResponse>;
}

export function removeContainer(
  containerId: PostContainerRemovePathParams['id'],
  body: ContainerRemoveRequest & PostContainerRemoveRequest,
) {
  return request.post<PostContainerRemoveData>({
    url: buildOpenApiRuntimePath('postContainerRemove', { id: containerId }),
    data: body,
  }) as Promise<ContainerActionResponse>;
}

export function batchContainerActions(body: ContainerBatchActionRequest & PostContainerBatchActionsRequest) {
  return request.post<PostContainerBatchActionsData>({
    url: OPENAPI_RUNTIME_PATH.postContainerBatchActions,
    data: body,
  }) as Promise<ContainerBatchActionResponse>;
}
