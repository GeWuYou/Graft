<template>
  <section class="backup-page" data-page-type="list-form-detail">
    <management-page-header
      :source="{ labelKey: 'backup.list.eyebrow', fallback: t('backup.list.eyebrow') }"
      title-key="backup.list.title"
      :title-fallback="t('backup.list.title')"
      description-key="backup.list.description"
      :description-fallback="t('backup.list.description')"
    />

    <management-toolbar>
      <template #actions>
        <t-button theme="default" variant="outline" :loading="loading" @click="loadBackups">
          <template #icon><refresh-icon /></template>
          {{ t('backup.list.refresh') }}
        </t-button>
        <t-button v-permission="permissionCodes.CREATE" theme="primary" @click="createDialogVisible = true">
          <template #icon><add-icon /></template>
          {{ t('backup.list.create') }}
        </t-button>
      </template>
    </management-toolbar>

    <t-alert
      v-if="errorMessage"
      class="backup-page__alert"
      theme="error"
      :title="t('backup.list.loadFailed')"
      :message="errorMessage"
    />

    <management-table-card>
      <template #head>
        <p class="backup-page__table-summary">{{ t('backup.list.description') }}</p>
      </template>
      <t-table
        row-key="id"
        :columns="tableColumns"
        :data="backups"
        :loading="loading"
        :pagination="pagination"
        :disable-data-page="true"
        @page-change="handlePageChange"
      >
        <template #purpose="{ row }">{{ purposeLabel(row.purpose) }}</template>
        <template #status="{ row }">
          <t-tag :theme="statusTheme(row.status)" variant="light-outline">{{ statusLabel(row.status) }}</t-tag>
        </template>
        <template #retain_until="{ row }">{{ formatDate(row.retain_until) }}</template>
        <template #created_at="{ row }">{{ formatDate(row.created_at) }}</template>
        <template #task_id="{ row }">
          <t-button v-if="row.task_id" size="small" variant="text" @click="openTask(row.task_id)">
            #{{ row.task_id }}
          </t-button>
          <span v-else>-</span>
        </template>
        <template #actions="{ row }">
          <t-button size="small" theme="primary" variant="text" @click="openBackup(row)">
            {{ t('backup.list.actions.view') }}
          </t-button>
        </template>
        <template #empty>
          <t-empty :description="t('backup.list.empty')" />
        </template>
      </t-table>
    </management-table-card>

    <t-dialog
      v-model:visible="createDialogVisible"
      :header="t('backup.createDialog.title')"
      :confirm-btn="{ content: t('backup.createDialog.confirm'), loading: submitting }"
      :cancel-btn="{ content: t('backup.createDialog.cancel') }"
      width="460px"
      @confirm="submitManualBackup"
    >
      <p class="backup-page__dialog-description">{{ t('backup.createDialog.description') }}</p>
      <t-form label-align="top">
        <t-form-item :label="t('backup.createDialog.retention')">
          <t-select v-model="retention" data-testid="backup-retention">
            <t-option
              v-for="option in retentionOptions"
              :key="option"
              :value="option"
              :label="retentionLabel(option)"
            />
          </t-select>
        </t-form-item>
      </t-form>
      <t-alert v-if="submitError" theme="error" :message="submitError" />
    </t-dialog>

    <t-drawer
      v-model:visible="backupDrawerVisible"
      :header="t('backup.detail.title')"
      :footer="false"
      placement="right"
      size="560px"
    >
      <t-loading :loading="detailLoading">
        <t-alert v-if="detailError" theme="error" :message="detailError" />
        <t-descriptions v-else-if="selectedBackup" bordered :column="1">
          <t-descriptions-item :label="t('backup.list.columns.id')">#{{ selectedBackup.id }}</t-descriptions-item>
          <t-descriptions-item :label="t('backup.list.columns.purpose')">
            {{ purposeLabel(selectedBackup.purpose) }}
          </t-descriptions-item>
          <t-descriptions-item :label="t('backup.list.columns.status')">
            <t-tag :theme="statusTheme(selectedBackup.status)" variant="light-outline">
              {{ statusLabel(selectedBackup.status) }}
            </t-tag>
          </t-descriptions-item>
          <t-descriptions-item :label="t('backup.list.columns.retention')">
            {{ formatDate(selectedBackup.retain_until) }}
          </t-descriptions-item>
          <t-descriptions-item :label="t('backup.list.columns.createdAt')">
            {{ formatDate(selectedBackup.created_at) }}
          </t-descriptions-item>
          <t-descriptions-item :label="t('backup.list.columns.task')">
            <t-button
              v-if="selectedBackup.task_id"
              size="small"
              variant="text"
              @click="openTask(selectedBackup.task_id)"
            >
              #{{ selectedBackup.task_id }}
            </t-button>
            <span v-else>-</span>
          </t-descriptions-item>
        </t-descriptions>
      </t-loading>
    </t-drawer>

    <task-detail-drawer
      v-model:visible="taskDrawerVisible"
      :task-id="selectedTaskId"
      :resolve-task-type="resolveTaskType"
    />
  </section>
