import { computed, type ComputedRef, type Ref, ref } from 'vue';

export type SavedQueryViewId = number | string;

/** A module-normalized private query view. */
export type SavedQueryView<TState, TId extends SavedQueryViewId = SavedQueryViewId> = {
  id: TId;
  name: string;
  state: TState;
};

export type SavedQueryViewInput<TState> = {
  name: string;
  state: TState;
};

export type PersistedSavedQueryView<TId extends SavedQueryViewId = SavedQueryViewId> = {
  id: TId;
  name: string;
  page_size: number;
  query_state: unknown;
  visible_columns: string[];
};

/** Normalizes an owner response into the shared query-view controller state. */
export function normalizeSavedQueryView<TState, TId extends SavedQueryViewId = SavedQueryViewId>(
  view: PersistedSavedQueryView<TId>,
): SavedQueryView<{ pageSize: number; queryState: TState; visibleColumns: string[] }, TId> {
  return {
    id: view.id,
    name: view.name,
    state: {
      pageSize: view.page_size,
      queryState: view.query_state as TState,
      visibleColumns: [...view.visible_columns],
    },
  };
}

/** Filters persisted column keys against the current list's supported columns. */
export function resolveSavedQueryViewColumns(visibleColumns: string[], supportedColumns: Iterable<string>) {
  const supported = new Set(supportedColumns);
  return visibleColumns.filter((key) => supported.has(key));
}

export type SavedQueryViewPresentationTarget = {
  pagination: { current: number; pageSize: number };
  supportedColumns: Iterable<string>;
  visibleColumnKeys: Ref<string[]>;
};

/** Restores page-size and supported column preferences without restoring the current page. */
export function applySavedQueryViewPresentation(
  state: { pageSize: number; visibleColumns: string[] },
  target: SavedQueryViewPresentationTarget,
) {
  target.pagination.pageSize = state.pageSize;
  target.pagination.current = 1;
  target.visibleColumnKeys.value = resolveSavedQueryViewColumns(state.visibleColumns, target.supportedColumns);
}

export type SavedQueryViewAdapter<TState, TId extends SavedQueryViewId = SavedQueryViewId> = {
  list: () => Promise<SavedQueryView<TState, TId>[]>;
  create: (input: SavedQueryViewInput<TState>) => Promise<SavedQueryView<TState, TId>>;
  update: (id: TId, input: SavedQueryViewInput<TState>) => Promise<SavedQueryView<TState, TId>>;
  remove: (id: TId) => Promise<void>;
};

export type SavedQueryViewOperation = 'apply' | 'create' | 'delete' | 'load' | 'update';

export type SavedQueryViewSuccess<TState, TId extends SavedQueryViewId = SavedQueryViewId> = {
  operation: SavedQueryViewOperation;
  view?: SavedQueryView<TState, TId>;
};

export type UseSavedQueryViewsOptions<TState, TId extends SavedQueryViewId = SavedQueryViewId> = {
  adapter: SavedQueryViewAdapter<TState, TId>;
  applyView: (view: SavedQueryView<TState, TId>) => void | Promise<void>;
  onError?: (error: unknown, operation: SavedQueryViewOperation) => void;
  onSuccess?: (result: SavedQueryViewSuccess<TState, TId>) => void;
  serializeCurrentState: () => TState;
};

export type SavedQueryViewController<TState, TId extends SavedQueryViewId = SavedQueryViewId> = {
  applying: Ref<boolean>;
  deleting: Ref<boolean>;
  hasSelectedView: ComputedRef<boolean>;
  isBusy: ComputedRef<boolean>;
  load: () => Promise<boolean>;
  loading: Ref<boolean>;
  removeSelected: () => Promise<boolean>;
  save: (name: string, mode: 'create' | 'update') => Promise<boolean>;
  selectedId: Ref<TId | undefined>;
  selectedView: ComputedRef<SavedQueryView<TState, TId> | undefined>;
  select: (id: SavedQueryViewId | undefined) => Promise<boolean>;
  submitting: Ref<boolean>;
  views: Ref<SavedQueryView<TState, TId>[]>;
};

/**
 * Manages private, server-persisted query views while keeping a page's filter
 * serialization and error presentation in that page's ownership.
 */
export function useSavedQueryViews<TState, TId extends SavedQueryViewId = SavedQueryViewId>(
  options: UseSavedQueryViewsOptions<TState, TId>,
): SavedQueryViewController<TState, TId> {
  const views = ref<SavedQueryView<TState, TId>[]>([]) as Ref<SavedQueryView<TState, TId>[]>;
  const selectedId = ref<TId>();
  const loading = ref(false);
  const submitting = ref(false);
  const deleting = ref(false);
  const applying = ref(false);

  const selectedView = computed(() => views.value.find((view) => view.id === selectedId.value));
  const hasSelectedView = computed(() => selectedView.value !== undefined);
  const isBusy = computed(() => loading.value || submitting.value || deleting.value || applying.value);

  async function load() {
    loading.value = true;
    try {
      const nextViews = await options.adapter.list();
      views.value = nextViews;
      if (!nextViews.some((view) => view.id === selectedId.value)) {
        selectedId.value = undefined;
      }
      options.onSuccess?.({ operation: 'load' });
      return true;
    } catch (error) {
      options.onError?.(error, 'load');
      return false;
    } finally {
      loading.value = false;
    }
  }

  async function select(id: SavedQueryViewId | undefined) {
    if (id === undefined) {
      selectedId.value = undefined;
      return true;
    }

    const view = views.value.find((candidate) => candidate.id === id);
    if (!view) {
      selectedId.value = undefined;
      return false;
    }

    applying.value = true;
    try {
      await options.applyView(view);
      selectedId.value = view.id;
      options.onSuccess?.({ operation: 'apply', view });
      return true;
    } catch (error) {
      options.onError?.(error, 'apply');
      return false;
    } finally {
      applying.value = false;
    }
  }

  async function save(name: string, mode: 'create' | 'update') {
    const normalizedName = name.trim();
    if (!normalizedName || (mode === 'update' && !selectedView.value)) {
      return false;
    }

    submitting.value = true;
    const operation: SavedQueryViewOperation = mode;
    try {
      const input: SavedQueryViewInput<TState> = {
        name: normalizedName,
        state: options.serializeCurrentState(),
      };
      const view =
        mode === 'update' && selectedView.value
          ? await options.adapter.update(selectedView.value.id, input)
          : await options.adapter.create(input);

      const index = views.value.findIndex((candidate) => candidate.id === view.id);
      views.value =
        index === -1
          ? [...views.value, view]
          : views.value.map((candidate) => (candidate.id === view.id ? view : candidate));
      selectedId.value = view.id;
      options.onSuccess?.({ operation, view });
      return true;
    } catch (error) {
      options.onError?.(error, operation);
      return false;
    } finally {
      submitting.value = false;
    }
  }

  async function removeSelected() {
    const view = selectedView.value;
    if (!view) {
      return false;
    }

    deleting.value = true;
    try {
      await options.adapter.remove(view.id);
      views.value = views.value.filter((candidate) => candidate.id !== view.id);
      selectedId.value = undefined;
      options.onSuccess?.({ operation: 'delete', view });
      return true;
    } catch (error) {
      options.onError?.(error, 'delete');
      return false;
    } finally {
      deleting.value = false;
    }
  }

  return {
    applying,
    deleting,
    hasSelectedView,
    isBusy,
    load,
    loading,
    removeSelected,
    save,
    selectedId,
    selectedView,
    select,
    submitting,
    views,
  };
}
