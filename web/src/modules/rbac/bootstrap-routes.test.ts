// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from 'vitest';

import { rbacBootstrapRouteRegistrations } from './bootstrap-routes';
import { RBAC_BOOTSTRAP_ROUTE } from './contract/bootstrap';

describe('rbac bootstrap route registrations', () => {
  it('uses the canonical RBAC bootstrap identity contract values', () => {
    expect(rbacBootstrapRouteRegistrations).toHaveLength(2);
    expect(rbacBootstrapRouteRegistrations).toEqual(
      expect.arrayContaining([
        expect.objectContaining({
          ...RBAC_BOOTSTRAP_ROUTE.ROLE_LIST,
          meta: expect.objectContaining({
            domain: 'rbac',
            pageKind: 'list',
          }),
        }),
        expect.objectContaining({
          ...RBAC_BOOTSTRAP_ROUTE.PERMISSION_LIST,
          meta: expect.objectContaining({
            domain: 'rbac',
            pageKind: 'list',
          }),
        }),
      ]),
    );
  });
});
