<!--
  Copyright (c) 2025-2026 GeWuYou
  SPDX-License-Identifier: Apache-2.0
-->

<template>
  <t-popup
    v-model:visible="visible"
    overlay-class-name="notification-bell-popup"
    placement="bottom-right"
    trigger="click"
    @visible-change="handleVisibleChange"
  >
    <template #content>
      <div class="notification-bell-panel">
        <div class="notification-bell-panel__head">
          <div>
            <h3>{{ t('notification.bell.title') }}</h3>
            <p>{{ t('notification.bell.unreadSummary', { count: unreadCount }) }}</p>
          </div>
          <t-button v-if="items.length" theme="primary" variant="text" size="small" @click="markAllRead">
            {{ t('notification.action.markAllRead') }}
          </t-button>
        </div>

        <t-list v-if="items.length" class="notification-bell-panel__list" :split="false" size="small">
          <t-list-item v-for="item in items" :key="item.delivery_id" @click="openDetail(item)">
            <div
              class="notification-bell-panel__item"
              :class="{ 'notification-bell-panel__item--unread': notificationView(item).status === 'unread' }"
            >
              <span
                v-if="notificationView(item).status === 'unread'"
                class="notification-bell-panel__unread-dot"
                aria-hidden="true"
              />
              <div class="notification-bell-panel__item-main">
                <strong>{{ notificationView(item).title }}</strong>
                <span>{{ notificationView(item).message }}</span>
                <small>{{ notificationView(item).compactMeta }}</small>
              </div>
              <t-tag :theme="notificationSeverityTheme(item.severity)" variant="light-outline" size="small">
                {{ notificationView(item).levelLabel }}
              </t-tag>
            </div>
          </t-list-item>
        </t-list>

        <t-empty
          v-else
          class="notification-bell-panel__empty"
          :type="previewError ? 'fail' : 'empty'"
          :title="emptyTitle"
          :description="emptyDescription"
        />

        <div class="notification-bell-panel__foot" @click="openAll">
          <t-button class="notification-bell-panel__open-center" block variant="text">
            {{ t('notification.action.viewAll') }}
          </t-button>
        </div>
      </div>
    </template>

    <t-badge :count="unreadCount" :max-count="99" :offset="[4, 4]">
      <t-button
        theme="default"
        shape="square"
        variant="text"
        :loading="loading"
        :aria-label="t('notification.bell.open')"
        :title="t('notification.bell.open')"
      >
        <t-icon name="mail" />
      </t-button>
    </t-badge>
  </t-popup>
</template>
<script setup lang="ts">
import { MessagePlugin } from 'tdesign-vue-next/es/message';
import { computed, onBeforeUnmount, onMounted, ref } from 'vue';
import { useI18n } from 'vue-i18n';
import { useRouter } from 'vue-router';

import { resolveLocalizedErrorMessage } from '@/shared/localized-api-error';

import { getNotifications, getNotificationUnreadCount, markNotificationsReadAll } from '../api/notification';
import { NOTIFICATION_ROUTE_PATH } from '../contract/paths';
import { NOTIFICATION_HEADER_REFRESH_EVENT } from '../contract/refresh';
import { notificationSeverityTheme, presentNotification } from '../shared/presentation';
import type { NotificationItem } from '../types/notification';

const { t, locale } = useI18n();
const router = useRouter();

const visible = ref(false);
const loading = ref(false);
const previewError = ref('');
const unreadCount = ref(0);
const items = ref<NotificationItem[]>([]);

const emptyTitle = computed(() => {
  if (loading.value) return t('notification.bell.loading');
  if (previewError.value) return t('notification.bell.errorTitle');
  return t('notification.bell.emptyTitle');
});
const emptyDescription = computed(() => {
  if (loading.value) return t('notification.bell.loadingDescription');
  if (previewError.value) return previewError.value;
  return t('notification.bell.emptyDescription');
});

onMounted(() => {
  void refreshUnreadCount();
  window.addEventListener(NOTIFICATION_HEADER_REFRESH_EVENT, refreshHeader);
});

onBeforeUnmount(() => {
  window.removeEventListener(NOTIFICATION_HEADER_REFRESH_EVENT, refreshHeader);
});

function refreshHeader() {
  if (visible.value) {
    void refreshPreview();
    return;
  }
  void refreshUnreadCount();
}

