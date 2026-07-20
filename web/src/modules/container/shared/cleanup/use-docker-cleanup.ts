import { computed, ref } from 'vue';

export type DockerCleanupItem = { id: string; size_bytes?: number | null };
export type CleanupBatchItem = { id: string; success: boolean; error_code?: string };
export type CleanupBatchOutcome = {
  items: CleanupBatchItem[];
  unknownResponseIds: string[];
  requestError?: unknown;
};

type CleanupExecutor = (ids: string[]) => Promise<CleanupBatchOutcome>;

/**
 * 管理 Docker 资源清理的候选快照、跨页选择和批量结果收敛；资源页面负责提供查询与执行适配器。
 */
export function useDockerCleanup<T extends DockerCleanupItem>(options: {
  fetchCandidates: () => Promise<T[]>;
  execute: CleanupExecutor;
  pageSize?: number;
  onOutcome?: (outcome: CleanupBatchOutcome, selected: T[]) => Promise<void> | void;
}) {
  const visible = ref(false);
  const loading = ref(false);
  const removing = ref(false);
  const items = ref<T[]>([]);
  const selectedIds = ref<string[]>([]);
  const previewPage = ref(1);
  const pageSize = options.pageSize ?? 8;
  const pageCount = computed(() => Math.max(1, Math.ceil(items.value.length / pageSize)));
  const previewItems = computed(() =>
    items.value.slice((previewPage.value - 1) * pageSize, previewPage.value * pageSize),
  );
  const selectedSize = computed(() => {
    const selected = new Set(selectedIds.value);
    return items.value.some((item) => item.size_bytes === null || item.size_bytes === undefined)
      ? null
      : items.value.reduce((total, item) => (selected.has(item.id) ? total + (item.size_bytes ?? 0) : total), 0);
  });
  const totalSize = computed(() =>
    items.value.some((item) => item.size_bytes === null || item.size_bytes === undefined)
      ? null
      : items.value.reduce((total, item) => total + (item.size_bytes ?? 0), 0),
  );

  async function open() {
    visible.value = true;
    loading.value = true;
    items.value = [];
    selectedIds.value = [];
    previewPage.value = 1;
    try {
      items.value = await options.fetchCandidates();
      selectedIds.value = items.value.map((item) => item.id);
    } finally {
      loading.value = false;
    }
  }
  async function reconcile(confirmedSuccessfulIds: Set<string>) {
    const candidates = await options.fetchCandidates();
    const candidateIds = new Set(candidates.map((item) => item.id));
    items.value = candidates;
    selectedIds.value = selectedIds.value.filter((id) => candidateIds.has(id) && !confirmedSuccessfulIds.has(id));
    previewPage.value = Math.min(previewPage.value, pageCount.value);
  }
  async function submit() {
    if (!selectedIds.value.length || removing.value) return;
    const selected = items.value.filter((item) => selectedIds.value.includes(item.id)) as T[];
    removing.value = true;
    try {
      const outcome = await options.execute(selected.map((item) => item.id));
      const successfulIds = new Set(outcome.items.filter((item) => item.success).map((item) => item.id));
      selectedIds.value = selectedIds.value.filter((id) => !successfulIds.has(id));
      items.value = items.value.filter((item) => !successfulIds.has(item.id));
      previewPage.value = Math.min(previewPage.value, pageCount.value);
      if (outcome.unknownResponseIds.length) await reconcile(successfulIds);
      await options.onOutcome?.(outcome, selected);
    } finally {
      removing.value = false;
    }
  }
  function select(rowKeys: Array<string | number>) {
    const currentPageIds = new Set(previewItems.value.map((item) => item.id));
    const preserved = selectedIds.value.filter((id) => !currentPageIds.has(id));
    selectedIds.value = [...preserved, ...rowKeys.filter((key) => currentPageIds.has(String(key))).map(String)];
  }
  function clearSelection() {
    selectedIds.value = [];
  }
  function previousPage() {
    previewPage.value = Math.max(1, previewPage.value - 1);
  }
  function nextPage() {
    previewPage.value = Math.min(pageCount.value, previewPage.value + 1);
  }
  return {
    visible,
    loading,
    removing,
    items,
    selectedIds,
    previewPage,
    pageSize,
    pageCount,
    previewItems,
    selectedSize,
    totalSize,
    open,
    reconcile,
    submit,
    select,
    clearSelection,
    previousPage,
    nextPage,
  };
}
