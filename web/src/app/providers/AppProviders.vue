<template>
  <t-config-provider :global-config="globalConfig">
    <div class="app-theme-surface" :class="mode" :data-theme-mode="mode">
      <router-view />
    </div>
    <setting-com v-if="showSetting" />
  </t-config-provider>
</template>
<script setup lang="ts">
import type { GlobalConfigProvider } from 'tdesign-vue-next';
import { computed, watch } from 'vue';

import { APP_RESULT_ROUTE_PATH, resolveRecoveryRoutePath } from '@/contracts/app/routes';
import SettingCom from '@/layouts/setting.vue';
import { useLocale } from '@/locales/useLocale';
import router from '@/router';
import { useSettingStore } from '@/store';
import { usePlatformAvailabilityStore } from '@/store/modules/platform-availability';
import { store as pinia } from '@/store/pinia';

// 应用 provider 统一承接主题、TDesign locale、标准模态框位置和路由树，避免业务页面重复声明平台级交互策略。
const store = useSettingStore();
const availability = usePlatformAvailabilityStore(pinia);
availability.bindRequestBridge();
let recoveryNavigation: Promise<void> | null = null;

// 浏览器刷新与错误页按钮都经过同一恢复入口，避免首屏探测和状态监听重复消费回跳目标。
function recoverFromServiceUnavailable() {
  if (recoveryNavigation) {
    return recoveryNavigation;
  }

  recoveryNavigation = (async () => {
    if (router.currentRoute.value.path !== APP_RESULT_ROUTE_PATH.SERVICE_UNAVAILABLE) {
      return;
    }

    const requestedRedirect =
      typeof router.currentRoute.value.query.redirect === 'string' ? router.currentRoute.value.query.redirect : null;
    const pendingPath = availability.consumePendingPath();
    const redirect = resolveRecoveryRoutePath(requestedRedirect, pendingPath);
    await router.replace(redirect);
  })().finally(() => {
    recoveryNavigation = null;
  });

  return recoveryNavigation;
}

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
      void recoverFromServiceUnavailable();
    }
  },
);

const mode = computed(() => {
  return store.displayMode;
});

// 普通页面的恢复探测不卸载工作台；服务不可用结果页保持纯恢复界面，避免探测状态切换导致悬浮入口闪现。
const showSetting = computed(
  () =>
    router.currentRoute.value.path !== APP_RESULT_ROUTE_PATH.SERVICE_UNAVAILABLE &&
    availability.status !== 'unavailable',
);

const { getComponentsLocale } = useLocale();
const globalConfig = computed<GlobalConfigProvider>(() => ({
  ...getComponentsLocale.value,
  dialog: {
    ...getComponentsLocale.value.dialog,
    placement: 'center',
  },
}));

if (import.meta.env.MODE !== 'test') {
  if (availability.status === 'unknown') {
    void router.isReady().then(async () => {
      const healthy = await availability.checkHealth();
      if (healthy) {
        await recoverFromServiceUnavailable();
        return;
      }

      if (router.currentRoute.value.path !== APP_RESULT_ROUTE_PATH.SERVICE_UNAVAILABLE) {
        availability.pendingPath = router.currentRoute.value.fullPath;
        await router.replace({
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
