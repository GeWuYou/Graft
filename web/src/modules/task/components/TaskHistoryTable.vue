<template>
  <section class="task-history" data-testid="task-history">
    <div class="task-history__heading">
      <div>
        <h3>{{ t('task.history.title') }}</h3>
        <p>{{ t('task.history.description') }}</p>
      </div>
      <t-button size="small" theme="default" variant="outline" :loading="loading" @click="load">
        {{ t('task.actions.refresh') }}
      </t-button>
    </div>
    <responsive-table presentation="entity">
      <template #cards>
        <t-loading :loading="loading">
          <responsive-card-list v-if="items.length">
            <article v-for="item in items" :key="item.id" class="task-history__card">
              <header class="task-history__card-header">
                <strong class="task-history__card-type">{{ taskTypeLabel(item.type) }}</strong>
              </header>
              <dl class="task-history__card-details">
                <div>
                  <dt>{{ t('task.history.columns.status') }}</dt>
                  <dd>
                    <t-tag :theme="taskStatusTheme(item.status)" size="small" variant="light-outline">
                      {{ taskStatusLabel(item.status) }}
                    </t-tag>
                  </dd>
                </div>
                <div>
                  <dt>{{ t('task.history.columns.stage') }}</dt>
                  <dd>{{ item.current_stage_key }}</dd>
                </div>
                <div>
                  <dt>{{ t('task.history.columns.createdAt') }}</dt>
                  <dd>{{ formatLocaleDateTime(item.created_at, locale) }}</dd>
                </div>
              </dl>
              <footer class="task-history__card-actions">
                <t-button size="small" theme="primary" variant="text" @click="$emit('open', item)">
                  {{ t('task.actions.view') }}
                </t-button>
              </footer>
            </article>
          </responsive-card-list>
          <t-empty v-else-if="!loading" :title="t('task.history.empty')" />
        </t-loading>
      </template>
      <t-table
        row-key="id"
        size="small"
        :columns="columns"
        :data="items"
        :loading="loading"
        :empty="t('task.history.empty')"
        @row-click="({ row }) => $emit('open', row as TaskSummary)"
      >
        <template #type="{ row }">
          {{ taskTypeLabel((row as TaskSummary).type) }}
        </template>
        <template #status="{ row }">
          <t-tag :theme="taskStatusTheme(row.status)" size="small" variant="light-outline">
            {{ taskStatusLabel(row.status) }}
          </t-tag>
        </template>
        <template #created_at="{ row }">
          {{ formatLocaleDateTime(row.created_at, locale) }}
        </template>
        <template #operation="{ row }">
          <t-button size="small" theme="primary" variant="text" @click.stop="$emit('open', row as TaskSummary)">
            {{ t('task.actions.view') }}
          </t-button>
        </template>
      </t-table>
    </responsive-table>
    <p v-if="errorMessage" class="task-history__error">{{ errorMessage }}</p>
  </section>
</template>
<script setup lang="ts">
// 历史表格只展示当前 owner 隔离范围内的任务记录；实时观察、取消和重试仍由详情抽屉交回任务边界。
import type { TableProps } from 'tdesign-vue-next';
import { computed, ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';

import ResponsiveCardList from '@/shared/components/responsive/ResponsiveCardList.vue';
import ResponsiveTable from '@/shared/components/responsive/ResponsiveTable.vue';
import { resolveLocalizedErrorMessage } from '@/shared/localized-api-error';
import { formatLocaleDateTime } from '@/shared/observability';

import { getTasks } from '../api/task';
import { taskStatusTheme } from '../shared/presentation';
import type { TaskStatus, TaskSummary } from '../types/task';

const props = defineProps<{
  ownerId: string;
  ownerType: string;
  resolveTaskType?: (taskType: string) => string | undefined;
}>();

defineEmits<{
  (event: 'open', task: TaskSummary): void;
}>();

const { locale, t } = useI18n();
const items = ref<TaskSummary[]>([]);
const loading = ref(false);
const errorMessage = ref('');

const columns = computed<NonNullable<TableProps['columns']>>(() => [
  { colKey: 'type', title: t('task.history.columns.type'), cell: 'type', ellipsis: true },
  { colKey: 'status', title: t('task.history.columns.status'), cell: 'status', width: 132 },
  { colKey: 'current_stage_key', title: t('task.history.columns.stage'), ellipsis: true },
  { colKey: 'created_at', title: t('task.history.columns.createdAt'), cell: 'created_at', width: 188 },
  { colKey: 'operation', title: t('task.history.columns.operation'), cell: 'operation', width: 92 },
]);

function taskStatusLabel(status: TaskStatus) {
  return t(`task.status.${status}`);
}

function taskTypeLabel(taskType: string) {
  return props.resolveTaskType?.(taskType) ?? taskType;
}

async function load() {
  if (!props.ownerId || !props.ownerType) return;
  loading.value = true;
  errorMessage.value = '';
  try {
    const response = await getTasks({ limit: 20, owner_id: props.ownerId, owner_type: props.ownerType });
    items.value = response.items.filter(
      (item) => item.owner_id === props.ownerId && item.owner_type === props.ownerType,
    );
  } catch (error) {
    errorMessage.value = resolveLocalizedErrorMessage(t, error, t('task.history.loadFailed'));
  } finally {
    loading.value = false;
  }
}

watch(() => [props.ownerId, props.ownerType], load, { immediate: true });
</script>
<style scoped lang="less">
.task-history {
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-16);
}

.task-history__heading {
  align-items: flex-start;
  display: flex;
  flex-wrap: wrap;
  gap: var(--graft-density-gap-16);
  justify-content: space-between;
}

.task-history__heading > div {
  min-width: 0;
}

.task-history__heading h3,
.task-history__heading p {
  margin: 0;
}

.task-history__heading p,
.task-history__error {
  color: var(--td-text-color-secondary);
  margin-top: var(--graft-density-gap-4);
}

.task-history__error {
  color: var(--td-error-color);
  margin-bottom: 0;
}

.task-history__card {
  background: var(--td-bg-color-container);
  border: 1px solid var(--td-component-stroke);
  border-radius: var(--td-radius-medium);
  display: grid;
  gap: var(--graft-density-gap-16);
  min-width: 0;
  padding: var(--graft-density-gap-16);
}

.task-history__card-header,
.task-history__card-details,
.task-history__card-details > div {
  display: grid;
  min-width: 0;
}

.task-history__card-type {
  color: var(--td-text-color-primary);
  font: var(--td-font-title-small);
  overflow-wrap: anywhere;
}

.task-history__card-details {
  gap: var(--graft-density-gap-12);
  grid-template-columns: repeat(2, minmax(0, 1fr));
  margin: 0;
}

.task-history__card-details > div {
  gap: var(--graft-density-gap-4);
}

.task-history__card-details > div:last-child {
  grid-column: 1 / -1;
}

.task-history__card-details dt {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
}

.task-history__card-details dd {
  color: var(--td-text-color-primary);
  margin: 0;
  overflow-wrap: anywhere;
}

.task-history__card-actions {
  border-top: 1px solid var(--td-component-stroke);
  display: flex;
  justify-content: flex-end;
  margin-top: calc(-1 * var(--graft-density-gap-4));
  padding-top: var(--graft-density-gap-12);
}
</style>
