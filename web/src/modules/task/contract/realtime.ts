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
  type: TaskRealtimeEventType | string;
}>;

export function buildTaskRealtimeTopicName(taskId: number) {
  return `${taskRealtimeTopicPrefix}${taskId}`;
}

export function parseTaskRealtimeNotification(raw: unknown): TaskRealtimeNotification | null {
  const data = parseRealtimeEnvelopeData(raw);
  if (!isRealtimePayloadObject(data) || typeof data.task_id !== 'number' || typeof data.type !== 'string') {
    return null;
  }

  return { task_id: data.task_id, type: data.type };
}
