<template>
  <section class="backup-page" data-page-type="list-form-detail">
    <management-page-header
      :source="{ labelKey: 'backup.list.eyebrow', fallback: t('backup.list.eyebrow') }"
      title-key="backup.list.title"
      :title-fallback="t('backup.list.title')"
      description-key="backup.list.description"
      :description-fallback="t('backup.list.description')"
    />

    <management-toolbar compact-action-layout="equal-width">
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

    <management-paged-table
      v-model:current="current"
      v-model:page-size="pageSize"
      :columns="columns"
      density-scope="viewport"
      entity-card-layout="compact"
      :empty-description="t('backup.list.empty')"
      :empty-title="t('backup.list.title')"
      :footer-summary="footerSummary"
      hide-footer-summary-on-compact
      :loading="loading"
      presentation="entity"
      :rows="backups"
      :total="total"
      @page-change="handlePageChange"
    >
      <template #head>
        <p class="backup-page__table-summary">{{ t('backup.list.description') }}</p>
      </template>
      <template #status="{ row }">
        <t-tag :theme="statusTheme(row.status)" variant="light-outline">{{ statusLabel(row.status) }}</t-tag>
      </template>
      <template #contents>{{ t('backup.content.summary') }}</template>
      <template #retain_until="{ row }">{{ formatDate(row.retain_until) }}</template>
      <template #created_at="{ row }">{{ formatDate(row.created_at) }}</template>
      <template #actions="{ row }">
        <t-button size="small" theme="primary" variant="text" @click="openBackup(row)">
          {{ t('backup.list.actions.view') }}
        </t-button>
      </template>
      <template #empty>
        <t-empty :description="t('backup.list.empty')">
          <template #action>
            <t-button v-permission="permissionCodes.CREATE" theme="primary" @click="createDialogVisible = true">
              {{ t('backup.list.create') }}
            </t-button>
          </template>
        </t-empty>
      </template>
      <template #cards>
        <template v-if="loading">
          <article v-for="index in 3" :key="`backup-card-skeleton-${index}`" class="backup-card backup-card--skeleton">
            <t-skeleton animation="gradient" :row-col="backupCardSkeletonRows" />
          </article>
        </template>
        <template v-else-if="backups.length">
          <article
            v-for="backup in backups"
            :key="backup.id"
            class="backup-card"
            :data-testid="`backup-card-${backup.id}`"
          >
            <header class="backup-card__header">
              <p class="backup-card__identifier">{{ t('backup.detail.identifier', { id: backup.id }) }}</p>
              <t-tag :theme="statusTheme(backup.status)" variant="light-outline">
                {{ statusLabel(backup.status) }}
              </t-tag>
            </header>
            <dl class="backup-card__details">
              <div>
                <dt>{{ t('backup.list.columns.contents') }}</dt>
                <dd>{{ t('backup.content.summary') }}</dd>
              </div>
              <div>
                <dt>{{ t('backup.list.columns.createdAt') }}</dt>
                <dd>{{ formatDate(backup.created_at) }}</dd>
              </div>
            </dl>
            <div class="backup-card__actions">
              <t-button theme="primary" variant="text" @click="openBackup(backup)">
                {{ t('backup.list.actions.view') }}
                <template #suffix><chevron-right-icon /></template>
              </t-button>
            </div>
          </article>
        </template>
        <t-empty v-else :description="t('backup.list.empty')">
          <template #action>
            <t-button v-permission="permissionCodes.CREATE" theme="primary" @click="createDialogVisible = true">
              {{ t('backup.list.create') }}
            </t-button>
          </template>
        </t-empty>
      </template>
    </management-paged-table>

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
      size="min(680px, 92vw)"
    >
      <t-loading :loading="detailLoading">
        <t-alert v-if="detailError" theme="error" :message="detailError" />
        <div v-else-if="selectedBackup" class="backup-detail" data-testid="backup-detail-drawer">
          <div class="backup-detail__summary">
            <div>
              <p class="backup-detail__identifier">{{ t('backup.detail.identifier', { id: selectedBackup.id }) }}</p>
              <h3>{{ purposeLabel(selectedBackup.purpose) }}</h3>
              <p>{{ t('backup.detail.summary.contents', { count: 2 }) }}</p>
            </div>
            <div class="backup-detail__summary-status">
              <t-tag :theme="statusTheme(selectedBackup.status)" variant="light-outline">
                {{ statusLabel(selectedBackup.status) }}
              </t-tag>
              <strong>{{ formatBytes(totalArtifactBytes(selectedBackup)) }}</strong>
            </div>
          </div>

          <t-descriptions bordered :column="1" :title="t('backup.detail.asset.title')">
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
          </t-descriptions>

          <section class="backup-detail__artifacts" :aria-label="t('backup.detail.contents.title')">
            <h3>{{ t('backup.detail.contents.title') }}</h3>
            <div class="backup-detail__artifact-grid">
              <backup-artifact-card
                v-for="artifact in artifactCards"
                :key="artifact.title"
                :copy-label="t('backup.detail.contents.copyChecksum')"
                :sha256="artifact.sha256"
                :size-bytes="artifact.sizeBytes"
                :title="t(artifact.title)"
                @copy="copyChecksum"
              />
            </div>
          </section>

          <section class="backup-detail__restore" :aria-label="t('backup.detail.restore.title')">
            <h3>{{ t('backup.detail.restore.title') }}</h3>
            <t-alert
              :theme="selectedBackup.restore_evidence.status === 'RECORDED' ? 'success' : 'info'"
              :title="restoreEvidenceTitle(selectedBackup.restore_evidence.status)"
              :message="restoreEvidenceMessage(selectedBackup.restore_evidence)"
            />
          </section>

          <section class="backup-detail__task" :aria-label="t('backup.detail.task.title')">
            <h3>{{ t('backup.detail.task.title') }}</h3>
            <t-button
              v-if="selectedBackup.task_id"
              theme="primary"
              variant="text"
              @click="openTask(selectedBackup.task_id)"
            >
              {{ t('backup.detail.task.view', { id: selectedBackup.task_id }) }}
            </t-button>
            <span v-else>{{ t('backup.detail.task.none') }}</span>
          </section>
        </div>
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
// Backup 页面只消费安全资产投影；存储位置、工件内容和恢复执行权始终留在服务端边界。
import { AddIcon, ChevronRightIcon, RefreshIcon } from 'tdesign-icons-vue-next';
import type { PageInfo, PrimaryTableCol } from 'tdesign-vue-next';
import { computed, onMounted, onUnmounted, ref } from 'vue';
import { useI18n } from 'vue-i18n';

