<template>
  <div class="app-shell" v-bind="shellSurfaceAttrs">
    <template v-if="setting.layout.value === 'side'">
      <t-layout key="side" :class="['app-shell__layout', mainLayoutCls]">
        <t-aside v-if="shouldRenderSidebar">
          <layout-side-nav
            :render-compact="sidebarRenderCompact"
            :width-compact="sidebarWidthCompact"
            :motion-phase="sidebarMotionPhase"
          />
        </t-aside>
        <t-layout class="app-shell__main">
          <t-header class="app-shell__header">
            <layout-header :render-compact="sidebarWidthCompact" />
          </t-header>
          <t-content class="app-shell__content"><layout-content /></t-content>
        </t-layout>
      </t-layout>
    </template>

    <template v-else>
      <t-layout key="no-side" class="app-shell__layout">
        <t-header class="app-shell__header">
          <layout-header :render-compact="sidebarWidthCompact" />
        </t-header>
        <t-layout :class="['app-shell__main', mainLayoutCls]">
          <layout-side-nav
            v-if="shouldRenderSidebar"
            :render-compact="sidebarRenderCompact"
            :width-compact="sidebarWidthCompact"
            :motion-phase="sidebarMotionPhase"
          />
          <layout-content />
        </t-layout>
      </t-layout>
    </template>
  </div>
  <force-password-change-dialog />
</template>
<script setup lang="ts">
import '@/style/layout.less';

import { storeToRefs } from 'pinia';
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue';
import { useRoute, useRouter } from 'vue-router';

import { prefix } from '@/config/global';
import { LOCALE } from '@/contracts/i18n/locales';
import { useRealtimeSchedulerStore, useSettingStore, useTabsRouterStore } from '@/store';
import { resolveRouteLocalizedTitle, toLocalizedTitle } from '@/utils/route/meta';
import { formatTabDebugTitle, formatTabsDebugSummary, logTabsDebug } from '@/utils/tabs-debug';
import type { AppRouteMeta } from '@/utils/types';

import ForcePasswordChangeDialog from './components/ForcePasswordChangeDialog.vue';
import LayoutContent from './components/LayoutContent.vue';
import LayoutHeader from './components/LayoutHeader.vue';
import LayoutSideNav from './components/LayoutSideNav.vue';
import { resolveSidebarMotionMode, type SidebarMotionPhase } from './layout-navigation';

const SIDEBAR_WIDTH_TRANSITION_MS = 320;
const SIDEBAR_COLLAPSE_SUBMENU_DELAY_MS = 124;
const SIDEBAR_COLLAPSE_TOPLEVEL_DELAY_MS = 208;
const SIDEBAR_EXPAND_TOPLEVEL_DELAY_MS = 128;
const SIDEBAR_EXPAND_SUBMENU_DELAY_MS = 224;

const route = useRoute();
const router = useRouter();
const settingStore = useSettingStore();
const realtimeSchedulerStore = useRealtimeSchedulerStore();
const tabsRouterStore = useTabsRouterStore();
const setting = storeToRefs(settingStore);
const sidebarRenderCompact = ref(settingStore.isSidebarCompact);
const sidebarWidthCompact = ref(settingStore.isSidebarCompact);
const sidebarMotionPhase = ref<SidebarMotionPhase>(settingStore.isSidebarCompact ? 'compact' : 'expanded');
const sidebarMotionTimers = new Set<number>();
const sidebarMotionFrameIds = new Set<number>();
let sidebarFreezeToken: number | null = null;
let sidebarResumeFrameId: number | null = null;
let sidebarResumeNestedFrameId: number | null = null;

const shellSurfaceAttrs = computed(() => ({
  'data-layout-mode': settingStore.layout,
  'data-page-type': 'shell',
  'data-sidebar-compact': String(sidebarWidthCompact.value),
  'data-sidebar-motion-phase': sidebarMotionPhase.value,
  'data-sidebar-motion-mode': resolveSidebarMotionMode(route.path),
  'data-sidebar-render-compact': String(sidebarRenderCompact.value),
  'data-sidebar-width-compact': String(sidebarWidthCompact.value),
  'data-sidebar-target-compact': String(settingStore.isSidebarCompact),
  'data-theme-mode': settingStore.displayMode,
}));

