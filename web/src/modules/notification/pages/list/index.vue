<!--
  Copyright (c) 2025-2026 GeWuYou
  SPDX-License-Identifier: Apache-2.0
-->

<template>
  <advanced-query-list-page
    root-class="notification-page"
    page-type="list-form-detail"
    title-key="notification.page.title"
    description-key="notification.page.description"
    :error-message="listError"
    :error-title="t('notification.page.errorTitle')"
    :loading="loading"
    :reload-label="t('notification.actions.refresh')"
    :retry-label="t('notification.actions.refresh')"
    :source="{ labelKey: 'notification.page.eyebrow', fallback: t('notification.page.eyebrow') }"
    @reload="fetchNotifications"
  >
    <template #actions>
      <t-button theme="default" variant="outline" :loading="loading" @click="fetchNotifications">
        {{ t('notification.actions.refresh') }}
      </t-button>
      <t-button theme="primary" :disabled="!canMarkAllRead" :loading="markingAll" @click="markAllRead">
        {{ t('notification.actions.markAllRead') }}
      </t-button>
    </template>

    <template #filters>
      <div class="notification-filter-stack">
        <t-tabs v-model="filters.status" theme="normal" @change="handleStatusChange">
          <t-tab-panel value="all" :label="t('notification.tabs.all')" />
          <t-tab-panel value="unread" :label="t('notification.tabs.unread')" />
          <t-tab-panel value="read" :label="t('notification.tabs.read')" />
        </t-tabs>
        <notification-filters
          v-model="filters"
          :loading="loading"
          :source-modules="sourceModules"
          @reset="resetFilters"
          @search="handleSearch"
        />
      </div>
    </template>

    <template #table>
      <notification-table
        :current="pagination.current"
        :items="rows"
        :loading="loading"
        :page-size="pagination.pageSize"
        :total="total"
        @delete="deleteRow"
        @detail="openDetail"
        @mark-read="markOneRead"
        @page-change="handlePageChange"
      />
    </template>

    <template #detail>
      <notification-detail-drawer v-model:visible="detailVisible" :item="detailRecord" @navigate="navigateToTarget" />
    </template>
  </advanced-query-list-page>
