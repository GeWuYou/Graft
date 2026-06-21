<template>
  <section class="assignment-summary">
    <div v-if="items.length > 0" class="assignment-summary__grid">
      <div v-for="item in items" :key="item.label" class="assignment-summary__item">
        <span class="assignment-summary__value">{{ item.value }}</span>
        <span class="assignment-summary__label">{{ item.label }}</span>
      </div>
    </div>

    <div v-if="hint" :data-testid="hintTestId" class="assignment-summary__hint">
      {{ hint }}
    </div>

    <div v-if="warning" class="assignment-summary__warning">
      <span>{{ warning }}</span>
      <t-button
        v-if="warningActionLabel"
        size="small"
        theme="primary"
        variant="text"
        :loading="warningActionLoading"
        @click="emit('warning-action')"
      >
        {{ warningActionLabel }}
      </t-button>
    </div>
  </section>
</template>
<script setup lang="ts">
export type AssignmentSummaryItem = {
  label: string;
  value: number | string;
};

withDefaults(
  defineProps<{
    hint?: string;
    hintTestId?: string;
    items?: AssignmentSummaryItem[];
    warning?: string;
    warningActionLabel?: string;
    warningActionLoading?: boolean;
  }>(),
  {
    hint: '',
    hintTestId: '',
    items: () => [],
    warning: '',
    warningActionLabel: '',
    warningActionLoading: false,
  },
);

const emit = defineEmits<{
  'warning-action': [];
}>();
</script>
<style scoped lang="less">
.assignment-summary,
.assignment-summary__grid,
.assignment-summary__item,
.assignment-summary__warning {
  display: flex;
}

.assignment-summary {
  flex-direction: column;
  gap: var(--td-comp-margin-m);
}

.assignment-summary__grid {
  gap: var(--td-comp-margin-m);
}

.assignment-summary__item {
  background: var(--td-bg-color-container);
  border: 1px solid var(--td-component-stroke);
  border-radius: var(--td-radius-medium);
  flex: 1 1 0;
  flex-direction: column;
  gap: var(--graft-density-gap-4);
  min-width: 0;
  padding: var(--td-comp-paddingTB-m) var(--td-comp-paddingLR-l);
}

.assignment-summary__value {
  color: var(--td-text-color-primary);
  font: var(--td-font-title-medium);
}

.assignment-summary__label,
.assignment-summary__hint {
  color: var(--td-text-color-secondary);
}

.assignment-summary__hint {
  font: var(--td-font-body-medium);
}

.assignment-summary__warning {
  align-items: center;
  background: color-mix(in srgb, var(--td-warning-color-5) 10%, var(--td-bg-color-container));
  border: 1px solid color-mix(in srgb, var(--td-warning-color-5) 24%, var(--td-component-stroke));
  border-radius: var(--td-radius-medium);
  color: var(--td-warning-color-7);
  gap: var(--td-comp-margin-m);
  justify-content: space-between;
  padding: var(--td-comp-paddingTB-s) var(--td-comp-paddingLR-l);
}

@media (width <= 768px) {
  .assignment-summary__grid,
  .assignment-summary__warning {
    flex-direction: column;
  }
}
</style>
