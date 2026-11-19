<template>
  <article class="management-entity-card">
    <slot name="header" />

    <dl
      v-if="cardFields.length"
      class="management-entity-card__fields"
      :class="{ 'management-entity-card__fields--stacked': stacked }"
    >
      <div v-for="field in cardFields" :key="field.key">
        <dt>{{ field.title }}</dt>
        <dd>
          <slot :name="`field-${field.key}`">{{ displayField(row[field.key]) }}</slot>
        </dd>
      </div>
    </dl>

    <slot name="actions" />
  </article>
</template>
<script setup lang="ts">
import type { TdBaseTableProps } from 'tdesign-vue-next';
import { computed } from 'vue';

// 卡片按列键投影同一条记录；字段优先级、格式覆盖和动作继续由业务列表提供。
const props = defineProps<{
  columns?: TdBaseTableProps['columns'];
  fields: readonly string[];
  row: Record<string, unknown>;
  stacked?: boolean;
}>();

const cardFields = computed(() => {
  const labels = new Map<string, string>();
  for (const { colKey, title } of props.columns ?? []) {
    if (colKey && typeof title === 'string') labels.set(colKey, title);
  }
  return props.fields.flatMap((key) => {
    const title = labels.get(key);
    return title ? [{ key, title }] : [];
  });
});

function displayField(value: unknown) {
  return typeof value === 'number' || (typeof value === 'string' && value.length) ? value : '-';
}
</script>
<style scoped lang="less">
@import '@/shared/components/card-surface.less';

.management-entity-card {
  .graft-entity-card-surface();

  display: grid;
  gap: var(--graft-density-gap-12);
  min-width: 0;
  padding: var(--graft-density-card-padding);
}

.management-entity-card__fields {
  display: grid;
  gap: var(--graft-density-gap-10);
  grid-template-columns: repeat(2, minmax(0, 1fr));
  margin: 0;
}

.management-entity-card__fields--stacked {
  grid-template-columns: 1fr;
}

.management-entity-card__fields > div {
  display: grid;
  gap: var(--graft-density-gap-4);
  min-width: 0;
}

.management-entity-card__fields dt {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
}

.management-entity-card__fields dd {
  color: var(--td-text-color-primary);
  font: var(--td-font-body-medium);
  font-variant-numeric: tabular-nums;
  margin: 0;
  overflow-wrap: anywhere;
}

@container (width < 360px) {
  .management-entity-card__fields {
    grid-template-columns: 1fr;
  }
}
</style>
