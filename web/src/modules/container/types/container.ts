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
export type ContainerMount = components['schemas']['ContainerMount'];
export type ContainerMountUsage = components['schemas']['ContainerMountUsage'];
export type ContainerMountUsageListResponse = components['schemas']['ContainerMountUsageListResponse'];
export type ContainerShellSessionRequest = components['schemas']['ContainerShellSessionRequest'];
export type ContainerShellSessionResponse = components['schemas']['ContainerShellSessionResponse'];
export type ContainerState = ContainerSummary['state'];
export type ContainerHealth = NonNullable<ContainerSummary['health']>;
export type ContainerAction = ContainerActionResponse['action'];
export type ContainerMountUsageStatus = ContainerMountUsage['status'];

export type ContainerOrchestratorInfo = components['schemas']['ContainerOrchestratorInfo'];
export type ContainerOrchestratorType = ContainerOrchestratorInfo['type'];
export type ContainerActionLevel = 'readonly' | 'warn' | 'allow';
export type ContainerOrchestratorWarningCode = string;
export type ContainerOrchestratorRecommendedAction = string;
export type ContainerSummaryRecord = ContainerSummary;
export type ContainerDetailRecord = ContainerDetail;

type ContainerListPath = (typeof CONTAINER_API_PATH)['LIST'];
type GetContainersOperation = paths[ContainerListPath]['get'];

type ContainerLogsPath = (typeof CONTAINER_API_PATH)['LOGS'];
type GetContainerLogsOperation = paths[ContainerLogsPath]['get'];

type ContainerMountUsagePath = (typeof CONTAINER_API_PATH)['MOUNTS_USAGE'];
type GetContainerMountUsageOperation = paths[ContainerMountUsagePath]['get'];

type ContainerMountUsageRefreshPath = (typeof CONTAINER_API_PATH)['MOUNT_USAGE_REFRESH'];
type PostContainerMountUsageRefreshOperation = paths[ContainerMountUsageRefreshPath]['post'];

export type ContainerListQuery = NonNullable<GetContainersOperation['parameters']['query']>;
export type ContainerListQueryWithOrchestrator = ContainerListQuery & {
  orchestrator?: ContainerOrchestratorType;
};
export type ContainerLogQuery = NonNullable<GetContainerLogsOperation['parameters']['query']>;
export type ContainerMountUsagePathParams = GetContainerMountUsageOperation['parameters']['path'];
export type ContainerMountUsageRefreshPathParams = PostContainerMountUsageRefreshOperation['parameters']['path'];

export type ContainerFilters = {
  keyword: string;
  orchestrator: ContainerOrchestratorType | 'all';
  status: ContainerState | 'all';
  health: ContainerHealth | 'all';
};