async function refreshUnreadCount() {
  try {
    const response = await getNotificationUnreadCount();
    unreadCount.value = response.count;
  } catch {
    unreadCount.value = 0;
  }
}

async function refreshPreview() {
  loading.value = true;
  previewError.value = '';
  try {
    const [listResponse, countResponse] = await Promise.all([
      getNotifications({ page: 1, page_size: 5, status: 'unread' }),
      getNotificationUnreadCount(),
    ]);
    items.value = listResponse.items;
    unreadCount.value = countResponse.count;
  } catch (error) {
    previewError.value = resolveLocalizedErrorMessage(t, error, t('notification.messages.loadFailed'));
    items.value = [];
    MessagePlugin.error(previewError.value);
  } finally {
    loading.value = false;
  }
}

function handleVisibleChange(nextVisible: boolean) {
  if (nextVisible) {
    void refreshPreview();
  }
}

async function markAllRead() {
  try {
    await markNotificationsReadAll();
    await refreshPreview();
    MessagePlugin.success(t('notification.messages.markAllReadSuccess'));
  } catch {
    MessagePlugin.error(t('notification.messages.markAllReadFailed'));
  }
}

function openDetail(item: NotificationItem) {
  visible.value = false;
  void router.push({
    path: NOTIFICATION_ROUTE_PATH.LIST,
    query: { delivery_id: String(item.delivery_id) },
  });
}

function openAll() {
  visible.value = false;
  void router.push(NOTIFICATION_ROUTE_PATH.LIST);
}

function notificationView(item: NotificationItem) {
  return presentNotification(item, t, locale.value);
}
</script>
<style scoped lang="less">
.notification-bell-panel {
  margin: calc(0px - var(--td-comp-paddingTB-xs)) calc(0px - var(--td-comp-paddingLR-s));
  max-width: calc(100vw - 32px);
  width: 420px;
}

.notification-bell-panel__head {
  align-items: flex-start;
  border-bottom: 1px solid var(--td-component-stroke);
  display: flex;
  justify-content: space-between;
  padding: var(--td-comp-paddingTB-l) var(--td-comp-paddingLR-xl);
}

.notification-bell-panel__head h3,
.notification-bell-panel__head p {
  margin: 0;
}

.notification-bell-panel__head h3 {
  color: var(--td-text-color-primary);
  font: var(--td-font-title-medium);
}

.notification-bell-panel__head p {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-medium);
  margin-top: var(--graft-density-gap-4);
}

.notification-bell-panel__list {
  max-height: 380px;
  overflow-y: auto;
  padding: var(--td-comp-paddingTB-s) var(--td-comp-paddingLR-s);
}

.notification-bell-panel__item {
  align-items: flex-start;
  cursor: pointer;
  display: flex;
  gap: var(--graft-density-gap-8);
  justify-content: space-between;
  width: 100%;
}

.notification-bell-panel__unread-dot {
  background: var(--td-brand-color);
  border-radius: 50%;
  flex: 0 0 auto;
  height: 6px;
  margin-top: var(--graft-density-gap-8);
  width: 6px;
}

.notification-bell-panel__item-main {
  display: flex;
  flex: 1;
  flex-direction: column;
  gap: var(--graft-density-gap-4);
  min-width: 0;
}

.notification-bell-panel__item :deep(.t-tag) {
  flex-shrink: 0;
}

.notification-bell-panel__item-main strong {
  color: var(--td-text-color-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.notification-bell-panel__item-main span {
  -webkit-box-orient: vertical;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  overflow: hidden;
}

.notification-bell-panel__item-main small {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.notification-bell-panel__item-main span,
.notification-bell-panel__item-main small {
  color: var(--td-text-color-secondary);
}

.notification-bell-panel__item--unread .notification-bell-panel__item-main strong {
  color: var(--td-brand-color);
}

.notification-bell-panel__empty {
  padding: var(--td-comp-paddingTB-xxl) var(--td-comp-paddingLR-l);
}

.notification-bell-panel__foot {
  border-top: 1px solid var(--td-component-stroke);
  cursor: pointer;
  padding: var(--td-comp-paddingTB-s) var(--td-comp-paddingLR-s);
  transition: background-color 0.2s ease;
}

.notification-bell-panel__foot:hover {
  background: var(--td-bg-color-container-hover);
}

.notification-bell-panel__open-center {
  color: var(--td-text-color-primary);
  font-weight: 500;
  justify-content: flex-start;
}
</style>
