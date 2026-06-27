import { reactive } from 'vue';

import { resolveStreamViewportState, type StreamViewportState } from '@/shared/observability';
import { openRealtimeTopicSocket, type RealtimeTopicSocketController } from '@/shared/realtime';

import { getContainerEvents } from '../api/container';
import {
  buildContainerEventsTopicName,
  isContainerEventsTopicForContainer,
  parseContainerEventsPayload,
} from '../contract/realtime';
import type { ContainerRuntimeEventRecord, ContainerRuntimeEventsResponse } from '../types/container';

type RealtimeSocketState = 'idle' | 'connecting' | 'open' | 'closed' | 'error';
type EventSnapshotSource = 'http-seed' | 'realtime';

type ContainerEventsEntry = {
  source: EventSnapshotSource;
  record: ContainerRuntimeEventRecord;
};

type ContainerEventsView = {
  items: ContainerRuntimeEventRecord[];
  resourceId: string;
  runtime: string;
};

type RealtimeSubscriptionEntry = {
  controller: RealtimeTopicSocketController | null;
  refCount: number;
  socketState: RealtimeSocketState;
  error: string;
  started: boolean;
  loading: boolean;
  view: ContainerEventsView | null;
};

const state = reactive<{
  subscriptionsById: Map<string, RealtimeSubscriptionEntry>;
}>({
  subscriptionsById: new Map<string, RealtimeSubscriptionEntry>(),
});

/**
 * 获取或创建指定容器的事件订阅条目。
 *
 * @param containerId - 容器 ID
 * @returns 容器对应的订阅条目
 */
function ensureEntry(containerId: string) {
  let entry = state.subscriptionsById.get(containerId);
  if (!entry) {
    entry = {
      controller: null,
      refCount: 0,
      socketState: 'idle',
      error: '',
      started: false,
      loading: false,
      view: null,
    };
    state.subscriptionsById.set(containerId, entry);
  }
  return entry;
}

/**
 * 按 `seq` 合并两组运行时事件记录。
 *
 * @param current - 现有事件记录
 * @param incoming - 新增事件记录
 * @param source - 新增记录的来源标记
 * @returns 合并后按 `seq` 降序排列的事件记录列表
 */
function mergeRecords(
  current: ContainerRuntimeEventRecord[],
  incoming: ContainerRuntimeEventRecord[],
  source: EventSnapshotSource,
) {
  const bySeq = new Map<number, ContainerEventsEntry>();
  for (const item of current) {
    bySeq.set(item.seq, { source: 'http-seed', record: item });
  }
  for (const item of incoming) {
    const existing = bySeq.get(item.seq);
    if (!existing || existing.source === 'http-seed' || source === 'realtime') {
      bySeq.set(item.seq, { source, record: item });
    }
  }
  return [...bySeq.values()].map((item) => item.record).sort((left, right) => right.seq - left.seq);
}

/**
 * 拉取并初始化容器运行时事件历史。
 *
 * @param containerId - 容器 ID
 */
async function seedHistory(containerId: string) {
  const entry = ensureEntry(containerId);
  entry.loading = true;
  entry.started = true;
  entry.error = '';
  try {
    const next = await getContainerEvents(containerId);
    const currentItems = entry.view?.items ?? [];
    entry.view = {
      items: mergeRecords(currentItems, next.items ?? [], 'http-seed'),
      resourceId: next.resource_id ?? containerId,
      runtime: next.context?.runtime ?? '',
    };
  } catch (error) {
    entry.error = error instanceof Error ? error.message : 'Failed to load container runtime events';
  } finally {
    entry.loading = false;
  }
}

/**
 * 建立容器运行时事件的实时订阅连接。
 *
 * @param containerId - 容器 ID
 */
