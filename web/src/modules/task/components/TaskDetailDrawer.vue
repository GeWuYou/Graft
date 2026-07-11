<template>
  <t-drawer
    :visible="visible"
    :header="t('task.detail.title')"
    :footer="false"
    destroy-on-close
    placement="right"
    size="820px"
    @update:visible="$emit('update:visible', $event)"
  >
    <t-loading :loading="loading && !task">
      <div v-if="task" class="task-detail" data-testid="task-detail-drawer">
        <div class="task-detail__summary">
          <div>
            <h3>{{ taskTypeLabel(task.type) }}</h3>
            <p>{{ t('task.detail.identifier', { id: task.id }) }}</p>
          </div>
          <t-tag :theme="taskStatusTheme(task.status)" variant="light-outline">{{
            taskStatusLabel(task.status)
          }}</t-tag>
        </div>

        <t-alert
          v-if="task.failure_message"
          theme="error"
          :title="t('task.detail.failure')"
          :message="task.failure_message"
        />

        <t-descriptions bordered :column="2" size="small" :title="t('task.detail.summary')">
          <t-descriptions-item :label="t('task.detail.currentStage')">{{
            task.current_stage_key || '-'
          }}</t-descriptions-item>
          <t-descriptions-item :label="t('task.detail.duration')">{{ durationLabel }}</t-descriptions-item>
          <t-descriptions-item :label="t('task.detail.createdAt')">{{
            formatTime(task.created_at)
          }}</t-descriptions-item>
          <t-descriptions-item :label="t('task.detail.connection')">{{ connectionLabel }}</t-descriptions-item>
        </t-descriptions>

        <section>
          <h4>{{ t('task.detail.stages') }}</h4>
          <t-steps layout="vertical" readonly :options="stageOptions" />
          <t-space v-if="retryableStages.length" size="small" break-line>
            <t-button
              v-for="stage in retryableStages"
              :key="stage.id"
              size="small"
              theme="default"
              variant="outline"
              :loading="retryingStageId === stage.id"
              @click="retry(stage.id)"
            >
              {{ t('task.actions.retry') }}: {{ stage.key }}
            </t-button>
          </t-space>
        </section>

        <section>
          <div class="task-detail__section-heading">
            <h4>{{ t('task.detail.logs') }}</h4>
            <t-button
              v-if="task.capabilities.download_log"
              size="small"
              theme="default"
              variant="text"
              @click="downloadLogs"
            >
              {{ t('task.actions.downloadLog') }}
            </t-button>
          </div>
          <log-viewer
            v-bind="logViewerBindings"
            :entries="structuredLogs"
            :content-version="projectLogContentVersion"
            :loading="logsLoading && !hasLoadedLogs"
            :error="logsError"
            :truncated="logsTruncated"
            @refresh="reload"
          />
        </section>

        <div class="task-detail__actions">
          <t-button
            v-if="task.capabilities.cancel"
            theme="warning"
            variant="outline"
            :loading="cancelling"
            @click="cancel"
          >
            {{ t('task.actions.cancel') }}
          </t-button>
        </div>
      </div>
      <t-alert v-else-if="errorMessage" theme="error" :message="errorMessage" />
    </t-loading>
  </t-drawer>