</template>
<script setup lang="ts">
import type { TabValue } from 'tdesign-vue-next';
import { MessagePlugin } from 'tdesign-vue-next';
import { computed, onMounted, ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import { useRoute, useRouter } from 'vue-router';

import { resolveLocalizedErrorMessage } from '@/modules/shared/localized-api-error';
import { AdvancedQueryListPage } from '@/shared/components/query-list';
import {
  localDateTimeToUtcIso,
  normalizePageStateRangeForRoute,
  normalizeRouteRangeForPageState,
} from '@/shared/observability';
import { createLogger } from '@/utils/logger';

import {
  deleteNotification,
  getNotifications,
  markNotificationRead,
  markNotificationsReadAll,
} from '../../api/notification';
import NotificationDetailDrawer from '../../components/NotificationDetailDrawer.vue';
import NotificationFilters from '../../components/NotificationFilters.vue';
import NotificationTable from '../../components/NotificationTable.vue';
import { resolveNotificationNavigationLocation } from '../../contract/navigation';
import type {
  NotificationFilterState,
  NotificationItem,
  NotificationListQuery,
  NotificationStatusFilter,
} from '../../types/notification';

defineOptions({
  name: 'NotificationListIndex',
});

const { t } = useI18n();
const route = useRoute();
const router = useRouter();
const logger = createLogger('notification.list');

const loading = ref(false);
const markingAll = ref(false);
const listError = ref('');
const rows = ref<NotificationItem[]>([]);
const total = ref(0);
const detailVisible = ref(false);
const detailRecord = ref<NotificationItem | null>(null);
const filters = ref<NotificationFilterState>(createDefaultFilters());
const pagination = ref({
  current: 1,
  pageSize: 20,
});

const sourceModules = computed(() =>
  Array.from(new Set(['audit', 'scheduler', ...rows.value.map((item) => item.source_module).filter(Boolean)])).sort(
    (left, right) => left.localeCompare(right),
  ),
);
const canMarkAllRead = computed(() => rows.value.some((item) => item.status === 'unread'));

onMounted(() => {
  hydrateFromRoute();
  void fetchNotifications();
});

watch(
  () => route.query.delivery_id,
  () => {
    openRouteDelivery();
  },
);

function createDefaultFilters(): NotificationFilterState {
  return {
    category: '',
    occurredRange: [],
    severity: '',
    sourceModule: '',
    status: 'all',
  };
}

function hydrateFromRoute() {
  const status = parseStatus(route.query.status);
  filters.value = {
    ...filters.value,
    status,
    occurredRange: normalizeRouteRangeForPageState([
      firstRouteQueryValue(route.query.occurred_from),
      firstRouteQueryValue(route.query.occurred_to),
    ]),
  };
}

function firstRouteQueryValue(value: unknown) {
  return Array.isArray(value) ? value[0] : value;
}

function parseStatus(value: unknown): NotificationStatusFilter {
  const raw = firstRouteQueryValue(value);
  return raw === 'unread' || raw === 'read' ? raw : 'all';
}

function buildQuery(): NotificationListQuery {
  const query: NotificationListQuery = {
    page: pagination.value.current,
    page_size: pagination.value.pageSize,
  };

  if (filters.value.status !== 'all') query.status = filters.value.status;
  if (filters.value.severity) query.severity = filters.value.severity;
  if (filters.value.category) query.category = filters.value.category;
  if (filters.value.sourceModule) query.source_module = filters.value.sourceModule;

  const [occurredFrom, occurredTo] = filters.value.occurredRange;
  if (occurredFrom) query.occurred_from = localDateTimeToUtcIso(occurredFrom);
  if (occurredTo) query.occurred_to = localDateTimeToUtcIso(occurredTo);

  return query;
}

async function fetchNotifications() {
  loading.value = true;
  listError.value = '';
  try {
    const response = await getNotifications(buildQuery());
    rows.value = response.items;
    total.value = response.total;
    pagination.value.current = response.page;
    pagination.value.pageSize = response.page_size;
    openRouteDelivery();
  } catch (error) {
    logger.error('failed to fetch notifications', error);
    rows.value = [];
    total.value = 0;
    listError.value = resolveLocalizedErrorMessage(t, error, t('notification.messages.loadFailed'));
  } finally {
    loading.value = false;
  }
}

function openRouteDelivery() {
  const deliveryId = Number(firstRouteQueryValue(route.query.delivery_id));
  if (!Number.isFinite(deliveryId)) return;
  const row = rows.value.find((item) => item.delivery_id === deliveryId);
  if (row) {
    openDetail(row);
  }
}

function syncRouteQuery() {
  const [occurredFrom, occurredTo] = normalizePageStateRangeForRoute(filters.value.occurredRange);
  void router.replace({
    query: {
      ...route.query,
      status: filters.value.status === 'all' ? undefined : filters.value.status,
      occurred_from: occurredFrom || undefined,
      occurred_to: occurredTo || undefined,
      delivery_id: undefined,
    },
  });
}

function handleStatusChange(value: TabValue) {
  filters.value.status = parseStatus(value);
  pagination.value.current = 1;
  syncRouteQuery();
  void fetchNotifications();
}

function handleSearch() {
  pagination.value.current = 1;
  syncRouteQuery();
  void fetchNotifications();
}

function resetFilters() {
  filters.value = createDefaultFilters();
  pagination.value.current = 1;
  syncRouteQuery();
  void fetchNotifications();
}

function handlePageChange(page: { current: number; pageSize: number }) {
  pagination.value = page;
  void fetchNotifications();
}

function openDetail(row: NotificationItem) {
  detailRecord.value = row;
  detailVisible.value = true;
}

async function markOneRead(row: NotificationItem) {
  try {
    const updated = await markNotificationRead(row.delivery_id);
    rows.value = rows.value.map((item) => (item.delivery_id === updated.delivery_id ? updated : item));
    if (detailRecord.value?.delivery_id === updated.delivery_id) {
      detailRecord.value = updated;
    }
    MessagePlugin.success(t('notification.messages.markReadSuccess'));
  } catch (error) {
    MessagePlugin.error(resolveLocalizedErrorMessage(t, error, t('notification.messages.markReadFailed')));
  }
}

async function markAllRead() {
  markingAll.value = true;
  try {
    await markNotificationsReadAll({
      ...(filters.value.severity ? { severity: filters.value.severity } : {}),
      ...(filters.value.category ? { category: filters.value.category } : {}),
      ...(filters.value.sourceModule ? { source_module: filters.value.sourceModule } : {}),
      ...(buildQuery().occurred_from ? { occurred_from: buildQuery().occurred_from } : {}),
      ...(buildQuery().occurred_to ? { occurred_to: buildQuery().occurred_to } : {}),
    });
    MessagePlugin.success(t('notification.messages.markAllReadSuccess'));
    await fetchNotifications();
  } catch (error) {
    MessagePlugin.error(resolveLocalizedErrorMessage(t, error, t('notification.messages.markAllReadFailed')));
  } finally {
    markingAll.value = false;
  }
}

async function deleteRow(row: NotificationItem) {
  try {
    await deleteNotification(row.delivery_id);
    rows.value = rows.value.filter((item) => item.delivery_id !== row.delivery_id);
    total.value = Math.max(0, total.value - 1);
    if (detailRecord.value?.delivery_id === row.delivery_id) {
      detailVisible.value = false;
      detailRecord.value = null;
    }
    MessagePlugin.success(t('notification.messages.deleteSuccess'));
  } catch (error) {
    MessagePlugin.error(resolveLocalizedErrorMessage(t, error, t('notification.messages.deleteFailed')));
  }
}

function navigateToTarget(row: NotificationItem) {
  const location = resolveNotificationNavigationLocation(row.navigation);
  if (!location) {
    MessagePlugin.warning(t('notification.messages.navigationUnavailable'));
    return;
  }

  void router.push(location);
}
</script>
<style scoped lang="less">
.notification-filter-stack {
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-12);
}
</style>
