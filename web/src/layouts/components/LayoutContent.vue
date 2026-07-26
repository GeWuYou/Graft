<template>
  <t-layout ref="layoutRoot" :class="layoutSurfaceCls" :data-page-type="pageSurfaceType">
    <t-tabs
      v-if="showRouteTabs"
      drag-sort
      theme="card"
      :class="[`${prefix}-layout-tabs-nav`, 'graft-scrollbar']"
      :value="activeTabKey"
      :style="{ position: 'sticky', top: 0, width: '100%' }"
      @change="(value) => handleChangeCurrentTab(value as string)"
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
          <t-dropdown
            trigger="context-menu"
            :hide-after-item-click="true"
            :min-column-width="128"
            :popup-props="{
              overlayClassName: 'route-tabs-dropdown',
              onVisibleChange: (visible: boolean, ctx: PopupVisibleChangeContext) =>
                handleTabMenuClick(visible, ctx, getTabKey(routeItem)),
              visible: activeTabKeyForMenu === getTabKey(routeItem),
            }"
          >
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
            <template #dropdown>
              <t-dropdown-menu>
                <t-dropdown-item @click="() => handleRefresh(routeItem)">
                  <t-icon name="refresh" />
                  {{ t('layout.tagTabs.refresh') }}
                </t-dropdown-item>
                <t-dropdown-item divider @click="() => handleDuplicateTab(routeItem)">
                  <t-icon name="copy" />
                  {{ t('layout.tagTabs.duplicate') }}
                </t-dropdown-item>
                <t-dropdown-item @click="() => handleCopyPageLink(routeItem)">
                  <t-icon name="link" />
                  {{ t('layout.tagTabs.copyLink') }}
                </t-dropdown-item>
                <t-dropdown-item @click="() => handleOpenInNewWindow(routeItem)">
                  <t-icon name="window" />
                  {{ t('layout.tagTabs.openInNewWindow') }}
                </t-dropdown-item>
                <t-dropdown-item v-if="!routeItem.isPinned" divider @click="() => handleTogglePinned(routeItem)">
                  <t-icon name="pin" />
                  {{ t('layout.tagTabs.pin') }}
                </t-dropdown-item>
                <t-dropdown-item v-else divider @click="() => handleTogglePinned(routeItem)">
                  <t-icon name="pin" />
                  {{ t('layout.tagTabs.unpin') }}
                </t-dropdown-item>
                <t-dropdown-item
                  divider
                  :disabled="!hasClosableTabsAhead(index)"
                  @click="() => handleCloseAhead(routeItem.path, index)"
                >
                  <t-icon name="arrow-left" />
                  {{ t('layout.tagTabs.closeLeft') }}
                </t-dropdown-item>
                <t-dropdown-item
                  :disabled="!hasClosableTabsBehind(index)"
                  @click="() => handleCloseBehind(routeItem.path, index)"
                >
                  <t-icon name="arrow-right" />
                  {{ t('layout.tagTabs.closeRight') }}
                </t-dropdown-item>
                <t-dropdown-item
                  :disabled="!hasClosableOther(routeItem)"
                  @click="() => handleCloseOther(routeItem.path, index)"
                >
                  <t-icon name="close-circle" />
                  {{ t('layout.tagTabs.closeOther') }}
                </t-dropdown-item>
                <t-dropdown-item :disabled="!hasClosableTabs" @click="handleCloseAll">
                  <t-icon name="close-circle" />
                  {{ t('layout.tagTabs.closeAll') }}
                </t-dropdown-item>
                <t-dropdown-item divider :disabled="!canReopenClosedTab" @click="handleReopenClosedTab">
                  <t-icon name="rollback" />
                  {{ t('layout.tagTabs.reopenClosed') }}
                </t-dropdown-item>
              </t-dropdown-menu>
            </template>
          </t-dropdown>
        </template>
      </t-tab-panel>
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
    <t-dialog
      v-model:visible="closeAllDialogVisible"
      attach="body"
      :header="t('layout.tagTabs.closeAll')"
      :body="t('layout.tagTabs.closeAllConfirm')"
      :cancel-btn="t('layout.tagTabs.cancel')"
      :confirm-btn="t('layout.tagTabs.closeAll')"
      placement="center"
      theme="warning"
      @confirm="handleConfirmCloseAll"
      @cancel="handleCancelCloseAll"
      @close="handleCancelCloseAll"
    />
  </t-layout>
