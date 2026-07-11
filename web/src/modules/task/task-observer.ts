import { openRealtimeTopicSocket, type RealtimeTopicSocketController } from '@/shared/realtime';

import { getTask } from './api/task';
import { buildTaskRealtimeTopicName, parseTaskRealtimeNotification } from './contract/realtime';
import type { TaskDetail, TaskStatus } from './types/task';

const defaultPollIntervalMs = 2_000;

const terminalTaskStatuses: readonly TaskStatus[] = ['success', 'failed', 'cancelled', 'needs_attention'];

export type TaskObserver = Readonly<{
  refresh: () => Promise<void>;
  stop: () => void;
}>;

export type TaskObserverOptions = Readonly<{
  onError?: (error: unknown) => void;
  onTask: (task: TaskDetail) => void;
  pollIntervalMs?: number;
}>;

/**
 * Reports whether a persisted Task state has reached a terminal outcome.
 */
export function isTerminalTaskStatus(status: TaskStatus) {
  return terminalTaskStatuses.includes(status);
}

/**
 * Observes one Task through its durable detail API and realtime topic.
 *
 * The observer is module-agnostic: consumers own presentation and resource
 * refresh behavior, while this module owns task transport and cleanup.
 */
export function observeTask(taskId: number, options: TaskObserverOptions): TaskObserver {
  let stopped = false;
  let refreshing = false;
  let refreshQueued = false;

  async function refresh() {
    if (stopped) return;
    if (refreshing) {
      refreshQueued = true;
      return;
    }

    refreshing = true;
    try {
      const task = await getTask(taskId);
      if (!stopped) options.onTask(task);
    } catch (error) {
      if (!stopped) options.onError?.(error);
    } finally {
      refreshing = false;
      if (refreshQueued && !stopped) {
        refreshQueued = false;
        void refresh();
      }
    }
  }

  const realtimeController: RealtimeTopicSocketController = openRealtimeTopicSocket({
    topic: buildTaskRealtimeTopicName(taskId),
    parseMessage: parseTaskRealtimeNotification,
    onMessage: (event) => {
      if (event.task_id === taskId) void refresh();
    },
    onStateChange: (state) => {
      if (state === 'open') void refresh();
    },
  });
  const pollInterval = Math.max(250, options.pollIntervalMs ?? defaultPollIntervalMs);
  const pollTimer = globalThis.setInterval(() => void refresh(), pollInterval);

  void refresh();

  return {
    refresh,
    stop: () => {
      if (stopped) return;
      stopped = true;
      globalThis.clearInterval(pollTimer);
      realtimeController.close();
    },
  };
}
