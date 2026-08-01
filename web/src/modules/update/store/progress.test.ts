import { flushPromises } from '@vue/test-utils';
import { createPinia, setActivePinia } from 'pinia';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import { getUpdateOperation, subscribeToUpdateOperation } from '../api/update';
import { useUpdateProgressStore } from './progress';

vi.mock('../api/update', () => ({
  getUpdateOperation: vi.fn(),
  subscribeToUpdateOperation: vi.fn(),
}));

const operation = (operationID: string, phase = 'READY', progress = 0) =>
  ({ operation_id: operationID, runner_id: 'runner-1', phase, progress, message: '' }) as never;
const acknowledgement = (operationID: string) => ({ operation_id: operationID, runner_id: 'runner-1' }) as never;
const streamController = () => ({ close: vi.fn(), reconnect: vi.fn() });

describe('update progress store', () => {
  beforeEach(() => {
    sessionStorage.clear();
    setActivePinia(createPinia());
    vi.clearAllMocks();
    vi.useFakeTimers();
    vi.mocked(getUpdateOperation).mockResolvedValue(operation('operation-1'));
    vi.mocked(subscribeToUpdateOperation).mockImplementation(streamController);
  });

  afterEach(() => vi.useRealTimers());

  it('reads the runner snapshot before subscribing when an update begins', async () => {
    const store = useUpdateProgressStore();

    await store.begin(acknowledgement('operation-1'));

    expect(getUpdateOperation).toHaveBeenCalledWith('operation-1');
    expect(subscribeToUpdateOperation).toHaveBeenCalledWith(
      'operation-1',
      expect.objectContaining({ onOperation: expect.any(Function), onStateChange: expect.any(Function) }),
    );
    expect(sessionStorage.getItem('graft.platform-update.operation-id')).toBe('operation-1');
  });

  it('discards an old snapshot after a new update session begins', async () => {
    let resolveFirst: (value: never) => void = () => undefined;
    vi.mocked(getUpdateOperation).mockImplementationOnce(
      () => new Promise((resolve) => (resolveFirst = resolve as (value: never) => void)),
    );
    vi.mocked(getUpdateOperation).mockResolvedValueOnce(operation('operation-2'));
    const store = useUpdateProgressStore();
    const first = store.begin(acknowledgement('operation-1'));
    await store.begin(acknowledgement('operation-2'));

    resolveFirst(operation('operation-1'));
    await first;

    expect(store.operationID).toBe('operation-2');
    expect(store.operation?.operation_id).toBe('operation-2');
  });

  it('restores an unfinished operation by reading its snapshot before reopening SSE', async () => {
    sessionStorage.setItem('graft.platform-update.operation-id', 'operation-1');
    setActivePinia(createPinia());
    const store = useUpdateProgressStore();

    store.resume();
    await flushPromises();

    expect(getUpdateOperation).toHaveBeenCalledWith('operation-1');
    expect(subscribeToUpdateOperation).toHaveBeenCalledWith('operation-1', expect.any(Object));
  });

  it('stops the stream and preserves a terminal runner failure snapshot', async () => {
    const store = useUpdateProgressStore();
    await store.begin(acknowledgement('operation-1'));
    const callback = vi.mocked(subscribeToUpdateOperation).mock.calls[0][1].onOperation;

    await callback(operation('operation-1', 'FAILED', 100));

    expect(store.phase).toBe('failed');
    expect(store.operation?.phase).toBe('FAILED');
    expect(sessionStorage.getItem('graft.platform-update.operation-id')).toBeNull();
  });

  it('polls the runner snapshot while the realtime transport is unavailable', async () => {
    const store = useUpdateProgressStore();
    await store.begin(acknowledgement('operation-1'));
    const onStateChange = vi.mocked(subscribeToUpdateOperation).mock.calls[0][1].onStateChange;

    onStateChange?.('closed');
    await vi.advanceTimersByTimeAsync(3000);

    expect(store.phase).toBe('running');
    expect(getUpdateOperation).toHaveBeenCalledTimes(2);
  });

  it('stops fallback polling after the realtime transport reopens', async () => {
    const store = useUpdateProgressStore();
    await store.begin(acknowledgement('operation-1'));
    const onStateChange = vi.mocked(subscribeToUpdateOperation).mock.calls[0][1].onStateChange;

    onStateChange?.('closed');
    onStateChange?.('open');
    await vi.advanceTimersByTimeAsync(3000);

    expect(store.phase).toBe('running');
    expect(getUpdateOperation).toHaveBeenCalledOnce();
  });

  it('clears a persisted operation after an unrecoverable snapshot error', async () => {
    vi.mocked(getUpdateOperation).mockRejectedValueOnce(
      Object.assign(new Error('operation not found'), { isApiRequestError: true, status: 404 }),
    );
    const store = useUpdateProgressStore();

    await store.begin(acknowledgement('operation-1'));

    expect(store.phase).toBe('failed');
    expect(store.operationID).toBeNull();
    expect(sessionStorage.getItem('graft.platform-update.operation-id')).toBeNull();
  });

  it('stops retrying after repeated recoverable snapshot errors', async () => {
    vi.mocked(getUpdateOperation).mockRejectedValue(new Error('service unavailable'));
    const store = useUpdateProgressStore();

    await store.begin(acknowledgement('operation-1'));
    await vi.advanceTimersByTimeAsync(12_000);

    expect(store.phase).toBe('failed');
    expect(store.operationID).toBeNull();
    expect(getUpdateOperation).toHaveBeenCalledTimes(5);
  });
});
