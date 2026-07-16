import type { components } from '@/contracts/openapi/generated/schema';
import { isRealtimePayloadObject, parseRealtimeEnvelopeData } from '@/shared/realtime';

const APPLICATION_REALTIME_TOPIC = {
  LIST_SUMMARY: 'application.list.summary',
  RUNTIME_PREFIX: 'application.runtime:',
  LIFECYCLE_CONFIG_PREFIX: 'application.lifecycle-config:',
  LOGS_PREFIX: 'application.logs:',
} as const;

export function getApplicationListSummaryTopicName(): string {
  return APPLICATION_REALTIME_TOPIC.LIST_SUMMARY;
}

export function buildApplicationRuntimeTopicName(applicationId: string): string {
  return `${APPLICATION_REALTIME_TOPIC.RUNTIME_PREFIX}${applicationId}`;
}

export function buildApplicationLifecycleConfigTopicName(applicationId: string): string {
  return `${APPLICATION_REALTIME_TOPIC.LIFECYCLE_CONFIG_PREFIX}${applicationId}`;
}

export function buildApplicationLogsTopicName(applicationId: string): string {
  return `${APPLICATION_REALTIME_TOPIC.LOGS_PREFIX}${applicationId}`;
}

type ApplicationDetailResponse = components['schemas']['ApplicationDetailResponse'];
type ApplicationOverviewResponse = components['schemas']['ApplicationOverviewResponse'];
type ApplicationServicesResponse = components['schemas']['ApplicationServicesResponse'];
type ApplicationLogEntry = components['schemas']['ApplicationLogEntry'];
type ApplicationContainerCounts = components['schemas']['ApplicationContainerCounts'];
type ApplicationDriftStatus = components['schemas']['ApplicationDriftStatus'];
type ApplicationRuntimeStatus = ApplicationDetailResponse['runtime_status'];

export type ApplicationListSummaryRealtimeItem = {
  application_id: string;
  runtime_status: ApplicationRuntimeStatus;
  service_count: number;
  container_counts: ApplicationContainerCounts;
  drift_status: ApplicationDriftStatus;
};

export type ApplicationRuntimeRealtimePayload = {
  topic: string;
  application_id: string;
  published_at: string;
  detail: ApplicationDetailResponse;
  overview: ApplicationOverviewResponse;
  services: ApplicationServicesResponse;
};

export type ApplicationLifecycleConfigRealtimePayload = {
  topic: string;
  application_id: string;
  published_at: string;
  detail: ApplicationDetailResponse;
};

export type ApplicationLogsRealtimePayload = {
  topic: string;
  entry: ApplicationLogEntry;
};

export type ApplicationListSummaryRealtimePayload = {
  topic: string;
  published_at: string;
  items: ApplicationListSummaryRealtimeItem[];
};

/** 实时通道是外部输入；结构或主题不匹配时必须丢弃，不能让页面状态被污染。 */
export function parseApplicationListSummaryRealtimePayload(raw: unknown): ApplicationListSummaryRealtimePayload | null {
  const data = parseRealtimeEnvelopeData(raw);
  if (!isRealtimePayloadObject(data)) {
    return null;
  }
  if (
    data.topic !== getApplicationListSummaryTopicName() ||
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
  return data as ApplicationListSummaryRealtimePayload;
}

function parseApplicationDetailEnvelopeData(
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

export function parseApplicationRuntimeRealtimePayload(raw: unknown): ApplicationRuntimeRealtimePayload | null {
  const data = parseApplicationDetailEnvelopeData(
    raw,
    buildApplicationRuntimeTopicName,
    (value) => isRealtimePayloadObject(value.overview) && isRealtimePayloadObject(value.services),
  );
  return data as ApplicationRuntimeRealtimePayload | null;
}

export function parseApplicationLifecycleConfigRealtimePayload(
  raw: unknown,
): ApplicationLifecycleConfigRealtimePayload | null {
  const data = parseApplicationDetailEnvelopeData(raw, buildApplicationLifecycleConfigTopicName);
  return data as ApplicationLifecycleConfigRealtimePayload | null;
}

export function parseApplicationLogsRealtimePayload(raw: unknown): ApplicationLogsRealtimePayload | null {
  const data = parseRealtimeEnvelopeData(raw);
  if (!isRealtimePayloadObject(data)) {
    return null;
  }
  if (
    typeof data.topic !== 'string' ||
    typeof data.application_id !== 'string' ||
    data.topic !== buildApplicationLogsTopicName(data.application_id) ||
    !isRealtimePayloadObject(data.entry) ||
    typeof data.entry.line !== 'string'
  ) {
    return null;
  }
  return data as ApplicationLogsRealtimePayload;
}
