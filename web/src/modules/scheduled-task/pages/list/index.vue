<template>
  <div class="scheduled-task-page" data-page-type="list-form-detail">
    <section class="scheduled-task-page__header">
      <div class="scheduled-task-page__title-block">
        <span class="scheduled-task-page__eyebrow">{{ t('scheduledTask.list.eyebrow') }}</span>
        <h1>{{ t('scheduledTask.list.title') }}</h1>
        <p>{{ t('scheduledTask.list.description') }}</p>
      </div>
      <t-button theme="default" variant="outline" :loading="loading" @click="refreshTasks">
        <template #icon><refresh-icon /></template>
        {{ t('scheduledTask.list.refresh') }}
      </t-button>
    </section>

    <section class="scheduled-task-metrics" aria-label="scheduled task metrics">
      <t-card
        v-for="metric in overviewMetrics"
        :key="metric.key"
        class="scheduled-task-metric-card"
        size="small"
        :bordered="true"
      >
        <t-statistic :title="metric.label" :value="metric.value" />
        <p>{{ metric.description }}</p>
      </t-card>
    </section>

    <t-card class="scheduled-task-table-card" :bordered="true">
      <template #header>
        <div class="scheduled-task-table-head">
          <div>
            <h2>{{ t('scheduledTask.list.tableTitle') }}</h2>
            <p>{{ t('scheduledTask.list.tableHint', { count: tasks.length }) }}</p>
          </div>
        </div>
      </template>

      <div v-if="errorMessage && !loading" class="scheduled-task-feedback scheduled-task-feedback--error">
        <span>{{ errorMessage }}</span>
        <t-button theme="primary" variant="outline" size="small" @click="refreshTasks">
          {{ t('scheduledTask.list.refresh') }}
        </t-button>
      </div>

      <t-table
        row-key="key"
        :data="tasks"
        :columns="columns"
        :loading="loading"
        table-layout="fixed"
        table-content-width="1060"
        cell-empty-content="-"
        hover
      >
        <template #task="{ row }">
          <div class="scheduled-task-identity">
            <span class="scheduled-task-identity__name">{{ taskDisplayName(row) }}</span>
            <span class="scheduled-task-identity__key">{{ row.key }}</span>
          </div>
        </template>

        <template #status="{ row }">
          <task-status-tag :status="row.status" />
        </template>

        <template #owner="{ row }">
          <div class="scheduled-task-owner">
            <span>{{ row.owner }}</span>
            <span>{{ row.module }}</span>
          </div>
        </template>

        <template #schedule="{ row }">
          <div class="scheduled-task-schedule">
            <t-tag variant="light-outline" theme="primary">{{ scheduleTypeLabel(row.schedule_type) }}</t-tag>
            <span>{{ row.schedule }}</span>
          </div>
        </template>

        <template #last_run="{ row }">
          <div v-if="row.last_run" class="scheduled-task-last-run">
            <task-status-tag :status="row.last_run.status" />
            <span>{{ formatTimestamp(row.last_run.started_at) }}</span>
          </div>
          <span v-else class="scheduled-task-muted">{{ t('scheduledTask.list.detail.none') }}</span>
        </template>

        <template #next_run_at="{ row }">
          <span>{{ formatTimestamp(row.next_run_at) }}</span>
        </template>

        <template #operation="{ row }">
          <t-space class="scheduled-task-actions" size="small" align="center">
            <t-button theme="primary" variant="text" size="small" @click="openDetail(row)">
              {{ t('scheduledTask.list.viewDetail') }}
            </t-button>
            <t-popconfirm
              :content="t('scheduledTask.list.runConfirm')"
              :confirm-btn="t('scheduledTask.list.runConfirmButton')"
              :cancel-btn="t('scheduledTask.list.runCancelButton')"
              @confirm="() => runTask(row)"
            >
              <t-button
                v-permission="permissionCodes.RUN"
                theme="primary"
                variant="outline"
                size="small"
                :disabled="!canRunTask(row)"
                :loading="runningTaskKey === row.key"
              >
                <template #icon><play-icon /></template>
                {{ t('scheduledTask.list.run') }}
              </t-button>
            </t-popconfirm>
          </t-space>
        </template>

        <template #empty>
          <div class="scheduled-task-empty">
            <t-empty
              :title="t('scheduledTask.list.emptyTitle')"
              :description="t('scheduledTask.list.emptyDescription')"
            >
              <template #action>
                <t-button theme="primary" variant="outline" @click="refreshTasks">
                  {{ t('scheduledTask.list.refresh') }}
                </t-button>
              </template>
            </t-empty>
          </div>
        </template>
      </t-table>
    </t-card>

    <t-drawer
      v-model:visible="detailVisible"
      :header="detailTitle"
      size="760px"
      placement="right"
      destroy-on-close
      :footer="false"
    >
      <div v-if="selectedTask" class="scheduled-task-detail">
        <section class="scheduled-task-detail__section">
          <div class="scheduled-task-detail__section-head">
            <h3>{{ t('scheduledTask.list.detail.basics') }}</h3>
            <task-status-tag :status="selectedTask.status" />
          </div>
          <t-descriptions :column="1" bordered size="small">
            <t-descriptions-item :label="t('scheduledTask.list.detail.key')">
              {{ selectedTask.key }}
            </t-descriptions-item>
            <t-descriptions-item :label="t('scheduledTask.list.detail.displayNameKey')">
              {{ selectedTask.display_name_key }}
            </t-descriptions-item>
            <t-descriptions-item :label="t('scheduledTask.list.detail.descriptionKey')">
              {{ selectedTask.description_key }}
            </t-descriptions-item>
            <t-descriptions-item :label="t('scheduledTask.list.detail.owner')">
              {{ selectedTask.owner }}
            </t-descriptions-item>
            <t-descriptions-item :label="t('scheduledTask.list.detail.module')">
              {{ selectedTask.module }}
            </t-descriptions-item>
            <t-descriptions-item :label="t('scheduledTask.list.detail.enabled')">
              {{
                selectedTask.enabled
                  ? t('scheduledTask.list.detail.enabledYes')
                  : t('scheduledTask.list.detail.enabledNo')
              }}
            </t-descriptions-item>
          </t-descriptions>
        </section>

        <section class="scheduled-task-detail__section">
          <h3>{{ t('scheduledTask.list.detail.schedule') }}</h3>
          <t-descriptions :column="1" bordered size="small">
            <t-descriptions-item :label="t('scheduledTask.list.detail.taskType')">
              {{ taskTypeLabel(selectedTask.task_type) }}
            </t-descriptions-item>
            <t-descriptions-item :label="t('scheduledTask.list.detail.scheduleType')">
              {{ scheduleTypeLabel(selectedTask.schedule_type) }}
            </t-descriptions-item>
            <t-descriptions-item :label="t('scheduledTask.list.detail.scheduleRule')">
              {{ selectedTask.schedule }}
            </t-descriptions-item>
            <t-descriptions-item :label="t('scheduledTask.list.detail.nextRun')">
              {{ formatTimestamp(selectedTask.next_run_at) }}
            </t-descriptions-item>
            <t-descriptions-item :label="t('scheduledTask.list.detail.running')">
              {{
                selectedTask.running
                  ? t('scheduledTask.list.detail.runningYes')
                  : t('scheduledTask.list.detail.runningNo')
              }}
            </t-descriptions-item>
          </t-descriptions>
        </section>

        <section class="scheduled-task-detail__section">
          <h3>{{ t('scheduledTask.list.detail.latestResult') }}</h3>
          <t-descriptions v-if="selectedTask.last_run" :column="1" bordered size="small">
            <t-descriptions-item :label="t('scheduledTask.list.detail.runId')">
              {{ selectedTask.last_run.id }}
            </t-descriptions-item>
            <t-descriptions-item :label="t('scheduledTask.list.detail.triggerType')">
              {{ triggerLabel(selectedTask.last_run.trigger_type) }}
            </t-descriptions-item>
            <t-descriptions-item :label="t('scheduledTask.list.detail.status')">
              <task-status-tag :status="selectedTask.last_run.status" />
            </t-descriptions-item>
            <t-descriptions-item :label="t('scheduledTask.list.detail.startedAt')">
              {{ formatTimestamp(selectedTask.last_run.started_at) }}
            </t-descriptions-item>
            <t-descriptions-item :label="t('scheduledTask.list.detail.finishedAt')">
              {{ formatTimestamp(selectedTask.last_run.finished_at) }}
            </t-descriptions-item>
            <t-descriptions-item :label="t('scheduledTask.list.detail.duration')">
              {{ formatDuration(selectedTask.last_run.duration_ms) }}
            </t-descriptions-item>
            <t-descriptions-item :label="t('scheduledTask.list.detail.errorSummary')">
              {{ selectedTask.last_run.error_summary || t('scheduledTask.list.detail.none') }}
            </t-descriptions-item>
          </t-descriptions>
          <p v-else class="scheduled-task-muted">{{ t('scheduledTask.list.detail.none') }}</p>
        </section>

        <section class="scheduled-task-detail__section">
          <div class="scheduled-task-detail__section-head">
            <h3>{{ t('scheduledTask.list.detail.recentRuns') }}</h3>
            <t-button size="small" theme="default" variant="outline" :loading="runsLoading" @click="refreshRuns">
              {{ t('scheduledTask.list.diagnose') }}
            </t-button>
          </div>
          <t-table
            row-key="id"
            size="small"
            :data="recentRuns"
            :columns="runColumns"
            :loading="runsLoading"
            table-layout="fixed"
            cell-empty-content="-"
          >
            <template #status="{ row }">
              <task-status-tag :status="row.status" />
            </template>
            <template #trigger_type="{ row }">
              {{ triggerLabel(row.trigger_type) }}
            </template>
            <template #started_at="{ row }">
              {{ formatTimestamp(row.started_at) }}
            </template>
            <template #duration_ms="{ row }">
              {{ formatDuration(row.duration_ms) }}
            </template>
            <template #empty>
              <div class="scheduled-task-runs-empty">
                {{ t('scheduledTask.list.detail.runsEmpty') }}
              </div>
            </template>
          </t-table>
        </section>
      </div>
    </t-drawer>
  </div>
