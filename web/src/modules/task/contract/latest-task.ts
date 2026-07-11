import { getTasks } from '../api/task';

/**
 * Stable owner reference used by other modules to open the most recent Task.
 */
export type TaskOwnerReference = Readonly<{
  ownerId: string;
  ownerType: string;
}>;

/**
 * Returns the most recently created Task for one authorized owner.
 *
 * The Task API guarantees descending `created_at, id` order for owner-scoped
 * history, so the first item is the newest persisted Task.
 */
export async function getLatestTaskForOwner(owner: TaskOwnerReference) {
  const response = await getTasks({ limit: 1, owner_id: owner.ownerId, owner_type: owner.ownerType });
  return response.items[0] ?? null;
}
