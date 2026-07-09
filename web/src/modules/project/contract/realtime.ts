import type { components } from '@/contracts/openapi/generated/schema';
import { isRealtimePayloadObject, parseRealtimeEnvelopeData } from '@/shared/realtime';

const PROJECT_REALTIME_TOPIC = {
  LIST_SUMMARY: 'project.list.summary',
  RUNTIME_PREFIX: 'project.runtime:',
  LIFECYCLE_CONFIG_PREFIX: 'project.lifecycle-config:',
  LOGS_PREFIX: 'project.logs:',
} as const;

export function getProjectListSummaryTopicName() {
  return PROJECT_REALTIME_TOPIC.LIST_SUMMARY;
}

export function buildProjectRuntimeTopicName(projectId: number) {
  return `${PROJECT_REALTIME_TOPIC.RUNTIME_PREFIX}${projectId}`;
}

export function buildProjectLifecycleConfigTopicName(projectId: number) {
  return `${PROJECT_REALTIME_TOPIC.LIFECYCLE_CONFIG_PREFIX}${projectId}`;
}

export function buildProjectLogsTopicName(projectId: number) {
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

export function parseProjectListSummaryRealtimePayload(raw: unknown): ProjectListSummaryRealtimePayload | null {
  const data = parseRealtimeEnvelopeData(raw);
  if (!isRealtimePayloadObject(data)) {
    return null;
  }
  if (
    typeof data.topic !== 'string' ||
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

function parseProjectDetailEnvelopeData(
  raw: unknown,
  validator?: (data: Record<string, unknown>) => boolean,
): Record<string, unknown> | null {
  const data = parseRealtimeEnvelopeData(raw);
  if (!isRealtimePayloadObject(data)) {
    return null;
  }
  if (
    typeof data.topic !== 'string' ||
    typeof data.project_id !== 'number' ||
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
    (value) => isRealtimePayloadObject(value.overview) && isRealtimePayloadObject(value.services),
  );
  return data as ProjectRuntimeRealtimePayload | null;
}

export function parseProjectLifecycleConfigRealtimePayload(raw: unknown): ProjectLifecycleConfigRealtimePayload | null {
  const data = parseProjectDetailEnvelopeData(raw);
  return data as ProjectLifecycleConfigRealtimePayload | null;
}

export function parseProjectLogsRealtimePayload(raw: unknown): ProjectLogsRealtimePayload | null {
  const data = parseRealtimeEnvelopeData(raw);
  if (!isRealtimePayloadObject(data)) {
    return null;
  }
  if (typeof data.topic !== 'string' || !isRealtimePayloadObject(data.entry) || typeof data.entry.line !== 'string') {
    return null;
  }
  return data as ProjectLogsRealtimePayload;
}