</template>
<script setup lang="ts">
import type { PopupVisibleChangeContext } from 'tdesign-vue-next';
import { MessagePlugin } from 'tdesign-vue-next/es/message';
import { type ComponentPublicInstance, computed, nextTick, ref, watch } from 'vue';
import type { LocationQueryRaw, RouteLocationRaw } from 'vue-router';
import { useRoute, useRouter } from 'vue-router';

import { prefix } from '@/config/global';
import { MESSAGE_KEY } from '@/contracts/api/messages';
import type { LocalizedTitle } from '@/contracts/i18n/locales';
import { LOCALE } from '@/contracts/i18n/locales';
import { t } from '@/locales';
import { useLocale } from '@/locales/useLocale';
import { useResponsiveVariant } from '@/shared/composables';
import { copyText } from '@/shared/observability/copy';
import { useSettingStore, useTabsRouterStore } from '@/store';
import { type PageSurfaceType, renderLocalizedTitle, resolvePageSurfaceType } from '@/utils/route/meta';
import { hasUnresolvedRouteTitleKey, isRouteTitleKey, localizeRouteTitleKey } from '@/utils/route/title';
import { logTabsDebug } from '@/utils/tabs-debug';
import type { AppRouteMeta, TRouterInfo, TTabRemoveOptions } from '@/utils/types';

import LContent from './Content.vue';
import PageContainer from './PageContainer.vue';

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
const activeTabKeyForMenu = ref<string | null>('');
const closeAllDialogVisible = ref(false);
const pendingCloseAllDialog = ref(false);
const activeTabKey = computed(() => tabsRouterStore.activeTabKey || route.path);
const canReopenClosedTab = computed(() => tabsRouterStore.canReopenClosedTab);
const hasClosableTabs = computed(() => tabRouters.value.some((route) => !route.isHome && !route.isPinned));
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

const normalizeQuery = (query?: TRouterInfo['query']): LocationQueryRaw | undefined => {
  return query;
};

const getTabKey = (route: TRouterInfo) => route.tabKey || route.path;