const shouldRenderSidebar = computed(
  () => settingStore.showSidebar && !(setting.layout.value === 'mix' && route.path === '/'),
);

const mainLayoutCls = computed(() => [
  {
    't-layout--with-sider': shouldRenderSidebar.value,
  },
]);

const clearSidebarMotionTimers = () => {
  sidebarMotionTimers.forEach((timerId) => {
    window.clearTimeout(timerId);
  });
  sidebarMotionTimers.clear();
};

const clearSidebarMotionFrames = () => {
  sidebarMotionFrameIds.forEach((frameId) => {
    window.cancelAnimationFrame(frameId);
  });
  sidebarMotionFrameIds.clear();
};

const clearSidebarResumeFrames = () => {
  if (sidebarResumeFrameId !== null) {
    window.cancelAnimationFrame(sidebarResumeFrameId);
    sidebarResumeFrameId = null;
  }
  if (sidebarResumeNestedFrameId !== null) {
    window.cancelAnimationFrame(sidebarResumeNestedFrameId);
    sidebarResumeNestedFrameId = null;
  }
};

const acquireSidebarFreeze = () => {
  clearSidebarResumeFrames();
  if (sidebarFreezeToken !== null) {
    return;
  }
  sidebarFreezeToken = realtimeSchedulerStore.freeze('shell-sidebar-motion');
};

const releaseSidebarFreeze = () => {
  if (sidebarFreezeToken === null) {
    return;
  }
  realtimeSchedulerStore.release(sidebarFreezeToken);
  sidebarFreezeToken = null;
};

const scheduleSidebarFreezeRelease = () => {
  clearSidebarResumeFrames();
  sidebarResumeFrameId = window.requestAnimationFrame(() => {
    sidebarResumeFrameId = null;
    sidebarResumeNestedFrameId = window.requestAnimationFrame(() => {
      sidebarResumeNestedFrameId = null;
      releaseSidebarFreeze();
    });
  });
};

const scheduleSidebarMotion = (callback: () => void, delay: number) => {
  const timerId = window.setTimeout(() => {
    sidebarMotionTimers.delete(timerId);
    callback();
  }, delay);
  sidebarMotionTimers.add(timerId);
};

const formatLayoutTabsSummary = () =>
  formatTabsDebugSummary(tabsRouterStore.tabRouters ?? tabsRouterStore.tabRouterList ?? []);

const formatCurrentRouteSummary = () =>
  `path=${route.path} fullPath=${route.fullPath} name=${String(route.name || '')} queryName=${String(
    route.query?.name || '',
  )}`;

const scheduleSidebarNextPaint = (callback: () => void) => {
  if (typeof document === 'undefined' || document.visibilityState !== 'visible') {
    scheduleSidebarMotion(callback, 0);
    return;
  }

  const firstFrameId = window.requestAnimationFrame(() => {
    sidebarMotionFrameIds.delete(firstFrameId);
    const secondFrameId = window.requestAnimationFrame(() => {
      sidebarMotionFrameIds.delete(secondFrameId);
      callback();
    });
    sidebarMotionFrameIds.add(secondFrameId);
  });
  sidebarMotionFrameIds.add(firstFrameId);
};

