<template>
  <t-dialog
    v-if="policy.surface === 'fullscreen'"
    v-bind="dialogBindings"
    @close-btn-click="emitVisible(false)"
    @esc-keydown="emitVisible(false)"
    @overlay-click="emitVisible(false)"
    @update:visible="emitVisible"
  >
    <dialog-content :policy="policy"><slot /></dialog-content>
  </t-dialog>
  <t-drawer
    v-else
    v-bind="drawerBindings"
    @close-btn-click="emitVisible(false)"
    @esc-keydown="emitVisible(false)"
    @overlay-click="emitVisible(false)"
    @update:visible="emitVisible"
  >
    <dialog-content :policy="policy"><slot /></dialog-content>
  </t-drawer>
</template>
<script setup lang="ts">
import { CloseIcon } from 'tdesign-icons-vue-next';
import { Dialog as TDialog } from 'tdesign-vue-next/es/dialog';
import { Drawer as TDrawer } from 'tdesign-vue-next/es/drawer';
import { computed, defineComponent, h, resolveComponent, useSlots } from 'vue';

import { useViewportResponsiveVariant } from '@/shared/composables';
import { RESPONSIVE_STYLE_TOKENS, type ResponsiveStyleToken } from '@/shared/responsive';

import {
  resolveResponsiveDialogPolicy,
  type ResponsiveDialogPurpose,
  type ResponsiveDialogSize,
} from './dialog-policy';

/** Dialog facade 统一选择 TDesign overlay，调用方只声明业务意图与尺寸语义。 */
const {
  closeOnEscKeydown = true,
  closeOnOverlayClick = true,
  purpose = 'detail',
  size = 'medium',
  title,
  visible,
} = defineProps<{
  closeOnEscKeydown?: boolean;
  closeOnOverlayClick?: boolean;
  purpose?: ResponsiveDialogPurpose;
  size?: ResponsiveDialogSize;
  title: string;
  visible: boolean;
}>();
const emit = defineEmits<{ 'update:visible': [visible: boolean] }>();
const viewportVariant = useViewportResponsiveVariant();
const slots = useSlots();
const policy = computed(() => {
  const widthByDensity = { compact: 0, comfortable: 768, spacious: 992 } as const;
  return resolveResponsiveDialogPolicy(widthByDensity[viewportVariant.value.density], purpose, size);
});
const dialogSizeToken = computed<ResponsiveStyleToken>(() => {
  if (size === 'compact') {
    return RESPONSIVE_STYLE_TOKENS.dialogCompactMax;
  }

  if (size === 'large') {
    return RESPONSIVE_STYLE_TOKENS.dialogLargeMax;
  }

  return RESPONSIVE_STYLE_TOKENS.dialogMediumMax;
});
const dialogBindings = computed(() => ({
  closeOnEscKeydown,
  closeOnOverlayClick,
  destroyOnClose: true,
  footer: false,
  visible,
  class: 'responsive-dialog__fullscreen-overlay',
  closeBtn: false,
  header: false,
  width: '100vw',
}));
const drawerBindings = computed(() => {
  const shared = {
    closeOnEscKeydown,
    closeOnOverlayClick,
    destroyOnClose: true,
    footer: false,
    visible,
  };

  return {
    ...shared,
    header: title,
    placement: policy.value.surface === 'sheet' ? ('bottom' as const) : ('right' as const),
    size: policy.value.surface === 'sheet' ? 'auto' : `var(${dialogSizeToken.value})`,
  };
});

const DialogContent = defineComponent({
  name: 'ResponsiveDialogContent',
  props: { policy: { type: Object, required: true } },
  setup() {
    const TButton = resolveComponent('t-button');

    return () =>
      h('section', { class: ['responsive-dialog', `responsive-dialog--${policy.value.surface}`] }, [
        policy.value.surface === 'fullscreen'
          ? h('header', { class: 'responsive-dialog__header responsive-dialog__header--fullscreen' }, [
              h('h1', title),
              h(
                TButton,
                {
                  'aria-label': title,
                  shape: 'square',
                  theme: 'default',
                  variant: 'text',
                  onClick: () => emitVisible(false),
                },
                { icon: () => h(CloseIcon) },
              ),
            ])
          : slots.header
            ? h('header', { class: 'responsive-dialog__header' }, slots.header({ policy: policy.value }))
            : null,
        h('div', { class: 'responsive-dialog__body' }, slots.default?.({ policy: policy.value })),
        slots.footer
          ? h('footer', { class: 'responsive-dialog__footer' }, slots.footer({ policy: policy.value }))
          : null,
      ]);
  },
});

function emitVisible(nextVisible: boolean) {
  emit('update:visible', nextVisible);
}
</script>
<style scoped lang="less">
.responsive-dialog {
  block-size: 100%;
  box-sizing: border-box;
  container-type: inline-size;
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-16);
  max-inline-size: var(--graft-responsive-dialog-max);
  max-width: 100%;
  min-width: 0;
}

.responsive-dialog--sheet,
.responsive-dialog--fullscreen {
  inline-size: 100%;
  max-inline-size: none;
}

.responsive-dialog__header:empty {
  display: none;
}

.responsive-dialog__header--fullscreen {
  align-items: center;
  border-bottom: 1px solid var(--td-component-stroke);
  display: flex;
  gap: var(--graft-density-gap-12);
  justify-content: space-between;
  padding: var(--td-comp-paddingTB-m) var(--td-comp-paddingLR-l);
}

.responsive-dialog__header--fullscreen h1 {
  color: var(--td-text-color-primary);
  font: var(--td-font-title-medium);
  margin: 0;
  min-width: 0;
  overflow-wrap: anywhere;
}

.responsive-dialog__body {
  flex: 1 1 auto;
  min-block-size: 0;
  overflow: auto;
}

.responsive-dialog__footer {
  display: flex;
  flex-wrap: wrap;
  gap: var(--graft-density-gap-12);
}

:deep(.responsive-dialog__fullscreen-overlay .t-dialog) {
  block-size: 100dvh;
  border-radius: 0;
  margin: 0;
  max-block-size: none;
  max-inline-size: none;
  padding: 0;
}

:deep(.responsive-dialog__fullscreen-overlay .t-dialog__body) {
  block-size: 100%;
  padding: 0;
}
</style>
