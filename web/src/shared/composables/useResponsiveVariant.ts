import { computed, type ComputedRef, type MaybeRefOrGetter, toValue } from 'vue';

import { resolveResponsiveVariant, type ResponsiveVariant, type ResponsiveVariantOptions } from '@/shared/responsive';

import { useContainerSize } from './useContainerSize';

/**
 * 将容器宽度收敛为可组合的布局语义；调用方不会获得或派生设备类型布尔值。
 */
export function useResponsiveVariant(
  target: MaybeRefOrGetter<HTMLElement | null | undefined>,
  options: MaybeRefOrGetter<ResponsiveVariantOptions> = {},
): ComputedRef<ResponsiveVariant> {
  const size = useContainerSize(target);

  return computed(() => resolveResponsiveVariant(size.value.width, toValue(options)));
}
