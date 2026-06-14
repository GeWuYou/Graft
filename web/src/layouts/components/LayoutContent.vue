<!--
  Copyright (c) 2025-2026 GeWuYou
  SPDX-License-Identifier: Apache-2.0
-->

<template>
  <t-layout :class="layoutSurfaceCls" :data-page-type="pageSurfaceType">
    <t-tabs
      v-if="settingStore.isUseTabsRouter"
      drag-sort
      theme="card"
      :class="`${prefix}-layout-tabs-nav`"
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
              <span class="route-tabs-label">
                <t-icon v-if="routeItem.isPinned" class="route-tabs-label__pin" name="pin" size="14px" />
                <span class="route-tabs-label__text">{{ renderTitle(routeItem.title) }}</span>
              </span>
            </template>
            <t-icon v-else name="home" />
            <template #dropdown>
              <t-dropdown-menu>
                <t-dropdown-item @click="() => handleRefresh(routeItem, index)">
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
        <page-container :show-footer="showFooter" :footer-text="footerText" :surface="pageSurfaceType">
          <l-content />
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
import { computed, nextTick, ref } from 'vue';
import type { LocationQueryRaw, RouteLocationRaw } from 'vue-router';
import { useRoute, useRouter } from 'vue-router';

import { prefix } from '@/config/global';
import { MESSAGE_KEY } from '@/contracts/api/messages';
import type { LocalizedTitle } from '@/contracts/i18n/locales';
import { LOCALE } from '@/contracts/i18n/locales';
import { t } from '@/locales';
import { useLocale } from '@/locales/useLocale';
import { copyText } from '@/shared/observability/copy';
import { useSettingStore, useTabsRouterStore } from '@/store';
import { renderLocalizedTitle, resolvePageSurfaceType } from '@/utils/route/meta';
import type { TRouterInfo, TTabRemoveOptions } from '@/utils/types';

import LContent from './Content.vue';
import PageContainer from './PageContainer.vue';

const route = useRoute();
const router = useRouter();

const settingStore = useSettingStore();
const tabsRouterStore = useTabsRouterStore();
const tabRouters = computed(() => tabsRouterStore.tabRouters.filter((route) => route.isAlive || route.isHome));
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
const pageSurfaceType = computed(() => resolvePageSurfaceType(route.meta));
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

const renderTitle = (title?: LocalizedTitle) => renderLocalizedTitle(title, locale.value);
const handleRefresh = (route: TRouterInfo, routeIdx: number) => {
  tabsRouterStore.startTabRefresh(routeIdx);
  nextTick(() => {
    tabsRouterStore.finishTabRefresh(routeIdx);
    void router.replace(resolveRouteLocation(route));
  });
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

// Defer until the dropdown has fully cleared; the guards only open the dialog
// for a pending close-all request when no tab menu remains active.
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

// 处理非当前路由操作的副作用
const handleOperationEffect = (type: 'other' | 'ahead' | 'behind', routeIndex: number) => {
  const { tabRouters } = tabsRouterStore;
  const currentKey = activeTabKey.value;

  const currentIdx = tabRouters.findIndex(
    (i) => getTabKey(i) === currentKey || i.path === router.currentRoute.value.path,
  );
  // 存在三种情况需要刷新当前路由
  // 点击非当前路由的关闭其他、点击非当前路由的关闭左侧且当前路由小于触发路由、点击非当前路由的关闭右侧且当前路由大于触发路由
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
  flex-direction: column;
  min-height: 100%;
}

.t-layout[data-page-type] :deep(.tdesign-starter-layout-tabs-nav) {
  background: var(--td-bg-color-container);
  border-bottom: 1px solid var(--td-component-stroke);
}

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
}

.t-layout[data-page-type] :deep(.tdesign-starter-content-layout) {
  display: flex;
  flex: 1;
  flex-direction: column;
  gap: var(--td-comp-margin-xl);
  min-height: 0;
  padding: var(--td-comp-paddingTB-xl) var(--td-comp-paddingLR-xl) 0;
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
