import type { UpdateStatus } from '../types/update';
import { hasAvailableUpdate } from './releaseSelection';

/** 判断快照是否允许执行 Compose 更新，调用方仍需自行处理导航和权限来源。 */
export function isUpgradeEligible(status: UpdateStatus | null, canManage: boolean): boolean {
  return (
    hasAvailableUpdate(status) && status?.installation_profile.capability === 'compose_upgrade_available' && canManage
  );
}
