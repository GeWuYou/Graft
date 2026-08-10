<template>
  <t-layout ref="layoutRoot" :class="layoutSurfaceCls" :data-page-type="pageSurfaceType">
    <t-tabs
      v-if="showRouteTabs"
      drag-sort
      theme="card"
      :class="[
        `${prefix}-layout-tabs-nav`,
        'graft-scrollbar-tabs',
        `graft-tab-indicator--${settingStore.tabIndicatorPosition}`,
      ]"
      :value="activeTabKey"
      :style="{ position: 'sticky', top: 0, width: '100%' }"
      @change="(value) => handleChangeCurrentTab(value as string)"
      @contextmenu.capture="handleTabBarContextMenu"
      @remove="handleRemove"
      @drag-sort="handleDragend"
    >
      <t-tab-panel
        v-for="(routeItem, index) in tabRouters"
        :key="getTabKey(routeItem)"
        :value="getTabKey(routeItem)"
        :removable="!routeItem.isHome"
        :draggable="!routeItem.isHome"
      >
        <template #label>
          <tab-actions-menu :tab="routeItem" :tab-index="index" trigger="context-menu">
            <template v-if="!routeItem.isHome">
              <span
                :ref="(element) => setTabLabelRef(getTabKey(routeItem), element)"
                class="route-tabs-label"
                :data-tab-key="getTabKey(routeItem)"
              >
                <t-icon v-if="routeItem.isPinned" class="route-tabs-label__pin" name="pin" size="14px" />
                <span class="route-tabs-label__text">{{ renderTabTitle(routeItem) }}</span>
              </span>
            </template>
            <span
              v-else
              :ref="(element) => setTabLabelRef(getTabKey(routeItem), element)"
              class="route-tabs-label"
              :data-tab-key="getTabKey(routeItem)"
            >
              <t-icon name="home" />
            </span>
          </tab-actions-menu>
        </template>
      </t-tab-panel>
      <template #action>
        <div v-if="activeTab" class="route-tabs-actions">
          <tab-actions-menu
            :tab="activeTab"
            :tab-index="activeTabIndex"
            global-actions-only
            placement="bottom-left"
            trigger="click"
            :popup-props-override="blankTabMenuPopupProps"
          >
            <span ref="blankTabMenuAnchor" class="route-tabs-blank-menu-anchor" :style="blankTabMenuAnchorStyle" />
          </tab-actions-menu>
          <tab-actions-menu :tab="activeTab" :tab-index="activeTabIndex" placement="bottom-right" trigger="click">
            <span class="route-tabs-actions__menu">
              <t-tooltip placement="bottom" :content="t('layout.tagTabs.actions')">
                <t-button
                  data-testid="route-tabs-actions"
                  class="t-tabs__btn route-tabs-actions__trigger"
                  :aria-label="t('layout.tagTabs.actions')"
                  theme="default"
                  shape="square"
                  variant="text"
                >
                  <ellipsis-icon />
                </t-button>
              </t-tooltip>
            </span>
          </tab-actions-menu>
        </div>
      </template>
    </t-tabs>
    <t-content :class="`${prefix}-content-layout`">
      <div :class="`${prefix}-content-layout__body`">
        <page-container
          :show-footer="showFooter"
          :footer-text="footerText"
          :surface="pageSurfaceType"
          @scroll="emit('page-scroll', $event)"
        >
          <l-content @page-surface-ready="handlePageSurfaceReady" />
        </page-container>
      </div>
    </t-content>
  </t-layout>
</template>
<script setup lang="ts">
import { EllipsisIcon } from 'tdesign-icons-vue-next';
import type { PopupVisibleChangeContext } from 'tdesign-vue-next';
import { type ComponentPublicInstance, computed, type CSSProperties, nextTick, ref, watch } from 'vue';
import type { LocationQueryRaw, RouteLocationRaw } from 'vue-router';
import { useRoute, useRouter } from 'vue-router';

