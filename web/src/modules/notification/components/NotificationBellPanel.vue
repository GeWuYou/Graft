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
            {{ t('notification.actions.markAllRead') }}
          </t-button>
        </div>

        <t-list v-if="items.length" class="notification-bell-panel__list" :split="false" size="small">
          <t-list-item v-for="item in items" :key="item.delivery_id" @click="openDetail(item)">
            <div
              class="notification-bell-panel__item"
              :class="{ 'notification-bell-panel__item--unread': item.status === 'unread' }"
            >
              <div class="notification-bell-panel__item-main">
                <strong>{{ notificationTitle(item, t) }}</strong>
                <span>{{ notificationMessage(item, t) }}</span>
                <small
                  >{{ notificationSourceLabel(item.source_module, t) }} ·
                  {{ formatCompactDateTime(item.occurred_at, locale) }}</small
                >
              </div>
              <t-tag :theme="notificationSeverityTheme(item.severity)" variant="light-outline" size="small">
                {{ t(`notification.severity.${item.severity}`) }}
              </t-tag>
            </div>
            <template #action>
              <t-button
                v-if="item.status === 'unread'"
                size="small"
                theme="default"
                variant="outline"
                @click.stop="markOneRead(item)"
              >
                {{ t('notification.actions.markRead') }}
              </t-button>
            </template>
          </t-list-item>
        </t-list>

        <t-empty v-else class="notification-bell-panel__empty" :title="emptyTitle" :description="emptyDescription" />

        <div class="notification-bell-panel__foot">
          <t-button block variant="text" @click="openAll">
            {{ t('notification.actions.viewAll') }}
          </t-button>
        </div>
      </div>
    </template>

    <t-badge :count="unreadCount" :max-count="99" :offset="[4, 4]">
      <t-button theme="default" shape="square" variant="text" :loading="loading">
        <t-icon name="mail" />
      </t-button>
    </t-badge>
  </t-popup>
</template>
<script setup lang="ts">
import { MessagePlugin } from 'tdesign-vue-next';
import { computed, onMounted, ref } from 'vue';
import { useI18n } from 'vue-i18n';
import { useRouter } from 'vue-router';

import { formatCompactDateTime } from '@/shared/components/management';

import {
  getNotifications,
  getNotificationUnreadCount,
  markNotificationRead,
  markNotificationsReadAll,
} from '../api/notification';
import { NOTIFICATION_ROUTE_PATH } from '../contract/paths';
import {
  notificationMessage,
  notificationSeverityTheme,
  notificationSourceLabel,
  notificationTitle,
} from '../shared/presentation';
import type { NotificationItem } from '../types/notification';

const { t, locale } = useI18n();
const router = useRouter();

const visible = ref(false);
const loading = ref(false);
const unreadCount = ref(0);
const items = ref<NotificationItem[]>([]);

const emptyTitle = computed(() => (loading.value ? t('notification.bell.loading') : t('notification.empty.title')));
const emptyDescription = computed(() =>
  loading.value ? t('notification.bell.loadingDescription') : t('notification.empty.description'),
);

onMounted(() => {
  void refreshUnreadCount();
});

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
  try {
    const [listResponse, countResponse] = await Promise.all([
      getNotifications({ page: 1, page_size: 5, status: 'unread' }),
      getNotificationUnreadCount(),
    ]);
    items.value = listResponse.items;
    unreadCount.value = countResponse.count;
  } catch {
    MessagePlugin.error(t('notification.messages.loadFailed'));
  } finally {
    loading.value = false;
  }
}

function handleVisibleChange(nextVisible: boolean) {
  if (nextVisible) {
    void refreshPreview();
  }
}

async function markOneRead(item: NotificationItem) {
  try {
    await markNotificationRead(item.delivery_id);
    await refreshPreview();
  } catch {
    MessagePlugin.error(t('notification.messages.markReadFailed'));
  }
}

async function markAllRead() {
  try {
    await markNotificationsReadAll();
    await refreshPreview();
    MessagePlugin.success(t('notification.messages.markAllReadSuccess'));
  } catch {
    MessagePlugin.error(t('notification.messages.markReadFailed'));
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
</script>
<style scoped lang="less">
.notification-bell-panel {
  margin: calc(0px - var(--td-comp-paddingTB-xs)) calc(0px - var(--td-comp-paddingLR-s));
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
  gap: var(--graft-density-gap-12);
  justify-content: space-between;
  width: 100%;
}

.notification-bell-panel__item-main {
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-4);
  min-width: 0;
}

.notification-bell-panel__item-main strong,
.notification-bell-panel__item-main span,
.notification-bell-panel__item-main small {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.notification-bell-panel__item-main strong {
  color: var(--td-text-color-primary);
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
  padding: var(--td-comp-paddingTB-s) var(--td-comp-paddingLR-s);
}
</style>
