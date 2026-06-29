import { reactive } from 'vue';

import type { RealtimeTopicSocketController } from '@/shared/realtime';

import type { ContainerDashboardSummary } from '../contract/dashboard-summary';
import type { ContainerDetailRecord, ContainerResourceSummary, ContainerSummaryRecord } from '../types/container';

export type ContainerMetadataRecord = Omit<ContainerSummaryRecord, 'resource'>;
export type ContainerDetailMetadataRecord = Omit<ContainerDetailRecord, 'resource'>;
export type ContainerSummaryCollectionKey = string;

export type StatsSnapshotSource = 'http-seed' | 'realtime';
export type RealtimeSocketState = 'idle' | 'connecting' | 'open' | 'closed' | 'error';

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

export type ContainerStatsEntry = {
  change: ContainerStatsChangeState;
  changeTick: number;
  highlightTimer: number | null;
  history: ContainerStatsSnapshot[];
  previousSnapshot: ContainerStatsSnapshot | null;
  snapshot: ContainerStatsSnapshot | null;
};

export type RealtimeSubscriptionEntry = {
  controller: RealtimeTopicSocketController | null;
  idleTimer: number | null;
  refCount: number;
  state: RealtimeSocketState;
};

export type ContainerDashboardSummarySnapshot = {
  source: StatsSnapshotSource;
  summary: ContainerDashboardSummary;
};

export type ContainerStatsManagerState = {
  dashboardSummary: ContainerDashboardSummarySnapshot | null;
  dashboardSummarySubscription: RealtimeSubscriptionEntry;
  detailMetadataById: Map<string, ContainerDetailMetadataRecord>;
  listCollections: Map<ContainerSummaryCollectionKey, string[]>;
  listTopicSubscription: RealtimeSubscriptionEntry;
  listMetadataByCollection: Map<ContainerSummaryCollectionKey, Map<string, ContainerMetadataRecord>>;
  statsById: Map<string, ContainerStatsEntry>;
  subscriptionsById: Map<string, RealtimeSubscriptionEntry>;
};

export type SnapshotWithSource<TSnapshot> = {
  source: StatsSnapshotSource;
  summary: TSnapshot;
};

export const SUBSCRIPTION_IDLE_GRACE_MS = 10_000;
export const DEFAULT_CONTAINER_LIST_COLLECTION_KEY = 'container:list';
export const CONTAINER_STATS_HISTORY_LIMIT = 12;
export const CONTAINER_STATS_CHANGE_HIGHLIGHT_MS = 800;

export const EMPTY_CONTAINER_STATS_CHANGE_STATE: ContainerStatsChangeState = {
  changedAt: null,
  cpu: 'none',
  memory: 'none',
};

export const state = reactive<ContainerStatsManagerState>({
  dashboardSummary: null,
  dashboardSummarySubscription: {
    controller: null,
    idleTimer: null,
    refCount: 0,
    state: 'idle',
  },
  detailMetadataById: new Map<string, ContainerDetailMetadataRecord>(),
  listCollections: new Map<ContainerSummaryCollectionKey, string[]>(),
  listTopicSubscription: {
    controller: null,
    idleTimer: null,
    refCount: 0,
    state: 'idle',
  },
  listMetadataByCollection: new Map<ContainerSummaryCollectionKey, Map<string, ContainerMetadataRecord>>(),
  statsById: new Map<string, ContainerStatsEntry>(),
  subscriptionsById: new Map<string, RealtimeSubscriptionEntry>(),
});