const resolveRouteLocation = (targetRoute: TRouterInfo): RouteLocationRaw => {
  return (
    tabsRouterStore.resolveNavigationTarget(targetRoute) || {
      path: targetRoute.path,
      query: normalizeQuery(targetRoute.query),
    }
  );
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

const runTabRefresh = async (route: TRouterInfo) => {
  const tabKey = getTabKey(route);

  if (tabsRouterStore.activeTabKey !== tabKey) {
    tabsRouterStore.setActiveTabKey(tabKey);
    await router.push(resolveRouteLocation(route));
  }

  await nextTick();
  tabsRouterStore.startTabRefresh(tabKey);
  await nextTick();
  tabsRouterStore.finishTabRefresh(tabKey);
};

const handleRefresh = (route: TRouterInfo) => {
  void runTabRefresh(route);
  activeTabKeyForMenu.value = null;
};
const handleCloseAhead = (tabKey: string, routeIdx: number) => {
  tabsRouterStore.subtractTabRouterAhead({ tabKey, path: '', routeIdx });

  handleOperationEffect('ahead', routeIdx);
};
const handleCloseBehind = (tabKey: string, routeIdx: number) => {
  tabsRouterStore.subtractTabRouterBehind({ tabKey, path: '', routeIdx });

  handleOperationEffect('behind', routeIdx);
};
const handleCloseOther = (tabKey: string, routeIdx: number) => {
  tabsRouterStore.subtractTabRouterOther({ tabKey, path: '', routeIdx });

  handleOperationEffect('other', routeIdx);
};

// 等待标签菜单完全收起后再打开关闭全部确认框，避免弹层与菜单同时占用交互焦点。
const openPendingCloseAllDialog = () => {
  void nextTick(() => {
    if (!pendingCloseAllDialog.value || activeTabKeyForMenu.value) {
      return;
    }

    pendingCloseAllDialog.value = false;
    closeAllDialogVisible.value = true;
  });
};

const handleCloseAll = () => {
  pendingCloseAllDialog.value = true;
  activeTabKeyForMenu.value = null;
  openPendingCloseAllDialog();
};

const handleCancelCloseAll = () => {
  pendingCloseAllDialog.value = false;
  closeAllDialogVisible.value = false;
  activeTabKeyForMenu.value = null;
};

const handleConfirmCloseAll = () => {
  pendingCloseAllDialog.value = false;
  closeAllDialogVisible.value = false;
  tabsRouterStore.closeAllClosableTabs();
  const nextRoute =
    tabsRouterStore.tabRouters.find((item) => getTabKey(item) === activeTabKey.value) ?? tabsRouterStore.tabRouters[0];
  navigateToTab(nextRoute);
  activeTabKeyForMenu.value = null;
};

const handleTogglePinned = (route: TRouterInfo) => {
  tabsRouterStore.togglePinnedTab(getTabKey(route));
  activeTabKeyForMenu.value = null;
};

const handleReopenClosedTab = () => {
  const restoredRoute = tabsRouterStore.reopenClosedTab();
  navigateToTab(restoredRoute);
  activeTabKeyForMenu.value = null;
};

const handleDuplicateTab = (route: TRouterInfo) => {
  const duplicatedRoute = tabsRouterStore.duplicateTab(getTabKey(route));
  navigateToTab(duplicatedRoute);
  activeTabKeyForMenu.value = null;
};

const resolveAbsolutePageUrl = (targetRoute: TRouterInfo) => {
  const resolved = router.resolve(resolveRouteLocation(targetRoute));
  return new URL(resolved.href, window.location.origin).href;
};

const handleCopyPageLink = async (targetRoute: TRouterInfo) => {
  try {
    const copied = await copyText(resolveAbsolutePageUrl(targetRoute));
    MessagePlugin[copied ? 'success' : 'error'](
      t(copied ? 'layout.tagTabs.copyLinkSuccess' : 'layout.tagTabs.copyLinkFail'),
    );
  } catch {
    MessagePlugin.error(t('layout.tagTabs.copyLinkFail'));
  }

  activeTabKeyForMenu.value = null;
};

const handleOpenInNewWindow = (route: TRouterInfo) => {
  window.open(resolveAbsolutePageUrl(route), '_blank', 'noopener,noreferrer');
  activeTabKeyForMenu.value = null;
};

const hasClosableTabsAhead = (routeIndex: number) => {
  return tabRouters.value.some((item, index) => index < routeIndex && !item.isHome && !item.isPinned);
};

const hasClosableTabsBehind = (routeIndex: number) => {
  return tabRouters.value.some((item, index) => index > routeIndex && !item.isHome && !item.isPinned);
};

const hasClosableOther = (routeItem: TRouterInfo) => {
  const routeKey = getTabKey(routeItem);
  return tabRouters.value.some((item) => !item.isHome && !item.isPinned && getTabKey(item) !== routeKey);
};

// 非当前标签的关闭操作可能改变当前路由，需要根据关闭方向补一次路由切换。
const handleOperationEffect = (type: 'other' | 'ahead' | 'behind', routeIndex: number) => {
  const { tabRouters } = tabsRouterStore;
  const currentKey = activeTabKey.value;

  const currentIdx = tabRouters.findIndex(
    (i) => getTabKey(i) === currentKey || i.path === router.currentRoute.value.path,
  );
  // 关闭其他、关闭左侧或关闭右侧后，只有当前标签被间接影响时才刷新路由。
  const needRefreshRouter =
    (type === 'other' && currentIdx !== routeIndex) ||
    (type === 'ahead' && currentIdx < routeIndex) ||
    (type === 'behind' && currentIdx === -1);
  if (needRefreshRouter) {
    const nextRouteIdx = type === 'behind' ? tabRouters.length - 1 : 1;
    const nextRouter = tabRouters[nextRouteIdx];
    navigateToTab(nextRouter);
  }

  activeTabKeyForMenu.value = null;
};
const handleTabMenuClick = (visible: boolean, ctx: PopupVisibleChangeContext, tabKey: string) => {
  if (visible) {
    activeTabKeyForMenu.value = tabKey;
    return;
  }

  if (activeTabKeyForMenu.value === tabKey || ctx.trigger === 'document') {
    activeTabKeyForMenu.value = null;
  }

  if (pendingCloseAllDialog.value) {
    openPendingCloseAllDialog();
  }
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
/* stylelint-enable selector-pseudo-class-no-unknown */

.route-tabs-label {
  align-items: center;
  display: inline-flex;
  gap: var(--td-comp-margin-xs);
  min-width: 0;
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
