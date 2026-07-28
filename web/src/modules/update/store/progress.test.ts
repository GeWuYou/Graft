import { flushPromises } from '@vue/test-utils';
import { createPinia, setActivePinia } from 'pinia';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import { getUpdateOperation, getUpdateOperationDiagnostic } from '../api/update';
import { useUpdateProgressStore } from './progress';

vi.mock('../api/update', () => ({
  getUpdateOperation: vi.fn(),
  getUpdateOperationDiagnostic: vi.fn(),
}));

function deferred<T>() {
  let resolve!: (value: T) => void;
  const promise = new Promise<T>((resolvePromise) => {
    resolve = resolvePromise;
  });
  return { promise, resolve };
}

const operation = (operationID: string, status = 'PLANNING') => ({ operation_id: operationID, status }) as never;

describe('update progress store', () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    vi.clearAllMocks();
  });

  it('discards an in-flight poll after a new update session begins', async () => {
    const firstPoll = deferred<never>();
    const secondPoll = deferred<never>();
    vi.mocked(getUpdateOperation).mockReturnValueOnce(firstPoll.promise).mockReturnValueOnce(secondPoll.promise);
    const store = useUpdateProgressStore();

    store.begin(operation('operation-1'));
    store.begin(operation('operation-2'));
    firstPoll.resolve(operation('operation-1', 'FAILED'));
    await flushPromises();

    expect(store.operation?.operation_id).toBe('operation-2');
    expect(store.phase).toBe('running');
    expect(getUpdateOperationDiagnostic).not.toHaveBeenCalled();

    secondPoll.resolve(operation('operation-2', 'PLANNING'));
    await flushPromises();
  });

  it('discards an in-flight poll after the progress dialog is reset', async () => {
    const poll = deferred<never>();
    vi.mocked(getUpdateOperation).mockReturnValueOnce(poll.promise);
    const store = useUpdateProgressStore();

    store.begin(operation('operation-1'));
    store.reset();
    poll.resolve(operation('operation-1', 'FAILED'));
    await flushPromises();

    expect(store.operation).toBeNull();
    expect(store.phase).toBe('idle');
    expect(store.diagnostic).toBeNull();
  });
});
