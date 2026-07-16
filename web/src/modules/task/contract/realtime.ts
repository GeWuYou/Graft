import { isRealtimePayloadObject, parseRealtimeEnvelopeData } from '@/shared/realtime';

const taskRealtimeTopicPrefix = 'task:';

export type TaskRealtimeEventType =
  | 'task.created'
  | 'task.started'
  | 'task.stage.started'
  | 'task.stage.completed'
  | 'task.stage.failed'
  | 'task.log.appended'
  | 'task.completed'
  | 'task.failed'
  | 'task.cancelled'
  | 'task.updated';

export type TaskRealtimeNotification = Readonly<{
  task_id: number;
  type: string;
}>;

export function buildTaskRealtimeTopicName(taskId: number) {
  return `${taskRealtimeTopicPrefix}${taskId}`;
}

/**
 * 解析原始数据中的任务实时通知。
 *
 * @param raw - 待解析的原始数据
 * @returns 解析后的任务实时通知；数据格式无效时返回 `null`
 */
export function parseTaskRealtimeNotification(raw: unknown): TaskRealtimeNotification | null {
  const data = parseRealtimeEnvelopeData(raw);
  if (!isRealtimePayloadObject(data) || typeof data.task_id !== 'number' || typeof data.type !== 'string') {
    return null;
  }

  return { task_id: data.task_id, type: data.type };
}
