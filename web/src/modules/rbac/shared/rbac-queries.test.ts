import { beforeEach, describe, expect, it } from 'vitest';

import { queryClient } from '@/shared/query';

import { invalidateRolesQuery, normalizePermissionFilters, rbacQueryKeys } from './rbac-queries';

describe('RBAC query cache', () => {
  beforeEach(() => {
    queryClient.clear();
  });

  it('normalizes empty permission filters into one stable key', () => {
    expect(normalizePermissionFilters({ keyword: ' user.read ', module: undefined })).toEqual({
      keyword: 'user.read',
      module: '',
      limit: 20,
      offset: 0,
    });
    expect(rbacQueryKeys.permissionList(normalizePermissionFilters())).toEqual([
      'rbac',
      'permissions',
      { keyword: '', module: '', limit: 20, offset: 0 },
    ]);
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
