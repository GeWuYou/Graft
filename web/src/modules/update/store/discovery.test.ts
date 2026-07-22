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
});