import { TaskDetailDrawer } from '@/modules/task/contract/task-ui';
import { isTerminalTaskStatus, observeTask, type TaskObserver } from '@/modules/task/task-observer';
import { ManagementPagedTable, ManagementPageHeader, ManagementToolbar } from '@/shared/components/management';
import { resolveLocalizedErrorMessage } from '@/shared/localized-api-error';
import { copyText, formatBytes, formatLocaleDateTime } from '@/shared/observability';

import { getBackup, listBackups, submitBackup } from '../../api/backup';
import BackupArtifactCard from '../../components/BackupArtifactCard.vue';
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
let listRequestSequence = 0;
let detailRequestSequence = 0;

const retentionOptions: readonly BackupRetention[] = ['1d', '7d', '30d'];
const footerSummary = computed(() => t('backup.list.footerTotal', { count: total.value }));
const backupCardSkeletonRows = [
  [
    { width: '40%', height: '18px' },
    { width: '72px', height: '24px', marginLeft: 'auto' },
  ],
  { width: '62%', height: '16px', marginTop: '20px' },
  { width: '74%', height: '16px', marginTop: '14px' },
  { width: '94px', height: '20px', marginTop: '20px' },
];

const artifactCards = computed(() => {
  if (!selectedBackup.value) return [];
  return [
    {
      title: 'backup.detail.contents.configSnapshot',
      sizeBytes: selectedBackup.value.config_snapshot.size_bytes,
      sha256: selectedBackup.value.config_snapshot.sha256,
    },
    {
      title: 'backup.detail.contents.databaseDump',
      sizeBytes: selectedBackup.value.database_dump.size_bytes,
      sha256: selectedBackup.value.database_dump.sha256,
    },
  ];
});
const columns = computed<PrimaryTableCol[]>(() => [
  { colKey: 'id', title: t('backup.list.columns.id'), width: 112 },
  { colKey: 'contents', title: t('backup.list.columns.contents'), minWidth: 190 },
  { colKey: 'status', title: t('backup.list.columns.status'), width: 130 },
  { colKey: 'retain_until', title: t('backup.list.columns.retention'), minWidth: 180 },
  { colKey: 'created_at', title: t('backup.list.columns.createdAt'), minWidth: 180 },
  { colKey: 'actions', title: t('backup.list.columns.actions'), width: 112, fixed: 'right' },
]);

