import { storeToRefs } from 'pinia';
import { effectScope, watch } from 'vue';

import { useRealtimeSchedulerStore } from '@/store/modules/realtime-scheduler';

type SnapshotCoalescer<TSnapshot> = (current: TSnapshot, next: TSnapshot) => TSnapshot;

type CreateRealtimeSnapshotGateOptions<TSnapshot> = {
  apply: (snapshot: TSnapshot) => void;
  coalesce?: SnapshotCoalescer<TSnapshot>;
};

export type RealtimeSnapshotGate<TSnapshot> = {
  clear: () => void;
  commit: (snapshot: TSnapshot) => void;
  dispose: () => void;
  flush: () => void;
};

export function createRealtimeSnapshotGate<TSnapshot>(
  options: CreateRealtimeSnapshotGateOptions<TSnapshot>,
): RealtimeSnapshotGate<TSnapshot> {
  const schedulerStore = useRealtimeSchedulerStore();
  const { phase } = storeToRefs(schedulerStore);
  const coalesce = options.coalesce ?? ((_: TSnapshot, next: TSnapshot) => next);
  let bufferedSnapshot: TSnapshot | null = null;
  const scope = effectScope(true);

  // 调度器暂停提交时只保留合并后的最新快照；恢复阶段同步 flush，避免异步 watcher 让旧快照短暂覆盖页面。
  const flush = () => {
    if (bufferedSnapshot === null || !schedulerStore.allowSnapshotCommit) {
      return;
    }
    const nextSnapshot = bufferedSnapshot;
    bufferedSnapshot = null;
    options.apply(nextSnapshot);
  };

  scope.run(() => {
    watch(
      phase,
      (nextPhase, previousPhase) => {
        if (nextPhase === 'running' && previousPhase !== 'running') {
          flush();
        }
      },
      { flush: 'sync' },
    );
  });

  return {
    clear() {
      bufferedSnapshot = null;
    },
    commit(snapshot) {
      if (schedulerStore.allowSnapshotCommit) {
        options.apply(snapshot);
        return;
      }
      bufferedSnapshot = bufferedSnapshot === null ? snapshot : coalesce(bufferedSnapshot, snapshot);
    },
    dispose() {
      bufferedSnapshot = null;
      scope.stop();
    },
    flush,
  };
}
