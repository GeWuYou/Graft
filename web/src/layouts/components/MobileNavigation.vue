<template>
  <nav class="graft-mobile-navigation" :aria-label="t('layout.mobileNavigation.label')">
    <template v-for="index in 2" :key="`left-${index}`">
      <t-button
        v-if="shortcutItems[index - 1]"
        class="graft-mobile-navigation__item"
        shape="square"
        variant="text"
        :aria-current="isActive(shortcutItems[index - 1]!) ? 'page' : undefined"
        @click="navigate(shortcutItems[index - 1]!)"
      >
        <template #icon>
          <graft-menu-icon :icon-key="menuIcon(shortcutItems[index - 1]!)" />
        </template>
        <span>{{ renderMenuTitle(shortcutItems[index - 1]!) }}</span>
      </t-button>
      <span v-else class="graft-mobile-navigation__placeholder" aria-hidden="true" />
    </template>

    <t-tooltip :content="t('layout.mobileNavigation.open')" placement="top">
      <t-button
        class="graft-mobile-navigation__all"
        theme="primary"
        shape="circle"
        :aria-label="t('layout.mobileNavigation.open')"
        :aria-pressed="visible"
        @click="emit('update:visible', true)"
      >
        <template #icon><t-icon name="application" size="22px" /></template>
      </t-button>
    </t-tooltip>

    <template v-for="index in 2" :key="`right-${index}`">
      <t-button
        v-if="shortcutItems[index + 1]"
        class="graft-mobile-navigation__item"
        shape="square"
        variant="text"
        :aria-current="isActive(shortcutItems[index + 1]!) ? 'page' : undefined"
        @click="navigate(shortcutItems[index + 1]!)"
      >
        <template #icon>
          <graft-menu-icon :icon-key="menuIcon(shortcutItems[index + 1]!)" />
        </template>
        <span>{{ renderMenuTitle(shortcutItems[index + 1]!) }}</span>
      </t-button>
      <span v-else class="graft-mobile-navigation__placeholder" aria-hidden="true" />
    </template>
  </nav>

  <t-drawer
    :visible="visible"
    attach="body"
    :close-btn="true"
    drawer-class-name="graft-mobile-navigation-sheet"
    :footer="false"
    :header="t('layout.mobileNavigation.all')"
    placement="bottom"
    :prevent-scroll-through="true"
    size="min(80dvh, 38rem)"
    @update:visible="emit('update:visible', $event)"
  >
    <l-side-nav
      :show-logo="false"
      :layout="settingStore.layout"
      drawer-mode
      :menu="menu"
      :theme="settingStore.displaySideMode"
      :is-compact="false"
      :render-compact="false"
      motion-phase="expanded"
    />
  </t-drawer>
</template>
<script setup lang="ts">
import { computed, watch } from 'vue';
import { useRoute, useRouter } from 'vue-router';

import type { LocalizedTitle } from '@/contracts/i18n/locales';
import { resolveMenuNavigationPath } from '@/layouts/layout-navigation';
import { t } from '@/locales';
import { useLocale } from '@/locales/useLocale';
import GraftMenuIcon from '@/shared/icons/MenuIcon.vue';
import { useSettingStore } from '@/store';
import type { MenuRoute } from '@/utils/types';

import LSideNav from './SideNav.vue';

// 窄屏导航只派生已授权菜单的快捷入口，完整菜单树仍由同一 bootstrap 结果提供。
const props = defineProps<{
  menu: MenuRoute[];
  visible: boolean;
}>();

const emit = defineEmits<{
  'update:visible': [visible: boolean];
}>();

const route = useRoute();
const router = useRouter();
const settingStore = useSettingStore();
const { locale } = useLocale();

const shortcutItems = computed(() =>
  [...props.menu]
    .filter((item) => item.meta?.hidden !== true)
    .sort((left, right) => (left.meta?.orderNo ?? 0) - (right.meta?.orderNo ?? 0))
    .slice(0, 4),
);

const menuIcon = (item: MenuRoute) => {
  const icon = item.icon ?? item.meta?.icon;
  return typeof icon === 'string' ? icon : undefined;
};

const renderMenuTitle = (item: MenuRoute) => {
  const title = item.title ?? item.meta?.title;
  return title?.[locale.value as keyof LocalizedTitle] ?? '';
};

const targetPath = (item: MenuRoute) => resolveMenuNavigationPath(item);

const isActive = (item: MenuRoute) => {
  const target = targetPath(item);
  return target === '/' ? route.path === '/' : route.path === target || route.path.startsWith(`${target}/`);
};

const navigate = (item: MenuRoute) => {
  const { frameBlank, frameSrc } = item.meta ?? {};
  if (frameBlank && frameSrc) {
    window.open(frameSrc, '_blank', 'noopener,noreferrer');
    return;
  }

  void router.push(targetPath(item));
};

// 成功切换路由后立即收起菜单抽屉，避免导航层遮挡目标页面的首屏操作区。
watch(
  () => route.fullPath,
  () => emit('update:visible', false),
);
</script>
