<template>
  <div :class="sideNavCls">
    <t-menu
      :class="menuCls"
      :theme="theme"
      :value="active"
      :collapsed="collapsed"
      :expanded="expanded"
      :expand-mutex="menuAutoCollapsed"
      :width="menuWidth"
      @expand="onExpanded"
    >
      <template #logo>
        <span v-if="showLogo" :class="`${prefix}-side-nav-logo-wrapper`" @click="goHome">
          <brand-identity
            :compact="logoCompact"
            :label-hidden="logoLabelHidden"
            :class="logoCls"
            :label="t('common.appName')"
          />
        </span>
      </template>
      <menu-content :nav-data="menu" :show-sections="!renderCompact" />
    </t-menu>
    <div :class="`${prefix}-side-nav-placeholder${isCompact ? '-hidden' : ''}`"></div>
  </div>
</template>
<script setup lang="ts">
import difference from 'lodash/difference';
import remove from 'lodash/remove';
import union from 'lodash/union';
import type { MenuValue } from 'tdesign-vue-next';
import type { PropType } from 'vue';
import { computed, onMounted, onUnmounted, ref, watch } from 'vue';
import { useRoute } from 'vue-router';

import { prefix } from '@/config/global';
import { findAllExpandedMenuPaths, findExpandedMenuPaths, type SidebarMotionPhase } from '@/layouts/layout-navigation';
import { useShellNavigation } from '@/layouts/useShellNavigation';
import { t } from '@/locales';
import { getActive } from '@/router';
import { BrandIdentity } from '@/shared/components/brand';
import { useSettingStore } from '@/store';
import type { MenuRoute, ModeType } from '@/utils/types';

import MenuContent from './MenuContent.vue';

// 侧栏以当前路由和后端菜单快照为输入，并在折叠动效期间暂缓展开态同步，避免菜单状态与过渡阶段互相覆盖。
const menuWidth = ['var(--graft-shell-sidebar-surface-width)', 'var(--graft-shell-sidebar-surface-width)'];

const { menu, showLogo, isFixed, layout, theme, isCompact, renderCompact, motionPhase } = defineProps({
  menu: {
    type: Array as PropType<MenuRoute[]>,
    default: () => [],
  },
  showLogo: {
    type: Boolean as PropType<boolean>,
    default: true,
  },
  isFixed: {
    type: Boolean as PropType<boolean>,
    default: true,
  },
  layout: {
    type: String as PropType<string>,
    default: '',
  },
  headerHeight: {
    type: String as PropType<string>,
    default: '64px',
  },
  theme: {
    type: String as PropType<ModeType>,
    default: 'light',
  },
  isCompact: {
    type: Boolean as PropType<boolean>,
    default: false,
  },
  renderCompact: {
    type: Boolean as PropType<boolean>,
    default: false,
  },
  motionPhase: {
    type: String as PropType<SidebarMotionPhase>,
    default: 'expanded',
  },
});

const MIN_POINT = 992 - 1;

const collapsed = computed(() => renderCompact);
const settingStore = useSettingStore();
const menuAutoCollapsed = computed(() => settingStore.menuAutoCollapsed);
const menuAlwaysExpanded = computed(() => settingStore.menuAlwaysExpanded);
const route = useRoute();

const currentNavigationPath = computed(() => {
  if (route.meta?.hiddenMenu) {
    return route.meta.navigationTargetPath ?? '';
  }

  return route.path || getActive();
});
const active = computed(() => currentNavigationPath.value);

const expanded = ref<MenuValue[]>([]);
const expandedBeforeCompact = ref<MenuValue[]>([]);
const pendingExpandedSync = ref(false);

const buildExpandedFromActive = () => {
  return findExpandedMenuPaths(menu, currentNavigationPath.value);
};

const getExpanded = () => {
  if (menuAlwaysExpanded.value) {
    expanded.value = findAllExpandedMenuPaths(menu);
    return;
  }

  const result = buildExpandedFromActive();

  expanded.value = menuAutoCollapsed.value ? result : union(result, expanded.value);
};

