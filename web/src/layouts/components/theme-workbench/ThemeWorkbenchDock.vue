<template>
  <div
    ref="dockRef"
    class="theme-workbench-dock"
    data-testid="theme-workbench-dock"
    :class="{
      'theme-workbench-dock--placed': Boolean(dockStyle),
      'theme-workbench-dock--dragging': dragState.isDragging,
      'theme-workbench-dock--armed': dragState.isPressing,
    }"
    :style="dockStyle"
  >
    <t-button
      class="theme-workbench-dock__main"
      :class="{ 'theme-workbench-dock__main--active': settingStore.showThemeWorkbench }"
      :title="dockMainTitle"
      variant="outline"
      @click="handleClick"
      @pointercancel="handlePointerCancel"
      @pointerdown="handlePointerDown"
      @pointermove="handlePointerMove"
      @pointerup="handlePointerUp"
    >
      <template #icon>
        <t-icon name="palette" size="20px" />
      </template>
      <span v-if="settingStore.showThemeWorkbench" class="theme-workbench-dock__action-label">
        {{ t('layout.setting.workbench.dock.title') }}
      </span>
    </t-button>
  </div>
</template>
<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, reactive, ref } from 'vue';

import { t } from '@/locales';
import { useSettingStore } from '@/store';

const LONG_PRESS_DELAY = 450;
const DRAG_DISTANCE_THRESHOLD = 8;
const VIEWPORT_MARGIN = 12;

const settingStore = useSettingStore();
const dockRef = ref<HTMLElement>();
const suppressNextClick = ref(false);
const dragState = reactive({
  isPressing: false,
  isDragging: false,
  pointerId: -1,
  startClientX: 0,
  startClientY: 0,
  currentClientX: 0,
  currentClientY: 0,
  offsetX: 0,
  offsetY: 0,
});

let longPressTimer: number | undefined;

const dockMainTitle = computed(() => {
  if (dragState.isDragging) {
    return undefined;
  }

  if (settingStore.showThemeWorkbench && settingStore.activeThemeWorkbenchGroup === 'overview') {
    return undefined;
  }

  return t('layout.setting.workbench.dock.title');
});

const dockStyle = computed(() => {
  const position = settingStore.themeWorkbenchDockPosition;
  if (!position) {
    return undefined;
  }

  return {
    left: `${position.xRatio * 100}%`,
    top: `${position.yRatio * 100}%`,
    bottom: 'auto',
    transform: 'translate(-50%, -50%)',
  };
});

const clearLongPressTimer = () => {
  if (longPressTimer === undefined) {
    return;
  }

  window.clearTimeout(longPressTimer);
  longPressTimer = undefined;
};

const getDockCenter = () => {
  const element = dockRef.value;
  if (!element) {
    return {
      x: window.innerWidth / 2,
      y: window.innerHeight - 56,
    };
  }

  const rect = element.getBoundingClientRect();
  return {
    x: rect.left + rect.width / 2,
    y: rect.top + rect.height / 2,
  };
};

const clampDockCenter = (clientX: number, clientY: number) => {
  const rect = dockRef.value?.getBoundingClientRect();
  const halfWidth = (rect?.width ?? 48) / 2;
  const halfHeight = (rect?.height ?? 48) / 2;
  const minX = VIEWPORT_MARGIN + halfWidth;
  const maxX = window.innerWidth - VIEWPORT_MARGIN - halfWidth;
  const minY = VIEWPORT_MARGIN + halfHeight;
  const maxY = window.innerHeight - VIEWPORT_MARGIN - halfHeight;

  return {
    x: Math.min(Math.max(clientX, minX), Math.max(minX, maxX)),
    y: Math.min(Math.max(clientY, minY), Math.max(minY, maxY)),
  };
};

const updateDockPosition = (clientX: number, clientY: number) => {
  const nextCenter = clampDockCenter(clientX, clientY);
  settingStore.setThemeWorkbenchDockPosition({
    xRatio: nextCenter.x / window.innerWidth,
    yRatio: nextCenter.y / window.innerHeight,
  });
};

