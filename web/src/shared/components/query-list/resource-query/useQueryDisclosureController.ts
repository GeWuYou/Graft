import { computed, type ComputedRef, ref, watch } from 'vue';

import type { QueryPanelTier } from './layout-engine';

/** 披露状态属于交互层，Grid renderer 只消费当前应显示的字段。 */
export function useQueryDisclosureController(tier: ComputedRef<QueryPanelTier>) {
  const visible = ref(false);
  const useDrawer = computed(() => tier.value === 'compact');

  watch(useDrawer, (drawer) => {
    if (!drawer) return;
    visible.value = false;
  });

  function open() {
    visible.value = true;
  }

  function toggle() {
    visible.value = !visible.value;
  }

  return { open, toggle, useDrawer, visible };
}
