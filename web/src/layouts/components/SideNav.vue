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
          <component :is="getLogo()" :class="logoCls" />
        </span>
      </template>
      <menu-content :nav-data="menu" />
      <template #operations>
        <span :class="versionCls"> {{ !collapsed ? t('common.appName') : '' }} {{ appVersion }} </span>
      </template>
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

import AssetLogoFull from '@/assets/assets-logo-full.svg?component';
import AssetLogo from '@/assets/assets-t-logo.svg?component';
import { prefix } from '@/config/global';
import { findExpandedMenuPaths, type SidebarMotionPhase } from '@/layouts/layout-navigation';
import { useShellNavigation } from '@/layouts/useShellNavigation';
import { t } from '@/locales';
import { getActive } from '@/router';
import { useSettingStore } from '@/store';
import type { MenuRoute, ModeType } from '@/utils/types';

import pgk from '../../../package.json';
import MenuContent from './MenuContent.vue';

const appVersion = 'version' in pgk ? String(pgk.version) : '';
const menuWidth = ['var(--graft-shell-sidebar-current-width)', 'var(--graft-shell-sidebar-current-width)'];

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
const menuAutoCollapsed = computed(() => useSettingStore().menuAutoCollapsed);

const active = computed(() => getActive());

const expanded = ref<MenuValue[]>([]);
const expandedBeforeCompact = ref<MenuValue[]>([]);

const buildExpandedFromActive = () => {
  return findExpandedMenuPaths(menu, getActive());
};

const getExpanded = () => {
  const result = buildExpandedFromActive();

  expanded.value = menuAutoCollapsed.value ? result : union(result, expanded.value);
};

watch(
  () => active.value,
  () => {
    if (
      collapsed.value ||
      motionPhase === 'collapsing-width' ||
      motionPhase === 'collapsing-submenu' ||
      motionPhase === 'collapsing-topmenu' ||
      motionPhase === 'expanding-width' ||
      motionPhase === 'expanding-topmenu'
    ) {
      return;
    }
    getExpanded();
  },
);

watch(
  () => menu,
  () => {
    if (
      collapsed.value ||
      motionPhase === 'collapsing-width' ||
      motionPhase === 'collapsing-submenu' ||
      motionPhase === 'collapsing-topmenu' ||
      motionPhase === 'expanding-width' ||
      motionPhase === 'expanding-topmenu'
    ) {
      return;
    }
    getExpanded();
  },
  { deep: true },
);

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
      expanded.value = menuAutoCollapsed.value ? routeExpanded : union(routeExpanded, expandedBeforeCompact.value);
    }
  },
);

const onExpanded = (value: MenuValue[]) => {
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
const logoCollapsed = computed(() => collapsed.value);
const logoCls = computed(() => {
  return [
    `${prefix}-side-nav-logo-${logoCollapsed.value ? 't' : 'tdesign'}-logo`,
    {
      [`${prefix}-side-nav-dark`]: sideMode.value,
    },
  ];
});
const versionCls = computed(() => {
  return [
    `version-container`,
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

const settingStore = useSettingStore();
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

const getLogo = () => {
  if (logoCollapsed.value) return AssetLogo;
  return AssetLogoFull;
};
</script>
<style lang="less" scoped></style>
