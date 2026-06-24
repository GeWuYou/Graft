import { reactive } from 'vue';

import { openRealtimeTopicSocket, type RealtimeTopicSocketController } from '@/shared/realtime';

import type { ContainerDetailRecord, ContainerResourceSummary, ContainerSummaryRecord } from '../types/container';
import {
  buildContainerListStatsTopic,
  buildContainerStatsTopic,
  parseContainerListStatsPayload,
  parseContainerStatsPayload,
} from './realtime-stats';

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
  changeTick: number;
  highlightTimer: number | null;
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

type ContainerStatsCollectionSubscriptionEntry = {
  controller: RealtimeTopicSocketController | null;
  idleTimer: number | null;
  refCount: number;
  state: RealtimeSocketState;
};

type ContainerStatsManagerState = {
  detailMetadataById: Map<string, ContainerDetailMetadataRecord>;
  listCollections: Map<ContainerSummaryCollectionKey, string[]>;
  listTopicSubscription: ContainerStatsCollectionSubscriptionEntry;
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
  listTopicSubscription: {
    controller: null,
    idleTimer: null,
    refCount: 0,
    state: 'idle',
  },
  listMetadataByCollection: new Map<ContainerSummaryCollectionKey, Map<string, ContainerMetadataRecord>>(),
  statsById: new Map<string, ContainerStatsEntry>(),
  subscriptionsById: new Map<string, ContainerStatsSubscriptionEntry>(),
});

/**
 * 规范化采集时间字符串。
 *
 * @param value - 待处理的采集时间
 * @returns 去除首尾空白后的字符串；当值缺失或为空时返回 `null`
 */
function normalizeCollectedAt(value?: string | null) {
  return value?.trim() || null;
}

/**
 * 获取资源的采集时间。
 *
 * @param resource - 容器资源摘要
 * @returns 规范化后的采集时间，缺失或为空时为 `null`
 */
function getCollectedAtValue(resource?: ContainerResourceSummary | null) {
  return normalizeCollectedAt(resource?.collected_at);
}

/**
 * 比较两个指标值的变化方向。
 *
 * @param previous - 之前的指标值
 * @param next - 之后的指标值
 * @returns `up` 表示升高，`down` 表示降低，`none` 表示保持不变或值无效
 */
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

/**
 * 生成容器统计变化状态。
 *
 * @param current - 当前快照
 * @param nextSnapshot - 新快照
 * @param source - 快照来源
 * @returns 包含 CPU、内存变化方向及高亮时间戳的状态
 */
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

/**
 * 判断两个统计快照是否指向同一采集时刻。
 *
 * @param current - 当前快照
 * @param nextSnapshot - 待写入的快照
 * @returns `true` 表示两个快照拥有相同的采集时间且该时间有效
 */
function hasSameCollectedAt(current: ContainerStatsSnapshot | null, nextSnapshot: ContainerStatsSnapshot) {
  const currentCollectedAt = getCollectedAtValue(current?.resource);
  const nextCollectedAt = getCollectedAtValue(nextSnapshot.resource);
  return Boolean(currentCollectedAt && nextCollectedAt && currentCollectedAt === nextCollectedAt);
}

/**
 * 判断变化状态是否仍处于高亮窗口内。
 *
 * @param change - 变化状态
 * @returns `true` if `changedAt` 存在且未超过高亮时长，`false` otherwise.
 */
function isChangeStateFresh(change: ContainerStatsChangeState) {
  return typeof change.changedAt === 'number' && Date.now() - change.changedAt <= CONTAINER_STATS_CHANGE_HIGHLIGHT_MS;
}

/**
 * 判断候选统计快照是否比当前快照更新。
 *
 * @param current - 当前已保存的统计快照
 * @param candidate - 待比较的统计资源
 * @param source - 候选快照来源
 * @returns `true` 如果候选快照应覆盖当前快照，`false` 其他情况
 */
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
 * 确保容器统计订阅条目存在。
 *
 * @param containerId - 容器标识
 * @returns 对应的订阅条目
 */
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

/**
 * 清除订阅条目的空闲定时器。
 */
