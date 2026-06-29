import type { ContainerDashboardSummary } from '../contract/dashboard-summary';
import type { ContainerDetailRecord, ContainerResourceSummary, ContainerSummaryRecord } from '../types/container';
import type {
  ContainerDashboardSummarySnapshot,
  ContainerDetailMetadataRecord,
  ContainerMetadataRecord,
  ContainerStatsChangeDirection,
  ContainerStatsChangeState,
  ContainerStatsSnapshot,
  SnapshotWithSource,
  StatsSnapshotSource,
} from './stats-manager-state';
import { CONTAINER_STATS_CHANGE_HIGHLIGHT_MS, EMPTY_CONTAINER_STATS_CHANGE_STATE } from './stats-manager-state';

function normalizeCollectedAt(value?: string | null) {
  return value?.trim() || null;
}

function getCollectedAtValue(resource?: ContainerResourceSummary | null) {
  return normalizeCollectedAt(resource?.collected_at);
}

function getDashboardSummaryCollectedAt(summary?: ContainerDashboardSummary | null) {
  return normalizeCollectedAt(summary?.overview.collectedAt);
}

function isNewerSnapshot<TSnapshot>(
  current: SnapshotWithSource<TSnapshot> | null,
  candidate: TSnapshot,
  source: StatsSnapshotSource,
  readCollectedAt: (snapshot: TSnapshot) => string | null,
) {
  if (!current) {
    return true;
  }

  const currentCollectedAt = readCollectedAt(current.summary);
  const candidateCollectedAt = readCollectedAt(candidate);

  if (candidateCollectedAt && currentCollectedAt) {
    return candidateCollectedAt >= currentCollectedAt;
  }
  if (candidateCollectedAt && !currentCollectedAt) {
    return true;
  }
  if (!candidateCollectedAt && currentCollectedAt) {
    return false;
  }

  return !(current.source === 'realtime' && source === 'http-seed');
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

export function buildChangeState(
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

export function hasSameCollectedAt(current: ContainerStatsSnapshot | null, nextSnapshot: ContainerStatsSnapshot) {
  const currentCollectedAt = getCollectedAtValue(current?.resource);
  const nextCollectedAt = getCollectedAtValue(nextSnapshot.resource);
  return Boolean(currentCollectedAt && nextCollectedAt && currentCollectedAt === nextCollectedAt);
}

export function isChangeStateFresh(change: ContainerStatsChangeState) {
  return typeof change.changedAt === 'number' && Date.now() - change.changedAt <= CONTAINER_STATS_CHANGE_HIGHLIGHT_MS;
}

export function isNewerStatsSnapshot(
  current: ContainerStatsSnapshot | null,
  candidate: ContainerResourceSummary,
  source: StatsSnapshotSource,
) {
  return isNewerSnapshot(
    current ? { source: current.source, summary: current.resource } : null,
    candidate,
    source,
    getCollectedAtValue,
  );
}

export function isNewerDashboardSummarySnapshot(
  current: ContainerDashboardSummarySnapshot | null,
  candidate: ContainerDashboardSummary,
  source: StatsSnapshotSource,
) {
  return isNewerSnapshot(current, candidate, source, getDashboardSummaryCollectedAt);
}

export function splitSummaryRecord(record: ContainerSummaryRecord) {
  const { resource, ...metadata } = record;
  return {
    metadata: metadata as ContainerMetadataRecord,
    resource,
  };
}

export function splitDetailRecord(record: ContainerDetailRecord) {
  const { resource, ...metadata } = record;
  return {
    metadata: metadata as ContainerDetailMetadataRecord,
    resource,
  };
}

export function attachLatestResource<TMetadata extends ContainerMetadataRecord | ContainerDetailMetadataRecord>(
  resource: ContainerResourceSummary | undefined | null,
  metadata: TMetadata | undefined,
) {
  if (!metadata) {
    return null;
  }

  return {
    ...metadata,
    resource: resource ?? undefined,
  };
}

export function createEmptyChangeState() {
  return { ...EMPTY_CONTAINER_STATS_CHANGE_STATE };
}
