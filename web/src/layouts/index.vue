<template>
  <div ref="shell" class="app-shell" v-bind="shellSurfaceAttrs">
    <template v-if="setting.layout.value === 'side'">
      <t-layout key="side" :class="['app-shell__layout', mainLayoutCls]">
        <t-aside v-if="shouldRenderPersistentSidebar">
          <layout-side-nav
            :drawer-visible="mobileNavigationVisible"
            :presentation="sidebarPresentation"
            :render-compact="effectiveSidebarRenderCompact"
            :width-compact="effectiveSidebarWidthCompact"
            :motion-phase="sidebarMotionPhase"
            @update:drawer-visible="mobileNavigationVisible = $event"
          />
        </t-aside>
        <t-layout class="app-shell__main">
          <t-header class="app-shell__header">
            <layout-header
              :presentation="sidebarPresentation"
              :render-compact="effectiveSidebarWidthCompact"
              @open-navigation="mobileNavigationVisible = true"
            />
          </t-header>
          <t-content class="app-shell__content"><layout-content @page-scroll="handlePageScroll" /></t-content>
        </t-layout>
      </t-layout>
    </template>

    <template v-else>
      <t-layout key="no-side" class="app-shell__layout">
        <t-header class="app-shell__header">
          <layout-header
            :presentation="sidebarPresentation"
            :render-compact="effectiveSidebarWidthCompact"
            @open-navigation="mobileNavigationVisible = true"
          />
        </t-header>
        <t-layout :class="['app-shell__main', mainLayoutCls]">
          <layout-side-nav
            v-if="shouldRenderPersistentSidebar"
            :drawer-visible="mobileNavigationVisible"
            :presentation="sidebarPresentation"
            :render-compact="effectiveSidebarRenderCompact"
            :width-compact="effectiveSidebarWidthCompact"
            :motion-phase="sidebarMotionPhase"
            @update:drawer-visible="mobileNavigationVisible = $event"
          />
          <layout-content @page-scroll="handlePageScroll" />
        </t-layout>
      </t-layout>
    </template>
    <mobile-navigation
      v-if="shouldRenderMobileNavigation"
      :menu="mobileNavigationMenu"
      :visible="mobileNavigationVisible"
      @update:visible="mobileNavigationVisible = $event"
    />
  </div>
  <component :is="updateProvider" />
  <force-password-change-dialog />
</template>
<script setup lang="ts">
import '@/style/layout.less';

import { storeToRefs } from 'pinia';
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue';
import { useRoute, useRouter } from 'vue-router';

import { prefix } from '@/config/global';
import { LOCALE } from '@/contracts/i18n/locales';
import { updateProvider } from '@/modules/update';
import { useResponsiveVariant } from '@/shared/composables';
import { emitDebugLog } from '@/shared/debug/runtime';
import { resolveResponsiveVariant } from '@/shared/responsive';
import { usePermissionStore, useRealtimeSchedulerStore, useSettingStore, useTabsRouterStore } from '@/store';
import { resolveRouteLocalizedTitle, toLocalizedTitle } from '@/utils/route/meta';
import { formatTabDebugTitle, formatTabsDebugSummary, logTabsDebug } from '@/utils/tabs-debug';
import type { AppRouteMeta, MenuRoute } from '@/utils/types';

import ForcePasswordChangeDialog from './components/ForcePasswordChangeDialog.vue';
import LayoutContent from './components/LayoutContent.vue';
import LayoutHeader from './components/LayoutHeader.vue';
import LayoutSideNav from './components/LayoutSideNav.vue';
import MobileNavigation from './components/MobileNavigation.vue';
import {
  resolveSidebarMotionMode,
  resolveSidebarPresentation,
  type SidebarMotionPhase,
  type SidebarPresentation,
} from './layout-navigation';

// 后台壳布局负责把菜单、tabs、侧栏动效和滚动状态接入统一容器；离开时必须释放所有动画与实时调度冻结。
const SIDEBAR_WIDTH_TRANSITION_MS = 320;
const SIDEBAR_COLLAPSE_SUBMENU_DELAY_MS = 124;
const SIDEBAR_COLLAPSE_TOPLEVEL_DELAY_MS = 208;
const SIDEBAR_EXPAND_TOPLEVEL_DELAY_MS = 128;
const SIDEBAR_EXPAND_SUBMENU_DELAY_MS = 224;
const SIDEBAR_PRESENTATION_STABILIZATION_MS = 180;

