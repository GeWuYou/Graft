import { reactive } from 'vue';

import { openRealtimeTopicSocket, type RealtimeTopicSocketController } from '@/shared/realtime';

import type { ContainerDetailRecord, ContainerResourceSummary, ContainerSummaryRecord } from '../types/container';
import { buildContainerStatsTopic, parseContainerStatsPayload } from './realtime-stats';

type ContainerMetadataRecord = Omit<ContainerSummaryRecord, 'resource'>;
type ContainerDetailMetadataRecord = Omit<ContainerDetailRecord, 'resource'>;

type StatsSnapshotSource = 'http-seed' | 'realtime';
type RealtimeSocketState = 'idle' | 'connecting' | 'open' | 'closed' | 'error';

export type ContainerStatsSnapshot = {
  resource: ContainerResourceSummary;
  source: StatsSnapshotSource;
};

type ContainerStatsEntry = {
  snapshot: ContainerStatsSnapshot | null;
};

type ContainerStatsSubscriptionEntry = {
  controller: RealtimeTopicSocketController | null;
  idleTimer: number | null;
  refCount: number;
  state: RealtimeSocketState;
};

type ContainerStatsManagerState = {
  detailMetadataById: Map<string, ContainerDetailMetadataRecord>;
  listOrder: string[];
  listMetadataById: Map<string, ContainerMetadataRecord>;
  statsById: Map<string, ContainerStatsEntry>;
  subscriptionsById: Map<string, ContainerStatsSubscriptionEntry>;
};

const SUBSCRIPTION_IDLE_GRACE_MS = 10_000;

const state = reactive<ContainerStatsManagerState>({
  detailMetadataById: new Map<string, ContainerDetailMetadataRecord>(),
  listOrder: [],
  listMetadataById: new Map<string, ContainerMetadataRecord>(),
  statsById: new Map<string, ContainerStatsEntry>(),
  subscriptionsById: new Map<string, ContainerStatsSubscriptionEntry>(),
});

function normalizeCollectedAt(value?: string | null) {
  return value?.trim() || null;
}

function getCollectedAtValue(resource?: ContainerResourceSummary | null) {
  return normalizeCollectedAt(resource?.collected_at);
}

function isNewerStatsSnapshot(
  current: ContainerStatsSnapshot | null,
  candidate: ContainerResourceSummary,
  source: StatsSnapshotSource,
) {
  if (!current) {
    return true;
  }

  const currentCollectedAt = getCollectedAtValue(current.resource);
  const candidateCollectedAt = getCollectedAtValue(candidate);

  if (candidateCollectedAt && currentCollectedAt) {
    return candidateCollectedAt >= currentCollectedAt;
  }
  if (candidateCollectedAt && !currentCollectedAt) {
    return true;
  }
  if (!candidateCollectedAt && currentCollectedAt) {
    return false;
  }

  if (current.source === 'realtime' && source === 'http-seed') {
    return false;
  }

  return true;
}

function upsertStatsSnapshot(containerId: string, resource: ContainerResourceSummary, source: StatsSnapshotSource) {
  const current = state.statsById.get(containerId)?.snapshot ?? null;
  if (!isNewerStatsSnapshot(current, resource, source)) {
    return current;
  }

  const nextSnapshot: ContainerStatsSnapshot = {
    resource: {
      ...resource,
    },
    source,
  };
  state.statsById.set(containerId, { snapshot: nextSnapshot });
  return nextSnapshot;
}

function clearListMetadata() {
  state.listOrder = [];
  state.listMetadataById.clear();
}

function ensureSubscriptionEntry(containerId: string) {
  const current = state.subscriptionsById.get(containerId);
  if (current) {
    return current;
  }

  const next: ContainerStatsSubscriptionEntry = {
    controller: null,
    idleTimer: null,
    refCount: 0,
    state: 'idle',
  };
  state.subscriptionsById.set(containerId, next);
  return next;
}

function clearIdleTimer(entry: ContainerStatsSubscriptionEntry) {
  if (entry.idleTimer !== null) {
    clearTimeout(entry.idleTimer);
    entry.idleTimer = null;
  }
}

function closeSubscriptionEntry(entry: ContainerStatsSubscriptionEntry) {
  clearIdleTimer(entry);
  entry.controller?.close();
  entry.controller = null;
  entry.state = 'idle';
}

