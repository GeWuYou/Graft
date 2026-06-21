<template>
  <span class="config-value-renderer">
    <strong>{{ presentation.value }}</strong>
    <slot name="description" :description="presentation.description" :mode="presentation.descriptionMode" />
  </span>
</template>
<script setup lang="ts">
import { computed } from 'vue';

import type { ConfigValuePresentation, ConfigValueRendererInput } from './value-renderer';
import { configValuePresentation } from './value-renderer';

const props = defineProps<ConfigValueRendererInput>();

const presentation = computed<ConfigValuePresentation>(() => configValuePresentation(props));
</script>
<style scoped>
.config-value-renderer {
  align-items: center;
  display: inline-flex;
  gap: var(--td-comp-margin-xs);
  min-width: 0;
}
</style>