import { prefix } from '@/config/global';
import { MESSAGE_KEY } from '@/contracts/api/messages';
import type { LocalizedTitle } from '@/contracts/i18n/locales';
import { LOCALE } from '@/contracts/i18n/locales';
import { t } from '@/locales';
import { useLocale } from '@/locales/useLocale';
import { useResponsiveVariant } from '@/shared/composables';
import { useSettingStore, useTabsRouterStore } from '@/store';
import { type PageSurfaceType, renderLocalizedTitle, resolvePageSurfaceType } from '@/utils/route/meta';
import { hasUnresolvedRouteTitleKey, isRouteTitleKey, localizeRouteTitleKey } from '@/utils/route/title';
import { logTabsDebug } from '@/utils/tabs-debug';
import type { AppRouteMeta, TRouterInfo, TTabRemoveOptions } from '@/utils/types';

import LContent from './Content.vue';
import PageContainer from './PageContainer.vue';
import TabActionsMenu from './TabActionsMenu.vue';

const route = useRoute();
const router = useRouter();
const emit = defineEmits<{
  'page-scroll': [event: Event];
}>();

const settingStore = useSettingStore();
const tabsRouterStore = useTabsRouterStore();
const layoutRoot = ref<ComponentPublicInstance | HTMLElement | null>(null);
const layoutRootElement = computed(() => {
  if (layoutRoot.value instanceof HTMLElement) {
    return layoutRoot.value;
  }

  return layoutRoot.value?.$el instanceof HTMLElement ? layoutRoot.value.$el : null;
});
const layoutVariant = useResponsiveVariant(layoutRootElement);
// 窄宽度优先保留页面首屏和底部领域导航，标签状态仍由 tabs store 持续维护。
const showRouteTabs = computed(() => settingStore.isUseTabsRouter && layoutVariant.value.density === 'spacious');
const tabRouters = computed(() => tabsRouterStore.tabRouters);
const tabLabelRefs = new Map<string, HTMLElement>();
const activeTabKey = computed(() => tabsRouterStore.activeTabKey || route.path);
const activeTab = computed(
  () => tabRouters.value.find((item) => getTabKey(item) === activeTabKey.value) ?? tabRouters.value[0],
);
const activeTabIndex = computed(() =>
  activeTab.value ? tabRouters.value.findIndex((item) => getTabKey(item) === getTabKey(activeTab.value)) : -1,
);
const blankTabMenuAnchor = ref<HTMLElement | null>(null);
const blankTabMenuVisible = ref(false);
const blankTabMenuPosition = ref({ left: 0, top: 0 });
const blankTabMenuAnchorStyle = computed<CSSProperties>(() => ({
  left: `${blankTabMenuPosition.value.left}px`,
  position: 'fixed',
  top: `${blankTabMenuPosition.value.top}px`,
}));
const blankTabMenuPopupProps = computed(() => ({
  overlayClassName: 'route-tabs-dropdown',
  onVisibleChange: (visible: boolean, context: PopupVisibleChangeContext) => {
    if (!visible || context.trigger !== 'context-menu') {
      blankTabMenuVisible.value = visible;
    }
  },
  visible: blankTabMenuVisible.value,
}));
const footerMeta = computed(() => route.meta.footer);
const showFooter = computed(() => {
  if (footerMeta.value === false) {
    return false;
  }

  return settingStore.showFooter;
});
const pageSurfaceType = ref<PageSurfaceType>(resolvePageSurfaceType(route.meta));
const layoutSurfaceCls = computed(() => [`${prefix}-layout`, `${prefix}-layout--${pageSurfaceType.value}`]);
const footerText = computed(() => {
  const footer = footerMeta.value;
  if (footer === false) {
    return t(MESSAGE_KEY.COMMON_COPYRIGHT);
  }

  const content = footer?.content;
  if (typeof content === 'string') {
    return content;
  }

  if (content) {
    return (
      content[locale.value as keyof LocalizedTitle] ||
      content[LOCALE.ZH_CN] ||
      content[LOCALE.EN_US] ||
      t(MESSAGE_KEY.COMMON_COPYRIGHT)
    );
  }

  return t(MESSAGE_KEY.COMMON_COPYRIGHT);
});

