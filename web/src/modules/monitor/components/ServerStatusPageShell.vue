<template>
  <div
    class="server-status-page-shell"
    :class="{ 'server-status-page-shell--compact': compactHeader }"
    data-page-type="overview-dashboard"
  >
    <header class="server-status-page-shell__header">
      <div class="server-status-page-shell__heading">
        <p class="server-status-page-shell__eyebrow">{{ eyebrow }}</p>
        <h1 class="server-status-page-shell__title">{{ title }}</h1>
        <p class="server-status-page-shell__description">{{ description }}</p>
        <div v-if="$slots.headerHint" class="server-status-page-shell__hint">
          <slot name="headerHint" />
        </div>
      </div>
      <div class="server-status-page-shell__toolbar">
        <slot name="toolbar" />
      </div>
    </header>

    <section v-if="$slots.summary" class="server-status-page-shell__summary">
      <slot name="summary" />
    </section>

    <section v-if="$slots.feedback" class="server-status-page-shell__feedback">
      <slot name="feedback" />
    </section>

    <main class="server-status-page-shell__main">
      <slot />
    </main>
  </div>
</template>
<script setup lang="ts">
defineProps<{
  eyebrow: string;
  title: string;
  description: string;
  compactHeader?: boolean;
}>();
</script>
<style scoped lang="less">
.server-status-page-shell {
  --server-status-page-shell-gap: 16px;
  --server-status-page-shell-header-gap: 16px;
  --server-status-page-shell-heading-gap: 6px;
  --server-status-page-shell-eyebrow-gap: 2px;

  display: flex;
  flex-direction: column;
  gap: var(--server-status-page-shell-gap);
  min-width: 0;
}

.server-status-page-shell__header {
  align-items: flex-start;
  display: flex;
  gap: var(--server-status-page-shell-header-gap);
  justify-content: space-between;
}

.server-status-page-shell__heading {
  display: flex;
  flex: 1 1 auto;
  flex-direction: column;
  gap: var(--server-status-page-shell-heading-gap);
  min-width: 0;
}

.server-status-page-shell__eyebrow {
  color: var(--td-text-color-secondary);
  font-size: 13px;
  font-weight: 600;
  line-height: 20px;
  margin: 0 0 var(--server-status-page-shell-eyebrow-gap);
}

.server-status-page-shell__title {
  color: var(--td-text-color-primary);
  font-size: 30px;
  font-weight: 700;
  letter-spacing: -0.02em;
  line-height: 38px;
  margin: 0;
}

.server-status-page-shell__description {
  color: var(--td-text-color-secondary);
  font-size: 14px;
  line-height: 22px;
  margin: 0;
  max-width: 760px;
}

.server-status-page-shell__hint {
  margin: 0;
}

.server-status-page-shell__toolbar {
  align-items: flex-start;
  display: flex;
  flex: 0 0 auto;
  justify-content: flex-end;
  max-width: 100%;
}

.server-status-page-shell__summary {
  display: grid;
  gap: 16px;
  grid-template-columns: repeat(4, minmax(0, 1fr));
}

.server-status-page-shell__feedback,
.server-status-page-shell__main {
  min-width: 0;
}

.server-status-page-shell--compact {
  --server-status-page-shell-gap: 16px;
  --server-status-page-shell-header-gap: 14px;
  --server-status-page-shell-heading-gap: 4px;
  --server-status-page-shell-eyebrow-gap: 0;
}

.server-status-page-shell--compact .server-status-page-shell__header {
  gap: var(--server-status-page-shell-header-gap);
}

.server-status-page-shell--compact .server-status-page-shell__title {
  font-size: 28px;
  line-height: 34px;
}

@media (width <= 1199px) {
  .server-status-page-shell__summary {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (width <= 991px) {
  .server-status-page-shell__header {
    flex-direction: column;
    gap: 12px;
  }

  .server-status-page-shell__toolbar {
    justify-content: flex-start;
    width: 100%;
  }
}

@media (width <= 767px) {
  .server-status-page-shell {
    --server-status-page-shell-gap: 16px;
    --server-status-page-shell-heading-gap: 4px;
  }

  .server-status-page-shell__title {
    font-size: 24px;
    line-height: 32px;
  }

  .server-status-page-shell__summary {
    grid-template-columns: 1fr;
  }
}
</style>
