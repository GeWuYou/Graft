<template>
  <management-table-card
    class="notification-table-card"
    :title="t('notification.table.title')"
    :description="t('notification.table.summary', { count: total })"
  >
    <div ref="tableHostRef" class="notification-table-card__table-host">
      <t-table
        row-key="delivery_id"
        :columns="columns"
        :data="items"
        :loading="loading"
        table-layout="fixed"
        :table-content-width="tableWidthPolicy.tableContentWidth"
        cell-empty-content="-"
        hover
      >
        <template #notification="{ row }">
          <div
            class="notification-title-cell"
            :class="{ 'notification-title-cell--unread': notificationView(row).status === 'unread' }"
          >
            <strong>{{ notificationView(row).title }}</strong>
            <span>{{ notificationView(row).message }}</span>
          </div>
        </template>

        <template #severity="{ row }">
          <t-tag :theme="notificationSeverityTheme(notificationRow(row).severity)" variant="light-outline" size="small">
            {{ notificationView(row).levelLabel }}
          </t-tag>
        </template>

        <template #category="{ row }">
          <t-tag variant="light" size="small">
            {{ notificationView(row).categoryLabel }}
          </t-tag>
        </template>

        <template #source_module="{ row }">
          {{ notificationView(row).sourceLabel }}
        </template>

        <template #status="{ row }">
          <t-tag :theme="notificationStatusTheme(notificationRow(row).status)" variant="light" size="small">
            {{ notificationView(row).statusLabel }}
          </t-tag>
        </template>

        <template #occurred_at="{ row }">
          {{ notificationView(row).occurredAtLabel }}
        </template>

        <template #operation="{ row }">
          <t-space size="small">
            <t-button size="small" theme="primary" variant="text" @click="$emit('detail', notificationRow(row))">
              {{ t('notification.action.detail') }}
            </t-button>
            <t-button size="small" theme="danger" variant="text" @click="$emit('delete', notificationRow(row))">
              {{ t('notification.action.delete') }}
            </t-button>
          </t-space>
        </template>

        <template #empty>
          <t-empty :title="emptyTitle" :description="emptyDescription" />
        </template>
      </t-table>
    </div>

    <template #footer>
      <management-table-pagination :summary="t('notification.table.summary', { count: total })">
        <t-pagination
          :current="current"
          :page-size="pageSize"
          :page-size-options="[10, 20, 50, 100]"
          :show-page-number="true"
          :total="total"
          @change="handlePageChange"
        />
      </management-table-pagination>
    </template>
  </management-table-card>
</template>
<script setup lang="ts">
import type { PageInfo, TdBaseTableProps } from 'tdesign-vue-next';
import { computed } from 'vue';
import { useI18n } from 'vue-i18n';

import {
  createActionColumn,
  createConfiguredColumns,
  ManagementTableCard,
  ManagementTablePagination,
  resolveTableWidthPolicy,
  useTableHostWidth,
} from '@/shared/components/management';

import { notificationSeverityTheme, notificationStatusTheme, presentNotification } from '../shared/presentation';
import type { NotificationItem } from '../types/notification';

defineProps<{
  current: number;
  emptyDescription: string;
  emptyTitle: string;
  items: NotificationItem[];
  loading?: boolean;
  pageSize: number;
  total: number;
}>();

const emit = defineEmits<{
  (e: 'delete', row: NotificationItem): void;
  (e: 'detail', row: NotificationItem): void;
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
  createActionColumn(t('notification.columns.actions'), 160),
]);

const { tableHostRef, tableHostWidth } = useTableHostWidth(() => columns.value);
const tableWidthPolicy = computed(() => resolveTableWidthPolicy(columns.value, tableHostWidth.value));

function notificationRow(row: unknown) {
  return row as NotificationItem;
}

function notificationView(row: unknown) {
  return presentNotification(notificationRow(row), t, locale.value);
}

function handlePageChange(pageInfo: PageInfo) {
  emit('page-change', {
    current: pageInfo.current,
    pageSize: pageInfo.pageSize,
  });
}
</script>
<style scoped lang="less">
.notification-table-card__table-host {
  max-width: 100%;
  min-width: 0;
  overflow-x: hidden;
  width: 100%;
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
