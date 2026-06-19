<!--
  Copyright (c) 2025-2026 GeWuYou
  SPDX-License-Identifier: Apache-2.0
-->

<template>
  <div class="color-value-preview" :class="{ 'color-value-preview--compact': compact }">
    <template v-if="isColorToken">
      <span class="color-value-preview__swatch" :style="{ background: previewHex }" />
      <span class="color-value-preview__text">{{ summaryValue }}</span>
    </template>
    <template v-else>
      <span class="color-value-preview__swatch color-value-preview__swatch--text" aria-hidden="true">Aa</span>
      <span class="color-value-preview__text">{{ summaryValue }}</span>
    </template>
  </div>
</template>
<script setup lang="ts">
import { computed } from 'vue';

import { formatThemeTokenSummaryValue, isThemeTokenColorKey, resolveThemeTokenPreviewHex } from './theme-token-color';

const props = withDefaults(
  defineProps<{
    compact?: boolean;
    tokenKey: string;
    value: string;
  }>(),
  {
    compact: false,
  },
);

const isColorToken = computed(() => isThemeTokenColorKey(props.tokenKey));
const previewHex = computed(() => resolveThemeTokenPreviewHex(props.value));
const summaryValue = computed(() => formatThemeTokenSummaryValue(props.tokenKey, props.value));
</script>
<style lang="less" scoped>
.color-value-preview {
  align-items: center;
  background: color-mix(in srgb, var(--td-bg-color-page) 88%, var(--td-bg-color-container));
  border: 1px solid color-mix(in srgb, var(--td-component-stroke) 88%, transparent);
  border-radius: 12px;
  display: grid;
  gap: var(--graft-density-gap-10);
  grid-template-columns: 18px minmax(0, 1fr);
  min-width: 0;
  padding: var(--graft-density-gap-10) var(--graft-density-gap-12);
}

.color-value-preview--compact {
  background: transparent;
  border: 0;
  min-width: 112px;
  padding: 0;
}

.color-value-preview__swatch {
  border: 1px solid color-mix(in srgb, var(--td-component-stroke) 92%, transparent);
  border-radius: 999px;
  box-shadow:
    inset 0 1px 0 rgb(255 255 255 / 14%),
    0 4px 12px rgb(15 23 42 / 10%);
  flex: 0 0 auto;
  height: 18px;
  width: 18px;
}

.color-value-preview__swatch--text {
  align-items: center;
  background: var(--td-bg-color-container);
  color: var(--td-text-color-primary);
  display: inline-flex;
  font: var(--td-font-body-small);
  font-weight: 700;
  justify-content: center;
  width: 24px;
}

.color-value-preview__text {
  color: var(--td-text-color-primary);
  font: var(--td-font-body-small);
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, 'Liberation Mono', monospace;
  font-variant-numeric: tabular-nums;
  font-weight: 600;
  min-width: 0;
  overflow: hidden;
  text-align: left;
  text-overflow: ellipsis;
  white-space: nowrap;
}
</style>