function connectRealtime(containerId: string) {
  const entry = ensureEntry(containerId);
  const topic = buildContainerEventsTopicName(containerId);
  if (entry.controller) {
    return;
  }
  entry.controller = openRealtimeTopicSocket({
    topic,
    parseMessage: parseContainerEventsPayload,
    onMessage: (payload) => {
      if (!payload || !isContainerEventsTopicForContainer(payload.topic, containerId)) {
        return;
      }
      const currentItems = entry.view?.items ?? [];
      entry.view = {
        items: mergeRecords(currentItems, [payload.record], 'realtime'),
        resourceId: payload.resource_id,
        runtime: payload.context.runtime,
      };
    },
    onStateChange: (nextState) => {
      const previousState = entry.socketState;
      entry.socketState = nextState;
      if (previousState === 'open' && (nextState === 'connecting' || nextState === 'open') && entry.started) {
        void seedHistory(containerId);
      }
    },
    onError: (message) => {
      entry.error = message;
    },
  });
}

/**
 * 关闭指定容器的实时事件连接。
 *
 * @param containerId - 容器 ID
 */
function closeRealtime(containerId: string) {
  const entry = state.subscriptionsById.get(containerId);
  if (!entry) {
    return;
  }
  entry.controller?.close();
  entry.controller = null;
  entry.socketState = 'idle';
}

/**
 * 为指定容器获取运行时事件订阅。
 *
 * 会先去除 `containerId` 两端空白；当结果为空时不执行任何操作。首次获取时会增加引用计数，并在需要时拉取历史事件并建立实时连接。
 *
 * @param containerId - 容器 ID
 */
export function acquireContainerEventsSubscription(containerId: string) {
  const normalizedContainerId = containerId.trim();
  if (!normalizedContainerId) {
    return;
  }
  const entry = ensureEntry(normalizedContainerId);
  entry.refCount += 1;
  if (!entry.started && !entry.loading) {
    void seedHistory(normalizedContainerId);
  }
  connectRealtime(normalizedContainerId);
}

/**
 * 释放指定容器运行时事件订阅的引用。
 *
 * 当引用计数降为 0 时，关闭对应的实时连接。
 *
 * @param containerId - 容器 ID
 */
export function releaseContainerEventsSubscription(containerId: string) {
  const normalizedContainerId = containerId.trim();
  const entry = state.subscriptionsById.get(normalizedContainerId);
  if (!entry) {
    return;
  }
  entry.refCount = Math.max(entry.refCount - 1, 0);
  if (entry.refCount === 0) {
    closeRealtime(normalizedContainerId);
  }
}

/**
 * 清除指定容器的事件订阅和缓存状态。
 *
 * @param containerId - 容器 ID
 */
export function clearContainerEvents(containerId: string) {
  const normalizedContainerId = containerId.trim();
  closeRealtime(normalizedContainerId);
  state.subscriptionsById.delete(normalizedContainerId);
}

/**
 * 获取容器当前合并后的运行时事件视图。
 *
 * @param containerId - 容器 ID
 * @returns 当前事件视图；如果尚未加载则返回 `null`
 */
export function selectContainerEventsView(containerId: string): ContainerRuntimeEventsResponse | null {
  const entry = state.subscriptionsById.get(containerId.trim());
  if (!entry?.view) {
    return null;
  }
  return {
    context: {
      runtime: entry.view.runtime,
    },
    items: entry.view.items,
    resource_id: entry.view.resourceId,
  };
}

/**
 * 获取容器运行时事件流的视口状态。
 *
 * @param containerId - 容器 ID
 * @returns 根据当前加载、连接、重连、断开和错误状态计算得到的视口状态
 */
export function selectContainerEventsViewportState(containerId: string): StreamViewportState {
  const entry = state.subscriptionsById.get(containerId.trim());
  return resolveStreamViewportState({
    hasContent: Boolean(entry?.view?.items?.length),
    hasStarted: Boolean(entry?.started),
    isConnecting: entry?.loading || entry?.socketState === 'connecting',
    isStreaming: entry?.socketState === 'open',
    isReconnecting: entry?.started && entry?.socketState === 'connecting' && Boolean(entry?.view?.items?.length),
    isDisconnected: entry?.started && (entry?.socketState === 'closed' || entry?.socketState === 'error'),
    error: entry?.error,
  });
}
