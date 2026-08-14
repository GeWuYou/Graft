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
          <t-select
            v-model="selectedUserIds"
            multiple
            filterable
            clearable
            :loading="assignmentsLoading || candidatesLoading"
            :options="assignmentOptions"
            :placeholder="t('runtimeTarget.detail.authorizedUsersPlaceholder')"
            class="runtime-target-assignment-select"
          />
          <div class="runtime-target-assignment-actions">
            <span>{{ t('runtimeTarget.detail.authorizedUsersHint') }}</span>
            <t-button theme="primary" :loading="assignmentsSaving" @click="saveAssignments">
              {{ t('runtimeTarget.detail.saveAuthorizedUsers') }}
            </t-button>
          </div>
        </t-card>
      </template>
      <t-empty v-else :title="t('runtimeTarget.detail.notFound')" />
    </management-page-content>
  </div>
</template>
<script setup lang="ts">
// 详情页展示 provider-owned 运行时投影；刷新只请求后端重新探测，不在前端改写运行时事实。
import { RefreshIcon } from 'tdesign-icons-vue-next';
import { computed, defineComponent, h, onMounted, type PropType, ref } from 'vue';
import { useI18n } from 'vue-i18n';
import { useRoute } from 'vue-router';

import { useLocale } from '@/locales/useLocale';
import { ManagementPageContent, ManagementPageHeader } from '@/shared/components/management';
import { formatBytes, formatLocaleDateTime } from '@/shared/observability';
import { getPermissionStore } from '@/store/modules/permission';

import {
  getRuntimeTarget,
  getRuntimeTargetAssignmentCandidates,
  getRuntimeTargetAssignments,
  refreshRuntimeTarget,
  replaceRuntimeTargetAssignments,
  type RuntimeTargetDetail,
  type RuntimeTargetUsageMetric,
} from '../../api/runtime-target';

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
const assignmentOptions = ref<Array<{ label: string; value: number }>>([]);
const selectedUserIds = ref<number[]>([]);
const assignmentsLoading = ref(false);
const candidatesLoading = ref(false);
const assignmentsSaving = ref(false);
const assignmentError = ref('');
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
  candidatesLoading.value = true;
  assignmentError.value = '';
  try {
    const [assignments, candidates] = await Promise.all([
      getRuntimeTargetAssignments(targetID.value),
      getRuntimeTargetAssignmentCandidates(targetID.value, { limit: 100, offset: 0 }),
    ]);
    const optionMap = new Map(candidates.items.map((item) => [item.id, item]));
    assignments.forEach((item) => {
      if (!optionMap.has(item.user_id))
        optionMap.set(item.user_id, {
          id: item.user_id,
          username: String(item.user_id),
          display: String(item.user_id),
          status: 'active',
        });
    });
    assignmentOptions.value = [...optionMap.values()].map((item) => ({
      label: item.display || item.username,
      value: item.id,
    }));
    selectedUserIds.value = assignments.map((item) => item.user_id);
  } catch {
    assignmentError.value = t('runtimeTarget.detail.authorizedUsersLoadError');
  } finally {
    assignmentsLoading.value = false;
    candidatesLoading.value = false;
  }
}
async function saveAssignments() {
  assignmentsSaving.value = true;
  assignmentError.value = '';
  try {
    await replaceRuntimeTargetAssignments(targetID.value, selectedUserIds.value);
  } catch {
    assignmentError.value = t('runtimeTarget.detail.authorizedUsersSaveError');
  } finally {
    assignmentsSaving.value = false;
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

.runtime-target-assignment-select {
  width: 100%;
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