const startSidebarMotion = (targetCompact: boolean) => {
  clearSidebarMotionTimers();
  clearSidebarMotionFrames();
  acquireSidebarFreeze();

  if (targetCompact) {
    sidebarRenderCompact.value = false;
    sidebarWidthCompact.value = false;
    sidebarMotionPhase.value = 'collapsing-width';
    sidebarWidthCompact.value = true;
    scheduleSidebarMotion(() => {
      sidebarMotionPhase.value = 'collapsing-submenu';
    }, SIDEBAR_COLLAPSE_SUBMENU_DELAY_MS);
    scheduleSidebarMotion(() => {
      sidebarMotionPhase.value = 'collapsing-topmenu';
    }, SIDEBAR_COLLAPSE_TOPLEVEL_DELAY_MS);
    scheduleSidebarMotion(() => {
      sidebarRenderCompact.value = true;
      sidebarMotionPhase.value = 'compact';
      scheduleSidebarFreezeRelease();
    }, SIDEBAR_WIDTH_TRANSITION_MS);
    return;
  }

  sidebarRenderCompact.value = false;
  sidebarWidthCompact.value = true;
  sidebarMotionPhase.value = 'expanding-width';
  scheduleSidebarNextPaint(() => {
    sidebarWidthCompact.value = false;
  });
  scheduleSidebarMotion(() => {
    sidebarMotionPhase.value = 'expanding-topmenu';
  }, SIDEBAR_EXPAND_TOPLEVEL_DELAY_MS);
  scheduleSidebarMotion(() => {
    sidebarMotionPhase.value = 'expanding-submenu';
  }, SIDEBAR_EXPAND_SUBMENU_DELAY_MS);
  scheduleSidebarMotion(() => {
    sidebarMotionPhase.value = 'expanded';
    scheduleSidebarFreezeRelease();
  }, SIDEBAR_WIDTH_TRANSITION_MS);
};

const appendNewRoute = () => {
  const {
    path,
    fullPath,
    query,
    params,
    meta: { hidden, hiddenMenu },
    name,
  } = route;

  if (hidden && !hiddenMenu) {
    return;
  }

  const titleObj = toLocalizedTitle((route.meta as AppRouteMeta).navigationTitle) ??
    toLocalizedTitle(resolveRouteLocalizedTitle(route.meta as AppRouteMeta, 'tab')) ??
    toLocalizedTitle(resolveRouteLocalizedTitle(route.meta as AppRouteMeta, 'page')) ?? {
      [LOCALE.ZH_CN]: '',
      [LOCALE.EN_US]: '',
    };
  logTabsDebug(
    'tabs.layout',
    () =>
      `tabs debug: layout appendNewRoute before active=${tabsRouterStore.activeTabKey} route=[path=${path} fullPath=${fullPath} name=${String(
        name || '',
      )} queryName=${String(query?.name || '')} title=${formatTabDebugTitle(titleObj)}] ${formatLayoutTabsSummary()}`,
  );
  logTabsDebug('tabs.layout', () => {
    return `tabs debug: append route into tabs router path=${path} name=${String(name || '')} queryKeys=${Object.keys(query).join(',')}`;
  });
  tabsRouterStore.appendTabRouterList({
    tabKey: path,
    path,
    fullPath,
    query,
    params,
    title: titleObj,
    name,
    isAlive: true,
    meta: route.meta as AppRouteMeta,
  });
  tabsRouterStore.setActiveRoute(route);
  logTabsDebug(
    'tabs.layout',
    () => `tabs debug: layout appendNewRoute after active=${tabsRouterStore.activeTabKey} ${formatLayoutTabsSummary()}`,
  );
};

onMounted(() => {
  logTabsDebug(
    'tabs.layout',
    () =>
      `tabs debug: layout onMounted before heal active=${tabsRouterStore.activeTabKey} route=[${formatCurrentRouteSummary()}] ${formatLayoutTabsSummary()}`,
  );
  tabsRouterStore.healPersistedState();
  logTabsDebug(
    'tabs.layout',
    () =>
      `tabs debug: layout onMounted after state heal active=${tabsRouterStore.activeTabKey} ${formatLayoutTabsSummary()}`,
  );
  tabsRouterStore.healPersistedRoutes(router);
  logTabsDebug(
    'tabs.layout',
    () =>
      `tabs debug: layout onMounted after route heal active=${tabsRouterStore.activeTabKey} ${formatLayoutTabsSummary()}`,
  );
  appendNewRoute();
  tabsRouterStore.setActiveRoute(route);
  logTabsDebug(
    'tabs.layout',
    () =>
      `tabs debug: layout onMounted after active sync active=${tabsRouterStore.activeTabKey} ${formatLayoutTabsSummary()}`,
  );
});

onBeforeUnmount(() => {
  clearSidebarMotionTimers();
  clearSidebarMotionFrames();
  clearSidebarResumeFrames();
  releaseSidebarFreeze();
});

