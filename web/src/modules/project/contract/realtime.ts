import type { components } from '@/contracts/openapi/generated/schema';
import { isRealtimePayloadObject, parseRealtimeEnvelopeData } from '@/shared/realtime';

const PROJECT_REALTIME_TOPIC = {
  LIST_SUMMARY: 'project.list.summary',
  RUNTIME_PREFIX: 'project.runtime:',
  LIFECYCLE_CONFIG_PREFIX: 'project.lifecycle-config:',
  LOGS_PREFIX: 'project.logs:',
} as const;

export function getProjectListSummaryTopicName(): string {
  return PROJECT_REALTIME_TOPIC.LIST_SUMMARY;
}

export function buildProjectRuntimeTopicName(applicationId: string): string {
  return `${PROJECT_REALTIME_TOPIC.RUNTIME_PREFIX}${applicationId}`;
}

export function buildProjectLifecycleConfigTopicName(applicationId: string): string {
  return `${PROJECT_REALTIME_TOPIC.LIFECYCLE_CONFIG_PREFIX}${applicationId}`;
}

export function buildProjectLogsTopicName(applicationId: string): string {
  return `${PROJECT_REALTIME_TOPIC.LOGS_PREFIX}${applicationId}`;
}

type ProjectDetailResponse = components['schemas']['ProjectDetailResponse'];
type ProjectOverviewResponse = components['schemas']['ProjectOverviewResponse'];
type ProjectServicesResponse = components['schemas']['ProjectServicesResponse'];
type ProjectLogEntry = components['schemas']['project-log-entry'];
type ProjectContainerCounts = components['schemas']['ProjectContainerCounts'];
type ProjectDriftStatus = components['schemas']['ProjectDriftStatus'];
type ProjectRuntimeStatus = ProjectDetailResponse['runtime_status'];

export type ProjectListSummaryRealtimeItem = {
  application_id: string;
  runtime_status: ProjectRuntimeStatus;
  service_count: number;
  container_counts: ProjectContainerCounts;
  drift_status: ProjectDriftStatus;
};

export type ProjectRuntimeRealtimePayload = {
  topic: string;
  application_id: string;
  published_at: string;
  detail: ProjectDetailResponse;
  overview: ProjectOverviewResponse;
  services: ProjectServicesResponse;
};

export type ProjectLifecycleConfigRealtimePayload = {
  topic: string;
  application_id: string;
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

/** 实时通道是外部输入；结构或主题不匹配时必须丢弃，不能让页面状态被污染。 */
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
        typeof item.application_id === 'string' &&
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

function parseProjectDetailEnvelopeData(
  raw: unknown,
  expectedTopic: (applicationId: string) => string,
  validator?: (data: Record<string, unknown>) => boolean,
): Record<string, unknown> | null {
  const data = parseRealtimeEnvelopeData(raw);
  if (!isRealtimePayloadObject(data)) {
    return null;
  }
  if (
    typeof data.topic !== 'string' ||
    typeof data.application_id !== 'string' ||
    data.topic !== expectedTopic(data.application_id) ||
    typeof data.published_at !== 'string' ||
    !isRealtimePayloadObject(data.detail) ||
    (validator && !validator(data))
  ) {
    return null;
  }
  return data;
}

export function parseProjectRuntimeRealtimePayload(raw: unknown): ProjectRuntimeRealtimePayload | null {
  const data = parseProjectDetailEnvelopeData(
    raw,
    buildProjectRuntimeTopicName,
    (value) => isRealtimePayloadObject(value.overview) && isRealtimePayloadObject(value.services),
  );
  return data as ProjectRuntimeRealtimePayload | null;
}

export function parseProjectLifecycleConfigRealtimePayload(raw: unknown): ProjectLifecycleConfigRealtimePayload | null {
  const data = parseProjectDetailEnvelopeData(raw, buildProjectLifecycleConfigTopicName);
  return data as ProjectLifecycleConfigRealtimePayload | null;
}

export function parseProjectLogsRealtimePayload(raw: unknown): ProjectLogsRealtimePayload | null {
  const data = parseRealtimeEnvelopeData(raw);
  if (!isRealtimePayloadObject(data)) {
    return null;
  }
  if (
    typeof data.topic !== 'string' ||
    typeof data.application_id !== 'string' ||
    data.topic !== buildProjectLogsTopicName(data.application_id) ||
    !isRealtimePayloadObject(data.entry) ||
    typeof data.entry.line !== 'string'
  ) {
    return null;
  }
  return data as ProjectLogsRealtimePayload;
}
