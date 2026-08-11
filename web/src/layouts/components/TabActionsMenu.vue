<template>
  <t-dropdown
    :trigger="trigger"
    :hide-after-item-click="true"
    :max-column-width="'min(320px, calc(100vw - 32px))'"
    :min-column-width="'min(192px, calc(100vw - 32px))'"
    :placement="placement"
    :popup-props="popupProps"
  >
    <slot />
    <template #dropdown>
      <t-dropdown-menu>
        <t-dropdown-item v-if="!globalActionsOnly" @click="handleRefresh">
          <t-icon name="refresh" />
          {{ t('layout.tagTabs.refresh') }}
        </t-dropdown-item>
        <t-dropdown-item v-if="!globalActionsOnly" divider :disabled="tab.isHome" @click="handleDuplicateTab">
          <t-icon name="copy" />
          {{ t('layout.tagTabs.duplicate') }}
        </t-dropdown-item>
        <t-dropdown-item v-if="!globalActionsOnly" @click="handleCopyPageLink">
          <t-icon name="link" />
          {{ t('layout.tagTabs.copyLink') }}
        </t-dropdown-item>
        <t-dropdown-item v-if="!globalActionsOnly" @click="handleOpenInNewWindow">
          <t-icon name="window" />
          {{ t('layout.tagTabs.openInNewWindow') }}
        </t-dropdown-item>
        <t-dropdown-item v-if="!globalActionsOnly && !tab.isHome && !tab.isPinned" divider @click="handleTogglePinned">
          <t-icon name="pin" />
          {{ t('layout.tagTabs.pin') }}
        </t-dropdown-item>
        <t-dropdown-item v-else-if="!globalActionsOnly && !tab.isHome" divider @click="handleTogglePinned">
          <t-icon name="pin" />
          {{ t('layout.tagTabs.unpin') }}
        </t-dropdown-item>
        <t-dropdown-item v-if="!globalActionsOnly" divider :disabled="!hasClosableTabsAhead" @click="handleCloseAhead">
          <t-icon name="arrow-left" />
          {{ t('layout.tagTabs.closeLeft') }}
        </t-dropdown-item>
        <t-dropdown-item v-if="!globalActionsOnly" :disabled="!hasClosableTabsBehind" @click="handleCloseBehind">
          <t-icon name="arrow-right" />
          {{ t('layout.tagTabs.closeRight') }}
        </t-dropdown-item>
        <t-dropdown-item v-if="!globalActionsOnly" :disabled="!hasClosableOther" @click="handleCloseOther">
          <t-icon name="close-circle" />
          {{ t('layout.tagTabs.closeOther') }}
        </t-dropdown-item>
        <t-dropdown-item :divider="globalActionsOnly" :disabled="!hasClosableTabs" @click="handleCloseAll">
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
</template>
<script setup lang="ts">
import type { DropdownProps, PopupProps, PopupVisibleChangeContext } from 'tdesign-vue-next';
import { MessagePlugin } from 'tdesign-vue-next/es/message';
import { computed, nextTick, ref } from 'vue';
import type { LocationQueryRaw, RouteLocationRaw } from 'vue-router';
import { useRoute, useRouter } from 'vue-router';

import { t } from '@/locales';
import { copyText } from '@/shared/observability/copy';
import { useTabsRouterStore } from '@/store';
import type { TRouterInfo } from '@/utils/types';

const props = withDefaults(
  defineProps<{
    tab: TRouterInfo;
    tabIndex: number;
    globalActionsOnly?: boolean;
    popupPropsOverride?: Partial<PopupProps>;
    trigger?: 'context-menu' | 'click';
    placement?: DropdownProps['placement'];
  }>(),
  {
    trigger: 'context-menu',
    placement: 'bottom-left',
    globalActionsOnly: false,
    popupPropsOverride: undefined,
  },
);

// 页签动作统一使用同一份菜单和关闭逻辑，避免点击入口与右键入口出现行为漂移。
const route = useRoute();
const router = useRouter();
const tabsRouterStore = useTabsRouterStore();
const activeTabKeyForMenu = ref<string | null>('');
const closeAllDialogVisible = ref(false);
const pendingCloseAllDialog = ref(false);

const tabRouters = computed(() => tabsRouterStore.tabRouters);
const globalActionsOnly = computed(() => props.globalActionsOnly);
const activeTabKey = computed(() => tabsRouterStore.activeTabKey || route.path);
const canReopenClosedTab = computed(() => tabsRouterStore.canReopenClosedTab);
const hasClosableTabs = computed(() => tabRouters.value.some((item) => !item.isHome && !item.isPinned));
const hasClosableTabsAhead = computed(() =>
  tabRouters.value.some((item, index) => index < props.tabIndex && !item.isHome && !item.isPinned),
);
const hasClosableTabsBehind = computed(() =>
  tabRouters.value.some((item, index) => index > props.tabIndex && !item.isHome && !item.isPinned),
);
const hasClosableOther = computed(() => {
  const tabKey = getTabKey(props.tab);
  return tabRouters.value.some((item) => !item.isHome && !item.isPinned && getTabKey(item) !== tabKey);
});

const normalizeQuery = (query?: TRouterInfo['query']): LocationQueryRaw | undefined => query;
const getTabKey = (tab: TRouterInfo) => tab.tabKey || tab.path;
const resolveRouteLocation = (targetTab: TRouterInfo): RouteLocationRaw =>
  tabsRouterStore.resolveNavigationTarget(targetTab) || {
    path: targetTab.path,
    query: normalizeQuery(targetTab.query),
  };