const { locale } = useLocale();
const renderTitle = (title?: LocalizedTitle) => renderLocalizedTitle(title, locale.value);

const setTabLabelRef = (tabKey: string, element: Element | ComponentPublicInstance | null) => {
  if (element instanceof HTMLElement) {
    tabLabelRefs.set(tabKey, element);
    return;
  }

  tabLabelRefs.delete(tabKey);
};

// 使用壳层导航轨道的原生滚动能力时，路由切换仍须将当前标签带回可视区域。
const revealActiveTab = () => {
  const activeTab = tabLabelRefs.get(activeTabKey.value);
  activeTab?.scrollIntoView?.({ behavior: 'smooth', block: 'nearest', inline: 'nearest' });
};

watch([activeTabKey, tabRouters], () => {
  void nextTick(revealActiveTab);
});

const hasSameLocalizedTitle = (left?: LocalizedTitle, right?: LocalizedTitle) =>
  left === right ||
  (Boolean(left && right) &&
    left?.[LOCALE.ZH_CN] === right?.[LOCALE.ZH_CN] &&
    left?.[LOCALE.EN_US] === right?.[LOCALE.EN_US]);

const isLocalizedTitleKey = (title?: LocalizedTitle, declaredTitleKey?: string) => {
  const titleKey = title?.[LOCALE.ZH_CN];
  return Boolean(titleKey && titleKey === title?.[LOCALE.EN_US] && isRouteTitleKey(titleKey, declaredTitleKey));
};

const localizeTitleKey = (title?: LocalizedTitle, declaredTitleKey?: string) =>
  isLocalizedTitleKey(title, declaredTitleKey) ? localizeRouteTitleKey(title![LOCALE.ZH_CN]) : title;

const resolveTabBaseTitle = (routeItem: TRouterInfo, routeMeta: AppRouteMeta | undefined) => {
  const routeTitle = routeItem.title;
  const defaultTabTitle = localizeTitleKey(
    routeMeta?.tabTitle ??
      routeMeta?.semanticTitle ??
      routeMeta?.title ??
      (routeMeta?.titleKey ? localizeRouteTitleKey(routeMeta.titleKey) : undefined),
    routeMeta?.titleKey,
  );

  if (
    !routeItem.isDuplicate &&
    routeItem.titleSource !== 'runtime' &&
    (hasSameLocalizedTitle(routeTitle, routeMeta?.navigationTitle) ||
      hasUnresolvedRouteTitleKey(routeTitle, routeMeta?.titleKey) ||
      Boolean(defaultTabTitle))
  ) {
    return defaultTabTitle ?? routeTitle;
  }

  return routeTitle ?? defaultTabTitle;
};

const getTabKey = (route: TRouterInfo) => route.tabKey || route.path;
const normalizeQuery = (query?: TRouterInfo['query']): LocationQueryRaw | undefined => query;
const resolveRouteLocation = (targetRoute: TRouterInfo): RouteLocationRaw =>
  tabsRouterStore.resolveNavigationTarget(targetRoute) || {
    path: targetRoute.path,
    query: normalizeQuery(targetRoute.query),
  };

const resolveLiveTabMeta = (routeItem: TRouterInfo): AppRouteMeta | undefined => {
  try {
    return {
      ...routeItem.meta,
      ...(router.resolve(resolveRouteLocation(routeItem)).meta as AppRouteMeta),
    };
  } catch {
    return routeItem.meta;
  }
};

