<template>
  <div class="runtime-target-page" data-page-type="list-form-detail">
    <management-page-content>
      <management-page-header :title="t('runtimeTarget.list.title')" :description="t('runtimeTarget.list.description')">
        <template #actions>
          <t-button
            theme="default"
            variant="outline"
            :loading="discovering"
            data-testid="runtime-target-discover-local"
            @click="discoverLocal"
          >
            <template #icon><search-icon /></template>
            {{ t('runtimeTarget.list.discoverLocal') }}
          </t-button>
          <t-button theme="primary" variant="outline" :loading="loading" @click="load">
            <template #icon><refresh-icon /></template>
            {{ t('runtimeTarget.list.reload') }}
          </t-button>
        </template>
      </management-page-header>

      <t-loading :loading="loading" size="large">
        <t-row v-if="items.length" :gutter="[16, 16]" class="runtime-target-page__grid">
          <t-col v-for="target in items" :key="target.id" :xs="12" :md="6" :xl="4">
            <article class="runtime-target-card" :data-testid="`runtime-target-card-${target.id}`">
              <header class="runtime-target-card__header">
                <div class="runtime-target-card__identity">
                  <div class="runtime-target-card__title-row">
                    <h2>{{ target.displayName }}</h2>
                    <t-tag :theme="target.availability ? 'success' : 'danger'" variant="light" size="small">
                      {{ availabilityText(target.availability) }}
                    </t-tag>
                  </div>
                  <p>{{ target.endpointLabel }}</p>
                </div>
                <t-tooltip :content="t('runtimeTarget.list.refresh')">
                  <t-button
                    theme="default"
                    variant="text"
                    shape="square"
                    :loading="refreshingId === target.id"
                    @click="refreshTarget(target.id)"
                  >
                    <template #icon><refresh-icon /></template>
                  </t-button>
                </t-tooltip>
              </header>

              <div class="runtime-target-card__counts">
                <section>
                  <span>{{ t('runtimeTarget.metrics.containers') }}</span>
                  <strong>{{ target.summary?.containers.total ?? '-' }}</strong>
                  <small v-if="target.summary">
                    {{ t('runtimeTarget.metrics.running') }} {{ target.summary.containers.running }} ·
                    {{ t('runtimeTarget.metrics.stopped') }} {{ target.summary.containers.stopped }}
                  </small>
                </section>
                <section>
                  <span>{{ t('runtimeTarget.metrics.images') }}</span>
                  <strong>{{ target.summary?.images.total ?? '-' }}</strong>
                  <small v-if="target.summary">
                    {{ t('runtimeTarget.metrics.used') }} {{ target.summary.images.used }} ·
                    {{ t('runtimeTarget.metrics.unused') }} {{ target.summary.images.unused }}
                  </small>
                </section>
              </div>

              <div class="runtime-target-card__metrics">
                <metric-progress :label="t('runtimeTarget.metrics.cpu')" :metric="target.summary?.cpu" />
                <metric-progress :label="t('runtimeTarget.metrics.memory')" :metric="target.summary?.memory" />
                <metric-progress :label="t('runtimeTarget.metrics.disk')" :metric="target.summary?.disk" />
              </div>
            </article>
          </t-col>
        </t-row>

        <t-empty
          v-else
          :title="t('runtimeTarget.list.emptyTitle')"
          :description="t('runtimeTarget.list.emptyDescription')"
          class="runtime-target-page__empty"
        >
          <template #action>
            <t-button theme="primary" :loading="discovering" @click="discoverLocal">
              {{ t('runtimeTarget.list.discoverLocal') }}
            </t-button>
          </template>
        </t-empty>
      </t-loading>

      <management-table-pagination :summary="t('runtimeTarget.list.summary', { count: total })">
        <t-pagination
          v-model:current="pagination.current"
          v-model:page-size="pagination.pageSize"
          :total="total"
          :page-size-options="[10, 20, 50, 100]"
          :show-page-number="true"
          @change="load"
        />
      </management-table-pagination>
    </management-page-content>
  </div>
</template>
<script setup lang="ts">
import { RefreshIcon, SearchIcon } from 'tdesign-icons-vue-next';
import { MessagePlugin } from 'tdesign-vue-next/es/message';
import { computed, defineComponent, h, onMounted, reactive, ref, resolveComponent } from 'vue';
import { useI18n } from 'vue-i18n';

import { ManagementPageContent, ManagementPageHeader, ManagementTablePagination } from '@/shared/components/management';
import { formatBytes } from '@/shared/observability';

import {
  discoverLocalDocker,
  listRuntimeTargetPage,
  refreshRuntimeTarget,
  type RuntimeTarget,
  type RuntimeTargetMetric,
} from '../../api/runtime-target';

const { t } = useI18n();
const loading = ref(false);
const discovering = ref(false);
const refreshingId = ref<number | null>(null);
const items = ref<RuntimeTarget[]>([]);
const total = ref(0);
const pagination = reactive({ current: 1, pageSize: 10 });