const startDrag = () => {
  if (!dragState.isPressing || !dockRef.value) {
    return;
  }

  const center = getDockCenter();
  dragState.offsetX = dragState.currentClientX - center.x;
  dragState.offsetY = dragState.currentClientY - center.y;
  dragState.isDragging = true;
  suppressNextClick.value = true;
  updateDockPosition(dragState.currentClientX - dragState.offsetX, dragState.currentClientY - dragState.offsetY);
};

const resetPointerState = () => {
  clearLongPressTimer();
  dragState.isPressing = false;
  dragState.isDragging = false;
  dragState.pointerId = -1;
};

const releasePointerCapture = (event: PointerEvent) => {
  const target = event.currentTarget as HTMLElement;
  if (target.hasPointerCapture(event.pointerId)) {
    target.releasePointerCapture(event.pointerId);
  }
};

const toggleOverview = () => {
  if (settingStore.showThemeWorkbench) {
    settingStore.cancelThemeDraft();
    return;
  }

  settingStore.openThemeWorkbench('overview');
};

const handleClick = () => {
  if (suppressNextClick.value) {
    suppressNextClick.value = false;
    return;
  }

  toggleOverview();
};

const handlePointerDown = (event: PointerEvent) => {
  if (event.button !== 0) {
    return;
  }

  (event.currentTarget as HTMLElement).setPointerCapture(event.pointerId);
  dragState.isPressing = true;
  dragState.isDragging = false;
  dragState.pointerId = event.pointerId;
  dragState.startClientX = event.clientX;
  dragState.startClientY = event.clientY;
  dragState.currentClientX = event.clientX;
  dragState.currentClientY = event.clientY;

  longPressTimer = window.setTimeout(startDrag, LONG_PRESS_DELAY);
};

const handlePointerMove = (event: PointerEvent) => {
  if (!dragState.isPressing || event.pointerId !== dragState.pointerId) {
    return;
  }

  dragState.currentClientX = event.clientX;
  dragState.currentClientY = event.clientY;

  const moveDistance = Math.hypot(event.clientX - dragState.startClientX, event.clientY - dragState.startClientY);
  if (!dragState.isDragging && moveDistance > DRAG_DISTANCE_THRESHOLD) {
    clearLongPressTimer();
    dragState.isPressing = false;
    return;
  }

  if (!dragState.isDragging) {
    return;
  }

  event.preventDefault();
  updateDockPosition(event.clientX - dragState.offsetX, event.clientY - dragState.offsetY);
};

const handlePointerUp = (event: PointerEvent) => {
  if (event.pointerId !== dragState.pointerId) {
    return;
  }

  releasePointerCapture(event);

  if (dragState.isDragging) {
    event.preventDefault();
    updateDockPosition(event.clientX - dragState.offsetX, event.clientY - dragState.offsetY);
  }

  resetPointerState();
};

const handlePointerCancel = (event: PointerEvent) => {
  if (event.pointerId !== dragState.pointerId) {
    return;
  }

  releasePointerCapture(event);

  resetPointerState();
};

const clampPersistedPosition = () => {
  if (!settingStore.themeWorkbenchDockPosition) {
    return;
  }

  const center = getDockCenter();
  updateDockPosition(center.x, center.y);
};

onMounted(() => {
  window.addEventListener('resize', clampPersistedPosition);
  nextTick(clampPersistedPosition);
});

onBeforeUnmount(() => {
  clearLongPressTimer();
  window.removeEventListener('resize', clampPersistedPosition);
});
</script>
<style lang="less" scoped>
.theme-workbench-dock {
  align-items: center;
  backdrop-filter: blur(22px) saturate(155%);
  background:
    linear-gradient(135deg, rgb(255 255 255 / 58%), rgb(255 255 255 / 20%)),
    color-mix(in srgb, var(--td-bg-color-container) 84%, transparent);
  border: 1px solid color-mix(in srgb, var(--td-component-stroke) 52%, rgb(255 255 255 / 46%));
  border-radius: 28px;
  bottom: calc(24px + env(safe-area-inset-bottom, 0px));
  box-shadow:
    0 14px 34px rgb(15 23 42 / 12%),
    inset 0 1px 0 rgb(255 255 255 / 42%);
  box-sizing: border-box;
  display: inline-flex;
  flex-wrap: nowrap;
  justify-content: center;
  left: 50%;
  max-width: calc(100vw - 24px);
  padding: 6px;
  position: fixed;
  touch-action: none;
  transform: translateX(-50%);
  width: max-content;
  z-index: 1090;
}

