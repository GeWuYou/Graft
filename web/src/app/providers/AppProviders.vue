<template>
  <t-config-provider :global-config="getComponentsLocale">
    <div class="app-theme-surface" :class="mode" :data-theme-mode="mode">
      <router-view />
    </div>
    <setting-com v-if="showSetting" />
  </t-config-provider>
</template>
<script setup lang="ts">
import { computed, watch } from 'vue';

import { APP_RESULT_ROUTE_PATH, resolveRecoveryRoutePath } from '@/contracts/app/routes';
import SettingCom from '@/layouts/setting.vue';
import { useLocale } from '@/locales/useLocale';
import router from '@/router';
import { useSettingStore } from '@/store';
import { usePlatformAvailabilityStore } from '@/store/modules/platform-availability';
import { store as pinia } from '@/store/pinia';

// 应用 provider 统一承接主题、TDesign locale 和路由树，业务页面只消费已准备好的运行时上下文。
const store = useSettingStore();
const availability = usePlatformAvailabilityStore(pinia);
availability.bindRequestBridge();

watch(
  () => availability.status,
  (status) => {
    const currentRoute = router.currentRoute.value;

    if (status === 'unavailable') {
      if (currentRoute.path === APP_RESULT_ROUTE_PATH.SERVICE_UNAVAILABLE) {
        return;
      }
      availability.pendingPath = currentRoute.fullPath;
      void router.replace({
        path: APP_RESULT_ROUTE_PATH.SERVICE_UNAVAILABLE,
        query: { redirect: currentRoute.fullPath },
      });
      return;
    }

    if (status === 'healthy' && currentRoute.path === APP_RESULT_ROUTE_PATH.SERVICE_UNAVAILABLE) {
      const requestedRedirect = typeof currentRoute.query.redirect === 'string' ? currentRoute.query.redirect : null;
      const pendingPath = availability.consumePendingPath();
      const redirect = resolveRecoveryRoutePath(requestedRedirect, pendingPath);
      void router.replace(redirect);
    }
  },
);

const mode = computed(() => {
  return store.displayMode;
});

// 恢复探测不卸载工作台，避免已打开的 Drawer 在健康状态恢复后被重新创建并播放进入动画。
const showSetting = computed(() => availability.status !== 'unavailable');

const { getComponentsLocale } = useLocale();

if (import.meta.env.MODE !== 'test') {
  if (availability.status === 'unknown') {
    void availability.checkHealth().then((healthy) => {
      if (!healthy && router.currentRoute.value.path !== APP_RESULT_ROUTE_PATH.SERVICE_UNAVAILABLE) {
        availability.pendingPath = router.currentRoute.value.fullPath;
        void router.replace({
          path: APP_RESULT_ROUTE_PATH.SERVICE_UNAVAILABLE,
          query: { redirect: router.currentRoute.value.fullPath },
        });
      }
    });
  }
}
</script>
<style lang="less" scoped>
.app-theme-surface {
  background: var(--td-bg-color-page);
  color: var(--td-text-color-primary);
  height: 100%;
  min-height: 100%;
}
</style>
