import { describe, expect, it, vi } from 'vitest';
import { ref } from 'vue';

import {
  applySavedQueryViewPresentation,
  resolveDefaultSavedQueryView,
  resolveSavedQueryViewColumns,
  type SavedQueryView,
  type SavedQueryViewController,
  type SavedQueryViewId,
  serializeSavedQueryViewRequest,
  useSavedQueryViews,
} from './saved-query-views';

type QueryState = {
  keyword: string;
  pageSize: number;
};

const initialView: SavedQueryView<QueryState, number> = {
  id: 1,
  name: 'Running applications',
  isDefault: false,
  state: { keyword: 'running', pageSize: 20 },
};

describe('useSavedQueryViews', () => {
  it('applies the unique default view only without explicit URL state', async () => {
    const defaultView = { ...initialView, isDefault: true };
    const applyView = vi.fn();
    const controller = useSavedQueryViews({
      adapter: {
        list: vi.fn().mockResolvedValue([defaultView]),
        create: vi.fn(),
        update: vi.fn(),
        remove: vi.fn(),
      },
      applyView,
      serializeCurrentState: () => initialView.state,
    });

    await expect(controller.load({ hasExplicitState: true })).resolves.toBe(true);
    expect(applyView).not.toHaveBeenCalled();
    await expect(controller.load()).resolves.toBe(true);
    expect(applyView).toHaveBeenCalledWith(defaultView);
    expect(controller.selectedId.value).toBe(defaultView.id);
  });

  it('does not resolve an ambiguous default view set', () => {
    const views = [
      { ...initialView, isDefault: true },
      { ...initialView, id: 2, isDefault: true },
    ];
    expect(resolveDefaultSavedQueryView(views, false)).toBeUndefined();
    expect(resolveDefaultSavedQueryView([{ ...initialView, isDefault: true }], true)).toBeUndefined();
  });

  it('serializes the common persisted request fields', () => {
    expect(
      serializeSavedQueryViewRequest({
        name: 'Pending applications',
        isDefault: true,
        state: { pageSize: 25, queryState: { keyword: 'pending' }, visibleColumns: ['name'] },
      }),
    ).toEqual({
      name: 'Pending applications',
      page_size: 25,
      query_state: { keyword: 'pending' },
      visible_columns: ['name'],
      is_default: true,
    });
  });

  it('blocks busy re-entry while preserving select(undefined) clearing behavior', async () => {
    let resolveList!: (views: SavedQueryView<QueryState, number>[]) => void;
    const list = vi.fn(
      () =>
        new Promise<SavedQueryView<QueryState, number>[]>((resolve) => {
          resolveList = resolve;
        }),
    );
    const adapter = {
      list,
      create: vi.fn().mockResolvedValue({ id: 2, name: 'Pending applications', state: initialView.state }),
      update: vi.fn().mockResolvedValue(initialView),
      remove: vi.fn().mockResolvedValue(undefined),
    };
    const controller = useSavedQueryViews({
      adapter,
      applyView: vi.fn(),
      serializeCurrentState: () => initialView.state,
    });

    const firstLoad = controller.load();
    await expect(controller.load()).resolves.toBe(false);
    await expect(controller.save('Pending applications', 'create')).resolves.toBe(false);
    await expect(controller.removeSelected()).resolves.toBe(false);
    expect(list).toHaveBeenCalledTimes(1);

    await expect(controller.select(undefined)).resolves.toBe(true);
    expect(controller.selectedId.value).toBeUndefined();

    resolveList([initialView]);
    await expect(firstLoad).resolves.toBe(true);

    let resolveApply!: () => void;
    const applyView = vi
      .fn<() => Promise<void>>()
      .mockResolvedValueOnce()
      .mockImplementationOnce(
        () =>
          new Promise<void>((resolve) => {
            resolveApply = resolve;
          }),
      );
    const selectController = useSavedQueryViews({
      adapter: { ...adapter, list: vi.fn().mockResolvedValue([initialView]) },
      applyView,
      serializeCurrentState: () => initialView.state,
    });
    await selectController.load();
    await selectController.select(initialView.id);
    const firstSelect = selectController.select(initialView.id);
    await expect(selectController.select(initialView.id)).resolves.toBe(false);
    await expect(selectController.load()).resolves.toBe(false);
    await expect(selectController.save('Pending applications', 'update')).resolves.toBe(false);
    await expect(selectController.removeSelected()).resolves.toBe(false);
    expect(applyView).toHaveBeenCalledTimes(2);
    expect(selectController.isBusy.value).toBe(true);

    await expect(selectController.select(undefined)).resolves.toBe(true);
    expect(selectController.selectedId.value).toBeUndefined();
    resolveApply();
    await expect(firstSelect).resolves.toBe(true);
  });

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
    expect(adapter.create).toHaveBeenCalledWith({
      name: 'Pending applications',
      isDefault: false,
      state: currentState,
    });
    expect(controller.selectedId.value).toBe(2);

    await expect(controller.select(1)).resolves.toBe(true);
    await expect(controller.save('Updated running applications', 'update')).resolves.toBe(true);
    expect(adapter.update).toHaveBeenCalledWith(1, {
      name: 'Updated running applications',
      isDefault: false,
      state: currentState,
    });
    expect(controller.selectedView.value?.name).toBe('Updated running applications');

    await expect(controller.removeSelected()).resolves.toBe(true);
    expect(adapter.remove).toHaveBeenCalledWith(1);
    expect(controller.selectedId.value).toBeUndefined();
    expect(controller.views.value.map((view) => view.id)).toEqual([2]);
  });

  it('clears other local defaults when the saved result is default', async () => {
    const existingDefault = { ...initialView, isDefault: true };
    const savedView = { ...initialView, id: 2, name: 'Pending applications', isDefault: true };
    const controller = useSavedQueryViews({
      adapter: {
        list: vi.fn().mockResolvedValue([existingDefault]),
        create: vi.fn().mockResolvedValue(savedView),
        update: vi.fn(),
        remove: vi.fn(),
      },
      applyView: vi.fn(),
      serializeCurrentState: () => initialView.state,
    });

    await controller.load({ hasExplicitState: true });
    await expect(controller.save('Pending applications', 'create', true)).resolves.toBe(true);
    expect(controller.views.value).toEqual([{ ...existingDefault, isDefault: false }, savedView]);
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
