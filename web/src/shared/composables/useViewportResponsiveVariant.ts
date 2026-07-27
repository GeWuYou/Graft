import { computed, type ComputedRef, onBeforeUnmount, onMounted, ref } from 'vue';

import { resolveResponsiveVariant, type ResponsiveVariant, type ResponsiveVariantOptions } from '@/shared/responsive';

/**
 * 将全局视口宽度转换为共享响应式语义，仅供需要遵循产品视口断点的 shared 组件使用。
 */
export function useViewportResponsiveVariant(options: ResponsiveVariantOptions = {}): ComputedRef<ResponsiveVariant> {
  const viewportWidth = ref(0);

  function updateViewportWidth() {
    viewportWidth.value = window.innerWidth;
  }

  onMounted(() => {
    updateViewportWidth();
    window.addEventListener('resize', updateViewportWidth, { passive: true });
  });

  onBeforeUnmount(() => {
    window.removeEventListener('resize', updateViewportWidth);
  });

  return computed(() => resolveResponsiveVariant(viewportWidth.value, options));
}
