<template>
  <section class="diagnostics-page">
    <page-header
      :source="{
        labelKey: 'network.outbound.connectivity.eyebrow',
        fallback: t('network.outbound.connectivity.eyebrow'),
      }"
      :title-key="'network.outbound.connectivity.diagnosticsTitle'"
      :title-fallback="targetId"
      :description-key="'network.outbound.connectivity.diagnosticsDescription'"
      :description-fallback="t('network.outbound.connectivity.diagnosticsDescription')"
    >
      <template #actions
        ><t-button theme="primary" :loading="store.running" @click="run">{{
          t('network.outbound.connectivity.run')
        }}</t-button></template
      >
    </page-header>
    <t-alert v-if="error" theme="error" :message="error" />
    <t-loading :loading="loading">
      <t-card :title="t('network.outbound.connectivity.probes')"
        ><t-table row-key="kind" :data="report?.probes ?? []" :columns="probeColumns"
      /></t-card>
      <div class="diagnostics-page__details">
        <t-card :title="t('network.outbound.connectivity.route')"
          ><t-descriptions v-if="report?.route" :column="1"
            ><t-descriptions-item :label="t('network.outbound.connectivity.strategy')">{{
              report.route.matched_strategy
            }}</t-descriptions-item
            ><t-descriptions-item :label="t('network.outbound.connectivity.decision')">{{
              report.route.decision
            }}</t-descriptions-item
            ><t-descriptions-item :label="t('network.outbound.connectivity.reason')">{{
              report.route.reason
            }}</t-descriptions-item></t-descriptions
          ><t-empty v-else
        /></t-card>
        <t-card :title="t('network.outbound.connectivity.history')"
          ><t-table row-key="check_id" :data="history" :columns="historyColumns" @row-click="selectReport"
        /></t-card>
      </div>
    </t-loading>
  </section>
</template>
<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import { useRoute } from 'vue-router';

import { PageHeader } from '@/shared/components/page';

import { useConnectivityStore } from '../../store/connectivity';
const { t, locale } = useI18n();
const route = useRoute();
const store = useConnectivityStore();
const loading = ref(false);
const error = ref('');
const report = ref<any>();
const history = ref<any[]>([]);
const targetId = computed(() => String(route.params.targetId ?? ''));
const probeColumns = computed(() => [
  { colKey: 'kind', title: t('network.outbound.connectivity.probe') },
  { colKey: 'status', title: t('network.outbound.connectivity.status') },
  {
    colKey: 'duration_ms',
    title: t('network.outbound.connectivity.latency'),
    cell: ({ row }: any) => `${row.duration_ms} ms`,
  },
  { colKey: 'summary', title: t('network.outbound.connectivity.result') },
]);
const historyColumns = computed(() => [
  {
    colKey: 'checked_at',
    title: t('network.outbound.connectivity.lastChecked'),
    cell: ({ row }: any) => new Date(row.checked_at).toLocaleString(locale.value),
  },
  { colKey: 'status', title: t('network.outbound.connectivity.status') },
  {
    colKey: 'latency_ms',
    title: t('network.outbound.connectivity.latency'),
    cell: ({ row }: any) => `${row.latency_ms} ms`,
  },
]);
async function load() {
  loading.value = true;
  error.value = '';
  try {
    history.value = await store.loadHistory(targetId.value);
    if (history.value[0]) report.value = await store.loadReport(targetId.value, history.value[0].check_id);
  } catch (value) {
    error.value = String(value);
  } finally {
    loading.value = false;
  }
}
async function run() {
  try {
    const result = await store.runTarget(targetId.value);
    report.value = result.report;
    await load();
  } catch (value) {
    error.value = String(value);
  }
}
async function selectReport({ row }: any) {
  report.value = await store.loadReport(targetId.value, row.check_id);
}
watch(targetId, () => void load());
onMounted(() => void load());
</script>
<style scoped>
.diagnostics-page__details {
  display: grid;
  gap: calc(16px * var(--graft-theme-density-scale));
  grid-template-columns: minmax(0, 1fr) minmax(0, 1fr);
  margin-top: calc(16px * var(--graft-theme-density-scale));
}

@media (width <= 800px) {
  .diagnostics-page__details {
    grid-template-columns: 1fr;
  }
}
</style>
