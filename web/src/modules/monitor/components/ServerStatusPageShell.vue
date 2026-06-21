<template>
  <governance-dashboard-shell
    domain="monitor"
    density="compact"
    :compact-header="compactHeader"
    :eyebrow="eyebrow"
    :title-key="titleKey"
    :title="title"
    :description-key="descriptionKey"
    :description="description"
    :source="source"
  >
    <template v-if="$slots.headerHint" #headerHint>
      <slot name="headerHint" />
    </template>
    <template v-if="$slots.toolbar" #actions>
      <slot name="toolbar" />
    </template>
    <template v-if="$slots.summary" #summary>
      <slot name="summary" />
    </template>
    <template v-if="$slots.feedback" #feedback>
      <slot name="feedback" />
    </template>
    <slot />
  </governance-dashboard-shell>
</template>
<script setup lang="ts">
import { GovernanceDashboardShell } from '@/shared/components/governance';
import type { PageHeaderSource } from '@/shared/components/page';

defineProps<{
  eyebrow: string;
  title?: string;
  description?: string;
  titleKey?: string;
  descriptionKey?: string;
  source?: PageHeaderSource;
  compactHeader?: boolean;
}>();
</script>
