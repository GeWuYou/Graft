<template>
  <div class="graft-code-block" :class="{ 'graft-code-block--wrap': wrap }">
    <div v-if="!code.trim()" class="graft-code-block__empty">
      {{ emptyText }}
    </div>
    <div
      v-else-if="renderedHtml"
      class="graft-code-block__content graft-scrollbar"
      :style="contentStyle"
      v-html="renderedHtml"
    />
    <pre v-else class="graft-code-block__fallback graft-scrollbar" :style="contentStyle"><code>{{ code }}</code></pre>
  </div>
</template>
<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue';

import { renderHighlightedCodeBlock, type SharedCodeBlockLanguage } from '@/shared/code/shiki';

const props = withDefaults(
  defineProps<{
    code: string;
    emptyText?: string;
    lang: SharedCodeBlockLanguage;
    maxHeight?: number | string;
    wrap?: boolean;
  }>(),
  {
    emptyText: '-',
    maxHeight: 420,
    wrap: true,
  },
);

const renderedHtml = ref('');
const themeMode = ref<'dark' | 'light'>(resolveThemeMode());
let themeObserver: MutationObserver | null = null;

const contentStyle = computed(() => ({
  maxHeight: typeof props.maxHeight === 'number' ? `${props.maxHeight}px` : props.maxHeight,
}));

watch(
  () => [props.code, props.lang, themeMode.value] as const,
  async ([code, lang, nextThemeMode]) => {
    if (!code.trim()) {
      renderedHtml.value = '';
      return;
    }

    try {
      renderedHtml.value = await renderHighlightedCodeBlock({
        code,
        lang,
        themeMode: nextThemeMode,
      });
    } catch {
      renderedHtml.value = '';
    }
  },
  { immediate: true },
);

onMounted(() => {
  if (typeof MutationObserver === 'undefined' || typeof document === 'undefined') {
    return;
  }

  themeObserver = new MutationObserver(() => {
    themeMode.value = resolveThemeMode();
  });
  themeObserver.observe(document.documentElement, {
    attributeFilter: ['theme-mode'],
    attributes: true,
  });
});

onBeforeUnmount(() => {
  themeObserver?.disconnect();
  themeObserver = null;
});

function resolveThemeMode() {
  if (typeof document === 'undefined') {
    return 'light';
  }

  return document.documentElement.getAttribute('theme-mode') === 'dark' ? 'dark' : 'light';
}
</script>
<style scoped lang="less">
.graft-code-block {
  min-width: 0;
}

.graft-code-block__content,
.graft-code-block__fallback,
.graft-code-block__empty {
  background: var(--td-bg-color-container-hover);
  border: 1px solid var(--td-border-level-1-color);
  border-radius: var(--td-radius-medium);
  box-sizing: border-box;
  margin: 0;
  overflow: auto;
  padding: var(--graft-density-gap-12);
  width: 100%;
}

.graft-code-block__empty {
  color: var(--td-text-color-secondary);
}

.graft-code-block__content :deep(pre),
.graft-code-block__fallback {
  background: transparent !important;
  border: 0 !important;
  color: var(--td-text-color-primary);
  font-family: var(--td-font-family-mono, monospace);
  font-size: var(--td-font-size-body-small);
  line-height: 1.6;
  margin: 0;
  min-width: 0;
  padding: 0 !important;
}

.graft-code-block__content :deep(code),
.graft-code-block__fallback code {
  background: transparent;
  font: inherit;
}

.graft-code-block--wrap .graft-code-block__content :deep(pre),
.graft-code-block--wrap .graft-code-block__fallback {
  overflow-wrap: anywhere;
  white-space: pre-wrap !important;
}

.graft-code-block:not(.graft-code-block--wrap) .graft-code-block__content :deep(pre),
.graft-code-block:not(.graft-code-block--wrap) .graft-code-block__fallback {
  white-space: pre !important;
}
</style>
