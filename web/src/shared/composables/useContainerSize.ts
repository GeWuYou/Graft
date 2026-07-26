import { type MaybeRefOrGetter, readonly, type Ref, ref, toValue, watch } from 'vue';

import {
  EMPTY_RESPONSIVE_CONTAINER_SIZE,
  normalizeResponsiveContainerSize,
  type ResponsiveContainerSize,
} from '@/shared/responsive';

/**
 * 以元素容器为唯一尺寸信号，避免 Split、Drawer 和 Dialog 内的内容误用视口宽度。
 */
export function useContainerSize(
  target: MaybeRefOrGetter<HTMLElement | null | undefined>,
): Readonly<Ref<ResponsiveContainerSize>> {
  const size = ref<ResponsiveContainerSize>({ ...EMPTY_RESPONSIVE_CONTAINER_SIZE });
  let observer: ResizeObserver | undefined;

  watch(
    () => toValue(target),
    (element, _, onCleanup) => {
      observer?.disconnect();
      observer = undefined;
      size.value = { ...EMPTY_RESPONSIVE_CONTAINER_SIZE };

      if (!element) {
        return;
      }

      size.value = normalizeResponsiveContainerSize(element.clientWidth, element.clientHeight);

      if (typeof ResizeObserver === 'undefined') {
        return;
      }

      observer = new ResizeObserver((entries) => {
        const entry = entries[0];
        if (!entry) {
          return;
        }

        size.value = normalizeResponsiveContainerSize(entry.contentRect.width, entry.contentRect.height);
      });
      observer.observe(element);
      onCleanup(() => observer?.disconnect());
    },
    { immediate: true },
  );

  return readonly(size);
}
