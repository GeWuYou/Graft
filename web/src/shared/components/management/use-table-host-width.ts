import { nextTick, onBeforeUnmount, onMounted, type Ref, ref, watch } from 'vue';

export function useTableHostWidth<T>(source: () => T) {
  const tableHostRef = ref<HTMLElement | null>(null);
  const tableHostWidth = ref(0);
  let resizeObserver: ResizeObserver | undefined;

  function updateTableHostWidth() {
    tableHostWidth.value = tableHostRef.value?.clientWidth ?? 0;
  }

  onMounted(() => {
    void nextTick(updateTableHostWidth);

    if (typeof ResizeObserver !== 'undefined' && tableHostRef.value) {
      resizeObserver = new ResizeObserver(updateTableHostWidth);
      resizeObserver.observe(tableHostRef.value);
    }
  });

  onBeforeUnmount(() => {
    resizeObserver?.disconnect();
  });

  watch(source, () => {
    void nextTick(updateTableHostWidth);
  });

  return {
    tableHostRef: tableHostRef as Ref<HTMLElement | null>,
    tableHostWidth,
  };
}
