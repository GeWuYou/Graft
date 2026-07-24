<template>
  <div class="update-preview-shell">
    <side-nav
      class="update-preview-shell__navigation"
      :menu="menu"
      :update-center-path="UPDATE_MOCK_CENTER_PATH"
      layout="side"
      :is-compact="false"
      :is-fixed="true"
      :render-compact="false"
      motion-phase="expanded"
      :show-logo="true"
      theme="light"
    />
    <main class="update-preview-shell__content">
      <router-view />
    </main>
  </div>
</template>
<script setup lang="ts">
// 开发预览壳复用正式侧栏和版本卡片，但只提供内存更新状态与开发路由。
import SideNav from '@/layouts/components/SideNav.vue';
import { usePermissionStore } from '@/store';
import { localizeRouteTitleKey } from '@/utils/route/title';
import type { MenuRoute } from '@/utils/types';

import { UPDATE_PERMISSION_CODE } from '../../contract/permissions';
import { updateCenterPreviewStatus } from '../../mock/update-center';
import { useUpdateDiscoveryStore } from '../../store/discovery';

const UPDATE_MOCK_CENTER_PATH = '/mock/platform/updates';

const permissionStore = usePermissionStore();
const discoveryStore = useUpdateDiscoveryStore();

permissionStore.setBootstrapSnapshot({
  permissions: [UPDATE_PERMISSION_CODE.READ, UPDATE_PERMISSION_CODE.MANAGE],
} as never);
discoveryStore.replaceSnapshot(updateCenterPreviewStatus);

const menu: MenuRoute[] = [
  {
    path: '/mock',
    meta: {
      title: localizeRouteTitleKey('app.home.title'),
    },
  },
  {
    path: UPDATE_MOCK_CENTER_PATH,
    meta: {
      title: localizeRouteTitleKey('update.route.center.title'),
    },
  },
];
</script>
<style scoped lang="less">
.update-preview-shell {
  background: var(--td-bg-color-page);
  display: grid;
  grid-template-columns: minmax(232px, var(--graft-shell-sidebar-surface-width)) minmax(0, 1fr);
  min-height: 100vh;
}

.update-preview-shell__navigation {
  border-right: 1px solid var(--td-component-stroke);
}

.update-preview-shell__content {
  min-width: 0;
  padding: var(--td-comp-paddingTB-xl) var(--td-comp-paddingLR-xl);
}

@media (width <= 900px) {
  .update-preview-shell {
    grid-template-columns: 1fr;
  }

  .update-preview-shell__navigation {
    display: none;
  }
}
</style>