const route = useRoute();
const router = useRouter();
const permissionStore = usePermissionStore();
const settingStore = useSettingStore();
const realtimeSchedulerStore = useRealtimeSchedulerStore();
const tabsRouterStore = useTabsRouterStore();
const setting = storeToRefs(settingStore);
const shell = ref<HTMLElement | null>(null);
const shellVariant = useResponsiveVariant(shell);
const sidebarRenderCompact = ref(settingStore.isSidebarCompact);
const sidebarWidthCompact = ref(settingStore.isSidebarCompact);
const sidebarMotionPhase = ref<SidebarMotionPhase>(settingStore.isSidebarCompact ? 'compact' : 'expanded');
const sidebarMotionTimers = new Set<number>();
const sidebarMotionFrameIds = new Set<number>();
const pageScrollTop = ref(0);
const mobileNavigationVisible = ref(false);
let sidebarFreezeToken: number | null = null;
let sidebarResumeFrameId: number | null = null;
let sidebarResumeNestedFrameId: number | null = null;
let pageScrollFrameId: number | null = null;
let pendingPageScrollTop = 0;
let sidebarPresentationTimer: number | null = null;
let sidebarPresentationMeasured = false;

const resolveViewportSidebarPresentation = () => {
  const width = typeof window === 'undefined' ? 0 : window.innerWidth;
  return resolveSidebarPresentation(resolveResponsiveVariant(width).density);
};

const sidebarPresentationCandidate = computed(() => {
  // 容器尚未挂载时沿用当前视口密度，避免窄屏先渲染桌面侧栏再由 ResizeObserver 撤销。
  if (!shell.value?.clientWidth) {
    return resolveViewportSidebarPresentation();
  }
  return resolveSidebarPresentation(shellVariant.value.density);
});
const sidebarPresentation = ref<SidebarPresentation>(resolveViewportSidebarPresentation());
const effectiveSidebarWidthCompact = computed(
  () =>
    sidebarPresentation.value === 'compact' || (sidebarPresentation.value === 'desktop' && sidebarWidthCompact.value),
);
const effectiveSidebarRenderCompact = computed(
  () =>
    sidebarPresentation.value === 'compact' || (sidebarPresentation.value === 'desktop' && sidebarRenderCompact.value),
);

const shellSurfaceAttrs = computed(() => ({
  'data-layout-mode': settingStore.layout,
  'data-page-type': 'shell',
  'data-sidebar-compact': String(effectiveSidebarWidthCompact.value),
  'data-sidebar-fixed': String(settingStore.isSidebarFixed),
  'data-sidebar-motion-phase': sidebarMotionPhase.value,
  'data-sidebar-motion-mode': resolveSidebarMotionMode(route.meta as AppRouteMeta),
  'data-sidebar-presentation': sidebarPresentation.value,
  'data-sidebar-render-compact': String(effectiveSidebarRenderCompact.value),
  'data-sidebar-width-compact': String(effectiveSidebarWidthCompact.value),
  'data-sidebar-target-compact': String(settingStore.isSidebarCompact),
  'data-theme-mode': settingStore.displayMode,
  style: {
    '--graft-shell-sidebar-scroll-translate-y': settingStore.isSidebarFixed ? '0px' : `-${pageScrollTop.value}px`,
  },
}));

const shouldRenderSidebar = computed(
  () => settingStore.showSidebar && !(setting.layout.value === 'mix' && route.path === '/'),
);
const shouldRenderPersistentSidebar = computed(
  () => shouldRenderSidebar.value && sidebarPresentation.value !== 'drawer',
);
const shouldRenderMobileNavigation = computed(() => sidebarPresentation.value === 'drawer');
const mobileNavigationMenu = computed(() => permissionStore.routers as MenuRoute[]);

const mainLayoutCls = computed(() => [
  {
    't-layout--with-sider': shouldRenderPersistentSidebar.value,
  },
]);

const clearSidebarMotionTimers = () => {
  sidebarMotionTimers.forEach((timerId) => {
    window.clearTimeout(timerId);
  });
  sidebarMotionTimers.clear();
};

