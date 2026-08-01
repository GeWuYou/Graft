import type { ResourceQueryState } from './resource-query/types';

export type ResourceQueryStateSource = 'default-view' | 'defaults' | 'recent' | 'url';

export const RESOURCE_QUERY_PAGE_SIZES = [10, 20, 50, 100] as const;

export type ResourceQueryStateResolution = {
  source: ResourceQueryStateSource;
  state: ResourceQueryState;
};

export type ResourceQueryStateRestoreOptions = {
  defaultState: ResourceQueryState;
  defaultViewState?: ResourceQueryState;
  recentState?: ResourceQueryState;
  urlState?: ResourceQueryState;
};

/** 返回页面稳定的最近查询存储键，surface 由列表页面 owner 定义。 */
export function resourceQueryStorageKey(surface: string) {
  return `graft.query.${surface}`;
}

/** 按 URL、默认视图、最近查询和静态默认值的优先级恢复查询状态。 */
export function resolveResourceQueryState(options: ResourceQueryStateRestoreOptions): ResourceQueryStateResolution {
  const candidates: Array<[ResourceQueryStateSource, ResourceQueryState | undefined]> = [
    ['url', options.urlState],
    ['default-view', options.defaultViewState],
    ['recent', options.recentState],
    ['defaults', options.defaultState],
  ];
  const [source, state] = candidates.find(([, candidate]) => candidate !== undefined) as [
    ResourceQueryStateSource,
    ResourceQueryState,
  ];
  return { source, state: cloneResourceQueryState(state) };
}

/** 读取并验证某个页面的最近查询；无效或过期结构直接丢弃。 */
export function readResourceQueryState(surface: string): ResourceQueryState | undefined {
  if (typeof window === 'undefined') return undefined;
  try {
    const raw = window.localStorage.getItem(resourceQueryStorageKey(surface));
    if (!raw) return undefined;
    const value: unknown = JSON.parse(raw);
    return isResourceQueryState(value) ? cloneResourceQueryState(value) : undefined;
  } catch {
    return undefined;
  }
}

/** 保存最近查询时不含当前页，避免返回列表时落在陈旧页码。 */
export function writeResourceQueryState(surface: string, state: ResourceQueryState) {
  if (typeof window === 'undefined') return;
  try {
    const persisted = { ...cloneResourceQueryState(state), page: 1 };
    window.localStorage.setItem(resourceQueryStorageKey(surface), JSON.stringify(persisted));
  } catch {
    // Storage may be disabled or full; querying must remain usable.
  }
}

function cloneResourceQueryState(state: ResourceQueryState): ResourceQueryState {
  return { ...state, filters: { ...state.filters } };
}

function isResourceQueryState(value: unknown): value is ResourceQueryState {
  if (!value || typeof value !== 'object') return false;
  const candidate = value as Record<string, unknown>;
  return (
    typeof candidate.keyword === 'string' &&
    typeof candidate.page === 'number' &&
    Number.isInteger(candidate.page) &&
    candidate.page > 0 &&
    typeof candidate.pageSize === 'number' &&
    Number.isInteger(candidate.pageSize) &&
    RESOURCE_QUERY_PAGE_SIZES.includes(candidate.pageSize as (typeof RESOURCE_QUERY_PAGE_SIZES)[number]) &&
    candidate.filters !== null &&
    typeof candidate.filters === 'object' &&
    !Array.isArray(candidate.filters)
  );
}