onMounted(() => void loadBackups());
onUnmounted(() => submittedTaskObserver?.stop());

async function loadBackups() {
  const requestSequence = ++listRequestSequence;
  loading.value = true;
  errorMessage.value = '';
  try {
    const response = await listBackups({ limit: pageSize.value, offset: (current.value - 1) * pageSize.value });
    if (requestSequence !== listRequestSequence) return;
    backups.value = response.items;
    total.value = response.total;
  } catch (error) {
    if (requestSequence !== listRequestSequence) return;
    errorMessage.value = resolveLocalizedErrorMessage(t, error, t('backup.list.loadFailed'));
  } finally {
    if (requestSequence === listRequestSequence) loading.value = false;
  }
}

function handlePageChange(pageInfo: PageInfo) {
  current.value = pageInfo.current;
  pageSize.value = pageInfo.pageSize;
  void loadBackups();
}

function openBackup(backup: BackupSummary) {
  const requestSequence = ++detailRequestSequence;
  backupDrawerVisible.value = true;
  selectedBackup.value = null;
  detailLoading.value = true;
  detailError.value = '';
  void getBackup(backup.id)
    .then((detail) => {
      if (requestSequence !== detailRequestSequence) return;
      selectedBackup.value = detail;
    })
    .catch((error: unknown) => {
      if (requestSequence !== detailRequestSequence) return;
      detailError.value = resolveLocalizedErrorMessage(t, error, t('backup.list.loadFailed'));
    })
    .finally(() => {
      if (requestSequence !== detailRequestSequence) return;
      detailLoading.value = false;
    });
}

function openTask(taskId: number) {
  backupDrawerVisible.value = false;
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
  if (crypto?.getRandomValues) {
    crypto.getRandomValues(entropy);
  } else {
    // 幂等键只用于避免同毫秒提交碰撞；非浏览器运行时以本地随机值保持可提交性，不承担密钥用途。
    for (let index = 0; index < entropy.length; index += 1) {
      entropy[index] = Math.floor(Math.random() * 0x1_0000_0000);
    }
  }
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
  if (key === 'available' || key === 'expired' || key === 'restored') return t(`backup.status.${key}`);
  return t('backup.status.unknown');
}

function statusTheme(value: string) {
  if (value.toLowerCase() === 'available') return 'success';
  if (value.toLowerCase() === 'restored') return 'primary';
  return 'default';
}

function restoreEvidenceTitle(status: BackupDetail['restore_evidence']['status']) {
  return status === 'RECORDED' ? t('backup.detail.restore.recordedTitle') : t('backup.detail.restore.notVerifiedTitle');
}

function totalArtifactBytes(backup: BackupDetail) {
  return backup.config_snapshot.size_bytes + backup.database_dump.size_bytes;
}

function copyChecksum(value: string) {
  void copyText(value);
}

