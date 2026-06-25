<template>
  <section class="content-viewer-frame" :class="{ 'content-viewer-frame--fullscreen': isFullscreen }">
    <div v-if="isFullscreen" class="content-viewer-frame__backdrop" />
    <div class="content-viewer-frame__panel" :style="panelStyle">
      <header v-if="showHeaderBar" class="content-viewer-frame__header">
        <div class="content-viewer-frame__header-main">
          <slot name="header" :fullscreen="isFullscreen" />
        </div>
        <div class="content-viewer-frame__header-actions">
          <slot name="header-actions" :fullscreen="isFullscreen" :toggle-fullscreen="toggleFullscreen" />
          <t-tooltip :content="isFullscreen ? exitFullscreenLabel : fullscreenLabel" theme="light">
            <t-button
              class="content-viewer-frame__fullscreen-button"
              theme="default"
              variant="outline"
              size="small"
              @click="toggleFullscreen"
            >
              {{ isFullscreen ? exitFullscreenLabel : fullscreenLabel }}
            </t-button>
          </t-tooltip>
        </div>
      </header>

      <div v-if="$slots.toolbar" class="content-viewer-frame__toolbar">
        <slot name="toolbar" :fullscreen="isFullscreen" :toggle-fullscreen="toggleFullscreen" />
      </div>

      <div
        class="content-viewer-frame__surface"
        :class="[
          `content-viewer-frame__surface--${surfacePadding}`,
          `content-viewer-frame__surface--fullscreen-${fullscreenSurfacePadding}`,
        ]"
      >
        <slot :fullscreen="isFullscreen" :toggle-fullscreen="toggleFullscreen" />
      </div>

      <div
        v-if="resizable && !isFullscreen"
        class="content-viewer-frame__resize-handle"
        role="separator"
        :aria-label="resizeHandleLabel"
        aria-orientation="horizontal"
        tabindex="0"
        @keydown.down.prevent="nudgeHeight(-24)"
        @keydown.left.prevent="nudgeHeight(-24)"
        @keydown.up.prevent="nudgeHeight(24)"
        @keydown.right.prevent="nudgeHeight(24)"
        @pointerdown.prevent="startResize"
      >
        <span class="content-viewer-frame__resize-grip" />
      </div>
    </div>
  </section>
</template>
<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, useSlots, watch } from 'vue';

type SurfacePadding = 'none' | 'normal' | 'wide';

const props = withDefaults(
  defineProps<{
    defaultDesktopOffset?: number;
    defaultMobileOffset?: number;
    defaultHeight?: number;
    exitFullscreenLabel: string;
    fullscreenLabel: string;
    fullscreenSurfacePadding?: SurfacePadding;
    minHeight?: number;
    mobileBreakpoint?: number;
    mobileMinHeight?: number;
    resizeHandleLabel: string;
    resizable?: boolean;
    storageKey: string;
    surfacePadding?: SurfacePadding;
  }>(),
  {
    defaultDesktopOffset: 340,
    defaultMobileOffset: 260,
    defaultHeight: 0,
    fullscreenSurfacePadding: 'normal',
    minHeight: 560,
    mobileBreakpoint: 768,
    mobileMinHeight: 420,
    resizable: true,
    surfacePadding: 'normal',
  },
);

const slots = useSlots();
const isFullscreen = ref(false);
const panelHeight = ref(resolveInitialHeight());
const viewportWidth = ref(typeof window !== 'undefined' ? window.innerWidth : 1440);
const viewportHeight = ref(typeof window !== 'undefined' ? window.innerHeight : 1080);
const dragStartY = ref(0);
const dragStartHeight = ref(0);
const storedBodyOverflow = ref('');
const storedHtmlOverflow = ref('');
let removeResizeListeners: (() => void) | null = null;

const showHeaderBar = computed(() => Boolean(slots.header || slots['header-actions']) || true);
const currentMinHeight = computed(() =>
  viewportWidth.value <= props.mobileBreakpoint ? props.mobileMinHeight : props.minHeight,
);
const panelStyle = computed(() => {
  if (isFullscreen.value) {
    return {
      height: 'calc(100vh - 32px)',
    };
  }

  return {
    height: `${clampHeight(panelHeight.value)}px`,
  };
});

