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
    >
      <template #actions>
        <t-space>
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
      </template>
    </page-header>

    <t-alert v-if="error" theme="error" :message="error" close @close="error = ''" />
    <t-loading :loading="store.loading">
      <section class="connectivity-page__snapshot" :aria-label="t('network.outbound.connectivity.snapshot')">
        <div class="connectivity-page__summary">
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
        <div class="connectivity-page__snapshot-meta">
          <span>{{ t('network.outbound.connectivity.lastRun') }}: {{ formatTime(store.aggregate?.last_run_at) }}</span>
          <span>{{ t('network.outbound.connectivity.successRate') }}: {{ successRate }}</span>
          <span>{{ t('network.outbound.connectivity.worst') }}: {{ worstTarget }}</span>
          <t-switch v-model="autoRefresh" :label="[t('network.outbound.connectivity.autoRefresh'), '']" />
        </div>
      </section>

      <div class="connectivity-page__toolbar">
        <t-select v-model="sortBy" :options="sortOptions" :aria-label="t('network.outbound.connectivity.sortBy')" />
      </div>
      <t-table row-key="id" :data="rows" :columns="columns" :hover="true" @row-click="openTarget">
        <template #target="{ row }"
          ><strong>{{ targetTitle(row) }}</strong></template
        >
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
      <t-empty v-if="!rows.length && !store.loading" :description="t('network.outbound.connectivity.noTargets')" />
    </t-loading>

    <t-dialog
      v-model:visible="customDialogVisible"
      :header="t('network.outbound.connectivity.addTarget')"
      :confirm-btn="t('network.outbound.connectivity.add')"
      :cancel-btn="t('network.outbound.connectivity.cancel')"
      :confirm-loading="creatingTarget"
      width="520px"
      destroy-on-close
      @confirm="createTarget"
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
    </t-dialog>
  </section>
</template>
<script setup lang="ts">
import { MessagePlugin } from 'tdesign-vue-next/es/message';
import { computed, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import { useRouter } from 'vue-router';

import { PageHeader } from '@/shared/components/page';
import { formatLocaleDateTime } from '@/shared/observability';

import type { ConnectivityCheck, ConnectivityTarget } from '../../api/connectivity';
import { useConnectivityStore } from '../../store/connectivity';

/** 批量健康页只消费共享 ConnectivityStore 的摘要投影；目标详情和完整诊断报告始终由 target 路由承载。 */
const NETWORK_PERMISSION = {
  WRITE: 'platform-network.write',
  MANAGE_TARGETS: 'platform-network.targets.manage',
} as const;
const AUTO_REFRESH_INTERVAL_MS = 30_000;

type ConnectivityRow = ConnectivityTarget & Partial<ConnectivityCheck>;
type SortMode = 'latency' | 'status';

const { t, te, locale } = useI18n();
const router = useRouter();
const store = useConnectivityStore();
const autoRefresh = ref(false);
const customDialogVisible = ref(false);
const creatingTarget = ref(false);
const error = ref('');
const sortBy = ref<SortMode>('latency');
const customTargetForm = reactive({ target_id: '', display_name: '', endpoint: '' });
let autoRefreshTimer: ReturnType<typeof setInterval> | undefined;

const rows = computed<ConnectivityRow[]>(() => {
  const latestByTarget = new Map(store.latest.map((check) => [check.target_id, check]));
  const result = store.targets.map((target) => ({ ...target, ...(latestByTarget.get(target.id) ?? {}) }));
  return result.sort((left, right) => {
    if (sortBy.value === 'status')
      return (
        statusOrder(left.status) - statusOrder(right.status) || targetTitle(left).localeCompare(targetTitle(right))
      );
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
  { label: t('network.outbound.connectivity.sortLatency'), value: 'latency' },
  { label: t('network.outbound.connectivity.sortStatus'), value: 'status' },
]);
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
  const target = row as ConnectivityRow;
  void router.push({ name: 'PlatformNetworkConnectivityDiagnostics', params: { targetId: target.id } });
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
  autoRefreshTimer = setInterval(() => void runAll(), AUTO_REFRESH_INTERVAL_MS);
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

.connectivity-page__target-form {
  margin-top: calc(16px * var(--graft-theme-density-scale));
}

@media (width <= 800px) {
  .connectivity-page__summary {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}
</style>
