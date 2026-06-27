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
        <router-view v-if="!isFramePage" v-slot="{ Component, route: viewRoute }">
          <transition name="fade" mode="out-in" @before-enter="() => handleBeforeEnter(viewRoute)">
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
import { computed } from 'vue';
import type { RouteLocationNormalizedLoaded } from 'vue-router';
import { useRoute } from 'vue-router';

import FramePage from '@/layouts/frame/index.vue';
import { t } from '@/locales';
import { routeLoading } from '@/router/route-loading';
import { useTabsRouterStore } from '@/store';
import { resolvePageSurfaceType } from '@/utils/route/meta';

const emit = defineEmits<{
  'page-surface-enter': [surface: ReturnType<typeof resolvePageSurfaceType>];
}>();

// <suspense>标签属于实验性功能，请谨慎使用
// 如果存在需解决/page/1=> /page/2 刷新数据问题 请修改代码 使用activeRouteFullPath 作为key
// <suspense>
//  <component :is="Component" :key="activeRouteFullPath" />
// </suspense>

// import { useRouter } from 'vue-router';
// const activeRouteFullPath = computed(() => {
//   const router = useRouter();
//   return router.currentRoute.value.fullPath;
// });

const activeTabRoute = computed(() => {
  const tabsRouterStore = useTabsRouterStore();
  return tabsRouterStore.tabRouters.find((tabRoute) => tabRoute.tabKey === tabsRouterStore.activeTabKey);
});

const shouldKeepActiveViewAlive = computed(() => {
  const tabRoute = activeTabRoute.value;
  const keepAliveConfig = tabRoute?.meta?.keepAlive ?? route.meta?.keepAlive;
  const isRouteKeepAlive = isUndefined(keepAliveConfig) || (isBoolean(keepAliveConfig) && keepAliveConfig); // 默认开启keepalive
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

const route = useRoute(); // 这个不能放到computed中，切换页面时会导致被缓存
const isFramePage = computed(() => {
  return !!route.meta?.frameSrc;
});

const handleBeforeEnter = (viewRoute?: RouteLocationNormalizedLoaded) => {
  emit('page-surface-enter', resolvePageSurfaceType(viewRoute?.meta));
};
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