function restoreEvidenceMessage(evidence: BackupDetail['restore_evidence']) {
  if (evidence.status !== 'RECORDED' || !evidence.recorded_at) return t('backup.detail.restore.notVerifiedMessage');
  return t('backup.detail.restore.recordedMessage', {
    time: formatDate(evidence.recorded_at),
    result: restoreResultLabel(evidence.result_code),
  });
}

function restoreResultLabel(resultCode?: string | null) {
  if (resultCode === 'manual_restore_verified') return t('backup.detail.restore.results.manualRestoreVerified');
  return t('backup.detail.restore.results.recorded');
}

function resolveTaskType(taskType: string) {
  return taskType === 'platform.backup.create.v1' ? t('backup.list.create') : undefined;
}
</script>
<style scoped lang="less">
@import '@/shared/components/card-surface.less';

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

.backup-card {
  .graft-entity-card-surface();

  display: grid;
  gap: var(--graft-density-gap-16);
  min-width: 0;
  padding: var(--graft-density-gap-16);
}

.backup-card__header,
.backup-card__actions {
  align-items: center;
  display: flex;
  gap: var(--graft-density-gap-12);
  justify-content: space-between;
  min-width: 0;
}

.backup-card__identifier {
  color: var(--td-text-color-primary);
  font: var(--td-font-title-medium);
  margin: 0;
  min-width: 0;
  overflow-wrap: anywhere;
}

.backup-card__details {
  display: grid;
  gap: var(--graft-density-gap-12);
  margin: 0;
}

.backup-card__details div {
  border-top: 1px solid var(--td-component-stroke);
  display: grid;
  gap: var(--graft-density-gap-4);
  padding-top: var(--graft-density-gap-12);
}

.backup-card__details dt {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
}

.backup-card__details dd {
  color: var(--td-text-color-primary);
  font: var(--td-font-body-medium);
  margin: 0;
  overflow-wrap: anywhere;
}

.backup-card__actions {
  border-top: 1px solid var(--td-component-stroke);
  justify-content: flex-end;
  padding-top: var(--graft-density-gap-12);
}

.backup-detail {
  display: grid;
  gap: var(--td-comp-margin-xl);
}

.backup-detail__restore,
.backup-detail__task,
.backup-detail__artifacts {
  display: grid;
  gap: var(--td-comp-margin-s);
}

.backup-detail h3,
.backup-detail p {
  margin: 0;
}

.backup-detail h3 {
  font-size: var(--td-font-size-title-medium);
  font-weight: var(--td-font-weight-medium);
}

.backup-detail__summary,
.backup-detail__summary-status {
  align-items: center;
  display: flex;
  gap: var(--td-comp-margin-s);
  justify-content: space-between;
}

.backup-detail__summary {
  border-bottom: 1px solid var(--td-component-stroke);
  padding-bottom: var(--td-comp-paddingTB-xl);
}

.backup-detail__summary p,
.backup-detail__artifact-label,
.backup-detail__task span {
  color: var(--td-text-color-secondary);
}

.backup-detail__identifier {
  color: var(--td-text-color-placeholder);
  font-size: var(--td-font-size-body-small);
}

.backup-detail__summary-status {
  align-items: flex-end;
  flex-direction: column;
}

.backup-detail__artifact-grid {
  display: grid;
  gap: var(--td-comp-margin-s);
  grid-template-columns: repeat(2, minmax(0, 1fr));
}

.backup-detail__artifact-size {
  display: block;
  font-size: var(--td-font-size-title-large);
}

.backup-detail__artifact-label {
  font-size: var(--td-font-size-body-small);
  margin-top: var(--td-comp-margin-s) !important;
}

.backup-detail__checksum {
  display: block;
  font-size: var(--td-font-size-body-small);
  margin-top: var(--td-comp-margin-xxs);
  overflow-wrap: anywhere;
}

@media (width <= 600px) {
  .backup-detail__artifact-grid {
    grid-template-columns: 1fr;
  }
}
</style>
