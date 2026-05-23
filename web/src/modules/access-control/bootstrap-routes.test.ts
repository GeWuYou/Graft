import { describe, expect, it } from 'vitest';

import { accessControlBootstrapRouteRegistrations } from './bootstrap-routes';
import { ACCESS_CONTROL_BOOTSTRAP_ROUTE } from './contract/bootstrap';

describe('access control bootstrap route registrations', () => {
  it('uses the canonical access control bootstrap contract values', () => {
    expect(accessControlBootstrapRouteRegistrations).toHaveLength(1);
    expect(accessControlBootstrapRouteRegistrations[0]).toMatchObject(ACCESS_CONTROL_BOOTSTRAP_ROUTE.OVERVIEW);
  });
});
