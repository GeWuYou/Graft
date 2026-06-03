<template>
  <div
    class="governance-dashboard-shell"
    :class="[
      `governance-dashboard-shell--${density}`,
      {
        'governance-dashboard-shell--compact': compactHeader,
      },
    ]"
    :data-governance-domain="domain"
    data-page-type="overview-dashboard"
  >
    <header class="governance-dashboard-shell__hero">
      <div class="governance-dashboard-shell__heading">
        <p v-if="eyebrow || $slots.eyebrow" class="governance-dashboard-shell__eyebrow">
          <slot name="eyebrow">{{ eyebrow }}</slot>
        </p>
        <h1 class="governance-dashboard-shell__title">{{ title }}</h1>
        <p v-if="description" class="governance-dashboard-shell__description">{{ description }}</p>
        <div v-if="$slots.headerHint" class="governance-dashboard-shell__hint">
          <slot name="headerHint" />
        </div>
      </div>
      <div v-if="$slots.actions" class="governance-dashboard-shell__actions">
        <slot name="actions" />
      </div>
    </header>

    <section v-if="$slots.summary" class="governance-dashboard-shell__summary">
      <slot name="summary" />
    </section>

    <section v-if="$slots.feedback" class="governance-dashboard-shell__feedback">
      <slot name="feedback" />
    </section>

    <main class="governance-dashboard-shell__main">
      <slot />
    </main>
  </div>
</template>
<script setup lang="ts">
withDefaults(
  defineProps<{
    title: string;
    description?: string;
    eyebrow?: string;
    domain?: 'audit' | 'monitor' | 'rbac' | 'access-control' | 'neutral';
    density?: 'comfortable' | 'compact';
    compactHeader?: boolean;
  }>(),
  {
    description: '',
    eyebrow: '',
    domain: 'neutral',
    density: 'comfortable',
    compactHeader: false,
  },
);
</script>
<style scoped lang="less">
.governance-dashboard-shell {
  --governance-shell-gap: 18px;
  --governance-shell-header-gap: 18px;
  --governance-shell-heading-gap: 6px;
  --governance-shell-summary-columns: 4;

  display: flex;
  flex-direction: column;
  gap: var(--governance-shell-gap);
  min-width: 0;
  padding-bottom: var(--graft-page-bottom-safe-area);
}

.governance-dashboard-shell__hero {
  align-items: flex-start;
  display: flex;
  gap: var(--governance-shell-header-gap);
  justify-content: space-between;
}

.governance-dashboard-shell__heading {
  display: flex;
  flex: 1 1 auto;
  flex-direction: column;
  gap: var(--governance-shell-heading-gap);
  min-width: 0;
}

.governance-dashboard-shell__eyebrow {
  color: var(--td-text-color-secondary);
  font: var(--td-font-title-small);
  margin: 0;
}

.governance-dashboard-shell__title {
  color: var(--td-text-color-primary);
  font: var(--td-font-headline-medium);
  letter-spacing: -0.02em;
  margin: 0;
}

.governance-dashboard-shell__description {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-medium);
  margin: 0;
  max-width: 760px;
}

.governance-dashboard-shell__hint,
.governance-dashboard-shell__feedback,
.governance-dashboard-shell__main {
  min-width: 0;
}

.governance-dashboard-shell__actions {
  display: flex;
  flex: 0 0 auto;
  justify-content: flex-end;
  max-width: 100%;
}

.governance-dashboard-shell__summary {
  display: grid;
  gap: 16px;
  grid-template-columns: repeat(var(--governance-shell-summary-columns), minmax(0, 1fr));
}

.governance-dashboard-shell--compact,
.governance-dashboard-shell--comfortable {
  --governance-shell-summary-columns: 4;
}

.governance-dashboard-shell--compact {
  --governance-shell-gap: 16px;
  --governance-shell-header-gap: 14px;
  --governance-shell-heading-gap: 4px;
}

.governance-dashboard-shell--compact .governance-dashboard-shell__title {
  font: var(--td-font-headline-medium);
}

.governance-dashboard-shell[data-governance-domain='audit'] {
  --governance-shell-accent: var(--td-warning-color-5);
}

.governance-dashboard-shell[data-governance-domain='monitor'] {
  --governance-shell-accent: var(--td-brand-color-6);
}

.governance-dashboard-shell[data-governance-domain='rbac'],
.governance-dashboard-shell[data-governance-domain='access-control'] {
  --governance-shell-accent: var(--td-success-color-6);
}

.governance-dashboard-shell__eyebrow::before {
  background: var(--governance-shell-accent, var(--td-brand-color-6));
  border-radius: 999px;
  content: '';
  display: inline-flex;
  height: 8px;
  margin-right: 8px;
  width: 8px;
}

@media (width <= 1199px) {
  .governance-dashboard-shell__summary {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (width <= 991px) {
  .governance-dashboard-shell__hero {
    flex-direction: column;
    gap: 12px;
  }

  .governance-dashboard-shell__actions {
    justify-content: flex-start;
    width: 100%;
  }
}

@media (width <= 767px) {
  .governance-dashboard-shell__title {
    font: var(--td-font-headline-small);
  }

  .governance-dashboard-shell__summary {
    grid-template-columns: 1fr;
  }
}
</style>