const clearSidebarPresentationTimer = () => {
  if (sidebarPresentationTimer === null) {
    return;
  }
  window.clearTimeout(sidebarPresentationTimer);
  sidebarPresentationTimer = null;
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

const stabilizeSidebarPresentation = (presentation: ReturnType<typeof resolveSidebarPresentation>) => {
  clearSidebarMotionTimers();
  clearSidebarMotionFrames();
  clearSidebarResumeFrames();
  releaseSidebarFreeze();

  if (presentation === 'desktop') {
    sidebarRenderCompact.value = settingStore.isSidebarCompact;
    sidebarWidthCompact.value = settingStore.isSidebarCompact;
    sidebarMotionPhase.value = settingStore.isSidebarCompact ? 'compact' : 'expanded';
    return;
  }

  sidebarRenderCompact.value = presentation === 'compact';
  sidebarWidthCompact.value = presentation === 'compact';
  sidebarMotionPhase.value = presentation === 'compact' ? 'compact' : 'expanded';
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

const handlePageScroll = (event: Event) => {
  const target = event.target;
  if (target instanceof HTMLElement) {
    pendingPageScrollTop = target.scrollTop;
    if (pageScrollFrameId !== null) {
      return;
    }
    pageScrollFrameId = window.requestAnimationFrame(() => {
      pageScrollFrameId = null;
      pageScrollTop.value = pendingPageScrollTop;
    });
  }
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
  clearSidebarPresentationTimer();
  clearSidebarMotionTimers();
  clearSidebarMotionFrames();
  clearSidebarResumeFrames();
  if (pageScrollFrameId !== null) {
    window.cancelAnimationFrame(pageScrollFrameId);
    pageScrollFrameId = null;
  }
  releaseSidebarFreeze();
});

watch(
  () => route.fullPath,
  (_, previousFullPath) => {
    const previousPath = previousFullPath ? router.resolve(previousFullPath).path : '';
    appendNewRoute();
    emitDebugLog('navigation', 'shell-route-observed', {
      currentPath: route.path,
      previousPath,
      sidebarPresentation: sidebarPresentation.value,
    });
    if (previousPath && previousPath !== route.path) {
      pageScrollTop.value = 0;
      document.querySelector(`.${prefix}-page-container`)?.scrollTo({ top: 0, behavior: 'smooth' });
    }
  },
);

watch(
  () => route.fullPath,
  () => {
    mobileNavigationVisible.value = false;
  },
);

// 外层布局偶发溢出时会短暂放大 ResizeObserver 的宽度；只有稳定候选值才允许重建侧栏形态。
watch(sidebarPresentationCandidate, (candidate) => {
  if (!shell.value?.clientWidth) {
    return;
  }

  if (!sidebarPresentationMeasured) {
    sidebarPresentationMeasured = true;
    sidebarPresentation.value = candidate;
    return;
  }

  if (candidate === sidebarPresentation.value) {
    clearSidebarPresentationTimer();
    return;
  }

  clearSidebarPresentationTimer();
  sidebarPresentationTimer = window.setTimeout(() => {
    sidebarPresentationTimer = null;
    sidebarPresentation.value = candidate;
  }, SIDEBAR_PRESENTATION_STABILIZATION_MS);
});

watch(sidebarPresentation, (presentation, previousPresentation) => {
  emitDebugLog('navigation', 'shell-presentation-changed', {
    current: presentation,
    previous: previousPresentation ?? 'initial',
    shellWidth: shell.value?.clientWidth ?? 0,
    viewportWidth: typeof window === 'undefined' ? 0 : window.innerWidth,
  });
  stabilizeSidebarPresentation(presentation);

  if (presentation !== 'drawer') {
    mobileNavigationVisible.value = false;
  }
});

watch(
  () => settingStore.isSidebarCompact,
  (nextCompact, previousCompact) => {
    if (nextCompact === previousCompact) {
      return;
    }

    if (sidebarPresentation.value !== 'desktop') {
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
  max-width: 100%;
  min-height: 0;
  min-width: 0;
  overflow: hidden;
  width: 100%;
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
  min-width: 0;
  width: 100%;
}

.app-shell__content {
  background: var(--graft-shell-content-bg);
  display: flex;
  flex: 1;
  flex-direction: column;
  min-height: 0;
  min-width: 0;
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
