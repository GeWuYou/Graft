<template>
  <div class="app-shell" v-bind="shellSurfaceAttrs">
    <template v-if="setting.layout.value === 'side'">
      <t-layout key="side" :class="['app-shell__layout', mainLayoutCls]">
        <t-aside>
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
import { createLogger } from '@/utils/logger';
import { resolveRouteLocalizedTitle, toLocalizedTitle } from '@/utils/route/meta';
import type { AppRouteMeta } from '@/utils/types';

import ForcePasswordChangeDialog from './components/ForcePasswordChangeDialog.vue';
import LayoutContent from './components/LayoutContent.vue';
import LayoutHeader from './components/LayoutHeader.vue';
import LayoutSideNav from './components/LayoutSideNav.vue';
import type { SidebarMotionPhase } from './layout-navigation';

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
const logger = createLogger('layout.tabs');
const sidebarRenderCompact = ref(settingStore.isSidebarCompact);
const sidebarWidthCompact = ref(settingStore.isSidebarCompact);
const sidebarMotionPhase = ref<SidebarMotionPhase>(settingStore.isSidebarCompact ? 'compact' : 'expanded');
const sidebarMotionTimers = new Set<number>();
let sidebarFreezeToken: number | null = null;
let sidebarResumeFrameId: number | null = null;
let sidebarResumeNestedFrameId: number | null = null;

const shellSurfaceAttrs = computed(() => ({
  'data-layout-mode': settingStore.layout,
  'data-page-type': 'shell',
  'data-sidebar-compact': String(sidebarWidthCompact.value),
  'data-sidebar-motion-phase': sidebarMotionPhase.value,
  'data-sidebar-render-compact': String(sidebarRenderCompact.value),
  'data-sidebar-width-compact': String(sidebarWidthCompact.value),
  'data-sidebar-target-compact': String(settingStore.isSidebarCompact),
  'data-theme-mode': settingStore.displayMode,
}));

const mainLayoutCls = computed(() => [
  {
    't-layout--with-sider': settingStore.showSidebar,
  },
]);

const clearSidebarMotionTimers = () => {
  sidebarMotionTimers.forEach((timerId) => {
    window.clearTimeout(timerId);
  });
  sidebarMotionTimers.clear();
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

const startSidebarMotion = (targetCompact: boolean) => {
  clearSidebarMotionTimers();
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
  sidebarWidthCompact.value = false;
  sidebarMotionPhase.value = 'expanding-width';
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

  const titleObj = toLocalizedTitle(resolveRouteLocalizedTitle(route.meta as AppRouteMeta, 'tab')) ??
    toLocalizedTitle(resolveRouteLocalizedTitle(route.meta as AppRouteMeta, 'page')) ?? {
      [LOCALE.ZH_CN]: '',
      [LOCALE.EN_US]: '',
    };
  logger.debug('append route into tabs router', {
    path,
    name,
    queryKeys: Object.keys(query),
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
};

onMounted(() => {
  tabsRouterStore.healPersistedRoutes(router);
  appendNewRoute();
});

onBeforeUnmount(() => {
  clearSidebarMotionTimers();
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
  flex: 0 0 var(--graft-shell-sidebar-current-width);
  max-width: var(--graft-shell-sidebar-current-width);
  min-width: var(--graft-shell-sidebar-current-width);
  transition:
    flex-basis 0.32s cubic-bezier(0.38, 0, 0.24, 1),
    max-width 0.32s cubic-bezier(0.38, 0, 0.24, 1),
    min-width 0.32s cubic-bezier(0.38, 0, 0.24, 1),
    width 0.32s cubic-bezier(0.38, 0, 0.24, 1);
  width: var(--graft-shell-sidebar-current-width);
  will-change: flex-basis, max-width, min-width, width;
}
</style>
