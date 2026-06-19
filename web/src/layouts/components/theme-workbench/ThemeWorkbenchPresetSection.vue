<!--
  Copyright (c) 2025-2026 GeWuYou
  SPDX-License-Identifier: Apache-2.0
-->

<template>
  <div class="section">
    <div class="section-heading">
      <div class="section-title">{{ title }}</div>
      <div v-if="description" class="section-desc">{{ description }}</div>
    </div>
    <theme-workbench-preset-grid
      :presets="presets"
      :active-preset-id="activePresetId"
      @select="$emit('select', $event)"
    />
  </div>
</template>
<script setup lang="ts">
import type { ThemePresetDefinition } from '@/types/theme';

import ThemeWorkbenchPresetGrid from './ThemeWorkbenchPresetGrid.vue';

defineProps<{
  title: string;
  description?: string;
  presets: ThemePresetDefinition[];
  activePresetId: string | null;
}>();

defineEmits<{
  select: [presetId: string];
}>();
</script>
<style lang="less" scoped>
.section-heading {
  display: grid;
  gap: var(--graft-density-gap-4);
}

.section-desc {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
  line-height: 22px;
}
</style>
