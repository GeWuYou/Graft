import { openRealtimeTopicSocket } from '@/shared/realtime';

import { mapContainerDashboardSummary } from '../api/dashboard-summary';
import type { ContainerDashboardSummary } from '../contract/dashboard-summary';
import type { ContainerDetailRecord, ContainerResourceSummary, ContainerSummaryRecord } from '../types/container';
import {
  buildContainerDashboardSummaryTopic,
  buildContainerListStatsTopic,
  buildContainerStatsTopic,
  parseContainerDashboardSummaryPayload,
  parseContainerListStatsPayload,
  parseContainerStatsPayload,
} from './realtime-stats';
import {
  attachLatestResource,
  buildChangeState,
  createEmptyChangeState,
  hasSameCollectedAt,
  isChangeStateFresh,
  isNewerDashboardSummarySnapshot,
  isNewerStatsSnapshot,
  splitDetailRecord,
  splitSummaryRecord,
} from './stats-manager-snapshot';
import type {
  ContainerMetadataRecord,
  ContainerStatsChangeState,
  ContainerStatsEntry,
  ContainerStatsSnapshot,
  RealtimeSocketState,
  RealtimeSubscriptionEntry,
  StatsSnapshotSource,
} from './stats-manager-state';
import {
  CONTAINER_STATS_CHANGE_HIGHLIGHT_MS,
  CONTAINER_STATS_HISTORY_LIMIT,
  DEFAULT_CONTAINER_LIST_COLLECTION_KEY,
  state,
  SUBSCRIPTION_IDLE_GRACE_MS,
} from './stats-manager-state';

export type {
  ContainerStatsChangeDirection,
  ContainerStatsChangeState,
  ContainerStatsSnapshot,
} from './stats-manager-state';

/**
 * 更新容器的统计快照。
 *
 * @param containerId - 容器标识
 * @param resource - 要写入的统计资源
 * @param source - 快照来源
 * @returns 新写入的快照；如果现有快照更新，则返回当前快照
 */
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
  const nextHistory = hasSameCollectedAt(current, nextSnapshot)
    ? [...(currentEntry?.history ?? []).slice(0, -1), nextSnapshot]
    : [...(currentEntry?.history ?? []), nextSnapshot];
  const nextEntry: ContainerStatsEntry = {
    change: nextChange,
    changeTick: currentEntry?.changeTick ?? 0,
    highlightTimer: currentEntry?.highlightTimer ?? null,
    history: nextHistory.slice(-CONTAINER_STATS_HISTORY_LIMIT),
    previousSnapshot: current,
    snapshot: nextSnapshot,
  };
  clearHighlightTimer(nextEntry);
  if (nextChange.changedAt !== null) {
    nextEntry.highlightTimer = window.setTimeout(() => {
      const latestEntry = state.statsById.get(containerId);
      if (!latestEntry || !latestEntry.change.changedAt || isChangeStateFresh(latestEntry.change)) {
        return;
      }
      latestEntry.change = {
        changedAt: null,
        cpu: 'none',
        memory: 'none',
      };
      latestEntry.changeTick += 1;
      clearHighlightTimer(latestEntry);
    }, CONTAINER_STATS_CHANGE_HIGHLIGHT_MS + 1);
  }
  state.statsById.set(containerId, nextEntry);
  return nextSnapshot;
}

/**
 * 规范化容器集合键。
 *
 * @param collectionKey - 待规范化的集合键
 * @returns 规范化后的集合键；当输入为空或仅包含空白时返回默认键 `container:list`
 */
function normalizeCollectionKey(collectionKey?: string) {
  return collectionKey?.trim() || DEFAULT_CONTAINER_LIST_COLLECTION_KEY;
}

/**
 * 确保列表集合的排序和元数据容器已初始化。
 *
 * @param collectionKey - 列表集合键
 * @returns 包含规范化集合键、顺序数组和按 ID 存储的元数据映射
 */
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

/**
 * 清空指定列表集合的顺序和元数据。
 *
 * @param collectionKey - 列集合键
 */
function clearListMetadata(collectionKey?: string) {
  const targetCollection = ensureListCollection(collectionKey);
  targetCollection.order.length = 0;
  targetCollection.metadataById.clear();
}

/**
 * 获取或创建容器的实时订阅条目。
 *
 * @param containerId - 容器标识
 * @returns 对应的订阅条目
 */