</template>
<script setup lang="ts">
import { PlayIcon, RefreshIcon } from 'tdesign-icons-vue-next';
import { MessagePlugin, Tag, type TdBaseTableProps } from 'tdesign-vue-next';
import { computed, defineComponent, h, onMounted, ref } from 'vue';
import { useI18n } from 'vue-i18n';

import { createLogger } from '@/utils/logger';

import { getScheduledTask, getScheduledTaskRuns, getScheduledTasks, runScheduledTask } from '../../api/scheduled-task';
import { SCHEDULED_TASK_PERMISSION_CODE } from '../../contract/permissions';
import type {
  ScheduledTaskItem,
  ScheduledTaskRunItem,
  ScheduledTaskRunStatus,
  ScheduledTaskRunTriggerType,
  ScheduledTaskStatus,
} from '../../types/scheduled-task';

defineOptions({
  name: 'ScheduledTaskListPage',
});

const TaskStatusTag = defineComponent({
  name: 'ScheduledTaskStatusTag',
  props: {
    status: {
      type: String,
      required: true,
    },
  },
  setup(props) {
    const { t } = useI18n();

    return () =>
      h(
        Tag,
        {
          theme: statusTheme(props.status as ScheduledTaskStatus | ScheduledTaskRunStatus),
          variant: 'light',
          class: 'scheduled-task-status-tag',
        },
        () => t(`scheduledTask.list.status.${props.status}`),
      );
  },
});

