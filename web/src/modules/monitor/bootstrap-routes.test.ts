// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from 'vitest';

import { monitorBootstrapRouteRegistrations } from './bootstrap-routes';
import { MONITOR_ROUTE_PATH } from './contract/paths';

describe('monitor bootstrap route registrations', () => {
  it('keeps module runtime dashboard semantics while using the paged table surface', () => {
    const moduleRuntimeRoute = monitorBootstrapRouteRegistrations.find(
      (registration) => registration.menuPath === MONITOR_ROUTE_PATH.SERVER_MODULES,
    );

    expect(moduleRuntimeRoute).toMatchObject({
      routeName: 'MonitorModuleRuntimeOverview',
      meta: expect.objectContaining({
        dashboard: true,
        pageKind: 'overview',
        pageSurface: 'paged-table',
      }),
    });
  });
});
