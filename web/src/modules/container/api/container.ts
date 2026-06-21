// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import type { paths } from '@/contracts/openapi/generated/schema';
import { request } from '@/utils/request';

import {
  buildContainerDetailApiPath,
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

/**
 * Retrieves a list of containers.
 *
 * @param query - Optional query parameters for filtering and pagination
 * @returns A Promise that resolves to the container list response data
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
 * Retrieves logs for a container.
 *
 * @param containerId - The ID of the container
 * @param query - Query parameters to filter or paginate the logs
 * @returns The container's log response data
 */
export function getContainerLogs(containerId: GetContainerLogsPathParams['id'], query: ContainerLogQuery) {
  return request.get<GetContainerLogsData>({
    url: buildContainerLogsApiPath(containerId),
    params: query,
  }) as Promise<ContainerLogResponse>;
}

/**
 * Retrieves mount usage information for a container.
 *
 * @returns A list of mount usage data for the container.
 */
export function getContainerMountUsage(containerId: ContainerMountUsagePathParams['id']) {
  return request.get<GetContainerMountUsageData>({
    url: buildContainerMountUsageApiPath(containerId),
  }) as Promise<ContainerMountUsageListResponse>;
}

/**
 * Refreshes the mount usage data for a specific container mount.
 *
 * @param containerId - The container ID
 * @param mountId - The mount ID
 * @returns The refreshed mount usage information
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
 * Creates a shell session for the specified container.
 *
 * @param containerId - The ID of the container
 * @param body - The shell session request parameters
 * @returns The created shell session response
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
 * Starts a container.
 *
 * @returns The action response.
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
