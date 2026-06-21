<template>
  <governance-section :title="title" :description="description" :min-height="minHeight" kind="trend">
    <template v-if="$slots.actions" #actions>
      <slot name="actions" />
    </template>
    <slot />
  </governance-section>
</template>
<script setup lang="ts">
import { GovernanceSection } from '@/shared/components/governance';

defineProps<{
  title?: string;
  description?: string;
  minHeight?: number | string;
}>();
</script>