function connectSubscription(containerId: string, entry: ContainerStatsSubscriptionEntry) {
  if (entry.controller) {
    return;
  }

  entry.state = 'connecting';
  entry.controller = openRealtimeTopicSocket({
    topic: buildContainerStatsTopic(containerId),
    parseMessage: parseContainerStatsPayload,
    onStateChange: (nextState) => {
      const latestEntry = state.subscriptionsById.get(containerId);
      if (!latestEntry) {
        return;
      }
      latestEntry.state = nextState;
      if (nextState === 'idle' && latestEntry.refCount > 0 && !latestEntry.controller) {
        connectSubscription(containerId, latestEntry);
      }
    },
    onMessage: (payload) => {
      if (payload.id && payload.id !== containerId) {
        return;
      }
      if (!payload.resource) {
        return;
      }
      applyContainerRealtimeStats(containerId, payload.resource);
    },
  });
}

function splitSummaryRecord(record: ContainerSummaryRecord) {
  const { resource, ...metadata } = record;
  return {
    metadata: metadata as ContainerMetadataRecord,
    resource,
  };
}

function splitDetailRecord(record: ContainerDetailRecord) {
  const { resource, ...metadata } = record;
  return {
    metadata: metadata as ContainerDetailMetadataRecord,
    resource,
  };
}

export function resetContainerStatsManager() {
  state.subscriptionsById.forEach((entry) => {
    closeSubscriptionEntry(entry);
  });
  state.listOrder = [];
  state.listMetadataById.clear();
  state.detailMetadataById.clear();
  state.statsById.clear();
  state.subscriptionsById.clear();
}

export function seedContainerList(items: ContainerSummaryRecord[]) {
  clearListMetadata();
  items.forEach((item) => {
    state.listOrder.push(item.id);
    const { metadata, resource } = splitSummaryRecord(item);
    state.listMetadataById.set(item.id, metadata);
    if (resource) {
      upsertStatsSnapshot(item.id, resource, 'http-seed');
    }
  });
}

export function seedContainerDetail(detail: ContainerDetailRecord) {
  const { metadata, resource } = splitDetailRecord(detail);
  state.detailMetadataById.set(detail.id, metadata);
  if (resource) {
    upsertStatsSnapshot(detail.id, resource, 'http-seed');
  }
}

export function clearContainerDetail(containerId?: string) {
  if (!containerId) {
    state.detailMetadataById.clear();
    return;
  }
  state.detailMetadataById.delete(containerId);
}

export function clearContainerListMetadata() {
  clearListMetadata();
}

export function applyContainerRealtimeStats(containerId: string, resource: ContainerResourceSummary) {
  return upsertStatsSnapshot(containerId, resource, 'realtime');
}

export function acquireContainerStatsSubscription(containerId: string) {
  const normalizedContainerId = containerId.trim();
  if (!normalizedContainerId) {
    return;
  }

  const entry = ensureSubscriptionEntry(normalizedContainerId);
  clearIdleTimer(entry);
  entry.refCount += 1;
  if (!entry.controller) {
    connectSubscription(normalizedContainerId, entry);
  }
}

export function releaseContainerStatsSubscription(containerId: string) {
  const normalizedContainerId = containerId.trim();
  if (!normalizedContainerId) {
    return;
  }

  const entry = state.subscriptionsById.get(normalizedContainerId);
  if (!entry) {
    return;
  }

  entry.refCount = Math.max(0, entry.refCount - 1);
  if (entry.refCount > 0) {
    return;
  }

  clearIdleTimer(entry);
  entry.idleTimer = window.setTimeout(() => {
    const latestEntry = state.subscriptionsById.get(normalizedContainerId);
    if (!latestEntry || latestEntry.refCount > 0) {
      return;
    }
    closeSubscriptionEntry(latestEntry);
  }, SUBSCRIPTION_IDLE_GRACE_MS);
}

export function selectContainerStatsRealtimeState(containerId: string): RealtimeSocketState {
  return state.subscriptionsById.get(containerId)?.state ?? 'idle';
}

function selectContainerSummaryView(containerId: string): ContainerSummaryRecord | null {
  const metadata = state.listMetadataById.get(containerId);
  if (!metadata) {
    return null;
  }

  const snapshot = state.statsById.get(containerId)?.snapshot ?? null;
  return {
    ...metadata,
    resource: snapshot?.resource,
  };
}

export function selectContainerListViews(): ContainerSummaryRecord[] {
  return state.listOrder.reduce<ContainerSummaryRecord[]>((items, containerId) => {
    const next = selectContainerSummaryView(containerId);
    if (next) {
      items.push(next);
    }
    return items;
  }, []);
}

export function selectContainerDetailView(containerId: string): ContainerDetailRecord | null {
  const metadata = state.detailMetadataById.get(containerId);
  if (!metadata) {
    return null;
  }

  const snapshot = state.statsById.get(containerId)?.snapshot ?? null;
  return {
    ...metadata,
    resource: snapshot?.resource,
  };
}
