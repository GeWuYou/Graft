<template>
  <div>
    <template v-for="(item, index) in list" :key="item.path">
      <div v-if="shouldRenderSectionLabel(item, index)" class="graft-menu-section-label" role="presentation">
        {{ renderMenuTitle(item.meta?.navigationSection?.title) }}
      </div>
      <template v-if="!item.children || !item.children.length || item.meta?.single">
        <t-menu-item
          v-if="getHref(item)"
          :class="depthClass"
          :name="item.path"
          :value="getMenuValue(item)"
          @click="openHref(item)"
        >
          <template #icon>
            <graft-menu-icon :icon-key="menuIcon(item)" class="t-icon" />
          </template>
          {{ renderMenuTitle(item.title ?? item.meta?.title) }}
        </t-menu-item>
        <t-menu-item
          v-else
          :class="depthClass"
          :name="item.path"
          :value="getMenuValue(item)"
          @click="handleMenuItemClick(item)"
        >
          <template #icon>
            <graft-menu-icon :icon-key="menuIcon(item)" class="t-icon" />
          </template>
          {{ renderMenuTitle(item.title ?? item.meta?.title) }}
        </t-menu-item>
      </template>
      <t-submenu
        v-else
        :class="depthClass"
        :name="item.path"
        :value="item.path"
        :title="renderMenuTitle(item.title ?? item.meta?.title)"
      >
        <template #icon>
          <graft-menu-icon :icon-key="menuIcon(item)" class="t-icon" />
        </template>
        <menu-content v-if="item.children" :nav-data="item.children" :depth="depth + 1" :show-sections="showSections" />
      </t-submenu>
    </template>
  </div>
</template>
<script setup lang="ts">
import type { PropType } from 'vue';
import { computed } from 'vue';
import { useRouter } from 'vue-router';

import type { LocalizedTitle } from '@/contracts/i18n/locales';
import { resolveMenuNavigationPath } from '@/layouts/layout-navigation';
import { useLocale } from '@/locales/useLocale';
import GraftMenuIcon from '@/shared/icons/MenuIcon.vue';
import type { MenuRoute } from '@/utils/types';

type ListItemType = MenuRoute;

const { navData, depth, showSections } = defineProps({
  navData: {
    type: Array as PropType<MenuRoute[]>,
    default: () => [],
  },
  depth: {
    type: Number,
    default: 1,
  },
  showSections: {
    type: Boolean,
    default: false,
  },
});

const router = useRouter();

const { locale } = useLocale();
const depthClass = computed(() => `graft-menu-depth-${depth}`);

const list = computed(() => {
  return getMenuList(navData);
});

const menuIcon = (item: ListItemType) => {
  const icon = item.icon ?? item.meta?.icon;
  return typeof icon === 'string' ? icon : undefined;
};

const renderMenuTitle = (title?: LocalizedTitle) => {
  if (!title) return '';
  return title[locale.value as keyof LocalizedTitle] || '';
};

const shouldRenderSectionLabel = (item: ListItemType, index: number) => {
  if (!showSections) return false;
  const section = item.meta?.navigationSection;
  if (!section?.key) return false;
  return list.value[index - 1]?.meta?.navigationSection?.key !== section.key;
};

function getMenuList(list: MenuRoute[]): MenuRoute[] {
  if (!list || list.length === 0) {
    return [];
  }
  // 菜单顺序由后端导航元数据的 orderNo 决定，缺省项按 0 参与排序。
  list.sort((a, b) => {
    return (a.meta?.orderNo || 0) - (b.meta?.orderNo || 0);
  });
  return list
    .map((item) => {
      return {
        path: item.path,
        title: item.meta?.title as LocalizedTitle | undefined,
        icon: item.meta?.icon,
        children: getMenuList(item.children ?? []),
        meta: item.meta,
        redirect: typeof item.redirect === 'string' ? item.redirect : undefined,
      } as MenuRoute;
    })
    .filter((item) => item.meta && item.meta.hidden !== true);
}

const getHref = (item: MenuRoute) => {
  const { frameSrc, frameBlank } = item.meta ?? {};
  if (frameSrc && frameBlank) {
    return frameSrc.match(/(https?):\/\/([\w.-]+)(?:\/\S*)?/);
  }
  return null;
};

const getPath = (item: ListItemType) => item.meta?.navigationTargetPath ?? item.path;

const getMenuValue = (item: ListItemType) => {
  return String(getPath(item) ?? item.path);
};

const openHref = (item: MenuRoute) => {
  const href = getHref(item)?.[0];
  if (href) {
    window.open(href);
  }
};

const handleMenuItemClick = (item: MenuRoute) => {
  const targetPath = resolveMenuNavigationPath(item);
  void router.push(targetPath);
};
</script>
<style scoped lang="less">
.graft-menu-section-label {
  color: var(--td-text-color-placeholder);
  font-size: var(--td-font-size-s);
  line-height: var(--td-line-height-body-medium);
  padding: var(--td-comp-paddingTB-s) var(--td-comp-paddingLR-l) var(--td-comp-paddingTB-xs);
  pointer-events: none;
  user-select: none;
}
</style>
