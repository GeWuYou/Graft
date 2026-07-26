<template>
  <div class="route-view-host route-loading-host">
    <t-loading
      class="route-page-loading"
      :delay="80"
      :loading="isPageLoading"
      size="small"
      :text="t('layout.routeLoading')"
    >
      <div class="route-view-shell">
        <router-view v-if="!isFramePage" v-slot="{ Component }">
          <transition name="fade" mode="out-in" @after-leave="handleAfterLeave">
            <keep-alive v-if="shouldKeepActiveViewAlive">
              <component :is="Component" :key="activeViewKey" />
            </keep-alive>
            <component :is="Component" v-else :key="activeViewKey" />
          </transition>
        </router-view>
        <frame-page v-else />
      </div>
    </t-loading>
  </div>
</template>
<script setup lang="ts">
import isBoolean from 'lodash/isBoolean';
import isUndefined from 'lodash/isUndefined';
import { computed, watch } from 'vue';
import { useRoute } from 'vue-router';

import FramePage from '@/layouts/frame/index.vue';
import { t } from '@/locales';
import { routeLoading } from '@/router/route-loading';
import { useTabsRouterStore } from '@/store';
import { resolvePageSurfaceType } from '@/utils/route/meta';
import { formatTabsDebugSummary, logTabsDebug } from '@/utils/tabs-debug';

const emit = defineEmits<{
  'page-surface-ready': [surface: ReturnType<typeof resolvePageSurfaceType>];
}>();

// 内容壳负责把动态路由页面接入过渡、keep-alive 和统一加载态；页面数据仍由各模块自行管理。

const activeTabRoute = computed(() => {
  const tabsRouterStore = useTabsRouterStore();
  return tabsRouterStore.tabRouters.find((tabRoute) => tabRoute.tabKey === tabsRouterStore.activeTabKey);
});

const shouldKeepActiveViewAlive = computed(() => {
  const tabRoute = activeTabRoute.value;
  const keepAliveConfig = tabRoute?.meta?.keepAlive ?? route.meta?.keepAlive;
  const isRouteKeepAlive = isUndefined(keepAliveConfig) || (isBoolean(keepAliveConfig) && keepAliveConfig);
  return Boolean(tabRoute?.isAlive) && isRouteKeepAlive;
});

const isRefreshing = computed(() => {
  const tabsRouterStore = useTabsRouterStore();
  return tabsRouterStore.refreshing;
});
const isPageLoading = computed(() => routeLoading.value || isRefreshing.value);

const activeViewKey = computed(() => {
  const tabsRouterStore = useTabsRouterStore();
  const activeTabRoute = tabsRouterStore.tabRouters.find(
    (tabRoute) => tabRoute.tabKey === tabsRouterStore.activeTabKey,
  );
  if (activeTabRoute?.path === route.path || activeTabRoute?.fullPath === route.fullPath) {
    const tabKey = tabsRouterStore.activeTabKey;
    const refreshNonce = tabsRouterStore.refreshNonceByTabKey[tabKey] ?? 0;
    return `${tabKey}::${refreshNonce}`;
  }

  return route.fullPath || route.path;
});

// route 必须保持在组件实例级别，否则放入 computed 会改变路由切换时的缓存行为。
const route = useRoute();
const isFramePage = computed(() => {
  return !!route.meta?.frameSrc;
});

const handleAfterLeave = () => {
  emit('page-surface-ready', resolvePageSurfaceType(route.meta));
};

// 仅在 tabs.layout 调试开关开启时输出，定位动态路由与标签激活不同步导致的视图空白问题。
watch(
  [() => route.fullPath, activeTabRoute, shouldKeepActiveViewAlive, activeViewKey],
  () => {
    const tabsRouterStore = useTabsRouterStore();
    const activeTab = activeTabRoute.value;
    logTabsDebug(
      'tabs.layout',
      () =>
        `tabs debug: content route=[path=${route.path} fullPath=${route.fullPath} name=${String(
          route.name || '',
        )}] active=[key=${tabsRouterStore.activeTabKey} path=${activeTab?.path || ''} fullPath=${
          activeTab?.fullPath || ''
        } name=${String(activeTab?.name || '')}] keepAlive=${String(shouldKeepActiveViewAlive.value)} viewKey=${
          activeViewKey.value
        } ${formatTabsDebugSummary(tabsRouterStore.tabRouters)}`,
    );
  },
  { immediate: true },
);
</script>
<style lang="less" scoped>
.fade-leave-active,
.fade-enter-active {
  transition: opacity @anim-duration-slow @anim-time-fn-easing;
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}

.route-loading-host {
  display: flex;
  flex: 1 0 auto;
  flex-direction: column;
  min-height: 0;
  min-width: 0;
  position: relative;
}

.route-page-loading,
.route-loading-host :deep(.t-loading__parent),
.route-view-shell {
  display: flex;
  flex: 1;
  flex-direction: column;
  min-height: 0;
  min-width: 0;
  width: 100%;
}
</style>
