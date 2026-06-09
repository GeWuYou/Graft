<!--
  Copyright (c) 2025-2026 GeWuYou
  SPDX-License-Identifier: Apache-2.0
-->

<template>
  <div class="preset-grid">
    <button
      v-for="preset in presets"
      :key="preset.id"
      type="button"
      class="preset-card"
      :class="{ 'preset-card--active': activePresetId === preset.id }"
      :style="thumbnailStyle(preset)"
      @click="$emit('select', preset.id)"
    >
      <span class="preset-card__thumbnail">
        <span class="preset-card__thumb-shell">
          <span class="preset-card__thumb-sidebar">
            <span class="preset-card__thumb-brand" />
            <span class="preset-card__thumb-menu preset-card__thumb-menu--active" />
            <span class="preset-card__thumb-menu" />
            <span class="preset-card__thumb-menu" />
          </span>
          <span class="preset-card__thumb-main">
            <span class="preset-card__thumb-header">
              <span class="preset-card__thumb-title" />
              <span class="preset-card__thumb-button" />
            </span>
            <span class="preset-card__thumb-tags">
              <span class="preset-card__thumb-tag" />
              <span class="preset-card__thumb-tag preset-card__thumb-tag--muted" />
            </span>
            <span class="preset-card__thumb-table">
              <span class="preset-card__thumb-table-head" />
              <span class="preset-card__thumb-table-row" />
              <span class="preset-card__thumb-table-row" />
            </span>
          </span>
        </span>
      </span>
      <span class="preset-card__title">{{ t(preset.labelKey) }}</span>
      <span class="preset-card__desc">{{ t(preset.descriptionKey) }}</span>
    </button>
  </div>
</template>
<script setup lang="ts">
import type { CSSProperties } from 'vue';

import { t } from '@/locales';
import type { ThemePresetDefinition } from '@/types/theme';

defineProps<{
  presets: ThemePresetDefinition[];
  activePresetId: string | null;
}>();

defineEmits<{
  select: [presetId: string];
}>();

const thumbnailStyle = (preset: ThemePresetDefinition): CSSProperties => {
  const baseBackground =
    preset.mode === 'dark'
      ? 'linear-gradient(180deg, rgba(255,255,255,0.05), rgba(255,255,255,0)), #111827'
      : 'linear-gradient(180deg, rgba(255,255,255,0.72), rgba(255,255,255,0.22)), #f5f7fa';

  const surfaceBackground = preset.mode === 'dark' ? 'rgba(255,255,255,0.08)' : 'rgba(255,255,255,0.9)';
  const borderColor = preset.mode === 'dark' ? 'rgba(255,255,255,0.08)' : 'rgba(17, 24, 39, 0.08)';
  const mutedColor = preset.mode === 'dark' ? 'rgba(255,255,255,0.16)' : 'rgba(148, 163, 184, 0.28)';

  return {
    '--preset-brand-color': preset.brandTheme,
    '--preset-thumbnail-background': baseBackground,
    '--preset-thumbnail-surface': surfaceBackground,
    '--preset-thumbnail-border': borderColor,
    '--preset-thumbnail-muted': mutedColor,
  } as CSSProperties;
};
</script>
<style lang="less" scoped>
.preset-grid {
  display: grid;
  gap: var(--graft-density-gap-12);
  grid-template-columns: repeat(auto-fit, minmax(190px, 1fr));
}

.preset-card {
  background: var(--td-bg-color-container);
  border: 1px solid var(--td-component-stroke);
  border-radius: var(--td-radius-large);
  display: grid;
  gap: var(--graft-density-gap-10);
  isolation: isolate;
  min-width: 0;
  overflow: hidden;
  padding: var(--graft-density-gap-12);
  position: relative;
  text-align: left;
  transition:
    border-color 220ms ease,
    box-shadow 220ms ease,
    transform 220ms ease;
}

.preset-card::before {
  background: radial-gradient(
    circle at 50% 34%,
    color-mix(in srgb, var(--preset-brand-color, var(--td-brand-color)) 18%, transparent),
    transparent 64%
  );
  content: '';
  inset: 0;
  opacity: 0;
  pointer-events: none;
  position: absolute;
  transition: opacity 220ms ease;
  z-index: 0;
}

.preset-card > * {
  position: relative;
  z-index: 1;
}

