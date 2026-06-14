// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import type { components, paths } from '@/contracts/openapi/generated/schema';

import type { CONTAINER_API_PATH } from '../contract/paths';

export type ContainerSummary = components['schemas']['ContainerSummary'];
export type ContainerDetail = components['schemas']['ContainerDetail'];
export type ContainerPort = components['schemas']['ContainerPort'];
export type ContainerRuntimeInfo = components['schemas']['ContainerRuntimeInfo'];
export type ContainerLogResponse = components['schemas']['ContainerLogResponse'];
export type ContainerActionResponse = components['schemas']['ContainerActionResponse'];
export type ContainerState = ContainerSummary['state'];
export type ContainerAction = ContainerActionResponse['action'];

type ContainerLogsPath = (typeof CONTAINER_API_PATH)['LOGS'];
type GetContainerLogsOperation = paths[ContainerLogsPath]['get'];

export type ContainerLogQuery = NonNullable<GetContainerLogsOperation['parameters']['query']>;

export type ContainerFilters = {
  keyword: string;
  status: ContainerState | 'all';
};
