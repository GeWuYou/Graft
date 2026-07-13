import { describe, expect, it, vi } from 'vitest';
import { ref } from 'vue';

import {
  applySavedQueryViewPresentation,
  resolveSavedQueryViewColumns,
  type SavedQueryView,
  type SavedQueryViewController,
  type SavedQueryViewId,
  useSavedQueryViews,
} from './saved-query-views';

type QueryState = {
  keyword: string;
  pageSize: number;
};

const initialView: SavedQueryView<QueryState, number> = {
  id: 1,
  name: 'Running applications',
  state: { keyword: 'running', pageSize: 20 },
};

describe('useSavedQueryViews', () => {
  it('drops persisted columns that the current list does not support', () => {
    expect(resolveSavedQueryViewColumns(['action', 'removed', 'result'], ['action', 'result'])).toEqual([
      'action',
      'result',
    ]);
  });

  it('restores view presentation settings without retaining the current page', () => {
    const visibleColumnKeys = ref(['old']);
    const pagination = { current: 4, pageSize: 10 };

    applySavedQueryViewPresentation(
      { pageSize: 50, visibleColumns: ['action', 'removed'] },
      { pagination, supportedColumns: ['action'], visibleColumnKeys },
    );

    expect(pagination).toEqual({ current: 1, pageSize: 50 });
    expect(visibleColumnKeys.value).toEqual(['action']);
  });

  it('loads, applies, saves, updates, and removes views through the page adapter', async () => {
    const currentState: QueryState = { keyword: 'pending', pageSize: 10 };
    const applyView = vi.fn<(view: SavedQueryView<QueryState, number>) => Promise<void>>().mockResolvedValue();
    const adapter = {
      list: vi.fn().mockResolvedValue([initialView]),
      create: vi.fn().mockResolvedValue({ id: 2, name: 'Pending applications', state: currentState }),
      update: vi.fn().mockResolvedValue({ id: 1, name: 'Updated running applications', state: currentState }),
      remove: vi.fn().mockResolvedValue(undefined),
    };
    const controller = useSavedQueryViews({
      adapter,
      applyView,
      serializeCurrentState: () => currentState,
    });
    const controlController: SavedQueryViewController<unknown, SavedQueryViewId> = controller;
    expect(controlController).toBe(controller);

    await expect(controller.load()).resolves.toBe(true);
    expect(controller.views.value).toEqual([initialView]);

    await expect(controller.select(1)).resolves.toBe(true);
    expect(applyView).toHaveBeenCalledWith(initialView);
    expect(controller.selectedId.value).toBe(1);

    await expect(controller.save(' Pending applications ', 'create')).resolves.toBe(true);
    expect(adapter.create).toHaveBeenCalledWith({ name: 'Pending applications', state: currentState });
    expect(controller.selectedId.value).toBe(2);

    await expect(controller.select(1)).resolves.toBe(true);
    await expect(controller.save('Updated running applications', 'update')).resolves.toBe(true);
    expect(adapter.update).toHaveBeenCalledWith(1, { name: 'Updated running applications', state: currentState });
    expect(controller.selectedView.value?.name).toBe('Updated running applications');

    await expect(controller.removeSelected()).resolves.toBe(true);
    expect(adapter.remove).toHaveBeenCalledWith(1);
    expect(controller.selectedId.value).toBeUndefined();
    expect(controller.views.value.map((view) => view.id)).toEqual([2]);
  });

  it('keeps the selected view when applying a replacement fails and delegates errors to the page', async () => {
    const onError = vi.fn();
    const controller = useSavedQueryViews({
      adapter: {
        list: vi.fn().mockResolvedValue([initialView]),
        create: vi.fn(),
        update: vi.fn(),
        remove: vi.fn(),
      },
      applyView: vi.fn().mockRejectedValue(new Error('invalid view state')),
      onError,
      serializeCurrentState: () => initialView.state,
    });

    await controller.load();
    await expect(controller.select(1)).resolves.toBe(false);
    expect(controller.selectedId.value).toBeUndefined();
    expect(onError).toHaveBeenCalledWith(expect.any(Error), 'apply');
  });

  it('does not invoke a persistence adapter for an empty view name', async () => {
    const create = vi.fn();
    const controller = useSavedQueryViews({
      adapter: {
        list: vi.fn(),
        create,
        update: vi.fn(),
        remove: vi.fn(),
      },
      applyView: vi.fn(),
      serializeCurrentState: () => initialView.state,
    });

    await expect(controller.save('   ', 'create')).resolves.toBe(false);
    expect(create).not.toHaveBeenCalled();
  });
});