.preset-card:hover,
.preset-card:focus-visible {
  border-color: color-mix(in srgb, var(--preset-brand-color, var(--td-brand-color)) 68%, var(--td-component-stroke));
  box-shadow:
    0 0 0 3px color-mix(in srgb, var(--preset-brand-color, var(--td-brand-color)) 18%, transparent),
    0 14px 30px color-mix(in srgb, var(--preset-brand-color, var(--td-brand-color)) 16%, transparent),
    var(--td-shadow-1);
  transform: translateY(-2px);
}

.preset-card:hover::before,
.preset-card:focus-visible::before {
  opacity: 1;
}

.preset-card--active {
  border-color: var(--preset-brand-color, var(--td-brand-color));
  box-shadow:
    0 0 0 1px color-mix(in srgb, var(--preset-brand-color, var(--td-brand-color)) 24%, transparent),
    var(--td-shadow-1);
}

.preset-card__thumbnail {
  background: var(--preset-thumbnail-background);
  border: 1px solid var(--preset-thumbnail-border);
  border-radius: calc(var(--td-radius-large) - 2px);
  display: block;
  overflow: hidden;
  padding: var(--graft-density-gap-8);
}

.preset-card__thumb-shell {
  background: var(--preset-thumbnail-surface);
  border: 1px solid var(--preset-thumbnail-border);
  border-radius: 12px;
  display: grid;
  gap: var(--graft-density-gap-8);
  grid-template-columns: 40px 1fr;
  min-height: 112px;
  overflow: hidden;
}

.preset-card__thumb-sidebar {
  background: color-mix(in srgb, var(--preset-brand-color) 12%, var(--preset-thumbnail-surface));
  border-right: 1px solid var(--preset-thumbnail-border);
  display: grid;
  gap: var(--graft-density-gap-6);
  padding: var(--graft-density-gap-8) var(--graft-density-gap-6);
}

.preset-card__thumb-brand,
.preset-card__thumb-menu,
.preset-card__thumb-title,
.preset-card__thumb-button,
.preset-card__thumb-tag,
.preset-card__thumb-table-head,
.preset-card__thumb-table-row {
  border-radius: 999px;
  display: block;
}

.preset-card__thumb-brand {
  background: var(--preset-brand-color);
  height: 7px;
  width: 18px;
}

.preset-card__thumb-menu {
  background: var(--preset-thumbnail-muted);
  height: 6px;
  width: 100%;
}

.preset-card__thumb-menu--active {
  background: color-mix(in srgb, var(--preset-brand-color) 74%, white 6%);
}

.preset-card__thumb-main {
  display: grid;
  gap: var(--graft-density-gap-8);
  padding: var(--graft-density-gap-10);
}

.preset-card__thumb-header {
  align-items: center;
  display: flex;
  gap: var(--graft-density-gap-8);
  justify-content: space-between;
}

.preset-card__thumb-title {
  background: color-mix(in srgb, var(--preset-brand-color) 22%, var(--preset-thumbnail-muted));
  height: 8px;
  width: 54px;
}

.preset-card__thumb-button {
  background: var(--preset-brand-color);
  height: 12px;
  width: 26px;
}

.preset-card__thumb-tags {
  display: flex;
  gap: var(--graft-density-gap-6);
}

.preset-card__thumb-tag {
  background: color-mix(in srgb, var(--preset-brand-color) 20%, transparent);
  border: 1px solid color-mix(in srgb, var(--preset-brand-color) 36%, transparent);
  height: 12px;
  width: 28px;
}

.preset-card__thumb-tag--muted {
  background: transparent;
  border-color: var(--preset-thumbnail-border);
}

.preset-card__thumb-table {
  background: color-mix(in srgb, var(--preset-thumbnail-surface) 85%, transparent);
  border: 1px solid var(--preset-thumbnail-border);
  border-radius: 10px;
  display: grid;
  gap: var(--graft-density-gap-6);
  padding: var(--graft-density-gap-8);
}

.preset-card__thumb-table-head {
  background: color-mix(in srgb, var(--preset-brand-color) 16%, var(--preset-thumbnail-muted));
  height: 8px;
  width: 72%;
}

.preset-card__thumb-table-row {
  background: var(--preset-thumbnail-muted);
  height: 6px;
  width: 100%;
}

.preset-card__thumb-table-row:last-child {
  width: 82%;
}

.preset-card__title {
  color: var(--td-text-color-primary);
  font: var(--td-font-title-small);
  font-weight: 700;
}

.preset-card__desc {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
  line-height: 1.6;
}
</style>
