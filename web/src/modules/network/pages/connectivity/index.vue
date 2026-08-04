<template>
  <section class="connectivity-page" data-page-type="overview-dashboard">
    <page-header
      :source="{
        labelKey: 'network.outbound.connectivity.eyebrow',
        fallback: t('network.outbound.connectivity.eyebrow'),
      }"
      title-key="network.outbound.connectivity.title"
      :title-fallback="t('network.outbound.connectivity.title')"
      description-key="network.outbound.connectivity.description"
      :description-fallback="t('network.outbound.connectivity.description')"
      compact-description-key="network.outbound.connectivity.compactDescription"
      :compact-description-fallback="t('network.outbound.connectivity.compactDescription')"
    >
      <template #actions>
        <t-space v-if="viewportVariant.density !== 'compact'">
          <t-button v-permission="NETWORK_PERMISSION.WRITE" variant="outline" @click="openOutboundPolicy">
            {{ t('network.outbound.connectivity.outboundPolicy') }}
          </t-button>
          <t-button
            v-permission="NETWORK_PERMISSION.MANAGE_TARGETS"
            variant="outline"
            @click="customDialogVisible = true"
          >
            {{ t('network.outbound.connectivity.addTarget') }}
          </t-button>
          <t-button theme="primary" :loading="store.running" @click="runAll">
            {{ t('network.outbound.connectivity.runAll') }}
          </t-button>
        </t-space>
        <t-space v-else class="connectivity-page__compact-actions">
          <t-button theme="primary" :loading="store.running" @click="runAll">
            {{ t('network.outbound.connectivity.runAll') }}
          </t-button>
          <t-dropdown placement="bottom-right" trigger="click">
            <t-button shape="square" variant="outline" :aria-label="t('network.outbound.connectivity.actions')">
              <template #icon><ellipsis-icon /></template>
            </t-button>
            <t-dropdown-menu>
              <t-dropdown-item v-permission="NETWORK_PERMISSION.MANAGE_TARGETS" @click="customDialogVisible = true">
                {{ t('network.outbound.connectivity.addTarget') }}
              </t-dropdown-item>
              <t-dropdown-item v-permission="NETWORK_PERMISSION.WRITE" @click="openOutboundPolicy">
                {{ t('network.outbound.connectivity.outboundPolicy') }}
              </t-dropdown-item>
              <t-dropdown-item>
                <t-switch v-model="autoRefresh" :label="[t('network.outbound.connectivity.autoRefresh'), '']" />
              </t-dropdown-item>
            </t-dropdown-menu>
          </t-dropdown>
        </t-space>
      </template>
    </page-header>

    <t-alert v-if="error" theme="error" :message="error" close @close="error = ''" />
    <t-loading :loading="store.loading">
      <section class="connectivity-page__snapshot" :aria-label="t('network.outbound.connectivity.snapshot')">
        <div v-if="isWideLayout" class="connectivity-page__summary">
          <t-card
            ><t-statistic
              :title="t('network.outbound.connectivity.healthy')"
              :value="store.aggregate?.healthy_count ?? 0"
          /></t-card>
          <t-card
            ><t-statistic
              :title="t('network.outbound.connectivity.degraded')"
              :value="store.aggregate?.degraded_count ?? 0"
          /></t-card>
          <t-card
            ><t-statistic
              :title="t('network.outbound.connectivity.failed')"
              :value="store.aggregate?.failed_count ?? 0"
          /></t-card>
          <t-card
            ><t-statistic
              :title="t('network.outbound.connectivity.average')"
              :value="store.aggregate?.average_latency_ms ?? 0"
              suffix="ms"
          /></t-card>
        </div>
        <div v-if="isWideLayout" class="connectivity-page__snapshot-meta">
          <span>{{ t('network.outbound.connectivity.lastRun') }}: {{ formatTime(store.aggregate?.last_run_at) }}</span>
          <span>{{ t('network.outbound.connectivity.successRate') }}: {{ successRate }}</span>
          <span>{{ t('network.outbound.connectivity.worst') }}: {{ worstTarget }}</span>
          <t-switch v-model="autoRefresh" :label="[t('network.outbound.connectivity.autoRefresh'), '']" />
        </div>
        <t-card v-else class="connectivity-page__hero">
          <template #header>
            <div>
              <p class="connectivity-page__hero-title">{{ connectionStatusLabel }}</p>
              <p class="connectivity-page__hero-explanation">{{ overallStatusExplanation }}</p>
            </div>
          </template>
          <template #actions>
            <t-tag :theme="statusTheme(overallStatus)" variant="light">{{ statusLabel(overallStatus) }}</t-tag>
          </template>
          <strong class="connectivity-page__hero-latency">
            {{ store.aggregate?.average_latency_ms ?? 0 }} <small>ms</small>
          </strong>
          <span class="connectivity-page__hero-latency-label">{{ t('network.outbound.connectivity.average') }}</span>
          <div class="connectivity-page__hero-details">
            <span
              >{{ t('network.outbound.connectivity.successRate') }} <strong>{{ successRate }}</strong></span
            >
            <span
              >{{ t('network.outbound.connectivity.lastRun') }}
              <strong>{{ formatTime(store.aggregate?.last_run_at) }}</strong></span
            >
          </div>
          <div v-if="trendSummary" class="connectivity-page__hero-trend">{{ trendSummary }}</div>
          <div class="connectivity-page__status-chips">
            <t-tag theme="success" variant="light"
              >{{ t('network.outbound.connectivity.healthy') }} {{ healthyCount }}</t-tag
            >
            <t-tag theme="warning" variant="light"
              >{{ t('network.outbound.connectivity.degraded') }} {{ degradedCount }}</t-tag
            >
            <t-tag theme="danger" variant="light"
              >{{ t('network.outbound.connectivity.failed') }} {{ failedCount }}</t-tag
            >
          </div>
        </t-card>
      </section>

      <div class="connectivity-page__toolbar">
        <t-select
          v-if="isWideLayout"
          v-model="sortBy"
          :options="sortOptions"
          :aria-label="t('network.outbound.connectivity.sortBy')"
        />
        <t-button v-else variant="outline" @click="sortDrawerVisible = true">
          {{ t('network.outbound.connectivity.sortBy') }}: {{ sortLabel }}
          <template #suffix><chevron-down-icon /></template>
        </t-button>
      </div>
      <t-table v-if="isWideLayout" row-key="id" :data="rows" :columns="columns" :hover="true" @row-click="openTarget">
        <template #target="{ row }">
          <t-link
            theme="primary"
            hover="color"
            :href="
              router.resolve({ name: 'PlatformNetworkConnectivityDiagnostics', params: { targetId: row.id } }).href
            "
            @click.stop.prevent="openTarget({ row })"
            @keydown.enter.stop.prevent="openTarget({ row })"
          >
            {{ targetTitle(row) }}
          </t-link>
        </template>
        <template #status="{ row }"
          ><t-tag :theme="statusTheme(row.status)" variant="light">{{ statusLabel(row.status) }}</t-tag></template
        >
        <template #latency="{ row }">{{ row.latency_ms === undefined ? '-' : `${row.latency_ms} ms` }}</template>
        <template #httpStatus="{ row }">{{
          row.http_status ?? t('network.outbound.connectivity.unavailable')
        }}</template>
        <template #checked="{ row }">{{ formatTime(row.checked_at) }}</template>
        <template #actions="{ row }">
          <t-popconfirm
            v-if="isCustomTarget(row.id)"
            v-permission="NETWORK_PERMISSION.MANAGE_TARGETS"
            theme="danger"
            :content="t('network.outbound.connectivity.deleteConfirm', { target: targetTitle(row) })"
            :confirm-btn="t('network.outbound.connectivity.delete')"
            :cancel-btn="t('network.outbound.connectivity.cancel')"
            @confirm="removeTarget(row.id)"
          >
            <t-button theme="danger" variant="text" size="small" @click.stop>{{
              t('network.outbound.connectivity.delete')
            }}</t-button>
          </t-popconfirm>
        </template>
      </t-table>
      <div v-else class="connectivity-page__target-list">
        <t-card
          v-for="row in rows"
          :key="row.id"
          class="connectivity-page__target-card"
          role="link"
          tabindex="0"
          @click="openTarget({ row })"
          @keydown.enter.prevent="openTarget({ row })"
          @keydown.space.prevent="openTarget({ row })"
        >
          <template #header>
            <div class="connectivity-page__target-heading">
              <component :is="capabilityIcon(row)" class="connectivity-page__capability-icon" />
              <div>
                <strong>{{ targetTitle(row) }}</strong>
              </div>
            </div>
          </template>
          <template #actions>
            <div @click.stop @keydown.stop>
              <t-dropdown placement="bottom-right" trigger="click">
                <t-button
                  shape="square"
                  size="small"
                  variant="text"
                  :aria-label="t('network.outbound.connectivity.actions')"
                >
                  <template #icon><ellipsis-icon /></template>
                </t-button>
                <t-dropdown-menu>
                  <t-dropdown-item @click="openTarget({ row })">{{
                    t('network.outbound.connectivity.run')
                  }}</t-dropdown-item>
                  <t-popconfirm
                    v-if="isCustomTarget(row.id)"
                    v-permission="NETWORK_PERMISSION.MANAGE_TARGETS"
                    theme="danger"
                    :content="t('network.outbound.connectivity.deleteConfirm', { target: targetTitle(row) })"
                    :confirm-btn="t('network.outbound.connectivity.delete')"
                    :cancel-btn="t('network.outbound.connectivity.cancel')"
                    @confirm="removeTarget(row.id)"
                  >
                    <t-dropdown-item theme="error" @click.stop>{{
                      t('network.outbound.connectivity.delete')
                    }}</t-dropdown-item>
                  </t-popconfirm>
                </t-dropdown-menu>
              </t-dropdown>
            </div>
          </template>
          <t-tag :theme="statusTheme(row.status)" variant="light">{{ statusLabel(row.status) }}</t-tag>
          <dl class="connectivity-page__target-details">
            <div>
              <dt>{{ t('network.outbound.connectivity.latency') }}</dt>
              <dd>{{ row.latency_ms === undefined ? '-' : `${row.latency_ms} ms` }}</dd>
            </div>
            <div>
              <dt>{{ t('network.outbound.connectivity.httpStatus') }}</dt>
              <dd>{{ row.http_status ?? t('network.outbound.connectivity.unavailable') }}</dd>
            </div>
            <div class="connectivity-page__target-checked">
              <dt>{{ t('network.outbound.connectivity.lastChecked') }}</dt>
              <dd>{{ formatTime(row.checked_at) }}</dd>
            </div>
          </dl>
        </t-card>
      </div>
      <t-empty v-if="!rows.length && !store.loading" :description="t('network.outbound.connectivity.noTargets')" />
    </t-loading>

    <t-drawer
      v-model:visible="sortDrawerVisible"
      :header="t('network.outbound.connectivity.sortBy')"
      :footer="false"
      placement="bottom"
      size="auto"
    >
      <t-radio-group
        v-model="sortBy"
        direction="vertical"
        class="connectivity-page__sort-options"
        @change="sortDrawerVisible = false"
      >
        <t-radio v-for="option in sortOptions" :key="option.value" :value="option.value">{{ option.label }}</t-radio>
      </t-radio-group>
    </t-drawer>

    <responsive-dialog
      :visible="customDialogVisible"
      :title="t('network.outbound.connectivity.addTarget')"
      :close-label="t('network.outbound.connectivity.cancel')"
      purpose="form"
      size="medium"
      @update:visible="customDialogVisible = $event"
    >
      <t-alert theme="info" :message="t('network.outbound.connectivity.customTargetHint')" />
      <t-form :data="customTargetForm" label-align="top" class="connectivity-page__target-form">
        <t-form-item :label="t('network.outbound.connectivity.targetId')" name="target_id">
          <t-input v-model="customTargetForm.target_id" placeholder="custom-status" />
        </t-form-item>
        <t-form-item :label="t('network.outbound.connectivity.displayName')" name="display_name">
          <t-input v-model="customTargetForm.display_name" />
        </t-form-item>
        <t-form-item :label="t('network.outbound.connectivity.url')" name="endpoint">
          <t-input v-model="customTargetForm.endpoint" placeholder="https://example.com/health" />
        </t-form-item>
      </t-form>
      <template #footer>
        <t-space>
          <t-button variant="outline" @click="customDialogVisible = false">{{
            t('network.outbound.connectivity.cancel')
          }}</t-button>
          <t-button theme="primary" :loading="creatingTarget" @click="createTarget">{{
            t('network.outbound.connectivity.add')
          }}</t-button>
        </t-space>
      </template>
    </responsive-dialog>
  </section>