const { t, locale } = useI18n();
const logger = createLogger('scheduled-task.list.page');
const permissionCodes = SCHEDULED_TASK_PERMISSION_CODE;

const tasks = ref<ScheduledTaskItem[]>([]);
const selectedTask = ref<ScheduledTaskItem | null>(null);
const recentRuns = ref<ScheduledTaskRunItem[]>([]);
const loading = ref(false);
const runsLoading = ref(false);
const detailVisible = ref(false);
const errorMessage = ref('');
const runningTaskKey = ref('');

const overviewMetrics = computed(() => [
  {
    key: 'total',
    label: t('scheduledTask.list.metric.total'),
    value: tasks.value.length,
    description: t('scheduledTask.list.metric.totalDescription'),
  },
  {
    key: 'enabled',
    label: t('scheduledTask.list.metric.enabled'),
    value: tasks.value.filter((task) => task.enabled).length,
    description: t('scheduledTask.list.metric.enabledDescription'),
  },
  {
    key: 'recentFailed',
    label: t('scheduledTask.list.metric.recentFailed'),
    value: tasks.value.filter((task) => task.last_run?.status === 'failed').length,
    description: t('scheduledTask.list.metric.recentFailedDescription'),
  },
  {
    key: 'running',
    label: t('scheduledTask.list.metric.running'),
    value: tasks.value.filter((task) => task.running).length,
    description: t('scheduledTask.list.metric.runningDescription'),
  },
]);

