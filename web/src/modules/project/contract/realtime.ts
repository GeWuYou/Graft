import type { components } from '@/contracts/openapi/generated/schema';
import { isRealtimePayloadObject, parseRealtimeEnvelopeData } from '@/shared/realtime';

const PROJECT_REALTIME_TOPIC = {
  DETAIL_PREFIX: 'project.detail:',
  LOGS_PREFIX: 'project.logs:',
} as const;

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
