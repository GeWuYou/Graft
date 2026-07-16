import { getTasks } from '../api/task';

/**
 * 供其他模块定位并打开指定归属对象最近一次任务的稳定引用。
 */
export type TaskOwnerReference = Readonly<{
  ownerId: string;
  ownerType: string;
}>;

/**
 * 返回指定授权归属对象最近创建的任务。
 *
 * 任务 API 保证归属对象范围内的历史记录按 `created_at, id` 降序返回，
 * 因此首项就是持久化时间最新的任务。
 */
export async function getLatestTaskForOwner(owner: TaskOwnerReference) {
  const response = await getTasks({ limit: 1, owner_id: owner.ownerId, owner_type: owner.ownerType });
  return response.items[0] ?? null;
}