</template>
<script setup lang="ts">
import {
  ChevronDownIcon,
  CloudIcon,
  CloudUploadIcon,
  EllipsisIcon,
  LinkIcon,
  LogoGithubIcon,
} from 'tdesign-icons-vue-next';
import { MessagePlugin } from 'tdesign-vue-next/es/message';
import { computed, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import { useRouter } from 'vue-router';

import { PageHeader } from '@/shared/components/page';
import ResponsiveDialog from '@/shared/components/responsive/ResponsiveDialog.vue';
import { useViewportResponsiveVariant } from '@/shared/composables';
import { formatLocaleDateTime } from '@/shared/observability';

import type { ConnectivityCheck, ConnectivityTarget } from '../../api/connectivity';
import { useConnectivityStore } from '../../store/connectivity';

/** 批量健康页只消费共享 ConnectivityStore 的摘要投影；目标详情和完整诊断报告始终由 target 路由承载。 */
const NETWORK_PERMISSION = {
  WRITE: 'platform-network.write',
  MANAGE_TARGETS: 'platform-network.targets.manage',
} as const;
const AUTO_REFRESH_INTERVAL_MS = 30_000;

type ConnectivityTargetProjection = ConnectivityTarget & Partial<ConnectivityCheck>;
type SortMode = 'recent' | 'latency' | 'status';

const { t, te, locale } = useI18n();
const router = useRouter();
const store = useConnectivityStore();
const autoRefresh = ref(false);
const customDialogVisible = ref(false);
const creatingTarget = ref(false);
const error = ref('');
const sortBy = ref<SortMode>('latency');
const sortDrawerVisible = ref(false);
const customTargetForm = reactive({ target_id: '', display_name: '', endpoint: '' });
let autoRefreshTimer: ReturnType<typeof setInterval> | undefined;
const viewportVariant = useViewportResponsiveVariant({ presentation: 'entity' });
const isWideLayout = computed(() => viewportVariant.value.density === 'spacious');

/** 表格与卡片共同消费此投影，避免窄屏页面形成第二份 Connectivity 业务模型。 */
const rows = computed<ConnectivityTargetProjection[]>(() => {
  const latestByTarget = new Map(store.latest.map((check) => [check.target_id, check]));
  const result = store.targets.map((target) => ({ ...target, ...(latestByTarget.get(target.id) ?? {}) }));
  return result.sort((left, right) => {
    if (sortBy.value === 'status')
      return (
        statusOrder(left.status) - statusOrder(right.status) || targetTitle(left).localeCompare(targetTitle(right))
      );
    if (sortBy.value === 'recent') return Date.parse(right.checked_at ?? '') - Date.parse(left.checked_at ?? '');
    return (left.latency_ms ?? Number.MAX_SAFE_INTEGER) - (right.latency_ms ?? Number.MAX_SAFE_INTEGER);
  });
});
const columns = computed(() => [
  { colKey: 'target', title: t('network.outbound.connectivity.target') },
  { colKey: 'status', title: t('network.outbound.connectivity.status') },
  { colKey: 'latency', title: t('network.outbound.connectivity.latency') },
  { colKey: 'httpStatus', title: t('network.outbound.connectivity.httpStatus') },
  { colKey: 'checked', title: t('network.outbound.connectivity.lastChecked') },
  { colKey: 'actions', title: t('network.outbound.connectivity.actions'), width: 96 },
]);
const sortOptions = computed(() => [
  { label: t('network.outbound.connectivity.sortRecent'), value: 'recent' },
  { label: t('network.outbound.connectivity.sortLatency'), value: 'latency' },
  { label: t('network.outbound.connectivity.sortStatus'), value: 'status' },
]);
const sortLabel = computed(() => sortOptions.value.find((option) => option.value === sortBy.value)?.label ?? '');
const healthyCount = computed(() => store.aggregate?.healthy_count ?? 0);
const degradedCount = computed(() => store.aggregate?.degraded_count ?? 0);
const failedCount = computed(() => store.aggregate?.failed_count ?? 0);
const targetCount = computed(() => store.aggregate?.target_count ?? rows.value.length);
const overallStatus = computed<ConnectivityCheck['status'] | undefined>(() => {
  if (!targetCount.value || (!healthyCount.value && !degradedCount.value && !failedCount.value)) return undefined;
  if (failedCount.value) return 'failed';
  if (degradedCount.value) return 'degraded';
  return 'healthy';
});
const connectionStatusLabel = computed(() => t('network.outbound.connectivity.connectionStatus'));
const overallStatusExplanation = computed(() => {
  if (!targetCount.value || !overallStatus.value) return t('network.outbound.connectivity.noCheckResults');
  if (healthyCount.value === targetCount.value) return t('network.outbound.connectivity.allTargetsNormal');
  if (failedCount.value) return t('network.outbound.connectivity.targetsFailed', { count: failedCount.value });
  return t('network.outbound.connectivity.targetsNormal', { count: healthyCount.value, total: targetCount.value });
});
// 当前 aggregate 不提供可比较的历史窗口；趋势区域在真实指标可用前保持不渲染。
const trendSummary = computed<string | null>(() => null);
const successRate = computed(() => {
  const total = store.aggregate?.target_count ?? 0;
  return total ? `${Math.round(((store.aggregate?.healthy_count ?? 0) / total) * 100)}%` : '-';
});
const worstTarget = computed(() => {
  const targetId = store.aggregate?.worst_target_id;
  if (!targetId) return '-';
  const target = store.targets.find((item) => item.id === targetId);
  return target
    ? `${targetTitle(target)} (${store.aggregate?.worst_latency_ms ?? 0} ms)`
    : `${targetId} (${store.aggregate?.worst_latency_ms ?? 0} ms)`;
});

function targetTitle(target: ConnectivityTarget) {
  const customTarget = store.customTargets.find((item) => item.id === target.id);
  if (customTarget) return customTarget.display_name;
  return te(target.title_key) ? t(target.title_key) : target.title_key;
}

function formatTime(value?: string | null) {
  return formatLocaleDateTime(value, locale.value);
}

function statusOrder(status?: ConnectivityCheck['status']) {
  return status === 'failed' ? 0 : status === 'degraded' ? 1 : status === 'healthy' ? 2 : 3;
}

function statusTheme(status?: ConnectivityCheck['status']) {
  return status === 'healthy'
    ? 'success'
    : status === 'degraded'
      ? 'warning'
      : status === 'failed'
        ? 'danger'
        : 'default';
}

function statusLabel(status?: ConnectivityCheck['status']) {
  return status ? t(`network.outbound.connectivity.statuses.${status}`) : t('network.outbound.connectivity.notChecked');
}

function isCustomTarget(targetId: string) {
  return store.customTargets.some((target) => target.id === targetId);
}

function openTarget({ row }: { row: unknown }) {
  const target = row as ConnectivityTargetProjection;
  void router.push({ name: 'PlatformNetworkConnectivityDiagnostics', params: { targetId: target.id } });
}

function capabilityIcon(target: ConnectivityTargetProjection) {
  if (target.category === 'git') return LogoGithubIcon;
  if (target.category === 'oci') return CloudUploadIcon;
  if (target.category === 'platform') return CloudIcon;
  return LinkIcon;
}

function openOutboundPolicy() {
  void router.push({ name: 'PlatformNetworkOutbound' });
}

async function runAll() {
  error.value = '';
  try {
    await store.runAll();
  } catch (value) {
    error.value = String(value);
  }
}

async function createTarget() {
  const targetId = customTargetForm.target_id.trim();
  const displayName = customTargetForm.display_name.trim();
  const endpoint = customTargetForm.endpoint.trim();
  if (!/^custom-[a-z0-9][a-z0-9-]{0,120}$/.test(targetId) || !displayName || !endpoint) {
    MessagePlugin.warning(t('network.outbound.connectivity.targetValidation'));
    return;
  }
  creatingTarget.value = true;
  try {
    await store.createCustomTarget({ target_id: targetId, display_name: displayName, endpoint });
    await store.refresh();
    customDialogVisible.value = false;
    customTargetForm.target_id = '';
    customTargetForm.display_name = '';
    customTargetForm.endpoint = '';
    MessagePlugin.success(t('network.outbound.connectivity.targetAdded'));
  } catch (value) {
    error.value = String(value);
  } finally {
    creatingTarget.value = false;
  }
}

async function removeTarget(targetId: string) {
  try {
    await store.deleteCustomTarget(targetId);
    await store.refresh();
    MessagePlugin.success(t('network.outbound.connectivity.targetDeleted'));
  } catch (value) {
    error.value = String(value);
  }
}

function startAutoRefresh() {
  stopAutoRefresh();
  autoRefreshTimer = setInterval(
    () => void store.refresh().catch((value) => (error.value = String(value))),
    AUTO_REFRESH_INTERVAL_MS,
  );
}

function stopAutoRefresh() {
  if (autoRefreshTimer) clearInterval(autoRefreshTimer);
  autoRefreshTimer = undefined;
}

watch(autoRefresh, (enabled) => (enabled ? startAutoRefresh() : stopAutoRefresh()));
onMounted(() => void store.refresh());
onBeforeUnmount(stopAutoRefresh);
</script>
<style scoped>
.connectivity-page__snapshot {
  margin-bottom: calc(16px * var(--graft-theme-density-scale));
}

.connectivity-page__summary {
  display: grid;
  gap: calc(16px * var(--graft-theme-density-scale));
  grid-template-columns: repeat(4, minmax(0, 1fr));
}

.connectivity-page__snapshot-meta {
  align-items: center;
  color: var(--td-text-color-secondary);
  display: flex;
  flex-wrap: wrap;
  font-size: var(--td-font-size-body-small);
  gap: calc(12px * var(--graft-theme-density-scale)) calc(24px * var(--graft-theme-density-scale));
  margin-top: calc(12px * var(--graft-theme-density-scale));
}

.connectivity-page__toolbar {
  display: flex;
  justify-content: flex-end;
  margin-bottom: calc(12px * var(--graft-theme-density-scale));
}

.connectivity-page__toolbar :deep(.t-select) {
  width: 180px;
}

.connectivity-page__compact-actions {
  align-items: stretch;
}

.connectivity-page__hero-title,
.connectivity-page__hero-explanation {
  margin: 0;
}

.connectivity-page__hero-title {
  color: var(--td-text-color-primary);
  font: var(--td-font-title-small);
}

.connectivity-page__hero-explanation,
.connectivity-page__hero-latency-label,
.connectivity-page__target-heading span,
.connectivity-page__target-details dt {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
}

.connectivity-page__hero-explanation {
  margin-top: var(--td-comp-margin-xxs);
}

.connectivity-page__hero-latency {
  color: var(--td-text-color-primary);
  display: block;
  font: var(--td-font-title-large);
}

.connectivity-page__hero-latency small {
  font: var(--td-font-body-medium);
}

.connectivity-page__hero-latency-label {
  display: block;
  margin-top: var(--td-comp-margin-xxs);
}

.connectivity-page__hero-details {
  display: grid;
  gap: var(--td-comp-margin-s);
  grid-template-columns: repeat(2, minmax(0, 1fr));
  margin-top: var(--td-comp-margin-l);
}

.connectivity-page__hero-details span {
  color: var(--td-text-color-secondary);
  display: grid;
  font: var(--td-font-body-small);
  gap: var(--td-comp-margin-xxs);
}

.connectivity-page__hero-details strong {
  color: var(--td-text-color-primary);
  font: var(--td-font-body-medium);
}

.connectivity-page__hero-trend {
  color: var(--td-success-color);
  font: var(--td-font-body-small);
  margin-top: var(--td-comp-margin-l);
}

.connectivity-page__status-chips {
  display: flex;
  flex-wrap: wrap;
  gap: var(--td-comp-margin-s);
  margin-top: var(--td-comp-margin-l);
}

.connectivity-page__target-list {
  display: grid;
  gap: var(--td-comp-margin-m);
}

.connectivity-page__target-card {
  cursor: pointer;
}

.connectivity-page__target-card:focus-visible {
  outline: 2px solid var(--td-brand-color);
  outline-offset: 2px;
}

.connectivity-page__target-heading {
  align-items: center;
  display: flex;
  gap: var(--td-comp-margin-s);
  min-width: 0;
}

.connectivity-page__target-heading > div {
  display: grid;
  gap: var(--td-comp-margin-xxs);
  min-width: 0;
}

.connectivity-page__target-heading strong,
.connectivity-page__target-heading span {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.connectivity-page__capability-icon {
  color: var(--td-brand-color);
  font-size: var(--td-font-size-title-large);
}

.connectivity-page__target-details {
  display: grid;
  gap: var(--td-comp-margin-m);
  grid-template-columns: repeat(2, minmax(0, 1fr));
  margin: var(--td-comp-margin-m) 0 0;
}

.connectivity-page__target-details div {
  display: grid;
  gap: var(--td-comp-margin-xxs);
}

.connectivity-page__target-details dd {
  color: var(--td-text-color-primary);
  font: var(--td-font-body-medium);
  margin: 0;
  overflow-wrap: anywhere;
}

.connectivity-page__target-details .connectivity-page__target-checked {
  grid-column: 1 / -1;
}

.connectivity-page__sort-options {
  display: grid;
  gap: var(--td-comp-margin-m);
}

.connectivity-page__target-form {
  margin-top: calc(16px * var(--graft-theme-density-scale));
}

@media (width < 768px) {
  .connectivity-page__snapshot {
    margin-bottom: var(--td-comp-margin-m);
  }

  .connectivity-page__toolbar {
    justify-content: flex-start;
  }

  .connectivity-page__hero :deep(.t-card__body),
  .connectivity-page__target-card :deep(.t-card__body) {
    padding: var(--td-comp-paddingTB-l) var(--td-comp-paddingLR-l);
  }
}
</style>