// 标签显示标题只依赖当前打开列表、locale 与实时路由元数据，旧持久化 tab metadata 不参与权威判断。
const tabDisplayEntries = computed(() => {
  const entries = tabRouters.value
    .filter((routeItem) => !routeItem.isHome)
    .map((routeItem) => {
      const routeMeta = resolveLiveTabMeta(routeItem);
      return { routeItem, routeMeta, title: resolveTabBaseTitle(routeItem, routeMeta) };
    });
  const titleCounts = new Map<string, number>();

  entries.forEach(({ title }) => {
    const label = renderTitle(title);
    if (label) {
      titleCounts.set(label, (titleCounts.get(label) ?? 0) + 1);
    }
  });

  return entries.map(({ routeItem, routeMeta, title }) => {
    const label = renderTitle(title);
    const displayTitle = label && (titleCounts.get(label) ?? 0) > 1 ? (routeMeta?.navigationTitle ?? title) : title;

    return { displayTitle, routeItem, routeMeta, title };
  });
});

const tabDisplayTitles = computed(
  () => new Map(tabDisplayEntries.value.map((entry) => [getTabKey(entry.routeItem), entry.displayTitle])),
);

watch(
  tabDisplayEntries,
  (entries) => {
    logTabsDebug(
      'tabs.layout',
      () =>
        `tabs debug: display titles locale=${locale.value} ${entries
          .map(
            ({ displayTitle, routeItem, routeMeta, title }) =>
              `[key=${getTabKey(routeItem)} stored=${renderTitle(routeItem.title)} base=${renderTitle(title)} display=${renderTitle(
                displayTitle,
              )} liveTab=${renderTitle(routeMeta?.tabTitle)} liveNavigation=${renderTitle(
                routeMeta?.navigationTitle,
              )} liveTitleKey=${routeMeta?.titleKey || ''}]`,
          )
          .join(' ')}`,
    );
  },
  { immediate: true },
);

const navigateToTab = (targetRoute?: TRouterInfo | null) => {
  if (!targetRoute) return;
  tabsRouterStore.setActiveTabKey(getTabKey(targetRoute));
  void router.push(resolveRouteLocation(targetRoute));
};

const handleChangeCurrentTab = (tabKey: string) => {
  const { tabRouters } = tabsRouterStore;
  const targetRoute = tabRouters.find((i) => getTabKey(i) === tabKey);
  navigateToTab(targetRoute);
};

// 空白标签栏只暴露不依赖当前标签的全局操作，标签项和右侧操作按钮继续由各自菜单处理。
const handleTabBarContextMenu = (event: MouseEvent) => {
  const target = event.target;
  if (!(target instanceof Element) || target.closest('.t-tabs__nav-item, .t-tabs__operations')) {
    return;
  }

  event.preventDefault();
  blankTabMenuPosition.value = { left: event.clientX, top: event.clientY };
  blankTabMenuVisible.value = true;
};

const handleRemove = (options: TTabRemoveOptions) => {
  const tabKey = options.value as string;
  const nextRouter = tabsRouterStore.getNextRouteAfterClose(tabKey);

  tabsRouterStore.subtractCurrentTabRouter({ tabKey, path: '', routeIdx: options.index });
  if (tabKey === activeTabKey.value && nextRouter) {
    navigateToTab(nextRouter);
  }
};

const renderTabTitle = (routeItem: TRouterInfo) =>
  renderTitle(tabDisplayTitles.value.get(getTabKey(routeItem)) ?? routeItem.title);
const handlePageSurfaceReady = (surface: PageSurfaceType) => {
  pageSurfaceType.value = surface;
};

const handleDragend = (options: { currentIndex: number; targetIndex: number }) => {
  const { tabRouters } = tabsRouterStore;

  [tabRouters[options.currentIndex], tabRouters[options.targetIndex]] = [
    tabRouters[options.targetIndex],
    tabRouters[options.currentIndex],
  ];
  tabsRouterStore.healPersistedState();
};
</script>
<style lang="less" scoped>
.t-layout[data-page-type] {
  background: transparent;
  display: flex;
  flex: 1;
  flex-direction: column;
  max-width: 100%;
  min-height: 0;
  min-width: 0;
  overflow: hidden;
  width: 100%;
}

