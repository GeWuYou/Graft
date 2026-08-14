<template>
  <div class="route-view-host route-loading-host" :aria-busy="isPageLoading">
    <div class="route-view-shell">
      <router-view v-if="!isFramePage" v-slot="{ Component }">
        <transition
          name="fade"
          mode="out-in"
          @after-enter="handleAfterEnter"
          @after-leave="handleAfterLeave"
          @before-enter="handleBeforeEnter"
        >
          <keep-alive v-if="shouldKeepActiveViewAlive">
            <component :is="Component" :key="activeViewKey" />
          </keep-alive>
          <component :is="Component" v-else :key="activeViewKey" />
        </transition>
      </router-view>
      <frame-page v-else />
    </div>
  </div>
</template>
<script setup lang="ts">
import isBoolean from 'lodash/isBoolean';
import isUndefined from 'lodash/isUndefined';
import { computed, onErrorCaptured, watch } from 'vue';
import { useRoute } from 'vue-router';

import FramePage from '@/layouts/frame/index.vue';
import { routeLoading } from '@/router/route-loading';
import { createBehaviorInvestigation } from '@/shared/debug/behavior-investigation';
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
  // 同一资源详情的 query 只表达页面内状态，不应因编辑模式或筛选参数变化重建页面实例。
  if (activeTabRoute?.path === route.path) {
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

// FRONTEND-INVESTIGATION-TEMP:route-transition-root-20260814 只记录路由过渡边界，定位白屏时的进入/离开顺序。
const routeTransitionInvestigation = createBehaviorInvestigation({
  investigationId: 'route-transition-root-20260814',
  maxEvents: 120,
  allowedSummaryKeys: ['isFramePage', 'keepAlive', 'viewKey'],
});

function emitRouteTransitionInvestigation(
  event: string,
  phase: 'LIFECYCLE' | 'ROUTE_NAVIGATION' | 'ERROR',
  payloadSummary?: { errorCode: string; name: string },
) {
  routeTransitionInvestigation.emit({
    asyncBoundary: 'sync',
    component: 'RouteContentTransition',
    event,
    phase,
    route: sanitizeDebugPath(route.path),
    source: 'layouts/components/Content.vue',
    stateSummary: {
      isFramePage: isFramePage.value,
      keepAlive: shouldKeepActiveViewAlive.value,
      viewKey: sanitizeDebugPath(activeViewKey.value),
    },
    ...(payloadSummary ? { payloadSummary } : {}),
  });
}

// 目标页面实际开始进入后再切换壳层表面，避免首次异步加载时让离场的宽表按表单宽度重排。
const handleBeforeEnter = () => {
  emitRouteTransitionInvestigation('transition.before-enter', 'ROUTE_NAVIGATION');
  emit('page-surface-ready', resolvePageSurfaceType(route.meta));
};

const handleAfterEnter = () => {
  emitRouteTransitionInvestigation('transition.after-enter', 'LIFECYCLE');
};

const handleAfterLeave = () => {
  emitRouteTransitionInvestigation('transition.after-leave', 'LIFECYCLE');
};

onErrorCaptured((error, _instance, info) => {
  emitRouteTransitionInvestigation('route-component.error-captured', 'ERROR', {
    errorCode: info,
    name: error instanceof Error ? error.name : typeof error,
  });
});

// 仅在 tabs.layout 调试开关开启时输出，定位动态路由与标签激活不同步导致的视图空白问题。
watch(
  [() => route.fullPath, activeTabRoute, shouldKeepActiveViewAlive, activeViewKey],
  () => {
    const tabsRouterStore = useTabsRouterStore();
    const activeTab = activeTabRoute.value;
    logTabsDebug(
      'tabs.layout',
      () =>
        `tabs debug: content route=[path=${sanitizeDebugPath(route.path)} fullPath=${sanitizeDebugPath(route.fullPath)} name=${String(
          route.name || '',
        )}] active=[key=${tabsRouterStore.activeTabKey} path=${activeTab?.path || ''} fullPath=${sanitizeDebugPath(
          activeTab?.fullPath,
        )} name=${String(activeTab?.name || '')}] keepAlive=${String(shouldKeepActiveViewAlive.value)} viewKey=${sanitizeDebugPath(
          activeViewKey.value,
        )} ${formatTabsDebugSummary(
          tabsRouterStore.tabRouters.map((tabRoute) => ({
            ...tabRoute,
            path: sanitizeDebugPath(tabRoute.path),
            fullPath: sanitizeDebugPath(tabRoute.fullPath),
          })),
        )}`,
    );
  },
  { immediate: true },
);

// 调试日志不应记录路由 query 或 hash，其中可能包含令牌、筛选条件或临时回跳信息。
function sanitizeDebugPath(path?: string) {
  return path?.split(/[?#]/, 1)[0] || '';
}
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

.route-view-shell {
  display: flex;
  flex: 1;
  flex-direction: column;
  min-height: 0;
  min-width: 0;
  width: 100%;
}
</style>