</template>
<script setup lang="ts">
import type { StepItemProps } from 'tdesign-vue-next';
import { computed, onUnmounted, ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';

import { resolveLocalizedErrorMessage } from '@/shared/localized-api-error';
import { formatLocaleDateTime, LogViewer, type StructuredLogEntry } from '@/shared/observability';
import { openRealtimeTopicSocket, type RealtimeTopicSocketController } from '@/shared/realtime';

import { cancelTask, getTask, getTaskLogs, retryTaskStage } from '../api/task';
import { buildTaskRealtimeTopicName, parseTaskRealtimeNotification } from '../contract/realtime';
import { taskStatusTheme } from '../shared/presentation';
import { TaskRealtimeRefreshScheduler } from '../shared/realtime-refresh-scheduler';
import { TaskLogRealtimeBatcher } from '../shared/task-log-realtime-batcher';
import { type TaskDetail, type TaskStage, type TaskStageStatus } from '../types/task';

const props = defineProps<{
  resolveTaskType?: (taskType: string) => string | undefined;
  taskId: number | null;
  visible: boolean;
}>();

defineEmits<{
  (event: 'update:visible', value: boolean): void;
}>();

const { locale, t } = useI18n();
const task = ref<TaskDetail | null>(null);
const structuredLogs = ref<readonly StructuredLogEntry[]>([]);
const projectLogContentVersion = ref(0);
const logsTruncated = ref(false);
const loading = ref(false);
const logsLoading = ref(false);
const hasLoadedLogs = ref(false);
const cancelling = ref(false);
const retryingStageId = ref<number | null>(null);
const errorMessage = ref('');
const logsError = ref('');
const socketState = ref<'idle' | 'connecting' | 'open' | 'closed' | 'error'>('idle');
let realtimeController: RealtimeTopicSocketController | null = null;
let viewEpoch = 0;

const taskLogRealtimeBatcher = new TaskLogRealtimeBatcher({
  onCommit: (snapshot) => {
    structuredLogs.value = snapshot.entries;
    projectLogContentVersion.value = snapshot.contentVersion;
    logsTruncated.value = snapshot.truncated;
  },
});
const stageOptions = computed<StepItemProps[]>(() =>
  (task.value?.stages ?? []).map((stage) => ({
    content: stageDescription(stage),
    status: stageStepStatus(stage.status),
    title: stage.key,
    value: stage.id,
  })),
);
const retryableStages = computed(() => (task.value?.stages ?? []).filter(canRetryStage));
const durationLabel = computed(() => {
  if (!task.value?.duration_ms) return '-';
  return t('task.detail.durationValue', { seconds: Math.max(1, Math.round(task.value.duration_ms / 1000)) });
});
const connectionLabel = computed(() => t(`task.connection.${socketState.value}`));
const logViewerBindings = computed(() => ({
  allLevelsLabel: t('task.logs.allLevels'),
  autoScrollLabel: t('task.logs.autoScroll'),
  autoScrollTooltipLabel: t('task.logs.autoScrollTooltip'),
  basicInfoLabel: t('task.logs.basicInfo'),
  clearLabel: t('task.logs.clear'),
  collapseDetailLabel: t('task.logs.collapseDetail'),
  copyErrorLabel: t('task.logs.copyError'),
  copyJsonLabel: t('task.logs.copyJson'),
  copyLabel: t('task.logs.copy'),
  copyLineLabel: t('task.logs.copyLine'),
  copyMessageLabel: t('task.logs.copyMessage'),
  copySuccessLabel: t('task.logs.copySuccess'),
  detailTitleLabel: t('task.logs.detailTitle'),
  downloadLabel: t('task.logs.download'),
  emptyLabel: t('task.logs.empty'),
  importantFieldsLabel: t('task.logs.importantFields'),
  jumpBottomLabel: t('task.logs.jumpBottom'),
  levelLabel: t('task.logs.level'),
  levelFilterLabel: t('task.logs.levelFilter'),
  matchCountLabel: t('task.logs.matchCount'),
  messageLabel: t('task.logs.message'),
  metadataLabel: t('task.logs.metadata'),
  operationLabel: t('task.logs.operation'),
  pauseLabel: t('task.logs.pause'),
  rawLabel: t('task.logs.raw'),
  reconnectLabel: t('task.logs.reconnect'),
  resumeLabel: t('task.logs.resume'),
  retryLabel: t('task.logs.retry'),
  searchPlaceholder: t('task.logs.search'),
  sourceLabel: t('task.logs.source'),
  stderrLabel: t('task.logs.stderr'),
  stdoutLabel: t('task.logs.stdout'),
  streamLabel: t('task.logs.stream'),
  timeLabel: t('task.logs.time'),
  truncatedLabel: t('task.logs.truncated'),
  viewDetailLabel: t('task.logs.viewDetail'),
  wrapLabel: t('task.logs.wrap'),
}));

function closeRealtime() {
  realtimeController?.close();
  realtimeController = null;
  socketState.value = 'idle';
}

async function loadLogs(afterSequence?: number, epoch = viewEpoch) {
  const taskId = props.taskId;
  if (!taskId) return;
  logsLoading.value = true;
  logsError.value = '';
  try {
    const response = await getTaskLogs(taskId, { after_sequence: afterSequence, limit: 1000 });
    if (epoch !== viewEpoch || taskId !== props.taskId) return;
    if (afterSequence === undefined) taskLogRealtimeBatcher.seed(response);
    else taskLogRealtimeBatcher.append(response);
  } catch (error) {
    logsError.value = resolveLocalizedErrorMessage(t, error, t('task.logs.loadFailed'));
  } finally {
    logsLoading.value = false;
    if (epoch === viewEpoch && taskId === props.taskId) hasLoadedLogs.value = true;
  }
}

async function refreshFromDurableState() {
  if (!props.taskId || !props.visible) return;
  const epoch = viewEpoch;
  try {
    const afterSequence = hasLoadedLogs.value ? taskLogRealtimeBatcher.nextAfterSequence() : undefined;
    const [nextTask] = await Promise.all([getTask(props.taskId), loadLogs(afterSequence, epoch)]);
    if (epoch !== viewEpoch || !props.visible) return;
    task.value = nextTask;
  } catch (error) {
    errorMessage.value = resolveLocalizedErrorMessage(t, error, t('task.detail.loadFailed'));
  }
}

const realtimeRefreshScheduler = new TaskRealtimeRefreshScheduler({
  onRefresh: refreshFromDurableState,
});

function openRealtime() {
  if (!props.taskId || realtimeController) return;
  const taskId = props.taskId;
  realtimeController = openRealtimeTopicSocket({
    topic: buildTaskRealtimeTopicName(taskId),
    parseMessage: parseTaskRealtimeNotification,
    onMessage: (event) => {
      if (event.task_id === taskId) void realtimeRefreshScheduler.request();
    },
    onStateChange: (state) => {
      socketState.value = state;
      if (state === 'open') void realtimeRefreshScheduler.request(true);
    },
    onError: (message) => {
      logsError.value = message;
    },
  });
}

async function reload() {
  if (!props.taskId) return;
  loading.value = true;
  errorMessage.value = '';
  try {
    await realtimeRefreshScheduler.request(true);
    openRealtime();
  } catch (error) {
    errorMessage.value = resolveLocalizedErrorMessage(t, error, t('task.detail.loadFailed'));
  } finally {
    loading.value = false;
  }
}

async function cancel() {
  if (!props.taskId) return;
  cancelling.value = true;
  try {
    task.value = await cancelTask(props.taskId);
    await realtimeRefreshScheduler.request(true);
  } catch (error) {
    errorMessage.value = resolveLocalizedErrorMessage(t, error, t('task.actions.cancelFailed'));
  } finally {
    cancelling.value = false;
  }
}

async function retry(stageId: number) {
  if (!props.taskId) return;
  retryingStageId.value = stageId;
  try {
    task.value = await retryTaskStage(props.taskId, stageId);
  } catch (error) {
    errorMessage.value = resolveLocalizedErrorMessage(t, error, t('task.actions.retryFailed'));
  } finally {
    retryingStageId.value = null;
  }
}

function downloadLogs() {
  const text = structuredLogs.value.map((entry) => `[${entry.occurredAt}] ${entry.stream}: ${entry.line}`).join('\n');
  const url = URL.createObjectURL(new Blob([text], { type: 'text/plain;charset=utf-8' }));
  const anchor = document.createElement('a');
  anchor.href = url;
  anchor.download = `task-${props.taskId}-logs.txt`;
  anchor.click();
  URL.revokeObjectURL(url);
}

function taskStatusLabel(status: TaskDetail['status']) {
  return t(`task.status.${status}`);
}

function taskTypeLabel(taskType: string) {
  return props.resolveTaskType?.(taskType) ?? taskType;
}

function stageStepStatus(status: TaskStageStatus): StepItemProps['status'] {
  if (status === 'success' || status === 'skipped' || status === 'cancelled') return 'finish';
  if (status === 'running') return 'process';
  if (status === 'failed' || status === 'unknown') return 'error';
  return 'default';
}

function stageDescription(stage: TaskStage) {
  if (stage.failure_message) return stage.failure_message;
  if (stage.duration_ms)
    return t('task.detail.durationValue', { seconds: Math.max(1, Math.round(stage.duration_ms / 1000)) });
  return t(`task.stageStatus.${stage.status}`);
}

function canRetryStage(stage: TaskStage) {
  return Boolean(task.value?.capabilities.retry && (stage.status === 'failed' || stage.status === 'unknown'));
}

function formatTime(value: string) {
  return formatLocaleDateTime(value, locale.value);
}

watch(
  () => [props.visible, props.taskId],
  ([visible, taskId]) => {
    viewEpoch += 1;
    realtimeRefreshScheduler.cancel();
    closeRealtime();
    task.value = null;
    taskLogRealtimeBatcher.clear();
    structuredLogs.value = [];
    projectLogContentVersion.value = 0;
    logsTruncated.value = false;
    hasLoadedLogs.value = false;
    if (visible && taskId) void reload();
  },
  { immediate: true },
);

onUnmounted(() => {
  realtimeRefreshScheduler.destroy();
  taskLogRealtimeBatcher.clear();
  closeRealtime();
});
</script>
<style scoped lang="less">
.task-detail {
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-24);
}

.task-detail h3,
.task-detail h4,
.task-detail p {
  margin: 0;
}

.task-detail__summary,
.task-detail__section-heading,
.task-detail__actions {
  align-items: center;
  display: flex;
  gap: var(--graft-density-gap-16);
  justify-content: space-between;
}

.task-detail__summary p {
  color: var(--td-text-color-secondary);
  margin-top: var(--graft-density-gap-4);
}
</style>
