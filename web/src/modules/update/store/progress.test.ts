import { flushPromises } from '@vue/test-utils';
import { createPinia, setActivePinia } from 'pinia';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import {
  getActiveUpdateOperation,
  getUpdateOperation,
  getUpdateOperationDiagnostic,
  getUpdateOperationEvents,
  recoverUpdateOperation,
  subscribeToUpdateOperation,
} from '../api/update';
import type { UpdateOperation } from '../types/update';
import { useUpdateProgressStore } from './progress';

vi.mock('../api/update', () => ({
  getActiveUpdateOperation: vi.fn(),
  recoverUpdateOperation: vi.fn(),
  getUpdateOperation: vi.fn(),
  getUpdateOperationDiagnostic: vi.fn(),
  getUpdateOperationEvents: vi.fn(),
  subscribeToUpdateOperation: vi.fn(),
}));

const operation = (operationID: string, phase: UpdateOperation['phase'] = 'READY', progress = 0): UpdateOperation =>
  ({
    operation_id: operationID,
    runner_id: 'runner-1',
    phase,
    progress,
    message: '',
    state_available: true,
    state_source: 'runner_state',
  }) as UpdateOperation;
const acknowledgement = (operationID: string) => ({ operation_id: operationID, runner_id: 'runner-1' }) as never;
const streamController = () => ({ close: vi.fn(), reconnect: vi.fn() });

