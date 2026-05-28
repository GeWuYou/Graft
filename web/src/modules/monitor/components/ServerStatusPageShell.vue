<template>
  <governance-dashboard-shell
    domain="monitor"
    density="compact"
    :compact-header="compactHeader"
    :eyebrow="eyebrow"
    :title="title"
    :description="description"
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

defineProps<{
  eyebrow: string;
  title: string;
  description: string;
  compactHeader?: boolean;
}>();
</script>
