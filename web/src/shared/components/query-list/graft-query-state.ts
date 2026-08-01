import type { GraftQueryState } from './graft-query-bar';

export type GraftQueryStateSource = 'default-view' | 'defaults' | 'recent' | 'url';

export type GraftQueryStateResolution = {
  source: GraftQueryStateSource;
  state: GraftQueryState;
};

export type GraftQueryStateRestoreOptions = {
  defaultState: GraftQueryState;
  defaultViewState?: GraftQueryState;
  recentState?: GraftQueryState;
  urlState?: GraftQueryState;
};

/** 返回页面稳定的最近查询存储键，surface 由列表页面 owner 定义。 */
export function graftQueryStorageKey(surface: string) {
  return `graft.query.${surface}`;
}

/** 按 URL、默认视图、最近查询和静态默认值的优先级恢复查询状态。 */
export function resolveGraftQueryState(options: GraftQueryStateRestoreOptions): GraftQueryStateResolution {
  const candidates: Array<[GraftQueryStateSource, GraftQueryState | undefined]> = [
    ['url', options.urlState],
    ['default-view', options.defaultViewState],
    ['recent', options.recentState],
    ['defaults', options.defaultState],
  ];
  const [source, state] = candidates.find(([, candidate]) => candidate !== undefined) as [
    GraftQueryStateSource,
    GraftQueryState,
  ];
  return { source, state: cloneGraftQueryState(state) };
}

/** 读取并验证某个页面的最近查询；无效或过期结构直接丢弃。 */
export function readGraftQueryState(surface: string): GraftQueryState | undefined {
  if (typeof window === 'undefined') return undefined;
  try {
    const raw = window.localStorage.getItem(graftQueryStorageKey(surface));
    if (!raw) return undefined;
    const value: unknown = JSON.parse(raw);
    return isGraftQueryState(value) ? cloneGraftQueryState(value) : undefined;
  } catch {
    return undefined;
  }
}

/** 保存最近查询时不含当前页，避免返回列表时落在陈旧页码。 */
export function writeGraftQueryState(surface: string, state: GraftQueryState) {
  if (typeof window === 'undefined') return;
  try {
    const persisted = { ...cloneGraftQueryState(state), page: 1 };
    window.localStorage.setItem(graftQueryStorageKey(surface), JSON.stringify(persisted));
  } catch {
    // Storage may be disabled or full; querying must remain usable.
  }
}

export function cloneGraftQueryState(state: GraftQueryState): GraftQueryState {
  return { ...state, filters: { ...state.filters } };
}

function isGraftQueryState(value: unknown): value is GraftQueryState {
  if (!value || typeof value !== 'object') return false;
  const candidate = value as Record<string, unknown>;
  return (
    typeof candidate.keyword === 'string' &&
    typeof candidate.page === 'number' &&
    Number.isInteger(candidate.page) &&
    candidate.page > 0 &&
    typeof candidate.pageSize === 'number' &&
    Number.isInteger(candidate.pageSize) &&
    candidate.pageSize > 0 &&
    candidate.filters !== null &&
    typeof candidate.filters === 'object' &&
    !Array.isArray(candidate.filters)
  );
}