const MetricProgress = defineComponent({
  props: {
    label: { type: String, required: true },
    metric: { type: Object as () => RuntimeTargetMetric | undefined, default: undefined },
  },
  setup(props) {
    const metricAvailable = computed(() => props.metric?.available === true);
    const percentage = computed(() => Math.max(0, Math.min(100, props.metric?.usagePercent ?? 0)));
    const usage = computed(() => {
      if (!metricAvailable.value) return t('runtimeTarget.metrics.unavailable');
      if (props.metric?.usedBytes !== undefined && props.metric.totalBytes !== undefined) {
        return `${formatBytes(props.metric.usedBytes)} / ${formatBytes(props.metric.totalBytes)}`;
      }
      return `${percentage.value.toFixed(1)}%`;
    });
    return () =>
      h('section', { class: 'runtime-target-metric' }, [
        h('div', { class: 'runtime-target-metric__head' }, [
          h('span', props.label),
          metricAvailable.value
            ? h('strong', `${percentage.value.toFixed(1)}%`)
            : h('span', { class: 'runtime-target-metric__unavailable' }, t('runtimeTarget.metrics.unavailable')),
        ]),
        metricAvailable.value
          ? h(resolveComponent('t-progress'), {
              percentage: percentage.value,
              theme: 'line',
              label: false,
              trackColor: 'var(--td-component-stroke)',
            })
          : h(
              'span',
              { class: 'runtime-target-metric__hint' },
              props.metric?.unavailableReason || t('runtimeTarget.metrics.unavailableHint'),
            ),
        h('small', usage.value),
      ]);
  },
});

function availabilityText(value: boolean) {
  return value ? t('runtimeTarget.status.available') : t('runtimeTarget.status.unavailable');
}

async function load() {
  loading.value = true;
  try {
    const page = await listRuntimeTargetPage({
      limit: pagination.pageSize,
      offset: (pagination.current - 1) * pagination.pageSize,
    });
    items.value = page.items;
    total.value = page.total;
  } finally {
    loading.value = false;
  }
}

async function refreshTarget(id: number) {
  refreshingId.value = id;
  try {
    await refreshRuntimeTarget(id);
    await load();
  } finally {
    refreshingId.value = null;
  }
}

async function discoverLocal() {
  discovering.value = true;
  try {
    await discoverLocalDocker();
    pagination.current = 1;
    await load();
    MessagePlugin.success(t('runtimeTarget.list.discoverSuccess'));
  } finally {
    discovering.value = false;
  }
}

onMounted(load);
</script>
<style scoped lang="less">
.runtime-target-page__grid {
  margin-top: var(--graft-density-gap-4);
}

.runtime-target-card {
  background: var(--td-bg-color-container);
  border: 1px solid var(--td-component-stroke);
  border-radius: var(--td-radius-medium);
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-18);
  min-height: 312px;
  padding: var(--graft-density-gap-20);
}

.runtime-target-card__header,
.runtime-target-card__title-row,
.runtime-target-metric__head {
  align-items: center;
  display: flex;
}

.runtime-target-card__header {
  align-items: flex-start;
  justify-content: space-between;
}

.runtime-target-card__identity {
  min-width: 0;
}

.runtime-target-card__title-row {
  flex-wrap: wrap;
  gap: var(--graft-density-gap-8);
}

.runtime-target-card h2,
.runtime-target-card p,
.runtime-target-card section,
.runtime-target-card small {
  margin: 0;
}

.runtime-target-card h2 {
  color: var(--td-text-color-primary);
  font: var(--td-font-title-medium);
}

.runtime-target-card p {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
  margin-top: var(--graft-density-gap-6);
  overflow-wrap: anywhere;
}

.runtime-target-card__counts {
  display: grid;
  gap: var(--graft-density-gap-12);
  grid-template-columns: repeat(2, minmax(0, 1fr));
}

.runtime-target-card__counts section {
  background: var(--td-bg-color-secondarycontainer);
  border: 1px solid var(--td-component-stroke);
  border-radius: var(--td-radius-small);
  display: flex;
  flex-direction: column;
  min-width: 0;
  padding: var(--graft-density-gap-12);
}

.runtime-target-card__counts span,
.runtime-target-metric__head span {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
}

.runtime-target-card__counts strong {
  color: var(--td-text-color-primary);
  font: var(--td-font-title-large);
  line-height: 1.2;
  margin-top: var(--graft-density-gap-4);
}

.runtime-target-card__counts small,
.runtime-target-metric small {
  color: var(--td-text-color-placeholder);
  font: var(--td-font-body-small);
  margin-top: var(--graft-density-gap-4);
}

.runtime-target-card__metrics {
  display: grid;
  gap: var(--graft-density-gap-12);
}

.runtime-target-metric__head {
  justify-content: space-between;
  margin-bottom: var(--graft-density-gap-6);
}

.runtime-target-metric__head strong {
  color: var(--td-text-color-primary);
  font: var(--td-font-body-small);
}

.runtime-target-metric__unavailable,
.runtime-target-metric__hint {
  color: var(--td-text-color-placeholder);
  font: var(--td-font-body-small);
}

.runtime-target-metric__hint {
  display: block;
  min-height: 18px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.runtime-target-page__empty {
  min-height: 280px;
}

@media (width <= 640px) {
  .runtime-target-card {
    min-height: 0;
    padding: var(--graft-density-gap-16);
  }
}
</style>
