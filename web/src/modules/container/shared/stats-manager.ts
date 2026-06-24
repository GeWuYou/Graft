import { reactive } from 'vue';

import { openRealtimeTopicSocket, type RealtimeTopicSocketController } from '@/shared/realtime';

import type { ContainerDetailRecord, ContainerResourceSummary, ContainerSummaryRecord } from '../types/container';
import { buildContainerStatsTopic, parseContainerStatsPayload } from './realtime-stats';

type ContainerMetadataRecord = Omit<ContainerSummaryRecord, 'resource'>;
type ContainerDetailMetadataRecord = Omit<ContainerDetailRecord, 'resource'>;
type ContainerSummaryCollectionKey = string;

type StatsSnapshotSource = 'http-seed' | 'realtime';
type RealtimeSocketState = 'idle' | 'connecting' | 'open' | 'closed' | 'error';

export type ContainerStatsSnapshot = {
  resource: ContainerResourceSummary;
  source: StatsSnapshotSource;
};

export type ContainerStatsChangeDirection = 'down' | 'none' | 'up';

export type ContainerStatsChangeState = {
  changedAt: number | null;
  cpu: ContainerStatsChangeDirection;
  memory: ContainerStatsChangeDirection;
};

type ContainerStatsEntry = {
  change: ContainerStatsChangeState;
  history: ContainerStatsSnapshot[];
  previousSnapshot: ContainerStatsSnapshot | null;
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
  listCollections: Map<ContainerSummaryCollectionKey, string[]>;
  listMetadataByCollection: Map<ContainerSummaryCollectionKey, Map<string, ContainerMetadataRecord>>;
  statsById: Map<string, ContainerStatsEntry>;
  subscriptionsById: Map<string, ContainerStatsSubscriptionEntry>;
};

const SUBSCRIPTION_IDLE_GRACE_MS = 10_000;
const DEFAULT_CONTAINER_LIST_COLLECTION_KEY = 'container:list';
const CONTAINER_STATS_HISTORY_LIMIT = 12;
const CONTAINER_STATS_CHANGE_HIGHLIGHT_MS = 800;

const state = reactive<ContainerStatsManagerState>({
  detailMetadataById: new Map<string, ContainerDetailMetadataRecord>(),
  listCollections: new Map<ContainerSummaryCollectionKey, string[]>(),
  listMetadataByCollection: new Map<ContainerSummaryCollectionKey, Map<string, ContainerMetadataRecord>>(),
  statsById: new Map<string, ContainerStatsEntry>(),
  subscriptionsById: new Map<string, ContainerStatsSubscriptionEntry>(),
});

function normalizeCollectedAt(value?: string | null) {
  return value?.trim() || null;
}

function getCollectedAtValue(resource?: ContainerResourceSummary | null) {
  return normalizeCollectedAt(resource?.collected_at);
}

function compareMetricDirection(previous?: number | null, next?: number | null): ContainerStatsChangeDirection {
  if (typeof previous !== 'number' || Number.isNaN(previous) || typeof next !== 'number' || Number.isNaN(next)) {
    return 'none';
  }
  if (next > previous) {
    return 'up';
  }
  if (next < previous) {
    return 'down';
  }
  return 'none';
}

function buildChangeState(
  current: ContainerStatsSnapshot | null,
  nextSnapshot: ContainerStatsSnapshot,
  source: StatsSnapshotSource,
): ContainerStatsChangeState {
  const currentTime = source === 'realtime' ? Date.now() : null;
  const cpu = compareMetricDirection(current?.resource.cpu_percent, nextSnapshot.resource.cpu_percent);
  const memory = compareMetricDirection(current?.resource.memory_percent, nextSnapshot.resource.memory_percent);
  const changed = source === 'realtime' && (cpu !== 'none' || memory !== 'none');

  return {
    changedAt: changed ? currentTime : null,
    cpu,
    memory,
  };
}

function isChangeStateFresh(change: ContainerStatsChangeState) {
  return typeof change.changedAt === 'number' && Date.now() - change.changedAt <= CONTAINER_STATS_CHANGE_HIGHLIGHT_MS;
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
  const currentEntry = state.statsById.get(containerId);
  const current = currentEntry?.snapshot ?? null;
  if (!isNewerStatsSnapshot(current, resource, source)) {
    return current;
  }

  const nextSnapshot: ContainerStatsSnapshot = {
    resource: {
      ...resource,
    },
    source,
  };
  const nextChange = buildChangeState(current, nextSnapshot, source);
  state.statsById.set(containerId, {
    change: nextChange,
    history: [...(currentEntry?.history ?? []), nextSnapshot].slice(-CONTAINER_STATS_HISTORY_LIMIT),
    previousSnapshot: current,
    snapshot: nextSnapshot,
  });
  return nextSnapshot;
}

