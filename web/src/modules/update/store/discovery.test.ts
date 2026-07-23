import { createPinia, setActivePinia } from 'pinia';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import { usePermissionStore } from '@/store';

import { getUpdateStatus } from '../api/update';
import { useUpdateDiscoveryStore } from './discovery';

vi.mock('../api/update', () => ({ getUpdateStatus: vi.fn() }));

describe('update discovery store', () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    vi.clearAllMocks();
  });

  it('loads the authorized discovery snapshot once for concurrent consumers', async () => {
    const permissions = usePermissionStore();
    permissions.setBootstrapSnapshot({ permissions: ['platform-update.read'] } as never);
    vi.mocked(getUpdateStatus).mockResolvedValue({ current_version: '1.0.0' } as never);
    const store = useUpdateDiscoveryStore();

    await Promise.all([store.ensureSnapshot(), store.ensureSnapshot()]);
    await store.ensureSnapshot();

    expect(getUpdateStatus).toHaveBeenCalledTimes(1);
    expect(store.status?.current_version).toBe('1.0.0');
  });

  it('does not request discovery without the read permission', async () => {
    const store = useUpdateDiscoveryStore();

    await store.ensureSnapshot();

    expect(getUpdateStatus).not.toHaveBeenCalled();
    expect(store.phase).toBe('idle');
  });

  it('does not expose stale or failed releases as an available update', () => {
    const permissions = usePermissionStore();
    permissions.setBootstrapSnapshot({ permissions: ['platform-update.read'] } as never);
    const store = useUpdateDiscoveryStore();
    const latest = { version: '1.1.0' };

    store.replaceSnapshot({ latest, cache_stale: true, check_error: '' } as never);
    expect(store.hasUpdate).toBe(false);

    store.replaceSnapshot({ latest, cache_stale: false, check_error: 'catalog-unavailable' } as never);
    expect(store.hasUpdate).toBe(false);

    store.replaceSnapshot({ latest, cache_stale: false, check_error: '' } as never);
    expect(store.hasUpdate).toBe(true);
  });

  it('allows a failed discovery request to be retried', async () => {
    const permissions = usePermissionStore();
    permissions.setBootstrapSnapshot({ permissions: ['platform-update.read'] } as never);
    vi.mocked(getUpdateStatus)
      .mockRejectedValueOnce(new Error('temporary failure'))
      .mockResolvedValueOnce({ current_version: '1.0.0' } as never);
    const store = useUpdateDiscoveryStore();

    await store.ensureSnapshot();
    expect(store.phase).toBe('error');
    expect(store.error).toBe('load-failed');

    await store.ensureSnapshot();
    expect(getUpdateStatus).toHaveBeenCalledTimes(2);
    expect(store.phase).toBe('ready');
    expect(store.error).toBe('');
  });

  it('does not let an older initial request overwrite a manual refresh', async () => {
    const permissions = usePermissionStore();
    permissions.setBootstrapSnapshot({ permissions: ['platform-update.read'] } as never);
    let resolveInitial!: (status: never) => void;
    vi.mocked(getUpdateStatus).mockReturnValueOnce(new Promise((resolve) => (resolveInitial = resolve)));
    const store = useUpdateDiscoveryStore();

    const initialRequest = store.ensureSnapshot();
    const refreshed = { current_version: '2.0.0' } as never;
    store.replaceSnapshot(refreshed);
    resolveInitial({ current_version: '1.0.0' } as never);
    await initialRequest;

    expect(store.status).toEqual(refreshed);
  });

  it('invalidates the current snapshot when a manual refresh fails', () => {
    const permissions = usePermissionStore();
    permissions.setBootstrapSnapshot({ permissions: ['platform-update.read'] } as never);
    const store = useUpdateDiscoveryStore();
    const current = { latest: { version: '1.1.0' }, cache_stale: false, check_error: '' } as never;

    store.replaceSnapshot(current);
    store.invalidateSnapshot();

    expect(store.phase).toBe('error');
    expect(store.hasUpdate).toBe(false);
    expect(store.status?.check_error).toBe('check-failed');
  });
});
