import type { UpdateStatus } from '../types/update';

/** 判断快照是否允许执行 Compose 更新，调用方仍需自行处理导航和权限来源。 */
export function isUpgradeEligible(status: UpdateStatus | null, canManage: boolean): boolean {
  return (
    Boolean(status?.latest) &&
    (status?.update_mode === 'stable_tracking' || status?.update_mode === 'beta_tracking') &&
    !status?.cache_stale &&
    !status?.check_error &&
    status?.installation_profile.capability === 'compose_upgrade_available' &&
    canManage
  );
}
