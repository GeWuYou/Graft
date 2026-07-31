import { flushPromises } from '@vue/test-utils';
import { createPinia, setActivePinia } from 'pinia';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import { getUpdateOperationDiagnostic, subscribeToUpdateOperation } from '../api/update';
import { useUpdateProgressStore } from './progress';

vi.mock('../api/update', () => ({
  getUpdateOperationDiagnostic: vi.fn(),
  subscribeToUpdateOperation: vi.fn(),
}));

const operation = (operationID: string, status = 'PLANNING') => ({ operation_id: operationID, status }) as never;
const streamController = () => ({ close: vi.fn(), reconnect: vi.fn() });

describe('update progress store', () => {
  beforeEach(() => {
    sessionStorage.clear();
    setActivePinia(createPinia());
    vi.clearAllMocks();
    vi.mocked(subscribeToUpdateOperation).mockImplementation(streamController);
  });

  it('uses one authenticated stream instead of polling when an update begins', () => {
    const store = useUpdateProgressStore();

    store.begin(operation('operation-1'));

    expect(subscribeToUpdateOperation).toHaveBeenCalledWith(
      'operation-1',
      expect.objectContaining({ onOperation: expect.any(Function), onStateChange: expect.any(Function) }),
    );
    expect(sessionStorage.getItem('graft.platform-update.operation-id')).toBe('operation-1');
  });

  it('discards an old stream event after a new update session begins', async () => {
    const store = useUpdateProgressStore();
    store.begin(operation('operation-1'));
    const firstCallback = vi.mocked(subscribeToUpdateOperation).mock.calls[0][1].onOperation;
    store.begin(operation('operation-2'));

    await firstCallback(operation('operation-1', 'FAILED'));

    expect(store.operation?.operation_id).toBe('operation-2');
    expect(store.phase).toBe('running');
    expect(getUpdateOperationDiagnostic).not.toHaveBeenCalled();
  });

  it('restores an unfinished operation from same-tab session storage', async () => {
    sessionStorage.setItem('graft.platform-update.operation-id', 'operation-1');
    setActivePinia(createPinia());
    const store = useUpdateProgressStore();

    expect(store.visible).toBe(true);
    expect(store.operation?.operation_id).toBe('operation-1');
    store.resume();
    await flushPromises();

    expect(subscribeToUpdateOperation).toHaveBeenCalledWith('operation-1', expect.any(Object));
  });

  it('stops the stream and loads controlled diagnostics after a terminal failure', async () => {
    vi.mocked(getUpdateOperationDiagnostic).mockResolvedValue({ request_id: 'request-1' } as never);
    const store = useUpdateProgressStore();
    store.begin(operation('operation-1'));
    const callback = vi.mocked(subscribeToUpdateOperation).mock.calls[0][1].onOperation;

    await callback(operation('operation-1', 'FAILED'));

    expect(store.phase).toBe('failed');
    expect(store.diagnostic?.request_id).toBe('request-1');
    expect(sessionStorage.getItem('graft.platform-update.operation-id')).toBeNull();
  });
});