function clearIdleTimer(entry: ContainerStatsSubscriptionEntry) {
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
 * 关闭单个容器统计订阅并重置其状态。
 */
function closeSubscriptionEntry(entry: ContainerStatsSubscriptionEntry) {
  clearIdleTimer(entry);
  entry.controller?.close();
  entry.controller = null;
  entry.state = 'idle';
}

/**
 * 关闭共享集合订阅并将其重置为空闲状态。
 */
function closeCollectionSubscriptionEntry(entry: ContainerStatsCollectionSubscriptionEntry) {
  clearIdleTimer(entry);
  entry.controller?.close();
  entry.controller = null;
  entry.state = 'idle';
}

/**
 * 为指定容器建立实时统计订阅。
 *
 * @param containerId - 容器 ID
 * @param entry - 该容器对应的订阅状态
 */
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

/**
 * 连接容器列表汇总的实时订阅。
 *
 * 当订阅控制器已存在时直接返回；否则打开共享的列表主题连接，并在收到列表项更新时同步应用到对应容器的实时统计。
 */
function connectCollectionSubscription(entry: ContainerStatsCollectionSubscriptionEntry) {
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
 * 拆分容器摘要记录中的资源与元数据。
 *
 * @param record - 容器摘要记录
 * @returns 包含 `metadata` 和 `resource` 的对象
 */
function splitSummaryRecord(record: ContainerSummaryRecord) {
  const { resource, ...metadata } = record;
  return {
    metadata: metadata as ContainerMetadataRecord,
    resource,
  };
}

/**
 * 将容器详情记录拆分为元数据和资源部分。
 *
 * @param record - 容器详情记录
 * @returns 拆分后的元数据与资源对象
 */
function splitDetailRecord(record: ContainerDetailRecord) {
  const { resource, ...metadata } = record;
  return {
    metadata: metadata as ContainerDetailMetadataRecord,
    resource,
  };
}

/**
 * 将容器元数据附加为带最新资源信息的视图对象。
 *
 * @param containerId - 容器 ID
 * @param metadata - 要附加资源信息的元数据
 * @returns 合并了当前统计快照中资源信息的对象；当元数据缺失时返回 `null`
 */
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

/**
 * 重置容器统计管理器的全部状态。
 */
export function resetContainerStatsManager() {
  closeCollectionSubscriptionEntry(state.listTopicSubscription);
  state.subscriptionsById.forEach((entry) => {
    closeSubscriptionEntry(entry);
  });
  state.statsById.forEach((entry) => {
    clearHighlightTimer(entry);
  });
  state.listCollections.clear();
  state.listMetadataByCollection.clear();
  state.detailMetadataById.clear();
  state.statsById.clear();
  state.subscriptionsById.clear();
  state.listTopicSubscription.refCount = 0;
  state.listTopicSubscription.idleTimer = null;
  state.listTopicSubscription.state = 'idle';
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
 * 将共享列表主题的引用计数减一；当引用计数降为 0 时，立即关闭该订阅。
 */
export function releaseContainerSummaryCollectionSubscription() {
  const entry = state.listTopicSubscription;
  entry.refCount = Math.max(0, entry.refCount - 1);
  if (entry.refCount > 0) {
    return;
  }

  clearIdleTimer(entry);
  entry.idleTimer = window.setTimeout(() => {
    if (state.listTopicSubscription.refCount > 0) {
      return;
    }
    closeCollectionSubscriptionEntry(state.listTopicSubscription);
  }, SUBSCRIPTION_IDLE_GRACE_MS);
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
 * 获取容器在默认列表集合中的摘要视图。
 *
 * @param containerId - 容器 ID
 * @returns 包含最新统计资源的容器摘要记录；如果未找到对应元数据则返回 `null`
 */
function selectContainerSummaryView(containerId: string): ContainerSummaryRecord | null {
  const metadata = state.listMetadataByCollection.get(DEFAULT_CONTAINER_LIST_COLLECTION_KEY)?.get(containerId);
  return attachLatestResource(containerId, metadata);
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
    const next = attachLatestResource(containerId, metadata);
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
  return attachLatestResource(containerId, metadata);
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
    return {
      changedAt: null,
      cpu: 'none',
      memory: 'none',
    };
  }
  void entry.changeTick;
  if (!isChangeStateFresh(entry.change)) {
    return {
      changedAt: null,
      cpu: 'none',
      memory: 'none',
    };
  }
  return entry.change;
}