watch(isFullscreen, (fullscreen) => {
  if (typeof document === 'undefined') {
    return;
  }

  if (fullscreen) {
    storedBodyOverflow.value = document.body.style.overflow;
    storedHtmlOverflow.value = document.documentElement.style.overflow;
    document.body.style.overflow = 'hidden';
    document.documentElement.style.overflow = 'hidden';
    return;
  }

  document.body.style.overflow = storedBodyOverflow.value;
  document.documentElement.style.overflow = storedHtmlOverflow.value;
});

onMounted(() => {
  if (typeof window === 'undefined') {
    return;
  }

  syncViewport();
  window.addEventListener('resize', handleWindowResize);
  window.addEventListener('keydown', handleWindowKeydown);
});

onBeforeUnmount(() => {
  if (typeof window !== 'undefined') {
    window.removeEventListener('resize', handleWindowResize);
    window.removeEventListener('keydown', handleWindowKeydown);
  }
  stopResize();
  if (typeof document !== 'undefined') {
    document.body.style.overflow = storedBodyOverflow.value;
    document.documentElement.style.overflow = storedHtmlOverflow.value;
  }
});

function resolveInitialHeight() {
  const fallback = resolvePreferredHeight();
  const stored = readStoredHeight();
  if (stored !== null) {
    return stored;
  }
  return props.defaultHeight > 0 ? props.defaultHeight : fallback;
}

function readStoredHeight() {
  if (typeof window === 'undefined') {
    return null;
  }
  const raw = window.localStorage.getItem(props.storageKey);
  if (!raw) {
    return null;
  }
  const parsed = Number(raw);
  if (!Number.isFinite(parsed)) {
    return null;
  }
  return parsed;
}

function writeStoredHeight(value: number) {
  if (typeof window === 'undefined') {
    return;
  }
  window.localStorage.setItem(props.storageKey, String(clampHeight(value)));
}

function resolvePreferredHeight() {
  const height = typeof window !== 'undefined' ? window.innerHeight : 1080;
  const width = typeof window !== 'undefined' ? window.innerWidth : 1440;
  const offset = width <= props.mobileBreakpoint ? props.defaultMobileOffset : props.defaultDesktopOffset;
  const minHeight = width <= props.mobileBreakpoint ? props.mobileMinHeight : props.minHeight;
  return Math.max(minHeight, height - offset);
}

function resolveMaxHeight() {
  return Math.max(currentMinHeight.value, viewportHeight.value - 120);
}

function clampHeight(value: number) {
  return Math.min(resolveMaxHeight(), Math.max(currentMinHeight.value, Math.round(value)));
}

function syncViewport() {
  if (typeof window === 'undefined') {
    return;
  }
  viewportWidth.value = window.innerWidth;
  viewportHeight.value = window.innerHeight;
  panelHeight.value = clampHeight(panelHeight.value || resolvePreferredHeight());
}

function handleWindowResize() {
  syncViewport();
  writeStoredHeight(panelHeight.value);
}

function handleWindowKeydown(event: KeyboardEvent) {
  if (event.key === 'Escape' && isFullscreen.value) {
    isFullscreen.value = false;
  }
}

function toggleFullscreen() {
  isFullscreen.value = !isFullscreen.value;
}

function nudgeHeight(delta: number) {
  if (isFullscreen.value) {
    return;
  }
  panelHeight.value = clampHeight(panelHeight.value + delta);
  writeStoredHeight(panelHeight.value);
}

function startResize(event: PointerEvent) {
  if (!props.resizable || isFullscreen.value || typeof window === 'undefined') {
    return;
  }

  dragStartY.value = event.clientY;
  dragStartHeight.value = panelHeight.value;

  const handlePointerMove = (moveEvent: PointerEvent) => {
    const delta = dragStartY.value - moveEvent.clientY;
    panelHeight.value = clampHeight(dragStartHeight.value + delta);
  };
  const handlePointerUp = () => {
    writeStoredHeight(panelHeight.value);
    stopResize();
  };

  window.addEventListener('pointermove', handlePointerMove);
  window.addEventListener('pointerup', handlePointerUp);
  removeResizeListeners = () => {
    window.removeEventListener('pointermove', handlePointerMove);
    window.removeEventListener('pointerup', handlePointerUp);
  };
}