</template>
<script setup lang="ts">
// Backup 页面只管理安全历史和 Task receipt，不读取工件、路径或恢复能力。
import { AddIcon, RefreshIcon } from 'tdesign-icons-vue-next';
import type { PageInfo, PrimaryTableCol } from 'tdesign-vue-next';
import { computed, onMounted, onUnmounted, ref } from 'vue';
import { useI18n } from 'vue-i18n';

import { TaskDetailDrawer } from '@/modules/task/contract/task-ui';
import { isTerminalTaskStatus, observeTask, type TaskObserver } from '@/modules/task/task-observer';
import { ManagementPageHeader, ManagementTableCard, ManagementToolbar } from '@/shared/components/management';
import { resolveLocalizedErrorMessage } from '@/shared/localized-api-error';
import { formatLocaleDateTime } from '@/shared/observability';

import { getBackup, listBackups, submitBackup } from '../../api/backup';
import { BACKUP_PERMISSION_CODE as permissionCodes } from '../../contract/permissions';
import type { BackupDetail, BackupRetention, BackupSummary } from '../../types/backup';

const { locale, t } = useI18n();
const backups = ref<BackupSummary[]>([]);
const loading = ref(false);
const submitting = ref(false);
const errorMessage = ref('');
const submitError = ref('');
const createDialogVisible = ref(false);
const taskDrawerVisible = ref(false);
const selectedTaskId = ref<number | null>(null);
const backupDrawerVisible = ref(false);
const selectedBackup = ref<BackupDetail | null>(null);
const detailLoading = ref(false);
const detailError = ref('');
const retention = ref<BackupRetention>('30d');
const total = ref(0);
const current = ref(1);
const pageSize = ref(20);
let submittedTaskObserver: TaskObserver | null = null;

const retentionOptions: readonly BackupRetention[] = ['1d', '7d', '30d'];
const pagination = computed(() => ({
  current: current.value,
  pageSize: pageSize.value,
  total: total.value,
}));
const columns = computed<PrimaryTableCol<BackupSummary>[]>(() => [
  { colKey: 'id', title: t('backup.list.columns.id'), width: 112 },
  { colKey: 'purpose', title: t('backup.list.columns.purpose'), minWidth: 160 },
  { colKey: 'status', title: t('backup.list.columns.status'), width: 130 },
  { colKey: 'retain_until', title: t('backup.list.columns.retention'), minWidth: 180 },
  { colKey: 'created_at', title: t('backup.list.columns.createdAt'), minWidth: 180 },
  { colKey: 'task_id', title: t('backup.list.columns.task'), width: 100 },
  { colKey: 'actions', title: t('backup.list.columns.actions'), width: 96, fixed: 'right' },
]);
const tableColumns = columns as unknown as PrimaryTableCol[];

