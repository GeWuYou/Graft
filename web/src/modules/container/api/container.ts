import type { paths } from '@/contracts/openapi/generated/schema';
import { request } from '@/utils/request';

import {
  buildContainerDetailApiPath,
  buildContainerEventsApiPath,
  buildContainerLogsApiPath,
  buildContainerMountUsageApiPath,
  buildContainerMountUsageRefreshApiPath,
  buildContainerRemoveApiPath,
  buildContainerRestartApiPath,
  buildContainerShellSessionsApiPath,
  buildContainerStartApiPath,
  buildContainerStopApiPath,
  CONTAINER_API_PATH,
} from '../contract/paths';
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

type ContainerListPath = (typeof CONTAINER_API_PATH)['LIST'];
type GetContainersOperation = paths[ContainerListPath]['get'];
type GetContainersEnvelope = GetContainersOperation['responses'][200]['content']['application/json'];
type GetContainersData = NonNullable<GetContainersEnvelope['data']>;

type ContainerDetailPath = (typeof CONTAINER_API_PATH)['DETAIL'];
type GetContainerOperation = paths[ContainerDetailPath]['get'];
type GetContainerEnvelope = GetContainerOperation['responses'][200]['content']['application/json'];
type GetContainerData = NonNullable<GetContainerEnvelope['data']>;
type GetContainerPathParams = GetContainerOperation['parameters']['path'];

type ContainerLogsPath = (typeof CONTAINER_API_PATH)['LOGS'];
type GetContainerLogsOperation = paths[ContainerLogsPath]['get'];
type GetContainerLogsEnvelope = GetContainerLogsOperation['responses'][200]['content']['application/json'];
type GetContainerLogsData = NonNullable<GetContainerLogsEnvelope['data']>;
type GetContainerLogsPathParams = GetContainerLogsOperation['parameters']['path'];

type ContainerEventsPath = (typeof CONTAINER_API_PATH)['EVENTS'];
type GetContainerEventsOperation = paths[ContainerEventsPath]['get'];
type GetContainerEventsEnvelope = GetContainerEventsOperation['responses'][200]['content']['application/json'];
type GetContainerEventsData = NonNullable<GetContainerEventsEnvelope['data']>;

type ContainerMountUsagePath = (typeof CONTAINER_API_PATH)['MOUNTS_USAGE'];
type GetContainerMountUsageOperation = paths[ContainerMountUsagePath]['get'];
type GetContainerMountUsageEnvelope = GetContainerMountUsageOperation['responses'][200]['content']['application/json'];
type GetContainerMountUsageData = NonNullable<GetContainerMountUsageEnvelope['data']>;

type ContainerMountUsageRefreshPath = (typeof CONTAINER_API_PATH)['MOUNT_USAGE_REFRESH'];
type PostContainerMountUsageRefreshOperation = paths[ContainerMountUsageRefreshPath]['post'];
type PostContainerMountUsageRefreshEnvelope =
  PostContainerMountUsageRefreshOperation['responses'][200]['content']['application/json'];
type PostContainerMountUsageRefreshData = NonNullable<PostContainerMountUsageRefreshEnvelope['data']>;

type ContainerShellSessionsPath = (typeof CONTAINER_API_PATH)['SHELL_SESSIONS'];
type PostContainerShellSessionOperation = paths[ContainerShellSessionsPath]['post'];
type PostContainerShellSessionEnvelope =
  PostContainerShellSessionOperation['responses'][200]['content']['application/json'];
type PostContainerShellSessionData = NonNullable<PostContainerShellSessionEnvelope['data']>;
type PostContainerShellSessionPathParams = PostContainerShellSessionOperation['parameters']['path'];
type PostContainerShellSessionRequest = NonNullable<
  PostContainerShellSessionOperation['requestBody']
>['content']['application/json'];

type ContainerStartPath = (typeof CONTAINER_API_PATH)['START'];
type PostContainerStartOperation = paths[ContainerStartPath]['post'];
type PostContainerStartEnvelope = PostContainerStartOperation['responses'][200]['content']['application/json'];
type PostContainerStartData = NonNullable<PostContainerStartEnvelope['data']>;
type PostContainerStartPathParams = PostContainerStartOperation['parameters']['path'];

type ContainerStopPath = (typeof CONTAINER_API_PATH)['STOP'];
type PostContainerStopOperation = paths[ContainerStopPath]['post'];
type PostContainerStopEnvelope = PostContainerStopOperation['responses'][200]['content']['application/json'];
type PostContainerStopData = NonNullable<PostContainerStopEnvelope['data']>;
type PostContainerStopPathParams = PostContainerStopOperation['parameters']['path'];

