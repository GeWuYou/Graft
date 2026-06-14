// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from 'vitest';

import { userBootstrapRouteRegistrations } from './bootstrap-routes';
import { USER_ROUTE_PATH } from './contract/paths';

describe('user bootstrap route registrations', () => {
  it('marks the user list as a paged list surface', () => {
    expect(userBootstrapRouteRegistrations).toHaveLength(1);
    expect(userBootstrapRouteRegistrations[0]).toMatchObject({
      menuPath: USER_ROUTE_PATH.LIST,
      routeName: 'UserList',
      meta: expect.objectContaining({
        pageKind: 'list',
      }),
    });
  });
});
