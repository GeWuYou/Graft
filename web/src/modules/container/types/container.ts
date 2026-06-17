// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import type { components, paths } from '@/contracts/openapi/generated/schema';

import type { CONTAINER_API_PATH } from '../contract/paths';

export type ContainerSummary = components['schemas']['ContainerSummary'];
export type ContainerDetail = components['schemas']['ContainerDetail'];
export type ContainerPort = components['schemas']['ContainerPort'];
export type ContainerRuntimeInfo = components['schemas']['ContainerRuntimeInfo'];
export type ContainerHealthcheck = components['schemas']['ContainerHealthcheck'];
export type ContainerListSummary = components['schemas']['ContainerListSummary'];
export type ContainerLogResponse = components['schemas']['ContainerLogResponse'];
export type ContainerActionResponse = components['schemas']['ContainerActionResponse'];
export type ContainerRemoveRequest = components['schemas']['ContainerRemoveRequest'];
export type ContainerBatchActionRequest = components['schemas']['ContainerBatchActionRequest'];
export type ContainerBatchActionResponse = components['schemas']['ContainerBatchActionResponse'];
export type ContainerBatchActionItem = components['schemas']['ContainerBatchActionItem'];
export type ContainerState = ContainerSummary['state'];
export type ContainerHealth = NonNullable<ContainerSummary['health']>;
export type ContainerAction = ContainerActionResponse['action'];

type ContainerListPath = (typeof CONTAINER_API_PATH)['LIST'];
type GetContainersOperation = paths[ContainerListPath]['get'];

type ContainerLogsPath = (typeof CONTAINER_API_PATH)['LOGS'];
type GetContainerLogsOperation = paths[ContainerLogsPath]['get'];

export type ContainerListQuery = NonNullable<GetContainersOperation['parameters']['query']>;
export type ContainerLogQuery = NonNullable<GetContainerLogsOperation['parameters']['query']>;

export type ContainerFilters = {
  keyword: string;
  status: ContainerState | 'all';
  health: ContainerHealth | 'all';
};
