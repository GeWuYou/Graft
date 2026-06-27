import { computed, shallowRef, triggerRef } from 'vue';

import type { StructuredLogEntry } from '@/shared/observability';

import type { ContainerLogResponse } from '../../types/container';
import type { ContainerLogRealtimeBatcherSnapshot } from './log-realtime-batcher';

type LogViewStoreState = Readonly<{
  snapshot: ContainerLogRealtimeBatcherSnapshot | null;
  error: string;
  loading: boolean;
  paused: boolean;
}>;

/**
 * 将实时日志快照转换为日志响应对象。
 *
 * @param snapshot - 实时日志快照
 * @returns 转换后的日志响应对象；当 `snapshot` 为空时返回 `null`
 */
function buildLogResponse(snapshot: ContainerLogRealtimeBatcherSnapshot | null): ContainerLogResponse | null {
  if (!snapshot) {
    return null;
  }

  return Object.freeze({
    id: snapshot.id,
    entries: snapshot.entryView.toArray().map((entry) => ({
      line: entry.line,
      occurred_at: entry.occurredAt,
      stream: entry.stream,
    })),
    runtime: snapshot.runtime,
    stderr: snapshot.stderr,
    stdout: snapshot.stdout,
    tail: snapshot.tail,
    timestamps: snapshot.timestamps,
    truncated: snapshot.truncated,
  });
}

/**
 * 创建容器详情日志视图的状态管理 store。
 *
 * @returns 包含日志快照、派生视图状态以及加载、错误、暂停、恢复和重置方法的 store
 */
export function createContainerDetailLogViewStore() {
  const state = shallowRef<LogViewStoreState>({
    snapshot: null,
    error: '',
    loading: false,
    paused: false,
  });
  let pendingSnapshot: ContainerLogRealtimeBatcherSnapshot | null = null;

  const version = computed(() => state.value.snapshot?.version ?? 0);
  const hasSnapshot = computed(() => state.value.snapshot !== null);
  const logs = computed(() => {
    void version.value;
    return buildLogResponse(state.value.snapshot);
  });
  const entries = computed(() => {
    void version.value;
    return (state.value.snapshot?.entryView.toArray() ?? []) as readonly StructuredLogEntry[];
  });
  const truncated = computed(() => {
    void version.value;
    return state.value.snapshot?.truncated ?? false;
  });

  function patch(next: Partial<LogViewStoreState>) {
    Object.assign(state.value, next);
    triggerRef(state);
  }

  function commitSnapshot(snapshot: ContainerLogRealtimeBatcherSnapshot) {
    patch({
      error: '',
      snapshot,
    });
  }

  return {
    state,
    version,
    hasSnapshot,
    logs,
    entries,
    truncated,
    paused: computed(() => state.value.paused),
    setLoading(loading: boolean) {
      patch({ loading });
    },
    setError(error: string) {
      patch({ error });
    },
    commit(snapshot: ContainerLogRealtimeBatcherSnapshot) {
      if (state.value.paused) {
        if (snapshot.entryView.size === 0) {
          pendingSnapshot = null;
          commitSnapshot(snapshot);
          return;
        }
        pendingSnapshot = snapshot;
        return;
      }

      commitSnapshot(snapshot);
    },
    pause() {
      if (state.value.paused) {
        return;
      }

      patch({ paused: true });
    },
    resume() {
      if (!state.value.paused) {
        return;
      }

      const nextSnapshot = pendingSnapshot;
      pendingSnapshot = null;
      patch({ paused: false });
      if (nextSnapshot) {
        commitSnapshot(nextSnapshot);
      }
    },
    reset() {
      pendingSnapshot = null;
      patch({
        snapshot: null,
        error: '',
        loading: false,
        paused: false,
      });
    },
  };
}
