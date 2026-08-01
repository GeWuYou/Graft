import { computed, type ComputedRef, type Ref, ref } from 'vue';

export type SavedQueryViewId = number | string;

/** 经模块归一化的私有查询视图；它不应被提升为跨模块查询契约。 */
export type SavedQueryView<TState, TId extends SavedQueryViewId = SavedQueryViewId> = {
  id: TId;
  name: string;
  isDefault: boolean;
  state: TState;
};

export type SavedQueryViewInput<TState> = {
  name: string;
  isDefault: boolean;
  state: TState;
};

export type SerializedSavedQueryViewRequest = {
  name: string;
  page_size: number;
  query_state: Record<string, unknown>;
  visible_columns: string[];
  is_default: boolean;
};

/** 将页面状态序列化为后端保存查询视图请求的公共字段。 */
export function serializeSavedQueryViewRequest<
  TState extends {
    pageSize: number;
    queryState: unknown;
    visibleColumns: string[];
  },
>(input: SavedQueryViewInput<TState>): SerializedSavedQueryViewRequest {
  return {
    name: input.name,
    page_size: input.state.pageSize,
    query_state: input.state.queryState as Record<string, unknown>,
    visible_columns: input.state.visibleColumns,
    is_default: input.isDefault,
  };
}

export type PersistedSavedQueryView<TId extends SavedQueryViewId = SavedQueryViewId> = {
  id: TId;
  name: string;
  page_size: number;
  query_state: unknown;
  visible_columns: string[];
  is_default: boolean;
};

/**
 * 将持久化查询视图转换为共享的查询视图状态。
 *
 * @param view - 持久化的查询视图数据
 * @returns 包含标识、名称以及分页大小、查询状态和可见列的标准化视图
 */
export function normalizeSavedQueryView<TState, TId extends SavedQueryViewId = SavedQueryViewId>(
  view: PersistedSavedQueryView<TId>,
): SavedQueryView<{ pageSize: number; queryState: TState; visibleColumns: string[] }, TId> {
  return {
    id: view.id,
    name: view.name,
    isDefault: view.is_default,
    state: {
      pageSize: view.page_size,
      queryState: view.query_state as TState,
      visibleColumns: [...view.visible_columns],
    },
  };
}

/**
 * 根据当前列表支持的列筛选已保存的列键。
 *
 * @param visibleColumns - 已保存的可见列键
 * @param supportedColumns - 当前列表支持的列键
 * @returns 仅包含当前列表支持的列键
 */
export function resolveSavedQueryViewColumns(visibleColumns: string[], supportedColumns: Iterable<string>) {
  const supported = new Set(supportedColumns);
  return visibleColumns.filter((key) => supported.has(key));
}

export type SavedQueryViewPresentationTarget = {
  pagination: { current: number; pageSize: number };
  supportedColumns: Iterable<string>;
  visibleColumnKeys: Ref<string[]>;
};

/**
 * 应用保存的分页大小和可见列偏好，并将当前页重置为第一页。
 *
 * @param state - 保存的分页大小和可见列配置
 * @param target - 接收展示配置的目标对象
 */
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
  save: (name: string, mode: 'create' | 'update', isDefault?: boolean) => Promise<boolean>;
  selectedId: Ref<TId | undefined>;
  selectedView: ComputedRef<SavedQueryView<TState, TId> | undefined>;
  select: (id: SavedQueryViewId | undefined) => Promise<boolean>;
  submitting: Ref<boolean>;
  views: Ref<SavedQueryView<TState, TId>[]>;
};

/**
 * 管理服务端持久化的私有查询视图及其选择、应用、保存和删除操作。
 *
 * @param options - 配置视图适配器、当前状态序列化方式、视图应用行为及操作回调
 * @returns 用于管理查询视图、选中状态和异步操作状态的控制器
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

  /**
   * 加载并更新已保存的查询视图列表。
   *
   * @returns 成功加载时为 `true`，发生错误时为 `false`。
   */
  async function load() {
    if (isBusy.value) {
      return false;
    }

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

  /**
   * 选择并应用指定的已保存查询视图。
   *
   * @param id - 要选择的视图标识；传入 `undefined` 可清除当前选择
   * @returns `true` 表示选择或应用成功，`false` 表示视图不存在或应用失败
   */
  async function select(id: SavedQueryViewId | undefined) {
    if (id === undefined) {
      selectedId.value = undefined;
      return true;
    }

    if (isBusy.value) {
      return false;
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

  /**
   * 创建或更新当前查询视图。
   *
   * @param name - 查询视图名称，首尾空白会被移除
   * @param mode - 保存模式，创建新视图或更新当前选中的视图
   * @returns 保存成功时为 `true`，否则为 `false`
   */
  async function save(name: string, mode: 'create' | 'update', isDefault = selectedView.value?.isDefault ?? false) {
    const normalizedName = name.trim();
    if (!normalizedName || (mode === 'update' && !selectedView.value) || isBusy.value) {
      return false;
    }

    submitting.value = true;
    const operation: SavedQueryViewOperation = mode;
    try {
      const input: SavedQueryViewInput<TState> = {
        name: normalizedName,
        isDefault,
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

  /**
   * 删除当前选中的查询视图。
   *
   * @returns 删除成功时为 `true`，未选中视图或删除失败时为 `false`
   */
  async function removeSelected() {
    const view = selectedView.value;
    if (!view || isBusy.value) {
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
