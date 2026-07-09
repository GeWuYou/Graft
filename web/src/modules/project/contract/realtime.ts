import type { components } from '@/contracts/openapi/generated/schema';
import { isRealtimePayloadObject, parseRealtimeEnvelopeData } from '@/shared/realtime';

const PROJECT_REALTIME_TOPIC = {
  LIST_SUMMARY: 'project.list.summary',
  RUNTIME_PREFIX: 'project.runtime:',
  LIFECYCLE_CONFIG_PREFIX: 'project.lifecycle-config:',
  LOGS_PREFIX: 'project.logs:',
} as const;

/**
 * 获取项目列表摘要的实时通信主题名称。
 *
 * @returns 项目列表摘要主题名称
 */
export function getProjectListSummaryTopicName(): string {
  return PROJECT_REALTIME_TOPIC.LIST_SUMMARY;
}

/**
 * 构建项目运行时实时通信主题名称。
 *
 * @param projectId - 项目标识
 * @returns 包含项目标识的运行时主题名称
 */
export function buildProjectRuntimeTopicName(projectId: number): string {
  return `${PROJECT_REALTIME_TOPIC.RUNTIME_PREFIX}${projectId}`;
}

/**
 * 构建项目生命周期配置实时主题名称。
 *
 * @param projectId - 项目标识
 * @returns 拼接项目标识后的生命周期配置实时主题名称
 */
export function buildProjectLifecycleConfigTopicName(projectId: number): string {
  return `${PROJECT_REALTIME_TOPIC.LIFECYCLE_CONFIG_PREFIX}${projectId}`;
}

/**
 * 构建项目日志实时消息的 topic 名称。
 *
 * @param projectId - 项目标识
 * @returns 拼接项目标识后的日志 topic 名称
 */
export function buildProjectLogsTopicName(projectId: number): string {
  return `${PROJECT_REALTIME_TOPIC.LOGS_PREFIX}${projectId}`;
}

type ProjectDetailResponse = components['schemas']['ProjectDetailResponse'];
type ProjectOverviewResponse = components['schemas']['ProjectOverviewResponse'];
type ProjectServicesResponse = components['schemas']['ProjectServicesResponse'];
type ProjectLogEntry = components['schemas']['project-log-entry'];
type ProjectContainerCounts = components['schemas']['ProjectContainerCounts'];
type ProjectDriftStatus = components['schemas']['ProjectDriftStatus'];
type ProjectRuntimeStatus = ProjectDetailResponse['runtime_status'];

export type ProjectListSummaryRealtimeItem = {
  project_id: number;
  runtime_status: ProjectRuntimeStatus;
  service_count: number;
  container_counts: ProjectContainerCounts;
  drift_status: ProjectDriftStatus;
};

export type ProjectRuntimeRealtimePayload = {
  topic: string;
  project_id: number;
  published_at: string;
  detail: ProjectDetailResponse;
  overview: ProjectOverviewResponse;
  services: ProjectServicesResponse;
};

export type ProjectLifecycleConfigRealtimePayload = {
  topic: string;
  project_id: number;
  published_at: string;
  detail: ProjectDetailResponse;
};

export type ProjectLogsRealtimePayload = {
  topic: string;
  entry: ProjectLogEntry;
};

export type ProjectListSummaryRealtimePayload = {
  topic: string;
  published_at: string;
  items: ProjectListSummaryRealtimeItem[];
};

/**
 * 解析项目列表摘要实时消息载荷。
 *
 * @param raw - 待解析的原始消息数据
 * @returns 验证通过的项目列表摘要实时载荷，数据格式无效时返回 `null`
 */
export function parseProjectListSummaryRealtimePayload(raw: unknown): ProjectListSummaryRealtimePayload | null {
  const data = parseRealtimeEnvelopeData(raw);
  if (!isRealtimePayloadObject(data)) {
    return null;
  }
  if (
    data.topic !== getProjectListSummaryTopicName() ||
    typeof data.published_at !== 'string' ||
    !Array.isArray(data.items) ||
    !data.items.every(
      (item) =>
        isRealtimePayloadObject(item) &&
        typeof item.project_id === 'number' &&
        typeof item.runtime_status === 'string' &&
        typeof item.service_count === 'number' &&
        isRealtimePayloadObject(item.container_counts) &&
        typeof item.drift_status === 'string',
    )
  ) {
    return null;
  }
  return data as ProjectListSummaryRealtimePayload;
}

/**
 * 校验项目详情实时消息信封并返回其数据。
 *
 * @param raw - 待解析的原始实时消息
 * @param validator - 可选的附加数据校验函数
 * @returns 校验通过的消息数据，或 `null`
 */
function parseProjectDetailEnvelopeData(
  raw: unknown,
  expectedTopic: (projectId: number) => string,
  validator?: (data: Record<string, unknown>) => boolean,
): Record<string, unknown> | null {
  const data = parseRealtimeEnvelopeData(raw);
  if (!isRealtimePayloadObject(data)) {
    return null;
  }
  if (
    typeof data.topic !== 'string' ||
    typeof data.project_id !== 'number' ||
    data.topic !== expectedTopic(data.project_id) ||
    typeof data.published_at !== 'string' ||
    !isRealtimePayloadObject(data.detail) ||
    (validator && !validator(data))
  ) {
    return null;
  }
  return data;
}

/**
 * 解析项目运行时实时消息载荷。
 *
 * @param raw - 待解析的原始数据
 * @returns 有效的项目运行时实时载荷；数据格式无效时返回 `null`
 */
export function parseProjectRuntimeRealtimePayload(raw: unknown): ProjectRuntimeRealtimePayload | null {
  const data = parseProjectDetailEnvelopeData(
    raw,
    buildProjectRuntimeTopicName,
    (value) => isRealtimePayloadObject(value.overview) && isRealtimePayloadObject(value.services),
  );
  return data as ProjectRuntimeRealtimePayload | null;
}

/**
 * 解析项目生命周期配置实时消息载荷。
 *
 * @param raw - 待解析的原始消息数据
 * @returns 有效的项目生命周期配置实时载荷，或 `null`（当数据格式无效时）
 */
export function parseProjectLifecycleConfigRealtimePayload(raw: unknown): ProjectLifecycleConfigRealtimePayload | null {
  const data = parseProjectDetailEnvelopeData(raw, buildProjectLifecycleConfigTopicName);
  return data as ProjectLifecycleConfigRealtimePayload | null;
}

/**
 * 解析项目日志实时消息载荷。
 *
 * @param raw - 待解析的原始数据
 * @returns 验证通过的项目日志实时载荷；数据格式无效时返回 `null`
 */
export function parseProjectLogsRealtimePayload(raw: unknown): ProjectLogsRealtimePayload | null {
  const data = parseRealtimeEnvelopeData(raw);
  if (!isRealtimePayloadObject(data)) {
    return null;
  }
  if (
    typeof data.topic !== 'string' ||
    !data.topic.startsWith(PROJECT_REALTIME_TOPIC.LOGS_PREFIX) ||
    !isRealtimePayloadObject(data.entry) ||
    typeof data.entry.line !== 'string'
  ) {
    return null;
  }
  return data as ProjectLogsRealtimePayload;
}
