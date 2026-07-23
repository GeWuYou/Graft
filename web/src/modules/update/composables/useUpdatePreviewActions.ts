import { computed, type Ref } from 'vue';
import { useRouter } from 'vue-router';

import { usePermissionStore } from '@/store';

import { UPDATE_ROUTE_PATH } from '../contract/paths';
import { UPDATE_PERMISSION_CODE } from '../contract/permissions';
import { useUpdateDiscoveryStore } from '../store/discovery';

/** 统一壳层更新入口的升级权限判定和导航，关闭各自的浮层后才进入更新中心。 */
export function useUpdatePreviewActions(visible: Ref<boolean>) {
  const router = useRouter();
  const permissionStore = usePermissionStore();
  const discoveryStore = useUpdateDiscoveryStore();
  const canStartUpgrade = computed(
    () =>
      Boolean(discoveryStore.status?.latest) &&
      !discoveryStore.status?.cache_stale &&
      !discoveryStore.status?.check_error &&
      discoveryStore.status?.installation_profile.capability === 'compose_upgrade_available' &&
      permissionStore.hasPermission(UPDATE_PERMISSION_CODE.MANAGE),
  );

  function openManagement() {
    visible.value = false;
    void router.push(UPDATE_ROUTE_PATH.CENTER);
  }

  function startUpgrade() {
    visible.value = false;
    void router.push({ path: UPDATE_ROUTE_PATH.CENTER, query: { upgrade: '1' } });
  }

  return { canStartUpgrade, openManagement, startUpgrade };
}
