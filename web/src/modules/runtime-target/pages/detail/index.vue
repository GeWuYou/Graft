<template>
  <div class="runtime-target-detail" data-page-type="overview-dashboard">
    <management-page-content>
      <management-page-header
        :title="target?.displayName || t('runtimeTarget.detail.title')"
        :description="t('runtimeTarget.detail.description')"
      >
        <template #actions
          ><t-button theme="primary" variant="outline" :loading="refreshing" @click="refresh"
            ><template #icon><refresh-icon /></template>{{ t('runtimeTarget.detail.refresh') }}</t-button
          ></template
        >
      </management-page-header>
      <t-alert v-if="errorMessage" theme="error" :message="errorMessage" class="runtime-target-feedback" />
      <t-card v-if="loading" :loading="true" />
      <template v-else-if="target">
        <t-card :title="t('runtimeTarget.detail.runtimeInformation')">
          <t-descriptions :column="2" bordered>
            <t-descriptions-item :label="t('runtimeTarget.detail.provider')">{{
              target.runtime.provider
            }}</t-descriptions-item>
            <t-descriptions-item :label="t('runtimeTarget.detail.runtimeType')">{{
              target.runtime.type
            }}</t-descriptions-item>
            <t-descriptions-item :label="t('runtimeTarget.detail.endpoint')">{{
              target.connection.endpoint
            }}</t-descriptions-item>
            <t-descriptions-item :label="t('runtimeTarget.detail.connectionKind')">{{
              target.connection.kind
            }}</t-descriptions-item>
            <t-descriptions-item :label="t('runtimeTarget.detail.version')">{{
              target.runtime.version
            }}</t-descriptions-item>
            <t-descriptions-item :label="t('runtimeTarget.detail.apiVersion')">{{
              target.runtime.apiVersion || t('runtimeTarget.metrics.unavailable')
            }}</t-descriptions-item>
            <t-descriptions-item :label="t('runtimeTarget.detail.health')"
              ><t-tag :theme="target.health.status === 'healthy' ? 'success' : 'danger'" variant="light">{{
                target.health.status === 'healthy'
                  ? t('runtimeTarget.status.healthy')
                  : t('runtimeTarget.status.unavailable')
              }}</t-tag></t-descriptions-item
            >
            <t-descriptions-item :label="t('runtimeTarget.detail.lastCheckedAt')">{{
              target.health.lastCheckedAt
                ? formatLocaleDateTime(target.health.lastCheckedAt, locale)
                : t('runtimeTarget.metrics.unavailable')
            }}</t-descriptions-item>
          </t-descriptions>
          <t-alert
            v-if="target.health.diagnostic"
            theme="warning"
            :message="target.health.diagnostic"
            class="runtime-target-diagnostic"
          />
        </t-card>
        <t-card :title="t('runtimeTarget.detail.resourceOverview')" class="runtime-target-section">
          <div class="runtime-target-stat-grid">
            <t-statistic
              :title="t('runtimeTarget.metrics.workloads')"
              :value="countValue(target.resources.workloads)"
              :extra="
                target.resources.workloads.available
                  ? t('runtimeTarget.metrics.activeCount', { count: target.resources.workloads.active })
                  : unavailableReason(target.resources.workloads.unavailableReason)
              "
            />
            <t-statistic
              :title="t('runtimeTarget.metrics.cpu')"
              :value="usageValue(target.resources.cpu)"
              unit="%"
              :extra="usageExtra(target.resources.cpu)"
            />
            <t-statistic
              :title="t('runtimeTarget.metrics.memory')"
              :value="usageValue(target.resources.memory)"
              unit="%"
              :extra="usageExtra(target.resources.memory)"
            />
            <t-statistic
              :title="t('runtimeTarget.metrics.storage')"
              :value="usageValue(target.resources.storage)"
              unit="%"
              :extra="usageExtra(target.resources.storage)"
            />
          </div>
        </t-card>
        <t-card :title="t('runtimeTarget.detail.providerDetails')" class="runtime-target-section"
          ><docker-provider-details
            v-if="target.providerDetails.provider === 'docker'"
            :details="target.providerDetails.docker"
        /></t-card>
        <t-card
          v-if="canManageAssignments"
          :title="t('runtimeTarget.detail.authorizedUsers')"
          class="runtime-target-section"
        >
          <t-alert v-if="assignmentError" theme="error" :message="assignmentError" class="runtime-target-feedback" />
          <div class="runtime-target-assignment-actions">
            <span>{{ t('runtimeTarget.detail.authorizedUsersHint') }}</span>
            <t-button variant="outline" :loading="assignmentsLoading" @click="assignmentDialogVisible = true">
              {{ t('runtimeTarget.detail.changeAuthorizedUsers') }}
            </t-button>
          </div>
          <t-table
            class="runtime-target-assignment-table"
            row-key="user_id"
            :columns="assignmentColumns"
            :data="pagedAssignments"
            :loading="assignmentsLoading"
          >
            <template #user="{ row }">
              {{ assignmentDisplay(row) }}
            </template>
            <template #username="{ row }">
              {{ assignmentUsername(row) }}
            </template>
            <template #accountStatus="{ row }">
              <t-tag :theme="accountStatusTheme(row)" variant="light">{{ accountStatusLabel(row) }}</t-tag>
            </template>
            <template #authorizationStatus="{ row }">
              <t-tag :theme="row.authorizationState === 'active' ? 'success' : 'default'" variant="light">
                {{
                  row.authorizationState === 'active'
                    ? t('runtimeTarget.detail.authorizationActive')
                    : t('runtimeTarget.detail.authorizationRevoked')
                }}
              </t-tag>
            </template>
            <template #authorizedAt="{ row }">
              {{ formatLocaleDateTime(row.authorized_at, locale) }}
            </template>
            <template #operation="{ row }">
              <t-button
                size="small"
                :theme="row.authorizationState === 'active' ? 'danger' : 'primary'"
                variant="text"
                :loading="assignmentChangingUserId === row.user_id"
                @click="toggleAssignment(row)"
              >
                {{
                  row.authorizationState === 'active'
                    ? t('runtimeTarget.detail.revokeAuthorization')
                    : t('runtimeTarget.detail.restoreAuthorization')
                }}
              </t-button>
            </template>
            <template #empty>
              <t-empty :title="t('runtimeTarget.detail.authorizedUsersEmpty')" />
            </template>
          </t-table>
          <div v-if="assignmentRows.length" class="runtime-target-assignment-pagination">
            <t-pagination
              v-model:current="assignmentPagination.current"
              v-model:page-size="assignmentPagination.pageSize"
              :page-size-options="[5, 10, 20]"
              :total="assignmentRows.length"
              :total-content="false"
              @change="normalizeAssignmentPage"
            />
          </div>
        </t-card>
      </template>
      <t-empty v-else :title="t('runtimeTarget.detail.notFound')" />
    </management-page-content>
    <runtime-target-assignment-dialog
      v-model:visible="assignmentDialogVisible"
      :target-id="targetID"
      @saved="applyDialogAssignments"
    />
  </div>
