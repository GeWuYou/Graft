import { describe, expect, it, vi } from 'vitest';

import type { ResourceQueryState } from '@/shared/components/query-list';

import { useDockerResourceSavedViews } from './docker-resource-saved-views';

const initialState: ResourceQueryState = {
  keyword: 'running',
  filters: { usage: 'used' },
  page: 3,
  pageSize: 50,
};

function persistedView(
  overrides: Partial<{ id: number; name: string; is_default: boolean; query_state: unknown }> = {},
) {
  return {
    id: overrides.id ?? 1,
    name: overrides.name ?? 'Running containers',
    page_size: 50,
    query_state: overrides.query_state ?? initialState,
    visible_columns: [],
    is_default: overrides.is_default ?? true,
  };
}

describe('docker resource saved views', () => {
  it('loads and applies one default view when the page has no explicit state', async () => {
    const applied: ResourceQueryState[] = [];
    const controller = useDockerResourceSavedViews({
      api: {
        list: vi.fn().mockResolvedValue([persistedView()]),
        create: vi.fn(),
        update: vi.fn(),
        remove: vi.fn(),
      },
      applyState: (state) => applied.push(state.queryState),
      getState: () => ({ pageSize: initialState.pageSize, queryState: initialState, visibleColumns: [] }),
    });

    await expect(controller.load({ hasExplicitState: false })).resolves.toBe(true);
    expect(applied).toEqual([initialState]);
    expect(controller.selectedId.value).toBe(1);
  });

  it('round-trips create, update, and delete through the module adapter', async () => {
    const api = {
      list: vi.fn().mockResolvedValue([]),
      create: vi.fn().mockResolvedValue(persistedView({ id: 2, name: 'Created', is_default: false })),
      update: vi.fn().mockResolvedValue(persistedView({ id: 2, name: 'Updated', is_default: true })),
      remove: vi.fn().mockResolvedValue(undefined),
    };
    const controller = useDockerResourceSavedViews({
      api,
      applyState: vi.fn(),
      getState: () => ({ pageSize: initialState.pageSize, queryState: initialState, visibleColumns: [] }),
    });

    await controller.load({ hasExplicitState: true });
    await expect(controller.save('Created', 'create')).resolves.toBe(true);
    await expect(controller.save('Updated', 'update', true)).resolves.toBe(true);
    await expect(controller.removeSelected()).resolves.toBe(true);

    expect(api.create).toHaveBeenCalledWith(
      expect.objectContaining({ page_size: 50, query_state: initialState, is_default: false }),
    );
    expect(api.update).toHaveBeenCalledWith(2, expect.objectContaining({ name: 'Updated', is_default: true }));
    expect(api.remove).toHaveBeenCalledWith(2);
    expect(controller.views.value).toEqual([]);
  });
});
