<template>
  <article class="governance-summary-card" :data-card-kind="kind">
    <div class="governance-summary-card__top">
      <span class="governance-summary-card__title">{{ title }}</span>
      <div v-if="$slots.badge || badge" class="governance-summary-card__badge">
        <slot name="badge">{{ badge }}</slot>
      </div>
    </div>

    <div class="governance-summary-card__value-row">
      <strong class="governance-summary-card__value">{{ value }}</strong>
      <span v-if="valueAside" class="governance-summary-card__aside">{{ valueAside }}</span>
    </div>

    <p v-if="description" class="governance-summary-card__description">{{ description }}</p>
  </article>
</template>
<script setup lang="ts">
withDefaults(
  defineProps<{
    title: string;
    value: string;
    description?: string;
    valueAside?: string;
    badge?: string;
    kind?: 'metric' | 'risk' | 'status' | 'activity';
  }>(),
  {
    description: '',
    valueAside: '',
    badge: '',
    kind: 'metric',
  },
);
</script>
<style scoped lang="less">
@import './card-surface.less';

.governance-summary-card {
  .governance-card-surface();

  display: flex;
  flex-direction: column;
  gap: 10px;
  min-height: 132px;
  padding: 16px 18px;
}

.governance-summary-card__top {
  align-items: flex-start;
  display: flex;
  gap: 12px;
  justify-content: space-between;
}

.governance-summary-card__title {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
  font-weight: 500;
}

.governance-summary-card__value-row {
  align-items: baseline;
  display: flex;
  flex: 1;
  gap: 10px;
}

.governance-summary-card__value {
  color: var(--td-text-color-primary);
  font: var(--td-font-headline-small);
  font-variant-numeric: tabular-nums;
  font-weight: 700;
  letter-spacing: 0;
  line-height: 1.1;
}

.governance-summary-card__aside {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
}

.governance-summary-card__description {
  color: var(--td-text-color-placeholder);
  font: var(--td-font-body-small);
  margin: 0;
}
</style>