describe('update progress store', () => {
  beforeEach(() => {
    sessionStorage.clear();
    setActivePinia(createPinia());
    vi.clearAllMocks();
    vi.useFakeTimers();
    vi.mocked(getActiveUpdateOperation).mockResolvedValue(null);
    vi.mocked(recoverUpdateOperation).mockResolvedValue({
      operation_id: 'operation-1',
      runner_id: 'recovery-runner-1',
    } as never);
    vi.mocked(getUpdateOperation).mockResolvedValue(operation('operation-1'));
    vi.mocked(getUpdateOperationEvents).mockResolvedValue([]);
    vi.mocked(getUpdateOperationDiagnostic).mockResolvedValue({
      summary: 'Runner exited before the update could continue.',
      detail: 'The runner could not persist its terminal state.',
    } as never);
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
    let resolveFirst: (value: UpdateOperation) => void = () => undefined;
    vi.mocked(getUpdateOperation).mockImplementationOnce(
      () => new Promise((resolve) => (resolveFirst = resolve as (value: UpdateOperation) => void)),
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

  it('discovers an active operation from the server when a new tab has no browser state', async () => {
    vi.mocked(getActiveUpdateOperation).mockResolvedValue(operation('operation-1'));
    const store = useUpdateProgressStore();

    await store.resume();

    expect(getActiveUpdateOperation).toHaveBeenCalledOnce();
    expect(store.operation?.operation_id).toBe('operation-1');
    expect(subscribeToUpdateOperation).toHaveBeenCalledWith('operation-1', expect.any(Object));
  });

  it('keeps active-operation recovery bound to its captured session after an async continuation', async () => {
    vi.mocked(getActiveUpdateOperation).mockResolvedValue(operation('operation-1'));
    const store = useUpdateProgressStore();
    vi.spyOn(store, 'applyOperation').mockImplementation(async () => {
      store.session += 1;
    });
    const refreshEvents = vi.spyOn(store, 'refreshEvents').mockResolvedValue();
    const connect = vi.spyOn(store, 'connect').mockImplementation(() => undefined);

    await store.resume();

    expect(refreshEvents).toHaveBeenCalledWith(1);
    expect(connect).toHaveBeenCalledWith(1);
  });

  it('keeps the shell idle when active operation discovery is temporarily unavailable', async () => {
    vi.mocked(getActiveUpdateOperation).mockRejectedValue(new Error('service unavailable'));
    const store = useUpdateProgressStore();

    await expect(store.resume()).resolves.toBeUndefined();

    expect(store.phase).toBe('idle');
  });

  it('replays missed node events and deduplicates an overlapping realtime revision', async () => {
    vi.mocked(getUpdateOperationEvents).mockResolvedValue([
      {
        operation_id: 'operation-1',
        revision: 1,
        phase: 'PREFLIGHT',
        message: 'checking_environment',
        occurred_at: '',
      },
    ] as never);
    const store = useUpdateProgressStore();
    await store.begin(acknowledgement('operation-1'));
    const onOperation = vi.mocked(subscribeToUpdateOperation).mock.calls[0][1].onOperation;

    await onOperation({
      event: {
        operation_id: 'operation-1',
        revision: 1,
        phase: 'PREFLIGHT',
        message: 'checking_environment',
        occurred_at: '',
      },
    });
    await onOperation({
      event: { operation_id: 'operation-1', revision: 2, phase: 'BACKUP', message: 'creating_backup', occurred_at: '' },
    });

    expect(store.events.map((event) => event.revision)).toEqual([1, 2]);
    expect(getUpdateOperationEvents).toHaveBeenCalledWith('operation-1', 0);
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

  it('does not render runner-state-unavailable as live READY progress', async () => {
    const store = useUpdateProgressStore();
    await store.begin(acknowledgement('operation-1'));
    const callback = vi.mocked(subscribeToUpdateOperation).mock.calls[0][1].onOperation;

    await callback({
      ...operation('operation-1'),
      state_available: false,
      state_source: 'runner_state_unavailable' as const,
    });

    expect(store.phase).toBe('unavailable');
    expect(store.operation?.operation_id).toBe('operation-1');
  });

  it('ends the browser session and loads protected diagnostics when the runner has terminated', async () => {
    const store = useUpdateProgressStore();
    await store.begin(acknowledgement('operation-1'));
    const callback = vi.mocked(subscribeToUpdateOperation).mock.calls[0][1].onOperation;

    const runnerTerminatedOperation: UpdateOperation = {
      ...operation('operation-1'),
      state_available: false,
      state_source: 'runner_terminated',
      failure_diagnostic_available: true,
    };
    await callback(runnerTerminatedOperation);

    expect(store.phase).toBe('failed');
    expect(store.operation?.phase).toBe('READY');
    expect(store.lastActivePhase).toBeNull();
    expect(store.operationID).toBeNull();
    expect(sessionStorage.getItem('graft.platform-update.operation-id')).toBeNull();
    expect(getUpdateOperationDiagnostic).toHaveBeenCalledWith('operation-1');
    expect(store.failureDiagnostic).toMatchObject({
      summary: 'Runner exited before the update could continue.',
    });
  });

  it('keeps runner termination terminal when the protected diagnostic cannot be read', async () => {
    vi.mocked(getUpdateOperationDiagnostic).mockRejectedValueOnce(new Error('diagnostic unavailable'));
    const store = useUpdateProgressStore();
    await store.begin(acknowledgement('operation-1'));
    const callback = vi.mocked(subscribeToUpdateOperation).mock.calls[0][1].onOperation;
    const runnerTerminatedOperation: UpdateOperation = {
      ...operation('operation-1'),
      state_available: false,
      state_source: 'runner_terminated',
      failure_diagnostic_available: true,
    };

    await callback(runnerTerminatedOperation);

    expect(store.phase).toBe('failed');
    expect(store.failureDiagnostic).toBeNull();
    expect(store.failureDiagnosticError).toBe(true);
    expect(store.stream).toBeNull();
    expect(store.pollTimer).toBeNull();
  });

  it('starts the protected recovery runner and resumes the returned operation session', async () => {
    const store = useUpdateProgressStore();
    store.$patch({
      operation: {
        ...operation('operation-1'),
        state_available: false,
        state_source: 'runner_terminated',
      } as UpdateOperation,
      phase: 'failed',
    });

    await store.recoverTerminatedRunner();

    expect(recoverUpdateOperation).toHaveBeenCalledWith('operation-1');
    expect(getUpdateOperation).toHaveBeenCalledWith('operation-1');
    expect(store.phase).toBe('running');
    expect(store.operationID).toBe('operation-1');
    expect(store.recoveryError).toBe(false);
  });

  it('keeps the terminated runner visible when protected recovery cannot be accepted', async () => {
    vi.mocked(recoverUpdateOperation).mockRejectedValueOnce(new Error('recovery unavailable'));
    const store = useUpdateProgressStore();
    store.$patch({
      operation: {
        ...operation('operation-1'),
        state_available: false,
        state_source: 'runner_terminated',
      } as UpdateOperation,
      phase: 'failed',
    });

    await store.recoverTerminatedRunner();

    expect(store.phase).toBe('failed');
    expect(store.operation?.state_source).toBe('runner_terminated');
    expect(store.recoveryError).toBe(true);
  });

  it('keeps polling when the recovery acknowledgement is followed by the stale terminated projection', async () => {
    vi.mocked(getUpdateOperation)
      .mockResolvedValueOnce({
        ...operation('operation-1'),
        state_available: false,
        state_source: 'runner_terminated',
      } as UpdateOperation)
      .mockResolvedValueOnce(operation('operation-1', 'FAILED', 100));
    const store = useUpdateProgressStore();
    store.$patch({
      operation: {
        ...operation('operation-1'),
        state_available: false,
        state_source: 'runner_terminated',
      } as UpdateOperation,
      phase: 'failed',
    });

    await store.recoverTerminatedRunner();

    expect(store.phase).toBe('reconnecting');
    expect(store.recoveryPending).toBe(true);
    expect(store.operationID).toBe('operation-1');
    const onStateChange = vi.mocked(subscribeToUpdateOperation).mock.calls[0][1].onStateChange;
    onStateChange?.('open');
    await flushPromises();

    expect(store.phase).toBe('reconnecting');
    await vi.advanceTimersByTimeAsync(3000);

    expect(store.phase).toBe('failed');
    expect(store.recoveryPending).toBe(false);
    expect(store.operation?.phase).toBe('FAILED');
  });

  it('ignores a snapshot that returns after runner termination invalidates its session', async () => {
    let resolveSnapshot: (value: UpdateOperation) => void = () => undefined;
    vi.mocked(getUpdateOperation)
      .mockResolvedValueOnce(operation('operation-1'))
      .mockImplementationOnce(
        () => new Promise((resolve) => (resolveSnapshot = resolve as (value: UpdateOperation) => void)),
      );
    const store = useUpdateProgressStore();
    await store.begin(acknowledgement('operation-1'));
    const staleRefresh = store.refreshSnapshot(store.session);
    const callback = vi.mocked(subscribeToUpdateOperation).mock.calls[0][1].onOperation;

    await callback({
      ...operation('operation-1'),
      state_available: false,
      state_source: 'runner_terminated',
      failure_diagnostic_available: false,
    } as UpdateOperation);
    resolveSnapshot(operation('operation-1', 'BACKUP', 20));
    await staleRefresh;

    expect(store.phase).toBe('failed');
    expect(store.operationID).toBeNull();
    expect(store.operation?.state_source).toBe('runner_terminated');
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

  it('keeps a persisted operation visible when its runner snapshot is unavailable', async () => {
    vi.mocked(getUpdateOperation).mockRejectedValueOnce(
      Object.assign(new Error('operation not found'), { isApiRequestError: true, status: 404 }),
    );
    const store = useUpdateProgressStore();

    await store.begin(acknowledgement('operation-1'));

    expect(store.phase).toBe('unavailable');
    expect(store.operationID).toBe('operation-1');
    expect(sessionStorage.getItem('graft.platform-update.operation-id')).toBe('operation-1');
  });

  it('stops retrying after repeated recoverable snapshot errors', async () => {
    vi.mocked(getUpdateOperation).mockRejectedValue(new Error('service unavailable'));
    const store = useUpdateProgressStore();

    await store.begin(acknowledgement('operation-1'));
    await vi.advanceTimersByTimeAsync(12_000);

    expect(store.phase).toBe('unavailable');
    expect(store.operationID).toBe('operation-1');
    expect(getUpdateOperation).toHaveBeenCalledTimes(5);
  });
});
