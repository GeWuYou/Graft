import { computed, type ComputedRef, type MaybeRefOrGetter } from 'vue';

import { useContainerSize } from '@/shared/composables/useContainerSize';

import { type QueryLayoutResult, resolveQueryLayout } from './layout-engine';
import type { ResourceQueryFilterDefinition } from './types';

/** 组件容器尺寸由 shared responsive 基础设施提供，renderer 不直接观测窗口或容器。 */
export function useQueryPanelLayout(
  target: MaybeRefOrGetter<HTMLElement | null | undefined>,
  fields: ComputedRef<ResourceQueryFilterDefinition[]>,
): ComputedRef<QueryLayoutResult> {
  const size = useContainerSize(target);
  return computed(() => resolveQueryLayout(fields.value, size.value.width));
}