.theme-workbench-dock--placed {
  right: auto;
}

.theme-workbench-dock--armed,
.theme-workbench-dock--dragging {
  cursor: grabbing;
}

.theme-workbench-dock--dragging {
  transition: none;
  user-select: none;
}

.theme-workbench-dock__main {
  flex: 0 0 auto;
  min-width: 48px;
}

.theme-workbench-dock__main--active {
  background: color-mix(in srgb, var(--td-brand-color) 10%, var(--td-bg-color-container));
  border-color: color-mix(in srgb, var(--td-brand-color) 24%, transparent);
  box-shadow:
    0 8px 18px color-mix(in srgb, var(--td-brand-color) 12%, transparent),
    inset 0 1px 0 rgb(255 255 255 / 28%);
  color: var(--td-brand-color);
}

:deep(.t-button--variant-outline) {
  backdrop-filter: blur(14px);
  background: color-mix(in srgb, var(--td-bg-color-container) 78%, transparent);
  border-color: color-mix(in srgb, var(--td-component-stroke) 44%, transparent);
  box-shadow:
    0 6px 16px rgb(15 23 42 / 7%),
    inset 0 1px 0 rgb(255 255 255 / 28%);
  color: var(--td-text-color-secondary);
  transition:
    transform 0.18s ease,
    border-color 0.18s ease,
    box-shadow 0.18s ease,
    background-color 0.18s ease,
    color 0.18s ease;
}

:deep(.t-button--variant-outline:hover) {
  background: color-mix(in srgb, var(--td-bg-color-container) 90%, transparent);
  border-color: color-mix(in srgb, var(--td-brand-color) 14%, var(--td-component-stroke));
  color: var(--td-text-color-primary);
  transform: translateY(-1px);
}

:deep(.theme-workbench-dock__main.t-button) {
  align-items: center;
  border-radius: 999px;
  display: inline-flex;
  font-weight: 600;
  height: 48px;
  justify-content: center;
  overflow: hidden;
  padding-inline: 0;
  transition:
    min-width 0.22s ease,
    width 0.22s ease,
    padding-inline 0.22s ease,
    background-color 0.18s ease,
    border-color 0.18s ease,
    box-shadow 0.18s ease,
    color 0.18s ease,
    transform 0.18s ease;
  width: 48px;
}

:deep(.theme-workbench-dock__main .t-button__content) {
  align-items: center;
  display: inline-flex;
  height: 100%;
  justify-content: center;
  width: 100%;
}

:deep(.theme-workbench-dock__main .t-button__prefix) {
  align-items: center;
  display: inline-flex;
  height: 20px;
  justify-content: center;
  line-height: 1;
  margin-right: 0;
  width: 20px;
}

:deep(.theme-workbench-dock__main .t-icon) {
  display: block;
  flex: 0 0 auto;
}

:deep(.theme-workbench-dock__main .t-button__text) {
  margin-left: 0;
  max-width: 0;
  opacity: 0;
  overflow: hidden;
  transition:
    max-width 0.22s ease,
    margin-left 0.22s ease,
    opacity 0.16s ease;
  white-space: nowrap;
}

:deep(.theme-workbench-dock__main--active.t-button) {
  min-width: 118px;
  padding-inline: 16px;
  width: auto;
}

:deep(.theme-workbench-dock__main--active .t-button__prefix) {
  margin-right: 8px;
}

:deep(.theme-workbench-dock__main--active .t-button__text) {
  max-width: 88px;
  opacity: 1;
}

@media (width <= 768px) {
  .theme-workbench-dock {
    bottom: 16px;
    padding: 6px;
  }

  :deep(.theme-workbench-dock__main.t-button) {
    height: 44px;
    min-width: 44px;
    width: 44px;
  }

  :deep(.theme-workbench-dock__main--active.t-button) {
    min-width: 108px;
    padding-inline: 14px;
    width: auto;
  }
}
</style>