</template>
<script setup lang="ts">
// 详情页展示 provider-owned 运行时投影；刷新只请求后端重新探测，不在前端改写运行时事实。
import { RefreshIcon } from 'tdesign-icons-vue-next';
import type { TableProps } from 'tdesign-vue-next';
import { MessagePlugin } from 'tdesign-vue-next/es/message';
import { computed, defineComponent, h, onMounted, type PropType, reactive, ref } from 'vue';
import { useI18n } from 'vue-i18n';
import { useRoute } from 'vue-router';

import { useLocale } from '@/locales/useLocale';
import { ManagementPageContent, ManagementPageHeader } from '@/shared/components/management';
import { formatBytes, formatLocaleDateTime } from '@/shared/observability';
import { getPermissionStore } from '@/store/modules/permission';
import { isApiRequestError } from '@/utils/request';

import {
  getRuntimeTarget,
  getRuntimeTargetAssignments,
  refreshRuntimeTarget,
  replaceRuntimeTargetAssignments,
  type RuntimeTargetAssignment,
  type RuntimeTargetDetail,
  type RuntimeTargetUsageMetric,
} from '../../api/runtime-target';
import RuntimeTargetAssignmentDialog from '../../components/RuntimeTargetAssignmentDialog.vue';

const { t } = useI18n();
const { locale } = useLocale();
const route = useRoute();
const target = ref<RuntimeTargetDetail | null>(null);
const loading = ref(false);
const refreshing = ref(false);
const errorMessage = ref('');
const targetID = computed(() => Number(route.params.id));
const permissionStore = getPermissionStore();
const canManageAssignments = computed(() => permissionStore.hasPermission('runtime_target.assignment.manage'));
type AssignmentRow = RuntimeTargetAssignment & { authorizationState: 'active' | 'revoked' };
const activeAssignments = ref<RuntimeTargetAssignment[]>([]);
// 已撤销行仅属于当前详情实例，用于支持即时反悔；重新进入页面时完全以服务端有效授权为准。
const revokedAssignments = ref(new Map<number, RuntimeTargetAssignment>());
const assignmentOrder = ref<number[]>([]);
const assignmentRevision = ref(1);
const assignmentDialogVisible = ref(false);
const assignmentsLoading = ref(false);
const assignmentChangingUserId = ref<number | null>(null);
const assignmentError = ref('');
const assignmentPagination = reactive({ current: 1, pageSize: 5 });
const assignmentRows = computed<AssignmentRow[]>(() => {
  const activeByID = new Map(activeAssignments.value.map((item) => [item.user_id, item]));
  const rows: AssignmentRow[] = [];
  assignmentOrder.value.forEach((userID) => {
    const active = activeByID.get(userID);
    if (active) {
      rows.push({ ...active, authorizationState: 'active' });
      return;
    }
    const revoked = revokedAssignments.value.get(userID);
    if (revoked) rows.push({ ...revoked, authorizationState: 'revoked' });
  });
  return rows;
});
const pagedAssignments = computed(() => {
  const start = (assignmentPagination.current - 1) * assignmentPagination.pageSize;
  return assignmentRows.value.slice(start, start + assignmentPagination.pageSize);
});
const assignmentColumns = computed<TableProps['columns']>(() => [
  { colKey: 'user', title: t('runtimeTarget.detail.assignmentUser'), minWidth: 150 },
  { colKey: 'username', title: t('runtimeTarget.detail.assignmentUsername'), minWidth: 150 },
  { colKey: 'accountStatus', title: t('runtimeTarget.detail.accountStatus'), width: 120 },
  { colKey: 'authorizationStatus', title: t('runtimeTarget.detail.authorizationState'), width: 120 },
  { colKey: 'authorizedAt', title: t('runtimeTarget.detail.authorizedAt'), minWidth: 180 },
  { colKey: 'operation', title: t('runtimeTarget.detail.operation'), width: 120, fixed: 'right' },
]);
type DockerProviderDetailsData = RuntimeTargetDetail['providerDetails']['docker'];

