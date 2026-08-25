<template>
  <div
    v-if="compact && (controller.topVisible.value || controller.bottomVisible.value)"
    ref="scrollEdgeActionsRef"
    class="scroll-edge-actions"
    data-scroll-edge-actions="true"
    role="group"
    :aria-label="labels.group"
  >
    <t-button
      v-if="controller.topVisible.value"
      key="scroll-edge-to-top"
      class="scroll-edge-actions__button"
      block
      shape="circle"
      size="medium"
      theme="primary"
      :aria-label="labels.toTop"
      :title="labels.toTop"
      data-scroll-edge-to-top="true"
      @click="handleScrollToTop"
    >
      <template #icon>
        <slot name="top-icon"><arrow-up-icon size="20" /></slot>
      </template>
    </t-button>
    <t-button
      v-if="controller.bottomVisible.value"
      key="scroll-edge-to-bottom"
      class="scroll-edge-actions__button"
      block
      shape="circle"
      size="medium"
      theme="primary"
      :aria-label="labels.toBottom"
      :title="labels.toBottom"
      data-scroll-edge-to-bottom="true"
      @click="handleScrollToBottom"
    >
      <template #icon>
        <slot name="bottom-icon"><arrow-down-icon size="20" /></slot>
      </template>
    </t-button>
  </div>
</template>
<script setup lang="ts">
import { ArrowDownIcon, ArrowUpIcon } from 'tdesign-icons-vue-next';
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue';

import {
  type ScrollEdgeActionsController,
  type ScrollEdgeActionsOptions,
  useScrollEdgeActions,
} from '@/shared/composables';
import { useViewportResponsiveVariant } from '@/shared/composables/useViewportResponsiveVariant';
import { emitScrollEdgeDebug } from '@/shared/debug/scroll-edge-actions-investigation';

const props = withDefaults(
  defineProps<
    ScrollEdgeActionsOptions & {
      controller?: ScrollEdgeActionsController;
      compact?: boolean;
      labels: { toTop: string; toBottom: string; group?: string };
    }
  >(),
  {
    behavior: 'smooth',
    compact: undefined,
    controller: undefined,
    threshold: 4,
  },
);

const internalController = useScrollEdgeActions(props);
const controller = computed(() => props.controller ?? internalController);
const viewportVariant = useViewportResponsiveVariant();
const compact = computed(() => props.compact ?? viewportVariant.value.density === 'compact');
const labels = computed(() => props.labels);
const scrollEdgeActionsRef = ref<HTMLElement | null>(null);

function rounded(value: number) {
  return Math.round(value * 100) / 100;
}

function measureElement(element: Element | null) {
  if (!(element instanceof HTMLElement)) {
    return undefined;
  }

  const rect = element.getBoundingClientRect();
  return {
    left: rounded(rect.left),
    right: rounded(rect.right),
    width: rounded(rect.width),
  };
}

async function emitLayoutDebug(event: string) {
  await nextTick();
  const root = scrollEdgeActionsRef.value;
  const topButton = root?.querySelector('[data-scroll-edge-to-top]') ?? null;
  const bottomButton = root?.querySelector('[data-scroll-edge-to-bottom]') ?? null;
  const rootRect = measureElement(root);
  const topRect = measureElement(topButton);
  const bottomRect = measureElement(bottomButton);

  emitScrollEdgeDebug('STATE_CHANGE', event, {
    compact: compact.value,
    buttonCount: Number(Boolean(topButton)) + Number(Boolean(bottomButton)),
    rootLeft: rootRect?.left,
    rootRight: rootRect?.right,
    rootWidth: rootRect?.width,
    topLeft: topRect?.left,
    topRight: topRect?.right,
    topWidth: topRect?.width,
    bottomLeft: bottomRect?.left,
    bottomRight: bottomRect?.right,
    bottomWidth: bottomRect?.width,
    targetAttached: Boolean(controller.value.metrics.value.target),
    scrollTop: controller.value.metrics.value.scrollTop,
    isScrollable: controller.value.metrics.value.isScrollable,
    atTop: controller.value.metrics.value.atTop,
    atBottom: controller.value.metrics.value.atBottom,
  });
}

function handleScrollToTop() {
  emitScrollEdgeDebug('USER_ACTION', 'scroll-to-top-click', { action: 'top' });
  controller.value.scrollToTop();
}

function handleScrollToBottom() {
  emitScrollEdgeDebug('USER_ACTION', 'scroll-to-bottom-click', { action: 'bottom' });
  controller.value.scrollToBottom();
}

watch(
  () => [compact.value, controller.value.topVisible.value, controller.value.bottomVisible.value],
  () => {
    void emitLayoutDebug('visibility-changed');
  },
  { immediate: true },
);

onMounted(() => {
  void emitLayoutDebug('mounted');
});

onBeforeUnmount(() => {
  emitScrollEdgeDebug('LIFECYCLE', 'unmounted');
});
</script>
<style scoped lang="less">
.scroll-edge-actions {
  --scroll-edge-actions-size: var(--graft-responsive-touch-target-min, 2.75rem);

  align-items: center;
  background: color-mix(in srgb, var(--td-bg-color-container) 90%, var(--td-brand-color) 10%);
  border: 1px solid color-mix(in srgb, var(--td-brand-color) 24%, var(--td-component-stroke));
  border-radius: var(--td-radius-round);
  bottom: calc(var(--graft-mobile-navigation-safe-area, 0px) + var(--td-comp-margin-xl, 24px));
  box-shadow: var(--td-shadow-2);
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-8);
  inline-size: var(--scroll-edge-actions-size);
  padding: var(--graft-density-gap-4);
  pointer-events: none;
  position: fixed;
  right: var(--graft-shell-floating-actions-right, 16px);
  width: var(--scroll-edge-actions-size);
  z-index: 1090;
}

.scroll-edge-actions__button {
  align-items: center;
  display: inline-flex;
  flex: 0 0 var(--scroll-edge-actions-size);
  height: var(--scroll-edge-actions-size);
  justify-content: center;
  margin: 0;
  min-width: var(--scroll-edge-actions-size);
  padding: 0;
  pointer-events: auto;
  width: var(--scroll-edge-actions-size);
}

.scroll-edge-actions :deep(.scroll-edge-actions__button.t-button) {
  height: var(--scroll-edge-actions-size);
  margin: 0 !important;
  min-width: var(--scroll-edge-actions-size);
  width: var(--scroll-edge-actions-size);
}
</style>
