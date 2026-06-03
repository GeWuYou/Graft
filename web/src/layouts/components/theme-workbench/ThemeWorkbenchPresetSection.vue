<template>
  <div class="section">
    <div class="section-title">{{ title }}</div>
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
  presets: ThemePresetDefinition[];
  activePresetId: string | null;
}>();

defineEmits<{
  select: [presetId: string];
}>();
</script>