const navigateToTab = (targetTab?: TRouterInfo | null) => {
  if (!targetTab) return;
  tabsRouterStore.setActiveTabKey(getTabKey(targetTab));
  void router.push(resolveRouteLocation(targetTab));
};

const popupProps = computed(() => ({
  overlayClassName: 'route-tabs-dropdown',
  onVisibleChange: (visible: boolean, context: PopupVisibleChangeContext) => handleMenuVisibleChange(visible, context),
  visible: activeTabKeyForMenu.value === getTabKey(props.tab),
  ...props.popupPropsOverride,
}));

const runTabRefresh = async () => {
  const tabKey = getTabKey(props.tab);
  if (tabsRouterStore.activeTabKey !== tabKey) {
    tabsRouterStore.setActiveTabKey(tabKey);
    await router.push(resolveRouteLocation(props.tab));
  }
  await nextTick();
  tabsRouterStore.startTabRefresh(tabKey);
  await nextTick();
  tabsRouterStore.finishTabRefresh(tabKey);
};

const handleRefresh = () => {
  void runTabRefresh();
  activeTabKeyForMenu.value = null;
};
const handleCloseAhead = () => {
  tabsRouterStore.subtractTabRouterAhead({ tabKey: getTabKey(props.tab), path: '', routeIdx: props.tabIndex });
  handleOperationEffect('ahead');
};
const handleCloseBehind = () => {
  tabsRouterStore.subtractTabRouterBehind({ tabKey: getTabKey(props.tab), path: '', routeIdx: props.tabIndex });
  handleOperationEffect('behind');
};
const handleCloseOther = () => {
  const retainedTabKey = getTabKey(props.tab);
  tabsRouterStore.subtractTabRouterOther({ tabKey: retainedTabKey, path: '', routeIdx: props.tabIndex });
  handleOperationEffect('other', retainedTabKey);
};

const openPendingCloseAllDialog = () => {
  void nextTick(() => {
    if (!pendingCloseAllDialog.value || activeTabKeyForMenu.value) return;
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
  navigateToTab(
    tabsRouterStore.tabRouters.find((item) => getTabKey(item) === activeTabKey.value) ?? tabsRouterStore.tabRouters[0],
  );
  activeTabKeyForMenu.value = null;
};
const handleTogglePinned = () => {
  tabsRouterStore.togglePinnedTab(getTabKey(props.tab));
  activeTabKeyForMenu.value = null;
};
const handleReopenClosedTab = () => {
  navigateToTab(tabsRouterStore.reopenClosedTab());
  activeTabKeyForMenu.value = null;
};
const handleDuplicateTab = () => {
  navigateToTab(tabsRouterStore.duplicateTab(getTabKey(props.tab)));
  activeTabKeyForMenu.value = null;
};

const resolveAbsolutePageUrl = (targetTab: TRouterInfo) => {
  const resolved = router.resolve(resolveRouteLocation(targetTab));
  return new URL(resolved.href, window.location.origin).href;
};
const handleCopyPageLink = async () => {
  try {
    const copied = await copyText(resolveAbsolutePageUrl(props.tab));
    MessagePlugin[copied ? 'success' : 'error'](
      t(copied ? 'layout.tagTabs.copyLinkSuccess' : 'layout.tagTabs.copyLinkFail'),
    );
  } catch {
    MessagePlugin.error(t('layout.tagTabs.copyLinkFail'));
  }
  activeTabKeyForMenu.value = null;
};
const handleOpenInNewWindow = () => {
  window.open(resolveAbsolutePageUrl(props.tab), '_blank', 'noopener,noreferrer');
  activeTabKeyForMenu.value = null;
};

const handleOperationEffect = (type: 'other' | 'ahead' | 'behind', retainedTabKey?: string) => {
  if (retainedTabKey) {
    const retainedTab = tabRouters.value.find((item) => getTabKey(item) === retainedTabKey);
    if (retainedTab) {
      navigateToTab(retainedTab);
      activeTabKeyForMenu.value = null;
      return;
    }
  }
  const currentIndex = tabRouters.value.findIndex((item) => getTabKey(item) === activeTabKey.value);
  const needRefreshRouter =
    (type === 'other' && currentIndex !== props.tabIndex) ||
    (type === 'ahead' && currentIndex < props.tabIndex) ||
    (type === 'behind' && currentIndex === -1);
  if (needRefreshRouter) {
    const fallbackTab =
      type === 'behind'
        ? (tabRouters.value.find((item) => getTabKey(item) === getTabKey(props.tab)) ??
          tabRouters.value[tabRouters.value.length - 1])
        : tabRouters.value[1];
    navigateToTab(fallbackTab);
  }
  activeTabKeyForMenu.value = null;
};
const handleMenuVisibleChange = (visible: boolean, context: PopupVisibleChangeContext) => {
  if (visible) {
    activeTabKeyForMenu.value = getTabKey(props.tab);
    return;
  }
  if (activeTabKeyForMenu.value === getTabKey(props.tab) || context.trigger === 'document') {
    activeTabKeyForMenu.value = null;
  }
  if (pendingCloseAllDialog.value) openPendingCloseAllDialog();
};
</script>