function ensureSubscriptionEntry(containerId: string) {
  const current = state.subscriptionsById.get(containerId);
  if (current) {
    return current;
  }

  const next: RealtimeSubscriptionEntry = {
    controller: null,
    idleTimer: null,
    refCount: 0,
    state: 'idle',
  };
  state.subscriptionsById.set(containerId, next);
  return next;
}

/**
 * 清除订阅条目的空闲定时器。
 */
function clearIdleTimer(entry: RealtimeSubscriptionEntry) {
  if (entry.idleTimer !== null) {
    clearTimeout(entry.idleTimer);
    entry.idleTimer = null;
  }
}

function clearHighlightTimer(entry: ContainerStatsEntry) {
  if (entry.highlightTimer !== null) {
    clearTimeout(entry.highlightTimer);
    entry.highlightTimer = null;
  }
}

/**
 * 关闭单个容器的实时订阅并将其状态重置为空闲。
 */
function closeSubscriptionEntry(entry: RealtimeSubscriptionEntry) {
  clearIdleTimer(entry);
  entry.controller?.close();
  entry.controller = null;
  entry.state = 'idle';
}

/**
 * 关闭共享集合订阅并将其重置为空闲状态。
 */
function closeCollectionSubscriptionEntry(entry: RealtimeSubscriptionEntry) {
  clearIdleTimer(entry);
  entry.controller?.close();
  entry.controller = null;
  entry.state = 'idle';
}

/**
 * 在共享订阅的引用计数降为 0 后，按空闲宽限期延迟关闭该订阅。
 *
 * @param entry - 要更新的共享订阅条目
 */
function releaseSharedSubscription(entry: RealtimeSubscriptionEntry) {
  entry.refCount = Math.max(0, entry.refCount - 1);
  if (entry.refCount > 0) {
    return;
  }

  clearIdleTimer(entry);
  entry.idleTimer = window.setTimeout(() => {
    if (entry.refCount > 0) {
      return;
    }
    closeCollectionSubscriptionEntry(entry);
  }, SUBSCRIPTION_IDLE_GRACE_MS);
}

/**
 * 建立指定容器的实时统计订阅连接。
 *
 * @param containerId - 容器 ID
 * @param entry - 该容器的订阅状态记录
 */
