import { buildContainerStatsTopicName } from '../contract/realtime';
import type { ContainerDetailRecord, ContainerSummaryRecord } from '../types/container';

type ContainerResourceSummary = NonNullable<ContainerSummaryRecord['resource']>;

function isObject(value: unknown): value is Record<string, unknown> {
  return Boolean(value && typeof value === 'object');
}

export function buildContainerStatsTopic(containerId: string) {
  return buildContainerStatsTopicName(containerId);
}

export function parseContainerStatsPayload(raw: unknown) {
  if (typeof raw !== 'string') {
    return null;
  }

  try {
    const parsed = JSON.parse(raw) as unknown;
    if (!isObject(parsed)) {
      return null;
    }

    const eventData = isObject(parsed.data) ? parsed.data : null;
    if (!eventData) {
      return null;
    }
    const resource = isObject(eventData.resource) ? (eventData.resource as ContainerResourceSummary) : null;
    if (!resource) {
      return null;
    }
    const id = typeof eventData.id === 'string' ? eventData.id : undefined;

    return {
      id,
      resource,
    };
  } catch {
    return null;
  }
}

export function applyRealtimeResourceToDetail(
  detail: ContainerDetailRecord,
  resource: ContainerResourceSummary,
): ContainerDetailRecord {
  return {
    ...detail,
    resource: {
      ...resource,
    },
  };
}

export function mergeDetailStructurePreservingRealtimeResource(
  current: ContainerDetailRecord | null | undefined,
  next: ContainerDetailRecord,
): ContainerDetailRecord {
  if (!current?.resource) {
    return next;
  }

  return {
    ...next,
    resource: {
      ...current.resource,
    },
  };
}
