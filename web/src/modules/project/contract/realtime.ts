import type { components } from '@/contracts/openapi/generated/schema';
import { isRealtimePayloadObject, parseRealtimeEnvelopeData } from '@/shared/realtime';

const PROJECT_REALTIME_TOPIC = {
  LIST_SUMMARY: 'project.list.summary',
  DETAIL_PREFIX: 'project.detail:',
  LOGS_PREFIX: 'project.logs:',
} as const;

export function getProjectListSummaryTopicName() {
  return PROJECT_REALTIME_TOPIC.LIST_SUMMARY;
}

export function buildProjectDetailTopicName(projectId: number) {
  return `${PROJECT_REALTIME_TOPIC.DETAIL_PREFIX}${projectId}`;
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

export type ProjectDetailRealtimePayload = {
  topic: string;
  project_id: number;
  published_at: string;
  detail: ProjectDetailResponse;
  overview: ProjectOverviewResponse;
  services: ProjectServicesResponse;
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

export function parseProjectDetailRealtimePayload(raw: unknown): ProjectDetailRealtimePayload | null {
  const data = parseRealtimeEnvelopeData(raw);
  if (!isRealtimePayloadObject(data)) {
    return null;
  }
  if (
    typeof data.topic !== 'string' ||
    typeof data.project_id !== 'number' ||
    typeof data.published_at !== 'string' ||
    !isRealtimePayloadObject(data.detail) ||
    !isRealtimePayloadObject(data.overview) ||
    !isRealtimePayloadObject(data.services)
  ) {
    return null;
  }
  return data as ProjectDetailRealtimePayload;
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