const isExpandedSyncDeferred = () =>
  collapsed.value ||
  motionPhase === 'collapsing-width' ||
  motionPhase === 'collapsing-submenu' ||
  motionPhase === 'collapsing-topmenu' ||
  motionPhase === 'expanding-width' ||
  motionPhase === 'expanding-topmenu';

const syncExpandedForCurrentRoute = () => {
  if (isExpandedSyncDeferred()) {
    pendingExpandedSync.value = true;
    return;
  }

  pendingExpandedSync.value = false;
  getExpanded();
};

watch(currentNavigationPath, syncExpandedForCurrentRoute, { flush: 'post' });

watch(() => menu, syncExpandedForCurrentRoute, { deep: true });

watch(menuAlwaysExpanded, syncExpandedForCurrentRoute);

watch(
  () => motionPhase,
  (nextPhase, previousPhase) => {
    if (nextPhase === previousPhase) {
      return;
    }

    if (nextPhase === 'collapsing-submenu') {
      expandedBeforeCompact.value = [...expanded.value];
      expanded.value = [];
      return;
    }

    if (nextPhase === 'compact') {
      expanded.value = [];
      return;
    }

    if (nextPhase === 'expanding-submenu') {
      const routeExpanded = buildExpandedFromActive();
      expanded.value = menuAlwaysExpanded.value
        ? findAllExpandedMenuPaths(menu)
        : menuAutoCollapsed.value
          ? routeExpanded
          : union(routeExpanded, expandedBeforeCompact.value);
      pendingExpandedSync.value = false;
      return;
    }

    if (nextPhase === 'expanded' && pendingExpandedSync.value) {
      syncExpandedForCurrentRoute();
    }
  },
);

const onExpanded = (value: MenuValue[]) => {
  if (menuAlwaysExpanded.value) {
    expanded.value = findAllExpandedMenuPaths(menu);
    return;
  }

  const requiredExpanded = buildExpandedFromActive();
  const currentOperationMenu = difference(expanded.value, value);
  const allExpanded = union(value, expanded.value, requiredExpanded);
  remove(allExpanded, (item) => currentOperationMenu.includes(item));
  requiredExpanded.forEach((item) => {
    if (!allExpanded.includes(item)) {
      allExpanded.push(item);
    }
  });
  expanded.value = allExpanded;
};

const sideMode = computed(() => {
  return theme === 'dark';
});
const sideNavCls = computed(() => {
  return [
    `${prefix}-sidebar-layout`,
    {
      [`${prefix}-sidebar-compact`]: isCompact,
    },
  ];
});
const logoCompact = computed(() => collapsed.value);
const logoLabelHidden = computed(
  () =>
    collapsed.value ||
    motionPhase === 'collapsing-width' ||
    motionPhase === 'collapsing-submenu' ||
    motionPhase === 'collapsing-topmenu' ||
    motionPhase === 'compact',
);
const logoCls = computed(() => {
  return [
    `${prefix}-side-nav-brand`,
    {
      [`${prefix}-side-nav-dark`]: sideMode.value,
    },
  ];
});
const menuCls = computed(() => {
  return [
    `${prefix}-side-nav`,
    {
      [`${prefix}-side-nav-no-logo`]: !showLogo,
      [`${prefix}-side-nav-no-fixed`]: !isFixed,
      [`${prefix}-side-nav-mix-fixed`]: layout === 'mix' && isFixed,
    },
  ];
});

const shellNavigation = useShellNavigation();

const autoCollapsed = () => {
  const isCompact = window.innerWidth <= MIN_POINT;
  settingStore.updateConfig({
    isSidebarCompact: isCompact,
  });
};

onMounted(() => {
  getExpanded();
  autoCollapsed();

  window.addEventListener('resize', autoCollapsed);
});

onUnmounted(() => {
  window.removeEventListener('resize', autoCollapsed);
});

const goHome = () => {
  void shellNavigation.goHome();
};
</script>
<style lang="less" scoped></style>
