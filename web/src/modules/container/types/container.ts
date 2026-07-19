import type { OPENAPI_RUNTIME_PATH } from '@/contracts/generated/openapi-runtime-paths';
import type { components, paths } from '@/contracts/openapi/generated/schema';

export type ContainerSummary = components['schemas']['ContainerSummary'];
export type ContainerDetail = components['schemas']['ContainerDetail'];
export type ContainerResourceSummary = components['schemas']['ContainerResourceSummary'];
export type ContainerPort = components['schemas']['ContainerPort'];
export type ContainerRuntimeInfo = components['schemas']['ContainerRuntimeInfo'];
export type ContainerHealthcheck = components['schemas']['ContainerHealthcheck'];
export type ContainerListSummary = components['schemas']['ContainerListSummary'];
export type ContainerLogEntry = components['schemas']['ContainerLogEntry'];
export type ContainerLogResponse = components['schemas']['ContainerLogResponse'];
export type ContainerRuntimeEventSeverity = components['schemas']['ContainerRuntimeEventSeverity'];
export type ContainerRuntimeEvent = components['schemas']['ContainerRuntimeEvent'];
export type ContainerRuntimeEventRecord = components['schemas']['ContainerRuntimeEventRecord'];
export type ContainerRuntimeEventStreamContext = components['schemas']['ContainerRuntimeEventStreamContext'];
export type ContainerRuntimeEventsResponse = components['schemas']['ContainerRuntimeEventsResponse'];
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

export type ContainerDeploymentInfo = components['schemas']['ContainerDeploymentInfo'];
type LegacyDeploymentContext = {
  display_name?: string | null;
  group_scope_kind?: ContainerListSourceScopeKind | null;
  group_value?: string | null;
  group_display_name?: string | null;
  member_scope_kind?: ContainerListSourceScopeKind | null;
  member_value?: string | null;
  member_display_name?: string | null;
};
// 运行时操作继续复用同一策略结构；展示层使用 deployment 术语以兼容不同编排来源。
export type ContainerOrchestratorInfo = ContainerDeploymentInfo & LegacyDeploymentContext;
export type ContainerOrchestratorType = ContainerDeploymentInfo['type'];
export type ContainerActionLevel = 'readonly' | 'warn' | 'allow';
export type ContainerOrchestratorWarningCode = string;
export type ContainerOrchestratorRecommendedAction = string;
export type ContainerSummaryRecord = ContainerSummary & { orchestrator?: ContainerOrchestratorInfo };
export type ContainerDetailRecord = ContainerDetail & { orchestrator?: ContainerOrchestratorInfo };

type ContainerListPath = typeof OPENAPI_RUNTIME_PATH.getContainers;
type GetContainersOperation = paths[ContainerListPath]['get'];

type ContainerLogsPath = typeof OPENAPI_RUNTIME_PATH.getContainerLogs;
type GetContainerLogsOperation = paths[ContainerLogsPath]['get'];

type ContainerEventsPath = typeof OPENAPI_RUNTIME_PATH.getContainerEvents;
type GetContainerEventsOperation = paths[ContainerEventsPath]['get'];

type ContainerMountUsagePath = typeof OPENAPI_RUNTIME_PATH.getContainerMountUsage;
type GetContainerMountUsageOperation = paths[ContainerMountUsagePath]['get'];

type ContainerMountUsageRefreshPath = typeof OPENAPI_RUNTIME_PATH.postContainerMountUsageRefresh;
type PostContainerMountUsageRefreshOperation = paths[ContainerMountUsageRefreshPath]['post'];

export type ContainerListQuery = NonNullable<GetContainersOperation['parameters']['query']>;
export type ContainerListQueryWithOrchestrator = ContainerListQuery & {
  orchestrator?: ContainerOrchestratorType;
  source_scope_kind?: ContainerListSourceScopeKind;
  source_scope?: string;
};
export type ContainerListSourceScopeKind =
  'compose_project' | 'compose_service' | 'swarm_stack' | 'swarm_task' | 'kubernetes_namespace' | 'kubernetes_pod';
export type ContainerLogQuery = NonNullable<GetContainerLogsOperation['parameters']['query']>;
export type ContainerRuntimeEventsPathParams = GetContainerEventsOperation['parameters']['path'];
export type ContainerMountUsagePathParams = GetContainerMountUsageOperation['parameters']['path'];
export type ContainerMountUsageRefreshPathParams = PostContainerMountUsageRefreshOperation['parameters']['path'];

export type ContainerSourceGroupKind = Extract<
  ContainerListSourceScopeKind,
  'compose_project' | 'swarm_stack' | 'kubernetes_namespace'
>;
export type ContainerSourceMemberKind = Extract<
  ContainerListSourceScopeKind,
  'compose_service' | 'swarm_task' | 'kubernetes_pod'
>;
export type ContainerFilters = {
  keyword: string;
  deploymentType: ContainerOrchestratorType | 'all';
  runtimeTargetId: number | 'all';
  status: ContainerState | 'all';
  health: ContainerHealth | 'all';
};