function normalizeCollectionKey(collectionKey?: string) {
  return collectionKey?.trim() || DEFAULT_CONTAINER_LIST_COLLECTION_KEY;
}

function ensureListCollection(collectionKey?: string) {
  const normalizedCollectionKey = normalizeCollectionKey(collectionKey);
  if (!state.listCollections.has(normalizedCollectionKey)) {
    state.listCollections.set(normalizedCollectionKey, []);
  }
  if (!state.listMetadataByCollection.has(normalizedCollectionKey)) {
    state.listMetadataByCollection.set(normalizedCollectionKey, new Map<string, ContainerMetadataRecord>());
  }

  return {
    key: normalizedCollectionKey,
    order: state.listCollections.get(normalizedCollectionKey)!,
    metadataById: state.listMetadataByCollection.get(normalizedCollectionKey)!,
  };
}

function clearListMetadata(collectionKey?: string) {
  const targetCollection = ensureListCollection(collectionKey);
  targetCollection.order.length = 0;
  targetCollection.metadataById.clear();
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

function attachLatestResource<TMetadata extends ContainerMetadataRecord | ContainerDetailMetadataRecord>(
  containerId: string,
  metadata: TMetadata | undefined,
) {
  if (!metadata) {
    return null;
  }

  const snapshot = state.statsById.get(containerId)?.snapshot ?? null;
  return {
    ...metadata,
    resource: snapshot?.resource,
  };
}

export function resetContainerStatsManager() {
  state.subscriptionsById.forEach((entry) => {
    closeSubscriptionEntry(entry);
  });
  state.listCollections.clear();
  state.listMetadataByCollection.clear();
  state.detailMetadataById.clear();
  state.statsById.clear();
  state.subscriptionsById.clear();
}

export function seedContainerList(items: ContainerSummaryRecord[], collectionKey?: string) {
  const targetCollection = ensureListCollection(collectionKey);
  targetCollection.order.length = 0;
  targetCollection.metadataById.clear();
  items.forEach((item) => {
    targetCollection.order.push(item.id);
    const { metadata, resource } = splitSummaryRecord(item);
    targetCollection.metadataById.set(item.id, metadata);
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

export function clearContainerSummaryCollection(collectionKey: string) {
  clearListMetadata(collectionKey);
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
  const metadata = state.listMetadataByCollection.get(DEFAULT_CONTAINER_LIST_COLLECTION_KEY)?.get(containerId);
  return attachLatestResource(containerId, metadata);
}

export function selectContainerListViews(): ContainerSummaryRecord[] {
  const order = state.listCollections.get(DEFAULT_CONTAINER_LIST_COLLECTION_KEY) ?? [];
  return order.reduce<ContainerSummaryRecord[]>((items, containerId) => {
    const next = selectContainerSummaryView(containerId);
    if (next) {
      items.push(next);
    }
    return items;
  }, []);
}

export function selectContainerSummaryCollectionViews(collectionKey: string): ContainerSummaryRecord[] {
  const normalizedCollectionKey = normalizeCollectionKey(collectionKey);
  const order = state.listCollections.get(normalizedCollectionKey) ?? [];
  const metadataById = state.listMetadataByCollection.get(normalizedCollectionKey);
  if (!metadataById) {
    return [];
  }

  return order.reduce<ContainerSummaryRecord[]>((items, containerId) => {
    const metadata = metadataById.get(containerId);
    const next = attachLatestResource(containerId, metadata);
    if (!next) {
      return items;
    }

    items.push(next);
    return items;
  }, []);
}

export function selectContainerDetailView(containerId: string): ContainerDetailRecord | null {
  const metadata = state.detailMetadataById.get(containerId);
  return attachLatestResource(containerId, metadata);
}

export function selectContainerStatsHistory(containerId: string): ContainerStatsSnapshot[] {
  return [...(state.statsById.get(containerId)?.history ?? [])];
}

export function selectContainerStatsChangeState(containerId: string): ContainerStatsChangeState {
  const entry = state.statsById.get(containerId);
  if (!entry) {
    return {
      changedAt: null,
      cpu: 'none',
      memory: 'none',
    };
  }
  if (!isChangeStateFresh(entry.change)) {
    return {
      changedAt: null,
      cpu: 'none',
      memory: 'none',
    };
  }
  return entry.change;
}