watch(
  () => route.fullPath,
  (_, previousFullPath) => {
    const previousPath = previousFullPath ? router.resolve(previousFullPath).path : '';
    appendNewRoute();
    if (previousPath && previousPath !== route.path) {
      document.querySelector(`.${prefix}-layout`)?.scrollTo({ top: 0, behavior: 'smooth' });
    }
  },
);

watch(
  () => settingStore.isSidebarCompact,
  (nextCompact, previousCompact) => {
    if (nextCompact === previousCompact) {
      return;
    }

    startSidebarMotion(nextCompact);
  },
);
</script>
<style lang="less" scoped>
.app-shell {
  --graft-shell-sidebar-width: 232px;
  --graft-shell-sidebar-width-compact: 72px;
  --graft-shell-sidebar-current-width: var(--graft-shell-sidebar-width);
  --graft-shell-sidebar-reserved-width: var(--graft-shell-sidebar-current-width);
  --graft-shell-sidebar-surface-width: var(--graft-shell-sidebar-current-width);
  --graft-shell-sidebar-translate-x: 0px;

  background: var(--graft-shell-bg);
  color: var(--td-text-color-primary);
  display: flex;
  flex-direction: column;
  height: 100%;
  min-height: 0;
  overflow: hidden;
}

.app-shell[data-sidebar-width-compact='true'] {
  --graft-shell-sidebar-current-width: var(--graft-shell-sidebar-width-compact);
}

.app-shell[data-sidebar-motion-mode='wide-table'] {
  --graft-shell-sidebar-surface-width: var(--graft-shell-sidebar-width);
}

.app-shell[data-sidebar-motion-mode='wide-table'][data-sidebar-width-compact='true'] {
  --graft-shell-sidebar-translate-x: calc(var(--graft-shell-sidebar-width-compact) - var(--graft-shell-sidebar-width));
}

.app-shell[data-sidebar-motion-mode='wide-table'][data-sidebar-motion-phase='compact'] {
  --graft-shell-sidebar-reserved-width: var(--graft-shell-sidebar-width-compact);
  --graft-shell-sidebar-surface-width: var(--graft-shell-sidebar-width-compact);
  --graft-shell-sidebar-translate-x: 0;
}

.app-shell[data-sidebar-motion-mode='wide-table'][data-sidebar-motion-phase='expanding-width'],
.app-shell[data-sidebar-motion-mode='wide-table'][data-sidebar-motion-phase='expanding-topmenu'],
.app-shell[data-sidebar-motion-mode='wide-table'][data-sidebar-motion-phase='expanding-submenu'] {
  --graft-shell-sidebar-reserved-width: var(--graft-shell-sidebar-width-compact);
}

.app-shell__layout,
.app-shell__main {
  background: transparent;
  flex: 1;
  min-height: 0;
}

.app-shell__content {
  background: var(--graft-shell-content-bg);
  display: flex;
  flex: 1;
  flex-direction: column;
  min-height: 0;
}

.app-shell__header {
  background: var(--graft-shell-header-bg);
  border-bottom: 1px solid var(--graft-shell-border-color);
}

.app-shell :deep(.t-layout),
.app-shell :deep(.t-layout__content) {
  min-height: 0;
}

.app-shell[data-layout-mode='side'] :deep(.t-layout__sider) {
  flex: 0 0 var(--graft-shell-sidebar-reserved-width);
  max-width: var(--graft-shell-sidebar-reserved-width);
  min-width: var(--graft-shell-sidebar-reserved-width);
  transition:
    flex-basis 0.32s cubic-bezier(0.38, 0, 0.24, 1),
    max-width 0.32s cubic-bezier(0.38, 0, 0.24, 1),
    min-width 0.32s cubic-bezier(0.38, 0, 0.24, 1),
    width 0.32s cubic-bezier(0.38, 0, 0.24, 1);
  width: var(--graft-shell-sidebar-reserved-width);
  will-change: flex-basis, max-width, min-width, width;
}

.app-shell[data-layout-mode='side'][data-sidebar-motion-mode='wide-table'] :deep(.t-layout__sider) {
  transition: none;
  will-change: auto;
}
</style>
