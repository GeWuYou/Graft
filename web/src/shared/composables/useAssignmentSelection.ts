import { computed, type ComputedRef, type Ref, ref, watch } from 'vue';

export type AssignmentMutationMode = 'replace' | 'add' | 'remove';

type MaybeRefIds = Ref<number[]> | ComputedRef<number[]>;

type UseAssignmentSelectionOptions = {
  active?: Ref<boolean>;
  mode: Ref<AssignmentMutationMode>;
  originalIds: MaybeRefIds;
};

function sortIds(ids: number[]) {
  return ids.slice().sort((left, right) => left - right);
}

export function useAssignmentSelection(options: UseAssignmentSelectionOptions) {
  const selectedIds = ref<number[]>([]);

  const resetSelection = () => {
    selectedIds.value = options.mode.value === 'replace' ? [...options.originalIds.value] : [];
  };

  watch(
    () => [options.active?.value ?? true, options.mode.value, options.originalIds.value.join(',')] as const,
    ([active]) => {
      if (!active) {
        return;
      }

      resetSelection();
    },
    { immediate: true },
  );

  const mutationIds = computed(() => {
    const original = new Set(options.originalIds.value);

    switch (options.mode.value) {
      case 'add':
        return sortIds(selectedIds.value.filter((id) => !original.has(id)));
      case 'remove':
        return sortIds(selectedIds.value.filter((id) => original.has(id)));
      default:
        return sortIds(selectedIds.value);
    }
  });

  return {
    mutationIds,
    resetSelection,
    selectedIds,
  };
}
