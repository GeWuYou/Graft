<!--
  Copyright (c) 2025-2026 GeWuYou
  SPDX-License-Identifier: Apache-2.0
-->

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
      <page-header
        :breadcrumb="resolvedBreadcrumb"
        :source="resolvedSource"
        :title-key="titleKey"
        :title-fallback="title"
        :description-key="descriptionKey"
        :description-fallback="description"
        :compact="compactHeader"
      >
        <template v-if="$slots.headerHint" #extra>
          <slot name="headerHint" />
        </template>
        <template v-if="$slots.actions" #actions>
          <slot name="actions" />
        </template>
      </page-header>
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
import { computed } from 'vue';

import { PageHeader, type PageHeaderBreadcrumbItem, type PageHeaderSource } from '@/shared/components/page';

const props = withDefaults(
  defineProps<{
    title?: string;
    description?: string;
    eyebrow?: string;
    titleKey?: string;
    descriptionKey?: string;
    breadcrumb?: PageHeaderBreadcrumbItem[];
    source?: PageHeaderSource;
    domain?: 'audit' | 'monitor' | 'rbac' | 'access-control' | 'neutral';
    density?: 'comfortable' | 'compact';
    compactHeader?: boolean;
  }>(),
  {
    description: '',
    eyebrow: '',
    domain: 'neutral',
    density: 'comfortable',
    breadcrumb: undefined,
    compactHeader: false,
    descriptionKey: '',
    source: undefined,
    title: '',
    titleKey: '',
  },
);

const domainSourceKeyMap: Record<NonNullable<typeof props.domain>, string> = {
  audit: 'menu.audit.title',
  monitor: 'menu.server.title',
  rbac: 'menu.access_control.title',
  'access-control': 'menu.access_control.title',
  neutral: '',
};

const domainSourceColorMap: Record<NonNullable<typeof props.domain>, string> = {
  audit: 'var(--td-warning-color-5)',
  monitor: 'var(--td-brand-color-6)',
  rbac: 'var(--td-success-color-6)',
  'access-control': 'var(--td-success-color-6)',
  neutral: 'var(--td-brand-color-6)',
};

const resolvedSource = computed<PageHeaderSource | undefined>(() => {
  if (props.source) {
    return props.source;
  }

  if (!props.eyebrow) {
    return undefined;
  }

  return {
    color: domainSourceColorMap[props.domain],
    fallback: props.eyebrow,
    labelKey: domainSourceKeyMap[props.domain] || props.eyebrow,
  };
});

const resolvedBreadcrumb = computed<PageHeaderBreadcrumbItem[]>(() => {
  if (props.breadcrumb) {
    return props.breadcrumb;
  }

  const source = resolvedSource.value;
  const titleKey = props.titleKey || props.title;
  const titleFallback = props.title;

  if (!source) {
    return [{ labelKey: titleKey, fallback: titleFallback }];
  }

  if (source.labelKey === titleKey || source.fallback === titleFallback) {
    return [{ labelKey: titleKey, fallback: titleFallback }];
  }

  return [
    { labelKey: source.labelKey, fallback: source.fallback },
    { labelKey: titleKey, fallback: titleFallback },
  ];
});
</script>
<style scoped lang="less">
.governance-dashboard-shell {
  --governance-shell-gap: var(--graft-density-gap-18);
  --governance-shell-header-gap: var(--graft-density-gap-18);
  --governance-shell-heading-gap: var(--graft-density-gap-6);
  --governance-shell-summary-columns: 4;

  display: flex;
  flex-direction: column;
  gap: var(--governance-shell-gap);
  min-width: 0;
  padding-bottom: var(--graft-page-bottom-safe-area);
}

.governance-dashboard-shell__hero {
  min-width: 0;
}

.governance-dashboard-shell__hint,
.governance-dashboard-shell__feedback,
.governance-dashboard-shell__main {
  min-width: 0;
}

.governance-dashboard-shell__summary {
  display: grid;
  gap: var(--graft-density-gap-16);
  grid-template-columns: repeat(var(--governance-shell-summary-columns), minmax(0, 1fr));
}

.governance-dashboard-shell--compact,
.governance-dashboard-shell--comfortable {
  --governance-shell-summary-columns: 4;
}

.governance-dashboard-shell--compact {
  --governance-shell-gap: var(--graft-density-gap-16);
  --governance-shell-header-gap: var(--graft-density-gap-14);
  --governance-shell-heading-gap: var(--graft-density-gap-4);
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

@media (width <= 1199px) {
  .governance-dashboard-shell__summary {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (width <= 767px) {
  .governance-dashboard-shell__summary {
    grid-template-columns: 1fr;
  }
}
</style>
