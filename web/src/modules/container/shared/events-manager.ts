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

function closeRealtime(containerId: string) {
  const entry = state.subscriptionsById.get(containerId);
  if (!entry) {
    return;
  }
  entry.controller?.close();
  entry.controller = null;
  entry.socketState = 'idle';
}

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

export function clearContainerEvents(containerId: string) {
  const normalizedContainerId = containerId.trim();
  closeRealtime(normalizedContainerId);
  state.subscriptionsById.delete(normalizedContainerId);
}

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
