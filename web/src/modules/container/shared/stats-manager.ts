import { reactive } from 'vue';

import type { ContainerDetailRecord, ContainerResourceSummary, ContainerSummaryRecord } from '../types/container';

type ContainerMetadataRecord = Omit<ContainerSummaryRecord, 'resource'>;
type ContainerDetailMetadataRecord = Omit<ContainerDetailRecord, 'resource'>;

type StatsSnapshotSource = 'http-seed' | 'realtime';

export type ContainerStatsSnapshot = {
  resource: ContainerResourceSummary;
  source: StatsSnapshotSource;
};

type ContainerStatsEntry = {
  snapshot: ContainerStatsSnapshot | null;
};

type ContainerStatsManagerState = {
  detailMetadataById: Map<string, ContainerDetailMetadataRecord>;
  listOrder: string[];
  listMetadataById: Map<string, ContainerMetadataRecord>;
  statsById: Map<string, ContainerStatsEntry>;
};

const state = reactive<ContainerStatsManagerState>({
  detailMetadataById: new Map<string, ContainerDetailMetadataRecord>(),
  listOrder: [],
  listMetadataById: new Map<string, ContainerMetadataRecord>(),
  statsById: new Map<string, ContainerStatsEntry>(),
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
  state.listOrder = [];
  state.listMetadataById.clear();
  state.detailMetadataById.clear();
  state.statsById.clear();
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

export function applyContainerRealtimeStats(containerId: string, resource: ContainerResourceSummary) {
  return upsertStatsSnapshot(containerId, resource, 'realtime');
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
