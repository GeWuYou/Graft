// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import type { paths } from '@/contracts/openapi/generated/schema';
import { request } from '@/utils/request';

import {
  buildContainerDetailApiPath,
  buildContainerLogsApiPath,
  buildContainerRestartApiPath,
  buildContainerStartApiPath,
  buildContainerStopApiPath,
  CONTAINER_API_PATH,
} from '../contract/paths';
import type {
  ContainerAction,
  ContainerActionResponse,
  ContainerDetail,
  ContainerLogQuery,
  ContainerLogResponse,
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

export type ContainerListResponse = GetContainersData;

export function getContainers() {
  return request.get<GetContainersData>({
    url: CONTAINER_API_PATH.LIST,
  }) as Promise<ContainerListResponse>;
}

export function getContainer(containerId: GetContainerPathParams['id']) {
  return request.get<GetContainerData>({
    url: buildContainerDetailApiPath(containerId),
  }) as Promise<ContainerDetail>;
}

export function getContainerLogs(containerId: GetContainerLogsPathParams['id'], query: ContainerLogQuery) {
  return request.get<GetContainerLogsData>({
    url: buildContainerLogsApiPath(containerId),
    params: query,
  }) as Promise<ContainerLogResponse>;
}

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

export function runContainerAction(action: ContainerAction, containerId: string) {
  if (action === 'start') {
    return startContainer(containerId);
  }
  if (action === 'stop') {
    return stopContainer(containerId);
  }
  return restartContainer(containerId);
}
