// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from 'vitest';

import { systemConfigBootstrapRouteRegistrations } from './bootstrap-routes';

describe('system config bootstrap route registrations', () => {
  it('uses the canonical system config bootstrap identity contract values', () => {
    expect(systemConfigBootstrapRouteRegistrations).toHaveLength(1);
    expect(systemConfigBootstrapRouteRegistrations[0]).toMatchObject({
      menuPath: '/server/system-config',
      routeName: 'SystemConfigList',
    });
  });
});