const detailTitle = computed(() =>
  selectedTask.value
    ? t('scheduledTask.list.detail.titleWithName', { name: taskDisplayName(selectedTask.value) })
    : t('scheduledTask.list.detail.title'),
);

const columns = computed<TdBaseTableProps['columns']>(() => [
  {
    colKey: 'task',
    title: t('scheduledTask.list.columns.task'),
    width: 240,
    fixed: 'left',
  },
  {
    colKey: 'status',
    title: t('scheduledTask.list.columns.status'),
    width: 120,
  },
  {
    colKey: 'owner',
    title: t('scheduledTask.list.columns.owner'),
    width: 160,
  },
  {
    colKey: 'schedule',
    title: t('scheduledTask.list.columns.schedule'),
    width: 180,
  },
  {
    colKey: 'last_run',
    title: t('scheduledTask.list.columns.lastRun'),
    width: 190,
  },
  {
    colKey: 'next_run_at',
    title: t('scheduledTask.list.columns.nextRun'),
    width: 170,
  },
  {
    colKey: 'operation',
    title: t('scheduledTask.list.columns.operation'),
    width: 190,
    fixed: 'right',
  },
]);

const runColumns = computed<TdBaseTableProps['columns']>(() => [
  {
    colKey: 'id',
    title: t('scheduledTask.list.detail.runId'),
    width: 90,
  },
  {
    colKey: 'trigger_type',
    title: t('scheduledTask.list.detail.triggerType'),
    width: 110,
  },
  {
    colKey: 'status',
    title: t('scheduledTask.list.detail.status'),
    width: 110,
  },
  {
    colKey: 'started_at',
    title: t('scheduledTask.list.detail.startedAt'),
    width: 180,
  },
  {
    colKey: 'duration_ms',
    title: t('scheduledTask.list.detail.duration'),
    width: 110,
  },
  {
    colKey: 'error_summary',
    title: t('scheduledTask.list.detail.errorSummary'),
    ellipsis: true,
  },
]);

onMounted(() => {
  void refreshTasks();
});

async function refreshTasks() {
  loading.value = true;
  errorMessage.value = '';

  try {
    const response = await getScheduledTasks();
    tasks.value = response.items;
  } catch (error) {
    logger.error(error instanceof Error ? error : 'load scheduled tasks failed', {
      operation: 'scheduled_task_list',
    });
    errorMessage.value = t('scheduledTask.list.loadError');
  } finally {
    loading.value = false;
  }
}

async function openDetail(row: ScheduledTaskItem) {
  errorMessage.value = '';
  selectedTask.value = row;
  detailVisible.value = true;

  try {
    const [detail] = await Promise.all([getScheduledTask(row.key), loadRuns(row.key)]);
    selectedTask.value = detail;
    syncTask(detail);
  } catch (error) {
    logger.error(error instanceof Error ? error : 'load scheduled task detail failed', {
      taskKey: row.key,
      operation: 'scheduled_task_detail',
    });
    void MessagePlugin.error(t('scheduledTask.list.detailLoadError'));
  }
}