type ContainerRestartPath = (typeof CONTAINER_API_PATH)['RESTART'];
type PostContainerRestartOperation = paths[ContainerRestartPath]['post'];
type PostContainerRestartEnvelope = PostContainerRestartOperation['responses'][200]['content']['application/json'];
type PostContainerRestartData = NonNullable<PostContainerRestartEnvelope['data']>;
type PostContainerRestartPathParams = PostContainerRestartOperation['parameters']['path'];

type ContainerRemovePath = (typeof CONTAINER_API_PATH)['REMOVE'];
type PostContainerRemoveOperation = paths[ContainerRemovePath]['post'];
type PostContainerRemoveEnvelope = PostContainerRemoveOperation['responses'][200]['content']['application/json'];
type PostContainerRemoveData = NonNullable<PostContainerRemoveEnvelope['data']>;
type PostContainerRemovePathParams = PostContainerRemoveOperation['parameters']['path'];
type PostContainerRemoveRequest = NonNullable<
  PostContainerRemoveOperation['requestBody']
>['content']['application/json'];

type ContainerBatchActionsPath = (typeof CONTAINER_API_PATH)['BATCH_ACTIONS'];
type PostContainerBatchActionsOperation = paths[ContainerBatchActionsPath]['post'];
type PostContainerBatchActionsEnvelope =
  PostContainerBatchActionsOperation['responses'][200]['content']['application/json'];
type PostContainerBatchActionsData = NonNullable<PostContainerBatchActionsEnvelope['data']>;
type PostContainerBatchActionsRequest = NonNullable<
  PostContainerBatchActionsOperation['requestBody']
>['content']['application/json'];

export type ContainerListResponse = GetContainersData;

type DockerImagesData = NonNullable<
  paths['/api/ops/docker/images']['get']['responses'][200]['content']['application/json']['data']
>;
type DockerNetworksData = NonNullable<
  paths['/api/ops/docker/networks']['get']['responses'][200]['content']['application/json']['data']
>;
type DockerVolumesData = NonNullable<
  paths['/api/ops/docker/volumes']['get']['responses'][200]['content']['application/json']['data']
>;
type DockerSystemData = NonNullable<
  paths['/api/ops/docker/system']['get']['responses'][200]['content']['application/json']['data']
>;

export const getDockerImages = () =>
  request.get<DockerImagesData>({ url: CONTAINER_API_PATH.DOCKER_IMAGES }) as Promise<DockerImagesData>;
export const getDockerNetworks = () =>
  request.get<DockerNetworksData>({ url: CONTAINER_API_PATH.DOCKER_NETWORKS }) as Promise<DockerNetworksData>;
export const getDockerVolumes = () =>
  request.get<DockerVolumesData>({ url: CONTAINER_API_PATH.DOCKER_VOLUMES }) as Promise<DockerVolumesData>;
export const getDockerSystem = () =>
  request.get<DockerSystemData>({ url: CONTAINER_API_PATH.DOCKER_SYSTEM }) as Promise<DockerSystemData>;

/**
 * 获取容器列表。
 *
 * @param query - 用于筛选、分页和编排来源定位的可选查询条件
 * @returns 容器列表响应数据
 */
export function getContainers(query?: ContainerListQueryWithOrchestrator) {
  return request.get<GetContainersData>({
    url: CONTAINER_API_PATH.LIST,
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
    url: buildContainerDetailApiPath(containerId),
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
    url: buildContainerLogsApiPath(containerId),
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
    url: buildContainerEventsApiPath(containerId),
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
    url: buildContainerMountUsageApiPath(containerId),
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
    url: buildContainerMountUsageRefreshApiPath(containerId, mountId),
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
    url: buildContainerShellSessionsApiPath(containerId),
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
    url: buildContainerStartApiPath(containerId),
  }) as Promise<ContainerActionResponse>;
}

export function stopContainer(containerId: PostContainerStopPathParams['id']) {
  return request.post<PostContainerStopData>({
    url: buildContainerStopApiPath(containerId),
  }) as Promise<ContainerActionResponse>;
}

export function restartContainer(containerId: PostContainerRestartPathParams['id']) {
  return request.post<PostContainerRestartData>({
    url: buildContainerRestartApiPath(containerId),
  }) as Promise<ContainerActionResponse>;
}

export function removeContainer(
  containerId: PostContainerRemovePathParams['id'],
  body: ContainerRemoveRequest & PostContainerRemoveRequest,
) {
  return request.post<PostContainerRemoveData>({
    url: buildContainerRemoveApiPath(containerId),
    data: body,
  }) as Promise<ContainerActionResponse>;
}

export function batchContainerActions(body: ContainerBatchActionRequest & PostContainerBatchActionsRequest) {
  return request.post<PostContainerBatchActionsData>({
    url: CONTAINER_API_PATH.BATCH_ACTIONS,
    data: body,
  }) as Promise<ContainerBatchActionResponse>;
}
