/** 选择模型以资源 ID 描述用户意图，不携带任何领域动作或持久化细节。 */
export type SelectionId = string | number;

export type ExplicitSelection<TId extends SelectionId = SelectionId> = {
  mode: 'explicit';
  selectedIds: Set<TId>;
};

/**
 * AllMatchingSelection 为将来服务端支持“选择当前查询的全部结果”预留。
 * 当前 PagedMultiSelect 不接受此模式，避免在没有查询快照协议时伪造全量选择。
 */
export type AllMatchingSelection<TFilter = Record<string, unknown>, TId extends SelectionId = SelectionId> = {
  excludedIds: Set<TId>;
  filter: TFilter;
  mode: 'all_matching';
};

export type SelectionModel<TId extends SelectionId = SelectionId, TFilter = Record<string, unknown>> =
  ExplicitSelection<TId> | AllMatchingSelection<TFilter, TId>;

/**
 * SelectionSubmitter 将选择意图交给业务持久化策略，组件不决定替换、差异或查询令牌语义。
 */
export interface SelectionSubmitter<
  TId extends SelectionId = SelectionId,
  TFilter = Record<string, unknown>,
  TContext = unknown,
> {
  submit(selection: SelectionModel<TId, TFilter>, context: TContext): Promise<void>;
}

export function createExplicitSelection<TId extends SelectionId>(
  selectedIds: Iterable<TId> = [],
): ExplicitSelection<TId> {
  return { mode: 'explicit', selectedIds: new Set(selectedIds) };
}

/**
 * replaceExplicitPageSelection 只替换当前页范围内的选择，保留其它已访问页面的 ID。
 * 通过返回新的 Set 保持 Vue 对选择模型变化的可观察性。
 */
export function replaceExplicitPageSelection<TId extends SelectionId>(
  selection: ExplicitSelection<TId>,
  pageIds: Iterable<TId>,
  selectedPageIds: Iterable<TId>,
): ExplicitSelection<TId> {
  const pageIdSet = new Set(pageIds);
  const nextIds = new Set(Array.from(selection.selectedIds).filter((id) => !pageIdSet.has(id)));

  for (const id of selectedPageIds) {
    if (pageIdSet.has(id)) {
      nextIds.add(id);
    }
  }

  return createExplicitSelection(nextIds);
}