function stopResize() {
  removeResizeListeners?.();
  removeResizeListeners = null;
}
</script>
<style scoped lang="less">
.content-viewer-frame {
  min-width: 0;
  position: relative;
}

.content-viewer-frame--fullscreen {
  inset: 0;
  position: fixed;
  z-index: 4500;
}

.content-viewer-frame__backdrop {
  backdrop-filter: blur(10px);
  background: color-mix(in srgb, var(--td-mask-active) 80%, transparent);
  inset: 0;
  position: absolute;
}

.content-viewer-frame__panel {
  align-items: stretch;
  background: var(--td-bg-color-container);
  border: 1px solid var(--td-component-stroke);
  border-radius: var(--td-radius-large);
  box-shadow: var(--td-shadow-3);
  contain: layout style;
  display: flex;
  flex-direction: column;
  isolation: isolate;
  min-height: 0;
  min-width: 0;
  overflow: hidden;
  position: relative;
}

.content-viewer-frame--fullscreen .content-viewer-frame__panel {
  border-radius: var(--td-radius-extraLarge);
  height: calc(100vh - 32px);
  inset: 16px;
  position: absolute;
}

.content-viewer-frame__header {
  align-items: flex-start;
  border-bottom: 1px solid var(--td-border-level-1-color);
  display: flex;
  gap: var(--graft-density-gap-12);
  justify-content: space-between;
  min-width: 0;
  padding: var(--graft-density-gap-12) var(--graft-density-gap-16);
}

.content-viewer-frame__header-main {
  flex: 1 1 auto;
  min-width: 0;
}

.content-viewer-frame__header-actions {
  align-items: center;
  display: flex;
  flex: 0 0 auto;
  flex-wrap: wrap;
  gap: var(--graft-density-gap-8);
  justify-content: flex-end;
}

.content-viewer-frame__toolbar {
  border-bottom: 1px solid var(--td-border-level-1-color);
  min-width: 0;
  padding: var(--graft-density-gap-10) var(--graft-density-gap-16);
}

.content-viewer-frame__surface {
  display: flex;
  flex: 1 1 auto;
  flex-direction: column;
  min-height: 0;
  min-width: 0;
}

.content-viewer-frame__surface--none {
  padding: 0;
}

.content-viewer-frame__surface--normal {
  padding: var(--graft-density-gap-12);
}

.content-viewer-frame__surface--wide {
  padding: var(--graft-density-gap-16);
}

.content-viewer-frame--fullscreen .content-viewer-frame__surface--fullscreen-none {
  padding: 0;
}

.content-viewer-frame--fullscreen .content-viewer-frame__surface--fullscreen-normal {
  padding: var(--graft-density-gap-16);
}

.content-viewer-frame--fullscreen .content-viewer-frame__surface--fullscreen-wide {
  padding: var(--graft-density-gap-24) var(--graft-density-gap-32);
}

.content-viewer-frame__resize-handle {
  align-items: center;
  cursor: ns-resize;
  display: flex;
  flex: 0 0 auto;
  justify-content: center;
  min-height: 14px;
  padding: 0 0 var(--graft-density-gap-6);
}

.content-viewer-frame__resize-grip {
  background: color-mix(in srgb, var(--td-text-color-placeholder) 48%, transparent);
  border-radius: 999px;
  display: inline-flex;
  height: 4px;
  width: 52px;
}

.content-viewer-frame__resize-handle:focus-visible .content-viewer-frame__resize-grip,
.content-viewer-frame__resize-handle:hover .content-viewer-frame__resize-grip {
  background: color-mix(in srgb, var(--td-brand-color) 64%, var(--td-text-color-placeholder));
}

@media (width <= 768px) {
  .content-viewer-frame__header {
    align-items: stretch;
    flex-direction: column;
  }

  .content-viewer-frame__header-actions {
    justify-content: flex-start;
  }

  .content-viewer-frame--fullscreen .content-viewer-frame__panel {
    border-radius: 0;
    height: 100vh;
    inset: 0;
  }
}
</style>