const DockerProviderDetails = defineComponent({
  props: { details: { type: Object as PropType<DockerProviderDetailsData>, required: true } },
  setup(props) {
    const countText = (metric: DockerProviderDetailsData['images']) =>
      metric.available ? String(metric.total) : unavailableReason(metric.unavailableReason);

    return () =>
      h('div', { class: 'runtime-target-stat-grid' }, [
        h('div', [h('strong', t('runtimeTarget.providerDetails.images')), h('p', countText(props.details.images))]),
        h('div', [h('strong', t('runtimeTarget.providerDetails.volumes')), h('p', countText(props.details.volumes))]),
        h('div', [h('strong', t('runtimeTarget.providerDetails.networks')), h('p', countText(props.details.networks))]),
      ]);
  },
});
function countValue(metric: { available: boolean; total: number; unavailableReason: string }) {
  return metric.available ? metric.total : undefined;
}
function unavailableReason(reason: string) {
  return reason || t('runtimeTarget.metrics.unavailable');
}
function usageValue(metric: RuntimeTargetUsageMetric) {
  return metric.available ? Number(metric.usagePercent.toFixed(1)) : undefined;
}
function usageExtra(metric: RuntimeTargetUsageMetric) {
  return metric.available
    ? metric.totalBytes > 0
      ? `${formatBytes(metric.usedBytes)} / ${formatBytes(metric.totalBytes)}`
      : ''
    : unavailableReason(metric.unavailableReason);
}
async function load() {
  if (!Number.isInteger(targetID.value) || targetID.value <= 0) {
    errorMessage.value = t('runtimeTarget.detail.loadError');
    return;
  }
  loading.value = true;
  errorMessage.value = '';
  try {
    target.value = await getRuntimeTarget(targetID.value);
    if (canManageAssignments.value) await loadAssignments();
  } catch {
    errorMessage.value = t('runtimeTarget.detail.loadError');
  } finally {
    loading.value = false;
  }
}
async function loadAssignments() {
  assignmentsLoading.value = true;
  assignmentError.value = '';
  try {
    const assignments = await getRuntimeTargetAssignments(targetID.value);
    replaceVisibleAssignments(assignments.items, assignments.revision, true);
  } catch {
    assignmentError.value = t('runtimeTarget.detail.authorizedUsersLoadError');
  } finally {
    assignmentsLoading.value = false;
  }
}

function replaceVisibleAssignments(items: RuntimeTargetAssignment[], revision: number, resetRevoked: boolean) {
  activeAssignments.value = items;
  assignmentRevision.value = revision;
  if (resetRevoked) revokedAssignments.value = new Map();
  const knownIDs = new Set(assignmentOrder.value);
  const nextIDs = items.map((item) => item.user_id);
  assignmentOrder.value = resetRevoked
    ? nextIDs
    : [...assignmentOrder.value.filter((id) => knownIDs.has(id)), ...nextIDs.filter((id) => !knownIDs.has(id))];
  normalizeAssignmentPage();
}

function applyDialogAssignments(result: { items: RuntimeTargetAssignment[]; revision: number }) {
  const nextIDs = new Set(result.items.map((item) => item.user_id));
  const nextRevoked = new Map(revokedAssignments.value);
  activeAssignments.value.forEach((item) => {
    if (!nextIDs.has(item.user_id)) nextRevoked.set(item.user_id, item);
  });
  result.items.forEach((item) => nextRevoked.delete(item.user_id));
  revokedAssignments.value = nextRevoked;
  replaceVisibleAssignments(result.items, result.revision, false);
}

