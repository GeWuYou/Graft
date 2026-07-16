import { beforeEach, describe, expect, it } from 'vitest';

import { queryClient } from '@/shared/query';

import type { RoleListResponse } from '../contract/role';
import { invalidateRolesQuery, normalizePermissionFilters, rbacQueryKeys, updateRoleListCache } from './rbac-queries';

describe('RBAC query cache', () => {
  beforeEach(() => {
    queryClient.clear();
  });

  it('normalizes empty permission filters into one stable key', () => {
    expect(normalizePermissionFilters({ keyword: ' user.read ', module: undefined })).toEqual({
      keyword: 'user.read',
      module: '',
    });
    expect(rbacQueryKeys.permissionList(normalizePermissionFilters())).toEqual([
      'rbac',
      'permissions',
      { keyword: '', module: '' },
    ]);
  });

  it('updates only the cached role list from a confirmed mutation response', () => {
    queryClient.setQueryData(rbacQueryKeys.roles(), {
      items: [
        {
          id: 4,
          name: 'auditor',
          display: 'Auditor',
          builtin: false,
          permission_count: 1,
          user_count: 0,
          updated_at: '2026-07-16T00:00:00Z',
        },
      ],
    });

    updateRoleListCache((items) =>
      items.map((item) => (item.id === 4 ? { ...item, display: 'Security Auditor' } : item)),
    );

    expect(queryClient.getQueryData<RoleListResponse>(rbacQueryKeys.roles())?.items[0]?.display).toBe(
      'Security Auditor',
    );
  });

  it('invalidates roles without clearing unrelated permission snapshots', async () => {
    const permissionKey = rbacQueryKeys.permissionList(normalizePermissionFilters());
    queryClient.setQueryData(rbacQueryKeys.roles(), { items: [] });
    queryClient.setQueryData(permissionKey, { items: [] });

    await invalidateRolesQuery();

    expect(queryClient.getQueryState(rbacQueryKeys.roles())?.isInvalidated).toBe(true);
    expect(queryClient.getQueryData(permissionKey)).toEqual({ items: [] });
  });
});
