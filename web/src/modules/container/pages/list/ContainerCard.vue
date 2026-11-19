<template>
  <management-entity-card
    :data-testid="`container-card-${row.id}`"
    :row="row"
    :columns="columns"
    :fields="['image', 'ports', 'network', 'cpu', 'memory']"
  >
    <template #header>
      <header class="container-card__header">
        <t-checkbox
          :checked="selected"
          :aria-label="t('container.list.batch.selectRow', { name: displayContainerName(row) })"
          @change="emit('select-change', Boolean($event))"
        />
        <div class="container-card__identity">
          <strong>{{ displayContainerName(row) }}</strong>
          <span :title="row.id">{{ row.short_id || shortContainerId(row.id) }}</span>
        </div>
        <t-tag :theme="stateTheme(row.state)" variant="light-outline" shape="round">{{
          t(`container.list.states.${row.state}`)
        }}</t-tag>
      </header>
    </template>
    <template #field-image>
      <span>{{ row.image || '-' }}</span>
      <span v-if="row.runtime_target?.display_name || row.runtime" class="container-card__target">
        {{ row.runtime_target?.display_name || row.runtime }}
      </span>
    </template>
    <template #field-ports>{{ portSummary }}</template>
    <template #field-network>{{ row.network_summary || row.primary_ip || '-' }}</template>
    <template #field-cpu
      ><strong>{{ cpuValue }}</strong></template
    >
    <template #field-memory
      ><strong>{{ memoryValue }}</strong></template
    >
    <template #actions>
      <docker-resource-card-actions
        :detail-label="t('container.list.actions.detail')"
        :more-actions="moreActionOptions"
        :more-label="t('container.list.actions.more')"
        @detail="$emit('detail', row)"
        @action="emitAction"
      />
    </template>
  </management-entity-card>
</template>
<script setup lang="ts">
import type { TdBaseTableProps } from 'tdesign-vue-next';
import { computed } from 'vue';
import { useI18n } from 'vue-i18n';

import { ManagementEntityCard } from '@/shared/components/management';

import DockerResourceCardActions from '../../components/DockerResourceCardActions.vue';
import { type ContainerResourceRowAction, displayContainerName, shortContainerId } from '../../shared/resource-table';
import type { ContainerSummaryRecord } from '../../types/container';

/** 卡片复用列表行与动作真值，不自行请求或推断容器能力。 */
const props = defineProps<{
  row: ContainerSummaryRecord;
  columns?: TdBaseTableProps['columns'];
  actions: ContainerResourceRowAction[];
  selected?: boolean;
}>();
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
  'select-change': [checked: boolean];
}>();
function emitAction(action: string) {
  emit('action', { action, row: props.row });
}
</script>
<style scoped lang="less">
.container-card__header {
  align-items: flex-start;
  display: flex;
  gap: var(--graft-density-gap-12);
  justify-content: space-between;
  min-width: 0;
}

.container-card__identity {
  display: grid;
  flex: 1 1 auto;
  gap: var(--graft-density-gap-4);
  min-width: 0;
}

.container-card__identity strong {
  color: var(--td-text-color-primary);
  font: var(--td-font-title-small);
  overflow-wrap: anywhere;
}

.container-card__identity span,
.container-card__target {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
}

.container-card__target {
  display: block;
}
</style>