function normalizeAssignmentPage() {
  const lastPage = Math.max(1, Math.ceil(assignmentRows.value.length / assignmentPagination.pageSize));
  assignmentPagination.current = Math.min(assignmentPagination.current, lastPage);
}

function assignmentDisplay(row: AssignmentRow) {
  return row.display?.trim() || row.username?.trim() || t('runtimeTarget.detail.missingUser', { id: row.user_id });
}

function assignmentUsername(row: AssignmentRow) {
  return row.username?.trim() || t('runtimeTarget.detail.accountUnavailable');
}

function accountStatusLabel(row: AssignmentRow) {
  if (!row.username) return t('runtimeTarget.detail.accountUnavailable');
  if (row.status === 'enabled') return t('runtimeTarget.detail.accountEnabled');
  if (row.status === 'disabled') return t('runtimeTarget.detail.accountDisabled');
  return t('runtimeTarget.detail.accountStatusUnknown');
}

function accountStatusTheme(row: AssignmentRow) {
  if (!row.username) return 'default';
  if (row.status === 'enabled') return 'success';
  if (row.status === 'disabled') return 'danger';
  return 'warning';
}

async function toggleAssignment(row: AssignmentRow) {
  if (assignmentChangingUserId.value !== null) return;
  const activeUserIDs = activeAssignments.value.map((item) => item.user_id);
  const nextUserIDs =
    row.authorizationState === 'active'
      ? activeUserIDs.filter((userID) => userID !== row.user_id)
      : [...activeUserIDs, row.user_id];
  assignmentChangingUserId.value = row.user_id;
  assignmentError.value = '';
  try {
    const result = await replaceRuntimeTargetAssignments(targetID.value, nextUserIDs, assignmentRevision.value);
    const nextRevoked = new Map(revokedAssignments.value);
    if (row.authorizationState === 'active') nextRevoked.set(row.user_id, row);
    else nextRevoked.delete(row.user_id);
    revokedAssignments.value = nextRevoked;
    replaceVisibleAssignments(result.items, result.revision, false);
    MessagePlugin.success(
      t(
        row.authorizationState === 'active'
          ? 'runtimeTarget.detail.authorizationRevokeSuccess'
          : 'runtimeTarget.detail.authorizationRestoreSuccess',
      ),
    );
  } catch (error) {
    if (isApiRequestError(error) && error.status === 409) {
      await loadAssignments();
      assignmentError.value = t('runtimeTarget.detail.authorizationConflict');
    } else {
      assignmentError.value = t('runtimeTarget.detail.authorizationChangeError');
    }
  } finally {
    assignmentChangingUserId.value = null;
  }
}
async function refresh() {
  if (!Number.isInteger(targetID.value) || targetID.value <= 0) {
    errorMessage.value = t('runtimeTarget.detail.refreshError');
    return;
  }
  refreshing.value = true;
  errorMessage.value = '';
  try {
    target.value = await refreshRuntimeTarget(targetID.value);
  } catch {
    errorMessage.value = t('runtimeTarget.detail.refreshError');
  } finally {
    refreshing.value = false;
  }
}
onMounted(() => void load());
</script>
<style scoped lang="less">
.runtime-target-feedback {
  margin-bottom: var(--td-comp-margin-l);
}

.runtime-target-section {
  margin-top: var(--td-comp-margin-l);
}

.runtime-target-diagnostic {
  margin-top: var(--td-comp-margin-l);
}

.runtime-target-assignment-actions {
  align-items: center;
  display: flex;
  gap: var(--td-comp-margin-l);
  justify-content: space-between;
  margin-top: var(--td-comp-margin-l);
}

.runtime-target-assignment-actions span {
  color: var(--td-text-color-secondary);
}

.runtime-target-assignment-table {
  margin-top: var(--td-comp-margin-l);
}

.runtime-target-assignment-pagination {
  display: flex;
  justify-content: flex-end;
  margin-top: var(--td-comp-margin-l);
}

@media (width <= 640px) {
  .runtime-target-assignment-actions {
    align-items: stretch;
    flex-direction: column;
  }
}

.runtime-target-stat-grid {
  display: grid;
  gap: var(--td-comp-margin-l);
  grid-template-columns: repeat(auto-fit, minmax(160px, 1fr));
}

.runtime-target-stat-grid > div {
  color: var(--td-text-color-secondary);
}

.runtime-target-stat-grid p {
  color: var(--td-text-color-primary);
  font: var(--td-font-title-large);
  margin: var(--graft-density-gap-4) 0 0;
}
</style>