.t-layout[data-page-type] :deep(.tdesign-starter-layout-tabs-nav) {
  background: var(--td-bg-color-container);
  border-bottom: 1px solid var(--td-component-stroke);
  max-width: 100%;
  min-width: 0;
}

/* stylelint-disable selector-pseudo-class-no-unknown -- Vue SFC deep selectors target TDesign internals. */
:deep(.tdesign-starter-layout-tabs-nav .t-tabs__nav-container) {
  max-width: 100%;
  overflow: hidden;
}

:deep(.tdesign-starter-layout-tabs-nav .t-tabs__nav-scroll) {
  overflow: auto hidden;
  touch-action: pan-x;
}

:deep(.tdesign-starter-layout-tabs-nav .t-tabs__nav-wrap) {
  min-width: max-content;
  width: max-content;
}

:deep(.tdesign-starter-layout-tabs-nav .t-tabs__nav) {
  flex-wrap: nowrap;
}

:deep(.tdesign-starter-layout-tabs-nav .t-tabs__nav-item) {
  flex: 0 0 auto;
  max-width: 72vw;
}

.route-tabs-actions__trigger {
  border-radius: 0;
  height: 100%;
  min-height: 100%;
  width: var(--td-comp-size-xxl);
}

.route-tabs-actions,
.route-tabs-actions__menu {
  align-items: stretch;
  display: flex;
  height: var(--td-comp-size-xxl);
}

.route-tabs-actions__menu :deep(.t-button) {
  height: 100%;
}

.route-tabs-blank-menu-anchor {
  height: 1px;
  pointer-events: none;
  width: 1px;
}
/* stylelint-enable selector-pseudo-class-no-unknown */

.route-tabs-label {
  align-items: center;
  display: inline-flex;
  gap: var(--td-comp-margin-xs);
  min-width: 0;
}

/* TDesign 已在完整导航项上标记激活状态，色条直接贴合可点击标签项的上下边缘。 */
:deep(.graft-tab-indicator--top .t-tabs__nav-item.t-is-active::before),
:deep(.graft-tab-indicator--bottom .t-tabs__nav-item.t-is-active::after) {
  background: var(--td-brand-color);
  content: '';
  display: block;
  height: 3px;
  inset-inline: 0;
  position: absolute;
  z-index: 1;
}

:deep(.graft-tab-indicator--top .t-tabs__nav-item.t-is-active::before) {
  top: 0;
}

:deep(.graft-tab-indicator--bottom .t-tabs__nav-item.t-is-active::after) {
  bottom: 0;
}

.route-tabs-label__pin {
  color: var(--td-brand-color);
  flex: 0 0 auto;
}

.route-tabs-label__text {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.t-layout[data-page-type] :deep(.t-layout__content) {
  display: flex;
  flex: 1;
  flex-direction: column;
  min-height: 0;
  min-width: 0;
}

.t-layout[data-page-type] :deep(.tdesign-starter-content-layout) {
  display: flex;
  flex: 1;
  flex-direction: column;
  gap: var(--td-comp-margin-xl);
  min-height: 0;
  overflow-x: clip;
  padding: var(--td-comp-paddingTB-xl) var(--graft-page-side-padding) 0;
}

.t-layout[data-page-type] :deep(.tdesign-starter-content-layout__body) {
  display: flex;
  flex: 1;
  flex-direction: column;
  gap: var(--td-comp-margin-xl);
  min-height: 0;
}

.t-layout[data-page-type='overview-dashboard'] :deep(.tdesign-starter-content-layout) {
  padding-top: var(--graft-density-gap-16);
}
</style>
