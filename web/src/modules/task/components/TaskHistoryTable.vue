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
    <t-table
      row-key="id"
      size="small"
      :columns="columns"
      :data="items"
      :loading="loading"
      :empty="t('task.history.empty')"
      @row-click="({ row }) => $emit('open', row as TaskSummary)"
    >
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
    <p v-if="errorMessage" class="task-history__error">{{ errorMessage }}</p>
  </section>
</template>
<script setup lang="ts">
import type { TableProps } from 'tdesign-vue-next';
import { computed, ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';

import { resolveLocalizedErrorMessage } from '@/shared/localized-api-error';
import { formatLocaleDateTime } from '@/shared/observability';

import { getTasks } from '../api/task';
import { taskStatusTheme } from '../shared/presentation';
import type { TaskStatus, TaskSummary } from '../types/task';

const props = defineProps<{
  ownerId: string;
  ownerType: string;
}>();

defineEmits<{
  (event: 'open', task: TaskSummary): void;
}>();

const { locale, t } = useI18n();
const items = ref<TaskSummary[]>([]);
const loading = ref(false);
const errorMessage = ref('');

const columns = computed<NonNullable<TableProps['columns']>>(() => [
  { colKey: 'type', title: t('task.history.columns.type'), ellipsis: true },
  { colKey: 'status', title: t('task.history.columns.status'), cell: 'status', width: 132 },
  { colKey: 'current_stage_key', title: t('task.history.columns.stage'), ellipsis: true },
  { colKey: 'created_at', title: t('task.history.columns.createdAt'), cell: 'created_at', width: 188 },
  { colKey: 'operation', title: t('task.history.columns.operation'), cell: 'operation', width: 92 },
]);

function taskStatusLabel(status: TaskStatus) {
  return t(`task.status.${status}`);
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
  gap: var(--graft-density-gap-16);
  justify-content: space-between;
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
</style>