onMounted(() => void loadBackups());
onUnmounted(() => submittedTaskObserver?.stop());

async function loadBackups() {
  loading.value = true;
  errorMessage.value = '';
  try {
    const response = await listBackups({ limit: pageSize.value, offset: (current.value - 1) * pageSize.value });
    backups.value = response.items;
    total.value = response.total;
  } catch (error) {
    errorMessage.value = resolveLocalizedErrorMessage(t, error, t('backup.list.loadFailed'));
  } finally {
    loading.value = false;
  }
}

function handlePageChange(pageInfo: PageInfo) {
  current.value = pageInfo.current;
  pageSize.value = pageInfo.pageSize;
  void loadBackups();
}

function openBackup(backup: BackupSummary) {
  backupDrawerVisible.value = true;
  selectedBackup.value = null;
  detailLoading.value = true;
  detailError.value = '';
  void getBackup(backup.id)
    .then((detail) => {
      selectedBackup.value = detail;
    })
    .catch((error: unknown) => {
      detailError.value = resolveLocalizedErrorMessage(t, error, t('backup.list.loadFailed'));
    })
    .finally(() => {
      detailLoading.value = false;
    });
}

function openTask(taskId: number) {
  selectedTaskId.value = taskId;
  taskDrawerVisible.value = true;
}

async function submitManualBackup() {
  submitting.value = true;
  submitError.value = '';
  try {
    const receipt = await submitBackup({ retention: retention.value }, createIdempotencyKey());
    createDialogVisible.value = false;
    observeSubmittedTask(receipt.task_id);
    openTask(receipt.task_id);
  } catch (error) {
    submitError.value = resolveLocalizedErrorMessage(t, error, t('backup.createDialog.submitFailed'));
  } finally {
    submitting.value = false;
  }
}

function observeSubmittedTask(taskId: number) {
  submittedTaskObserver?.stop();
  submittedTaskObserver = observeTask(taskId, {
    onTask: (task) => {
      if (!isTerminalTaskStatus(task.status)) return;
      submittedTaskObserver?.stop();
      submittedTaskObserver = null;
      if (task.status === 'success') void loadBackups();
    },
  });
}

function createIdempotencyKey() {
  const crypto = globalThis.crypto;
  const uuid = crypto?.randomUUID?.();
  if (uuid) return uuid;
  const entropy = new Uint32Array(4);
  crypto?.getRandomValues?.(entropy);
  return `platform-backup-${Date.now()}-${Array.from(entropy, (value) => value.toString(36)).join('')}`;
}

function formatDate(value: string) {
  return formatLocaleDateTime(value, locale.value);
}

function retentionLabel(value: BackupRetention) {
  return t(`backup.createDialog.retentionOptions.${value}`);
}

function purposeLabel(value: string) {
  if (value === 'platform_manual') return t('backup.purpose.manual');
  if (value === 'platform_update') return t('backup.purpose.platformUpdate');
  return value;
}

function statusLabel(value: string) {
  const key = value.toLowerCase();
  if (key === 'available' || key === 'restored') return t(`backup.status.${key}`);
  return t('backup.status.unknown');
}

function statusTheme(value: string) {
  if (value.toLowerCase() === 'available') return 'success';
  if (value.toLowerCase() === 'restored') return 'primary';
  return 'default';
}

function resolveTaskType(taskType: string) {
  return taskType === 'platform.backup.create.v1' ? t('backup.list.create') : undefined;
}
</script>
<style scoped lang="less">
.backup-page {
  display: grid;
  gap: var(--td-comp-margin-xl);
}

.backup-page__alert,
.backup-page__table-summary,
.backup-page__dialog-description {
  margin: 0;
}

.backup-page__table-summary,
.backup-page__dialog-description {
  color: var(--td-text-color-secondary);
}
</style>