async function refreshRuns() {
  if (!selectedTask.value) {
    return;
  }

  try {
    await loadRuns(selectedTask.value.key);
  } catch (error) {
    logger.error(error instanceof Error ? error : 'load scheduled task runs failed', {
      taskKey: selectedTask.value.key,
      operation: 'scheduled_task_runs',
    });
    void MessagePlugin.error(t('scheduledTask.list.detailLoadError'));
  }
}

async function loadRuns(taskKey: string) {
  runsLoading.value = true;
  try {
    const response = await getScheduledTaskRuns(taskKey, { limit: 10, offset: 0 });
    recentRuns.value = response.items;
    return response;
  } finally {
    runsLoading.value = false;
  }
}

async function runTask(task: ScheduledTaskItem) {
  if (!canRunTask(task)) {
    return;
  }

  runningTaskKey.value = task.key;

  try {
    const run = await runScheduledTask(task.key);
    recentRuns.value = [run, ...recentRuns.value.filter((item) => item.id !== run.id)].slice(0, 10);
    const detail = await getScheduledTask(task.key);
    syncTask(detail);
    if (selectedTask.value?.key === detail.key) {
      selectedTask.value = detail;
    }
    void MessagePlugin.success(t('scheduledTask.list.runSuccess'));
  } catch (error) {
    logger.error(error instanceof Error ? error : 'run scheduled task failed', {
      taskKey: task.key,
      operation: 'scheduled_task_run',
    });
    void MessagePlugin.error(t('scheduledTask.list.runError'));
  } finally {
    runningTaskKey.value = '';
  }
}

function syncTask(detail: ScheduledTaskItem) {
  const index = tasks.value.findIndex((task) => task.key === detail.key);
  if (index === -1) {
    tasks.value = [detail, ...tasks.value];
    return;
  }

  tasks.value = tasks.value.map((task) => (task.key === detail.key ? detail : task));
}

function canRunTask(task: ScheduledTaskItem) {
  return task.enabled && !task.running && runningTaskKey.value !== task.key;
}

function taskDisplayName(task: ScheduledTaskItem) {
  return task.display_name_key || task.key;
}

function taskTypeLabel(type: ScheduledTaskItem['task_type']) {
  return t(`scheduledTask.list.taskType.${type}`);
}

function scheduleTypeLabel(type: ScheduledTaskItem['schedule_type']) {
  return t(`scheduledTask.list.scheduleType.${type}`);
}

function triggerLabel(type: ScheduledTaskRunTriggerType) {
  return t(`scheduledTask.list.trigger.${type}`);
}

function statusTheme(status: ScheduledTaskStatus | ScheduledTaskRunStatus) {
  switch (status) {
    case 'success':
      return 'success';
    case 'running':
      return 'primary';
    case 'failed':
      return 'danger';
    case 'idle':
    case 'unknown':
    default:
      return 'default';
  }
}

function formatTimestamp(value?: string | null) {
  if (!value) {
    return t('scheduledTask.list.detail.notAvailable');
  }

  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }

  return new Intl.DateTimeFormat(locale.value, {
    dateStyle: 'medium',
    timeStyle: 'medium',
  }).format(date);
}

function formatDuration(value?: number | null) {
  if (value === undefined || value === null) {
    return t('scheduledTask.list.detail.notAvailable');
  }

  if (value < 1000) {
    return `${value} ms`;
  }

  const seconds = value / 1000;
  if (seconds < 60) {
    return `${seconds.toFixed(seconds >= 10 ? 0 : 1)} s`;
  }

  const minutes = Math.floor(seconds / 60);
  const remainingSeconds = Math.round(seconds % 60);
  return `${minutes} min ${remainingSeconds} s`;
}
</script>
<style scoped lang="less">
.scheduled-task-page {
  box-sizing: border-box;
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-16);
}

