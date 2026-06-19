<!--
  Copyright (c) 2025-2026 GeWuYou
  SPDX-License-Identifier: Apache-2.0
-->

<template>
  <article class="theme-token-item" :class="{ 'theme-token-item--expanded': expanded }">
    <button type="button" class="theme-token-item__summary" :aria-expanded="expanded" @click="expanded = !expanded">
      <span class="theme-token-item__meta">
        <span class="theme-token-item__label">{{ t(token.labelKey) }}</span>
        <span class="theme-token-item__key">{{ token.key }}</span>
      </span>
      <span class="theme-token-item__summary-side">
        <color-value-preview compact :token-key="token.key" :value="value" />
        <span class="theme-token-item__toggle">
          {{ expanded ? t('layout.setting.workbench.token.collapse') : t('layout.setting.workbench.token.expand') }}
        </span>
      </span>
    </button>

    <theme-token-value-editor
      v-if="expanded"
      :has-override="hasOverride"
      :token="token"
      :value="value"
      @commit="(nextValue) => $emit('commit', nextValue)"
      @reset="$emit('reset')"
    />
  </article>
</template>
<script setup lang="ts">
import { ref } from 'vue';

import { t } from '@/locales';
import type { ThemeTokenDefinition } from '@/types/theme';

import ColorValuePreview from './ColorValuePreview.vue';
import ThemeTokenValueEditor from './ThemeTokenValueEditor.vue';

defineProps<{
  hasOverride: boolean;
  token: ThemeTokenDefinition;
  value: string;
}>();

defineEmits<{
  commit: [value: string];
  reset: [];
}>();

const expanded = ref(false);
</script>
<style lang="less" scoped>
@import './theme-surface.less';

.theme-token-item {
  .theme-workbench-surface();

  display: grid;
  gap: var(--graft-density-gap-12);
  min-width: 0;
  overflow: hidden;
  padding: var(--graft-density-gap-12);
  transition:
    border-color 180ms ease,
    box-shadow 180ms ease,
    background 180ms ease;
}

.theme-token-item:hover {
  border-color: color-mix(in srgb, var(--td-brand-color) 24%, var(--td-component-stroke));
}

.theme-token-item--expanded {
  border-color: color-mix(in srgb, var(--td-brand-color) 36%, var(--td-component-stroke));
  box-shadow: 0 0 0 1px color-mix(in srgb, var(--td-brand-color) 12%, transparent);
}

.theme-token-item__summary {
  align-items: start;
  appearance: none;
  background: transparent;
  border: 0;
  cursor: pointer;
  display: grid;
  gap: var(--graft-density-gap-12);
  grid-template-columns: minmax(0, 1fr) auto;
  min-width: 0;
  padding: 0;
  text-align: left;
  width: 100%;
}

.theme-token-item__meta {
  display: grid;
  gap: var(--graft-density-gap-4);
  min-width: 0;
}

.theme-token-item__label {
  color: var(--td-text-color-primary);
  font: var(--td-font-body-medium);
  font-weight: 600;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.theme-token-item__key {
  color: var(--td-text-color-placeholder);
  font: var(--td-font-body-small);
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, 'Liberation Mono', monospace;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.theme-token-item__summary-side {
  display: grid;
  gap: var(--graft-density-gap-10);
  grid-template-columns: minmax(112px, max-content) auto;
  min-width: 0;
  place-items: center end;
}

.theme-token-item__toggle {
  color: var(--td-brand-color);
  font: var(--td-font-body-small);
  font-weight: 600;
  white-space: nowrap;
}

@media (width <= 768px) {
  .theme-token-item__summary {
    grid-template-columns: 1fr;
  }

  .theme-token-item__summary-side {
    grid-template-columns: minmax(0, 1fr);
    justify-items: start;
  }
}
</style>
