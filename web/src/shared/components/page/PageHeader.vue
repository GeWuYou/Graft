<template>
  <header
    class="page-header"
    :class="{ 'page-header--compact': compact, 'page-header--inline-actions': actionLayout === 'inline' }"
  >
    <div v-if="resolvedSource" class="page-header__source">
      <span class="page-header__source-dot" :style="{ background: resolvedSource.color || defaultSourceColor }" />
      <span>{{ resolveText(resolvedSource.labelKey, resolvedSource.fallback) }}</span>
    </div>

    <div class="page-header__main">
      <div class="page-header__copy">
        <h1 class="page-header__title">{{ resolvedTitle }}</h1>
        <p v-if="resolvedDescription" class="page-header__description">
          {{ resolvedDescription }}
        </p>
      </div>

      <div v-if="$slots.actions || $slots.compactActions || $slots.extra" class="page-header__side">
        <div v-if="$slots.extra" class="page-header__extra">
          <slot name="extra" />
        </div>
        <div v-if="$slots.actions" class="page-header__actions">
          <slot name="actions" />
        </div>
        <div v-if="$slots.compactActions" class="page-header__compact-actions">
          <slot name="compactActions" />
        </div>
      </div>
    </div>
  </header>
</template>
<script setup lang="ts">
import { computed } from 'vue';
import { useI18n } from 'vue-i18n';

import type { PageHeaderSource } from './types';

/** PageHeader 只解析页面文案并编排操作 slots，容器内空间不足时由自身重排，不依赖壳层视口。 */
const props = withDefaults(
  defineProps<{
    source?: PageHeaderSource;
    titleKey?: string;
    titleFallback?: string;
    descriptionKey?: string;
    descriptionFallback?: string;
    compact?: boolean;
    actionLayout?: 'responsive' | 'inline';
  }>(),
  {
    compact: false,
    actionLayout: 'responsive',
    descriptionFallback: '',
    descriptionKey: '',
    source: undefined,
    titleFallback: '',
    titleKey: '',
  },
);

const defaultSourceColor = 'var(--td-brand-color-6)';

const translate = (() => {
  try {
    return useI18n({ useScope: 'global' }).t;
  } catch {
    return undefined;
  }
})();

function resolveText(key: string | undefined, fallback = '') {
  if (!key) {
    return fallback;
  }

  const translated = translate?.(key);
  return translated && translated !== key ? translated : fallback;
}

const resolvedSource = computed(() => props.source);
const resolvedTitle = computed(() => resolveText(props.titleKey, props.titleFallback));
const resolvedDescription = computed(() => {
  if (!props.descriptionKey && !props.descriptionFallback) {
    return '';
  }

  return resolveText(props.descriptionKey, props.descriptionFallback || '');
});
</script>
<style scoped lang="less">
.page-header {
  container-type: inline-size;
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-6);
  min-width: 0;
}

.page-header__source {
  align-items: center;
  color: var(--td-text-color-secondary);
  display: inline-flex;
  font: var(--td-font-body-small);
  gap: var(--graft-density-gap-8);
  min-width: 0;
}

.page-header__source-dot {
  border-radius: var(--td-radius-circle);
  display: inline-flex;
  flex: 0 0 auto;
  height: 8px;
  width: 8px;
}

.page-header__main {
  align-items: flex-start;
  display: flex;
  gap: var(--graft-density-gap-16);
  justify-content: space-between;
  min-width: 0;
}

.page-header__copy {
  display: flex;
  flex: 1 1 auto;
  flex-direction: column;
  gap: var(--graft-density-gap-4);
  min-width: 0;
}

.page-header__title {
  color: var(--td-text-color-primary);
  font: var(--td-font-title-large);
  margin: 0;
  overflow-wrap: anywhere;
}

.page-header__description {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-medium);
  margin: 0;
  max-width: 760px;
  overflow-wrap: anywhere;
}

.page-header__side {
  align-items: flex-end;
  display: flex;
  flex: 0 0 auto;
  flex-direction: column;
  gap: var(--graft-density-gap-12);
  max-width: 100%;
}

.page-header__actions,
.page-header__compact-actions,
.page-header__extra {
  align-items: center;
  display: flex;
  flex-wrap: wrap;
  gap: var(--graft-density-gap-12);
  justify-content: flex-end;
  max-width: 100%;
}

.page-header__compact-actions {
  display: none;
}

.page-header--compact {
  gap: var(--graft-density-gap-4);
}

.page-header--compact .page-header__title {
  font: var(--td-font-headline-small);
}

@container (width < @screen-sm) {
  .page-header:not(.page-header--inline-actions) .page-header__main {
    flex-direction: column;
  }

  .page-header:not(.page-header--inline-actions) .page-header__side,
  .page-header:not(.page-header--inline-actions) .page-header__actions,
  .page-header:not(.page-header--inline-actions) .page-header__compact-actions,
  .page-header:not(.page-header--inline-actions) .page-header__extra {
    align-items: stretch;
    justify-content: flex-start;
    width: 100%;
  }

  .page-header:not(.page-header--inline-actions) .page-header__actions:has(+ .page-header__compact-actions) {
    display: none;
  }

  .page-header:not(.page-header--inline-actions) .page-header__compact-actions {
    display: flex;
  }

  .page-header--inline-actions .page-header__side {
    align-items: flex-end;
  }

  .page-header--inline-actions .page-header__actions,
  .page-header--inline-actions .page-header__compact-actions,
  .page-header--inline-actions .page-header__extra {
    justify-content: flex-end;
  }
}
</style>