.scheduled-task-page__header,
.scheduled-task-table-head,
.scheduled-task-detail__section-head {
  align-items: center;
  display: flex;
  gap: var(--graft-density-gap-12);
  justify-content: space-between;
}

.scheduled-task-page__title-block {
  min-width: 0;
}

.scheduled-task-page__eyebrow {
  color: var(--td-brand-color);
  display: inline-block;
  font: var(--td-font-body-small);
  font-weight: 600;
  margin-bottom: var(--graft-density-gap-4);
}

.scheduled-task-page h1,
.scheduled-task-table-head h2,
.scheduled-task-detail h3 {
  color: var(--td-text-color-primary);
  margin: 0;
}

.scheduled-task-page h1 {
  font: var(--td-font-headline-small);
}

.scheduled-task-table-head h2,
.scheduled-task-detail h3 {
  font: var(--td-font-title-medium);
}

.scheduled-task-page__title-block p,
.scheduled-task-table-head p,
.scheduled-task-metric-card p,
.scheduled-task-muted,
.scheduled-task-identity__key,
.scheduled-task-owner span + span,
.scheduled-task-last-run span {
  color: var(--td-text-color-secondary);
}

.scheduled-task-page__title-block p,
.scheduled-task-table-head p,
.scheduled-task-metric-card p {
  margin: var(--graft-density-gap-4) 0 0;
}

.scheduled-task-metrics {
  display: grid;
  gap: var(--graft-density-gap-12);
  grid-template-columns: repeat(4, minmax(0, 1fr));
}

.scheduled-task-metric-card {
  min-width: 0;
}

.scheduled-task-table-card :deep(.t-card__body) {
  padding-top: 0;
}

.scheduled-task-feedback {
  align-items: center;
  background: color-mix(in srgb, var(--td-error-color-5) 10%, var(--td-bg-color-container));
  border: 1px solid color-mix(in srgb, var(--td-error-color-5) 28%, var(--td-component-stroke));
  border-radius: var(--td-radius-medium);
  color: var(--td-error-color-7);
  display: flex;
  justify-content: space-between;
  margin-bottom: var(--graft-density-gap-12);
  padding: var(--graft-density-gap-10) var(--graft-density-gap-12);
}

.scheduled-task-identity,
.scheduled-task-owner,
.scheduled-task-schedule,
.scheduled-task-last-run {
  display: flex;
  min-width: 0;
}

.scheduled-task-identity,
.scheduled-task-owner {
  flex-direction: column;
}

.scheduled-task-schedule,
.scheduled-task-last-run {
  align-items: center;
  gap: var(--graft-density-gap-8);
}

.scheduled-task-identity__name {
  color: var(--td-text-color-primary);
  font: var(--td-font-title-small);
}

.scheduled-task-identity__key,
.scheduled-task-owner span,
.scheduled-task-schedule span,
.scheduled-task-last-run span {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.scheduled-task-actions {
  justify-content: flex-end;
  width: 100%;
}

.scheduled-task-empty,
.scheduled-task-runs-empty {
  align-items: center;
  color: var(--td-text-color-secondary);
  display: flex;
  justify-content: center;
  min-height: 220px;
  padding: var(--graft-density-gap-24);
}

.scheduled-task-runs-empty {
  min-height: 120px;
}

.scheduled-task-detail {
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-20);
}

.scheduled-task-detail__section {
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-12);
}

:deep(.scheduled-task-status-tag) {
  border-radius: 999px;
  font-weight: 600;
}

:deep(.t-table th),
:deep(.t-table td) {
  white-space: nowrap;
}

:deep(.t-descriptions__label) {
  width: 160px;
}

@media (width <= 1200px) {
  .scheduled-task-metrics {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (width <= 768px) {
  .scheduled-task-page__header,
  .scheduled-task-table-head,
  .scheduled-task-feedback {
    align-items: stretch;
    flex-direction: column;
  }

  .scheduled-task-metrics {
    grid-template-columns: 1fr;
  }
}
</style>