function connectSubscription(containerId: string, entry: RealtimeSubscriptionEntry) {
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

/**
 * 连接容器列表汇总的实时订阅。
 *
 * @param entry - 容器列表共享订阅条目
 */
function connectCollectionSubscription(entry: RealtimeSubscriptionEntry) {
  if (entry.controller) {
    return;
  }

  entry.state = 'connecting';
  entry.controller = openRealtimeTopicSocket({
    topic: buildContainerListStatsTopic(),
    parseMessage: parseContainerListStatsPayload,
    onStateChange: (nextState) => {
      state.listTopicSubscription.state = nextState;
      if (nextState === 'idle' && state.listTopicSubscription.refCount > 0 && !state.listTopicSubscription.controller) {
        connectCollectionSubscription(state.listTopicSubscription);
      }
    },
    onMessage: (payload) => {
      payload.items.forEach((item) => {
        applyContainerRealtimeStats(item.id, item.resource);
      });
    },
  });
}

/**
 * 连接容器仪表盘汇总的实时主题订阅。
 *
 * @param entry - 仪表盘汇总共享订阅条目
 */
function connectDashboardSummarySubscription(entry: RealtimeSubscriptionEntry) {
  if (entry.controller) {
    return;
  }

  entry.state = 'connecting';
  entry.controller = openRealtimeTopicSocket({
    topic: buildContainerDashboardSummaryTopic(),
    parseMessage: parseContainerDashboardSummaryPayload,
    onStateChange: (nextState) => {
      state.dashboardSummarySubscription.state = nextState;
      if (
        nextState === 'idle' &&
        state.dashboardSummarySubscription.refCount > 0 &&
        !state.dashboardSummarySubscription.controller
      ) {
        connectDashboardSummarySubscription(state.dashboardSummarySubscription);
      }
    },
    onMessage: (payload) => {
      seedContainerDashboardSummary(mapContainerDashboardSummary(payload), 'realtime');
    },
  });
}

/**
 * 更新容器仪表盘汇总快照。
 *
 * @param summary - 仪表盘汇总数据
 * @param source - 快照来源
 * @returns 当前生效的仪表盘汇总；若输入未更新，则返回现有汇总
 */
function upsertDashboardSummarySnapshot(summary: ContainerDashboardSummary, source: StatsSnapshotSource) {
  const current = state.dashboardSummary;
  if (!isNewerDashboardSummarySnapshot(current, summary, source)) {
    return current?.summary ?? null;
  }

  state.dashboardSummary = {
    source,
    summary,
  };
  return summary;
}

/**
 * 重置容器统计管理器的全部运行状态。
 *
 * 会关闭共享订阅和按容器订阅，清除统计高亮定时器，并清空仪表盘汇总、列表、详情和统计缓存。
 */
export function resetContainerStatsManager() {
  closeCollectionSubscriptionEntry(state.dashboardSummarySubscription);
  closeCollectionSubscriptionEntry(state.listTopicSubscription);
  state.subscriptionsById.forEach((entry) => {
    closeSubscriptionEntry(entry);
  });
  state.statsById.forEach((entry) => {
    clearHighlightTimer(entry);
  });
  state.listCollections.clear();
  state.listMetadataByCollection.clear();
  state.dashboardSummary = null;
  state.detailMetadataById.clear();
  state.statsById.clear();
  state.subscriptionsById.clear();
  state.dashboardSummarySubscription.refCount = 0;
  state.dashboardSummarySubscription.idleTimer = null;
  state.dashboardSummarySubscription.state = 'idle';
  state.listTopicSubscription.refCount = 0;
  state.listTopicSubscription.idleTimer = null;
  state.listTopicSubscription.state = 'idle';
}

/**
 * 预置容器仪表盘汇总快照。
 *
 * @param summary - 要写入的容器仪表盘汇总
 * @param source - 快照来源
 */
export function seedContainerDashboardSummary(
  summary: ContainerDashboardSummary,
  source: StatsSnapshotSource = 'http-seed',
) {
  upsertDashboardSummarySnapshot(summary, source);
}

/**
 * 清除容器仪表盘汇总快照。
 */
export function clearContainerDashboardSummary() {
  state.dashboardSummary = null;
}

/**
 * 预置容器列表并写入对应统计快照。
 *
 * @param items - 要写入的容器列表项
 * @param collectionKey - 列表集合键
 */
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

/**
 * 预置容器详情及其最新统计快照。
 *
 * @param detail - 容器详情记录
 */
export function seedContainerDetail(detail: ContainerDetailRecord) {
  const { metadata, resource } = splitDetailRecord(detail);
  state.detailMetadataById.set(detail.id, metadata);
  if (resource) {
    upsertStatsSnapshot(detail.id, resource, 'http-seed');
  }
}

/**
 * 清除容器详情元数据。
 *
 * @param containerId - 容器 ID；不传则清除全部详情元数据
 */
export function clearContainerDetail(containerId?: string) {
  if (!containerId) {
    state.detailMetadataById.clear();
    return;
  }
  state.detailMetadataById.delete(containerId);
}

/**
 * 清空默认容器列表集合的元数据和顺序。
 */
export function clearContainerListMetadata() {
  clearListMetadata();
}

/**
 * 清除指定容器汇总集合的元数据和顺序。
 *
 * @param collectionKey - 集合键
 */
export function clearContainerSummaryCollection(collectionKey: string) {
  clearListMetadata(collectionKey);
}

/**
 * 应用容器的实时统计资源更新。
 *
 * @param containerId - 容器 ID
 * @param resource - 实时统计资源
 * @returns 最新的统计快照；如果传入数据未更新现有快照，则返回现有快照
 */
export function applyContainerRealtimeStats(containerId: string, resource: ContainerResourceSummary) {
  return upsertStatsSnapshot(containerId, resource, 'realtime');
}

/**
 * 获取容器统计实时订阅。
 *
 * @param containerId - 容器 ID
 */
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

/**
 * 获取容器汇总集合的实时订阅。
 */
export function acquireContainerSummaryCollectionSubscription() {
  const entry = state.listTopicSubscription;
  clearIdleTimer(entry);
  entry.refCount += 1;
  if (!entry.controller) {
    connectCollectionSubscription(entry);
  }
}

/**
 * 释放容器统计实时订阅的一个引用，并在空闲宽限期后关闭连接。
 *
 * @param containerId - 容器 ID
 */
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

/**
 * 释放容器汇总集合的实时订阅引用。
 *
 * @remarks
 * 当引用计数降为 0 时，会在宽限期后关闭共享列表主题订阅。
 */
export function releaseContainerSummaryCollectionSubscription() {
  releaseSharedSubscription(state.listTopicSubscription);
}

/**
 * 获取容器仪表盘汇总的实时订阅。
 */
export function acquireContainerDashboardSummarySubscription() {
  const entry = state.dashboardSummarySubscription;
  clearIdleTimer(entry);
  entry.refCount += 1;
  if (!entry.controller) {
    connectDashboardSummarySubscription(entry);
  }
}

/**
 * 释放容器仪表盘汇总实时订阅引用。
 */
export function releaseContainerDashboardSummarySubscription() {
  releaseSharedSubscription(state.dashboardSummarySubscription);
}

/**
 * 获取容器统计实时订阅的连接状态。
 *
 * @param containerId - 容器 ID
 * @returns 该容器订阅的实时连接状态；若不存在订阅则返回 `idle`
 */
export function selectContainerStatsRealtimeState(containerId: string): RealtimeSocketState {
  return state.subscriptionsById.get(containerId)?.state ?? 'idle';
}

/**
 * 获取容器仪表盘汇总的视图数据。
 *
 * @returns 当前仪表盘汇总；未写入时返回 `null`
 */
export function selectContainerDashboardSummaryView(): ContainerDashboardSummary | null {
  return state.dashboardSummary?.summary ?? null;
}

/**
 * 获取容器仪表盘汇总实时订阅的连接状态。
 *
 * @returns 容器仪表盘汇总主题的实时连接状态
 */
export function selectContainerDashboardRealtimeState(): RealtimeSocketState {
  return state.dashboardSummarySubscription.state;
}

/**
 * 获取容器在默认列表集合中的摘要视图。
 *
 * @param containerId - 容器 ID
 * @returns 包含最新统计资源的容器摘要记录；如果未找到对应元数据则返回 `null`
 */
function selectContainerSummaryView(containerId: string): ContainerSummaryRecord | null {
  const metadata = state.listMetadataByCollection.get(DEFAULT_CONTAINER_LIST_COLLECTION_KEY)?.get(containerId);
  return attachLatestResource(state.statsById.get(containerId)?.snapshot?.resource, metadata);
}

/**
 * 获取默认容器列表集合的视图数据。
 *
 * @returns 按集合顺序返回的容器摘要记录数组，缺少元数据的项会被跳过。
 */
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

/**
 * 获取指定集合的容器列表视图。
 *
 * @param collectionKey - 集合键
 * @returns 该集合中按顺序排列的容器摘要记录数组
 */
export function selectContainerSummaryCollectionViews(collectionKey: string): ContainerSummaryRecord[] {
  const normalizedCollectionKey = normalizeCollectionKey(collectionKey);
  const order = state.listCollections.get(normalizedCollectionKey) ?? [];
  const metadataById = state.listMetadataByCollection.get(normalizedCollectionKey);
  if (!metadataById) {
    return [];
  }

  return order.reduce<ContainerSummaryRecord[]>((items, containerId) => {
    const metadata = metadataById.get(containerId);
    const next = attachLatestResource(state.statsById.get(containerId)?.snapshot?.resource, metadata);
    if (!next) {
      return items;
    }

    items.push(next);
    return items;
  }, []);
}

/**
 * 获取容器详情视图。
 *
 * @param containerId - 容器 ID
 * @returns 包含最新统计资源的容器详情记录；如果没有详情元数据则返回 `null`
 */
export function selectContainerDetailView(containerId: string): ContainerDetailRecord | null {
  const metadata = state.detailMetadataById.get(containerId);
  return attachLatestResource(state.statsById.get(containerId)?.snapshot?.resource, metadata);
}

/**
 * 获取容器统计快照历史。
 *
 * @param containerId - 容器 ID
 * @returns 该容器已保存的统计快照历史数组
 */
export function selectContainerStatsHistory(containerId: string): ContainerStatsSnapshot[] {
  return [...(state.statsById.get(containerId)?.history ?? [])];
}

/**
 * 获取容器统计的变化状态。
 *
 * @param containerId - 容器 ID
 * @returns 容器的变化状态；在没有统计记录或变化高亮已过期时，返回 `changedAt: null` 且 CPU、内存方向均为 `none` 的状态
 */
export function selectContainerStatsChangeState(containerId: string): ContainerStatsChangeState {
  const entry = state.statsById.get(containerId);
  if (!entry) {
    return createEmptyChangeState();
  }
  void entry.changeTick;
  if (!isChangeStateFresh(entry.change)) {
    return createEmptyChangeState();
  }
  return entry.change;
}
