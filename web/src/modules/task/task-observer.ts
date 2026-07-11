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
 * 判断任务状态是否已达到终态。
 *
 * @returns `true` 表示任务已结束，`false` 表示任务仍在进行中。
 */
export function isTerminalTaskStatus(status: TaskStatus) {
  return terminalTaskStatuses.includes(status);
}

/**
 * 创建任务观察器，通过详情接口、实时通知和轮询持续获取任务更新。
 *
 * @param taskId - 要观察的任务标识
 * @param options - 任务更新与刷新错误回调，以及可选的轮询间隔
 * @returns 可手动刷新或停止观察的任务观察器
 */
export function observeTask(taskId: number, options: TaskObserverOptions): TaskObserver {
  let stopped = false;
  let refreshing = false;
  let refreshQueued = false;

  /**
   * 刷新任务详情，并在成功或失败时通知相应回调。
   */
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
