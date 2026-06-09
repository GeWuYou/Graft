<!--
  Copyright (c) 2025-2026 GeWuYou
  SPDX-License-Identifier: Apache-2.0
-->

<template>
  <governance-section :title="title" :description="description" :kind="kind" :min-height="minHeight">
    <div class="governance-action-panel">
      <slot />
    </div>
  </governance-section>
</template>
<script setup lang="ts">
import GovernanceSection from './GovernanceSection.vue';

withDefaults(
  defineProps<{
    title: string;
    description?: string;
    minHeight?: number | string;
    kind?: 'investigation' | 'workflow' | 'navigation';
  }>(),
  {
    description: '',
    minHeight: undefined,
    kind: 'investigation',
  },
);
</script>
<style scoped lang="less">
.governance-action-panel {
  display: grid;
  gap: var(--graft-density-gap-12);
}
</style>
