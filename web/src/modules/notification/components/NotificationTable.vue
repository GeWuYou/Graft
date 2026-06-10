<!--
  Copyright (c) 2025-2026 GeWuYou
  SPDX-License-Identifier: Apache-2.0
-->

<template>
  <t-card class="notification-table-card" :bordered="true">
    <template #header>
      <div class="notification-table-card__head">
        <div>
          <h2>{{ t('notification.table.title') }}</h2>
          <p>{{ t('notification.table.summary', { count: total }) }}</p>
        </div>
      </div>
    </template>

    <t-table
      row-key="delivery_id"
      :columns="columns"
      :data="items"
      :loading="loading"
      :pagination="paginationConfig"
      table-layout="fixed"
      :table-content-width="tableContentWidth"
      cell-empty-content="-"
      hover
      @page-change="handlePageChange"
    >
      <template #notification="{ row }">
        <div
          class="notification-title-cell"
          :class="{ 'notification-title-cell--unread': notificationRow(row).status === 'unread' }"
        >
          <strong>{{ notificationTitle(notificationRow(row), t) }}</strong>
          <span>{{ notificationMessage(notificationRow(row), t) }}</span>
        </div>
      </template>

      <template #severity="{ row }">
        <t-tag :theme="notificationSeverityTheme(notificationRow(row).severity)" variant="light-outline" size="small">
          {{ t(`notification.severity.${notificationRow(row).severity}`) }}
        </t-tag>
      </template>

      <template #category="{ row }">
        <t-tag variant="light" size="small">
          {{ t(`notification.category.${notificationRow(row).category}`) }}
        </t-tag>
      </template>

      <template #source_module="{ row }">
        {{ notificationSourceLabel(notificationRow(row).source_module, t) }}
      </template>

      <template #status="{ row }">
        <t-tag :theme="notificationStatusTheme(notificationRow(row).status)" variant="light" size="small">
          {{ t(`notification.status.${notificationRow(row).status}`) }}
        </t-tag>
      </template>

      <template #occurred_at="{ row }">
        {{ formatCompactDateTime(notificationRow(row).occurred_at, locale) }}
      </template>

      <template #operation="{ row }">
        <t-space size="small">
          <t-button size="small" theme="primary" variant="text" @click="$emit('detail', notificationRow(row))">
            {{ t('notification.actions.detail') }}
          </t-button>
          <t-button
            v-if="notificationRow(row).status === 'unread'"
            size="small"
            theme="default"
            variant="outline"
            @click="$emit('mark-read', notificationRow(row))"
          >
            {{ t('notification.actions.markRead') }}
          </t-button>
          <t-button size="small" theme="danger" variant="text" @click="$emit('delete', notificationRow(row))">
            {{ t('notification.actions.delete') }}
          </t-button>
        </t-space>
      </template>

      <template #empty>
        <t-empty :title="t('notification.empty.title')" :description="t('notification.empty.description')" />
      </template>
    </t-table>
  </t-card>
</template>
<script setup lang="ts">
import type { PageInfo, TdBaseTableProps } from 'tdesign-vue-next';
import { computed } from 'vue';
import { useI18n } from 'vue-i18n';

import {
  calculateTableContentWidth,
  createActionColumn,
  createConfiguredColumns,
  formatCompactDateTime,
} from '@/shared/components/management';

import {
  notificationMessage,
  notificationSeverityTheme,
  notificationSourceLabel,
  notificationStatusTheme,
  notificationTitle,
} from '../shared/presentation';
import type { NotificationItem } from '../types/notification';

const props = defineProps<{
  current: number;
  items: NotificationItem[];
  loading?: boolean;
  pageSize: number;
  total: number;
}>();

const emit = defineEmits<{
  (e: 'delete', row: NotificationItem): void;
  (e: 'detail', row: NotificationItem): void;
  (e: 'mark-read', row: NotificationItem): void;
  (e: 'page-change', page: { current: number; pageSize: number }): void;
}>();

const { t, locale } = useI18n();

const columns = computed<TdBaseTableProps['columns']>(() => [
  ...createConfiguredColumns([
    { key: 'notification', title: t('notification.columns.notification'), config: { minWidth: 360 } },
    {
      key: 'severity',
      title: t('notification.columns.severity'),
      config: { width: 116, align: 'center', ellipsis: false },
    },
    {
      key: 'category',
      title: t('notification.columns.category'),
      config: { width: 132, align: 'center', ellipsis: false },
    },
    { key: 'source_module', title: t('notification.columns.sourceModule'), config: { width: 148 } },
    {
      key: 'status',
      title: t('notification.columns.status'),
      config: { width: 112, align: 'center', ellipsis: false },
    },
    { kind: 'time', key: 'occurred_at', title: t('notification.columns.occurredAt'), width: 184 },
  ]),
  createActionColumn(t('notification.columns.actions'), 224),
]);

const tableContentWidth = computed(() => calculateTableContentWidth(columns.value));
const paginationConfig = computed(() => ({
  current: props.current,
  pageSize: props.pageSize,
  pageSizeOptions: [10, 20, 50],
  showPageNumber: true,
  total: props.total,
}));

function notificationRow(row: unknown) {
  return row as NotificationItem;
}

function handlePageChange(pageInfo: PageInfo) {
  emit('page-change', {
    current: pageInfo.current,
    pageSize: pageInfo.pageSize,
  });
}
</script>
<style scoped lang="less">
.notification-table-card__head {
  align-items: center;
  display: flex;
  justify-content: space-between;
}

.notification-table-card__head h2,
.notification-table-card__head p {
  margin: 0;
}

.notification-table-card__head h2 {
  color: var(--td-text-color-primary);
  font: var(--td-font-title-medium);
}

.notification-table-card__head p {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-medium);
  margin-top: var(--graft-density-gap-4);
}

.notification-title-cell {
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-4);
  min-width: 0;
}

.notification-title-cell strong {
  color: var(--td-text-color-primary);
  font-weight: 600;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.notification-title-cell span {
  color: var(--td-text-color-secondary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.notification-title-cell--unread strong {
  color: var(--td-brand-color);
}
</style>
