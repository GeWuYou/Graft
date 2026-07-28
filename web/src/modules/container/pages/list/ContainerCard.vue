<template>
  <article class="container-card" :data-testid="`container-card-${row.id}`">
    <header class="container-card__header">
      <div class="container-card__identity">
        <strong>{{ displayContainerName(row) }}</strong>
        <span :title="row.id">{{ row.short_id || shortContainerId(row.id) }}</span>
      </div>
      <t-tag :theme="stateTheme(row.state)" variant="light-outline" shape="round">{{
        t(`container.list.states.${row.state}`)
      }}</t-tag>
    </header>
    <dl class="container-card__details">
      <div>
        <dt>{{ t('container.list.columns.image') }}</dt>
        <dd>{{ row.image || '-' }}</dd>
      </div>
      <div>
        <dt>{{ t('container.list.filters.runtimeTarget') }}</dt>
        <dd>{{ row.runtime_target?.display_name || row.runtime || '-' }}</dd>
      </div>
      <div>
        <dt>{{ t('container.list.columns.ports') }}</dt>
        <dd>{{ portSummary }}</dd>
      </div>
      <div>
        <dt>{{ t('container.list.columns.network') }}</dt>
        <dd>{{ row.network_summary || row.primary_ip || '-' }}</dd>
      </div>
    </dl>
    <section class="container-card__metrics" :aria-label="t('container.list.columns.resource')">
      <div>
        <span>{{ t('container.list.columns.cpu') }}</span
        ><strong>{{ cpuValue }}</strong>
      </div>
      <div>
        <span>{{ t('container.list.columns.memory') }}</span
        ><strong>{{ memoryValue }}</strong>
      </div>
    </section>
    <docker-resource-card-actions
      :detail-label="t('container.list.actions.detail')"
      :more-actions="moreActionOptions"
      :more-label="t('container.list.actions.more')"
      @detail="$emit('detail', row)"
      @action="emitAction"
    />
  </article>
</template>
<script setup lang="ts">
import { computed } from 'vue';
import { useI18n } from 'vue-i18n';

import DockerResourceCardActions from '../../components/DockerResourceCardActions.vue';
import { type ContainerResourceRowAction, displayContainerName, shortContainerId } from '../../shared/resource-table';
import type { ContainerSummaryRecord } from '../../types/container';

/** 卡片复用列表行与动作真值，不自行请求或推断容器能力。 */
const props = defineProps<{ row: ContainerSummaryRecord; actions: ContainerResourceRowAction[] }>();
const { t } = useI18n();
const moreActionOptions = computed(() =>
  props.actions
    .filter((action) => action.value !== 'detail')
    .map((action) => ({
      danger: action.danger,
      disabled: action.disabled,
      label: action.fallbackLabel,
      value: action.value,
    })),
);
const cpuValue = computed(() =>
  props.row.resource?.cpu_percent === undefined
    ? t('container.list.stats.unavailable')
    : `${props.row.resource.cpu_percent.toFixed(2)}%`,
);
const memoryValue = computed(() =>
  props.row.resource?.memory_usage_bytes === undefined
    ? t('container.list.stats.unavailable')
    : `${(props.row.resource.memory_usage_bytes / 1048576).toFixed(2)} MiB`,
);
const portSummary = computed(
  () =>
    props.row.ports
      .map((port) => `${port.public_port ? `${port.public_port}->` : ''}${port.private_port}/${port.type}`)
      .join(', ') || '-',
);
function stateTheme(state: ContainerSummaryRecord['state']) {
  return state === 'running'
    ? 'success'
    : state === 'dead'
      ? 'danger'
      : state === 'created' || state === 'paused' || state === 'restarting'
        ? 'warning'
        : 'default';
}
const emit = defineEmits<{
  detail: [row: ContainerSummaryRecord];
  action: [payload: { action: string; row: ContainerSummaryRecord }];
}>();
function emitAction(action: string) {
  emit('action', { action, row: props.row });
}
</script>
<style scoped lang="less">
@import '@/shared/components/card-surface.less';

.container-card {
  .graft-entity-card-surface();

  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-14);
  min-width: 0;
  padding: var(--graft-density-card-padding);
}

.container-card__header {
  align-items: flex-start;
  display: flex;
  gap: var(--graft-density-gap-12);
  justify-content: space-between;
  min-width: 0;
}

.container-card__identity {
  display: flex;
  flex: 1 1 auto;
  flex-direction: column;
  gap: var(--graft-density-gap-4);
  min-width: 0;
}

.container-card__identity strong {
  color: var(--td-text-color-primary);
  font: var(--td-font-title-small);
  overflow-wrap: anywhere;
}

.container-card__identity span,
.container-card__details dt,
.container-card__metrics span {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
}

.container-card__details {
  display: grid;
  gap: var(--graft-density-gap-10);
  grid-template-columns: repeat(2, minmax(0, 1fr));
  margin: 0;
}

.container-card__details div,
.container-card__metrics div {
  min-width: 0;
}

.container-card__details dt,
.container-card__details dd {
  margin: 0;
}

.container-card__details dd {
  color: var(--td-text-color-primary);
  font: var(--td-font-body-medium);
  overflow-wrap: anywhere;
}

.container-card__metrics {
  display: grid;
  gap: var(--graft-density-gap-10);
  grid-template-columns: repeat(2, minmax(0, 1fr));
}

.container-card__metrics div {
  align-content: start;
  background: var(--td-bg-color-container-hover);
  border-radius: var(--td-radius-default);
  display: grid;
  padding: var(--graft-density-gap-10);
  row-gap: var(--graft-density-gap-4);
}

.container-card__metrics strong {
  color: var(--td-text-color-primary);
  font: var(--td-font-title-small);
}

@container (width < 360px) {
  .container-card__details {
    grid-template-columns: 1fr;
  }
}
</style>
