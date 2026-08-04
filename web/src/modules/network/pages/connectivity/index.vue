<template>
  <section class="connectivity-page">
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
      <template #actions
        ><t-button theme="primary" :loading="store.running" @click="store.runAll">{{
          t('network.outbound.connectivity.runAll')
        }}</t-button></template
      >
    </page-header>
    <t-loading :loading="store.loading">
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
          ><t-statistic :title="t('network.outbound.connectivity.failed')" :value="store.aggregate?.failed_count ?? 0"
        /></t-card>
        <t-card
          ><t-statistic
            :title="t('network.outbound.connectivity.average')"
            :value="store.aggregate?.average_latency_ms ?? 0"
            suffix="ms"
        /></t-card>
      </div>
      <t-table row-key="id" :data="rows" :columns="columns" :hover="true" @row-click="openTarget" />
    </t-loading>
  </section>
</template>
<script setup lang="ts">
import { computed, onMounted } from 'vue';
import { useI18n } from 'vue-i18n';
import { useRouter } from 'vue-router';

import { PageHeader } from '@/shared/components/page';

import { useConnectivityStore } from '../../store/connectivity';
const { t, locale } = useI18n();
const router = useRouter();
const store = useConnectivityStore();
const rows = computed(() =>
  store.targets.map((target) => ({
    ...target,
    ...(store.latest.find((check) => check.target_id === target.id) ?? {}),
  })),
);
const columns = computed(() => [
  { colKey: 'title_key', title: t('network.outbound.connectivity.target') },
  { colKey: 'status', title: t('network.outbound.connectivity.status'), cell: ({ row }: any) => row.status ?? '-' },
  {
    colKey: 'latency_ms',
    title: t('network.outbound.connectivity.latency'),
    cell: ({ row }: any) => (row.latency_ms === null || row.latency_ms === undefined ? '-' : `${row.latency_ms} ms`),
  },
  {
    colKey: 'checked_at',
    title: t('network.outbound.connectivity.lastChecked'),
    cell: ({ row }: any) => (row.checked_at ? new Date(row.checked_at).toLocaleString(locale.value) : '-'),
  },
]);
function openTarget({ row }: any) {
  void router.push({ name: 'PlatformNetworkConnectivityDiagnostics', params: { targetId: row.id } });
}
onMounted(() => void store.refresh());
</script>
<style scoped>
.connectivity-page__summary {
  display: grid;
  gap: calc(16px * var(--graft-theme-density-scale));
  grid-template-columns: repeat(4, minmax(0, 1fr));
  margin-bottom: calc(16px * var(--graft-theme-density-scale));
}

@media (width <= 800px) {
  .connectivity-page__summary {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}
</style>
