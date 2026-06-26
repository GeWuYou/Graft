import { computed, shallowRef, triggerRef } from 'vue';

import type { ContainerLogResponse } from '../../types/container';
import type { ContainerLogRealtimeBatcherSnapshot } from './log-realtime-batcher';

type LogViewStoreState = Readonly<{
  snapshot: ContainerLogRealtimeBatcherSnapshot | null;
  error: string;
  loading: boolean;
  paused: boolean;
}>;

function buildLogResponse(snapshot: ContainerLogRealtimeBatcherSnapshot | null): ContainerLogResponse | null {
  if (!snapshot) {
    return null;
  }

  return Object.freeze({
    id: snapshot.id,
    lines: [...snapshot.lineView.toArray()],
    runtime: snapshot.runtime,
    stderr: snapshot.stderr,
    stdout: snapshot.stdout,
    tail: snapshot.tail,
    timestamps: snapshot.timestamps,
    truncated: snapshot.truncated,
  });
}

export function createContainerDetailLogViewStore() {
  const state = shallowRef<LogViewStoreState>({
    snapshot: null,
    error: '',
    loading: false,
    paused: false,
  });
  let pendingSnapshot: ContainerLogRealtimeBatcherSnapshot | null = null;

  const version = computed(() => state.value.snapshot?.version ?? 0);
  const logs = computed(() => {
    void version.value;
    return buildLogResponse(state.value.snapshot);
  });
  const lines = computed(() => {
    void version.value;
    return state.value.snapshot?.lineView.toArray() ?? [];
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
    logs,
    lines,
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
